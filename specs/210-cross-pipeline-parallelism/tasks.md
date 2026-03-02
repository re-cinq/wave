# Tasks

## Phase 1: Types and Foundation

- [X] Task 1.1: Create `internal/pipeline/batch_types.go` with `PipelineBatchConfig`, `PipelineBatchResult`, `PipelineRunResult`, and error policy types
- [X] Task 1.2: Add batch event state constants to `internal/event/emitter.go` (`StateBatchStarted`, `StateBatchPipelineStarted`, `StateBatchPipelineCompleted`, `StateBatchPipelineFailed`, `StateBatchCompleted`)
- [X] Task 1.3: Add `BatchConfig` to `internal/manifest/types.go` under `Runtime` struct with `MaxConcurrentPipelines` and `DefaultOnFailure` fields

## Phase 2: Core Implementation

- [X] Task 2.1: Implement `PipelineBatchExecutor` struct and constructor in `internal/pipeline/batch.go` [P]
- [X] Task 2.2: Implement dependency tier computation — extract or replicate Kahn's algorithm for pipeline-level DAG resolution [P]
- [X] Task 2.3: Implement `ExecuteBatch` method — tier-based execution loop with `errgroup.SetLimit()` for concurrency control
- [X] Task 2.4: Implement `continue` error policy — failed pipelines skip downstream dependents but don't abort siblings
- [X] Task 2.5: Implement `abort-all` error policy — context cancellation propagates to all running pipelines in batch
- [X] Task 2.6: Implement `BatchArtifactRegistry` for cross-pipeline artifact path registration and downstream injection
- [X] Task 2.7: Implement per-pipeline progress event emission with `BatchID` field for grouping
- [X] Task 2.8: Implement result aggregation — collect `PipelineRunResult` per pipeline with status, artifacts, token count, duration

## Phase 3: Testing

- [X] Task 3.1: Write unit tests for `PipelineBatchConfig` validation (empty batch, cycle detection, missing dependencies) [P]
- [X] Task 3.2: Write unit tests for tier computation (independent pipelines in single tier, dependent pipelines across tiers) [P]
- [X] Task 3.3: Write unit tests for `ExecuteBatch` with mock adapter — verify concurrent execution of independent pipelines [P]
- [X] Task 3.4: Write unit tests for `continue` error policy — verify failed pipeline doesn't abort siblings, downstream dependents skipped
- [X] Task 3.5: Write unit tests for `abort-all` error policy — verify context cancellation on failure
- [X] Task 3.6: Write unit tests for `MaxConcurrentPipelines` resource limiting
- [X] Task 3.7: Write unit tests for cross-pipeline artifact injection
- [X] Task 3.8: Run `go test -race ./internal/pipeline/...` to validate no data races

## Phase 4: Polish

- [X] Task 4.1: Verify `go test ./...` passes with all existing tests unaffected
- [X] Task 4.2: Verify `go vet ./...` passes
