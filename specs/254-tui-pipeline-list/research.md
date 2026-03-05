# Research: TUI Pipeline List Left Pane

**Feature**: #254 — TUI Pipeline List Left Pane  
**Date**: 2026-03-05

## R1: Bubble Tea List Component Patterns

**Decision**: Build a custom list model from scratch rather than using `charmbracelet/bubbles/list`.

**Rationale**: The spec requires three grouped sections (Running, Finished, Available) with section headers as navigable cursor targets, section collapse, and custom rendering per section type (elapsed time, status badges, etc.). The `bubbles/list` component is designed for flat, homogeneous lists with a built-in filter — it doesn't support grouped sections, heterogeneous item rendering, or navigable section headers. Building from scratch gives full control and avoids fighting the library.

**Alternatives Rejected**:
- `bubbles/list` — flat list, no sections, no heterogeneous rendering
- `bubbles/table` — tabular layout, wrong UX metaphor
- Third-party list libraries — adds dependencies, no clear win for this use case

## R2: Content Area Left/Right Split

**Decision**: Refactor `ContentModel` to compose `PipelineListModel` (left) and a placeholder right pane using `lipgloss.JoinHorizontal`. Left pane is 30% width (min 25, max 50 columns).

**Rationale**: This matches the spec's FR-001 and FR-017. `lipgloss.JoinHorizontal` is the idiomatic approach for side-by-side layout in Bubble Tea. The 30% ratio with min/max bounds follows industry-standard TUI sidebar patterns (k9s, lazygit).

**Alternatives Rejected**:
- Single-column layout — blocks future detail pane work
- Fixed-width sidebar — doesn't adapt to terminal width

## R3: Data Provider Interface Pattern

**Decision**: Create a `PipelineDataProvider` interface following the established `MetadataProvider` pattern from `header_provider.go`. Uses `NewReadOnlyStateStore` for concurrent database access.

**Rationale**: The header bar already uses this exact pattern — an interface with fetch methods, a default implementation wrapping external sources, and mock injection for tests. Reusing the pattern reduces cognitive load and ensures consistency. `NewReadOnlyStateStore` provides WAL-mode read access suitable for polling.

**Alternatives Rejected**:
- Direct state store injection — couples TUI to state package, harder to test
- Channel-based data push — over-engineered for 5-second polling

## R4: Navigation Model (Flat Cursor)

**Decision**: Maintain a flat index into a computed list of navigable items (section headers + pipeline items). The list is recomputed on data refresh, filter change, or section collapse/expand. Cursor clamps at boundaries (no wrapping).

**Rationale**: A flat cursor simplifies keyboard handling — ↑/↓ just increment/decrement the index. The navigable items list acts as a view model over the underlying section data. Section headers are distinguishable by type, enabling different visual treatment and suppression of `PipelineSelectedMsg` when a header is selected.

**Alternatives Rejected**:
- Two-level cursor (section index + item index) — complex cross-section traversal logic
- Separate header/item navigation modes — confusing UX

## R5: Async Polling with tea.Tick

**Decision**: Use `tea.Tick` at 5-second intervals for Running/Finished sections. Available pipelines loaded once at startup via `Init()`.

**Rationale**: The header already uses `tea.Tick` for git refresh (30s). A 5-second interval balances responsiveness for active pipeline monitoring with SQLite read load. Available pipelines rarely change (only on manifest edit), so one-time load is sufficient.

**Alternatives Rejected**:
- File watching for manifest changes — adds complexity, low value for prototype phase
- Shorter polling interval (1s) — unnecessary SQLite load

## R6: Message Forwarding Architecture

**Decision**: `PipelineListModel` emits `PipelineSelectedMsg` (existing type from `header_messages.go`). `ContentModel.Update()` forwards all messages to `PipelineListModel` and returns its commands. `AppModel.Update()` already forwards all messages to the header; it will also forward to content.

**Rationale**: Reusing `PipelineSelectedMsg` leverages the header's existing branch override handler. The unidirectional message flow (list → content → app → header) is the standard Bubble Tea pattern.

**Alternatives Rejected**:
- New message type — unnecessary when existing type has all needed fields
- Direct header↔list communication — breaks Bubble Tea's parent-mediated message model

## R7: Filter Implementation

**Decision**: Embed a `textinput.Model` from `charmbracelet/bubbles` for the filter input. Activated on `/`, dismissed on Escape. Filter applies case-insensitive substring match across all sections simultaneously.

**Rationale**: `bubbles/textinput` handles cursor rendering, key input, and styling. It's already an indirect dependency via `charmbracelet/huh`. Using it avoids reimplementing text input.

**Alternatives Rejected**:
- Custom text input — unnecessary when bubbles provides one
- Regex filter — over-complex for pipeline names; substring match is sufficient
