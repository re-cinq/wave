# Feature Specification: TUI Pipeline Detail Right Pane

**Feature Branch**: `255-tui-pipeline-detail`  
**Created**: 2026-03-06  
**Status**: Draft  
**Issue**: [#255](https://github.com/re-cinq/wave/issues/255) (part 4 of 10, parent: [#251](https://github.com/re-cinq/wave/issues/251))  
**Input**: Implement the right pane of the Pipelines view showing detail views for the currently selected pipeline. Available pipelines show configuration metadata; finished pipelines show execution summary with step results and action hints. Focus management via Enter/Esc between left and right panes.

## Clarifications

The following ambiguities were identified and resolved during specification refinement:

### C1: Detail pane content lifecycle — cursor-move preview vs Enter-only

**Ambiguity**: User stories say "presses Enter" to display detail, but FR-012 says "when a new pipeline is selected (cursor moves to a different item), the right pane scroll position MUST reset" — implying the right pane already has content on cursor movement. It was unclear whether the detail pane renders on cursor movement or only on Enter.

**Resolution**: The right pane renders a **preview** of the selected pipeline's detail content on cursor movement (following the existing `PipelineSelectedMsg` pattern which fires on every cursor move). Pressing Enter additionally **focuses** the right pane for keyboard interaction (scrolling). This matches the ranger/IDE split-pane pattern where the preview is always visible and Enter shifts keyboard focus. The 70% right pane width would be wasted showing only a placeholder until Enter is pressed.

### C2: Available pipeline identity in `PipelineSelectedMsg`

**Ambiguity**: The existing `PipelineSelectedMsg` sends `{RunID: "", BranchName: ""}` when an available pipeline is selected (see `pipeline_list.go:313`). The detail pane has no way to identify *which* available pipeline was selected, since neither the pipeline name nor any index is included.

**Resolution**: Extend `PipelineSelectedMsg` to include a `Name string` field carrying the pipeline name. For available pipelines, `RunID` remains empty and `Name` is set to the pipeline name. For finished/running pipelines, both `RunID` and `Name` are populated. Additionally, add a `Kind itemKind` field so the detail pane can distinguish available/finished/running selections without inferring from empty/non-empty fields. This is a minimal extension to an existing message type, avoiding a new message.

### C3: Running pipeline detail view

**Ambiguity**: The spec covers Available and Finished detail views but does not specify what the right pane shows when a Running pipeline is selected in the left pane. Running pipelines exist in the list (section 0) and emit `PipelineSelectedMsg` on cursor movement.

**Resolution**: Running pipelines are **out of scope** for this issue. When a running pipeline is selected, the right pane shows a brief informational message: the pipeline name, "Running" status with an elapsed time indicator, and a note that detailed progress monitoring is planned for a future issue (#258 — Real-time progress). This avoids blocking on real-time data while still providing useful context. Enter on a running pipeline item does NOT transfer focus to the right pane (there is no scrollable content to interact with).

### C4: Zero artifacts display

**Ambiguity**: The edge case states "The 'Artifacts' section either shows 'No artifacts' or is omitted entirely" — the spec doesn't commit to one approach.

**Resolution**: When a finished pipeline has zero artifacts, the "Artifacts" section header is still displayed with a "No artifacts produced" message beneath it. This is preferred over omission because: (a) it confirms the system checked for artifacts rather than silently failing, (b) it maintains consistent section layout across all finished pipeline details, and (c) it matches how empty state is handled in the left pane (section headers always appear even when empty).

### C5: Status bar dynamic hint mechanism

**Ambiguity**: FR-015 requires the status bar to update its key hints based on the current focus context, but the existing `StatusBarModel` renders static hints (`"↑↓: navigate  /: filter  q: quit  ctrl+c: exit"`) with no mechanism for dynamic updates.

**Resolution**: Introduce a `FocusChangedMsg` message type carrying the current focus context (left pane or right pane). The `StatusBarModel` receives this message and switches its rendered hints accordingly. When the left pane is focused: `"↑↓: navigate  Enter: view  /: filter  q: quit"`. When the right pane is focused: `"↑↓: scroll  Esc: back  q: quit"`. `ContentModel` emits `FocusChangedMsg` whenever focus changes. This follows the existing message-passing pattern used by `RunningCountMsg` and `PipelineSelectedMsg`.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - View Available Pipeline Details (Priority: P1)

A developer navigates to an available pipeline in the left pane. As the cursor moves to the pipeline item, the right pane immediately renders a preview of the pipeline's static configuration: its description, the list of steps (with persona assignments), required inputs with an example, output artifacts, and any tool/skill dependencies. The developer can press Enter to focus the right pane and scroll through the detail content. This gives the developer enough context to decide whether to run the pipeline without needing to open the YAML file.

**Why this priority**: The right pane is currently a placeholder ("Select a pipeline to view details"). Rendering available pipeline details is the foundational content that makes the two-pane layout useful. It's the simplest detail view (no runtime data needed) and the most common first interaction — users browse available pipelines before running them.

**Independent Test**: Can be tested by selecting an available pipeline in the list and verifying the right pane renders the expected metadata fields: description, steps list, inputs, outputs, and dependencies.

**Acceptance Scenarios**:

1. **Given** the left pane lists available pipelines, **When** the user moves the cursor to one, **Then** the right pane displays a preview of the pipeline's detail view with: name, description, steps list (showing step ID and persona), input requirements, output artifacts, and dependencies.
2. **Given** the right pane is showing an available pipeline detail, **When** the pipeline has 6 steps, **Then** each step is listed with its ID and associated persona name.
3. **Given** the right pane is showing an available pipeline detail, **When** the pipeline has input requirements (e.g., `source: github_issue_url`), **Then** the input source and example are displayed.
4. **Given** the right pane is showing an available pipeline detail, **When** the pipeline defines output artifacts across its steps, **Then** the artifact names are listed.
5. **Given** the right pane is showing an available pipeline detail, **When** the pipeline has skill or tool dependencies in its `requires` block, **Then** those dependencies are displayed.

---

### User Story 2 - View Finished Pipeline Summary (Priority: P1)

A developer navigates to a finished pipeline in the left pane. As the cursor moves to the pipeline item, the right pane immediately renders the execution summary: final status (completed/failed/cancelled), total duration, the git branch used, start and end timestamps, and a step-by-step results table showing each step's status and individual timing. Below the summary, the produced artifacts are listed. The developer can press Enter to focus the right pane and scroll, revealing action hints at the bottom: `[Enter] Open chat`, `[b] Checkout branch`, `[d] View diff`.

**Why this priority**: Equally critical to US-1. Users need to inspect completed pipeline runs to understand what happened — whether it succeeded, how long each step took, and what artifacts were produced. Without this, users must fall back to `wave status` CLI commands or manually browse the filesystem.

**Independent Test**: Can be tested by selecting a finished pipeline (completed or failed) in the list and verifying the right pane renders status, duration, branch, step results with individual timings, artifacts, and action hints.

**Acceptance Scenarios**:

1. **Given** the left pane lists a completed pipeline run, **When** the user moves the cursor to it, **Then** the right pane displays: pipeline name, "Completed" status with success indicator, total duration, branch name, start time, and end time.
2. **Given** the right pane is showing a finished pipeline with 6 steps, **When** the detail renders, **Then** each step is listed with its individual status (success/failure indicator) and execution duration.
3. **Given** the right pane is showing a finished pipeline, **When** the pipeline produced artifacts, **Then** the artifact names and paths are listed in an "Artifacts" section.
4. **Given** the right pane is showing a failed pipeline, **When** the pipeline has an error message, **Then** the error is displayed in the summary along with which step failed.
5. **Given** the right pane is showing a finished pipeline, **When** the user focuses the right pane and scrolls to the bottom, **Then** action hints are displayed: `[Enter] Open chat`, `[b] Checkout branch`, `[d] View diff`, `[Esc] Back`.

---

### User Story 3 - Navigate Between Left and Right Panes (Priority: P1)

A developer uses Enter to move focus from the left pane to the right pane, and Esc to return. When the right pane is focused, the left pane's selection indicator dims or de-emphasizes, and the right pane gains a visual focus indicator. Arrow keys in the focused right pane scroll the detail content instead of moving the left pane's selection. Pressing Esc returns focus to the left pane, restoring its full highlight and keeping the previously selected item. Enter on a running pipeline item does NOT transfer focus (see C3).

**Why this priority**: Focus management is essential for the two-pane interaction model to work. Without it, users cannot distinguish which pane accepts input, leading to confusion about whether arrow keys navigate the list or scroll the detail.

**Independent Test**: Can be tested by pressing Enter to focus the right pane, verifying focus indicator changes, pressing arrow keys to confirm they scroll detail (not the list), and pressing Esc to return focus to the left pane.

**Acceptance Scenarios**:

1. **Given** the left pane is focused and a pipeline item (available or finished) is selected, **When** the user presses Enter, **Then** focus moves to the right pane — the left pane's selection dims, and the right pane gains a visual focus indicator (e.g., a highlighted border or title).
2. **Given** the right pane is focused, **When** the user presses Esc, **Then** focus returns to the left pane — the left pane's selection restores full highlight, and the right pane's focus indicator is removed.
3. **Given** the right pane is focused with detail content, **When** the user presses ↑/↓ arrow keys, **Then** the detail content scrolls vertically (not the left pane selection).
4. **Given** the right pane is focused, **When** the user presses any left pane navigation key (↑/↓), **Then** only the right pane scrolls — the left pane selection does not change.
5. **Given** the left pane is focused and the cursor is on a section header, **When** the user presses Enter, **Then** the section collapses/expands (existing behavior) — Enter on a section header does NOT focus the right pane.
6. **Given** the left pane is focused and the cursor is on a running pipeline item, **When** the user presses Enter, **Then** focus does NOT move to the right pane (running detail is informational only, no scrollable content).

---

### User Story 4 - Scroll Long Detail Content (Priority: P2)

A developer views a pipeline detail with more content than fits in the visible area (e.g., many steps, many artifacts, long description). The content is scrollable using ↑/↓ keys when the right pane is focused. A scroll position indicator shows the user where they are in the content. The viewport starts at the top when a new pipeline is selected.

**Why this priority**: Scrolling becomes necessary for real-world pipelines with many steps (8+ steps, multiple artifacts). Without it, the bottom of the detail view is unreachable. However, most common pipelines (4-6 steps) may fit without scrolling, so this is secondary to basic rendering.

**Independent Test**: Can be tested by selecting a pipeline with enough detail content to exceed the visible height, pressing Enter to focus the right pane, and using ↑/↓ to scroll through all content.

**Acceptance Scenarios**:

1. **Given** the right pane is focused and detail content exceeds the visible height, **When** the user presses ↓, **Then** the content scrolls down by one line.
2. **Given** the right pane is scrolled down, **When** the user presses ↑, **Then** the content scrolls up by one line.
3. **Given** the user selects a different pipeline from the left pane, **When** the right pane updates to the new detail, **Then** the scroll position resets to the top.
4. **Given** the content is scrolled to the bottom, **When** the user presses ↓, **Then** the scroll does not go past the end of the content.

---

### User Story 5 - Placeholder State When No Pipeline Selected (Priority: P2)

When the TUI first launches or when no pipeline item is selected (e.g., the cursor is on a section header), the right pane shows a centered placeholder message indicating that the user should select a pipeline. This replaces the current static "Select a pipeline to view details" text and persists the same behavior.

**Why this priority**: The placeholder is important for user orientation but is functionally already present in the current codebase. This story ensures the placeholder behavior is maintained and integrated with the new focus model.

**Independent Test**: Can be tested by launching the TUI and verifying the placeholder appears before any selection, and by navigating to a section header and verifying the placeholder re-appears.

**Acceptance Scenarios**:

1. **Given** the TUI has just launched, **When** no pipeline item has been selected yet (cursor is on a section header), **Then** the right pane displays a centered placeholder message.
2. **Given** the user navigates from a pipeline item to a section header, **When** the right pane updates, **Then** the placeholder message is shown instead of a detail view.
3. **Given** the right pane is showing a placeholder, **When** the user presses Enter on a section header, **Then** the section collapses — the right pane remains as placeholder.

---

### Edge Cases

- What happens when a finished pipeline's branch has been deleted? The branch name still displays (it's stored in the run record), but the `[b] Checkout branch` hint shows as disabled or displays "(deleted)" next to the branch name, matching the existing `BranchDeleted` field in `PipelineSelectedMsg`.
- What happens when a finished pipeline has zero artifacts? The "Artifacts" section header is still displayed with a "No artifacts produced" message beneath it (see C4).
- What happens when a finished pipeline has a very long error message? The error message is truncated with ellipsis or wraps within the available width, not breaking the layout.
- What happens when the terminal is resized while the detail is visible? The detail re-renders at the new dimensions, preserving scroll position where possible.
- What happens when the right pane width is very narrow (e.g., 30 columns on an 80-column terminal)? Content wraps or truncates gracefully — long step names and artifact paths are truncated with ellipsis.
- What happens when a finished pipeline has steps that were skipped? Skipped steps show a distinct indicator (e.g., "—" or "skip") and no duration.
- What happens when the state database returns an error while loading detail data? An error message is shown in the right pane (e.g., "Failed to load pipeline details") rather than crashing.
- What happens when a running pipeline is selected? The right pane shows a brief informational message with the pipeline name, "Running" status, and elapsed time. Enter does NOT focus the right pane (see C3).

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST render a right pane adjacent to the existing left pipeline list pane. The right pane occupies the remaining width after the left pane (currently 70% of content area, minimum ~30 columns at 80-column terminal).
- **FR-002**: System MUST display a centered placeholder message in the right pane when no pipeline item is selected (e.g., cursor is on a section header, or TUI just launched).
- **FR-003**: When an **available** pipeline is selected (cursor moves to it), the right pane MUST display: pipeline name, description, category (if set), step count, a list of steps showing step ID and persona, input source and example, output artifact names, and dependency requirements (skills, tools).
- **FR-004**: When a **finished** pipeline is selected (cursor moves to it), the right pane MUST display: pipeline name, final status (completed/failed/cancelled) with a visual indicator (✓/✗), total duration, git branch name, start time, end time, and a step results table showing each step's status and individual duration.
- **FR-005**: The finished pipeline detail MUST display an "Artifacts" section listing the name, path, and type of each artifact produced during the run. When zero artifacts exist, the section header MUST still display with a "No artifacts produced" message (see C4).
- **FR-006**: The finished pipeline detail MUST display action hints at the bottom: `[Enter] Open chat`, `[b] Checkout branch`, `[d] View diff`, `[Esc] Back`. These are display-only hints for this issue — the actions themselves are implemented in separate issues.
- **FR-007**: When a finished pipeline's branch has been deleted (`BranchDeleted` is true in `PipelineSelectedMsg`), the branch display MUST indicate this (e.g., appending "(deleted)" or using strikethrough styling), and the `[b] Checkout branch` hint MUST appear disabled.
- **FR-008**: System MUST support two-pane focus management: Enter on a pipeline item (available or finished, NOT running or section header) in the left pane moves focus to the right pane; Esc from the right pane returns focus to the left pane.
- **FR-009**: When the right pane is focused, ↑/↓ keys MUST scroll the detail content vertically. When the left pane is focused, ↑/↓ keys MUST navigate the pipeline list (existing behavior unchanged).
- **FR-010**: The left pane MUST visually indicate when it has lost focus (e.g., dimmed selection highlight). The right pane MUST visually indicate when it has gained focus (e.g., border color change or title highlight).
- **FR-011**: Enter on a section header in the left pane MUST continue to collapse/expand the section (existing behavior) — it MUST NOT move focus to the right pane.
- **FR-012**: When a new pipeline is selected (cursor moves to a different item), the right pane scroll position MUST reset to the top and the detail content MUST update immediately (preview on cursor move, see C1).
- **FR-013**: The right pane MUST handle content that exceeds the visible height by enabling scrolling when focused. Scroll MUST clamp at top and bottom boundaries (no over-scroll).
- **FR-014**: The detail view for a failed pipeline MUST display the error message from the run record, in addition to identifying which step failed.
- **FR-015**: System MUST update the status bar key hints to reflect the current focus context via a `FocusChangedMsg` (see C5). When the right pane is focused, hints MUST include scrolling keys and Esc. When the left pane is focused, hints MUST include navigation, Enter, and filter.
- **FR-016**: Data for finished pipeline details (step results, artifacts, metrics) MUST be fetched asynchronously via a provider interface, following the established `PipelineDataProvider` / `MetadataProvider` pattern. Loading state SHOULD show a brief indicator (e.g., "Loading...") while data is being fetched.
- **FR-017**: The right pane MUST re-render when the terminal is resized, adapting content layout to the new dimensions.
- **FR-018**: System MUST respect `NO_COLOR` environment variable — all styling in the detail pane degrades to plain text formatting.
- **FR-019**: `PipelineSelectedMsg` MUST be extended with `Name string` and `Kind itemKind` fields to carry pipeline identity for all item types (see C2). Available pipeline selections MUST populate `Name` with the pipeline name.
- **FR-020**: When a **running** pipeline is selected (cursor moves to it), the right pane MUST display a brief informational message showing the pipeline name, "Running" status, and elapsed time. Enter on a running pipeline item MUST NOT transfer focus to the right pane (see C3).

### Key Entities

- **PipelineDetailModel**: The Bubble Tea model for the right pane, implementing `Init()`, `Update()`, `View()`. Owns the current detail state (which pipeline is selected, detail data, scroll position, focus state).
- **AvailableDetail**: Data projection for rendering an available pipeline's configuration. Contains name, description, category, steps (ID + persona), input config, artifact names, and dependencies. Derived from parsing the pipeline YAML via the existing `Pipeline` type.
- **FinishedDetail**: Data projection for rendering a finished pipeline's execution summary. Contains run ID, status, duration, branch, timestamps, step results (step ID + status + duration), artifacts (name + path + type), and error message. Derived from state store queries (run record, events, artifacts, performance metrics).
- **FocusState**: Tracks which pane currently has keyboard focus (left list or right detail). Determines key routing and visual styling. Owned by `ContentModel` as the parent of both panes. `ContentModel` propagates focus changes to child models via `SetFocused(bool)` method calls, following the existing `SetSize` pattern.
- **FocusChangedMsg**: Message emitted by `ContentModel` when focus changes between panes. Consumed by `StatusBarModel` to update key hints (see C5). Carries the current focus context (left pane or right pane).
- **DetailDataProvider**: Interface for fetching detailed pipeline data beyond what `PipelineDataProvider` currently supplies. Methods for fetching run details (events, artifacts, step metrics) and available pipeline configuration (full YAML parse). Enables mock injection for testing. Separate from `PipelineDataProvider` to maintain single responsibility — list-level data vs detail-level data.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Selecting an available pipeline (cursor move) renders a detail preview with all specified fields (name, description, steps, inputs, artifacts, dependencies) within the right pane — verified by unit tests with mock data.
- **SC-002**: Selecting a finished pipeline (cursor move) renders an execution summary with status, duration, branch, step results, artifacts, and action hints — verified by unit tests with mock state store data.
- **SC-003**: Focus transitions between left and right panes via Enter/Esc are immediate (within one render cycle) and produce correct visual focus indicators on both panes — verified by tests simulating key sequences.
- **SC-004**: Arrow key scrolling in the focused right pane traverses all detail content without affecting the left pane selection — verified by tests with oversized content.
- **SC-005**: All existing tests (`go test ./internal/tui/...`) continue to pass after integration — the new right pane does not break the left pane, header, or status bar components.
- **SC-006**: The detail pane renders without visual artifacts at terminal widths from 80 to 300 columns and heights from 24 to 100 rows — verified by rendering tests at boundary dimensions.
- **SC-007**: Failed pipeline details correctly display error messages and identify the failing step — verified by unit tests with failed run mock data.
