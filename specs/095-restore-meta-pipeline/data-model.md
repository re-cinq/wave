# Data Model: Restore and Stabilize `wave meta` Dynamic Pipeline Generation

**Date**: 2026-03-16
**Spec**: `specs/095-restore-meta-pipeline/spec.md`

## Existing Entities (No Changes)

### MetaPipelineExecutor (`internal/pipeline/meta.go:31`)

```go
type MetaPipelineExecutor struct {
    runner           adapter.AdapterRunner
    emitter          event.EventEmitter
    executor         PipelineExecutor      // child pipeline executor (injected)
    loader           *YAMLPipelineLoader
    currentDepth     int
    totalStepsUsed   int
    totalTokensUsed  int
    parentPipelineID string
}
```

No structural changes needed. All fields serve their purpose.

### MetaExecutionResult (`internal/pipeline/meta.go:81`)

```go
type MetaExecutionResult struct {
    GeneratedPipeline *Pipeline
    TotalSteps        int
    TotalTokens       int
    Depth             int
    ChildResults      []MetaExecutionResult
}
```

No changes needed.

### PipelineGenerationResult (`internal/pipeline/meta.go:559`)

```go
type PipelineGenerationResult struct {
    PipelineYAML string
    Schemas      map[string]string // filepath -> schema JSON content
}
```

No changes needed.

### MetaConfig (`internal/manifest/types.go:166`)

```go
type MetaConfig struct {
    MaxDepth       int `yaml:"max_depth,omitempty"`
    MaxTotalSteps  int `yaml:"max_total_steps,omitempty"`
    MaxTotalTokens int `yaml:"max_total_tokens,omitempty"`
    TimeoutMin     int `yaml:"timeout_minutes,omitempty"`
}
```

No changes needed. Defaults applied in `getMetaConfig()`.

### MetaOptions (`cmd/wave/commands/meta.go:20`)

```go
type MetaOptions struct {
    Save     string
    Manifest string
    Mock     bool
    DryRun   bool
    Output   OutputConfig
    Model    string
}
```

No changes needed. All CLI flags already wired.

## Modified Functions

### ValidateGeneratedPipeline — Add Manifest-Aware Validation

**Current signature**: `ValidateGeneratedPipeline(p *Pipeline) error`

**New signature**: `ValidateGeneratedPipeline(p *Pipeline, opts ...ValidationOption) error`

New option type:

```go
type ValidationOption func(*validationConfig)

type validationConfig struct {
    manifest *manifest.Manifest
}

func WithManifest(m *manifest.Manifest) ValidationOption {
    return func(c *validationConfig) { c.manifest = m }
}
```

When manifest is provided, validation additionally checks:
- All step personas exist in `m.Personas`
- Adapter for each persona exists in `m.Adapters`

**Impact**: Backward compatible — existing callers without options continue to work.

### normalizeGeneratedPipeline — New Function

```go
func normalizeGeneratedPipeline(p *Pipeline) {
    for i := range p.Steps {
        step := &p.Steps[i]
        if step.Handover.Contract.Type == "json_schema" && len(step.OutputArtifacts) == 0 {
            step.OutputArtifacts = []OutputArtifact{{
                Name: step.ID + "-output",
                Path: ".wave/artifact.json",
                Type: "json",
            }}
        }
    }
}
```

Called after parsing, before validation. Ensures FR-011 compliance.

### MetaPipelineExecutor.Execute — Add Internal Timeout

Wrap context with timeout from manifest config inside `Execute()`:

```go
func (e *MetaPipelineExecutor) Execute(ctx context.Context, task string, m *manifest.Manifest) (*MetaExecutionResult, error) {
    config := e.getMetaConfig(m)
    timeout := e.getTimeout(m)
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    // ... rest of existing logic
}
```

**Impact**: CLI layer can remove its redundant timeout wrapping, or keep it as a secondary safety net.

## Data Flow

```
User → "wave meta <task>" → CLI (meta.go)
  ↓
MetaOptions parsed → manifest loaded → adapter resolved
  ↓
MetaPipelineExecutor.GenerateOnly() or .Execute()
  ↓
invokePhilosopherWithSchemas()
  → buildPhilosopherPrompt(task, depth)
  → adapter.Run(ctx, cfg)  [philosopher persona]
  → extractPipelineAndSchemas(output)
  → saveSchemaFiles(schemas)
  ↓
normalizeGeneratedPipeline(pipeline)   ← NEW
  ↓
ValidateGeneratedPipeline(pipeline, WithManifest(m))  ← ENHANCED
  ↓
[if Execute] → childExecutor.Execute(ctx, pipeline, m, task)
  → standard pipeline execution with contract validation
  ↓
MetaExecutionResult
```

## Error Taxonomy

| Error | Source | User Message |
|-------|--------|-------------|
| Missing philosopher persona | `invokePhilosopherWithSchemas` | "philosopher persona not found in manifest" |
| Missing adapter | `invokePhilosopherWithSchemas` | "adapter %q for philosopher not found" |
| Malformed YAML | `loader.Unmarshal` | "failed to parse generated pipeline YAML: %w" + raw YAML dump |
| Circular dependencies | `DAGValidator.ValidateDAG` | "invalid DAG: %w" |
| First step not navigator | `ValidateGeneratedPipeline` | "first step must use navigator persona" |
| Missing contract | `ValidateGeneratedPipeline` | "step %q missing handover.contract" |
| Invalid schema file | `validateSchemaFile` | "schema file %s contains invalid JSON" |
| Unknown persona | `ValidateGeneratedPipeline` | "step %q references unknown persona %q" ← NEW |
| Depth limit | `checkDepthLimit` | "meta-pipeline depth limit reached" + call stack |
| Step limit | `checkStepLimit` | "meta-pipeline step limit exceeded" |
| Token limit | `checkTokenLimit` | "meta-pipeline token limit exceeded" |
| Timeout | context cancellation | "context deadline exceeded" |
