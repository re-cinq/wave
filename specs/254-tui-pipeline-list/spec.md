# Feature Specification: TUI Pipeline List Left Pane

**Feature Branch**: `254-tui-pipeline-list`  
**Created**: 2026-03-05  
**Status**: Clarified  
**Input**: User description: "https://github.com/re-cinq/wave/issues/254 — feat(tui): pipeline list left pane with running/finished/available sections and navigation"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - View Pipeline Inventory at a Glance (Priority: P1)

A developer launches Wave TUI and immediately sees all pipelines organized into three distinct sections: Running (active executions with elapsed time), Finished (completed/failed with status and duration), and Available (all configured pipelines from the manifest). Each section header shows a count (e.g., "Running (2)"). The left pane is focused by default, giving the user an instant overview of the pipeline landscape without any interaction.

**Why this priority**: The pipeline list is the foundational navigation surface for the entire TUI — every downstream feature (detail views, launching, live output) depends on it being visible and populated. Without this, the TUI has no actionable content area.

**Independent Test**: Can be tested by launching the TUI with a combination of running, finished, and available pipelines and verifying that all three sections render with correct counts, items, and ordering.

**Acceptance Scenarios**:

1. **Given** the TUI is launched with 2 running pipelines, 5 finished pipelines, and 8 available pipelines in `wave.yaml`, **When** the main view renders, **Then** the left pane displays three sections: "Running (2)", "Finished (5)", "Available (8)" with correct items in each.
2. **Given** a pipeline is currently running, **When** it appears in the Running section, **Then** an elapsed time indicator shows how long it has been active, updating periodically.
3. **Given** multiple running pipelines exist, **When** the Running section renders, **Then** items are sorted newest-first (most recently started at top).
4. **Given** finished pipelines exist with mixed statuses, **When** the Finished section renders, **Then** each item shows its terminal status (completed/failed/cancelled) and total duration.
5. **Given** no pipelines are running, **When** the Running section renders, **Then** the section header shows "Running (0)" and the section body is empty or shows a placeholder message.

---

### User Story 2 - Navigate the Pipeline List with Keyboard (Priority: P1)

A developer uses arrow keys (↑/↓) to move a visual selection cursor through the pipeline list. The navigable items include both section headers and pipeline items. Section headers are navigable targets (required for collapse/expand in US-5) but are visually distinct from pipeline items. The cursor moves seamlessly from the last item in one section to the header of the next section. The currently selected item is highlighted with a visual indicator (▶ or ›). The selection state determines what appears in the right detail pane (handled by a future issue).

**Why this priority**: Navigation is essential to make the list interactive. Without it, the list is read-only and the user cannot select pipelines for further action — blocking all downstream detail/action features.

**Independent Test**: Can be tested by rendering the list with items in all three sections, simulating ↑/↓ key events, and asserting the cursor position updates correctly including cross-section traversal and section header stops.

**Acceptance Scenarios**:

1. **Given** the left pane is focused with pipelines in all three sections, **When** the user presses ↓, **Then** the selection moves to the next navigable item (section header or pipeline item) in the list.
2. **Given** the selection is on the first item in the list (the Running section header), **When** the user presses ↑, **Then** the selection does not wrap to the bottom (stays at top).
3. **Given** the selection is on the last item in the list, **When** the user presses ↓, **Then** the selection does not wrap to the top (stays at bottom).
4. **Given** a pipeline item is selected, **When** the view renders, **Then** the selected item displays a visual indicator (▶) and is styled distinctly from unselected items. Section headers use a different highlight style (e.g., inverse/bold) when selected.
5. **Given** the left pane has focus, **When** the selection changes to a pipeline item (not a section header), **Then** a `PipelineSelectedMsg` is emitted to notify other components (header, future right pane) of the newly selected pipeline. The message includes the pipeline name and, for Running/Finished items, the `RunID` and `BranchName` from the `RunRecord`. For Available items, `RunID` is empty.

---

### User Story 3 - Filter Pipelines by Search (Priority: P2)

A developer presses `/` to activate a search/filter input at the top of the left pane. As they type, the list filters in real-time to show only pipelines whose names match the query (case-insensitive substring match). Pressing Escape clears the filter and restores the full list. The filter applies across all three sections simultaneously.

**Why this priority**: Search becomes important as the number of pipelines grows, but the list is usable without it for smaller projects. It enhances efficiency but is not a prerequisite for core navigation.

**Independent Test**: Can be tested by rendering a list with 10+ pipelines, activating search with `/`, typing a partial name, and verifying the displayed items match the query across all sections.

**Acceptance Scenarios**:

1. **Given** the left pane is focused, **When** the user presses `/`, **Then** a text input field appears at the top of the pane with a cursor and filter icon.
2. **Given** the filter input is active and the user types "spec", **When** the list re-renders, **Then** only pipelines whose names contain "spec" (case-insensitive) remain visible.
3. **Given** the filter is active with results showing, **When** the user presses Escape, **Then** the filter input closes, the query clears, and the full list is restored.
4. **Given** the filter matches zero pipelines, **When** the list renders, **Then** a "No matching pipelines" message is displayed.
5. **Given** the filter is active, **When** the user presses ↑/↓, **Then** navigation works within the filtered results.

---

### User Story 4 - Scroll Through Long Pipeline Lists (Priority: P2)

A developer with many pipelines (more items than fit in the visible area) scrolls the list using arrow key navigation. The viewport follows the selection cursor, keeping the selected item visible. Section headers remain contextually visible when scrolling through their items.

**Why this priority**: Scrolling is necessary for real-world usage but only becomes critical when the pipeline count exceeds the terminal height. Most early users will have fewer pipelines, so this is secondary to basic rendering and navigation.

**Independent Test**: Can be tested by creating a list taller than the available content height, navigating past the visible area, and verifying the viewport scrolls to keep the selected item in view.

**Acceptance Scenarios**:

1. **Given** the list contains more items than the visible height allows, **When** the user navigates past the bottom edge, **Then** the viewport scrolls down to keep the selected item visible.
2. **Given** the viewport has scrolled down, **When** the user navigates back up past the top edge, **Then** the viewport scrolls up accordingly.
3. **Given** a section has many items, **When** scrolling through them, **Then** the section header remains visible or the current section context is otherwise indicated.

---

### User Story 5 - Collapse and Expand Sections (Priority: P3)

A developer can collapse a section (e.g., the Available section) to reduce visual noise and focus on Running or Finished pipelines. Pressing a toggle key on a section header collapses or expands that section. Collapsed sections show only the header with count and a collapsed indicator.

**Why this priority**: Section collapsing is a convenience feature that improves the experience with many pipelines but is not required for core functionality.

**Independent Test**: Can be tested by rendering all three sections, collapsing one, verifying items are hidden, and expanding it again to restore visibility.

**Acceptance Scenarios**:

1. **Given** the cursor is on a section header, **When** the user presses Enter or a toggle key, **Then** the section collapses — hiding all items and showing a collapsed indicator (▸) on the header.
2. **Given** a section is collapsed, **When** the user presses the toggle key on its header, **Then** the section expands and all items reappear.
3. **Given** a section is collapsed, **When** the user navigates with ↑/↓, **Then** the cursor skips over the hidden items and moves to the next visible item or section header.

---

### Edge Cases

- What happens when the SQLite state database is inaccessible or empty? The Running and Finished sections show zero items; the Available section still populates from `wave.yaml`.
- What happens when `wave.yaml` is missing or malformed? The Available section shows an error indicator; Running and Finished sections still populate from the database.
- What happens when the terminal is resized below the minimum width? The left pane degrades gracefully — truncates pipeline names rather than breaking layout.
- What happens when a pipeline run changes status while the TUI is open (e.g., running → completed)? The list updates on the next data refresh tick (5-second polling interval for run state, matching the urgency of running pipeline status), moving the item from Running to Finished.
- What happens when a pipeline name is very long? The name is truncated with ellipsis (…) to fit within the pane width.
- What happens when all sections are empty? A centered "No pipelines found" placeholder is shown in the left pane.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST render a left pane in the main content area displaying pipeline items grouped into three sections: Running, Finished, and Available. The left pane occupies 30% of the content area width (minimum 25 columns, maximum 50 columns). The remaining width is reserved for a future right detail pane (rendered as empty/placeholder for now).
- **FR-002**: Each section header MUST display the section name and the count of items in parentheses (e.g., "Running (2)"). Section headers are navigable items in the cursor list.
- **FR-003**: Running section items MUST display the pipeline name and an elapsed time indicator, sorted newest-first by start time.
- **FR-004**: Finished section items MUST display the pipeline name, terminal status (completed/failed/cancelled), and total duration.
- **FR-005**: Available section items MUST display pipeline names discovered from the pipeline manifest directory, sorted alphabetically.
- **FR-006**: System MUST support ↑/↓ arrow key navigation with a visual selection indicator (▶ for pipeline items, bold/inverse for section headers) that moves through all navigable items including section headers.
- **FR-007**: Navigation MUST clamp at list boundaries — no wrapping from top to bottom or vice versa.
- **FR-008**: System MUST emit a `PipelineSelectedMsg` (reusing the existing message type from `header_messages.go`) when the cursor moves to a pipeline item. For Running/Finished items, the message includes `RunID` and `BranchName` from the `RunRecord`. For Available items, `RunID` is empty and `BranchName` is empty. Moving to a section header does NOT emit a selection message.
- **FR-009**: System MUST support a `/` key binding that activates a text filter input for case-insensitive substring matching across all sections.
- **FR-010**: System MUST support Escape key to dismiss the filter input and restore the full list.
- **FR-011**: System MUST scroll the viewport to keep the selected item visible when the list exceeds the visible area height.
- **FR-012**: The left pane MUST be focused by default when the TUI launches.
- **FR-013**: Running pipelines MUST query their data from the SQLite state database via a `PipelineDataProvider` interface (following the `MetadataProvider` pattern from `header_provider.go`). The provider wraps `GetRunningRuns()` from the `StateStore`.
- **FR-014**: Finished pipelines MUST query their data from the SQLite state database via the same `PipelineDataProvider` interface, wrapping `ListRuns()` with a terminal status filter. Default limit: 20 most recent finished runs.
- **FR-015**: Available pipelines MUST be loaded using the existing `DiscoverPipelines` function or equivalent manifest-based discovery, also exposed via the `PipelineDataProvider` interface.
- **FR-016**: System SHOULD support section collapse/expand toggling on section headers.
- **FR-017**: System MUST integrate with the existing `ContentModel` component, replacing the placeholder text currently rendered in `content.go`. The `ContentModel` is refactored to compose a `PipelineListModel` (left pane) and a placeholder right pane using `lipgloss.JoinHorizontal`.
- **FR-018**: System MUST respect `NO_COLOR` environment variable and terminal theme conventions via lipgloss.
- **FR-019**: System MUST poll for data updates using a `tea.Tick` at a 5-second interval for run state (Running/Finished sections). Available pipelines are loaded once at startup and on manifest change.

### Key Entities

- **PipelineListModel**: The Bubble Tea model for the left pane, implementing `Init()`, `Update()`, and `View()`. Owns the section data, selection state, filter state, and scroll offset.
- **PipelineListSection**: A grouping container (Running, Finished, Available) with a label, item count, collapsed state, and a slice of `PipelineListItem`.
- **PipelineListItem**: A single entry within a section — holds a pipeline name, display metadata (status, elapsed time, duration), and a reference to the underlying data source (RunRecord for Running/Finished, PipelineInfo for Available).
- **PipelineDataProvider**: Interface for fetching pipeline data, following the `MetadataProvider` pattern. Methods: `FetchRunningPipelines() ([]RunRecord, error)`, `FetchFinishedPipelines(limit int) ([]RunRecord, error)`, `FetchAvailablePipelines() ([]PipelineInfo, error)`. Enables mock injection for testing.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All three sections (Running, Finished, Available) render correctly with accurate counts when the TUI launches with known pipeline data.
- **SC-002**: Arrow key navigation traverses all navigable items (section headers and pipeline items) across all sections in the correct order without errors or panics.
- **SC-003**: The `/` search filter reduces the displayed list to only matching items within 1 render cycle of each keystroke.
- **SC-004**: Viewport scrolling keeps the selected item visible for lists up to 100 items tall.
- **SC-005**: The left pane renders without visual artifacts at terminal widths from 80 to 300 columns and heights from 24 to 100 rows.
- **SC-006**: Component integration with the existing `AppModel` and `ContentModel` does not break any existing tests (`go test ./internal/tui/...` passes).
- **SC-007**: The pipeline list correctly reflects state database contents, verified by unit tests with mock `PipelineDataProvider` implementations.

## Clarifications _(resolved during refinement)_

### C1: Content area split strategy (left pane width and layout)

**Ambiguity**: The current `ContentModel` is a single component with no left/right split. FR-001 references a "left pane" but the spec did not define width ratio, right pane behavior, or how `ContentModel` is refactored.

**Resolution**: The left pane occupies 30% of the content area width (min 25 columns, max 50 columns). `ContentModel` is refactored to compose a `PipelineListModel` (left) and a placeholder right pane using `lipgloss.JoinHorizontal`. This matches standard TUI sidebar patterns (e.g., k9s, lazygit) and leaves room for the detail pane in a future issue.

**Rationale**: 30% is the industry-standard sidebar ratio for master-detail TUI layouts. Min/max bounds ensure usability at both 80-column and ultra-wide terminals.

### C2: Section headers as navigable cursor targets

**Ambiguity**: US-2 says cursor "moves seamlessly across section boundaries" (could imply headers are skipped), but US-5 says "the cursor is on a section header" for collapse (requires headers to be navigable).

**Resolution**: Section headers ARE navigable items in the cursor list. The cursor traverses: `[Running header] → [running item 1] → ... → [Finished header] → [finished item 1] → ... → [Available header] → [available item 1] → ...`. Section headers use a visually distinct selection style (bold/inverse) compared to pipeline items (▶ indicator). Selection messages are only emitted when a pipeline item (not a header) is selected.

**Rationale**: Making headers navigable is required for the collapse/expand feature (US-5). It also provides clear visual orientation when scrolling through long lists. The distinct styling prevents user confusion between selecting a header vs. an item.

### C3: Data refresh mechanism (polling interval and approach)

**Ambiguity**: Edge case mentions "next data refresh cycle" but the spec did not define the polling interval, refresh approach, or which sections refresh.

**Resolution**: A `tea.Tick` at 5-second intervals triggers a data refresh for Running and Finished sections. Available pipelines are loaded once at startup (and could be refreshed on manifest file change in a future enhancement). This interval balances responsiveness (running pipelines need timely updates) with system load. The header already uses `gitRefreshInterval = 30 * time.Second` for lower-priority git state; 5 seconds is appropriate for active pipeline monitoring.

**Rationale**: 5 seconds provides near-real-time feedback for running pipelines while being reasonable for SQLite read load. It's a common polling interval in monitoring dashboards (e.g., Grafana defaults to 5s).

### C4: Selection message type for mixed item types

**Ambiguity**: `PipelineSelectedMsg` already exists in `header_messages.go` with `RunID` and `BranchName` fields. The spec's FR-008 said "a message is emitted" but didn't specify whether to reuse the existing type. Available pipelines have no `RunID`, creating a type mismatch.

**Resolution**: Reuse the existing `PipelineSelectedMsg` type. For Running/Finished items, populate `RunID` and `BranchName` from the `RunRecord`. For Available items, send with empty `RunID` and empty `BranchName`. The header already handles `PipelineSelectedMsg` for branch override display, so this maintains consistency. No new message types needed.

**Rationale**: Reusing the existing message avoids message type proliferation and leverages the header's existing `PipelineSelectedMsg` handler. Empty `RunID` is a clear sentinel for "no active run" which downstream components can branch on.

### C5: State store injection pattern (provider interface)

**Ambiguity**: FR-013/014 reference `GetRunningRuns` and `ListRuns` from the state database, but the spec didn't specify how the state store is injected into the pipeline list component. The TUI already uses a `MetadataProvider` interface for the header — should the list follow the same pattern?

**Resolution**: Introduce a `PipelineDataProvider` interface following the established `MetadataProvider` pattern from `header_provider.go`. The interface wraps `FetchRunningPipelines()`, `FetchFinishedPipelines(limit)`, and `FetchAvailablePipelines()`. A `DefaultPipelineDataProvider` implementation wraps the `StateStore` (using `NewReadOnlyStateStore` for concurrent access) and `DiscoverPipelines`. This enables straightforward mock injection in tests.

**Rationale**: The provider interface pattern is already proven in the codebase (`MetadataProvider` / `DefaultMetadataProvider`). It decouples the TUI component from the state package, enables unit testing with mocks, and allows future data source changes without modifying the list component.
