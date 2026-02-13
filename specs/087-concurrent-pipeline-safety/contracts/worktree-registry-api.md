# Contract: Worktree Registry API

**Package**: `internal/worktree`

## Public API Contract

### Types

```go
// WorktreeEntry represents a single worktree created during pipeline execution.
type WorktreeEntry struct {
    StepID       string // Pipeline step that created this worktree
    WorktreePath string // Absolute filesystem path to the worktree
    RepoRoot     string // Canonical repository root path
}

// WorktreeRegistry tracks worktrees created during a single pipeline execution
// for targeted cleanup on completion or failure.
type WorktreeRegistry struct {
    // unexported fields
}
```

### Constructor

```go
// NewWorktreeRegistry creates an empty worktree registry.
func NewWorktreeRegistry() *WorktreeRegistry
```

### Operations

```go
// Register adds a worktree entry to the registry.
// Safe for concurrent use.
func (r *WorktreeRegistry) Register(entry WorktreeEntry)

// Entries returns a copy of all registered worktree entries.
// Safe for concurrent use.
func (r *WorktreeRegistry) Entries() []WorktreeEntry

// Count returns the number of registered worktrees.
// Safe for concurrent use.
func (r *WorktreeRegistry) Count() int
```

## Thread Safety

All methods are safe for concurrent use (internal mutex).

## Lifecycle

1. Created by `Execute()` when initializing `PipelineExecution`.
2. Entries added by `createStepWorkspace()` after successful worktree creation.
3. Iterated by `cleanupWorktrees()` at pipeline completion or failure.
4. Garbage collected with `PipelineExecution` after cleanup.

## Integration with Pipeline Executor

The `cleanupWorktrees` function in `executor.go` changes from:

```go
// BEFORE: scan WorkspacePaths for __worktree_repo_root suffix
for key, repoRoot := range execution.WorkspacePaths {
    if !strings.HasSuffix(key, "__worktree_repo_root") {
        continue
    }
    ...
}
```

To:

```go
// AFTER: iterate typed registry
for _, entry := range execution.Worktrees.Entries() {
    mgr, err := worktree.NewManager(entry.RepoRoot)
    if err != nil { ... }
    if err := mgr.Remove(ctx, entry.WorktreePath); err != nil { ... }
}
```
