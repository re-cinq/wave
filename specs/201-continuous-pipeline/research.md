# Research: Continuous Pipeline Execution

**Feature**: #201 — Continuous Pipeline Execution
**Date**: 2026-03-16

## Decision 1: Loop Controller Placement

**Decision**: New `internal/continuous/` package containing a `Runner` struct that wraps the existing `DefaultPipelineExecutor.Execute()` call in a loop. The loop controller lives outside the executor.

**Rationale**: The executor already handles single-pipeline execution with full lifecycle (workspace creation, artifact injection, contract validation, state persistence). Embedding loop logic inside the executor would violate single responsibility and complicate the existing 800+ line `executor.go`. A separate package keeps the loop controller focused on iteration concerns (source polling, dedup, failure policy, signal handling) while delegating actual execution to the existing machinery.

**Alternatives Rejected**:
- **Embed in `runRun()`**: Would bloat `cmd/wave/commands/run.go` with business logic that belongs in `internal/`.
- **Extend `Execute()` with loop params**: Would add branching to every execution path and complicate the common single-run case.

## Decision 2: Work Item Source Abstraction

**Decision**: `WorkItemSource` interface with `Next(ctx) (*WorkItem, error)` method. Two implementations for v1: `GitHubSource` (wraps `gh issue list`) and `FileSource` (reads lines from a file).

**Rationale**: The spec defines `--source "github:label=bug,state=open"` with a `<provider>:<params>` URI scheme. An interface allows adding forge providers (`gitlab:`, `bitbucket:`) without changing the loop controller. The `gh` CLI is already a runtime dependency (forge detection in `internal/forge/detect.go` already shells out to git). Using `gh issue list --json` avoids building a GitHub API client — the existing `internal/github/client.go` is for issue enhancement, not listing.

**Alternatives Rejected**:
- **Use `internal/github/client.go` directly**: This client is for single-issue enrichment, not list queries. Building a list API would duplicate what `gh issue list` provides for free.
- **Generic HTTP source**: Over-engineered for v1 when only GitHub and file are needed.

## Decision 3: Signal Handling for Graceful Shutdown

**Decision**: The continuous loop monitors a `context.Context` derived from the existing SIGINT handler in `runRun()`. Between iterations, check `ctx.Err()` — if cancelled, exit without starting the next item. During an iteration, the existing executor already respects context cancellation (its `pollCancellation` goroutine and adapter subprocess propagation).

**Rationale**: The current `runRun()` already sets up `signal.Notify(sigChan, os.Interrupt)` with `ctx, cancel := context.WithCancel(context.Background())`. The continuous runner receives this context. Between iterations, it checks `ctx.Err()` and breaks. The executor's existing cancellation machinery handles mid-step shutdown. This reuses infrastructure without duplication.

**Alternatives Rejected**:
- **Custom signal handler in loop controller**: Would conflict with the existing handler in `runRun()` and require careful deregistration.
- **Atomic flag polling**: Less idiomatic Go than context cancellation; the context is already threaded through all execution paths.

## Decision 4: State Persistence Model

**Decision**: `ContinuousRun` is an in-memory struct (per spec clarification C3). Each child iteration uses the existing `store.CreateRun()` to get a unique run ID. No new SQLite tables.

**Rationale**: The spec explicitly states the continuous run is not persisted. Each child run is a normal pipeline run in the state store, queryable via `wave logs <run-id>`. The continuous session only needs to track iteration counts and processed item IDs for in-session dedup — ephemeral data that lives and dies with the process.

**Alternatives Rejected**:
- **New `continuous_runs` table**: Adds schema migration complexity with no clear benefit — cross-session dedup is handled by source query filters (e.g., `state=open` excludes already-closed issues).

## Decision 5: Event Emission for Observability

**Decision**: New event states `loop_iteration_start`, `loop_iteration_complete`, `loop_iteration_failed`, and `loop_summary`. These are emitted by the continuous runner using the same `event.EventEmitter` that the executor uses, extending the `Event` struct with optional iteration metadata fields.

**Rationale**: The spec requires structured progress events per iteration (FR-007). The existing `Event` struct is extensible (it already has optional fields like `Outcomes`, `RecoveryHints`, etc.). Adding `Iteration`, `TotalIterations`, `WorkItemID` fields follows the established pattern.

**Alternatives Rejected**:
- **Separate event stream**: Would fragment observability — `wave logs` and TUI should see all events in one stream.
- **Log-only (no structured events)**: Would violate Principle 10 (Observable Progress) — events must be machine-parseable.

## Decision 6: CLI Flag Integration

**Decision**: Add `--continuous`, `--source`, `--max-iterations`, `--delay`, and `--on-failure` flags to the existing `wave run` command. Mutual exclusion with `--from-step` validated early in `runRun()`.

**Rationale**: The continuous mode is a modifier on `wave run`, not a new command. Users already have muscle memory for `wave run <pipeline>`. Adding flags keeps the CLI surface flat and discoverable. The `RunOptions` struct in `run.go` is the natural place to add these fields.

**Alternatives Rejected**:
- **New `wave continuous` subcommand**: Fragments the CLI surface. Users would need to learn a new command for what is conceptually "run, but keep going".
- **Config in `wave.yaml`**: Sources and failure policies are per-invocation, not project-wide configuration. CLI flags are the right scope.

## Decision 7: Source URI Parsing

**Decision**: Simple `strings.SplitN(uri, ":", 2)` to extract provider and params. Params are `key=value` pairs separated by commas. No external URI parsing library.

**Rationale**: The grammar is simple (`github:label=bug,state=open`, `file:queue.txt`). A full URI parser adds a dependency for negligible benefit. The parser validates known keys per provider and returns structured config.

## Decision 8: Deduplication Strategy

**Decision**: In-memory `map[string]bool` tracking processed work item IDs within the continuous session. The work item ID is the unique identifier from the source (e.g., GitHub issue number, file line content).

**Rationale**: Per spec clarification C3, cross-session dedup is handled by source query filters. In-session dedup only needs to prevent the same item from being processed twice in a single continuous run — a simple map suffices. The map is checked before each iteration begins.

## Codebase Integration Points

| Component | File | Integration |
|-----------|------|-------------|
| CLI flags | `cmd/wave/commands/run.go` | Add fields to `RunOptions`, flags to `NewRunCmd()`, call `continuous.NewRunner()` when `--continuous` |
| Loop controller | `internal/continuous/runner.go` (new) | Wraps `DefaultPipelineExecutor`, manages iteration loop |
| Source abstraction | `internal/continuous/source.go` (new) | `WorkItemSource` interface, `GitHubSource`, `FileSource` |
| Source parsing | `internal/continuous/parse.go` (new) | URI parsing for `--source` flag |
| Event extension | `internal/event/emitter.go` | Add iteration metadata fields to `Event` struct |
| State store | `internal/state/store.go` | No changes — each iteration uses existing `CreateRun()` |
| Forge detection | `internal/forge/detect.go` | No changes — GitHub source uses `gh` CLI directly |
