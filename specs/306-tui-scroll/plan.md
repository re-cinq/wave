# Implementation Plan: TUI Scroll Fix (#306)

## 1. Objective

Fix the TUI configure view so that both the pipeline/compose list (left pane) and the detail/form panel (right pane) are scrollable on small terminals, ensuring all content is reachable regardless of terminal height.

## 2. Approach

There are three distinct scroll problems to fix:

### A. ComposeListModel — Add scroll offset tracking (like PipelineListModel)

The `PipelineListModel` already has a working `scrollOffset` + `adjustScrollOffset()` pattern. The `ComposeListModel` has no equivalent — it renders all lines and clips. We'll add the same scroll pattern.

### B. PipelineDetailModel (stateConfiguring) — Wrap launch form in viewport

When in `stateConfiguring`, the detail pane renders a `huh.Form` directly. On small terminals, fields below the viewport are clipped and unreachable. We'll wrap the form output in a `bubbles/viewport` so it becomes scrollable.

### C. ComposeListModel.View() — Apply scroll window to rendered lines

After rendering all lines, apply a scroll window (like PipelineListModel does) to show only `m.height` lines starting from `m.scrollOffset`.

## 3. File Mapping

| File | Action | What Changes |
|------|--------|-------------|
| `internal/tui/compose_list.go` | modify | Add `scrollOffset int` field, `adjustScrollOffset()` method, apply scroll window in `View()` |
| `internal/tui/pipeline_detail.go` | modify | Wrap `stateConfiguring` form view in viewport for scroll support |
| `internal/tui/compose_list_test.go` | modify | Add scroll tests for ComposeListModel |
| `internal/tui/pipeline_detail_test.go` | modify | Add tests for form scrollability in stateConfiguring |

## 4. Architecture Decisions

- **Reuse the PipelineListModel scroll pattern** for ComposeListModel rather than introducing a viewport dependency — the list renders discrete items, so offset-based scrolling is simpler and consistent.
- **For the form/detail pane**, use the existing `viewport.Model` that `PipelineDetailModel` already has — the form's rendered string is set as viewport content, allowing native scroll via arrow keys when focused.
- **No mouse wheel in this iteration** — the stretch goal can be a follow-up. Keyboard scrolling is the primary fix.

## 5. Risks

| Risk | Mitigation |
|------|-----------|
| huh.Form may consume arrow keys before viewport can scroll | The form already consumes Tab/Shift+Tab for field navigation; we rely on huh's built-in scrolling within multi-select fields. The viewport wrapping handles the *outer* scroll when the entire form exceeds the pane height. |
| ComposeListModel scroll may break picker overlay positioning | The picker overlay is rendered as part of the line list and will scroll with it — no special handling needed since the picker replaces the list content when active. |
| Form height calculation may interact with viewport height | We'll set viewport height to `m.height - headerLines` and let the form render at its natural height inside the viewport content. |

## 6. Testing Strategy

- **Unit tests**: Add tests in `compose_list_test.go` verifying that when items exceed the view height, scrolling down moves `scrollOffset` and the first item scrolls out of view.
- **Unit tests**: Add tests in `pipeline_detail_test.go` verifying that in `stateConfiguring` with a small viewport, the form content is placed in the viewport and is scrollable.
- **Existing tests**: Ensure all existing tests in `internal/tui/` continue to pass — the PipelineListModel scroll tests already validate that pattern.
- **Manual validation**: Test with a terminal sized to ~30 rows to confirm both panels scroll correctly.
