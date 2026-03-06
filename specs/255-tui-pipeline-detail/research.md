# Research: TUI Pipeline Detail Right Pane

**Date**: 2026-03-06 | **Feature**: #255

## R1: Bubble Tea Viewport for Scrollable Content

**Decision**: Use `charmbracelet/bubbles/viewport` for the right pane scrollable content.

**Rationale**: The `viewport.Model` from bubbles provides built-in scrolling, line clamping, and percentage-based position tracking. It handles content overflow natively with `SetContent(string)` and processes `tea.KeyMsg` for ↑/↓ scrolling. This avoids reimplementing scroll logic that `PipelineListModel` already manually handles for the left pane.

**Alternatives**:
- Manual scroll tracking (like `PipelineListModel`): Works but duplicates scroll logic. The list model's `adjustScrollOffset` is item-based; the detail pane needs line-based scrolling for free-form rendered text.
- `charmbracelet/bubbles/paginator`: Designed for page-based content (pagination), not continuous scrolling. Wrong model for detail views.

**Note**: `bubbles/viewport` is already an indirect dependency via `huh` (used in `run_selector.go`). No new dependency introduced.

## R2: Focus Management Pattern in Bubble Tea

**Decision**: `ContentModel` owns a `FocusState` enum (`FocusLeft`/`FocusRight`) and conditionally routes key messages to the focused child. Each child has a `SetFocused(bool)` method to update visual styling.

**Rationale**: This follows the established Bubble Tea pattern where parent models own focus state and delegate key routing. The existing `PipelineListModel.focused` field already supports this — it's set to `true` by default and guards `handleKeyMsg`. The detail model will have the same pattern: ignore keys when unfocused.

**Alternatives**:
- Global focus manager: Overengineered for two panes. Not needed until the TUI has 3+ focusable regions.
- Message-based focus notifications between siblings: Creates coupling. Parent-owned focus with `SetFocused()` is simpler and follows the existing `SetSize()` pattern.

## R3: Data Fetching Strategy for Detail Pane

**Decision**: Introduce a `DetailDataProvider` interface separate from `PipelineDataProvider`. The detail provider has methods for fetching run details (step events, artifacts, performance metrics) and parsing full pipeline YAML for available pipeline details.

**Rationale**: `PipelineDataProvider` supplies list-level summary data (name, status, duration). The detail pane needs much more: step-by-step results with individual timings, artifacts with paths, error messages, and for available pipelines, full step definitions with persona assignments. Separate interfaces maintain single responsibility.

**Alternatives**:
- Extending `PipelineDataProvider` with detail methods: Violates single responsibility. List tests would need to mock methods they don't use.
- Fetching detail data inline in the model: Mixes I/O with UI logic and blocks testing with mocks.

## R4: PipelineSelectedMsg Extension

**Decision**: Add `Name string` and `Kind itemKind` fields to the existing `PipelineSelectedMsg` struct.

**Rationale**: The current `PipelineSelectedMsg{RunID: "", BranchName: ""}` for available pipelines provides no identity — the detail pane cannot determine which available pipeline was selected. Adding `Name` (pipeline name) and `Kind` (item type: running/finished/available) enables the detail pane to dispatch to the correct view and fetch the right data. This is a minimal, backward-compatible extension to an existing message.

**Alternatives**:
- New message type (e.g., `AvailablePipelineSelectedMsg`): Creates two parallel message flows for the same concept (selection). Every consumer would need to handle both.
- Including the full data in the message: Bloats the message. Better to send an identifier and let the consumer fetch what it needs.

## R5: Content Rendering Strategy

**Decision**: Render detail content as a single styled string using lipgloss, then set it as viewport content. No custom line-by-line rendering needed.

**Rationale**: The detail pane displays structured, read-only information (not interactive items like the list). A single rendered string with sections (heading styles, tables, indentation) works well with `viewport.SetContent()`. This approach:
- Separates rendering logic (pure function: data → string) from scroll management (viewport)
- Makes rendering testable without viewport state
- Handles terminal resize naturally (re-render at new width, reset viewport content)

**Alternatives**:
- Line-by-line rendering with manual scrolling: More complex, same result. The viewport abstraction is better.
- Interactive elements within detail pane: Out of scope — action hints (#256+) will handle interactivity later.
