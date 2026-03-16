# feat(tui): complete guided flow — suggest detail pane and fleet integration

**Issue**: [#443](https://github.com/re-cinq/wave/issues/443)
**Labels**: enhancement
**Author**: nextlevelshit

## Description

TUI guided flow (issues #248, #209) is ~75% complete. The state machine works but:

### Missing
1. **Suggest detail pane is a stub** — `renderSuggestDetail()` only shows title + rationale, no DAG rendering, no pipeline details, no "why suggested" explanation
2. **Fleet view not integrated** — parallel execution monitor not wired into guided flow
3. **Phase-view binding incomplete** — views don't fully respect `GuidedFlowState.Phase` during rendering

### Working
- State machine transitions (Health → Proposals → Fleet → Attached)
- Phase idempotency
- Tab switching between Health/Proposals

## Code Analysis

After inspecting the codebase, significant implementation already exists from commit 98a1bf5. The issue was reopened because gaps remain:

### What's Already Implemented
- `suggest_detail.go`: Full detail pane with title, description, complexity indicator, pipeline steps (tree connectors), "Why Suggested" section, DAG visualization, multi-select execution plan, input, sequence, key hints
- `suggest_dag.go`: `RenderDAG()` for sequence and parallel proposal visualization
- `guided_flow.go`: State machine with all transitions, `ViewForPhase()`, `TabTarget()`
- `content.go`: `guidedViewAllowed()` restricts views per phase, `SuggestLaunchMsg` transitions to fleet, health check auto-transition
- `statusbar.go`: Phase-aware hints for guided mode (health, proposals, fleet, attached)

### Remaining Gaps
1. **`guidedPhase` on `SuggestDetailModel` is set but never used** — `View()` ignores the phase for contextual rendering (e.g., different hints per phase)
2. **`SuggestModifyMsg` is a stub** — just redirects to `SuggestLaunchMsg` (content.go:1298-1303)
3. **No fleet auto-refresh after guided launch** — transitioning from proposals to fleet relies on normal polling with potential delay
4. **No guided-mode progress summary in fleet** — when in guided fleet phase, no indicator of which pipelines were launched from the suggest flow
5. **No fleet-to-suggest return context** — when Tab-toggling back to proposals from fleet, no visual indicator of already-launched proposals

## Acceptance Criteria

1. `SuggestDetailModel.View()` renders phase-contextual content using `guidedPhase`
2. `SuggestModifyMsg` opens an input editor for modifying proposal input before launch
3. Fleet view auto-refreshes when transitioning from guided proposals
4. Launched proposals are visually marked in the suggest list after returning via Tab
5. All new behavior covered by tests
