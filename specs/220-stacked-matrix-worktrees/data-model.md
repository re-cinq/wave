# Data Model: Stacked Worktrees for Dependent Matrix Child Pipelines

**Feature**: #220 | **Date**: 2026-03-16

## Entity Changes

### MatrixStrategy (Modified)

**File**: `internal/pipeline/types.go:302`

```go
type MatrixStrategy struct {
    Type           string `yaml:"type"`
    ItemsSource    string `yaml:"items_source"`
    ItemKey        string `yaml:"item_key"`
    MaxConcurrency int    `yaml:"max_concurrency,omitempty"`
    ItemIDKey      string `yaml:"item_id_key,omitempty"`
    DependencyKey  string `yaml:"dependency_key,omitempty"`
    ChildPipeline  string `yaml:"child_pipeline,omitempty"`
    InputTemplate  string `yaml:"input_template,omitempty"`
    Stacked        bool   `yaml:"stacked,omitempty"`         // NEW: Enable branch stacking between tiers
}
```

**Impact**: YAML schema change. Backward compatible — omitting field defaults to `false`.

### MatrixResult (Modified)

**File**: `internal/pipeline/matrix.go:21`

```go
type MatrixResult struct {
    ItemIndex     int
    Item          interface{}
    WorkspacePath string
    Output        map[string]interface{}
    ModifiedFiles []string
    Error         error
    Skipped       bool
    SkipReason    string
    ItemID        string
    OutputBranch  string  // NEW: Branch name from completed child pipeline
}
```

**Impact**: Internal struct only. No YAML schema change.

### TierContext (New)

**File**: `internal/pipeline/matrix.go` (new type)

```go
// TierContext tracks accumulated branch state during stacked tier execution.
// Maps item IDs to their output branch names for downstream tier resolution.
type TierContext struct {
    OutputBranches map[string]string // itemID -> output branch name
}
```

**Impact**: Internal struct. Used only within `tieredExecution` to track cross-tier branch state.

### IntegrationBranch (New — Behavioral, Not a Struct)

Integration branches are not modeled as a separate type. They are created via `worktree.Manager` and tracked by name for cleanup. The naming convention is:

```
integration/<pipeline-id>/<item-id>
```

Tracked in a `cleanupBranches []string` slice within `tieredExecution`.

### DefaultPipelineExecutor (Modified)

**File**: `internal/pipeline/executor.go:130`

Add field and method to expose child pipeline execution state:

```go
type DefaultPipelineExecutor struct {
    // ... existing fields ...
    lastExecution *PipelineExecution  // NEW: Most recent execution for child state access
}

// LastExecution returns the most recently executed pipeline's execution state.
func (e *DefaultPipelineExecutor) LastExecution() *PipelineExecution {
    return e.lastExecution
}
```

**Impact**: Internal API. No YAML or user-facing change.

## Data Flow

```
tieredExecution()
  │
  ├── Tier 0: items run with pipeline's default base branch
  │     └── Each result captures OutputBranch from child executor
  │
  ├── TierContext collects: {itemID: outputBranch, ...}
  │
  ├── Tier 1 (stacked=true): resolve base branch per item
  │     ├── Single parent dep → use parent's OutputBranch as base
  │     ├── Multi parent deps → create integration/<pipeline>/<item> branch
  │     │     └── Sequential merge of all parent branches
  │     └── Each result captures OutputBranch
  │
  ├── Tier N: same pattern
  │
  └── Cleanup: delete integration branches
```

## Relationships

```
MatrixStrategy --[configures]--> tieredExecution()
MatrixResult   --[carries]-----> OutputBranch (from child executor)
TierContext    --[maps]--------> itemID → branch name (cross-tier state)
WorktreeInfo   --[tracks]------> created worktrees (existing behavior)
```
