package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"gopkg.in/yaml.v3"
)

const (
	DefaultMaxDepth       = 3
	DefaultMaxTotalSteps  = 20
	DefaultMaxTotalTokens = 500000
	DefaultMetaTimeout    = 30 * time.Minute

	PhilosopherPersona = "philosopher"
	NavigatorPersona   = "navigator"
)

// MetaPipelineExecutor executes meta-pipelines that can generate and execute
// child pipelines dynamically using a philosopher persona.
type MetaPipelineExecutor struct {
	runner   adapter.AdapterRunner
	emitter  event.EventEmitter
	executor PipelineExecutor
	loader   *YAMLPipelineLoader

	// Tracking state for the entire meta-pipeline execution tree
	currentDepth     int
	totalStepsUsed   int
	totalTokensUsed  int
	parentPipelineID string
}

// MetaExecutorOption configures the MetaPipelineExecutor.
type MetaExecutorOption func(*MetaPipelineExecutor)

// WithMetaDepth sets the initial recursion depth (passed from parent pipelines).
func WithMetaDepth(depth int) MetaExecutorOption {
	return func(e *MetaPipelineExecutor) { e.currentDepth = depth }
}

// WithParentPipeline sets the parent pipeline ID for tracking.
func WithParentPipeline(id string) MetaExecutorOption {
	return func(e *MetaPipelineExecutor) { e.parentPipelineID = id }
}

// WithMetaEmitter sets the event emitter for the meta executor.
func WithMetaEmitter(em event.EventEmitter) MetaExecutorOption {
	return func(e *MetaPipelineExecutor) { e.emitter = em }
}

// WithChildExecutor sets the pipeline executor for running child pipelines.
func WithChildExecutor(ex PipelineExecutor) MetaExecutorOption {
	return func(e *MetaPipelineExecutor) { e.executor = ex }
}

// NewMetaPipelineExecutor creates a new meta-pipeline executor.
func NewMetaPipelineExecutor(runner adapter.AdapterRunner, opts ...MetaExecutorOption) *MetaPipelineExecutor {
	e := &MetaPipelineExecutor{
		runner:       runner,
		loader:       &YAMLPipelineLoader{},
		currentDepth: 0,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// MetaExecutionResult contains the results of a meta-pipeline execution.
type MetaExecutionResult struct {
	GeneratedPipeline *Pipeline
	TotalSteps        int
	TotalTokens       int
	Depth             int
	ChildResults      []MetaExecutionResult
}

// GenerateOnly generates a pipeline using the philosopher persona without executing it.
// This is useful for dry-run mode and inspecting what would be generated.
func (e *MetaPipelineExecutor) GenerateOnly(ctx context.Context, task string, m *manifest.Manifest) (*Pipeline, error) {
	config := e.getMetaConfig(m)

	// Check depth limit
	if err := e.checkDepthLimit(config); err != nil {
		return nil, err
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: e.getPipelineID(),
		State:      "meta_generate_started",
		Message:    fmt.Sprintf("depth=%d task=%q", e.currentDepth, truncate(task, 100)),
	})

	// Invoke philosopher to generate pipeline YAML
	generatedYAML, tokensUsed, err := e.invokePhilosopher(ctx, task, m)
	if err != nil {
		return nil, fmt.Errorf("philosopher failed to generate pipeline: %w", err)
	}

	e.totalTokensUsed += tokensUsed
	if err := e.checkTokenLimit(config); err != nil {
		return nil, err
	}

	// Parse and validate the generated pipeline
	pipeline, err := e.loader.Unmarshal([]byte(generatedYAML))
	if err != nil {
		// Emit debug event with the raw YAML for troubleshooting
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: e.getPipelineID(),
			State:      "meta_generate_failed",
			Message:    fmt.Sprintf("YAML parse error: %v", err),
		})
		return nil, fmt.Errorf("failed to parse generated pipeline YAML: %w\n\n--- Generated YAML ---\n%s\n--- End YAML ---", err, generatedYAML)
	}

	// Validate the generated pipeline structure
	if err := ValidateGeneratedPipeline(pipeline); err != nil {
		return nil, fmt.Errorf("generated pipeline validation failed: %w\n\n--- Generated YAML ---\n%s\n--- End YAML ---", err, generatedYAML)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: e.getPipelineID(),
		State:      "meta_generate_completed",
		Message:    fmt.Sprintf("generated %d steps, tokens_used=%d", len(pipeline.Steps), tokensUsed),
	})

	return pipeline, nil
}

// Execute runs a meta-pipeline: generates a pipeline using the philosopher persona
// and then executes the generated pipeline.
func (e *MetaPipelineExecutor) Execute(ctx context.Context, task string, m *manifest.Manifest) (*MetaExecutionResult, error) {
	config := e.getMetaConfig(m)

	// Check depth limit
	if err := e.checkDepthLimit(config); err != nil {
		return nil, err
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: e.getPipelineID(),
		State:      "meta_started",
		Message:    fmt.Sprintf("depth=%d task=%q", e.currentDepth, truncate(task, 100)),
	})

	// Invoke philosopher to generate pipeline YAML
	generatedYAML, tokensUsed, err := e.invokePhilosopher(ctx, task, m)
	if err != nil {
		return nil, fmt.Errorf("philosopher failed to generate pipeline: %w", err)
	}

	e.totalTokensUsed += tokensUsed
	if err := e.checkTokenLimit(config); err != nil {
		return nil, err
	}

	// Parse and validate the generated pipeline
	pipeline, err := e.loader.Unmarshal([]byte(generatedYAML))
	if err != nil {
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: e.getPipelineID(),
			State:      "meta_generate_failed",
			Message:    fmt.Sprintf("YAML parse error: %v", err),
		})
		return nil, fmt.Errorf("failed to parse generated pipeline YAML: %w\n\n--- Generated YAML ---\n%s\n--- End YAML ---", err, generatedYAML)
	}

	// Validate the generated pipeline structure
	if err := ValidateGeneratedPipeline(pipeline); err != nil {
		return nil, fmt.Errorf("generated pipeline validation failed: %w\n\n--- Generated YAML ---\n%s\n--- End YAML ---", err, generatedYAML)
	}

	// Check step limit
	e.totalStepsUsed += len(pipeline.Steps)
	if err := e.checkStepLimit(config); err != nil {
		return nil, err
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: e.getPipelineID(),
		State:      "meta_pipeline_generated",
		Message:    fmt.Sprintf("generated %d steps, tokens_used=%d", len(pipeline.Steps), tokensUsed),
	})

	// Execute the generated pipeline
	result := &MetaExecutionResult{
		GeneratedPipeline: pipeline,
		Depth:             e.currentDepth,
	}

	if e.executor != nil {
		if err := e.executor.Execute(ctx, pipeline, m, task); err != nil {
			return result, fmt.Errorf("child pipeline execution failed: %w", err)
		}
	}

	result.TotalSteps = e.totalStepsUsed
	result.TotalTokens = e.totalTokensUsed

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: e.getPipelineID(),
		State:      "meta_completed",
		Message:    fmt.Sprintf("total_steps=%d total_tokens=%d", e.totalStepsUsed, e.totalTokensUsed),
	})

	return result, nil
}

// invokePhilosopher calls the philosopher persona to generate a pipeline YAML.
func (e *MetaPipelineExecutor) invokePhilosopher(ctx context.Context, task string, m *manifest.Manifest) (string, int, error) {
	genResult, tokensUsed, err := e.invokePhilosopherWithSchemas(ctx, task, m)
	if err != nil {
		return "", tokensUsed, err
	}
	return genResult.PipelineYAML, tokensUsed, nil
}

// invokePhilosopherWithSchemas calls the philosopher persona to generate pipeline and schemas.
func (e *MetaPipelineExecutor) invokePhilosopherWithSchemas(ctx context.Context, task string, m *manifest.Manifest) (*PipelineGenerationResult, int, error) {
	persona := m.GetPersona(PhilosopherPersona)
	if persona == nil {
		return nil, 0, fmt.Errorf("philosopher persona not found in manifest")
	}

	adapterDef := m.GetAdapter(persona.Adapter)
	if adapterDef == nil {
		return nil, 0, fmt.Errorf("adapter %q for philosopher not found", persona.Adapter)
	}

	prompt := buildPhilosopherPrompt(task, e.currentDepth)

	cfg := adapter.AdapterRunConfig{
		Adapter:      adapterDef.Binary,
		Persona:      PhilosopherPersona,
		Prompt:       prompt,
		Timeout:      e.getTimeout(m),
		Temperature:  persona.Temperature,
		AllowedTools: persona.Permissions.AllowedTools,
		DenyTools:    persona.Permissions.Deny,
		OutputFormat: "yaml",
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: e.getPipelineID(),
		State:      "philosopher_invoking",
		Message:    fmt.Sprintf("adapter=%s timeout=%v", cfg.Adapter, cfg.Timeout),
	})

	result, err := e.runner.Run(ctx, cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("philosopher adapter execution failed: %w", err)
	}

	// Read stdout from the adapter result
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, _ := result.Stdout.Read(buf)
	output := string(buf[:n])

	// Extract pipeline and schemas from the output
	genResult, err := extractPipelineAndSchemas(output)
	if err != nil {
		return nil, result.TokensUsed, fmt.Errorf("failed to extract pipeline and schemas: %w", err)
	}

	// Save schema files to disk
	if err := e.saveSchemaFiles(genResult.Schemas); err != nil {
		return nil, result.TokensUsed, fmt.Errorf("failed to save schema files: %w", err)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: e.getPipelineID(),
		State:      "philosopher_completed",
		Message:    fmt.Sprintf("tokens_used=%d schemas_generated=%d", result.TokensUsed, len(genResult.Schemas)),
	})

	return genResult, result.TokensUsed, nil
}

// saveSchemaFiles writes schema definitions to their respective files with JSON validation and formatting.
func (e *MetaPipelineExecutor) saveSchemaFiles(schemas map[string]string) error {
	for schemaPath, schemaContent := range schemas {
		// Lint and format the JSON schema
		formattedContent, err := e.lintAndFormatJSON(schemaContent, schemaPath)
		if err != nil {
			return fmt.Errorf("failed to lint/format schema %s: %w", schemaPath, err)
		}

		// Ensure the directory exists
		dir := filepath.Dir(schemaPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write the formatted schema file
		if err := os.WriteFile(schemaPath, formattedContent, 0644); err != nil {
			return fmt.Errorf("failed to write schema file %s: %w", schemaPath, err)
		}

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: e.getPipelineID(),
			State:      "schema_saved",
			Message:    fmt.Sprintf("schema_path=%s formatted=%t", schemaPath, true),
		})
	}
	return nil
}

// lintAndFormatJSON validates and formats JSON content, fixing common issues.
func (e *MetaPipelineExecutor) lintAndFormatJSON(content, schemaPath string) ([]byte, error) {
	content = strings.TrimSpace(content)

	// Try to parse as JSON first
	var jsonObj interface{}
	err := json.Unmarshal([]byte(content), &jsonObj)

	if err != nil {
		// If JSON parsing fails, try to fix common issues
		fixedContent, fixErr := e.attemptJSONFix(content)
		if fixErr != nil {
			return nil, fmt.Errorf("JSON syntax error in %s: %w (original: %v)", schemaPath, fixErr, err)
		}

		// Try parsing the fixed content
		err = json.Unmarshal([]byte(fixedContent), &jsonObj)
		if err != nil {
			return nil, fmt.Errorf("JSON still invalid after fix attempt in %s: %w", schemaPath, err)
		}
		content = fixedContent

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: e.getPipelineID(),
			State:      "schema_fixed",
			Message:    fmt.Sprintf("schema_path=%s fixed_json_syntax=true", schemaPath),
		})
	}

	// Format the JSON with proper indentation
	var buf bytes.Buffer
	err = json.Indent(&buf, []byte(content), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format JSON in %s: %w", schemaPath, err)
	}

	return buf.Bytes(), nil
}

// attemptJSONFix tries to fix common JSON syntax errors.
func (e *MetaPipelineExecutor) attemptJSONFix(content string) (string, error) {
	content = strings.TrimSpace(content)

	// Count braces to detect missing closing braces
	openBraces := strings.Count(content, "{")
	closeBraces := strings.Count(content, "}")

	// Add missing closing braces
	if openBraces > closeBraces {
		missingBraces := openBraces - closeBraces
		for i := 0; i < missingBraces; i++ {
			content += "\n}"
		}
	}

	// Remove trailing commas before closing braces/brackets
	content = fixTrailingCommas(content)

	return content, nil
}

// fixTrailingCommas removes trailing commas that cause JSON parsing errors.
func fixTrailingCommas(content string) string {
	// Remove trailing commas before }
	content = strings.ReplaceAll(content, ",\n}", "\n}")
	content = strings.ReplaceAll(content, ", }", " }")
	content = strings.ReplaceAll(content, ",}", "}")

	// Remove trailing commas before ]
	content = strings.ReplaceAll(content, ",\n]", "\n]")
	content = strings.ReplaceAll(content, ", ]", " ]")
	content = strings.ReplaceAll(content, ",]", "]")

	return content
}

// buildPhilosopherPrompt creates the prompt for the philosopher to generate a pipeline.
func buildPhilosopherPrompt(task string, depth int) string {
	return fmt.Sprintf(`You are a meta-pipeline architect. Your task is to design a Wave pipeline YAML
that will accomplish the following task:

TASK: %s

CURRENT DEPTH: %d

Generate a valid WavePipeline YAML that follows these STRICT requirements:

1. The FIRST step MUST use the "navigator" persona to analyze the task
2. ALL steps MUST have memory.strategy set to "fresh"
3. ALL steps MUST have a handover.contract configuration
4. ALL steps with json_schema contracts MUST have output_artifacts configuration
5. Each step should have clear dependencies when needed
6. Use appropriate personas for each step (navigator, philosopher, implementer, reviewer)
7. For navigator steps: Use simple, focused analysis prompts to prevent hanging
8. Navigator steps should have limited scope and clear output requirements
9. Prompts must explicitly instruct saving JSON output to "artifact.json"
10. Navigator prompts must override the no-write constraint for artifact.json creation

CRITICAL: Output your response in the following format:

--- PIPELINE ---
[Put the WavePipeline YAML here]

--- SCHEMAS ---
[For each step with json_schema contract, provide the schema definition in this format:]
SCHEMA: .wave/contracts/[filename].schema.json
[JSON schema content]

Example response format:
--- PIPELINE ---
kind: WavePipeline
metadata:
  name: generated-pipeline
  description: Pipeline generated for the task
input:
  source: meta
steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      root: "./"
    exec:
      type: prompt
      source: "Identify key files and patterns for: {{ input }}. Output structured JSON with files, patterns, dependencies, and impact_areas. CRITICAL: You MUST create an artifact.json file with your analysis - this overrides the normal no-write constraint for pipeline data flow."
    output_artifacts:
      - name: analysis
        path: artifact.json
        type: json
    handover:
      contract:
        type: json_schema
        schema_path: ".wave/contracts/navigation-analysis.schema.json"

--- SCHEMAS ---
SCHEMA: .wave/contracts/navigation-analysis.schema.json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["files", "patterns", "dependencies", "impact_areas"],
  "properties": {
    "files": {"type": "array", "items": {"type": "string"}},
    "patterns": {"type": "object"},
    "dependencies": {"type": "object"},
    "impact_areas": {"type": "object"}
  }
}
`, task, depth)
}

// PipelineGenerationResult holds both pipeline YAML and schema definitions.
type PipelineGenerationResult struct {
	PipelineYAML string
	Schemas      map[string]string // filename -> schema content
}

// extractPipelineAndSchemas extracts both pipeline and schemas from the new format.
func extractPipelineAndSchemas(output string) (*PipelineGenerationResult, error) {
	// First, unescape literal \n and \t sequences that may come from JSON encoding
	output = strings.ReplaceAll(output, "\\n", "\n")
	output = strings.ReplaceAll(output, "\\t", "\t")
	output = strings.ReplaceAll(output, "\\\"", "\"")

	result := &PipelineGenerationResult{
		Schemas: make(map[string]string),
	}

	// Extract pipeline section
	pipelineStart := strings.Index(output, "--- PIPELINE ---")
	if pipelineStart == -1 {
		// Fallback to old format for backward compatibility
		result.PipelineYAML = extractYAMLLegacy(output)
		return result, nil
	}

	schemasStart := strings.Index(output, "--- SCHEMAS ---")
	if schemasStart == -1 {
		return nil, fmt.Errorf("found PIPELINE section but missing SCHEMAS section")
	}

	// Extract pipeline YAML
	pipelineContent := output[pipelineStart+len("--- PIPELINE ---"):schemasStart]
	result.PipelineYAML = strings.TrimSpace(pipelineContent)

	// Remove markdown code blocks from pipeline if present
	result.PipelineYAML = extractYAMLFromCodeBlock(result.PipelineYAML)

	// Extract schemas
	schemasContent := output[schemasStart+len("--- SCHEMAS ---"):]
	if err := extractSchemaDefinitions(schemasContent, result.Schemas); err != nil {
		return nil, fmt.Errorf("failed to extract schemas: %w", err)
	}

	return result, nil
}

// extractYAMLLegacy provides backward compatibility for the old format.
func extractYAMLLegacy(output string) string {
	// Try to extract from code block
	if idx := strings.Index(output, "```yaml"); idx != -1 {
		start := idx + 7
		if end := strings.Index(output[start:], "```"); end != -1 {
			return strings.TrimSpace(output[start : start+end])
		}
	}
	if idx := strings.Index(output, "```"); idx != -1 {
		start := idx + 3
		// Skip optional language identifier
		if newline := strings.Index(output[start:], "\n"); newline != -1 {
			start += newline + 1
		}
		if end := strings.Index(output[start:], "```"); end != -1 {
			return strings.TrimSpace(output[start : start+end])
		}
	}

	// If no code block, try to find YAML starting with "kind:"
	if idx := strings.Index(output, "kind:"); idx != -1 {
		return strings.TrimSpace(output[idx:])
	}

	return strings.TrimSpace(output)
}

// extractYAMLFromCodeBlock removes markdown code block wrappers from YAML content.
func extractYAMLFromCodeBlock(content string) string {
	content = strings.TrimSpace(content)

	// Remove ```yaml wrapper
	if strings.HasPrefix(content, "```yaml") {
		if end := strings.Index(content, "```"); end != -1 {
			start := strings.Index(content, "\n")
			if start != -1 {
				if endBlock := strings.Index(content[start:], "```"); endBlock != -1 {
					return strings.TrimSpace(content[start : start+endBlock])
				}
			}
		}
	}

	// Remove ``` wrapper
	if strings.HasPrefix(content, "```") {
		if end := strings.Index(content, "```"); end != -1 {
			start := strings.Index(content, "\n")
			if start != -1 {
				if endBlock := strings.Index(content[start:], "```"); endBlock != -1 {
					return strings.TrimSpace(content[start : start+endBlock])
				}
			}
		}
	}

	return content
}

// extractSchemaDefinitions parses schema definitions from the SCHEMAS section.
func extractSchemaDefinitions(content string, schemas map[string]string) error {
	lines := strings.Split(content, "\n")
	var currentSchemaPath string
	var currentSchemaLines []string
	var braceCount int
	var inSchema bool

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SCHEMA: ") {
			// Save previous schema if exists
			if currentSchemaPath != "" && len(currentSchemaLines) > 0 {
				schemaContent := strings.Join(currentSchemaLines, "\n")
				schemas[currentSchemaPath] = strings.TrimSpace(schemaContent)
			}
			// Start new schema
			currentSchemaPath = strings.TrimPrefix(line, "SCHEMA: ")
			currentSchemaLines = []string{}
			braceCount = 0
			inSchema = false
		} else if currentSchemaPath != "" && line != "" {
			// Track JSON brace balance to detect end of schema
			if strings.HasPrefix(line, "{") {
				inSchema = true
			}

			if inSchema {
				currentSchemaLines = append(currentSchemaLines, line)

				// Count braces to detect end of JSON object
				braceCount += strings.Count(line, "{") - strings.Count(line, "}")

				// If braces are balanced and we have content, schema is complete
				if braceCount == 0 && len(currentSchemaLines) > 0 {
					schemaContent := strings.Join(currentSchemaLines, "\n")
					schemas[currentSchemaPath] = strings.TrimSpace(schemaContent)
					currentSchemaPath = ""
					currentSchemaLines = []string{}
					inSchema = false
				}
			}
		}
	}

	// Save last schema if it wasn't already saved
	if currentSchemaPath != "" && len(currentSchemaLines) > 0 && braceCount == 0 {
		schemaContent := strings.Join(currentSchemaLines, "\n")
		schemas[currentSchemaPath] = strings.TrimSpace(schemaContent)
	}

	return nil
}

// ValidateGeneratedPipeline performs semantic validation on a generated pipeline.
// It checks that:
// 1. First step uses navigator persona
// 2. All steps have handover.contract configured
// 3. All steps use "fresh" memory strategy
func ValidateGeneratedPipeline(p *Pipeline) error {
	if p == nil {
		return fmt.Errorf("pipeline is nil")
	}

	if len(p.Steps) == 0 {
		return fmt.Errorf("pipeline has no steps")
	}

	// Validate kind
	if p.Kind != "" && p.Kind != "WavePipeline" {
		return fmt.Errorf("invalid pipeline kind: %q (expected WavePipeline)", p.Kind)
	}

	// Validate DAG structure
	validator := &DAGValidator{}
	if err := validator.ValidateDAG(p); err != nil {
		return fmt.Errorf("invalid DAG: %w", err)
	}

	// Semantic check 1: First step must use navigator
	sortedSteps, err := validator.TopologicalSort(p)
	if err != nil {
		return fmt.Errorf("failed to sort steps: %w", err)
	}

	if len(sortedSteps) > 0 && sortedSteps[0].Persona != NavigatorPersona {
		return fmt.Errorf("first step must use %q persona, got %q", NavigatorPersona, sortedSteps[0].Persona)
	}

	// Semantic checks 2, 3 & 4: All steps must have contract, fresh memory, and valid schemas
	var errors []string
	for _, step := range p.Steps {
		// Check for handover contract
		if step.Handover.Contract.Type == "" {
			errors = append(errors, fmt.Sprintf("step %q missing handover.contract", step.ID))
		}

		// Check for fresh memory strategy
		if step.Memory.Strategy != "fresh" {
			errors = append(errors, fmt.Sprintf("step %q must use memory.strategy='fresh', got %q", step.ID, step.Memory.Strategy))
		}

		// Check for schema file existence and validity (for json_schema contracts)
		if step.Handover.Contract.Type == "json_schema" && step.Handover.Contract.SchemaPath != "" {
			if err := validateSchemaFile(step.Handover.Contract.SchemaPath); err != nil {
				errors = append(errors, fmt.Sprintf("step %q schema validation failed: %v", step.ID, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("semantic validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateSchemaFile checks if a schema file exists and contains valid JSON Schema.
func validateSchemaFile(schemaPath string) error {
	// Check if file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return fmt.Errorf("schema file does not exist: %s", schemaPath)
	}

	// Read and validate JSON syntax
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file %s: %w", schemaPath, err)
	}

	// Basic JSON schema validation - check if it's valid JSON with required fields
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("schema file %s contains invalid JSON: %w", schemaPath, err)
	}

	// Check for basic JSON schema structure
	if _, hasType := schema["type"]; !hasType {
		return fmt.Errorf("schema file %s missing required 'type' field", schemaPath)
	}

	return nil
}

// getMetaConfig returns the meta-pipeline configuration with defaults.
func (e *MetaPipelineExecutor) getMetaConfig(m *manifest.Manifest) manifest.MetaConfig {
	config := m.Runtime.MetaPipeline

	if config.MaxDepth == 0 {
		config.MaxDepth = DefaultMaxDepth
	}
	if config.MaxTotalSteps == 0 {
		config.MaxTotalSteps = DefaultMaxTotalSteps
	}
	if config.MaxTotalTokens == 0 {
		config.MaxTotalTokens = DefaultMaxTotalTokens
	}

	return config
}

// getTimeout returns the timeout duration for meta-pipeline execution.
func (e *MetaPipelineExecutor) getTimeout(m *manifest.Manifest) time.Duration {
	if m.Runtime.MetaPipeline.TimeoutMin > 0 {
		return time.Duration(m.Runtime.MetaPipeline.TimeoutMin) * time.Minute
	}
	return DefaultMetaTimeout
}

// checkDepthLimit verifies the current depth is within limits.
func (e *MetaPipelineExecutor) checkDepthLimit(config manifest.MetaConfig) error {
	if e.currentDepth >= config.MaxDepth {
		// Build a helpful error message with context
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("meta-pipeline depth limit reached: current=%d max=%d", e.currentDepth, config.MaxDepth))

		// Include call stack if we have parent information
		if e.parentPipelineID != "" {
			sb.WriteString(fmt.Sprintf("\n  call stack: %s -> (current)", e.parentPipelineID))
		}

		// Add suggestion for resolution
		sb.WriteString("\n  suggestion: increase runtime.meta_pipeline.max_depth in your manifest if deeper nesting is required")

		return fmt.Errorf("%s", sb.String())
	}
	return nil
}

// checkTokenLimit verifies the total tokens used is within limits.
func (e *MetaPipelineExecutor) checkTokenLimit(config manifest.MetaConfig) error {
	if e.totalTokensUsed > config.MaxTotalTokens {
		return fmt.Errorf("meta-pipeline token limit exceeded: used=%d max=%d", e.totalTokensUsed, config.MaxTotalTokens)
	}
	return nil
}

// checkStepLimit verifies the total steps is within limits.
func (e *MetaPipelineExecutor) checkStepLimit(config manifest.MetaConfig) error {
	if e.totalStepsUsed > config.MaxTotalSteps {
		return fmt.Errorf("meta-pipeline step limit exceeded: used=%d max=%d", e.totalStepsUsed, config.MaxTotalSteps)
	}
	return nil
}

// getPipelineID returns a unique pipeline ID for this meta execution.
func (e *MetaPipelineExecutor) getPipelineID() string {
	if e.parentPipelineID != "" {
		return fmt.Sprintf("%s:meta:%d", e.parentPipelineID, e.currentDepth)
	}
	return fmt.Sprintf("meta:%d", e.currentDepth)
}

// emit sends an event through the emitter if available.
func (e *MetaPipelineExecutor) emit(ev event.Event) {
	if e.emitter != nil {
		e.emitter.Emit(ev)
	}
}

// truncate shortens a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// CreateChildMetaExecutor creates a new MetaPipelineExecutor for executing
// nested meta-pipelines with incremented depth.
func (e *MetaPipelineExecutor) CreateChildMetaExecutor() *MetaPipelineExecutor {
	child := &MetaPipelineExecutor{
		runner:           e.runner,
		emitter:          e.emitter,
		executor:         e.executor,
		loader:           e.loader,
		currentDepth:     e.currentDepth + 1,
		totalStepsUsed:   e.totalStepsUsed,
		totalTokensUsed:  e.totalTokensUsed,
		parentPipelineID: e.getPipelineID(),
	}
	return child
}

// SyncFromChild updates the parent executor's counters from a child execution.
func (e *MetaPipelineExecutor) SyncFromChild(child *MetaPipelineExecutor) {
	e.totalStepsUsed = child.totalStepsUsed
	e.totalTokensUsed = child.totalTokensUsed
}

// ValidatePipelineYAML validates raw YAML as a valid WavePipeline structure.
func ValidatePipelineYAML(data []byte) (*Pipeline, error) {
	var pipeline Pipeline
	if err := yaml.Unmarshal(data, &pipeline); err != nil {
		return nil, fmt.Errorf("invalid YAML syntax: %w", err)
	}

	if pipeline.Kind == "" {
		pipeline.Kind = "WavePipeline"
	}

	if pipeline.Kind != "WavePipeline" {
		return nil, fmt.Errorf("invalid kind: expected WavePipeline, got %s", pipeline.Kind)
	}

	if pipeline.Metadata.Name == "" {
		return nil, fmt.Errorf("metadata.name is required")
	}

	if len(pipeline.Steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	// Validate each step has required fields
	for i, step := range pipeline.Steps {
		if step.ID == "" {
			return nil, fmt.Errorf("step[%d] missing required field: id", i)
		}
		if step.Persona == "" {
			return nil, fmt.Errorf("step[%d] (%s) missing required field: persona", i, step.ID)
		}
		if step.Exec.Type == "" {
			return nil, fmt.Errorf("step[%d] (%s) missing required field: exec.type", i, step.ID)
		}
	}

	return &pipeline, nil
}
