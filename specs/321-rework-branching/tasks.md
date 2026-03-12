# Tasks

## Phase 1: Type Definitions and Schema

- [X] Task 1.1: Add `ReworkConfig` struct to `internal/pipeline/types.go` with `TargetStep`, `TargetPipeline`, `MaxReworkDepth`, `InjectFailureContext` fields
- [X] Task 1.2: Add `Rework *ReworkConfig` field to `RetryConfig` struct in `internal/pipeline/types.go`
- [X] Task 1.3: Add `StateReworking = "reworking"` constant to `internal/pipeline/types.go`
- [X] Task 1.4: Add `ReworkContext` struct to `internal/pipeline/types.go` with attempt history, partial artifacts, original step metadata, and depth counter
- [X] Task 1.5: Add `Validate()` method to `ReworkConfig` enforcing mutual exclusivity of `TargetStep`/`TargetPipeline` and valid depth limits
- [X] Task 1.6: Add `StateReworking` to `internal/state/store.go` state constants

## Phase 2: Event and State Infrastructure

- [X] Task 2.1: Add rework event state constants (`StateReworkStarted`, `StateReworkCompleted`, `StateReworkFailed`) to `internal/event/types.go` [P]
- [X] Task 2.2: Add `ReworkRecord` type to `internal/state/types.go` with `RunID`, `StepID`, `TargetStep`, `ReworkDepth`, `FailureContext` fields [P]
- [X] Task 2.3: Add DB migration for `step_rework_history` table tracking rework transitions [P]
- [X] Task 2.4: Add `RecordReworkTransition` and `GetReworkHistory` methods to `StateStore` interface and implementation [P]

## Phase 3: DAG Validation

- [X] Task 3.1: Extend `DAGValidator.ValidateDAG` to validate rework target steps exist when `on_failure: rework` with `target_step` is set
- [X] Task 3.2: Add cycle detection for rework references (A→B→A rework chains) in `DAGValidator`
- [X] Task 3.3: Add `ReworkConfig.Validate()` call during pipeline YAML loading in `dag.go`

## Phase 4: Core Executor Implementation

- [X] Task 4.1: Add `"rework"` case to the `on_failure` switch in `executor.go:577` — call new `executeRework` method
- [X] Task 4.2: Implement `executeRework(ctx, execution, step, lastErr)` method that: serializes `ReworkContext` to artifact, resolves rework target, executes target step/pipeline
- [X] Task 4.3: Add rework depth tracking to `PipelineExecution` (new `ReworkDepths map[string]int` field)
- [X] Task 4.4: Implement rework context artifact injection — write `ReworkContext` as `.wave/artifacts/rework_context` JSON in target step's workspace
- [X] Task 4.5: Emit rework events (started, completed, failed) with structured metadata
- [X] Task 4.6: Handle rework to sub-pipeline (when `target_pipeline` is set) — delegate to sub-pipeline loader

## Phase 5: Resume Support

- [X] Task 5.1: Extend `loadResumeState` in `resume.go` to detect rework-in-progress state from DB
- [X] Task 5.2: Reconstruct `ReworkContext` from persisted state when resuming a rework step

## Phase 6: Error Handling

- [X] Task 6.1: Add `ReworkError` type to `errors.go` wrapping original failure + rework target info [P]
- [X] Task 6.2: Add rework troubleshooting guidance to `ErrorMessageProvider.FormatPhaseFailureError` [P]

## Phase 7: Testing

- [X] Task 7.1: Unit tests for `ReworkConfig.Validate()` — mutually exclusive fields, depth limits, zero-value defaults [P]
- [X] Task 7.2: Unit tests for DAG validation with rework targets — valid targets, missing targets, circular references [P]
- [X] Task 7.3: Unit tests for `ReworkContext` serialization/deserialization [P]
- [X] Task 7.4: Integration test: step exhausts retries → rework target step executes with failure context
- [X] Task 7.5: Integration test: rework target succeeds → pipeline continues normally
- [X] Task 7.6: Integration test: rework target fails → pipeline fails with rework error (depth limit)
- [X] Task 7.7: Integration test: rework to sub-pipeline loads and executes correctly
- [X] Task 7.8: Integration test: resume from rework-in-progress state
- [X] Task 7.9: Edge case test: `max_rework_depth: 0` acts like `fail`
- [X] Task 7.10: Edge case test: concurrent steps where one enters rework

## Phase 8: Polish

- [X] Task 8.1: Add rework state display support to event emitter output
- [X] Task 8.2: Verify `go test ./...` passes with race detector
- [X] Task 8.3: Verify `go vet ./...` passes
