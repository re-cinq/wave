# refactor(tui): unify selection highlighting and focus styles across panes

**Issue**: [#366](https://github.com/re-cinq/wave/issues/366)
**Labels**: enhancement, ux, frontend
**Author**: nextlevelshit
**Complexity**: medium

## Problem

Selection and highlighting styles are inconsistent across the TUI. The current cyan color and caret indicator are not effective visual cues for the selected/hovered item.

## Current Behavior

- Selected/hovered items use cyan foreground color (`lipgloss.Color("6")`) and caret indicators (`>`, `›`, `▶`, `▸`)
- Highlighting style is not unified across left and right panes
- When focus moves to the right pane, the left pane selection retains full highlight intensity (only `Faint(true)` is applied to the entire pane)

## Expected Behavior

- **Active selection**: Light/white background with dark text for the currently hovered/selected item
- **Inactive pane selection**: Dimmed background matching the border color between panes (indicates item is still selected but pane is not focused)
- **Remove caret indicator**: The background highlight alone should be sufficient to indicate selection

## Acceptance Criteria

- [ ] Selected items in the active pane use light/white background with dark foreground text
- [ ] When focus switches to the right pane, the left pane's selected item dims to match the border color
- [ ] When focus switches back to the left pane, full highlight intensity is restored
- [ ] Caret (`>`) selection indicator is removed
- [ ] Cyan color is no longer used for selection highlighting
- [ ] Consistent highlight style applied across all list/selectable components

## Affected Components

- `internal/tui/theme.go` — central style definitions
- `internal/tui/pipeline_list.go` — pipeline list selection rendering
- `internal/tui/issue_list.go` — issue list selection rendering
- `internal/tui/compose_list.go` — compose sequence list selection rendering
- `internal/tui/persona_list.go` — persona list selection rendering
- `internal/tui/skill_list.go` — skill list selection rendering
- `internal/tui/health_list.go` — health check list selection rendering
- `internal/tui/contract_list.go` — contract list selection rendering
- `internal/tui/suggest_list.go` — suggestion list selection rendering
- `internal/tui/content.go` — pane focus management and rendering
