# Feature Specification: Continuous Pipeline Execution

**Feature Branch**: `201-continuous-pipeline`
**Created**: 2026-03-16
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/201

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Batch Issue Processing (Priority: P1)

As a developer managing a backlog of issues, I want to run a pipeline in continuous mode so that it automatically picks up and processes issues one after another without me re-invoking the command each time.

**Why this priority**: This is the core value proposition — eliminating the manual loop of `wave run` per issue. Without this, the feature has no purpose.

**Independent Test**: Can be fully tested by running `wave run --continuous impl-issue --source "github:label=bug"` and verifying it processes multiple matching issues sequentially, each in an isolated workspace.

**Acceptance Scenarios**:

1. **Given** a repository with 3 open issues labeled "bug", **When** the user runs `wave run --continuous impl-issue --source "github:label=bug"`, **Then** the pipeline executes once for each issue, creating a fresh workspace and run record per iteration.
2. **Given** no matching issues exist, **When** the user runs `wave run --continuous impl-issue --source "github:label=bug"`, **Then** the pipeline exits cleanly with a message indicating no work items were found.
3. **Given** a continuous pipeline is running, **When** the user presses Ctrl+C once, **Then** the current pipeline iteration completes and the loop exits without starting the next issue.

---

### User Story 2 - Graceful Shutdown (Priority: P1)

As a developer running a long-lived continuous pipeline, I want to be able to stop processing cleanly so that the current issue's work is not lost or left in a broken state.

**Why this priority**: Without graceful shutdown, users risk corrupted PRs or half-finished worktrees. This is essential for the feature to be safe to use.

**Independent Test**: Can be tested by starting a continuous run, sending SIGINT during an active step, and verifying the current iteration completes fully before the process exits.

**Acceptance Scenarios**:

1. **Given** a continuous pipeline is executing step 3 of 5 for an issue, **When** SIGINT is received, **Then** the current pipeline run completes all remaining steps, the run is marked "completed" in state, and the loop exits.
2. **Given** a continuous pipeline is between iterations (not currently executing a step), **When** SIGINT is received, **Then** the process exits immediately without starting the next iteration.
3. **Given** a continuous pipeline is running, **When** SIGTERM is received, **Then** the behavior is identical to SIGINT — graceful drain of the current iteration.

---

### User Story 3 - Work Item Source Configuration (Priority: P2)

As a developer, I want to configure where the continuous pipeline gets its next work item from so that I can use it with different issue trackers, label filters, or custom queues.

**Why this priority**: The continuous loop is only useful if it can dynamically discover work. Hard-coding GitHub issue listing would limit adoption and violate Wave's forge-agnostic design.

**Independent Test**: Can be tested by configuring different source strategies (GitHub labels, file-based queue) and verifying each produces the correct sequence of inputs.

**Acceptance Scenarios**:

1. **Given** a source configured as `github:label=enhancement,state=open,sort=created,direction=asc`, **When** the continuous loop polls for the next item, **Then** it returns the oldest open issue with the "enhancement" label.
2. **Given** a source configured as `file:queue.txt`, **When** the continuous loop polls for the next item, **Then** it reads and removes the first line from the file as the next input.
3. **Given** a source has returned all available items, **When** the loop polls again, **Then** it receives an empty result and exits the loop.

---

### User Story 4 - Iteration Observability (Priority: P2)

As a developer monitoring a continuous pipeline, I want progress events emitted per iteration so that I can track which issue is being processed, how many have completed, and whether any failed.

**Why this priority**: Without observability, the user has no visibility into a long-running batch process. This enables monitoring via `wave logs` and the TUI.

**Independent Test**: Can be tested by running a continuous pipeline and verifying NDJSON events include iteration metadata (iteration number, work item identifier, cumulative counts).

**Acceptance Scenarios**:

1. **Given** a continuous pipeline processing issue #42 as the 3rd iteration, **When** the iteration starts, **Then** a `loop_iteration` event is emitted with iteration number, input identifier, and total completed count.
2. **Given** an iteration completes successfully, **When** the next iteration begins, **Then** cumulative success/failure counts in events are accurate.
3. **Given** an iteration fails, **When** the `on_failure` policy is `"skip"`, **Then** a `loop_iteration_failed` event is emitted with error details and the loop continues to the next item.

---

### User Story 5 - Failure Policy (Priority: P3)

As a developer, I want to configure what happens when an iteration fails so that I can choose between halting the entire batch or skipping the failed issue and continuing.

**Why this priority**: Batch processing is fragile if one failure stops the entire queue. But some users want fail-fast behavior. Making this configurable serves both use cases.

**Independent Test**: Can be tested by injecting a failure into one iteration and verifying the configured policy (halt vs. skip) is followed.

**Acceptance Scenarios**:

1. **Given** `on_failure: skip` is configured, **When** an iteration fails, **Then** the failure is logged, the issue is recorded as failed, and the loop continues with the next item.
2. **Given** `on_failure: halt` is configured (default), **When** an iteration fails, **Then** the loop exits with a non-zero exit code after the failed iteration completes.
3. **Given** `on_failure: skip` and 3 out of 5 iterations fail, **When** the loop completes, **Then** the exit summary shows 2 succeeded, 3 failed, and the exit code is non-zero.

---

### Edge Cases

- What happens when the source returns an issue that was already processed in a previous continuous run? The system MUST track processed items within the current session and skip duplicates. Cross-session deduplication is handled by the source query (e.g., closed issues excluded by GitHub filter).
- What happens when rate limits are hit while fetching the next work item? The system MUST retry with exponential backoff, respecting `Retry-After` headers.
- What happens when the user runs `--continuous` with `--from-step`? This combination MUST be rejected — resumption applies to a single run, not a loop.
- What happens when a concurrent pipeline limit is reached? Each iteration is a full sequential pipeline run. The `max_concurrency` setting from the manifest applies within each iteration's steps, not across iterations (iterations are sequential).
- What happens when the source produces hundreds of items? The system SHOULD support an optional `--max-iterations N` flag to cap the number of iterations.
- What happens when the process is killed with SIGKILL? The current iteration's state is lost. This is expected and acceptable — the state store will show the run as "running" until a future garbage collection or manual cleanup.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST accept a `--continuous` flag on the `wave run` command that enables looping execution mode.
- **FR-002**: System MUST accept a `--source` flag that specifies where to fetch the next work item from (e.g., `github:label=bug,state=open`).
- **FR-003**: Each iteration of the continuous loop MUST create a new run record in the state store with a unique run ID.
- **FR-004**: Each iteration MUST execute in a fresh, isolated workspace with no shared state from previous iterations.
- **FR-005**: Each iteration MUST start with fresh agent memory (no chat history inheritance), consistent with Wave's existing step isolation model.
- **FR-006**: System MUST handle SIGINT/SIGTERM by completing the current iteration and then exiting the loop.
- **FR-007**: System MUST emit structured progress events for each iteration, including iteration number, work item identifier, and cumulative success/failure counts.
- **FR-008**: System MUST exit the loop when the source returns no more items.
- **FR-009**: System MUST support an `on_failure` policy for iterations: `halt` (default) or `skip`.
- **FR-010**: System MUST reject the combination of `--continuous` with `--from-step` (these are mutually exclusive modes).
- **FR-011**: System MUST support a `--max-iterations N` flag to limit the number of loop iterations.
- **FR-012**: System MUST track which items have been processed within a continuous session to avoid re-processing duplicates.
- **FR-013**: System MUST support configurable delay between iterations via `--delay` flag (default: `0s`, no delay). Rate limiting for API calls (e.g., GitHub) is already handled by the existing `RateLimiter` in `internal/github/ratelimit.go`, so a smart default at the loop level is unnecessary. Users who want to throttle iterations can set `--delay 5s` explicitly.
- **FR-014**: System MUST print a summary at the end of the continuous run showing total iterations, successes, failures, and skipped items.

### Key Entities

- **ContinuousRun**: Represents the overall continuous session. Contains the source configuration, iteration count, success/failure tallies, and references to individual run records. Acts as a parent grouping for multiple pipeline runs.
- **WorkItemSource**: An abstraction that produces the next input string for the pipeline. Implementations include GitHub issue queries and file-based queues. Responsible for filtering, ordering, and deduplication.
- **IterationResult**: The outcome of a single pipeline execution within the loop. Contains the run ID, input, status (success/failure/skipped), duration, and error details if applicable.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A user can process a batch of 10 GitHub issues with a single `wave run --continuous` invocation, with each issue getting its own isolated pipeline execution.
- **SC-002**: Pressing Ctrl+C during a continuous run results in the current iteration completing successfully (no corrupted state) and the process exiting within 30 seconds of the current step finishing.
- **SC-003**: Each iteration's progress is queryable via `wave logs <run-id>`, and the continuous session's overall progress is visible in the event stream.
- **SC-004**: A failed iteration with `on_failure: skip` does not prevent subsequent iterations from executing.
- **SC-005**: The continuous mode adds no overhead to single-run execution — running without `--continuous` behaves identically to the current implementation.

## Clarifications

The following ambiguities were identified and resolved based on codebase context:

### C1: Default delay between iterations (FR-013)

**Ambiguity**: Should the `--delay` flag default to `0` (no delay) or use a smart rate-limit-aware default?

**Resolution**: Default is `0s` (no delay). The existing `RateLimiter` in `internal/github/ratelimit.go` already handles API-level rate limiting with `Retry-After`/`X-RateLimit-*` header awareness. Adding a loop-level smart delay would duplicate this logic. Users who want explicit throttling can set `--delay 5s`.

**Rationale**: Separation of concerns — transport-level rate limiting belongs in the API client, not the loop controller.

### C2: Source URI scheme and forge-agnostic design

**Ambiguity**: The `--source` flag uses a URI-like syntax (`github:label=bug`, `file:queue.txt`) but the grammar is not formally defined, and it's unclear how this interacts with Wave's forge-agnostic design (GitHub, GitLab, Bitbucket, Gitea per `internal/forge/detect.go`).

**Resolution**: The source URI scheme follows the pattern `<provider>:<key=value,...>`. For v1, two providers are supported:
- `github:<filters>` — queries GitHub issues via `gh` CLI. Supported filter keys: `label`, `state` (default: `open`), `sort` (default: `created`), `direction` (default: `asc`), `limit` (default: `100`).
- `file:<path>` — reads lines from a local file, consuming one line per iteration.

Future forge providers (`gitlab:`, `bitbucket:`) can be added as new `WorkItemSource` implementations without changing the loop controller. The source abstraction is designed to be extensible via the provider prefix.

**Rationale**: Matches Wave's existing forge template variable pattern (`{{ forge.cli_tool }}`) while keeping v1 scope achievable. GitHub is the primary forge used today.

### C3: ContinuousRun state persistence

**Ambiguity**: The `ContinuousRun` entity is described as a parent grouping for multiple pipeline runs, but it's unclear whether it gets its own record in the SQLite state store or exists only in memory.

**Resolution**: `ContinuousRun` is an **in-memory struct** that is NOT persisted to the SQLite state store. Each child pipeline run is persisted as a normal run record (the existing state store model). The `ContinuousRun` tracks iteration counts, processed item IDs (for deduplication), and the failure/success tally during the session. The final summary (FR-014) is printed from this in-memory state. Cross-session deduplication is handled by source query filters (e.g., excluding closed issues), not by persistent state.

**Rationale**: Adding a new entity to the SQLite schema increases complexity without clear benefit — the existing per-run records already provide queryable history via `wave logs <run-id>`. The continuous loop is a CLI-session concept, not a durable entity.

### C4: Exit code semantics

**Ambiguity**: The spec states non-zero exit on failures but doesn't define the exact exit code.

**Resolution**: Exit code `0` when all iterations succeed. Exit code `1` when any iteration fails (regardless of `on_failure` policy). This matches Go CLI conventions and `cobra`'s default error handling. The summary (FR-014) provides detailed counts; the exit code is a simple pass/fail signal.

**Rationale**: Standard Unix convention. More granular exit codes (e.g., `2` for partial failure) add complexity without benefit since the summary already provides detail.

### C5: Interaction between iteration-level and step-level on_failure

**Ambiguity**: The codebase already has step-level `on_failure` policies (`skip`, `rework` per `internal/event/emitter.go` states). The spec's iteration-level `on_failure: halt|skip` could conflict.

**Resolution**: The two levels are independent and compose naturally:
- **Step-level `on_failure`** (existing): Controls what happens when a single step within a pipeline run fails (skip the step, rework it, etc.). This is defined in the pipeline YAML per step.
- **Iteration-level `on_failure`** (new): Controls what happens when an entire pipeline run (iteration) fails after all step-level recovery has been exhausted. `halt` stops the loop; `skip` logs the failure and continues to the next work item.

An iteration is considered "failed" only if the pipeline executor's `Execute()` returns an error — meaning step-level retries and on_failure policies have already been applied.

**Rationale**: Layered failure handling — step recovery is internal to a run, iteration recovery is the loop controller's concern. No ambiguity in which level applies.
