# fix(webui): DAG layout places rework_only steps at layer 0

**Issue**: [re-cinq/wave#691](https://github.com/re-cinq/wave/issues/691)
**Parent**: #687 (item 4)
**Author**: nextlevelshit
**State**: OPEN

## Problem

Steps with `rework_only: true` and no explicit `dependencies` (like `fix-implement` in `impl-issue`) get placed at layer 0 by Kahn's algorithm in `assignLayers()`, stacking them alongside the first real step. This makes the pipeline graph look broken.

## Expected Behavior

Rework-only steps should either:
- **Option A (preferred)**: Be hidden from the DAG entirely -- they're internal retry mechanics, not user-visible workflow steps
- **Option B**: Be positioned adjacent to their parent step (the step that references them via `retry.rework_step`)

## Files to Change

- `internal/webui/dag.go` -- filter `rework_only` steps in `ComputeDAGLayout` or `assignLayers`
- `internal/webui/handlers_pipelines.go` -- where `DAGStepInput` is built, skip or annotate rework-only steps
- `internal/webui/handlers_runs.go` -- also builds `DAGStepInput` for run detail pages

## Acceptance Criteria

- [ ] `rework_only` steps don't appear in the DAG visualization
- [ ] Or if shown, they're placed next to their parent step with a distinct visual style
- [ ] Existing tests pass
