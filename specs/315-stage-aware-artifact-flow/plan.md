# Implementation Plan: Stage-Aware Artifact Flow

## 1. Objective

Make the compose detail pane's artifact flow visualization stage-aware so it only shows cross-stage artifact flows, not misleading linear flows between pipelines in the same parallel stage.

## 2. Approach

The current `ValidateSequence()` treats the pipeline sequence as a flat linear chain, producing `ArtifactFlow` entries for every adjacent pair (i, i+1). When parallel mode is enabled with stage breaks, this is wrong — pipelines within the same stage run independently.

**Strategy**: Add a new `ValidateSequenceWithStages()` function that accepts stage groupings and only produces artifact flows at stage boundaries. The rendering functions already receive the `CompatibilityResult`, so once the validation is stage-aware, the rendering automatically becomes correct.

Key design decisions:
1. **New function, not refactor**: Keep `ValidateSequence()` as-is for backward compatibility (sequential mode). Add `ValidateSequenceWithStages()` for parallel mode.
2. **Cross-stage flow definition**: The flow boundary is between the *last pipeline(s) of stage N* and the *first pipeline(s) of stage N+1*. Since parallel pipelines within a stage are independent, we aggregate all outputs from all pipelines in stage N and match against all inputs of all pipelines in stage N+1.
3. **Integrated rendering**: Merge the execution plan visualization into the artifact flow section so they don't contradict each other. In parallel mode, render stage headers with pipeline groups, then show cross-stage flows between stage blocks.

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/compose.go` | modify | Add `ValidateSequenceWithStages()` function |
| `internal/tui/compose_detail.go` | modify | Update `renderArtifactFlow`, `renderArtifactFlowCompact`, `renderArtifactFlowFull` to be stage-aware; integrate execution plan into artifact flow for parallel mode |
| `internal/tui/compose_detail_test.go` | modify | Add tests for stage-aware validation and rendering |

## 4. Architecture Decisions

### 4.1 ValidateSequenceWithStages signature

```go
func ValidateSequenceWithStages(seq Sequence, stages [][]int) CompatibilityResult
```

- Takes the same `Sequence` plus the `stages` slice from `buildStages()`
- Returns the same `CompatibilityResult` type (no new types needed)
- For each stage boundary (stage N → stage N+1): collects all outputs from all pipelines in stage N, all inputs from all pipelines in stage N+1, and matches them
- The `ArtifactFlow.SourcePipeline` becomes a stage label (e.g. "Stage 1") and `TargetPipeline` becomes the next stage label, OR we use a more descriptive approach showing the actual pipeline names involved

### 4.2 Rendering approach for parallel mode

When `parallel=true` and stages are present, the artifact flow rendering should:
1. Show each stage as a grouped block with its pipelines listed
2. Between stage blocks, show the cross-stage artifact flows
3. Remove the separate `renderExecutionPlan` call — integrate it directly into the flow visualization so there's one unified view

### 4.3 Sequential mode preservation

When `parallel=false` or `stages` is empty/single-stage, behavior is identical to current code. `ValidateSequence()` continues to be used.

## 5. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Aggregating outputs from multiple parallel pipelines may produce name collisions | Medium | Document that artifact names should be unique across a stage; use pipeline-qualified names in display |
| Integrated rendering may be complex for many stages | Low | The render functions already handle multi-boundary flows; stages just reduce the number of boundaries |
| Breaking existing tests | Medium | Run `go test ./internal/tui/...` after each change; existing tests cover sequential mode which should be unaffected |

## 6. Testing Strategy

### Unit Tests
- `TestValidateSequenceWithStages_SingleStage` — all pipelines in one stage = no flows (all parallel)
- `TestValidateSequenceWithStages_TwoStages` — verifies cross-stage flows are correct
- `TestValidateSequenceWithStages_MultiStage` — 3+ stages with mixed parallel/sequential
- `TestValidateSequenceWithStages_MatchesAcrossStage` — output from stage N pipeline matches input of stage N+1 pipeline

### Rendering Tests
- `TestRenderArtifactFlowParallel_StageHeaders` — parallel mode shows stage groupings
- `TestRenderArtifactFlowParallel_NoCrossStageFlow` — pipelines in same stage show no inter-flow
- `TestRenderArtifactFlowSequential_Unchanged` — sequential mode produces same output as before

### Integration
- Existing `ComposeSequenceChangedMsg` test updated to verify stage-aware content
