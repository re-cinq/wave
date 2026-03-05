# Implementation Plan: TUI Pipeline List Left Pane

**Branch**: `254-tui-pipeline-list` | **Date**: 2026-03-05 | **Spec**: `specs/254-tui-pipeline-list/spec.md`
**Input**: Feature specification from `/specs/254-tui-pipeline-list/spec.md`

## Summary

Add a pipeline list left pane to the TUI content area, displaying Running, Finished, and Available pipelines in three navigable sections. The `ContentModel` is refactored from a single placeholder to a left/right split composition. A `PipelineDataProvider` interface (following the established `MetadataProvider` pattern) abstracts data fetching from the state store and manifest discovery. Keyboard navigation (↑/↓), search/filter (`/`), viewport scrolling, and section collapse/expand are implemented as an interactive `PipelineListModel` Bubble Tea component.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `charmbracelet/bubbletea` v1.3.10, `charmbracelet/lipgloss` v1.1.0, `charmbracelet/bubbles` (textinput — already indirect dep)
**Storage**: SQLite via `internal/state` (read-only `NewReadOnlyStateStore`)
**Testing**: `go test` with `testify/assert`, `testify/require`
**Target Platform**: Linux/macOS terminal (80–300 columns, 24–100 rows)
**Project Type**: Single Go binary — all changes in `internal/tui/` package
**Constraints**: No new external dependencies (bubbles/textinput is already indirect); must not break existing tests

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new runtime dependencies. bubbles/textinput is already indirect via huh. |
| P2: Manifest as SSOT | ✅ Pass | Available pipelines loaded via `DiscoverPipelines` (manifest-based). |
| P3: Persona-Scoped Execution | N/A | TUI component, not a persona. |
| P4: Fresh Memory at Step Boundary | N/A | TUI component, not a pipeline step. |
| P5: Navigator-First Architecture | N/A | TUI component, not a pipeline execution. |
| P6: Contracts at Every Handover | N/A | TUI component, no inter-step handover. |
| P7: Relay via Dedicated Summarizer | N/A | TUI component, no context compaction. |
| P8: Ephemeral Workspaces | N/A | TUI component, no workspace creation. |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling. |
| P10: Observable Progress | ✅ Pass | Running pipelines show elapsed time; status visible at a glance. |
| P11: Bounded Recursion | N/A | No recursion in TUI. |
| P12: Minimal Step State Machine | ✅ Pass | Uses existing step states from state store. |
| P13: Test Ownership | ✅ Pass | All new code will have tests; existing tests updated for refactored ContentModel. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/254-tui-pipeline-list/
├── plan.md              # This file
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/tui/
├── app.go                    # MODIFY — accept PipelineDataProvider, forward msgs to content
├── app_test.go               # MODIFY — update for new ContentModel + provider injection
├── content.go                # MODIFY — refactor to left/right pane composition
├── content_test.go           # MODIFY — update for new ContentModel behavior
├── header.go                 # UNCHANGED
├── header_logo.go            # UNCHANGED
├── header_messages.go        # UNCHANGED (PipelineSelectedMsg already defined)
├── header_metadata.go        # UNCHANGED
├── header_provider.go        # UNCHANGED
├── header_provider_test.go   # UNCHANGED
├── header_test.go            # UNCHANGED
├── pipeline_list.go          # NEW — PipelineListModel (Init, Update, View)
├── pipeline_list_test.go     # NEW — comprehensive tests for list model
├── pipeline_messages.go      # NEW — PipelineDataMsg, PipelineRefreshTickMsg
├── pipeline_provider.go      # NEW — PipelineDataProvider interface + DefaultPipelineDataProvider
├── pipeline_provider_test.go # NEW — provider tests with mock state store
├── pipelines.go              # UNCHANGED (PipelineInfo, DiscoverPipelines)
├── pipelines_test.go         # UNCHANGED
├── run_selector.go           # UNCHANGED
├── run_selector_test.go      # UNCHANGED
├── statusbar.go              # UNCHANGED
├── statusbar_test.go         # UNCHANGED
└── theme.go                  # UNCHANGED
```

**Structure Decision**: All changes are within `internal/tui/`. New files follow the established naming convention (`pipeline_*.go` matching `header_*.go`). No new packages needed.

## Implementation Strategy

### Phase A: Data Layer (provider + messages)

**Files**: `pipeline_provider.go`, `pipeline_provider_test.go`, `pipeline_messages.go`

1. Define `RunningPipeline` and `FinishedPipeline` value types (TUI-specific projections of `state.RunRecord`)
2. Define `PipelineDataProvider` interface with three fetch methods
3. Implement `DefaultPipelineDataProvider` wrapping `state.StateStore` + `DiscoverPipelines`
   - `FetchRunningPipelines()` → calls `store.GetRunningRuns()`, maps to `RunningPipeline`
   - `FetchFinishedPipelines(limit)` → calls `store.ListRuns(ListRunsOptions{Status: in("completed","failed","cancelled"), Limit: limit})`, maps to `FinishedPipeline`. Note: the `ListRunsOptions.Status` is a single string; we'll need three separate queries (completed, failed, cancelled) merged and sorted, OR use a custom query. Simplest: query without status filter, then filter in Go for terminal statuses. Actually, looking at the store, `ListRuns` sorts by `started_at DESC` and supports limit. We can call `ListRuns` with no status filter and limit=20, then filter in Go for terminal statuses (completed/failed/cancelled). Or make 3 calls. Simplest approach: call with no status filter, limit=60 (buffer), filter in Go to terminal statuses, take first 20.
   - `FetchAvailablePipelines()` → calls `DiscoverPipelines(pipelinesDir)`
4. Define `PipelineDataMsg` and `PipelineRefreshTickMsg` message types
5. Tests: mock state store, verify mapping logic, verify error handling

### Phase B: List Model (core component)

**Files**: `pipeline_list.go`, `pipeline_list_test.go`

1. Define `navigableItem` and `itemKind` types
2. Implement `PipelineListModel` struct with all fields from data model
3. Implement `NewPipelineListModel(provider PipelineDataProvider)` constructor
4. Implement `Init()` — returns batch of `fetchPipelineData` + `refreshTick()`
5. Implement `buildNavigableItems()` — computes flat list from sections, respecting filter and collapse state
6. Implement `Update()`:
   - `PipelineDataMsg` → update section data, rebuild navigable items, emit `RunningCountMsg` for header
   - `PipelineRefreshTickMsg` → return `tea.Batch(fetchPipelineData, refreshTick())`
   - `tea.KeyMsg` (↑/↓) → move cursor, clamp, emit `PipelineSelectedMsg` if on a pipeline item
   - `tea.KeyMsg` (`/`) → activate filter mode
   - `tea.KeyMsg` (Escape) → dismiss filter, clear query, rebuild items
   - `tea.KeyMsg` (Enter on section header) → toggle collapse
   - Forward to `filterInput` when filtering
7. Implement `View()`:
   - Render each navigable item with appropriate styling
   - Selection indicator: `▶` for pipeline items, bold/inverse for section headers
   - Section headers: `"Running (N)"`, `"Finished (N)"`, `"Available (N)"`
   - Running items: `"name   2m30s"`
   - Finished items: `"name   ✓ completed  1m15s"` or `"name   ✗ failed  45s"`
   - Available items: `"name"`
   - Viewport scrolling: calculate visible window based on height and cursor position
   - Filter input rendered at top when active
   - Truncation: pipeline names truncated with `…` when exceeding pane width
   - Empty state: `"No pipelines found"` when all sections empty
   - Filter empty: `"No matching pipelines"` when filter matches nothing
8. Implement `SetSize(w, h)` for resize handling
9. Tests:
   - Navigation: cursor movement, boundary clamping, cross-section traversal
   - Filter: activation, matching, dismissal, empty results
   - Collapse: toggle, cursor skip over hidden items
   - Selection messages: emitted for pipeline items, not headers
   - View rendering: section headers with counts, item formatting
   - Data refresh: section data updates, cursor preservation
   - Edge cases: empty sections, all empty, very long names

### Phase C: Content Model Refactoring

**Files**: `content.go`, `content_test.go`

1. Refactor `ContentModel` to hold a `PipelineListModel` (left pane)
2. Add `NewContentModel(provider PipelineDataProvider)` — injects provider into list model
3. Implement `Init()` — delegates to `PipelineListModel.Init()`
4. Implement `Update()` — forwards all messages to `PipelineListModel`, returns combined commands
5. Implement `View()`:
   - Calculate left pane width: `min(max(width*30/100, 25), 50)`
   - Right pane width: `width - leftWidth`
   - Left: `PipelineListModel.View()`
   - Right: placeholder text (centered "Pipeline details coming soon")
   - Join with `lipgloss.JoinHorizontal`
6. Implement `SetSize(w, h)` — propagate to list model with left pane width
7. Update tests:
   - Remove "Pipelines view coming soon" assertion (replaced by list)
   - Add tests for left/right split width calculation
   - Add tests for message forwarding to list model

### Phase D: App Model Integration

**Files**: `app.go`, `app_test.go`

1. Update `NewAppModel` to accept `PipelineDataProvider` in addition to `MetadataProvider`
2. Pass provider to `NewContentModel(provider)`
3. Update `Init()` — return `tea.Batch(m.header.Init(), m.content.Init())`
4. Update `Update()`:
   - Forward all messages to `m.content.Update()` (in addition to existing header forwarding)
   - Collect commands from both header and content updates
   - For `PipelineSelectedMsg`: forward to both header (existing) and content
   - Route ↑/↓ and `/` keys to content instead of (or in addition to) header
5. Handle key routing: when left pane is focused (default), keys go to content
6. Update `RunTUI()` — create `DefaultPipelineDataProvider` and pass to `NewAppModel`
7. Update tests:
   - `NewAppModel` now takes both providers
   - Window resize propagates to content with list model
   - Key events forwarded to content
   - Integration: PipelineSelectedMsg flows from content to header

### Phase E: Integration & Polish

1. Verify `go test ./internal/tui/...` passes
2. Verify `go test ./...` passes (no regressions)
3. Manual smoke test at 80×24 and 200×50
4. Verify NO_COLOR compliance
5. Verify status bar key hints update (add ↑↓ / filter hints)

## Complexity Tracking

_No constitution violations identified._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none)    | —         | —                                    |
