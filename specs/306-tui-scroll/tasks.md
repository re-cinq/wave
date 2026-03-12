# Tasks

## Phase 1: ComposeListModel Scroll Infrastructure

- [X] Task 1.1: Add `scrollOffset int` field to `ComposeListModel` struct in `compose_list.go`
- [X] Task 1.2: Add `adjustScrollOffset(visibleHeight int)` method to `ComposeListModel`, mirroring `PipelineListModel.adjustScrollOffset()`
- [X] Task 1.3: Update `ComposeListModel.View()` to compute `visibleHeight`, call `adjustScrollOffset()`, and render only the visible window of lines (from `scrollOffset` to `scrollOffset + visibleHeight`)
- [X] Task 1.4: Ensure cursor movement in `handleKeyMsg()` still works correctly with scroll — the `adjustScrollOffset` call in `View()` handles this automatically

## Phase 2: PipelineDetailModel Configure Form Scroll

- [X] Task 2.1: In `PipelineDetailModel.View()` for `stateConfiguring`, wrap the form output in the existing `m.viewport` — set the form's rendered string as viewport content and return `m.viewport.View()` instead of raw form output [P]
- [X] Task 2.2: In `PipelineDetailModel.Update()` for `stateConfiguring`, after forwarding messages to `m.launchForm`, also forward key messages to `m.viewport` so scroll keys work when the form doesn't consume them [P]
- [X] Task 2.3: In `ConfigureFormMsg` handler, set viewport content after creating the form and call `m.viewport.GotoTop()` to reset scroll position

## Phase 3: Testing

- [X] Task 3.1: Add scroll tests to `compose_list_test.go` — test that with height=5 and 10+ items, cursor navigation scrolls the view and first items disappear from rendered output [P]
- [X] Task 3.2: Add test to `pipeline_detail_test.go` — verify that in stateConfiguring with a small viewport, the form content is placed in the viewport model [P]
- [X] Task 3.3: Run `go test ./internal/tui/...` to verify all existing tests pass

## Phase 4: Polish

- [X] Task 4.1: Verify edge cases: empty compose list, single-item list, height=0, picker overlay with scroll
- [X] Task 4.2: Final validation with `go test -race ./...`
