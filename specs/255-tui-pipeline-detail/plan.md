# Implementation Plan: TUI Pipeline Detail Right Pane

**Branch**: `255-tui-pipeline-detail` | **Date**: 2026-03-06 | **Spec**: `specs/255-tui-pipeline-detail/spec.md`
**Input**: Feature specification from `/specs/255-tui-pipeline-detail/spec.md`

## Summary

Add a pipeline detail right pane to the TUI showing context-sensitive detail views for the currently selected pipeline. Available pipelines show configuration metadata (description, steps with personas, inputs, outputs, dependencies). Finished pipelines show execution summaries (status, duration, branch, step results, artifacts, action hints). Running pipelines show a brief informational message. Focus management via Enter/Esc between left and right panes enables scrollable detail content. A `DetailDataProvider` interface abstracts data fetching, following the established `PipelineDataProvider` / `MetadataProvider` pattern.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `charmbracelet/bubbletea` v1.3.10, `charmbracelet/lipgloss` v1.1.0, `charmbracelet/bubbles/viewport` (already indirect dep via `huh`)
**Storage**: SQLite via `internal/state` (read-only queries for run details, artifacts, performance metrics)
**Testing**: `go test` with `testify/assert`, `testify/require`
**Target Platform**: Linux/macOS terminal (80–300 columns, 24–100 rows)
**Project Type**: Single Go binary — all changes in `internal/tui/` package
**Constraints**: No new external dependencies (bubbles/viewport is already indirect via huh); must not break existing tests

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new runtime dependencies. bubbles/viewport already indirect via huh. |
| P2: Manifest as SSOT | ✅ Pass | Available pipeline details parsed from YAML files discovered via manifest. |
| P3: Persona-Scoped Execution | N/A | TUI component, not a persona. |
| P4: Fresh Memory at Step Boundary | N/A | TUI component, not a pipeline step. |
| P5: Navigator-First Architecture | N/A | TUI component, not a pipeline execution. |
| P6: Contracts at Every Handover | N/A | TUI component, no inter-step handover. |
| P7: Relay via Dedicated Summarizer | N/A | TUI component, no context compaction. |
| P8: Ephemeral Workspaces | N/A | TUI component, no workspace creation. |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling. |
| P10: Observable Progress | ✅ Pass | Detail view surfaces step-level execution data for observability. |
| P11: Bounded Recursion | N/A | No recursion in TUI. |
| P12: Minimal Step State Machine | ✅ Pass | Uses existing step states from state store. |
| P13: Test Ownership | ✅ Pass | All new code will have tests; existing tests updated for modified types. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/255-tui-pipeline-detail/
├── plan.md              # This file
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/tui/
├── app.go                    # MODIFY — handle FocusChangedMsg, update q-quit guard
├── app_test.go               # MODIFY — update for FocusChangedMsg, focus-aware q-quit tests
├── content.go                # MODIFY — add focus management, PipelineDetailModel, key routing
├── content_test.go           # MODIFY — add focus transition tests, detail rendering tests
├── header.go                 # UNCHANGED
├── header_logo.go            # UNCHANGED
├── header_messages.go        # MODIFY — extend PipelineSelectedMsg with Name+Kind, add FocusChangedMsg+DetailDataMsg
├── header_metadata.go        # UNCHANGED
├── header_provider.go        # UNCHANGED
├── header_provider_test.go   # UNCHANGED
├── header_test.go            # MODIFY — update PipelineSelectedMsg literals for new fields
├── pipeline_detail.go        # NEW — PipelineDetailModel (Init, Update, View, rendering)
├── pipeline_detail_test.go   # NEW — comprehensive detail model tests
├── pipeline_detail_provider.go      # NEW — DetailDataProvider interface + DefaultDetailDataProvider
├── pipeline_detail_provider_test.go # NEW — provider tests with mock state store
├── pipeline_list.go          # MODIFY — update emitSelectionMsg to populate Name+Kind, handle Enter for focus
├── pipeline_list_test.go     # MODIFY — update PipelineSelectedMsg assertions for new fields
├── pipeline_messages.go      # UNCHANGED
├── pipeline_provider.go      # UNCHANGED
├── pipeline_provider_test.go # UNCHANGED
├── pipelines.go              # UNCHANGED
├── pipelines_test.go         # UNCHANGED
├── run_selector.go           # UNCHANGED
├── run_selector_test.go      # UNCHANGED
├── statusbar.go              # MODIFY — handle FocusChangedMsg, dynamic key hints
├── statusbar_test.go         # MODIFY — add focus-aware hint tests
└── theme.go                  # UNCHANGED
```

**Structure Decision**: All changes within `internal/tui/`. New files follow the established naming convention (`pipeline_detail*.go` matching `pipeline_list*.go`). No new packages needed.

## Implementation Strategy

### Phase A: Message Types & Provider Interface

**Files**: `header_messages.go`, `pipeline_detail_provider.go`, `pipeline_detail_provider_test.go`

1. **Extend `PipelineSelectedMsg`** with `Name string` and `Kind itemKind` fields:
   ```go
   type PipelineSelectedMsg struct {
       RunID         string
       BranchName    string
       BranchDeleted bool
       Name          string   // pipeline name for all item types
       Kind          itemKind // running/finished/available/sectionHeader
   }
   ```

2. **Add new message types**:
   ```go
   type FocusChangedMsg struct { Pane FocusPane }
   type DetailDataMsg struct {
       AvailableDetail *AvailableDetail
       FinishedDetail  *FinishedDetail
       Err             error
   }
   ```
   Note: `FocusPane` type (enum `FocusPaneLeft`/`FocusPaneRight`) also defined here.

3. **Define data projection types**:
   - `AvailableDetail` — name, description, category, steps (ID+persona), inputs, artifacts, dependencies
   - `FinishedDetail` — run ID, status, duration, branch, timestamps, error, failed step, step results, artifacts
   - `StepSummary` — step ID + persona
   - `StepResult` — step ID + status + duration + persona
   - `ArtifactInfo` — name + path + type

4. **Define `DetailDataProvider` interface**:
   ```go
   type DetailDataProvider interface {
       FetchAvailableDetail(name string) (*AvailableDetail, error)
       FetchFinishedDetail(runID string) (*FinishedDetail, error)
   }
   ```

5. **Implement `DefaultDetailDataProvider`**:
   - `FetchAvailableDetail(name)`: Scan pipelines directory for YAML file matching name, parse `pipeline.Pipeline`, map to `AvailableDetail` (steps with IDs and personas, input config, output artifacts across all steps, requires block)
   - `FetchFinishedDetail(runID)`: Query `store.GetRun(runID)` for run record, `store.GetPerformanceMetrics(runID, "")` for step results, `store.GetArtifacts(runID, "")` for artifacts. Compose into `FinishedDetail` with derived `FailedStep` from first failed metric.

6. **Tests**: Mock state store tests for `FetchFinishedDetail`, filesystem-based tests for `FetchAvailableDetail`.

### Phase B: PipelineSelectedMsg Emitter Update

**Files**: `pipeline_list.go`, `pipeline_list_test.go`, `header_test.go`

1. **Update `emitSelectionMsg()`** in `pipeline_list.go`:
   - Running items: populate `Name: r.Name`, `Kind: itemKindRunning`
   - Finished items: populate `Name: f.Name`, `Kind: itemKindFinished`
   - Available items: populate `Name: a.Name`, `Kind: itemKindAvailable`

2. **Update `handleKeyMsg`**: When Enter is pressed on a non-header, non-running pipeline item, emit a command that `ContentModel` can intercept for focus transition. The list itself does NOT change focus — `ContentModel` owns focus state.

3. **Update test assertions**: All `PipelineSelectedMsg` assertions in `pipeline_list_test.go` and `header_test.go` to include the new `Name` and `Kind` fields.

### Phase C: Pipeline Detail Model

**Files**: `pipeline_detail.go`, `pipeline_detail_test.go`

1. **Implement `PipelineDetailModel` struct**:
   ```go
   type PipelineDetailModel struct {
       width, height int
       focused       bool
       viewport      viewport.Model
       selectedName  string
       selectedKind  itemKind
       selectedRunID string
       availableDetail *AvailableDetail
       finishedDetail  *FinishedDetail
       branchDeleted   bool
       loading         bool
       errorMsg        string
       provider        DetailDataProvider
   }
   ```

2. **Constructor**: `NewPipelineDetailModel(provider DetailDataProvider)` — initializes viewport with zero dimensions.

3. **`SetSize(w, h)`**: Update viewport dimensions, re-render content.

4. **`SetFocused(bool)`**: Update `focused` field, reconfigure viewport key bindings.

5. **`Update(msg)`**:
   - `PipelineSelectedMsg`: Store selection, set loading=true, return async fetch command. If Kind is `itemKindSectionHeader`, clear detail and show placeholder.
   - `DetailDataMsg`: Store detail data, set loading=false, re-render content into viewport.
   - `tea.KeyMsg` (when focused): Forward to viewport for scroll handling.

6. **`View()`**: Renders one of:
   - **Placeholder**: Centered "Select a pipeline to view details" when no selection
   - **Loading**: "Loading..." indicator
   - **Error**: Error message
   - **Available detail**: Rendered configuration view
   - **Finished detail**: Rendered execution summary
   - **Running info**: Brief informational message with name and "Running" status

7. **Rendering functions** (pure functions, data → string):
   - `renderAvailableDetail(detail *AvailableDetail, width int) string`: Sections for name/description, steps table, input config, output artifacts, dependencies
   - `renderFinishedDetail(detail *FinishedDetail, width int, branchDeleted bool) string`: Sections for status/duration/branch, step results table, artifacts, action hints
   - `renderRunningInfo(name string, width int) string`: Brief informational message

8. **Tests**:
   - Placeholder rendering when no selection
   - Available detail rendering with all fields
   - Finished detail rendering (completed, failed, cancelled)
   - Failed pipeline shows error and failed step
   - Branch deleted indicator
   - Zero artifacts shows "No artifacts produced"
   - Focus state change
   - Scroll handling when focused
   - Resize re-renders content
   - Selection change resets scroll position

### Phase D: Content Model Focus Management

**Files**: `content.go`, `content_test.go`

1. **Add focus state and detail model** to `ContentModel`:
   ```go
   type ContentModel struct {
       width, height int
       list          PipelineListModel
       detail        PipelineDetailModel
       focus         FocusPane
   }
   ```

2. **Update `NewContentModel`**: Accept both `PipelineDataProvider` and `DetailDataProvider`. Initialize detail model.

3. **Update `SetSize`**: Propagate right pane dimensions to detail model.

4. **Update `Init()`**: Return `tea.Batch(m.list.Init(), m.detail.Init())`.

5. **Update `Update(msg)`**:
   - **`PipelineSelectedMsg`**: Forward to both list (unchanged) and detail model. If a new pipeline is selected while right pane is focused, keep focus on right but reset scroll.
   - **`DetailDataMsg`**: Forward to detail model only.
   - **`tea.KeyMsg` (Enter)**: When focus is left and cursor is on a non-header, non-running item, transition focus to right. Emit `FocusChangedMsg{FocusPaneRight}`. Call `m.list.SetFocused(false)`, `m.detail.SetFocused(true)`.
   - **`tea.KeyMsg` (Esc)**: When focus is right, transition focus to left. Emit `FocusChangedMsg{FocusPaneLeft}`. Call `m.list.SetFocused(true)`, `m.detail.SetFocused(false)`.
   - **Other `tea.KeyMsg`**: Route to focused child only.
   - **Other messages**: Forward to both children (data refresh, etc.).

6. **Update `View()`**: Replace static placeholder with `m.detail.View()`. Apply visual focus indicators: dimmed left pane border when right is focused, highlighted right pane border when focused.

7. **Handle Enter key conflict**: Currently `PipelineListModel.handleKeyMsg` handles Enter for section collapse. Now `ContentModel` needs to intercept Enter for focus transition. Solution: `ContentModel` checks if cursor is on a focusable item BEFORE forwarding Enter to list. If focusable (available/finished), transition focus. If not (header/running), forward Enter to list for existing behavior.

8. **Tests**:
   - Focus starts on left pane
   - Enter on available item transitions focus to right
   - Enter on finished item transitions focus to right
   - Enter on section header does NOT transition focus (collapses section)
   - Enter on running item does NOT transition focus
   - Esc from right pane returns focus to left
   - Arrow keys in right pane scroll detail, not list
   - Arrow keys in left pane navigate list, not detail
   - FocusChangedMsg emitted on transitions
   - Selection change resets scroll in detail
   - Right pane width calculation (width - leftPaneWidth)

### Phase E: Status Bar Dynamic Hints

**Files**: `statusbar.go`, `statusbar_test.go`

1. **Add focus-aware hints** to `StatusBarModel`:
   ```go
   type StatusBarModel struct {
       width        int
       contextLabel string
       focusPane    FocusPane
   }
   ```

2. **Handle `FocusChangedMsg`** in `Update()`:
   ```go
   func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
       switch msg := msg.(type) {
       case FocusChangedMsg:
           m.focusPane = msg.Pane
       }
       return m, nil
   }
   ```

3. **Dynamic hints in `View()`**:
   - Left pane focused: `"↑↓: navigate  Enter: view  /: filter  q: quit"`
   - Right pane focused: `"↑↓: scroll  Esc: back  q: quit"`

4. **Tests**:
   - Default hints (left pane focused)
   - Hints update on FocusChangedMsg
   - Right pane hints include scroll and Esc

### Phase F: App Model Integration

**Files**: `app.go`, `app_test.go`

1. **Update `NewAppModel`**: Accept `DetailDataProvider` in addition to existing parameters. Pass to `NewContentModel`.

2. **Forward `FocusChangedMsg`**: Route to status bar in `Update()`.

3. **Forward `DetailDataMsg`**: Route to content model in `Update()`. (Already handled by forwarding all messages to content.)

4. **Update q-quit guard**: Currently `msg.String() == "q" && !m.content.list.filtering` — also check that right pane is not focused (or allow q-quit from both panes as-is, since the right pane has no text input).

5. **Update `RunTUI()`**: Create `DefaultDetailDataProvider` and pass to `NewAppModel`.

6. **Tests**:
   - FocusChangedMsg forwarded to status bar
   - DetailDataMsg forwarded to content
   - Updated constructor signatures

### Phase G: Integration & Polish

1. Verify `go test ./internal/tui/...` passes
2. Verify `go test ./...` passes (no regressions)
3. Verify NO_COLOR compliance (detail pane rendering respects lipgloss.HasDarkBackground check)
4. Manual smoke test at 80×24, 120×40, and 200×50
5. Verify edge cases:
   - Terminal resize while detail is visible
   - Narrow terminal (80 columns → right pane ~55 columns)
   - Very long pipeline names and error messages truncate gracefully
   - Scrolling clamps at boundaries

## Complexity Tracking

_No constitution violations identified._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none)    | —         | —                                    |
