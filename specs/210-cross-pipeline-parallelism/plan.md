# Implementation Plan: Cross-Pipeline Parallelism

## Objective

Introduce a `PipelineBatchExecutor` that orchestrates multiple independent pipeline DAGs concurrently, respecting inter-pipeline dependency ordering, workspace isolation, configurable error handling, resource limits, and result aggregation — while leaving existing intra-pipeline parallelism (`executeStepBatch`) untouched.

## Approach

### Architecture: Higher-Level Orchestrator Pattern

Rather than modifying `DefaultPipelineExecutor.Execute()`, introduce a new `PipelineBatchExecutor` in its own file (`internal/pipeline/batch.go`). This follows the same composition pattern as `MatrixExecutor`: a higher-level coordinator that delegates to existing `DefaultPipelineExecutor` instances.

The batch executor:
1. Accepts a `PipelineBatchConfig` describing pipelines, dependencies, error policy, and concurrency limits
2. Computes dependency tiers using Kahn's algorithm (same approach as `MatrixExecutor.computeTiers`)
3. Executes each tier concurrently via `errgroup` with `SetLimit` for resource control
4. Each pipeline gets a fresh `DefaultPipelineExecutor` via `NewChildExecutor()`
5. Aggregates results and artifact paths for downstream injection

### Key Design Decisions

1. **Composition over modification**: `PipelineBatchExecutor` wraps `DefaultPipelineExecutor`, not extends it. No changes to existing `Execute()` flow.
2. **Tiered execution**: Pipelines are grouped into dependency tiers (like `MatrixExecutor.tieredExecution`). Independent pipelines run in parallel; dependent pipelines wait for their prerequisites.
3. **Workspace isolation**: Each pipeline already gets its own workspace under `.wave/workspaces/<pipeline-id>/`. No additional isolation needed — `DefaultPipelineExecutor` handles this.
4. **Error policies**: Two modes via `OnFailure` field:
   - `continue` (default): Other pipelines in the tier continue; only downstream dependents are skipped
   - `abort-all`: Context cancellation propagates to all running pipelines
5. **Artifact aggregation**: Cross-pipeline artifact injection via a `CrossPipelineArtifactRef` mapping. Downstream pipelines can reference `<upstream-pipeline-name>:<step-id>:<artifact>`.
6. **Resource limits**: `MaxConcurrentPipelines` in `PipelineBatchConfig`, enforced via `errgroup.SetLimit()`.
7. **Progress events**: New event states (`batch_started`, `batch_pipeline_started`, `batch_pipeline_completed`, `batch_pipeline_failed`, `batch_completed`) with a `BatchID` field.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/pipeline/batch.go` | create | `PipelineBatchExecutor` — core orchestrator |
| `internal/pipeline/batch_test.go` | create | Unit tests for batch executor |
| `internal/pipeline/batch_types.go` | create | Types: `PipelineBatchConfig`, `PipelineBatchResult`, `PipelineRunResult` |
| `internal/event/emitter.go` | modify | Add batch-related event state constants |
| `internal/manifest/types.go` | modify | Add `BatchConfig` to `Runtime` for default concurrency limits |

## Architecture Decisions

### AD-1: Separate file vs. extending executor.go
**Decision**: New file `batch.go` with a standalone `PipelineBatchExecutor`.
**Rationale**: `executor.go` is already 1800+ lines. The batch executor is a distinct concern (multi-pipeline orchestration vs. single-pipeline execution). Follows the `matrix.go` pattern.

### AD-2: errgroup vs. custom goroutine management
**Decision**: Use `errgroup` with `SetLimit()` for concurrency control.
**Rationale**: Consistent with existing patterns (`executeStepBatch`, `MatrixExecutor.Execute`). Provides context cancellation and error collection for free.

### AD-3: Dependency resolution reuse
**Decision**: Reuse Kahn's algorithm from `MatrixExecutor.computeTiers`.
**Rationale**: Exact same problem (topological ordering with parallel tiers). Extract into a shared utility or duplicate the small algorithm.

### AD-4: Cross-pipeline artifact injection
**Decision**: Post-execution artifact path registration in a shared `BatchArtifactRegistry` map, queried by downstream pipeline setup.
**Rationale**: Keeps the existing `injectArtifacts` flow unchanged. The batch executor just populates the registry before launching each tier.

### AD-5: Stub interfaces for #208/#209 dependencies
**Decision**: Define a `PipelineBatchConfig` input type that the proposal engine (#208) and TUI (#209) will produce. For now, tests construct this directly.
**Rationale**: Decouples implementation. The batch executor doesn't need to know how the config was generated.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| SQLite contention from concurrent pipeline state writes | High | WAL mode already enabled (store.go:129); single-connection pool mitigates write contention. If needed, batch executor can use a shared store with serialized writes |
| Event emission ordering from concurrent pipelines | Medium | Events already include PipelineID; consumers must handle interleaved events. Add BatchID for grouping |
| Resource exhaustion with many concurrent adapters | High | `MaxConcurrentPipelines` defaults to a conservative value (e.g., 3). Each adapter spawns a subprocess |
| Dependencies #208/#209 not yet implemented | Low | Batch executor accepts a `PipelineBatchConfig` struct — no coupling to proposal/TUI code |

## Testing Strategy

1. **Unit tests** (`batch_test.go`):
   - Independent pipelines execute concurrently (verify all complete)
   - Tiered execution respects dependencies (pipeline B waits for pipeline A)
   - `abort-all` error policy cancels remaining pipelines
   - `continue` error policy allows independent pipelines to finish
   - Failed pipeline skips downstream dependents
   - `MaxConcurrentPipelines` limits goroutine count
   - Cross-pipeline artifact injection works
   - Empty batch (zero pipelines) handled gracefully
   - Cycle detection in pipeline dependencies

2. **Integration test patterns** (mock adapter):
   - End-to-end batch execution with mock adapters
   - Progress event emission verified
   - State store records per-pipeline state

3. **Race detector**: All tests must pass with `go test -race`
