# Contract: Worktree Manager API

**Package**: `internal/worktree`

## Public API Contract

### Constructor

```go
// NewManager creates a worktree manager for the given git repository root.
// If repoRoot is empty, auto-detects via `git rev-parse --show-toplevel`.
// Options:
//   - WithLockTimeout(d time.Duration): sets lock acquisition timeout (default: 30s)
func NewManager(repoRoot string, opts ...ManagerOption) (*Manager, error)

type ManagerOption func(*Manager)

func WithLockTimeout(d time.Duration) ManagerOption
```

**Preconditions**:
- `repoRoot` is either empty (auto-detect) or points to a valid git repository (`.git` exists).

**Postconditions**:
- Returns a `Manager` that coordinates with all other `Manager` instances targeting the same repository.
- The manager's `repoRoot` is the canonical (absolute, symlink-resolved) path.

### Create

```go
// Create creates a new git worktree at worktreePath on the specified branch.
// Acquires repository-scoped lock for the duration of git operations.
// If branch doesn't exist, creates a new branch from HEAD.
// Runs `git worktree prune` before creation to clean stale worktrees.
func (m *Manager) Create(ctx context.Context, worktreePath, branch string) error
```

**Preconditions**:
- `ctx` is not nil. Context timeout governs lock acquisition timeout.
- `worktreePath` is non-empty.
- `branch` is non-empty.

**Postconditions (success)**:
- Worktree exists at `worktreePath` with `branch` checked out.
- Repository lock is released.

**Postconditions (failure)**:
- No worktree created.
- Repository lock is released.
- Error returned describes the failure (lock timeout, git error, etc.).

**Lock behavior**:
- Acquires repository-scoped lock before any git operation.
- Lock is held for: prune + stale cleanup + worktree add.
- Lock is NOT held after return (FR-011).

### Remove

```go
// Remove removes the git worktree at worktreePath.
// Acquires repository-scoped lock for the duration of git operations.
// Falls back to force removal if normal removal fails (dirty worktree).
func (m *Manager) Remove(ctx context.Context, worktreePath string) error
```

**Preconditions**:
- `ctx` is not nil.
- `worktreePath` is non-empty.

**Postconditions (success)**:
- Worktree at `worktreePath` is removed.
- Repository lock is released.

**Postconditions (failure)**:
- Worktree may or may not exist (best-effort removal).
- Repository lock is released.
- Error returned with details and remediation hints (FR: User Story 3 Scenario 4).

**Lock behavior**: Same as `Create`.

### RepoRoot

```go
func (m *Manager) RepoRoot() string
```

No lock needed — immutable after construction.

## Internal Functions (package-private)

```go
// getRepoLock returns the lock for the given canonical repository path,
// creating one atomically if it doesn't exist.
func getRepoLock(canonicalPath string) *repoLock

// canonicalPath resolves symlinks and returns the absolute path.
func canonicalPath(path string) (string, error)
```

## Error Types

| Error Condition | Error Message Pattern |
|-----------------|----------------------|
| Lock timeout | `"lock acquisition timed out for repository %s: %w"` |
| Empty path | `"worktree path cannot be empty"` |
| Empty branch | `"branch name cannot be empty"` |
| Git error | `"git worktree add failed: %w\noutput: %s"` |
| Stale cleanup failure | `"failed to clean up stale worktree at %s: %w"` |
| Not a git repo | `"not a git repository: %s"` |

## Thread Safety

- All public methods are safe for concurrent use.
- Concurrent calls targeting the **same repository** are serialized.
- Concurrent calls targeting **different repositories** execute in parallel.
