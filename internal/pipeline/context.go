package pipeline

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
)

// unresolvedProjectVarRe matches unresolved {{ project.* }} template placeholders.
// These are stripped after all known variables have been substituted so that
// missing project config fields resolve to empty strings instead of leaking
// literal mustache syntax into prompts and contract commands.
var (
	unresolvedProjectVarRe = regexp.MustCompile(`\{\{\s*(?:project|ontology)\.\w+(?:\.\w+)*\s*\}\}`)
	threeDigitPrefixRe     = regexp.MustCompile(`^(\d{3})-`)
	numericPrefixRe        = regexp.MustCompile(`(\d+)[-_]`)
	invalidPathCharRe      = regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	consecutiveDashRe      = regexp.MustCompile(`-+`)
)

// PipelineContext holds dynamic variables for template resolution during pipeline execution
type PipelineContext struct {
	mu              sync.Mutex               `json:"-"` // protects map access during concurrent steps
	BranchName      string                   `json:"branch_name"`
	FeatureNum      string                   `json:"feature_num"`
	SpeckitMode     bool                     `json:"speckit_mode"`
	PipelineID      string                   `json:"pipeline_id"`   // Runtime ID with hash suffix
	PipelineName    string                   `json:"pipeline_name"` // Logical pipeline name
	StepID          string                   `json:"step_id"`
	Input           string                   `json:"input,omitempty"` // Pipeline input for {{ input }} resolution
	CustomVariables map[string]string        `json:"custom_variables,omitempty"`
	ArtifactPaths   map[string]string        `json:"artifact_paths,omitempty"` // Artifact name -> path for template resolution
	GateDecisions   map[string]*GateDecision `json:"gate_decisions,omitempty"` // stepID -> gate decision
}

// NewPipelineContext creates a new pipeline context with auto-detected values.
// pipelineID is the runtime ID (with hash suffix), pipelineName is the logical name.
func NewPipelineContext(pipelineID, pipelineName, stepID string) *PipelineContext {
	ctx := &PipelineContext{
		PipelineID:      pipelineID,
		PipelineName:    pipelineName,
		StepID:          stepID,
		CustomVariables: make(map[string]string),
	}

	// Auto-detect git branch name
	if branchName, err := getCurrentGitBranch(); err == nil {
		ctx.BranchName = branchName

		// Try to extract feature number from branch name (###-feature-name pattern)
		if featureNum := extractFeatureNumber(branchName); featureNum != "" {
			ctx.FeatureNum = featureNum
			ctx.SpeckitMode = true
		}
	}

	return ctx
}

// ResolvePlaceholders replaces template placeholders in a string with actual values
func (ctx *PipelineContext) ResolvePlaceholders(template string) string {
	if template == "" {
		return template
	}

	result := template

	// replaceBoth replaces both {{key}} and {{ key }} variants
	replaceBoth := func(s, key, value string) string {
		s = strings.ReplaceAll(s, "{{"+key+"}}", value)
		s = strings.ReplaceAll(s, "{{ "+key+" }}", value)
		return s
	}

	// Take a snapshot of maps under lock to avoid holding the lock during string ops
	ctx.mu.Lock()
	artifactPathsCopy := make(map[string]string, len(ctx.ArtifactPaths))
	for k, v := range ctx.ArtifactPaths {
		artifactPathsCopy[k] = v
	}
	customVarsCopy := make(map[string]string, len(ctx.CustomVariables))
	for k, v := range ctx.CustomVariables {
		customVarsCopy[k] = v
	}
	ctx.mu.Unlock()

	// Replace pipeline context variables (both spaced and unspaced)
	// Support both prefixed (pipeline_context.pipeline_id) and bare (pipeline_id) forms
	result = replaceBoth(result, "pipeline_context.branch_name", ctx.BranchName)
	result = replaceBoth(result, "pipeline_context.feature_num", ctx.FeatureNum)
	result = replaceBoth(result, "pipeline_context.pipeline_id", ctx.PipelineID)
	result = replaceBoth(result, "pipeline_context.pipeline_name", ctx.PipelineName)
	result = replaceBoth(result, "pipeline_context.step_id", ctx.StepID)

	// Also resolve bare variable names ({{ pipeline_id }}, {{ step_id }}, etc.)
	result = replaceBoth(result, "pipeline_id", ctx.PipelineID)
	result = replaceBoth(result, "pipeline_name", ctx.PipelineName)
	result = replaceBoth(result, "step_id", ctx.StepID)
	result = replaceBoth(result, "branch_name", ctx.BranchName)
	result = replaceBoth(result, "feature_num", ctx.FeatureNum)
	if ctx.Input != "" {
		result = replaceBoth(result, "input", ctx.Input)
	}

	// Traceability aliases: {{ run.id }} resolves to pipeline_id for artifact correlation
	result = replaceBoth(result, "run.id", ctx.PipelineID)
	result = replaceBoth(result, "run.name", ctx.PipelineName)

	// Replace artifact path references ({{ artifacts.<name> }})
	for name, path := range artifactPathsCopy {
		result = replaceBoth(result, "artifacts."+name, path)
	}

	// Replace custom variables (support both {{key}} and {{ key }} formats)
	for key, value := range customVarsCopy {
		result = replaceBoth(result, key, value)
	}

	// Strip unresolved {{ project.* }} placeholders so they don't leak into
	// prompts or contract commands when a project field is not configured.
	result = unresolvedProjectVarRe.ReplaceAllString(result, "")

	return result
}

// newContextWithProject creates a PipelineContext and injects project variables from the manifest.
func newContextWithProject(pipelineID, pipelineName, stepID string, m *manifest.Manifest) *PipelineContext {
	ctx := NewPipelineContext(pipelineID, pipelineName, stepID)
	if m != nil && m.Project != nil {
		for k, v := range m.Project.ProjectVars() {
			ctx.SetCustomVariable(k, v)
		}
	}
	if m != nil && m.Ontology != nil {
		for k, v := range m.Ontology.OntologyVars() {
			ctx.SetCustomVariable(k, v)
		}
	}
	return ctx
}

// InjectForgeVariables populates forge.* template variables in the context.
// All variables are set atomically under a single lock acquisition to
// prevent interleaved writes from concurrent callers.
func InjectForgeVariables(ctx *PipelineContext, info forge.ForgeInfo) {
	ctx.setCustomVariablesBatch(map[string]string{
		"forge.type":       string(info.Type),
		"forge.host":       info.Host,
		"forge.owner":      info.Owner,
		"forge.repo":       info.Repo,
		"forge.cli_tool":   info.CLITool,
		"forge.prefix":     info.PipelinePrefix,
		"forge.pr_term":    info.PRTerm,
		"forge.pr_command": info.PRCommand,
	})
}

// SetCustomVariable adds a custom template variable
func (ctx *PipelineContext) SetCustomVariable(key, value string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.CustomVariables == nil {
		ctx.CustomVariables = make(map[string]string)
	}
	ctx.CustomVariables[key] = value
}

// setCustomVariablesBatch atomically sets multiple custom variables under a
// single lock acquisition. This prevents interleaved writes from concurrent
// goroutines that would leave the context in an inconsistent state (e.g.,
// forge.type from one forge and forge.cli_tool from another).
func (ctx *PipelineContext) setCustomVariablesBatch(vars map[string]string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.CustomVariables == nil {
		ctx.CustomVariables = make(map[string]string)
	}
	for k, v := range vars {
		ctx.CustomVariables[k] = v
	}
}

// SetArtifactPath registers an artifact path for template resolution.
// The artifact will be accessible via {{ artifacts.<name> }} or {{ artifacts.<name> }} syntax.
func (ctx *PipelineContext) SetArtifactPath(name, path string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.ArtifactPaths == nil {
		ctx.ArtifactPaths = make(map[string]string)
	}
	ctx.ArtifactPaths[name] = path
}

// GetArtifactPath returns the registered path for an artifact, or empty string if not found.
func (ctx *PipelineContext) GetArtifactPath(name string) string {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.ArtifactPaths == nil {
		return ""
	}
	return ctx.ArtifactPaths[name]
}

// SetGateDecision records a gate decision for template variable resolution.
// After calling this, downstream steps can resolve {{ gate.<stepID>.choice }},
// {{ gate.<stepID>.text }}, and {{ gate.<stepID>.timestamp }}.
func (ctx *PipelineContext) SetGateDecision(stepID string, decision *GateDecision) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.GateDecisions == nil {
		ctx.GateDecisions = make(map[string]*GateDecision)
	}
	ctx.GateDecisions[stepID] = decision

	// Also set as custom variables for template resolution.
	// Strip Go template delimiters from freeform text to prevent injection
	// into downstream template resolution.
	if ctx.CustomVariables == nil {
		ctx.CustomVariables = make(map[string]string)
	}
	prefix := "gate." + stepID
	ctx.CustomVariables[prefix+".choice"] = stripTemplateDelimiters(decision.Label)
	ctx.CustomVariables[prefix+".key"] = stripTemplateDelimiters(decision.Choice)
	ctx.CustomVariables[prefix+".text"] = stripTemplateDelimiters(decision.Text)
	ctx.CustomVariables[prefix+".timestamp"] = decision.Timestamp.Format(time.RFC3339)
	ctx.CustomVariables[prefix+".target"] = stripTemplateDelimiters(decision.Target)
}

// MergeFrom merges child pipeline context variables and artifact paths into the parent.
// Artifact paths from the child are namespaced with the given prefix to avoid collisions.
// Custom variables from the child overwrite parent values on conflict (last-writer-wins).
// Snapshot the child under its own lock first, then merge into the parent to avoid
// holding two locks simultaneously (which could deadlock if called bidirectionally).
func (ctx *PipelineContext) MergeFrom(child *PipelineContext, namespace string) {
	if child == nil {
		return
	}

	// Snapshot child state under child's lock only
	child.mu.Lock()
	childVars := make(map[string]string, len(child.CustomVariables))
	for k, v := range child.CustomVariables {
		childVars[k] = v
	}
	childArtifacts := make(map[string]string, len(child.ArtifactPaths))
	for k, v := range child.ArtifactPaths {
		childArtifacts[k] = v
	}
	child.mu.Unlock()

	// Merge snapshot into parent under parent's lock
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if ctx.CustomVariables == nil {
		ctx.CustomVariables = make(map[string]string)
	}
	if ctx.ArtifactPaths == nil {
		ctx.ArtifactPaths = make(map[string]string)
	}

	for k, v := range childVars {
		ctx.CustomVariables[k] = v
	}

	for name, path := range childArtifacts {
		key := name
		if namespace != "" {
			key = namespace + "." + name
		}
		ctx.ArtifactPaths[key] = path
	}
}

// IsSpeckitCompatible returns true if the current context appears to be for Speckit workflows
func (ctx *PipelineContext) IsSpeckitCompatible() bool {
	return ctx.SpeckitMode ||
		strings.Contains(ctx.BranchName, "-") ||
		ctx.FeatureNum != ""
}

// GetSpeckitPath generates a Speckit-compatible path for the current context
func (ctx *PipelineContext) GetSpeckitPath(filename string) string {
	// Only generate Speckit paths when explicitly in Speckit mode or when we have clear indicators
	if !ctx.SpeckitMode && ctx.FeatureNum == "" {
		// Check if branch indicates Speckit workflow
		if strings.Contains(ctx.BranchName, "-") {
			// This looks like it might be Speckit, but let's generate a path anyway
			featureDir := "999-" + sanitizeBranchName(ctx.BranchName)
			return "specs/" + featureDir + "/" + filename
		}
		return filename
	}

	// Use feature number if available, otherwise derive from branch
	featureDir := ctx.FeatureNum
	if featureDir == "" && ctx.BranchName != "" {
		// Generate a feature number from branch name if not present
		if num := extractFeatureNumber(ctx.BranchName); num != "" {
			featureDir = num
		} else if ctx.SpeckitMode {
			// Generate a simple numeric prefix for non-standard branch names when in Speckit mode
			featureDir = "999-" + sanitizeBranchName(ctx.BranchName)
		}
	}

	if featureDir == "" {
		featureDir = "000-feature"
	}

	return "specs/" + featureDir + "/" + filename
}

// ToTemplateVars converts the context to a map for use with existing template systems
func (ctx *PipelineContext) ToTemplateVars() map[string]string {
	vars := map[string]string{
		"pipeline_id":                    ctx.PipelineID,
		"pipeline_name":                  ctx.PipelineName,
		"step_id":                        ctx.StepID,
		"branch_name":                    ctx.BranchName,
		"feature_num":                    ctx.FeatureNum,
		"pipeline_context.branch_name":   ctx.BranchName,
		"pipeline_context.feature_num":   ctx.FeatureNum,
		"pipeline_context.pipeline_id":   ctx.PipelineID,
		"pipeline_context.pipeline_name": ctx.PipelineName,
		"pipeline_context.step_id":       ctx.StepID,
		"run.id":                         ctx.PipelineID,
		"run.name":                       ctx.PipelineName,
	}

	// Snapshot maps under lock
	ctx.mu.Lock()
	for key, value := range ctx.CustomVariables {
		vars[key] = value
	}
	for name, path := range ctx.ArtifactPaths {
		vars["artifacts."+name] = path
	}
	ctx.mu.Unlock()

	return vars
}

// getCurrentGitBranch gets the current git branch name
func getCurrentGitBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	branchName := strings.TrimSpace(string(output))
	return branchName, nil
}

// extractFeatureNumber extracts feature number from branch name (supports ###-name format)
func extractFeatureNumber(branchName string) string {
	// Match patterns like "018-enhanced-progress", "001-feature-name", etc.
	matches := threeDigitPrefixRe.FindStringSubmatch(branchName)
	if len(matches) > 1 {
		return branchName // Return full branch name as feature identifier
	}

	// Try other common patterns like "feature/123-name"
	matches2 := numericPrefixRe.FindStringSubmatch(branchName)
	if len(matches2) > 1 {
		// Pad to 3 digits
		num, _ := strconv.Atoi(matches2[1])
		return strings.ReplaceAll(branchName, matches2[1], padNumber(num))
	}

	return ""
}

// sanitizeBranchName removes invalid characters from branch names for use in paths
func sanitizeBranchName(branchName string) string {
	// Replace invalid path characters
	sanitized := invalidPathCharRe.ReplaceAllString(branchName, "-")

	// Remove consecutive dashes
	sanitized = consecutiveDashRe.ReplaceAllString(sanitized, "-")

	// Trim leading/trailing dashes
	sanitized = strings.Trim(sanitized, "-")

	// Limit length
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}

	return sanitized
}

// padNumber pads a number to 3 digits with leading zeros
func padNumber(num int) string {
	return fmt.Sprintf("%03d", num)
}

// ResolveArtifactPath resolves an artifact path using the pipeline context
func (ctx *PipelineContext) ResolveArtifactPath(artifactDef ArtifactDef) string {
	path := ctx.ResolvePlaceholders(artifactDef.Path)

	// If this looks like a Speckit workflow and the path is a simple filename,
	// try to place it in the appropriate Speckit directory
	if ctx.IsSpeckitCompatible() && !strings.Contains(path, "/") {
		if strings.HasSuffix(path, ".md") {
			return ctx.GetSpeckitPath(path)
		}
	}

	return path
}

// ResolveContractSource resolves a contract source path using the pipeline context
func (ctx *PipelineContext) ResolveContractSource(contractCfg ContractConfig) string {
	if contractCfg.Source != "" {
		return ctx.ResolvePlaceholders(contractCfg.Source)
	}
	return ""
}

// stripTemplateDelimiters removes Go template delimiters ({{ and }}) from a
// string to prevent template injection when user-supplied values (e.g. gate
// freeform text) are stored as custom template variables.
func stripTemplateDelimiters(s string) string {
	s = strings.ReplaceAll(s, "{{", "")
	s = strings.ReplaceAll(s, "}}", "")
	return s
}
