# Implementation Plan: DAG layout hides rework_only steps

## Objective

Filter out `rework_only: true` steps from the DAG visualization in the web UI so they don't stack at layer 0 and break the graph layout. Option A (hide entirely) is preferred per the issue.

## Approach

Add a `ReworkOnly` field to `DAGStepInput` and filter rework-only steps out at the point where `DAGStepInput` slices are built -- in both `handlers_pipelines.go` (pipeline detail page) and `handlers_runs.go` (run detail page). This keeps `ComputeDAGLayout` and `assignLayers` generic and unaware of step semantics. Dependencies pointing to filtered-out steps are also stripped so the remaining graph stays valid.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/webui/handlers_pipelines.go` | modify | Skip `rework_only` steps when building `dagSteps` in `handlePipelineDetailPage` |
| `internal/webui/handlers_runs.go` | modify | Skip `rework_only` steps when building `dagSteps` in run detail handler |
| `internal/webui/dag_test.go` | modify | Add test verifying rework-only steps are excluded from layout |

## Architecture Decisions

1. **Filter at call site, not in `ComputeDAGLayout`**: The DAG layout engine is a pure graph algorithm. Filtering by pipeline-specific semantics (`rework_only`) belongs in the handlers that translate `pipeline.Step` to `DAGStepInput`. This keeps `dag.go` reusable.
2. **Strip dangling dependency references**: When a rework-only step is filtered out, any other step that lists it as a dependency must have that reference removed. In practice this shouldn't happen (rework-only steps are targets of `retry.rework_step`, not dependency sources), but defensive cleanup prevents broken graphs.
3. **No `DAGStepInput` schema change needed**: Since filtering happens before `DAGStepInput` construction, the struct doesn't need a `ReworkOnly` field.

## Risks

| Risk | Mitigation |
|------|------------|
| A normal step depends on a rework-only step | Strip the dependency reference; log a warning. In practice this doesn't happen -- rework steps are triggered by retry, not the DAG. |
| Rework steps have edges that reference them | The `edgeConditionMap` and `addEdge` in `ComputeDAGLayout` already skip edges to unknown nodes (no position found). |

## Testing Strategy

- Add a unit test to `dag_test.go` confirming that when a rework-only step is present in the input slice, the handler-level filtering produces a `DAGStepInput` slice without it.
- Run existing `dag_test.go` tests to confirm no regressions.
- Run `go test ./internal/webui/...` for full package coverage.
