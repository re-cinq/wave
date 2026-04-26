# ADR-005: Graph Execution Model

## Status

Accepted

## Date

2026-03-27 (proposed and shipped same day in commit `3bca8bd2`) — 2026-04-26 (accepted)

## Implementation Status

Landed:
- `executeGraphPipeline()` in `internal/pipeline/executor.go` (~line 1097) drives graph-mode execution.
- `GraphWalker` in `internal/pipeline/graph.go` (~278 LOC) handles backward edges with `max_visits` enforcement.
- `Step.Type` accepts `"conditional"` and `"command"`; `Step.Edges []EdgeConfig` and `Step.MaxVisits` / `Pipeline.MaxStepVisits` are wired through.
- Dual-mode dispatch: graph mode auto-activates when steps declare `edges` or `type: conditional`/`command`; otherwise legacy `TopologicalSort`/`executeStepBatch` path runs.
- `internal/pipeline/dag.go` retained for the strict-DAG path; cycles are gated by graph-mode detection.

## Context

Wave's pipeline executor uses a strict DAG execution model: a DFS topological sort in `dag.go` rejects all backward edges, and the ~3,400 LOC monolithic `executor.go` drives execution through a `findReadySteps()` / `executeStepBatch()` polling loop. This model cannot express three capabilities that issue #577 identifies as necessary:

1. **Cycles with bounded visits** -- Pipelines need backward edges (e.g., retry loops, iterative refinement) with `max_visits` guards to prevent infinite execution. The only existing backward-flow mechanism is rework, which is special-cased rather than general.

2. **Conditional edges** -- Steps should be able to route execution along different edges based on prior step outcomes. `BranchConfig` exists for sub-pipeline routing, but edge-level conditions would operate at the graph traversal layer using the template expression engine.

3. **Command steps** -- `ExecConfig.Type="command"` already exists in the type system but has no executor implementation. These steps need a non-adapter execution path that runs shell commands under sandbox enforcement.

The topological sort fundamentally assumes acyclicity, creating an impedance mismatch with any backward-edge support. Additionally, ADR-002 has proposed extracting a `StepExecutor` from the monolithic executor, and the chosen approach here should align with -- or at least not impede -- that decomposition.

Codebase analysis identified 26 affected files across 16 components, with state persistence, progress display, and pipeline resume being the trickiest areas for cycle support.

## Decision

**Extract a GraphScheduler component that encapsulates all step scheduling logic behind a clean interface, while retaining the existing executor as an orchestration shell** (Option 3: Hybrid).

The new `GraphScheduler` replaces both `TopologicalSort()` and `findReadySteps()` with a single `NextSteps(completedStep, outcome)` method. It owns dependency resolution, visit counting, condition evaluation, and backward edge handling. The executor retains its orchestration role: workspace setup, artifact injection, adapter/command invocation, and contract validation.

Command steps are handled by adding a type-dispatch branch in the executor's step execution path, bypassing adapter invocation while maintaining sandbox enforcement.

## Options Considered

### Option 1: Full Graph Walker Replacement

Replace the entire `findReadySteps` / `executeStepBatch` polling loop and DFS topological sort with a dedicated `GraphWalker` that natively understands cycles, conditional edge routing, and step type dispatch.

**Pros:**
- Cycles, conditions, and linear execution are all first-class in a unified scheduler
- Visit counting is core state, not bolted-on counters
- Eliminates the topological-sort / backward-edge impedance mismatch
- Aligns with ADR-002 StepExecutor extraction
- Simplifies future extensions (priority edges, dynamic graph modification)

**Cons:**
- Largest implementation effort -- rewrites the core scheduling loop that works correctly for all existing pipelines
- High regression risk across ~3,400 LOC of executor code
- All existing composition primitives (Branch, Iterate, Gate, Loop, Aggregate) must be re-implemented in the walker
- 26 affected files across 16 components -- large blast radius for a single change

**Effort:** Large | **Risk:** High | **Reversibility:** Moderate

### Option 2: Incremental Augmentation of Existing Executor

Keep the current polling loop and topological sort. Layer on cycle support (backward edges with `max_visits`), condition evaluation in `findReadySteps`, and a command step branch in `runStepExecution` as independent, incremental changes.

**Pros:**
- Lowest risk -- each feature is an independent, testable increment
- Backward compatibility guaranteed by construction
- Smallest diff per change; features can ship independently
- Command step support is trivially addable since `ExecConfig.Type="command"` already exists
- Rework mechanism proves the pattern -- backward flow as special-cased logic works

**Cons:**
- Increases complexity of an already complex executor; `findReadySteps` grows into a multi-concern scheduler
- Topological sort assumes acyclicity -- bolting cycles onto it is a conceptual contradiction
- Visit counting, condition evaluation, and backward edge re-enqueueing are three concerns mixed into one loop
- Makes future ADR-002 extraction harder by further entangling scheduling with execution logic
- Technical debt: topological sort becomes misleading (suggests acyclicity but runtime allows cycles)

**Effort:** Medium | **Risk:** Medium | **Reversibility:** Easy

### Option 3: Hybrid -- Extract Graph Scheduler, Retain Executor Shell (Recommended)

Extract a `GraphScheduler` component behind a clean interface. The existing executor delegates "what runs next?" to the scheduler via `NextSteps(completedStep, outcome)`. The scheduler replaces both `TopologicalSort` and `findReadySteps`. Command steps are handled by type-dispatch in the executor.

**Pros:**
- Clean separation: scheduler owns graph semantics, executor owns step lifecycle
- Scheduler can be developed and tested independently against pure graph structures
- Backward compatible by design -- scheduler output for acyclic graphs is identical to `findReadySteps`
- Directly enables ADR-002 StepExecutor extraction along the same boundary
- Visit counting, `max_visits` enforcement, and condition evaluation are co-located
- Incremental migration: build scheduler, wire it in replacing `findReadySteps`, then add cycle/condition support
- Resume support is cleaner -- scheduler state is a well-defined, serializable snapshot

**Cons:**
- Requires defining a scheduler interface that captures all composition primitives (Branch, Iterate, Gate, Loop, Aggregate)
- Medium effort -- less than full rewrite but more than incremental patches
- Boundary between scheduler and executor must be carefully designed
- Composition primitives currently mix scheduling and execution -- extraction requires untangling
- Two-phase migration: refactor first (no user-visible value), then add features

**Effort:** Medium | **Risk:** Medium | **Reversibility:** Moderate

### Option 4: Event-Driven State Machine

Replace the polling loop with an event-driven engine where each step completion emits an event triggering downstream edge evaluation. A central state machine processes events, evaluates conditions, enforces visit limits, and enqueues runnable steps.

**Pros:**
- Most flexible model -- naturally supports cycles, conditions, dynamic graphs, and priority scheduling
- Every state transition is an observable event; integrates well with existing `EventEmitter`
- Clean mapping to SQLite state persistence
- Concurrent step execution falls out naturally

**Cons:**
- Ground-up rewrite of the execution engine -- largest departure from current architecture
- Event-driven systems are harder to debug; causality chains cross event boundaries
- No incremental migration path; all 26 files would need updating simultaneously
- Over-engineered for the immediate requirements (`max_visits` cycles and simple conditions)
- Risk of second-system effect

**Effort:** Epic | **Risk:** High | **Reversibility:** Difficult

## Consequences

### Positive

- The monolithic executor is decomposed along its natural scheduling/execution boundary, reducing the cognitive load of working in the ~3,400 LOC file
- Cycle support with `max_visits` and conditional edges land in their natural home (the scheduler), avoiding the conceptual contradiction of cycles in a topological sort
- The scheduler can be unit tested against pure graph structures without spinning up workspaces or adapters, dramatically improving testability
- ADR-002 StepExecutor extraction becomes easier since scheduling concerns are already separated
- Existing linear pipelines require no changes -- backward compatibility is structural, not conditional
- Pipeline resume benefits from explicit, serializable scheduler state (visit counts, edge conditions)

### Negative

- The refactoring phase (extracting the scheduler) delivers no user-visible value on its own -- it is pure architectural investment
- Composition primitives (Branch, Iterate, Gate, Loop, Aggregate) that currently mix scheduling and execution logic must be untangled, which is non-trivial design work
- A poorly drawn scheduler/executor boundary could leave scheduling logic stranded in the executor or pull execution concerns into the scheduler

### Neutral

- The `GraphScheduler` interface definition becomes a key design artifact that must be reviewed carefully before implementation begins
- Fresh-memory constitutional requirement (no chat history inheritance between steps) is unaffected -- each step visit remains isolated regardless of scheduling model
- Sandbox enforcement for command steps operates at the executor layer and is independent of the scheduling changes

## Implementation Notes

### Phase 1: Extract GraphScheduler (Refactor)

1. Define the `GraphScheduler` interface in a new `internal/scheduler/` package:
   - `NewGraphScheduler(dag *DAG, opts ...Option) *GraphScheduler`
   - `NextSteps(completed StepID, outcome StepOutcome) []StepID`
   - `State() SchedulerState` (for persistence/resume)
   - `RestoreState(SchedulerState)` (for resume from checkpoint)
2. Implement the scheduler to replicate current `findReadySteps` behavior exactly -- dependency tracking, batch readiness detection, composition primitive support
3. Wire the scheduler into `executor.go`, replacing `findReadySteps` calls with `scheduler.NextSteps`
4. Validate with `go test ./...` -- all existing pipeline tests must pass with zero behavioral changes

### Phase 2: Add New Capabilities

5. **Command steps**: Add type-dispatch in executor's `runStepExecution` -- when `ExecConfig.Type == "command"`, execute via sandbox subprocess instead of adapter invocation
6. **Cycle support**: Modify DAG validation in `dag.go` to accept backward edges annotated with `max_visits`. Add visit counting to `GraphScheduler`. Re-enqueue steps when visit count < `max_visits`
7. **Conditional edges**: Add edge condition evaluation to `GraphScheduler.NextSteps` using the template expression engine. Edges without conditions always fire (current behavior)

### Key Files

- `internal/pipeline/executor.go` -- extract scheduling logic, add command step dispatch
- `internal/pipeline/dag.go` -- relax cycle rejection for annotated backward edges
- `internal/scheduler/scheduler.go` -- new package, core scheduler implementation
- `internal/scheduler/scheduler_test.go` -- unit tests against pure graph structures
- `internal/state/` -- extend state persistence for scheduler snapshots (visit counts, edge states)
- `internal/manifest/types.go` -- ensure `max_visits` and edge condition fields are defined
- `internal/display/` and `internal/event/` -- update progress reporting for revisited steps

### Migration

No data migration required. Existing `wave.yaml` manifests continue to work unchanged. New manifest fields (`max_visits`, edge conditions) are optional and additive. The scheduler extraction is a pure internal refactor with no user-facing API changes.
