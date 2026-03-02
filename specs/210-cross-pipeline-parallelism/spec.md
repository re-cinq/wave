# feat(pipeline): cross-pipeline parallelism — orchestrate multiple independent pipeline runs concurrently

**Issue**: [#210](https://github.com/re-cinq/wave/issues/210)
**Parent**: #184
**Labels**: enhancement, needs-design, pipeline, priority: high
**Author**: nextlevelshit
**State**: OPEN

## Summary

Implement cross-pipeline parallelism to orchestrate multiple independent pipeline runs concurrently. The existing parallel execution (`executeStepBatch` in `internal/pipeline/executor.go:388` using `errgroup`) operates within a single pipeline DAG — running concurrent steps of one pipeline. This issue addresses the remaining gap: running multiple entire pipeline DAGs simultaneously when they have independent inputs and no shared workspace requirements. This is the execution engine that powers the parallel groups selected in the interactive TUI.

## Acceptance Criteria

- [ ] Multiple independent pipelines can execute concurrently in isolated workspaces
- [ ] Cross-pipeline execution respects dependency ordering from the proposal (pipeline B waits for pipeline A if dependent)
- [ ] Each parallel pipeline gets its own isolated workspace (no shared state unless explicitly configured via artifact injection)
- [ ] Error handling: failure in one parallel pipeline does not abort others by default (configurable)
- [ ] Aggregated results from parallel pipelines are available for downstream dependent pipelines via artifact injection
- [ ] Resource management: configurable maximum concurrent pipeline count to avoid overwhelming the host
- [ ] Progress events are emitted per-pipeline for monitoring (extends existing `internal/event/` system)
- [ ] Existing intra-pipeline parallelism (`executeStepBatch`) continues to work unchanged within each concurrent pipeline
- [ ] Integration with the existing state management (`internal/state/`) — each pipeline run has its own state tracking

## Dependencies

- #208 — Pipeline proposal engine (determines which pipelines can run in parallel)
- #209 — TUI selection (user selects parallel groups)

## Scope Notes

- **In scope**: Concurrent pipeline execution, workspace isolation for parallel runs, error handling across pipelines, result aggregation, resource limits, progress event emission
- **Out of scope**: Intra-pipeline parallelism changes (already works via `executeStepBatch`), shared mutable state between parallel pipelines (violates isolation principle), distributed execution across multiple machines
- **Design consideration**: May extend `internal/pipeline/executor.go` with a new `ExecutePipelineBatch` method analogous to `executeStepBatch`, or introduce a higher-level orchestrator that composes multiple `PipelineExecutor` instances
- **Overlap note**: #201 (continuous/long-running execution) and #29 (concurrency property for steps) are related but distinct — this issue is about running multiple full pipelines concurrently, not continuous iteration or step-level concurrency
