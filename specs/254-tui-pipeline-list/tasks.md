# Tasks: TUI Pipeline List Left Pane

**Feature**: #254 — TUI Pipeline List Left Pane
**Branch**: `254-tui-pipeline-list`
**Generated**: 2026-03-05
**Spec**: `specs/254-tui-pipeline-list/spec.md`
**Plan**: `specs/254-tui-pipeline-list/plan.md`

---

## Phase 1: Setup & Data Layer

- [X] T001 [P1] [Setup] Define `RunningPipeline` and `FinishedPipeline` value types in `internal/tui/pipeline_provider.go`
  - Create TUI-specific projection types derived from `state.RunRecord`
  - `RunningPipeline`: `RunID`, `Name`, `BranchName`, `StartedAt time.Time`
  - `FinishedPipeline`: `RunID`, `Name`, `BranchName`, `Status string`, `StartedAt time.Time`, `CompletedAt time.Time`, `Duration time.Duration`
  - File: `internal/tui/pipeline_provider.go`

- [X] T002 [P1] [Setup] Define `PipelineDataProvider` interface in `internal/tui/pipeline_provider.go`
  - Follow the `MetadataProvider` pattern from `header_metadata.go:79`
  - Methods: `FetchRunningPipelines() ([]RunningPipeline, error)`, `FetchFinishedPipelines(limit int) ([]FinishedPipeline, error)`, `FetchAvailablePipelines() ([]PipelineInfo, error)`
  - File: `internal/tui/pipeline_provider.go`

- [X] T003 [P1] [Setup] Implement `DefaultPipelineDataProvider` in `internal/tui/pipeline_provider.go`
  - Wraps `state.StateStore` (via `NewReadOnlyStateStore`) + `DiscoverPipelines`
  - `FetchRunningPipelines()` → calls `store.GetRunningRuns()`, maps `state.RunRecord` → `RunningPipeline`
  - `FetchFinishedPipelines(limit)` → calls `store.ListRuns(ListRunsOptions{Limit: limit*3})`, filters in Go for terminal statuses (completed/failed/cancelled), takes first `limit` results, maps to `FinishedPipeline`
  - `FetchAvailablePipelines()` → calls `DiscoverPipelines(pipelinesDir)`
  - File: `internal/tui/pipeline_provider.go`

- [X] T004 [P1] [Setup] [P] Define `PipelineDataMsg` and `PipelineRefreshTickMsg` message types in `internal/tui/pipeline_messages.go`
  - `PipelineDataMsg`: `Running []RunningPipeline`, `Finished []FinishedPipeline`, `Available []PipelineInfo`, `Err error`
  - `PipelineRefreshTickMsg`: empty struct (timer tick)
  - File: `internal/tui/pipeline_messages.go`

- [X] T005 [P1] [Setup] Write tests for `DefaultPipelineDataProvider` in `internal/tui/pipeline_provider_test.go`
  - Mock `state.StateStore` to return known `RunRecord` slices
  - Verify `RunningPipeline` mapping (name, runID, branchName, startedAt)
  - Verify `FinishedPipeline` mapping with duration computation
  - Verify terminal status filtering (only completed/failed/cancelled)
  - Verify `FetchAvailablePipelines` calls `DiscoverPipelines`
  - Verify error propagation from store
  - File: `internal/tui/pipeline_provider_test.go`

---

## Phase 2: Foundational — Navigation Types & Core List Model

- [X] T006 [P1] [US-1] Define `navigableItem`, `itemKind`, and `PipelineListModel` struct in `internal/tui/pipeline_list.go`
  - `itemKind` enum: `itemKindSectionHeader`, `itemKindRunning`, `itemKindFinished`, `itemKindAvailable`
  - `navigableItem`: `kind itemKind`, `sectionIndex int`, `dataIndex int`, `label string`
  - `PipelineListModel` struct: `width`, `height`, `provider`, `running []RunningPipeline`, `finished []FinishedPipeline`, `available []PipelineInfo`, `cursor int`, `navigable []navigableItem`, `filtering bool`, `filterInput textinput.Model`, `filterQuery string`, `collapsed [3]bool`, `focused bool`, `scrollOffset int`
  - File: `internal/tui/pipeline_list.go`

- [X] T007 [P1] [US-1] Implement `NewPipelineListModel(provider)` constructor and `Init()` in `internal/tui/pipeline_list.go`
  - Constructor: initializes `filterInput` from `textinput.New()`, sets `focused: true`
  - `Init()`: returns `tea.Batch(fetchPipelineData, refreshTick())` where `refreshTick` uses 5-second `tea.Tick` returning `PipelineRefreshTickMsg`
  - `fetchPipelineData`: async command calling all three provider methods, returning `PipelineDataMsg`
  - File: `internal/tui/pipeline_list.go`

- [X] T008 [P1] [US-1] Implement `buildNavigableItems()` in `internal/tui/pipeline_list.go`
  - Builds flat slice from sections: `[Running header] → [running items...] → [Finished header] → [finished items...] → [Available header] → [available items...]`
  - Respects `collapsed` state: collapsed sections include header only, skip items
  - Respects `filterQuery`: when filtering, include only items whose name contains the query (case-insensitive substring); always include section headers even when filtering (but hide headers with zero matching items)
  - Section header labels: `"Running (N)"`, `"Finished (N)"`, `"Available (N)"` where N is the count of items (after filtering)
  - File: `internal/tui/pipeline_list.go`

- [X] T009 [P1] [US-1] Implement `View()` for section rendering in `internal/tui/pipeline_list.go`
  - Render each `navigableItem` with appropriate styling per `itemKind`
  - Section headers: bold text, collapsed indicator `▸`/`▾` prefix
  - Running items: `"  name   2m30s"` with elapsed time from `time.Since(StartedAt)`
  - Finished items: `"  name   ✓ completed  1m15s"` or `"  name   ✗ failed  45s"` or `"  name   ✗ cancelled  30s"`
  - Available items: `"  name"`
  - Truncate pipeline names with `…` when exceeding pane width minus indicator and metadata
  - Empty state: `"No pipelines found"` centered when all sections empty
  - Handle `width <= 0 || height <= 0` gracefully
  - File: `internal/tui/pipeline_list.go`

- [X] T010 [P1] [US-1] Implement `Update()` for `PipelineDataMsg` and `PipelineRefreshTickMsg` in `internal/tui/pipeline_list.go`
  - `PipelineDataMsg`: update `running`, `finished`, `available` fields; call `buildNavigableItems()`; clamp cursor to new bounds; emit `RunningCountMsg{Count: len(running)}` as a command; if cursor is on a pipeline item, re-emit `PipelineSelectedMsg`
  - `PipelineRefreshTickMsg`: return `tea.Batch(fetchPipelineData, refreshTick())`
  - File: `internal/tui/pipeline_list.go`

- [X] T011 [P1] [US-1] Implement `SetSize(w, h)` for resize handling in `internal/tui/pipeline_list.go`
  - Update `width` and `height` fields
  - File: `internal/tui/pipeline_list.go`

- [X] T012 [P1] [US-1] Write tests for section rendering in `internal/tui/pipeline_list_test.go`
  - Create a `mockPipelineDataProvider` returning known data
  - Test: all three sections render with correct counts in headers
  - Test: running items show elapsed time format
  - Test: finished items show status icon and duration
  - Test: available items show name only
  - Test: empty sections show header with count 0
  - Test: all sections empty shows "No pipelines found"
  - Test: long pipeline names are truncated with `…`
  - File: `internal/tui/pipeline_list_test.go`

---

## Phase 3: User Story 2 — Keyboard Navigation (P1)

- [X] T013 [P1] [US-2] Implement `Update()` key handling for ↑/↓ navigation in `internal/tui/pipeline_list.go`
  - `tea.KeyUp`: decrement `cursor`, clamp at 0 (no wrap)
  - `tea.KeyDown`: increment `cursor`, clamp at `len(navigable)-1` (no wrap)
  - After cursor change: if new item is a pipeline item (not header), emit `PipelineSelectedMsg` with appropriate `RunID`/`BranchName` (empty for Available items)
  - If new item is a section header, do NOT emit `PipelineSelectedMsg`
  - File: `internal/tui/pipeline_list.go`

- [X] T014 [P1] [US-2] Implement selection indicator rendering in `View()` in `internal/tui/pipeline_list.go`
  - Selected pipeline item: prepend `▶ ` and use distinct foreground color (cyan)
  - Selected section header: render with bold + inverse style
  - Unselected items: normal rendering (no indicator prefix)
  - File: `internal/tui/pipeline_list.go`

- [X] T015 [P1] [US-2] Write tests for keyboard navigation in `internal/tui/pipeline_list_test.go`
  - Test: ↓ moves cursor from 0 to 1
  - Test: ↑ at cursor 0 stays at 0 (no wrap)
  - Test: ↓ at last item stays at last (no wrap)
  - Test: cross-section traversal (last running item → finished header → first finished item)
  - Test: `PipelineSelectedMsg` emitted when cursor moves to pipeline item
  - Test: `PipelineSelectedMsg` NOT emitted when cursor is on section header
  - Test: `PipelineSelectedMsg` for Running item includes `RunID` and `BranchName`
  - Test: `PipelineSelectedMsg` for Available item has empty `RunID` and `BranchName`
  - File: `internal/tui/pipeline_list_test.go`

---

## Phase 4: User Story 3 — Search/Filter (P2)

- [X] T016 [P2] [US-3] Implement filter activation and dismissal in `Update()` in `internal/tui/pipeline_list.go`
  - `/` key: set `filtering = true`, focus `filterInput`, clear previous query
  - `Escape` key (when filtering): set `filtering = false`, clear `filterQuery`, rebuild navigable items, reset cursor to 0
  - When filtering: forward key events to `filterInput.Update()`; on each change, update `filterQuery` from `filterInput.Value()`, rebuild navigable items
  - File: `internal/tui/pipeline_list.go`

- [X] T017 [P2] [US-3] Implement filter input rendering in `View()` in `internal/tui/pipeline_list.go`
  - When `filtering` is true: render filter input at the top of the pane with a search icon (🔍 or `/`) and the `filterInput.View()` output
  - Reduce available height for items by 1 line when filter input is visible
  - When filter matches zero items: display `"No matching pipelines"` message
  - File: `internal/tui/pipeline_list.go`

- [X] T018 [P2] [US-3] Write tests for search/filter in `internal/tui/pipeline_list_test.go`
  - Test: `/` key activates filter mode
  - Test: typing "spec" filters to only pipelines containing "spec" (case-insensitive)
  - Test: filter applies across all sections simultaneously
  - Test: Escape dismisses filter and restores full list
  - Test: filter with zero matches shows "No matching pipelines"
  - Test: ↑/↓ navigation works within filtered results
  - File: `internal/tui/pipeline_list_test.go`

---

## Phase 5: User Story 4 — Viewport Scrolling (P2)

- [X] T019 [P2] [US-4] Implement viewport scrolling in `View()` in `internal/tui/pipeline_list.go`
  - Calculate visible window: `scrollOffset` adjusted so cursor is always within `[scrollOffset, scrollOffset + visibleHeight)`
  - When cursor moves below visible area: scroll down
  - When cursor moves above visible area: scroll up
  - Render only the items within the visible window
  - File: `internal/tui/pipeline_list.go`

- [X] T020 [P2] [US-4] Write tests for viewport scrolling in `internal/tui/pipeline_list_test.go`
  - Test: with height=5 and 20 items, navigating to item 6 scrolls viewport
  - Test: scrolling back up keeps selected item visible
  - Test: cursor at top of viewport doesn't scroll unnecessarily
  - File: `internal/tui/pipeline_list_test.go`

---

## Phase 6: User Story 5 — Section Collapse/Expand (P3)

- [X] T021 [P3] [US-5] Implement collapse/expand toggle in `Update()` in `internal/tui/pipeline_list.go`
  - Enter key on a section header: toggle `collapsed[sectionIndex]`
  - After toggle: rebuild navigable items; if cursor was on a now-hidden item, move cursor to the section header
  - File: `internal/tui/pipeline_list.go`

- [X] T022 [P3] [US-5] Write tests for collapse/expand in `internal/tui/pipeline_list_test.go`
  - Test: Enter on Running header collapses Running section, items hidden
  - Test: Enter again on collapsed header expands it, items reappear
  - Test: cursor skips hidden items during ↑/↓ navigation
  - Test: collapsed section header shows `▸` indicator, expanded shows `▾`
  - File: `internal/tui/pipeline_list_test.go`

---

## Phase 7: ContentModel Refactoring & App Integration

- [X] T023 [P1] [Integration] Refactor `ContentModel` to compose `PipelineListModel` in `internal/tui/content.go`
  - Add `list PipelineListModel` field to `ContentModel`
  - Update `NewContentModel(provider PipelineDataProvider)` to accept and forward provider
  - Add `Init()` method delegating to `list.Init()`
  - Add `Update(msg) (ContentModel, tea.Cmd)` method forwarding messages to `list.Update()`
  - Implement `View()` with left/right split: left pane width = `min(max(width*30/100, 25), 50)`, right pane = placeholder "Select a pipeline to view details"
  - Use `lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)`
  - Propagate `SetSize` to list model with computed left pane width and full content height
  - File: `internal/tui/content.go`

- [X] T024 [P1] [Integration] Update `AppModel` to accept and forward `PipelineDataProvider` in `internal/tui/app.go`
  - Update `NewAppModel(metaProvider MetadataProvider, pipelineProvider PipelineDataProvider)` signature
  - Pass `pipelineProvider` to `NewContentModel(pipelineProvider)`
  - Update `Init()` to `tea.Batch(m.header.Init(), m.content.Init())`
  - Update `Update()`: forward messages to `m.content.Update()` and collect commands from both header and content
  - Route ↑/↓, `/`, Escape, Enter keys to content when not in quit flow
  - Update `RunTUI()` to create `DefaultPipelineDataProvider` and pass both providers
  - File: `internal/tui/app.go`

- [X] T025 [P1] [Integration] Update `content_test.go` for refactored `ContentModel` in `internal/tui/content_test.go`
  - Replace "Pipelines view coming soon" assertions with new left/right pane assertions
  - Test: left pane width is 30% (min 25, max 50) of total width
  - Test: placeholder right pane renders
  - Test: `SetSize` propagates to list model
  - Test: `Init()` returns commands from list model
  - File: `internal/tui/content_test.go`

- [X] T026 [P1] [Integration] Update `app_test.go` for new `NewAppModel` signature in `internal/tui/app_test.go`
  - Add `mockPipelineDataProvider` to test helpers (or reuse from `pipeline_list_test.go`)
  - Update all `NewAppModel(&mockProvider{})` calls to `NewAppModel(&mockProvider{}, &mockPipelineDataProvider{})`
  - Test: `Init()` returns batch including content init commands
  - Test: `WindowSizeMsg` propagates to content with correct dimensions
  - Test: key events (↑/↓) forwarded to content model
  - Test: `PipelineSelectedMsg` flows from content to header (existing test updated)
  - Remove or update assertion for "Pipelines view coming soon" in `TestAppModel_View_AfterReady`
  - File: `internal/tui/app_test.go`

---

## Phase 8: Polish & Cross-Cutting

- [X] T027 [P2] [Polish] [P] Update status bar key hints in `internal/tui/statusbar.go`
  - Add `↑↓: navigate  /: filter` hints to the existing hint string
  - File: `internal/tui/statusbar.go`

- [X] T028 [P1] [Polish] Verify all existing tests pass with `go test ./internal/tui/...`
  - Run full TUI test suite
  - Fix any regressions from ContentModel/AppModel refactoring
  - Ensure no test uses `t.Skip()` without a linked issue

- [X] T029 [P1] [Polish] Verify full project tests pass with `go test ./...`
  - Run complete project test suite
  - Ensure no regressions outside TUI package

- [X] T030 [P2] [Polish] [P] Verify `NO_COLOR` compliance in pipeline list rendering
  - Ensure lipgloss respects `NO_COLOR` environment variable
  - No hardcoded ANSI escape sequences — all styling via lipgloss

---

## Dependency Graph

```
T001 ─┬─► T002 ─► T003 ─► T005
      │
T004 ─┘
      │
      ▼
T006 ─► T007 ─► T008 ─► T009 ─► T010 ─► T011 ─► T012
                                    │
                                    ▼
                              T013 ─► T014 ─► T015
                                        │
                                        ▼
                                  T016 ─► T017 ─► T018
                                            │
                                            ▼
                                      T019 ─► T020
                                        │
                                        ▼
                                  T021 ─► T022
                                        │
                                        ▼
                              T023 ─► T024 ─► T025
                                        │       │
                                        ▼       ▼
                                      T026    T027 [P]
                                        │
                                        ▼
                                  T028 ─► T029
                                    │
                                    ▼
                                  T030 [P]
```

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| Phase 1: Setup & Data Layer | T001–T005 | Provider interface, value types, messages, tests |
| Phase 2: Core List Model | T006–T012 | Navigation types, model struct, View/Update, section rendering |
| Phase 3: Navigation (P1) | T013–T015 | ↑/↓ keyboard nav, selection indicators, PipelineSelectedMsg |
| Phase 4: Filter (P2) | T016–T018 | `/` search activation, real-time filtering, Escape dismissal |
| Phase 5: Scrolling (P2) | T019–T020 | Viewport follows cursor, visible window calculation |
| Phase 6: Collapse (P3) | T021–T022 | Section toggle on Enter, cursor skip over hidden items |
| Phase 7: Integration | T023–T026 | ContentModel refactor, AppModel wiring, test updates |
| Phase 8: Polish | T027–T030 | Status bar hints, test verification, NO_COLOR compliance |

**Total tasks**: 30
**Parallelizable**: T004, T027, T030 (marked with [P])
