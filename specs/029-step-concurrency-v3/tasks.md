# Tasks

## Phase 1: Type Definitions and Parsing

- [X] Task 1.1: Add `Concurrency int` field to `Step` struct in `internal/pipeline/types.go` with yaml tag `concurrency,omitempty`
- [X] Task 1.2: Add `MaxConcurrency int` field to `Runtime` struct in `internal/manifest/types.go` with yaml tag `max_concurrency,omitempty` and a `GetMaxConcurrency()` method that defaults to 10
- [X] Task 1.3: Add DAG validation in `internal/pipeline/dag.go` to reject steps that set both `concurrency > 1` and a matrix `strategy`

## Phase 2: Core Implementation

- [X] Task 2.1: Create `internal/pipeline/concurrency.go` with `ConcurrencyExecutor` struct and `Execute()` method [P]
  - Constructor: `NewConcurrencyExecutor(executor *DefaultPipelineExecutor)`
  - `Execute(ctx, execution, step)` orchestrates N agents via `errgroup.WithContext()`
  - Per-agent workspace creation under `.wave/workspaces/<pipeline>/<step>/agent_<N>/`
  - Artifact injection per agent workspace
  - Calls `executor.runStepExecution()` for each agent
  - Result aggregation into `agent_results`, `agent_workspaces`, `total_agents`, `success_count`, `fail_count`
  - Respects `Runtime.MaxConcurrency` cap via `errgroup.SetLimit()`
  - Emits events: `concurrency_start`, `concurrency_agent_start`, `concurrency_agent_complete`, `concurrency_agent_failed`, `concurrency_complete`, `concurrency_failed`
- [X] Task 2.2: Add routing in `executeStep()` in `internal/pipeline/executor.go` — check `step.Concurrency > 1` before the matrix strategy check, dispatch to `executeConcurrentStep()` [P]
  - Add `executeConcurrentStep()` method that creates `ConcurrencyExecutor`, calls `Execute()`, and handles state transitions (similar to `executeMatrixStep()`)

## Phase 3: Testing

- [X] Task 3.1: Create `internal/pipeline/concurrency_test.go` with table-driven tests [P]
  - TestConcurrencyExecutor_BasicExecution: 3 agents, all succeed
  - TestConcurrencyExecutor_FailFast: 1 agent fails, step fails
  - TestConcurrencyExecutor_MaxConcurrencyCap: concurrency capped by Runtime.MaxConcurrency
  - TestConcurrencyExecutor_SingleAgent: concurrency=1 falls through to normal execution
  - TestConcurrencyExecutor_ZeroConcurrency: concurrency=0 falls through to normal execution
  - TestConcurrencyExecutor_ResultAggregation: verify merged result format
  - TestConcurrencyExecutor_WorkspaceIsolation: each agent gets unique path
- [X] Task 3.2: Add DAG validation test for concurrency+matrix mutual exclusion [P]
- [X] Task 3.3: Run `go test -race ./internal/pipeline/...` to verify no race conditions

## Phase 4: Polish

- [X] Task 4.1: Verify all existing tests still pass (`go test ./...`)
- [X] Task 4.2: Run `go vet ./...` and lint checks
