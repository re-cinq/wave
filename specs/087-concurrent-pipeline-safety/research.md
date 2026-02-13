# Research: Concurrent Pipeline Safety

**Branch**: `087-concurrent-pipeline-safety` | **Date**: 2026-02-13

## Phase 0 — Unknowns & Research

### Unknown 1: Repository-Scoped Mutex Coordination Pattern

**Question**: What is the best Go pattern for keying mutexes by a dynamic key (repository path) across multiple struct instances within the same process?

**Decision**: Package-level `sync.Map` storing `*sync.Mutex` values keyed by canonical (absolute, symlink-resolved) repository root path.

**Rationale**:
- `sync.Map` is optimized for the read-heavy, append-only access pattern (keys are added once per repository, then read many times).
- Using `filepath.EvalSymlinks` + `filepath.Abs` ensures that different paths to the same repository (e.g., `./`, `/home/user/repo`, symlinks) resolve to a single canonical key.
- The mutex is created via `sync.Map.LoadOrStore` which is atomic — no TOCTOU race between "check if exists" and "create new".
- This pattern is widely used in Go standard library and ecosystem (e.g., `singleflight` uses a similar keyed-mutex approach).

**Alternatives Rejected**:
1. **Filesystem advisory locks (`flock`)**: Adds cross-process coordination complexity (stale lock files, NFS incompatibility) without benefit — Wave runs as a single process. CLR-001 explicitly decided against this.
2. **Global mutex (single lock for all repos)**: Violates FR-003 — operations on different repositories should not block each other.
3. **Channel-based semaphore**: More complex than mutex for simple serialization, and harder to reason about with `defer`-based cleanup.
4. **`sync.Pool` of mutexes**: Wrong abstraction — pools are for reusable objects, not persistent per-key state.

### Unknown 2: Context-Based Lock Timeout

**Question**: How to implement timeout on `sync.Mutex` acquisition in Go, since `sync.Mutex.Lock()` is unconditional and blocks forever?

**Decision**: Wrap the mutex in a helper that uses a channel-based `tryLock` with `context.WithTimeout`.

**Rationale**:
- `sync.Mutex` doesn't support `TryLock` with timeout natively (Go's `TryLock` is non-blocking only).
- Pattern: Use a buffered channel of capacity 1 as a semaphore. `Lock` = send to channel (blocks if full). `Unlock` = receive from channel. `LockWithContext` = `select` between channel send and `ctx.Done()`.
- This is the idiomatic Go approach for cancelable lock acquisition, used in production systems like CockroachDB and etcd.
- Default timeout: 30 seconds (SC-007), configurable via the `Manager` constructor.

**Alternatives Rejected**:
1. **Spin-loop with `TryLock`**: CPU-wasteful and doesn't integrate with `context.Context`.
2. **No timeout**: Risks indefinite blocking if a goroutine leaks while holding the lock. FR-004 explicitly requires timeout support.
3. **`sync.Mutex` + periodic TryLock polling**: Adds latency (polling interval) and is inelegant compared to channel-based select.

**Implementation sketch**:
```go
type repoLock struct {
    sem chan struct{}
}

func newRepoLock() *repoLock {
    rl := &repoLock{sem: make(chan struct{}, 1)}
    return rl
}

func (rl *repoLock) LockWithContext(ctx context.Context) error {
    select {
    case rl.sem <- struct{}{}:
        return nil
    case <-ctx.Done():
        return fmt.Errorf("lock acquisition timed out: %w", ctx.Err())
    }
}

func (rl *repoLock) Unlock() {
    <-rl.sem
}
```

### Unknown 3: Stale Worktree Detection and Recovery

**Question**: How should the system handle stale worktrees from previously-crashed Wave processes?

**Decision**: Run `git worktree prune` at the start of each `Create()` operation, under the repository lock. This is the existing behavior — no change needed except ensuring it happens under the new repository-scoped lock.

**Rationale**:
- `git worktree prune` is idempotent and safe to run concurrently (when serialized by our lock).
- It removes worktree entries from `.git/worktrees/` where the working directory no longer exists on disk.
- Per CLR-004, the scope of stale recovery is limited to detecting stale *worktree directories*, not stale lock files (since coordination is in-process).
- The current code already calls `prune` inside `Create()` (worktree.go:60-61).

**Alternatives Rejected**:
1. **Separate prune goroutine on a timer**: Over-engineering — prune-on-create is sufficient and simpler.
2. **Startup prune (once at process start)**: Doesn't cover worktrees that become stale during process lifetime.
3. **Manual prune command**: Doesn't satisfy FR-008's requirement for automatic detection.

### Unknown 4: Cleanup Registry Structure

**Question**: What data structure should replace the `__worktree_repo_root` suffix convention in `WorkspacePaths`?

**Decision**: A dedicated `WorktreeRegistry` struct with typed fields, embedded as a field on `PipelineExecution`.

**Rationale**:
- The current `__worktree_repo_root` suffix convention is a string-based hack that's fragile and unclear.
- A dedicated struct provides type safety and better discoverability.
- The struct stores `[]WorktreeEntry` where each entry pairs a step ID with the worktree path and repo root.
- Cleanup iterates the registry rather than scanning `WorkspacePaths` for magic suffix keys.
- Per CLR-002, this is in-memory only — no SQLite persistence needed.

**Alternatives Rejected**:
1. **Keep `__worktree_repo_root` convention**: Works but is fragile, undiscoverable, and pollutes the `WorkspacePaths` map with non-workspace entries.
2. **Separate SQLite table**: Over-engineering for in-memory per-execution tracking (CLR-002).
3. **Global worktree registry**: Wrong scope — each pipeline execution should own its own cleanup list.

### Unknown 5: Run ID Uniqueness Verification

**Question**: Is the existing `GenerateRunID()` mechanism sufficient for FR-007 (workspace path uniqueness)?

**Decision**: Yes — the existing `GenerateRunID()` using `crypto/rand` with 8 hex characters (32 bits of entropy) is sufficient. Per CLR-003, no changes to path generation are required.

**Rationale**:
- 32 bits of entropy = ~4 billion possible values. Birthday paradox gives 50% collision probability at ~65k concurrent runs, far beyond realistic usage.
- `crypto/rand` is cryptographically secure, so collisions are genuinely random (no time-based clustering).
- The timestamp fallback provides additional uniqueness if `crypto/rand` fails.
- Implementation should add a test verifying concurrent `GenerateRunID()` calls produce distinct IDs.

**Alternatives Rejected**:
1. **UUID v4**: More entropy but longer paths — unnecessary given the collision probability.
2. **Atomic counter**: Simpler but not unique across process restarts.
3. **Hash of (pipeline name + timestamp + PID)**: Deterministic components risk collisions under concurrent startup.

### Unknown 6: Matrix Worker Worktree Coordination

**Question**: How should matrix workers coordinate worktree operations with the global repository lock?

**Decision**: Matrix workers use the same `worktree.Manager` (which will use the repository-scoped lock) as regular pipeline steps. No special handling needed.

**Rationale**:
- FR-010 requires matrix workers to use the same coordination mechanism as pipeline-level operations.
- Since the repository lock is per-repository (not per-manager-instance), all worktree operations from all sources (pipeline steps, matrix workers, concurrent pipelines) are automatically serialized.
- The lock is held only for the duration of individual git operations (FR-011), so matrix workers don't block each other during step execution — only during worktree create/remove.
- The existing `errgroup`-based concurrency in `MatrixExecutor.Execute()` works correctly with repository-scoped locking.

**Alternatives Rejected**:
1. **Matrix-specific lock**: Introduces two coordination mechanisms — more complex, risk of deadlocks.
2. **Sequential matrix worktree creation**: Unnecessarily serializes the entire matrix step startup. The repo lock serializes only the git operations, not workspace setup.

### Unknown 7: Observability Enhancements

**Question**: What changes are needed to satisfy FR-009 (pipeline run ID in all worktree operation logs)?

**Decision**: Pass the pipeline run ID (and optionally step ID) to `worktree.Manager` operations, and include them in structured log output via the event emitter.

**Rationale**:
- The current `Manager.Create()` and `Manager.Remove()` don't accept context or log parameters.
- Rather than modifying the `Manager` API to accept loggers (which would couple `worktree` to `event`), the caller (pipeline executor) should emit events before and after worktree operations.
- The executor already emits events with `PipelineID` and `StepID` — worktree operation events should follow the same pattern.
- SC-005 requires log messages to include run ID, which the executor provides.

**Alternatives Rejected**:
1. **Logger injection into Manager**: Couples worktree package to event package — violates single responsibility.
2. **Global logger**: Anti-pattern in Go; makes testing harder.
3. **Return structured results from Manager**: Over-engineering for simple success/fail logging.
