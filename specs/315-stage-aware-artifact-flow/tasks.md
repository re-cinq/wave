# Tasks

## Phase 1: Core Validation Logic
- [X] Task 1.1: Add `ValidateSequenceWithStages()` to `internal/tui/compose.go` — accepts `Sequence` and `stages [][]int`, produces `CompatibilityResult` with flows only at stage boundaries. Aggregates outputs from all pipelines in stage N and matches against inputs of all pipelines in stage N+1.

## Phase 2: Rendering Updates
- [X] Task 2.1: Update `renderArtifactFlow()` in `compose_detail.go` to accept `parallel bool` and `stages [][]int` parameters and dispatch to stage-aware rendering when parallel mode is active [P]
- [X] Task 2.2: Update `renderArtifactFlowCompact()` to render stage-grouped artifact flows — show stage headers with pipeline lists, then cross-stage flow matches between stage blocks [P]
- [X] Task 2.3: Update `renderArtifactFlowFull()` to render stage-grouped artifact flows with box-drawing — stage blocks as grouped boxes, cross-stage flows between them [P]
- [X] Task 2.4: Integrate execution plan into the artifact flow rendering for parallel mode — remove separate `renderExecutionPlan` call from `Update()`, fold stage structure into the artifact flow visualization

## Phase 3: Wiring
- [X] Task 3.1: Update `ComposeDetailModel.Update()` to call `ValidateSequenceWithStages()` instead of using the pre-computed validation when parallel mode is active
- [X] Task 3.2: Update `ComposeDetailModel.SetSize()` to pass parallel/stages to the re-render path
- [X] Task 3.3: Update `ComposeDetailModel.View()` to handle the case where parallel mode has no cross-stage flows (all pipelines in one stage)

## Phase 4: Testing
- [X] Task 4.1: Write unit tests for `ValidateSequenceWithStages()` — single stage (no flows), two stages (cross-stage flows), multi-stage, artifact matching across stages [P]
- [X] Task 4.2: Write rendering tests for parallel mode — stage headers visible, no intra-stage flows, cross-stage flows correct [P]
- [X] Task 4.3: Write backward-compatibility test — sequential mode produces identical output to current behavior [P]
- [X] Task 4.4: Update existing `ComposeSequenceChangedMsg` test to verify stage-aware behavior

## Phase 5: Validation
- [X] Task 5.1: Run `go test ./internal/tui/...` and fix any failures
- [X] Task 5.2: Run `go vet ./internal/tui/...` and fix any issues
