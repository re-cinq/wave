# Implementation Plan: Continuous Pipeline Execution

## Objective

Add a `--continuous` flag to `wave run` that enables pipelines to automatically iterate over matching GitHub issues (or other work items), processing each one in a fully isolated execution with fresh workspace and memory, until no more items remain or the user interrupts.

## Approach

### High-Level Strategy

Create a new `internal/continuous/` package that encapsulates the continuous execution loop. This package wraps the existing `pipeline.DefaultPipelineExecutor` and orchestrates iteration-level concerns (issue polling, state tracking, delay, graceful shutdown) while delegating actual pipeline execution to the existing executor. This avoids modifying the core executor's single-run semantics.

The design follows Wave's existing patterns:
- **Fresh isolation per iteration**: Each issue gets its own run ID, workspace, and executor instance
- **Signal handling**: Integrates with Go's `context.Context` cancellation, extending the existing SIGINT handler in `run.go`
- **Event emission**: New iteration lifecycle events (`iteration_started`, `iteration_completed`, `iteration_failed`, `continuous_exhausted`) extend the existing `event.Event` vocabulary
- **State tracking**: Uses the existing SQLite `state.StateStore` to record processed issues, preventing re-processing across restarts

### Issue Polling Strategy

The continuous runner uses a **provider** interface to fetch work items. The first implementation is a GitHub provider that:
1. Lists open issues matching optional label/milestone filters via `gh issue list`
2. Orders by oldest first (creation date ascending)
3. Skips issues already tracked as processed in the state store
4. Returns the next unprocessed issue URL as the pipeline input

This is pluggable — future providers for GitLab, Gitea, etc. can implement the same interface.

### Failure Handling

Default: **skip and continue**. A failed iteration logs the failure, emits a `continuous_iteration_failed` event, and moves to the next issue. A `--continuous-halt-on-error` flag allows users to switch to halt-on-first-failure behavior.

### Delay Between Iterations

A configurable delay (default: 10s) between iterations prevents API rate limiting and allows time for external state changes. Configurable via `--continuous-delay` flag.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/continuous/runner.go` | **create** | Core continuous execution loop with `Runner` type |
| `internal/continuous/provider.go` | **create** | `WorkItemProvider` interface and GitHub implementation |
| `internal/continuous/state.go` | **create** | Processed-issue tracking via state store |
| `internal/continuous/runner_test.go` | **create** | Unit tests for runner logic |
| `internal/continuous/provider_test.go` | **create** | Unit tests for issue provider |
| `cmd/wave/commands/run.go` | **modify** | Add `--continuous`, `--continuous-delay`, `--continuous-halt-on-error` flags; wire up runner |
| `internal/event/emitter.go` | **modify** | Add continuous-mode event state constants |
| `internal/state/store.go` | **modify** | Add `MarkIssueProcessed`/`IsIssueProcessed` methods to `StateStore` interface |
| `internal/state/migrations.go` | **modify** | Add migration for `continuous_processed_items` table |
| `internal/state/migration_definitions.go` | **modify** | Define schema for the new table |
| `internal/pipeline/types.go` | **modify** | Add `ContinuousConfig` to `InputConfig` for manifest-level continuous settings |

## Architecture Decisions

### 1. Separate package (`internal/continuous/`) vs. extending executor
**Decision**: Separate package.
**Rationale**: The executor's `Execute()` method has clear single-run semantics. Continuous mode is an orchestration concern that wraps execution, not a core execution feature. This keeps the executor simple and testable.

### 2. Provider interface vs. hardcoded GitHub polling
**Decision**: Provider interface with GitHub as first implementation.
**Rationale**: Wave already supports GitHub, GitLab, Gitea, and Bitbucket forges. The `internal/forge` package detects the platform. Using an interface allows each forge to provide its own issue-listing mechanism without coupling the continuous runner to any specific platform.

### 3. State tracking via SQLite vs. GitHub labels
**Decision**: SQLite state store.
**Rationale**: Using GitHub labels to mark issues as "in-progress" requires write permissions and can conflict with user workflows. SQLite tracking is local, fast, and already exists in Wave's state infrastructure. The trade-off is that state is not shared across machines, which is acceptable for the prototype phase.

### 4. CLI flags vs. manifest-level configuration
**Decision**: Both. CLI flags for quick use (`--continuous`), manifest `input.continuous` block for pipeline-level defaults.
**Rationale**: CLI flags provide immediate usability. Manifest configuration lets pipeline authors define default continuous behavior (e.g., label filters, delay) that users can override via CLI.

### 5. Graceful shutdown scope
**Decision**: Complete current **pipeline execution** (all remaining steps), then stop.
**Rationale**: Stopping mid-pipeline would leave partial work (e.g., a branch with no PR). Completing the current iteration ensures clean artifacts.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| GitHub API rate limiting under high iteration count | Pipeline stalls or errors | Default 10s delay between iterations; respect `gh` CLI rate limit headers |
| State store corruption during concurrent continuous runs | Re-processing or skipped issues | SQLite WAL mode already handles concurrent reads; add row-level locking for writes |
| Long-running process memory growth | OOM kill | Each iteration creates a fresh executor; old executions are garbage-collected |
| User confusion about which issues were processed | Support burden | Clear progress events and `wave list runs` showing continuous run history |
| Interrupted iteration leaving stale worktrees | Disk space growth | Existing workspace cleanup handles this; add cleanup on graceful shutdown |

## Testing Strategy

### Unit Tests
- `internal/continuous/runner_test.go`: Test iteration loop with mock provider and mock executor
  - Happy path: processes 3 issues then exhausts
  - Graceful shutdown: cancellation stops after current iteration
  - Halt-on-error: stops on first failure
  - Skip-and-continue: skips failed issue, processes next
  - Delay between iterations
  - Empty provider returns immediately
- `internal/continuous/provider_test.go`: Test GitHub provider with mocked `gh` CLI output
  - Parse `gh issue list --json` output
  - Filter out already-processed issues
  - Label filtering
  - Empty result set

### Integration Tests
- Test continuous run with mock adapter processing 2 issues end-to-end
- Test state persistence: run, interrupt, resume skips already-processed issues
- Test event emission sequence for continuous mode

### Manual Testing
- `wave run --continuous gh-implement --mock` with test issues
- Verify Ctrl+C graceful shutdown behavior
- Verify `wave list runs` shows individual iteration runs
