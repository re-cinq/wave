# ADR-002: Extract StepExecutor from Pipeline Executor

## Status
Proposed (not started — file has grown since)

## Date
2026-03-12

## Implementation Status

Not started as of 2026-04-26. `internal/pipeline/executor.go` has grown from 2,493 lines (when the ADR was written) to ~6,706 lines (+168%). No `StepExecutor` struct, no `step_executor.go` file. The `StepExecutor` callback in `graph.go` (introduced by ADR-005) is unrelated — it is a function callback passed into `GraphWalker`, not the proposed extraction.

The extraction proposed here is more urgent now than when first written but interacts with ADR-005 (Graph Execution Model, accepted) and ADR-013 (Failure Taxonomy, accepted). Re-scope before starting.

## Context

The pipeline executor (`internal/pipeline/executor.go`) has grown to 2,493 lines, making it the largest single file in the Wave codebase. Its companion test file spans 3,753 lines. The executor currently owns at least 11 distinct responsibilities: DAG traversal, topological sorting, concurrency management, workspace creation, artifact injection, adapter invocation, contract validation, state persistence, relay monitoring, error recovery, and resume logic.

This growth is a natural consequence of Wave's rapid prototype phase — the project moved from v0.15.0 to v0.32.0 in approximately two weeks, adding DAG-level concurrent execution (v0.25.0), per-step artifact isolation (v0.26.0), progressive contract recovery (v0.28.0), and cross-pipeline parallelism (v0.32.0). Each feature was correctly added to the executor because that was where step lifecycle logic lived. But the accumulation has reached a tipping point:

- **Code review friction**: Every orchestration change requires reviewing a 2,493-line file where artifact injection, permission enforcement, and workspace isolation are interleaved.
- **Test brittleness**: Changing contract retry logic can break unrelated artifact injection tests because both share test setup in the same file.
- **Security audit difficulty**: The security-sensitive path (artifact injection, permission enforcement, workspace isolation) is entangled with orchestration concerns (DAG traversal, concurrency).
- **Contributor onboarding**: New contributors must internalize 11+ responsibilities in a single file before making changes.
- **Development parallelism**: Multiple contributors cannot work on different executor concerns without merge conflicts.

Wave's constitutional principles already establish that each step runs in ephemeral isolation with fresh memory. The step isolation boundary exists conceptually but is not reflected in the code structure — the same function handles both "which steps run next" and "how a single step executes."

Relevant prior decisions:
- DAG-level concurrent execution (v0.25.0) established the executor's concurrency model
- Per-step artifact isolation (v0.26.0) resolved shared-path collisions with per-step scoping
- Progressive contract recovery (v0.28.0) added retry/recovery logic to contract validation
- Persona architecture evaluation (Spec #131) identified executor decomposition as a concern

## Decision

Extract a `StepExecutor` component that owns the complete lifecycle of a single pipeline step. The existing `PipelineExecutor` retains DAG traversal, topological sorting, concurrency management, and cross-step coordination, but delegates per-step execution to the new `StepExecutor`.

The `StepExecutor` encapsulates:
1. Workspace setup (creation, mount configuration)
2. Artifact injection (resolving dependencies, copying artifacts, schema validation)
3. Runtime CLAUDE.md assembly (persona prompt, contract compliance, restrictions)
4. Adapter invocation (subprocess execution, permission enforcement)
5. Contract validation (dispatch to validators, retry strategy, recovery)
6. Output artifact extraction and archival
7. State persistence for step-level events

Dependencies are provided via constructor injection: workspace manager, contract validator, adapter, state store, event emitter, and relay monitor reference.

## Options Considered

### Option 1: Status Quo with Improved Testing

Keep the monolithic `executor.go` as-is but invest in better test coverage, documentation, and code comments. Add integration test scenarios for edge cases and use table-driven tests to reduce test file complexity without restructuring production code.

**Pros:**
- Zero refactoring risk — no production code changes
- All existing tests pass without modification
- Fastest to implement — no interface design needed
- Preserves existing mental model of the codebase
- Aligns with rapid prototype phase mandate favoring speed

**Cons:**
- The file continues growing with every new feature (relay strategies, contract types, matrix execution)
- Code review friction increases as the file grows
- Security audit surface remains entangled across concerns
- Test brittleness worsens — unrelated tests share setup and can interfere
- New contributors face a steep learning curve
- Parallel development is blocked by merge conflicts in a single file

### Option 2: Targeted Extraction of StepExecutor (Recommended)

Extract a single `StepExecutor` component owning per-step lifecycle. The `PipelineExecutor` retains DAG traversal and concurrency but delegates step execution. Dependencies provided via constructor injection.

**Pros:**
- Reduces `executor.go` by approximately 40-50%
- `StepExecutor` has a clear, testable contract: given a step spec and injected artifacts, produce validated output or error
- Creates a clean security audit boundary around the per-step execution path
- Enables independent testing of step lifecycle without mocking DAG traversal
- `PipelineExecutor` becomes focused on orchestration rather than execution details
- Achievable in a single PR without destabilizing the pipeline
- Aligns with the constitutional principle of ephemeral workspace isolation — `StepExecutor` becomes the code embodiment of the step isolation boundary

**Cons:**
- Introduces a new interface boundary between `PipelineExecutor` and `StepExecutor`
- Shared state (relay token counter, cross-step artifact registry) requires careful ownership decisions
- Some step behaviors depend on pipeline-level context (matrix strategy, conditional execution)
- Test file must be split — mechanical but time-consuming
- Does not address other architectural tensions (adapter unification, relay strategy)

### Option 3: Full Component Decomposition

Decompose the executor into five packages: `PipelineOrchestrator`, `StepExecutor`, `ArtifactManager`, `ContractCoordinator`, and `RelayCoordinator`, each with its own interface. An `EventBus` connects them for observable execution.

**Pros:**
- Maximum separation of concerns — each component has a single responsibility
- Independent testability with mock dependencies
- Granular security audit — each component reviewed independently
- Enables parallel development across components
- Aligns with Go best practices of small, focused packages

**Cons:**
- Significant effort — estimated 2-3 weeks including test migration
- Interface proliferation (5+ new interfaces) increases abstraction overhead
- Cross-component state coordination becomes complex
- Breaks all existing executor tests, requiring complete rewrites
- Risk of over-engineering for current project size (25 packages, single primary adapter)
- Conflicts with rapid prototype phase mandate
- Dependency injection graph becomes complex (5+ constructor parameters)

### Option 4: Event-Driven Executor with Plugin Architecture

Replace the imperative executor with an event-driven architecture where step lifecycle transitions emit events and handlers subscribe to them. Each concern is implemented as a plugin.

**Pros:**
- Maximum extensibility — new features are plugins, not executor modifications
- Natural fit for Wave's observable execution requirement
- Enables user-defined hooks at any lifecycle point
- Event log serves as audit trail

**Cons:**
- Highest implementation complexity — event schema, subscription management, ordering guarantees
- Debugging becomes harder — control flow is implicit rather than explicit
- Event ordering is critical and difficult (ContractValidator vs. ArtifactManager race conditions)
- Significant departure from current architecture requiring new mental model
- Over-engineered for current scale (~47 pipelines, 1 primary adapter)
- Error handling across event handlers is complex
- Effectively irreversible once adopted

## Consequences

### Positive
- `executor.go` shrinks by ~40-50%, bringing it within a manageable range for code review and comprehension
- The security-sensitive per-step execution path becomes independently auditable
- Step lifecycle testing no longer requires DAG traversal setup, reducing test complexity and brittleness
- The `StepExecutor` interface establishes a natural extension point for future decomposition (extracting `ArtifactManager` or `ContractCoordinator` later without redesigning the boundary)
- Multiple contributors can work on orchestration and step execution concurrently with reduced merge conflict risk
- New contributors can understand step execution in isolation before tackling the full pipeline orchestrator

### Negative
- Introduces an interface boundary that must be maintained as both components evolve
- Some behaviors at the step-pipeline boundary (matrix expansion, conditional execution, relay token budget) require careful API design to avoid a leaky abstraction
- One-time test migration effort — the existing 3,753-line test file must be divided between orchestrator and step executor tests

### Neutral
- The `internal/pipeline/` package retains both files — no new package is created at this stage
- The public API of the `pipeline` package does not change — `PipelineExecutor` remains the entry point for callers
- Existing pipeline definitions, personas, contracts, and manifests are unaffected
- This decision does not preclude the full component decomposition (Option 3) in the future; it establishes the first and most valuable extraction that Option 3 would also require

## Implementation Notes

- **New file**: `internal/pipeline/step_executor.go` containing the `StepExecutor` struct and its methods
- **New test file**: `internal/pipeline/step_executor_test.go` with step-lifecycle-focused tests
- **Modified file**: `internal/pipeline/executor.go` — extract per-step methods into `StepExecutor`, replace inline step execution with delegation calls
- **Modified test file**: `internal/pipeline/executor_test.go` — migrate step-focused tests to `step_executor_test.go`, retain orchestration-focused tests
- **Interface definition**: Define a `StepRunner` interface that `PipelineExecutor` depends on, allowing test doubles for orchestration tests:
  ```go
  type StepRunner interface {
      ExecuteStep(ctx context.Context, step *StepSpec, artifacts ArtifactSet) (*StepResult, error)
  }
  ```
- **Constructor injection**: `StepExecutor` receives its dependencies (workspace manager, contract validator, adapter, state store, event emitter) via `NewStepExecutor(...)` constructor
- **Shared state boundary**: The relay token counter and cross-step artifact registry remain owned by `PipelineExecutor` and are passed to `StepExecutor` per-invocation rather than shared as mutable state
- **Migration approach**: Extract incrementally — move one responsibility at a time (e.g., workspace setup first, then artifact injection, then contract validation) with `go test ./...` passing at each step
- **Validation**: Run `go test -race ./...` after the full extraction to verify no concurrency issues were introduced at the new boundary
