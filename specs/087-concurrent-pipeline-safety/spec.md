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
3. **Given** a worktree lock held by a process that crashes, **When** another process attempts to acquire the lock, **Then** the stale lock is detected and recovered after a configurable timeout.
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
- What happens when the lock file becomes corrupted? The system should detect corruption, log a warning, remove the corrupted lock, and recreate it.
- What happens when a pipeline run takes extremely long and its lock prevents other pipelines from proceeding? The lock should be scoped to individual git operations (create/remove), not held for the entire pipeline execution.
- What happens when `os.MkdirAll` races between two concurrent workspace creations? The system should handle `EEXIST` gracefully and use the existing directory or generate a new unique path.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The system MUST provide a repository-scoped coordination mechanism for git worktree operations that works across all `worktree.Manager` instances targeting the same repository.
- **FR-002**: The coordination mechanism MUST serialize `git worktree add`, `git worktree remove`, and `git worktree prune` operations on the same repository to prevent concurrent git state corruption.
- **FR-003**: The coordination mechanism MUST NOT serialize operations on different repositories (per-repo scoping).
- **FR-004**: The coordination mechanism MUST handle stale locks from crashed processes, recovering automatically after a configurable timeout.
- **FR-005**: The pipeline executor MUST clean up worktrees in a `defer`-style pattern that executes even when pipeline steps fail or panic.
- **FR-006**: Worktree cleanup MUST be coordinated through the same mechanism as worktree creation to prevent cleanup/creation races.
- **FR-007**: Workspace paths for concurrent pipeline runs MUST include a unique component (e.g., run ID) that prevents path collisions between runs of the same pipeline definition.
- **FR-008**: The system MUST detect and prune stale worktrees from failed previous runs before attempting to create new ones.
- **FR-009**: All worktree and workspace operations MUST include the pipeline run ID in log messages and progress events for observability.
- **FR-010**: Matrix worker worktree operations MUST be coordinated through the same mechanism as pipeline-level worktree operations.
- **FR-011**: The system MUST NOT hold coordination locks during pipeline step execution — locks are acquired only for the duration of individual git operations.
- **FR-012**: The system MUST pass `go test -race ./...` with concurrent worktree and workspace operations under test.

### Key Entities

- **Repository Lock**: A per-repository synchronization primitive that coordinates all git worktree operations targeting a specific repository. Scoped by the repository root path. Supports timeout-based stale lock recovery.
- **Worktree Manager**: The existing `worktree.Manager` struct, extended to participate in repository-scoped coordination rather than relying solely on per-instance mutexes.
- **Pipeline Run ID**: A unique identifier for each pipeline execution that provides workspace path uniqueness and log correlation. Already partially implemented via the hash-suffixed pipeline IDs from issue #25.
- **Workspace Path**: The filesystem path where a pipeline step executes. Unique across all concurrent executions through incorporation of the pipeline run ID.
- **Cleanup Registry**: A record of worktrees owned by a specific pipeline run, enabling targeted cleanup on failure without affecting other runs' worktrees.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: 10 concurrent pipeline runs that each create and remove worktrees on the same repository all complete successfully with zero git errors, verified by an integration test.
- **SC-002**: `go test -race ./internal/worktree/...` passes with tests that exercise concurrent worktree creation and removal across multiple manager instances.
- **SC-003**: After a simulated pipeline failure during worktree usage, the next pipeline run on the same branch succeeds without manual cleanup intervention.
- **SC-004**: Workspace paths for two concurrent runs of the same pipeline definition are guaranteed to differ, verified by test.
- **SC-005**: All worktree operation log messages include the pipeline run ID, verifiable by log output inspection in tests.
- **SC-006**: The coordination mechanism adds less than 100ms of overhead per worktree operation compared to uncoordinated execution (measured under no contention).
- **SC-007**: Stale lock recovery triggers within 30 seconds of the lock holder's crash (configurable timeout).
- **SC-008**: No deadlocks occur when 5 matrix workers and 3 concurrent pipelines operate on the same repository simultaneously, verified by a test with `-race` and a reasonable timeout.
