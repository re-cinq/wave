# Feature Specification: Concurrent Pipeline Safety

**Feature Branch**: `087-concurrent-pipeline-safety`
**Created**: 2026-02-13
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/29

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Safe concurrent pipeline execution (Priority: P1)

As a Wave user, when I trigger multiple pipeline runs simultaneously (e.g., processing several GitHub issues at once), each pipeline executes in complete isolation without corrupting git state, overwriting files, or interfering with other running pipelines.

**Why this priority**: This is the core value proposition. Without safe concurrent execution, Wave users must serialize all pipeline runs manually, which defeats the purpose of a multi-agent orchestrator. Git repository corruption from concurrent worktree operations is the highest-severity failure mode.

**Independent Test**: Can be fully tested by launching 3+ pipelines concurrently that each create worktrees on different branches and verifying that all complete successfully with correct outputs and no git corruption.

**Acceptance Scenarios**:

1. **Given** 3 pipelines configured to run concurrently, **When** all 3 are started at the same time, **Then** all 3 complete successfully without git errors, file conflicts, or corrupted worktree references.
2. **Given** 2 pipelines that both create worktrees from the same repository root, **When** they execute concurrently, **Then** their git worktree operations are serialized to prevent `.git/worktrees/` reference file corruption.
3. **Given** a pipeline that fails mid-execution while other pipelines are running, **When** the failed pipeline's worktree is cleaned up, **Then** the cleanup does not interfere with worktrees belonging to other active pipelines.
4. **Given** concurrent pipeline execution, **When** pipelines complete, **Then** each pipeline's workspace directory contains only its own artifacts with no cross-contamination.

---

### User Story 2 - Global worktree operation coordination (Priority: P1)

As the Wave runtime, all git worktree operations (create, remove, prune) across all pipeline executions are coordinated through a single synchronization mechanism, preventing race conditions on the shared `.git/worktrees/` directory.

**Why this priority**: The current per-instance mutex in `worktree.Manager` does not coordinate across multiple manager instances. Each pipeline execution creates its own manager, resulting in uncoordinated concurrent git operations on the same repository. This is the root cause of the concurrency concern in issue #29.

**Independent Test**: Can be tested by creating 10 worktree manager instances pointing at the same repository and invoking create/remove operations concurrently, then verifying git state consistency.

**Acceptance Scenarios**:

1. **Given** multiple `worktree.Manager` instances targeting the same repository, **When** they perform create/remove operations concurrently, **Then** all operations are serialized through a shared coordination mechanism.
2. **Given** a repository-scoped lock, **When** a worktree create operation is in progress, **Then** another create operation targeting the same repository blocks until the first completes.
3. **Given** a goroutine holding the repository lock that panics or whose context is cancelled, **When** another goroutine attempts to acquire the lock, **Then** the lock is released via `defer`-based cleanup and the waiting goroutine proceeds. If the lock cannot be acquired within the configurable timeout (default 30s), the operation fails with a clear error.
4. **Given** worktree operations on two different repositories, **When** they execute concurrently, **Then** they do NOT block each other (locks are per-repository, not global).

---

### User Story 3 - Resilient worktree cleanup (Priority: P1)

As a Wave user, when a pipeline fails or is interrupted, its worktrees and workspace directories are reliably cleaned up without leaving stale git references that could block future pipeline runs.

**Why this priority**: Stale worktrees from failed runs accumulate over time and eventually prevent new worktree creation (git refuses to create a worktree if a branch is already checked out in a stale one). This is a blocking failure mode for repeated pipeline execution.

**Independent Test**: Can be tested by simulating pipeline failures at various stages of worktree lifecycle and verifying that subsequent pipeline runs succeed without manual cleanup.

**Acceptance Scenarios**:

1. **Given** a pipeline that fails after creating a worktree but before completing, **When** the pipeline executor handles the failure, **Then** the worktree is removed and its branch reference is freed.
2. **Given** a stale worktree from a previous failed run, **When** a new pipeline run attempts to create a worktree on the same branch, **Then** the stale worktree is automatically detected and cleaned up before the new one is created.
3. **Given** multiple concurrent pipelines, **When** one pipeline's cleanup runs while another pipeline is actively using its worktree, **Then** the cleanup only affects the failed pipeline's worktree — not the active pipeline's.
4. **Given** a worktree that cannot be removed (e.g., due to filesystem permissions), **When** cleanup fails, **Then** a clear error is logged with the worktree path and remediation instructions, and the failure does not block other pipelines.

---

### User Story 4 - Concurrent matrix step execution (Priority: P2)

As a pipeline author using matrix strategies, when matrix workers execute concurrently within a single pipeline step, their worktree and workspace operations do not conflict with each other or with operations from other pipelines.

**Why this priority**: Matrix execution is already implemented with `errgroup` concurrency, but worktree operations from matrix workers are not coordinated with cross-pipeline operations. This extends the global coordination to cover intra-pipeline concurrency.

**Independent Test**: Can be tested by running a matrix step with 5+ concurrent workers that each create worktrees and verifying all workers complete with correct outputs.

**Acceptance Scenarios**:

1. **Given** a matrix step with 5 concurrent workers, **When** all workers create worktrees simultaneously, **Then** all 5 worktrees are created successfully through the shared coordination mechanism.
2. **Given** a matrix step where one worker fails, **When** the failed worker's worktree is cleaned up, **Then** the other workers continue executing without interruption.
3. **Given** matrix workers and a separate concurrent pipeline, **When** both use worktrees, **Then** all operations are coordinated through the same mechanism without deadlocks.

---

### User Story 5 - Workspace path uniqueness guarantee (Priority: P2)

As the Wave runtime, workspace paths for concurrent executions are guaranteed to be unique, preventing any possibility of file collisions between pipeline runs or matrix workers.

**Why this priority**: Workspace path collisions lead to silent data corruption — artifacts from one pipeline overwrite another's. While current paths use pipeline ID and step ID, concurrent runs of the same pipeline could collide without additional uniqueness guarantees.

**Independent Test**: Can be tested by starting two concurrent runs of the same pipeline and verifying their workspace paths are distinct and contain independent artifacts.

**Acceptance Scenarios**:

1. **Given** two concurrent runs of the same pipeline definition, **When** workspace paths are generated, **Then** each run receives a unique workspace directory that does not overlap with the other.
2. **Given** a workspace path that already exists from a previous (incomplete) run, **When** a new run attempts to use that path, **Then** the system either uses a fresh unique path or cleans up the stale directory first.
3. **Given** concurrent pipeline runs, **When** artifacts are injected into workspaces, **Then** each run's artifact injection is isolated and does not read or write the other run's artifacts.

---

### User Story 6 - Observability for concurrent execution (Priority: P3)

As a Wave operator monitoring concurrent pipeline runs, I can distinguish between pipelines in logs and events, and I can identify which pipeline owns which worktree and workspace resources.

**Why this priority**: When concurrent pipelines fail, operators need to correlate errors with specific pipeline runs to diagnose issues. Without clear ownership tracking, debugging concurrent failures becomes intractable.

**Independent Test**: Can be tested by running concurrent pipelines with debug logging enabled and verifying that each log line and event includes the pipeline run ID.

**Acceptance Scenarios**:

1. **Given** concurrent pipeline runs with debug logging, **When** worktree operations occur, **Then** each log message includes the pipeline run ID and the worktree path.
2. **Given** a failed concurrent pipeline, **When** the operator inspects the event stream, **Then** they can identify which worktrees belong to the failed run and their cleanup status.

---

### Edge Cases

- What happens when the filesystem runs out of space during concurrent worktree creation? The system should fail the current operation cleanly and not leave partial worktree state. Other active pipelines should continue unaffected.
- What happens when `git worktree add` is interrupted by SIGTERM/SIGKILL during concurrent execution? The next pipeline run should detect and clean up partial worktree state during its prune phase.
- What happens when two pipelines attempt to create worktrees on the same branch name simultaneously? The coordination mechanism should serialize these operations; the second attempt should fail with a clear error (branch already checked out) rather than corrupting state.
- What happens when a goroutine panics while holding the repository lock? The `defer`-based unlock ensures the mutex is released. The worktree may be left in a partial state, which is cleaned up by `git worktree prune` on the next create operation.
- What happens when a pipeline run takes extremely long and its lock prevents other pipelines from proceeding? The lock should be scoped to individual git operations (create/remove), not held for the entire pipeline execution.
- What happens when `os.MkdirAll` races between two concurrent workspace creations? The system should handle `EEXIST` gracefully and use the existing directory or generate a new unique path.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The system MUST provide a repository-scoped coordination mechanism for git worktree operations that works across all `worktree.Manager` instances targeting the same repository.
- **FR-002**: The coordination mechanism MUST serialize `git worktree add`, `git worktree remove`, and `git worktree prune` operations on the same repository to prevent concurrent git state corruption.
- **FR-003**: The coordination mechanism MUST NOT serialize operations on different repositories (per-repo scoping).
- **FR-004**: The system MUST detect and recover from stale worktree directories left by previously-crashed Wave processes (via `git worktree prune` during creation). Lock acquisition MUST support context-based timeout to prevent indefinite blocking when a goroutine holds the repository lock for too long.
- **FR-005**: The pipeline executor MUST clean up worktrees in a `defer`-style pattern that executes even when pipeline steps fail or panic.
- **FR-006**: Worktree cleanup MUST be coordinated through the same mechanism as worktree creation to prevent cleanup/creation races.
- **FR-007**: Workspace paths for concurrent pipeline runs MUST include a unique component (e.g., run ID) that prevents path collisions between runs of the same pipeline definition.
- **FR-008**: The system MUST detect and prune stale worktrees from failed previous runs before attempting to create new ones.
- **FR-009**: All worktree and workspace operations MUST include the pipeline run ID in log messages and progress events for observability.
- **FR-010**: Matrix worker worktree operations MUST be coordinated through the same mechanism as pipeline-level worktree operations.
- **FR-011**: The system MUST NOT hold coordination locks during pipeline step execution — locks are acquired only for the duration of individual git operations.
- **FR-012**: The system MUST pass `go test -race ./...` with concurrent worktree and workspace operations under test.

### Key Entities

- **Repository Lock**: An in-process `sync.Mutex` keyed by canonical repository root path, managed via a package-level `sync.Map` in `internal/worktree/`. Coordinates all git worktree operations (create, remove, prune) targeting a specific repository across all `worktree.Manager` instances within the same Wave process. Lock acquisition supports context-based timeout to prevent indefinite blocking.
- **Worktree Manager**: The existing `worktree.Manager` struct, extended to participate in repository-scoped coordination rather than relying solely on per-instance mutexes.
- **Pipeline Run ID**: A unique identifier for each pipeline execution that provides workspace path uniqueness and log correlation. Already partially implemented via the hash-suffixed pipeline IDs from issue #25.
- **Workspace Path**: The filesystem path where a pipeline step executes. Unique across all concurrent executions through incorporation of the pipeline run ID.
- **Cleanup Registry**: An in-memory data structure within `PipelineExecution` that tracks worktrees created during a pipeline run. Formalizes the existing `WorkspacePaths` tracking (currently using `__worktree_repo_root` suffix convention) into a dedicated typed structure. Enables targeted cleanup on failure without affecting other runs' worktrees. Not persisted to SQLite — stale worktree recovery from prior process crashes is handled by `git worktree prune` at creation time.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: 10 concurrent pipeline runs that each create and remove worktrees on the same repository all complete successfully with zero git errors, verified by an integration test.
- **SC-002**: `go test -race ./internal/worktree/...` passes with tests that exercise concurrent worktree creation and removal across multiple manager instances.
- **SC-003**: After a simulated pipeline failure during worktree usage, the next pipeline run on the same branch succeeds without manual cleanup intervention.
- **SC-004**: Workspace paths for two concurrent runs of the same pipeline definition are guaranteed to differ, verified by test.
- **SC-005**: All worktree operation log messages include the pipeline run ID, verifiable by log output inspection in tests.
- **SC-006**: The coordination mechanism adds less than 100ms of overhead per worktree operation compared to uncoordinated execution (measured under no contention).
- **SC-007**: Lock acquisition times out after 30 seconds (configurable) if the repository lock cannot be obtained, preventing indefinite blocking. Stale worktree directories from prior process crashes are automatically cleaned up via `git worktree prune` during creation.
- **SC-008**: No deadlocks occur when 5 matrix workers and 3 concurrent pipelines operate on the same repository simultaneously, verified by a test with `-race` and a reasonable timeout.

## Clarifications _(resolved)_

### CLR-001: Coordination mechanism implementation strategy

**Ambiguity**: The spec references a "repository-scoped coordination mechanism" and "Repository Lock" entity but does not specify the concrete implementation: in-process `sync.Mutex` keyed by repo path, filesystem-based advisory locks (`flock`/lock files), or a hybrid approach.

**Resolution**: Use an **in-process `sync.Mutex` keyed by canonical repository root path**, managed via a package-level `sync.Map` in `internal/worktree/`. This is the correct approach because:

1. **Deployment model**: Wave runs as a single Go process — all pipeline goroutines share the same address space. Filesystem locks add complexity and failure modes (stale lock files, NFS incompatibility) without benefit for single-process coordination.
2. **Codebase precedent**: The existing `worktree.Manager` already uses `sync.Mutex`; this extends the same pattern from per-instance to per-repository scoping.
3. **FR-004 (stale lock recovery)**: Since coordination is in-process, "stale locks from crashed processes" becomes a non-issue — when the process dies, all locks are released. FR-004's scope is narrowed to detecting stale *worktree directories* from previous process crashes (already partially handled by `git worktree prune`), not stale lock files.

**Sections updated**: FR-004 is reinterpreted; User Story 2 Scenario 3 is reframed from "process crash" to "goroutine panic/context cancellation".

### CLR-002: Cleanup Registry — in-memory per-execution tracking

**Ambiguity**: The "Cleanup Registry" key entity is described as "a record of worktrees owned by a specific pipeline run" but the spec doesn't specify whether it should be persisted (SQLite) or held in memory, or what its lifecycle is.

**Resolution**: The Cleanup Registry is an **in-memory data structure within `PipelineExecution`**, not a persistent store. Rationale:

1. **Existing pattern**: The current code already tracks worktrees via the `WorkspacePaths` map with `__worktree_repo_root` suffix keys — this IS the cleanup registry, just unnamed.
2. **Lifecycle**: The registry lives for the duration of a pipeline execution and is used in the deferred cleanup at the end of `Execute()`. No persistence needed because worktree cleanup must happen before the process exits (and stale worktree recovery from prior crashes is handled by `git worktree prune` at creation time).
3. **Implementation**: Formalize the existing `WorkspacePaths` tracking into a dedicated `WorktreeRegistry` field on `PipelineExecution` (or refactor the `__worktree_repo_root` convention into a proper struct), rather than introducing a new SQLite table.

**Sections updated**: Key Entities (Cleanup Registry definition clarified).

### CLR-003: Workspace path uniqueness — existing mechanism is sufficient

**Ambiguity**: FR-007 requires workspace paths to include a "unique component (e.g., run ID)" but the existing `GenerateRunID()` already produces `{name}-{8-char-hex}` using `crypto/rand`, and workspace paths are already `{wsRoot}/{pipelineID}/{stepID}`. The spec doesn't state whether changes are needed or this is documenting existing behavior.

**Resolution**: The existing `GenerateRunID()` mechanism **already satisfies FR-007**. The 8-character hex suffix from `crypto/rand` provides 32 bits of entropy (4 billion possible values), making collisions astronomically unlikely for concurrent runs. No changes to the path generation are required. Implementation should:

1. **Verify** (via test) that two concurrent `GenerateRunID()` calls for the same pipeline name produce different IDs.
2. **Document** that workspace path uniqueness is guaranteed by the existing run ID generation, not by additional mechanisms.
3. **Handle the edge case** of a stale workspace directory from a previous run at the same path (already handled by `os.RemoveAll` in `Execute()`).

**Sections updated**: FR-007 (confirmed as existing behavior); User Story 5 (test focus clarified).

### CLR-004: Scope is intra-process coordination, not inter-process

**Ambiguity**: FR-004 references "crashed processes" and SC-007 mentions "lock holder's crash," which implies inter-process coordination. However, Wave's architecture runs pipelines as goroutines within a single `wave run` or `wave serve` process — multiple OS processes managing the same repository simultaneously is not a supported deployment model.

**Resolution**: The coordination scope is **intra-process (goroutine-level)**, not inter-process. Clarifications:

1. **FR-004 reinterpretation**: "Stale locks from crashed processes" is reinterpreted as "stale worktree directories left by a previously-crashed Wave process." Recovery means detecting these via `git worktree prune` at creation time (already implemented in `Manager.Create()`), not implementing filesystem lock timeout mechanisms.
2. **SC-007 reinterpretation**: "Stale lock recovery" refers to stale *worktree cleanup*, not lock file recovery. The 30-second timeout applies to the wait time for a goroutine holding a `sync.Mutex` before the waiter gives up (via `context.WithTimeout` on lock acquisition).
3. **User Story 2 Scenario 3**: "Process that crashes" is reframed as "goroutine that panics or whose context is cancelled." `defer`-based unlock in the mutex wrapper ensures the lock is always released.

**Sections updated**: FR-004, SC-007, User Story 2 Scenario 3.

### CLR-005: Lock granularity — prune runs under the same repository lock

**Ambiguity**: FR-002 lists `git worktree prune` as a separately-serialized operation, but the current code calls `prune` as part of the `Create()` flow (line 60-61 in worktree.go). The spec doesn't clarify whether `prune` needs its own lock acquisition or runs under the create/remove lock.

**Resolution**: `git worktree prune` runs **under the same repository-scoped lock** as the enclosing `create` or `remove` operation — it does NOT require separate lock acquisition. Rationale:

1. **Current behavior**: `prune` is called within `Create()` as a cleanup step before the actual `git worktree add`. It's not invoked as a standalone operation.
2. **Simplicity**: A single repository-scoped lock for all git worktree operations (create, remove, prune) is simpler and avoids reentrant lock complexity.
3. **Lock scope**: Per FR-011, the lock covers only the git operation duration. Since `prune` is part of the create flow, the lock held during `Create()` naturally covers both prune and add operations.
4. **Future proofing**: If a standalone `Prune()` method is added later, it should acquire the same repository lock independently.

**Sections updated**: FR-002 (clarified that prune runs under create/remove lock).
