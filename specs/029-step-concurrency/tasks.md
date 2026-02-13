# Tasks

## Phase 1: Type Definitions and Validation

- [ ] Task 1.1: Add `Concurrency` field to `Step` struct in `internal/pipeline/types.go`
- [ ] Task 1.2: Add mutual exclusion validation in `internal/pipeline/dag.go` — reject steps with both `concurrency > 1` and `strategy.matrix` set
- [ ] Task 1.3: Add concurrency value validation in `internal/pipeline/dag.go` — reject `concurrency < 0`

## Phase 2: Core Implementation

- [ ] Task 2.1: Create `internal/pipeline/concurrency.go` with `ConcurrencyExecutor` struct and constructor
- [ ] Task 2.2: Implement `ConcurrencyExecutor.Execute()` — spawn N workers via `errgroup` with concurrency limit from `runtime.max_concurrent_workers`
- [ ] Task 2.3: Implement `ConcurrencyExecutor.createWorkerWorkspace()` — create isolated workspace per worker at `.wave/workspaces/<pipeline>/<step>/worker_<index>/`
- [ ] Task 2.4: Implement result aggregation reusing `MatrixResult` and the same output shape (`worker_results`, `worker_workspaces`, counts)
- [ ] Task 2.5: Wire `executeConcurrentStep` into `executeStep` dispatch in `internal/pipeline/executor.go` — detect `step.Concurrency > 1` before matrix check

## Phase 3: Testing

- [ ] Task 3.1: Create `internal/pipeline/concurrency_test.go` — unit tests for ConcurrencyExecutor [P]
- [ ] Task 3.2: Add validation tests to `internal/pipeline/dag.go` or create `internal/pipeline/dag_test.go` — mutual exclusion and value validation [P]
- [ ] Task 3.3: Add YAML parsing test — verify `concurrency` field is correctly deserialized from pipeline YAML [P]
- [ ] Task 3.4: Run full test suite (`go test ./...`) and fix any regressions

## Phase 4: Polish

- [ ] Task 4.1: Add event emissions for concurrency lifecycle (start, worker_start, worker_complete, worker_failed, complete)
- [ ] Task 4.2: Verify state store updates work correctly with concurrent step execution
- [ ] Task 4.3: Final validation — `go test -race ./...` to check for race conditions
