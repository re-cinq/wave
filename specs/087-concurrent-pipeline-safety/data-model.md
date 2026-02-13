# Data Model: Concurrent Pipeline Safety

**Branch**: `087-concurrent-pipeline-safety` | **Date**: 2026-02-13

## Entity Definitions

### 1. `repoLock` (new — `internal/worktree/`)

Channel-based semaphore providing context-aware lock acquisition for a single repository.

```go
// repoLock provides a context-aware mutex for serializing git worktree
// operations on a specific repository.
type repoLock struct {
    sem chan struct{} // Buffered channel of capacity 1 acting as a semaphore
}
```

**Fields**:
| Field | Type | Description |
|-------|------|-------------|
| `sem` | `chan struct{}` | Buffered channel (cap 1). Send = lock, receive = unlock |

**Operations**:
| Method | Signature | Description |
|--------|-----------|-------------|
| `LockWithContext` | `(ctx context.Context) error` | Acquires lock or returns error on context cancellation/timeout |
| `Unlock` | `()` | Releases the lock. Must be called exactly once per successful `LockWithContext` |

**Invariants**:
- A goroutine that successfully acquires the lock MUST release it (enforced via `defer` at call sites).
- `Unlock()` on an already-unlocked `repoLock` will block — callers must never double-unlock.

---

### 2. `repoLocks` (new — package-level in `internal/worktree/`)

Package-level registry mapping canonical repository paths to their locks.

```go
// repoLocks is a package-level registry of per-repository locks.
// Keys are canonical (absolute, symlink-resolved) repository root paths.
var repoLocks sync.Map // map[string]*repoLock
```

**Operations**:
| Function | Signature | Description |
|----------|-----------|-------------|
| `getRepoLock` | `(repoRoot string) (*repoLock, error)` | Returns existing lock or atomically creates a new one for the given canonical path |
| `canonicalPath` | `(path string) (string, error)` | Resolves symlinks and returns absolute path |

**Invariants**:
- `getRepoLock` is safe for concurrent use — uses `sync.Map.LoadOrStore` for atomic upsert.
- Different path representations of the same repository resolve to the same lock.

---

### 3. `Manager` (modified — `internal/worktree/worktree.go`)

Extended to participate in repository-scoped coordination.

```go
type Manager struct {
    repoRoot    string
    lockTimeout time.Duration // Default: 30s. Configurable via option.
}
```

**Changes from current**:
| Change | From | To | Rationale |
|--------|------|----|-----------|
| Remove `mu sync.Mutex` | Per-instance mutex | Repository-scoped lock via `repoLocks` | FR-001: Cross-instance coordination |
| Add `lockTimeout` | N/A | `time.Duration` (default 30s) | SC-007: Configurable lock timeout |
| `Create` accepts `context.Context` | No context | `(ctx, path, branch) error` | FR-004: Context-based timeout |
| `Remove` accepts `context.Context` | No context | `(ctx, path) error` | FR-006: Coordinated cleanup |

**Operations** (updated signatures):
| Method | New Signature | Description |
|--------|---------------|-------------|
| `Create` | `(ctx context.Context, worktreePath, branch string) error` | Acquires repo lock, prunes, creates worktree |
| `Remove` | `(ctx context.Context, worktreePath string) error` | Acquires repo lock, removes worktree |
| `RepoRoot` | `() string` | Returns the repository root path (unchanged) |

---

### 4. `WorktreeEntry` (new — `internal/worktree/`)

Tracks a single worktree created during pipeline execution. Used by the cleanup registry.

```go
type WorktreeEntry struct {
    StepID       string // Pipeline step that created this worktree
    WorktreePath string // Absolute filesystem path to the worktree
    RepoRoot     string // Repository root this worktree belongs to
}
```

---

### 5. `WorktreeRegistry` (new — `internal/pipeline/` or embedded concept)

In-memory registry tracking worktrees created during a single pipeline execution. Replaces the `__worktree_repo_root` suffix convention in `WorkspacePaths`.

```go
// WorktreeRegistry tracks worktrees created during pipeline execution
// for targeted cleanup on completion or failure.
type WorktreeRegistry struct {
    mu      sync.Mutex
    entries []WorktreeEntry
}
```

**Operations**:
| Method | Signature | Description |
|--------|-----------|-------------|
| `Register` | `(entry WorktreeEntry)` | Adds a worktree to the cleanup list |
| `Entries` | `() []WorktreeEntry` | Returns a copy of all registered entries |
| `Count` | `() int` | Returns number of registered worktrees |

**Lifecycle**: Created with `PipelineExecution`, used in deferred cleanup at end of `Execute()`. Not persisted.

---

### 6. `PipelineExecution` (modified — `internal/pipeline/executor.go`)

Extended to use `WorktreeRegistry` instead of the `__worktree_repo_root` suffix convention.

```go
type PipelineExecution struct {
    Pipeline       *Pipeline
    Manifest       *manifest.Manifest
    States         map[string]string
    Results        map[string]map[string]interface{}
    ArtifactPaths  map[string]string
    WorkspacePaths map[string]string
    Input          string
    Status         *PipelineStatus
    Context        *PipelineContext
    Worktrees      *WorktreeRegistry // NEW: replaces __worktree_repo_root convention
}
```

**Changes**:
| Change | Description |
|--------|-------------|
| Add `Worktrees` field | `*WorktreeRegistry` — typed cleanup registry |
| Remove `__worktree_repo_root` entries | No longer written to `WorkspacePaths` |

---

## Relationship Diagram

```
┌──────────────────────────────────────────────────────────┐
│                    Package Level                         │
│  repoLocks (sync.Map)                                    │
│    key: "/abs/path/to/repo" → value: *repoLock           │
│    key: "/abs/path/to/other" → value: *repoLock          │
└───────────────────────┬──────────────────────────────────┘
                        │ getRepoLock()
                        ▼
┌──────────────────────────────────────────────────────────┐
│  Manager (per worktree.NewManager call)                   │
│    repoRoot: "/abs/path/to/repo"                          │
│    lockTimeout: 30s                                       │
│                                                           │
│    Create(ctx, path, branch) ──► acquires repoLock        │
│    Remove(ctx, path) ──────────► acquires repoLock        │
└───────────────────────┬──────────────────────────────────┘
                        │ used by
                        ▼
┌──────────────────────────────────────────────────────────┐
│  PipelineExecution                                        │
│    Worktrees: *WorktreeRegistry                           │
│      entries: []WorktreeEntry                             │
│        - {StepID: "plan", Path: "/tmp/wt-1", Repo: "…"}  │
│        - {StepID: "impl", Path: "/tmp/wt-2", Repo: "…"}  │
│                                                           │
│    cleanupWorktrees() iterates Worktrees.Entries()        │
│    and calls mgr.Remove(ctx, entry.WorktreePath)          │
└──────────────────────────────────────────────────────────┘
```

## Data Flow

1. **Pipeline startup**: `Execute()` creates `PipelineExecution` with empty `WorktreeRegistry`.
2. **Step workspace creation**: `createStepWorkspace()` calls `worktree.NewManager(repoRoot)` → `mgr.Create(ctx, path, branch)`.
   - `Create()` calls `getRepoLock(canonicalRepoRoot)` → `lock.LockWithContext(ctx)` → runs `git worktree prune` + `git worktree add` → `lock.Unlock()`.
   - Step registers `WorktreeEntry` in `execution.Worktrees`.
3. **Pipeline completion/failure**: `cleanupWorktrees()` iterates `execution.Worktrees.Entries()` and calls `mgr.Remove(ctx, path)` for each entry.
   - `Remove()` acquires the same repo lock, ensuring cleanup doesn't race with creation from other pipelines.
4. **Concurrent access**: Multiple pipeline goroutines calling `Create`/`Remove` on the same repository are serialized by the shared `repoLock`. Operations on different repositories proceed in parallel (FR-003).

## Migration Notes

- The `__worktree_repo_root` suffix convention in `WorkspacePaths` must be removed from all callers.
- Existing tests using `WorkspacePaths` for worktree tracking must be updated to use `Worktrees`.
- The `Manager.Create` and `Manager.Remove` signature changes require updating all call sites in `executor.go`.
- The per-instance `mu sync.Mutex` in `Manager` is removed — tests that relied on it being per-instance must be updated to reflect repository-scoped coordination.
