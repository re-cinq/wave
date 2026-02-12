# Implementation Plan: TUI Pause Audit (#56)

## Objective

Remove the non-functional `p`/pause keybinding and its associated UI elements from Wave's TUI, and audit the rest of the TUI codebase for other non-functional features.

## Approach

**Strategy: Remove rather than implement.** Implementing true pipeline pause/resume would require significant changes across the executor, adapter, and state layers (adding a pause channel, handling mid-subprocess interruption, persisting pause state). This is disproportionate to the value it provides during prototype phase. The simpler and more honest approach is to remove the misleading keybinding and UI text, keeping only `q`/`ctrl+c` for quitting.

If pause/resume is desired in the future, it should be designed as a separate feature with proper pipeline-level signaling.

## File Mapping

| File | Action | Changes |
|------|--------|---------|
| `internal/display/bubbletea_model.go` | modify | Remove `paused` field, remove `p` case from `Update()`, remove pause-related status line, simplify `View()` |
| `internal/display/dashboard.go` | modify | Remove `p=pause` from the help text in `renderHeader()` |
| `internal/display/bubbletea_progress.go` | no change | No direct pause references (the `paused` field is on `ProgressModel`, not here) |

## Architecture Decisions

1. **Remove, don't hide**: Rather than just removing the keybinding while keeping the `paused` field, remove all traces of the pause functionality. Dead code invites confusion.

2. **Keep `q`/`ctrl+c` only**: The status line should show only `Press: q=quit` (or `ctrl+c`). This accurately reflects available controls.

3. **No `StatePaused` in display types**: The `ProgressState` enum in `types.go` does not have a paused state, so no changes needed there. The `"paused"` state in the state store (`state` package) and `resume.go` command are for pipeline-level pause/resume (a different concept from TUI pause) and should remain untouched.

4. **Broader audit as separate issues**: Per the acceptance criteria, any other non-functional features discovered during the audit should be filed as separate GitHub issues rather than fixed inline.

## Risks

| Risk | Mitigation |
|------|------------|
| Breaking existing tests that reference pause | Search for test references to `paused`, `pause` in display tests; update or remove as needed |
| The `dashboard.go` help text is used in tests | Check `dashboard_test.go` for string assertions on "p=pause" |
| Users who rely on `p` to "freeze" the display | This was never documented as a feature; the tick pause only stopped animations, not output. No user impact expected. |

## Testing Strategy

1. **Unit tests**: Verify `ProgressModel.Update()` no longer responds to `p` keypress
2. **Unit tests**: Verify `View()` shows only `q=quit` in the status line
3. **Existing tests**: Run `go test ./internal/display/...` to ensure no regressions
4. **Manual verification**: Run a pipeline and confirm the TUI no longer shows pause controls
5. **Audit output**: Document findings from the broader TUI audit (other non-functional features) as new issues
