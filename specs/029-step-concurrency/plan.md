# Implementation Plan: Step Concurrency

## Objective

Add a `concurrency` field to pipeline step definitions so that a single step can spawn multiple identical agent instances executing in parallel, each in its own isolated workspace. Results are collected and merged following the same pattern as the existing `MatrixExecutor`.

## Approach

Extend the existing pipeline execution infrastructure with a `ConcurrencyExecutor` that reuses patterns from `MatrixExecutor`. The key difference: matrix execution fans out over *different items*, while concurrency fans out over *identical copies* of the same step. The implementation leverages the same `errgroup`-based concurrency, workspace isolation, and result aggregation patterns.

### High-Level Flow

1. YAML parsing picks up `concurrency: N` on a step
2. DAG validation rejects steps with both `concurrency` and `strategy.matrix`
3. `executeStep` detects `step.Concurrency > 1` and delegates to `executeConcurrentStep`
4. `executeConcurrentStep` spawns N goroutines via `errgroup` (capped by `runtime.max_concurrent_workers`)
5. Each goroutine creates an isolated workspace, runs `runStepExecution`, and sends results to a channel
6. Results are aggregated into the execution state using the same shape as matrix results

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | modify | Add `Concurrency int` field to `Step` struct |
| `internal/pipeline/dag.go` | modify | Add validation: `concurrency` and `strategy.matrix` are mutually exclusive |
| `internal/pipeline/executor.go` | modify | Add `executeConcurrentStep` method, wire into `executeStep` dispatch |
| `internal/pipeline/concurrency.go` | create | `ConcurrencyExecutor` with worker spawning, workspace creation, result aggregation |
| `internal/pipeline/concurrency_test.go` | create | Unit tests for concurrency executor |
| `internal/pipeline/dag_test.go` | create | Tests for mutual exclusion validation |
| `internal/manifest/types.go` | no change | `Runtime.MaxConcurrentWorkers` already exists |

## Architecture Decisions

### 1. Separate file for ConcurrencyExecutor

Following the pattern established by `matrix.go`, the concurrency execution logic lives in its own file (`concurrency.go`). This keeps `executor.go` focused on orchestration dispatch.

### 2. Reuse MatrixResult and aggregation pattern

The `MatrixResult` struct and `aggregateResults` logic are already well-suited. The `ConcurrencyExecutor` will reuse `MatrixResult` directly and produce output in the same shape. This means downstream steps consuming `worker_results` work identically regardless of whether the upstream used matrix or concurrency.

### 3. Worker workspace naming

Workers get workspaces at `.wave/workspaces/<pipeline>/<step>/worker_<index>/`, matching the matrix worker workspace convention.

### 4. Global concurrency cap

`runtime.max_concurrent_workers` (already in `manifest.Runtime`) serves as the upper bound. If a step declares `concurrency: 20` but `max_concurrent_workers: 5`, only 5 run in parallel.

### 5. Prompt is identical across workers

Unlike matrix execution (which injects `{{ task }}` per item), all concurrent workers receive the exact same prompt built by `buildStepPrompt`. Each worker's isolation comes from having a separate workspace, not different prompts.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Resource exhaustion with high concurrency | Medium | High | Cap via `runtime.max_concurrent_workers`, default max of 10 |
| Workspace conflicts when using worktree type | Low | High | Each worker gets its own worktree path; #76 confirms this is solved |
| Result aggregation confusion with existing matrix | Low | Medium | Use identical result shape; document mutual exclusion clearly |
| State store concurrent writes | Low | Medium | SQLite WAL mode handles concurrent writes; step state updated after all workers complete |

## Testing Strategy

1. **Unit tests** (`concurrency_test.go`):
   - Concurrency=1 behaves identically to non-concurrent execution
   - Concurrency=3 spawns 3 workers with isolated workspaces
   - Concurrency capped by `max_concurrent_workers`
   - Results properly aggregated (worker_results, success/fail counts)
   - Partial failure: first failure cancels remaining workers

2. **Validation tests** (`dag_test.go`):
   - Step with both `concurrency` and `strategy.matrix` is rejected
   - Step with `concurrency: 0` or `concurrency: 1` is treated as non-concurrent
   - Step with `concurrency: -1` is rejected

3. **YAML parsing tests**:
   - `concurrency` field parsed correctly from pipeline YAML
   - Missing `concurrency` field defaults to 0 (non-concurrent)
