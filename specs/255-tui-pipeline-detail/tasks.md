# Tasks: TUI Pipeline Detail Right Pane

**Feature**: #255 | **Branch**: `255-tui-pipeline-detail` | **Generated**: 2026-03-06

---

## Phase 1: Setup & Foundation

- [X] T001 [P1] Setup — Add `charmbracelet/bubbles/viewport` as a direct dependency in `go.mod` (already indirect via `huh`; promote to direct) by running `go get github.com/charmbracelet/bubbles` and verifying `go mod tidy` succeeds. **File**: `go.mod`

---

## Phase 2: Message Types & Data Model (Foundational — blocks all downstream phases)

- [X] T002 [P] [P1] [US-1,US-2,US-3] Extend `PipelineSelectedMsg` in `internal/tui/header_messages.go` — Add `Name string` field (pipeline name for all item types) and `Kind itemKind` field (item type: running/finished/available/sectionHeader) to the existing struct. Update the comment to document the new fields. **File**: `internal/tui/header_messages.go`

- [X] T003 [P] [P1] [US-3,US-5] Add `FocusChangedMsg` and `FocusPane` type to `internal/tui/header_messages.go` — Define `FocusPane` as `int` enum with `FocusPaneLeft` (default, iota) and `FocusPaneRight`. Define `FocusChangedMsg` struct with `Pane FocusPane` field. **File**: `internal/tui/header_messages.go`

- [X] T004 [P] [P1] [US-1,US-2] Add `DetailDataMsg` message type to `internal/tui/header_messages.go` — Struct with `AvailableDetail *AvailableDetail`, `FinishedDetail *FinishedDetail`, and `Err error` fields. This carries async-fetched detail data back to the model. **File**: `internal/tui/header_messages.go`

- [X] T005 [P] [P1] [US-1] Define `AvailableDetail` and `StepSummary` data projection types in a new file `internal/tui/pipeline_detail_provider.go` — `AvailableDetail` contains: `Name`, `Description`, `Category` (strings), `StepCount` (int), `Steps []StepSummary`, `InputSource`, `InputExample` (strings), `Artifacts []string`, `Skills []string`, `Tools []string`. `StepSummary` contains: `ID`, `Persona` (strings). **File**: `internal/tui/pipeline_detail_provider.go`

- [X] T006 [P] [P1] [US-2] Define `FinishedDetail`, `StepResult`, and `ArtifactInfo` data projection types in `internal/tui/pipeline_detail_provider.go` — `FinishedDetail`: `RunID`, `Name`, `Status` (strings), `Duration` (time.Duration), `BranchName` (string), `StartedAt`, `CompletedAt` (time.Time), `ErrorMessage`, `FailedStep` (strings), `Steps []StepResult`, `Artifacts []ArtifactInfo`. `StepResult`: `ID`, `Status`, `Persona` (strings), `Duration` (time.Duration). `ArtifactInfo`: `Name`, `Path`, `Type` (strings). **File**: `internal/tui/pipeline_detail_provider.go`

- [X] T007 [P] [P1] [US-1,US-2] Define `DetailDataProvider` interface in `internal/tui/pipeline_detail_provider.go` — Two methods: `FetchAvailableDetail(name string) (*AvailableDetail, error)` and `FetchFinishedDetail(runID string) (*FinishedDetail, error)`. **File**: `internal/tui/pipeline_detail_provider.go`

---

## Phase 3: Provider Implementation (US-1 — Available Pipeline Details)

- [X] T008 [P1] [US-1] Implement `DefaultDetailDataProvider` struct in `internal/tui/pipeline_detail_provider.go` — Fields: `store state.StateStore`, `pipelinesDir string`. Constructor: `NewDefaultDetailDataProvider(store state.StateStore, pipelinesDir string) *DefaultDetailDataProvider`. **File**: `internal/tui/pipeline_detail_provider.go`

- [X] T009 [P1] [US-1] Implement `FetchAvailableDetail(name string)` on `DefaultDetailDataProvider` — Scan `pipelinesDir` for YAML files, parse each with `yaml.Unmarshal` into `pipeline.Pipeline`, match by `Metadata.Name == name`. Map steps to `[]StepSummary` (step ID + persona), extract `Input.Source`, `Input.Example`, collect output artifact names across all steps (`step.OutputArtifacts[].Name`), and extract `Requires.SkillNames()` and `Requires.Tools`. Return `*AvailableDetail`. Return error if pipeline not found. **File**: `internal/tui/pipeline_detail_provider.go`

- [X] T010 [P1] [US-1] Write tests for `FetchAvailableDetail` in `internal/tui/pipeline_detail_provider_test.go` — Create temp directory with test YAML pipeline file. Verify: correct name/description/category extraction, step list with IDs and personas, input source and example, output artifact names, skill and tool dependencies, error when pipeline name not found. **File**: `internal/tui/pipeline_detail_provider_test.go`

---

## Phase 4: Provider Implementation (US-2 — Finished Pipeline Summary)

- [X] T011 [P1] [US-2] Implement `FetchFinishedDetail(runID string)` on `DefaultDetailDataProvider` — Query `store.GetRun(runID)` for run record (status, timestamps, error, branch). Query `store.GetPerformanceMetrics(runID, "")` for step results (map each `PerformanceMetricRecord` to `StepResult` with ID, status derived from `Success` bool, duration from `DurationMs`, persona). Query `store.GetArtifacts(runID, "")` for artifacts (map to `ArtifactInfo`). Derive `FailedStep` from first metric where `Success == false`. Compute `Duration` from `CompletedAt - StartedAt` (or `CancelledAt - StartedAt`). **File**: `internal/tui/pipeline_detail_provider.go`

- [X] T012 [P1] [US-2] Write tests for `FetchFinishedDetail` in `internal/tui/pipeline_detail_provider_test.go` — Use mock `StateStore` implementation. Test cases: completed run with all fields, failed run with error message and failed step identification, cancelled run, run with zero artifacts, run with multiple step results in order. Verify duration computation, status mapping, artifact listing. **File**: `internal/tui/pipeline_detail_provider_test.go`

---

## Phase 5: Pipeline List Emitter Update (US-1,US-2 — required before detail model can receive data)

- [X] T013 [P1] [US-1,US-2] Update `emitSelectionMsg()` in `internal/tui/pipeline_list.go` to populate `Name` and `Kind` fields — Running items: `Name: r.Name, Kind: itemKindRunning`. Finished items: `Name: f.Name, Kind: itemKindFinished`. Available items: `Name: a.Name, Kind: itemKindAvailable`. For section headers (which currently return nil), no change needed. **File**: `internal/tui/pipeline_list.go:289-319`

- [X] T014 [P] [P1] [US-1,US-2] Update `PipelineSelectedMsg` test assertions in `internal/tui/pipeline_list_test.go` — All existing tests that assert on `PipelineSelectedMsg` must include the new `Name` and `Kind` fields. **File**: `internal/tui/pipeline_list_test.go`

- [X] T015 [P] [P1] [US-1,US-2] Update `PipelineSelectedMsg` test assertions in `internal/tui/header_test.go` — Any test that creates `PipelineSelectedMsg` literals must compile with the new fields (zero values are fine since header only reads `RunID`, `BranchName`, `BranchDeleted`). **File**: `internal/tui/header_test.go`

---

## Phase 6: Pipeline Detail Model (US-1,US-2,US-4,US-5 — the core right pane component)

- [X] T016 [P1] [US-5] Create `internal/tui/pipeline_detail.go` — Define `PipelineDetailModel` struct with fields: `width, height int`, `focused bool`, `viewport viewport.Model`, `selectedName string`, `selectedKind itemKind`, `selectedRunID string`, `availableDetail *AvailableDetail`, `finishedDetail *FinishedDetail`, `branchDeleted bool`, `loading bool`, `errorMsg string`, `provider DetailDataProvider`. **File**: `internal/tui/pipeline_detail.go`

- [X] T017 [P1] [US-5] Implement `NewPipelineDetailModel(provider DetailDataProvider) PipelineDetailModel` constructor — Initialize viewport with zero dimensions, set `focused = false`. **File**: `internal/tui/pipeline_detail.go`

- [X] T018 [P1] [US-5] Implement `SetSize(w, h int)` and `SetFocused(bool)` methods on `PipelineDetailModel` — `SetSize`: update width/height, resize viewport (`viewport.Width = w`, `viewport.Height = h`), re-render content into viewport if data exists. `SetFocused`: update `focused` field, enable/disable viewport keybindings accordingly (viewport handles up/down when enabled). **File**: `internal/tui/pipeline_detail.go`

- [X] T019 [P1] [US-1,US-2,US-5] Implement `Init()` and `Update(msg tea.Msg)` on `PipelineDetailModel` — `Init()`: return nil (no initial commands). `Update`: handle `PipelineSelectedMsg` (store selection state, clear old data, set `loading=true`, return async fetch command for the appropriate provider method based on `Kind`; if `itemKindSectionHeader`, clear all data and show placeholder). Handle `DetailDataMsg` (store data, set `loading=false`, render content into viewport, reset viewport to top). Handle `tea.KeyMsg` when focused (forward to viewport for scroll). **File**: `internal/tui/pipeline_detail.go`

- [X] T020 [P1] [US-5] Implement `View()` on `PipelineDetailModel` — Render one of: (a) placeholder "Select a pipeline to view details" when `selectedName == ""`, (b) "Loading..." when `loading == true`, (c) error message when `errorMsg != ""`, (d) available detail view via `renderAvailableDetail()`, (e) finished detail view via `renderFinishedDetail()`, (f) running info via `renderRunningInfo()`. Use `lipgloss.Place` for centering placeholder. When focused with scrollable content, use `viewport.View()`. **File**: `internal/tui/pipeline_detail.go`

- [X] T021 [P1] [US-1] Implement `renderAvailableDetail(detail *AvailableDetail, width int) string` pure rendering function — Sections: title (name, bold), description (if non-empty), category (if non-empty), steps table (numbered list with step ID and persona), input (source + example), output artifacts (bullet list), dependencies (skills + tools). Use lipgloss styles consistent with existing theme. **File**: `internal/tui/pipeline_detail.go`

- [X] T022 [P1] [US-2] Implement `renderFinishedDetail(detail *FinishedDetail, width int, branchDeleted bool) string` pure rendering function — Sections: title (name, bold), status line with checkmark/cross indicator, duration, branch name (with "(deleted)" suffix if `branchDeleted`), timestamps (started/completed), error message (if failed, with failed step ID highlighted), step results table (step ID, status indicator, duration, persona), artifacts section (name + path, or "No artifacts produced" if empty), action hints line at bottom (`[Enter] Open chat  [b] Checkout branch  [d] View diff  [Esc] Back`, with checkout hint dimmed if branch deleted). **File**: `internal/tui/pipeline_detail.go`

- [X] T023 [P2] [US-2] Implement `renderRunningInfo(name string, width int) string` pure rendering function — Brief informational message: pipeline name, "Running" status indicator, and note that real-time progress monitoring is planned for issue #258. **File**: `internal/tui/pipeline_detail.go`

- [X] T024 [P1] [US-1,US-2,US-4,US-5] Write comprehensive tests for `PipelineDetailModel` in `internal/tui/pipeline_detail_test.go` — Create mock `DetailDataProvider`. Test cases: (a) placeholder rendering when no selection, (b) available detail rendering with all fields, (c) finished detail completed rendering, (d) finished detail failed rendering with error and failed step, (e) cancelled pipeline rendering, (f) branch deleted indicator, (g) zero artifacts shows "No artifacts produced", (h) focus state change, (i) scroll handling when focused (send up/down keys), (j) `PipelineSelectedMsg` triggers async fetch, (k) `DetailDataMsg` populates view, (l) selection change resets scroll to top, (m) running pipeline shows informational message, (n) section header selection shows placeholder. **File**: `internal/tui/pipeline_detail_test.go`

---

## Phase 7: Content Model Focus Management (US-3 — pane navigation)

- [X] T025 [P1] [US-3] Add `detail PipelineDetailModel` and `focus FocusPane` fields to `ContentModel` in `internal/tui/content.go` — Update struct to include both child models and focus tracking. **File**: `internal/tui/content.go:9-13`

- [X] T026 [P1] [US-3] Update `NewContentModel` signature to accept `DetailDataProvider` — `NewContentModel(provider PipelineDataProvider, detailProvider DetailDataProvider) ContentModel`. Initialize `PipelineDetailModel` with provider. Set `focus = FocusPaneLeft` as default. **File**: `internal/tui/content.go:16-20`

- [X] T027 [P1] [US-3] Update `SetSize` in `ContentModel` to propagate dimensions to detail model — Calculate `rightWidth = w - leftPaneWidth()`. Call `m.detail.SetSize(rightWidth, h)`. **File**: `internal/tui/content.go:28-34`

- [X] T028 [P1] [US-3] Update `Init()` in `ContentModel` — Return `tea.Batch(m.list.Init(), m.detail.Init())`. **File**: `internal/tui/content.go:23-25`

- [X] T029 [P1] [US-3] Implement focus-aware `Update(msg)` in `ContentModel` — Handle `tea.KeyMsg` Enter: if `focus == FocusPaneLeft` and cursor is on a non-header, non-running item, transition focus to right pane (`m.focus = FocusPaneRight`, call `m.list.SetFocused(false)`, `m.detail.SetFocused(true)`), return `FocusChangedMsg{FocusPaneRight}` as command. If cursor is on header or running item, forward Enter to list for existing collapse/no-op behavior. Handle `tea.KeyMsg` Esc: if `focus == FocusPaneRight`, transition focus to left pane, return `FocusChangedMsg{FocusPaneLeft}`. Route other `tea.KeyMsg` to focused child only. Forward `PipelineSelectedMsg` and `DetailDataMsg` to detail model. Forward `PipelineSelectedMsg`, `PipelineDataMsg`, `PipelineRefreshTickMsg` to list. **File**: `internal/tui/content.go:37-41`

- [X] T030 [P1] [US-3] Add helper method to `ContentModel` to check if cursor is on a focusable item — `cursorOnFocusableItem() bool`: check `m.list.navigable[m.list.cursor].kind` is `itemKindAvailable` or `itemKindFinished`. Guard against empty navigable list and out-of-bounds cursor. **File**: `internal/tui/content.go`

- [X] T031 [P1] [US-3] Update `View()` in `ContentModel` to render detail pane and focus indicators — Replace static placeholder with `m.detail.View()`. When `focus == FocusPaneRight`, apply dimmed style to left pane and highlighted border/style to right pane. When `focus == FocusPaneLeft`, render normally. **File**: `internal/tui/content.go:44-61`

- [X] T032 [P1] [US-3] Update `ContentModel` tests in `internal/tui/content_test.go` — Update all `NewContentModel` calls to pass a nil or mock `DetailDataProvider`. Add tests: (a) focus starts on left pane, (b) Enter on available item transitions focus right, (c) Enter on finished item transitions focus right, (d) Enter on section header does NOT transition (collapses section), (e) Enter on running item does NOT transition, (f) Esc from right pane returns focus left, (g) arrow keys in right pane scroll detail not list, (h) arrow keys in left pane navigate list not detail, (i) `FocusChangedMsg` emitted on transitions, (j) right pane width = total - leftPaneWidth. **File**: `internal/tui/content_test.go`

---

## Phase 8: Status Bar Dynamic Hints (US-3)

- [X] T033 [P2] [US-3] Add `focusPane FocusPane` field to `StatusBarModel` and handle `FocusChangedMsg` — Add `Update(msg tea.Msg) (StatusBarModel, tea.Cmd)` method to `StatusBarModel`. Handle `FocusChangedMsg`: update `m.focusPane = msg.Pane`. **File**: `internal/tui/statusbar.go`

- [X] T034 [P2] [US-3] Update `View()` in `StatusBarModel` for dynamic key hints — When `focusPane == FocusPaneLeft` (default): `"↑↓: navigate  Enter: view  /: filter  q: quit  ctrl+c: exit"`. When `focusPane == FocusPaneRight`: `"↑↓: scroll  Esc: back  q: quit  ctrl+c: exit"`. **File**: `internal/tui/statusbar.go:28-57`

- [X] T035 [P] [P2] [US-3] Write status bar dynamic hint tests in `internal/tui/statusbar_test.go` — Test: default hints (left pane), hints update on `FocusChangedMsg{FocusPaneRight}`, hints revert on `FocusChangedMsg{FocusPaneLeft}`. **File**: `internal/tui/statusbar_test.go`

---

## Phase 9: App Model Integration (wires everything together)

- [X] T036 [P1] Update `NewAppModel` signature in `internal/tui/app.go` — Accept `DetailDataProvider` as third parameter. Pass to `NewContentModel`. **File**: `internal/tui/app.go:30-36`

- [X] T037 [P1] Forward `FocusChangedMsg` to `StatusBarModel` in `AppModel.Update()` — Add a type switch case for `FocusChangedMsg` that calls `m.statusBar, _ = m.statusBar.Update(msg)`. Alternatively, forward all messages to statusbar generically (add statusbar update alongside header and content forwarding). **File**: `internal/tui/app.go:44-91`

- [X] T038 [P1] Update q-quit guard in `AppModel.Update()` — Currently `msg.String() == "q" && !m.content.list.filtering`. Also ensure q-quit works from the right pane (no text input in right pane, so q should still quit). If the right pane is focused, q still quits since there's no interactive input. No change needed if current logic suffices; verify by testing. **File**: `internal/tui/app.go:57-59`

- [X] T039 [P1] Update `RunTUI()` in `internal/tui/app.go` — Create `DefaultDetailDataProvider` alongside existing `DefaultPipelineDataProvider`. Pass both to `NewAppModel`. Requires determining the pipelines directory path (same as used by `DefaultPipelineDataProvider`) and the state store instance. **File**: `internal/tui/app.go:116-121`

- [X] T040 [P1] Update `AppModel` tests in `internal/tui/app_test.go` — Update `NewAppModel` calls to pass nil or mock `DetailDataProvider`. Add test: `FocusChangedMsg` forwarded to status bar. **File**: `internal/tui/app_test.go`

---

## Phase 10: Polish & Cross-Cutting Concerns

- [X] T041 [P] [P2] Edge case — Terminal resize re-renders detail pane — Verify `tea.WindowSizeMsg` propagates through `AppModel -> ContentModel -> PipelineDetailModel.SetSize()`. The detail model should re-render content at new width and update viewport dimensions. Write a test in `pipeline_detail_test.go` verifying resize behavior. **File**: `internal/tui/pipeline_detail_test.go`

- [X] T042 [P] [P2] Edge case — Narrow terminal (80 columns) renders detail without artifacts — With 80-column terminal, right pane gets ~55 columns. Verify all rendering functions handle narrow widths: truncate long step names, artifact paths with ellipsis. Write boundary rendering tests at width=30 (minimum useful). **File**: `internal/tui/pipeline_detail_test.go`

- [X] T043 [P] [P2] Edge case — Long error messages truncate/wrap gracefully — In `renderFinishedDetail`, ensure error messages that exceed available width wrap within the content area rather than breaking layout. Use lipgloss `Width()` style or manual wrapping. **File**: `internal/tui/pipeline_detail.go`

- [X] T044 [P] [P2] Edge case — Skipped steps display — In `renderFinishedDetail`, handle steps with status "skipped" (or "pending") showing a dash indicator and no duration, distinct from completed/failed steps. **File**: `internal/tui/pipeline_detail.go`

- [X] T045 [P] [P2] Edge case — State store error shows error message — When `DetailDataMsg.Err != nil`, the detail model sets `errorMsg` and renders "Failed to load pipeline details" (plus error) instead of crashing. Verify this path in `pipeline_detail_test.go`. **File**: `internal/tui/pipeline_detail_test.go`

- [X] T046 [P] [P2] NO_COLOR compliance — Verify detail pane rendering respects `NO_COLOR` env var. lipgloss automatically degrades when `NO_COLOR` is set (via `lipgloss.HasDarkBackground` / `lipgloss.NewRenderer`). Ensure all styles in `pipeline_detail.go` use lipgloss styles (no hardcoded ANSI codes). **File**: `internal/tui/pipeline_detail.go`

- [X] T047 [P1] Run full test suite — Execute `go test ./internal/tui/...` to verify all new and existing tests pass. Then run `go test ./...` to check for no regressions across the entire project. Fix any compilation errors or test failures. **File**: N/A (verification step)

- [X] T048 [P1] Run race detector — Execute `go test -race ./internal/tui/...` to check for race conditions in async data fetching and message passing. **File**: N/A (verification step)

---

## Dependency Graph

```
T001 (go.mod setup)
  |
T002-T007 (message types + data model) -- all parallelizable [P]
  |
T008-T009 (available provider impl) -> T010 (available provider tests)
T011 (finished provider impl) -> T012 (finished provider tests)
T013-T015 (list emitter update) -- parallelizable [P]
  |
T016-T023 (detail model) -> T024 (detail model tests)
  |
T025-T031 (content model focus) -> T032 (content model tests)
  |
T033-T034 (status bar) -> T035 (status bar tests)
  |
T036-T039 (app integration) -> T040 (app tests)
  |
T041-T046 (polish & edge cases) -- parallelizable [P]
  |
T047-T048 (verification)
```

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 48 |
| P1 (critical) | 36 |
| P2 (important) | 12 |
| Parallelizable | 14 |
| Phases | 10 |
| New files | 4 (`pipeline_detail.go`, `pipeline_detail_test.go`, `pipeline_detail_provider.go`, `pipeline_detail_provider_test.go`) |
| Modified files | 11 (`go.mod`, `header_messages.go`, `pipeline_list.go`, `pipeline_list_test.go`, `header_test.go`, `content.go`, `content_test.go`, `statusbar.go`, `statusbar_test.go`, `app.go`, `app_test.go`) |
