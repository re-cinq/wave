# Implementation Plan — #443 TUI Guided Flow Completion

## Objective

Complete the remaining gaps in the TUI guided flow: make `SuggestDetailModel` phase-aware, implement the `SuggestModifyMsg` input editor, add fleet auto-refresh on guided transitions, and track launched proposals visually.

## Approach

Focus on the four concrete gaps identified in the code analysis. Each gap is a localized change within `internal/tui/`. No new packages or architectural changes needed — this is wiring existing state into rendering logic.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/tui/suggest_detail.go` | modify | Add phase-contextual rendering in `View()` using `guidedPhase` |
| `internal/tui/suggest_list.go` | modify | Track launched proposals, render "launched" badge |
| `internal/tui/suggest_messages.go` | modify | Add `SuggestLaunchedMsg` for tracking launched proposals |
| `internal/tui/content.go` | modify | Wire `SuggestModifyMsg` to input form, emit `SuggestLaunchedMsg`, trigger fleet refresh on transition |
| `internal/tui/suggest_detail_test.go` | modify | Add tests for phase-contextual rendering |
| `internal/tui/suggest_list_test.go` | modify | Add tests for launched badge rendering |
| `internal/tui/content_test.go` | modify | Add tests for modify flow, fleet refresh, launched tracking |

## Architecture Decisions

1. **Phase-contextual hints in detail pane**: Use the existing `guidedPhase` field to render different footer hints. In `GuidedPhaseProposals` show launch/select hints; in `GuidedPhaseFleet` show "viewing fleet" context. This is a simple switch in `View()`.

2. **Launched tracking via message**: Add `SuggestLaunchedMsg{Name string}` message. When `SuggestLaunchMsg` is handled in `content.go`, also emit `SuggestLaunchedMsg`. The `SuggestListModel` tracks launched names in a `map[string]bool` and renders a `[✓]` badge.

3. **Input modification via existing form**: `SuggestModifyMsg` should transition to the pipeline config form (`ConfigureFormMsg`) pre-populated with the proposal's pipeline name and input. The existing form infrastructure handles the rest.

4. **Fleet auto-refresh**: After `SuggestLaunchMsg` transitions to fleet, emit a `PipelineRefreshMsg` (or call the list's refresh cmd) to trigger an immediate data fetch rather than waiting for the next poll interval.

## Risks

1. **Form pre-population**: The `ConfigureFormMsg` may not support pre-filling the input field. Mitigation: check the form model's API and add pre-fill if missing.
2. **Poll race**: Immediate refresh after launch may not find the new run if SQLite write hasn't committed. Mitigation: add a small tick delay (500ms) before the refresh.

## Testing Strategy

- Unit tests for `SuggestDetailModel.View()` with each `GuidedFlowPhase` value
- Unit tests for `SuggestListModel` launched badge rendering
- Integration-style tests in `content_test.go` for the modify flow and fleet transition
- Table-driven tests following existing patterns in `suggest_detail_test.go`
