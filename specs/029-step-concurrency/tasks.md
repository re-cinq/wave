# Tasks

## Phase 1: Data Model

- [X] Task 1.1: Add `Concurrency int` field to `Step` struct in `internal/pipeline/types.go` with YAML tag `concurrency,omitempty`
- [X] Task 1.2: Add `EffectiveConcurrency()` method on `Step` that returns capped value (min(concurrency, 10), defaulting 0/1 to 1)
- [X] Task 1.3: Add `MaxStepConcurrency int` field to `Runtime` struct in `internal/manifest/types.go` with YAML tag `max_step_concurrency,omitempty` and a `GetMaxStepConcurrency()` method defaulting to 10

## Phase 2: Validation

- [X] Task 2.1: Add mutual exclusion validation — `concurrency > 1` is incompatible with `strategy` and `iterate` fields (in pipeline validation or DAG builder)
- [X] Task 2.2: Add validation that `concurrency` is non-negative (reject negative values at manifest parse time)

## Phase 3: Core Implementation

- [X] Task 3.1: Create `internal/pipeline/concurrent.go` with `ConcurrentExecutor` struct holding reference to `*DefaultPipelineExecutor`
- [X] Task 3.2: Implement `ConcurrentExecutor.Execute(ctx, execution, step)` — creates N workspaces, injects artifacts, spawns N adapter runs via `errgroup.Group` with `SetLimit`, collects results
- [X] Task 3.3: Implement workspace creation for concurrent agents — each agent gets `<step_workspace>_agent_<N>/` path with artifact injection
- [X] Task 3.4: Implement result aggregation — collect per-agent output artifacts, merge into indexed set, write merged artifacts to primary workspace
- [X] Task 3.5: Wire from `executeStep()` — add `step.Concurrency > 1` check before retry loop, delegate to `executeConcurrentStep()` similar to `executeMatrixStep()` pattern
- [X] Task 3.6: Implement per-agent state tracking with suffixed step IDs (`<step_id>_agent_0`) and aggregate state on parent step

## Phase 4: Testing

- [X] Task 4.1: Create `internal/pipeline/concurrent_test.go` with table-driven tests for ConcurrentExecutor [P]
  - concurrency=1 → single agent (passthrough)
  - concurrency=3 → 3 agents, 3 results
  - concurrency=15 → capped at 10
  - one agent fails → step fails
  - all succeed → merged artifacts
- [X] Task 4.2: Add race condition tests in `concurrent_test.go` using `go test -race` patterns [P]
- [X] Task 4.3: Add integration test in `executor_test.go` for `concurrency` field on step [P]
- [X] Task 4.4: Add validation tests for mutual exclusion (concurrency + strategy) and negative values [P]
- [X] Task 4.5: Add types_test.go test for `EffectiveConcurrency()` method [P]

## Phase 5: Schema & Documentation

- [X] Task 5.1: Add `concurrency` to step definition in `.wave/schemas/wave-pipeline.schema.json` [P]
- [X] Task 5.2: Add `max_step_concurrency` to runtime config in `.wave/schemas/wave-manifest.schema.json` [P]
- [X] Task 5.3: Document `concurrency` field in `docs/reference/manifest-schema.md` with examples [P]
- [X] Task 5.4: Run `go test -race ./...` to validate all changes compile and pass
