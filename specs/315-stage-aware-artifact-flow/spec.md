# fix(tui): compose detail pane ignores stage breaks and parallel structure

**Issue**: [#315](https://github.com/re-cinq/wave/issues/315)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

## Problem

The compose mode right pane shows artifact flow as a simple linear chain between adjacent pipelines, completely ignoring stage breaks and parallel structure. This is misleading — pipelines within the same parallel stage run independently and don't flow artifacts into each other.

**Current behavior:**
```
gh-refresh → gh-implement        ← wrong: these are in different stages
gh-implement → gh-pr-review      ← wrong: shows linear flow
gh-pr-review → security-scan     ← wrong: these may be in same parallel stage
security-scan → doc-fix
```

**Expected behavior:**
- Only show artifact flows between cross-stage boundaries
- Show stage groupings visually (parallel pipelines side-by-side)
- Don't show flows between pipelines within the same parallel stage

## Root Cause

`renderArtifactFlow()` in `compose_detail.go` always validates and renders adjacent-pair flows via `CompatibilityResult.Flows`, which is computed by `ValidateSequence()` as a simple linear chain. Neither function is stage-aware.

The `renderExecutionPlan()` at the bottom does show stages, but the artifact flow section above contradicts it.

## Scope

- `internal/tui/compose_detail.go` — `renderArtifactFlow`, `renderArtifactFlowCompact`, `renderArtifactFlowFull`
- `internal/tui/compose.go` — `ValidateSequence()` needs a stage-aware variant
- `internal/tui/compose_detail_test.go` — update tests

## Acceptance Criteria

- [ ] Artifact flow only shown between stage boundaries (last pipeline of stage N → first pipeline of stage N+1)
- [ ] Pipelines within same parallel stage shown as independent (no inter-flow arrows)
- [ ] Execution plan and artifact flow sections are visually integrated (not contradictory)
- [ ] Non-parallel (sequential) mode continues to work as before
