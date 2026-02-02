package pipeline

import (
	"context"
	"fmt"
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
	persona := m.GetPersona(PhilosopherPersona)
	if persona == nil {
		return "", 0, fmt.Errorf("philosopher persona not found in manifest")
	}

	adapterDef := m.GetAdapter(persona.Adapter)
	if adapterDef == nil {
		return "", 0, fmt.Errorf("adapter %q for philosopher not found", persona.Adapter)
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

	result, err := e.runner.Run(ctx, cfg)
	if err != nil {
		return "", 0, fmt.Errorf("philosopher adapter execution failed: %w", err)
	}

	// Read stdout from the adapter result
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, _ := result.Stdout.Read(buf)
	output := string(buf[:n])

	// Extract YAML from the output (may be wrapped in markdown code blocks)
	yamlContent := extractYAML(output)

	return yamlContent, result.TokensUsed, nil
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
4. Each step should have clear dependencies when needed
5. Use appropriate personas for each step (navigator, philosopher, implementer, reviewer)

Output ONLY valid YAML for a WavePipeline. Do not include any explanations or markdown formatting.

Example structure:
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
      source: "Analyze: {{ input }}"
    handover:
      contract:
        type: json_schema
        schema_path: ".wave/contracts/analysis.schema.json"
  - id: implement
    persona: implementer
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: context
    workspace:
      root: "./"
    exec:
      type: prompt
      source: "Implement based on analysis"
    handover:
      contract:
        type: test_suite
        command: "go test ./..."
`, task, depth)
}

// extractYAML extracts YAML content from potentially markdown-wrapped output.
func extractYAML(output string) string {
	// First, unescape literal \n and \t sequences that may come from JSON encoding
	output = strings.ReplaceAll(output, "\\n", "\n")
	output = strings.ReplaceAll(output, "\\t", "\t")
	output = strings.ReplaceAll(output, "\\\"", "\"")

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

	// Semantic checks 2 & 3: All steps must have contract and fresh memory
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
	}

	if len(errors) > 0 {
		return fmt.Errorf("semantic validation failed:\n  - %s", strings.Join(errors, "\n  - "))
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
