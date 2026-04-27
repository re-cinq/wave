package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TemplateContext holds resolved values for template interpolation in composition steps.
type TemplateContext struct {
	StepOutputs   map[string][]byte // stepID → raw artifact content (JSON)
	Input         string            // Pipeline input string
	Item          json.RawMessage   // Current iteration item (for iterate steps)
	Iteration     int               // Current loop iteration (0-based)
	WorkspaceRoot string            // Base workspace path for artifact resolution
	// Env carries arbitrary string vars propagated from a parent sub-pipeline
	// step's Config.Env. Resolved as {{ env.<key> }}. Distinct from process
	// environment — never reads from os.Environ.
	Env map[string]string
}

// NewTemplateContext creates an empty template context.
func NewTemplateContext(input string, workspaceRoot string) *TemplateContext {
	return &TemplateContext{
		StepOutputs:   make(map[string][]byte),
		Input:         input,
		WorkspaceRoot: workspaceRoot,
		Env:           make(map[string]string),
	}
}

// SetStepOutput records the output artifact content for a completed step.
func (tc *TemplateContext) SetStepOutput(stepID string, data []byte) {
	tc.StepOutputs[stepID] = data
}

// templatePattern matches {{expressions}} in template strings.
var templatePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ResolveTemplate resolves template expressions in a string.
//
// Supported patterns:
//   - {{input}}                → pipeline input string
//   - {{step_id.output}}      → full artifact content from step
//   - {{step_id.output.field}} → extracted field from step artifact (via dot-path)
//   - {{item}}                → current iteration item (JSON)
//   - {{item.field}}          → field from current iteration item
//   - {{iteration}}           → current loop iteration number
//   - {{env.<key>}}           → value from sub-pipeline-supplied Env map (NOT process env)
func ResolveTemplate(tmpl string, ctx *TemplateContext) (string, error) {
	var resolveErr error
	result := templatePattern.ReplaceAllStringFunc(tmpl, func(match string) string {
		if resolveErr != nil {
			return match
		}
		// Strip {{ and }}
		expr := strings.TrimSpace(match[2 : len(match)-2])
		val, err := resolveExpression(expr, ctx)
		if err != nil {
			resolveErr = fmt.Errorf("failed to resolve {{%s}}: %w", expr, err)
			return match
		}
		return val
	})
	if resolveErr != nil {
		return "", resolveErr
	}
	return result, nil
}

// resolveExpression resolves a single template expression.
func resolveExpression(expr string, ctx *TemplateContext) (string, error) {
	switch {
	case expr == "input":
		return ctx.Input, nil

	case expr == "iteration":
		return fmt.Sprintf("%d", ctx.Iteration), nil

	case expr == "item":
		if ctx.Item == nil {
			return "", fmt.Errorf("no iteration item in context")
		}
		// Unquote JSON strings so "audit-security" → audit-security
		var s string
		if err := json.Unmarshal(ctx.Item, &s); err == nil {
			return s, nil
		}
		return string(ctx.Item), nil

	case strings.HasPrefix(expr, "item."):
		if ctx.Item == nil {
			return "", fmt.Errorf("no iteration item in context")
		}
		field := expr[5:] // strip "item."
		return ExtractJSONPath(ctx.Item, "."+field)

	case strings.HasPrefix(expr, "env."):
		// Sub-pipeline env passthrough — distinct from process environment.
		// Missing keys resolve to empty string so callers can rely on
		// {{ env.profile }} cleanly defaulting via branch `cases.default`.
		key := expr[4:] // strip "env."
		if ctx.Env == nil {
			return "", nil
		}
		return ctx.Env[key], nil

	default:
		// Must be step_id.output or step_id.output.field
		return resolveStepOutput(expr, ctx)
	}
}

// resolveStepOutput resolves a step output reference like "step_id.output" or "step_id.output.field.nested".
func resolveStepOutput(expr string, ctx *TemplateContext) (string, error) {
	parts := strings.SplitN(expr, ".", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid expression %q: expected step_id.output or step_id.output.field", expr)
	}

	stepID := parts[0]
	if parts[1] != "output" {
		return "", fmt.Errorf("invalid expression %q: second segment must be 'output'", expr)
	}

	data, ok := ctx.StepOutputs[stepID]
	if !ok {
		return "", fmt.Errorf("no output found for step %q", stepID)
	}

	if len(parts) == 2 {
		// {{step_id.output}} → return full content
		return string(data), nil
	}

	// {{step_id.output.field.nested}} → extract via JSON path
	field := parts[2]
	return ExtractJSONPath(data, "."+field)
}

// LoadStepArtifact reads a step's output artifact from the workspace filesystem.
// It looks for the artifact in the step's workspace output directory.
func LoadStepArtifact(workspaceRoot, pipelineID, stepID, artifactName string) ([]byte, error) {
	// Try common artifact locations
	candidates := []string{
		filepath.Join(workspaceRoot, pipelineID, stepID, ".agents", "output", artifactName),
		filepath.Join(workspaceRoot, pipelineID, stepID, ".agents", "artifacts", artifactName),
		filepath.Join(workspaceRoot, pipelineID, stepID, artifactName),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("artifact %q not found for step %q (checked %d locations)", artifactName, stepID, len(candidates))
}
