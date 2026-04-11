# Feature Specification: Expandable Running Pipelines Section

**Feature Branch**: `772-webui-running-pipelines`  
**Created**: 2026-04-11  
**Status**: Draft  
**Input**: User description: "feat(webui): add expandable running-pipelines section to runs overview — expandable section sits below filter/search bar, expanded by default on first load, shows placeholder CTA when no pipelines running, clicking completed/failed runs navigates to run detail"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - View Active Pipelines at a Glance (Priority: P1)

An operator opens the runs overview page and immediately sees all currently running pipelines in a dedicated section at the top, without having to scroll through completed or failed runs. This gives instant situational awareness of live work.

**Why this priority**: The most critical value proposition is surfacing active work instantly. Operators need to know what is happening right now without filtering or scrolling.

**Independent Test**: Load the runs overview page while at least one pipeline is running. The running pipelines section renders at the top, below the filter bar, with all active runs listed. Delivers value as a standalone feature: users can monitor active pipelines independently of any other story.

**Acceptance Scenarios**:

1. **Given** at least one pipeline has status `running`, **When** the user navigates to the runs overview (`/runs`), **Then** the running-pipelines section is visible below the filter/search bar and displays all pipelines with status `running`.
2. **Given** the runs overview page has loaded, **When** the running-pipelines section is present, **Then** it is expanded by default (i.e., the run cards are visible without any user interaction).
3. **Given** multiple pipelines are running, **When** the section renders, **Then** each running pipeline is shown as a card consistent with the existing run card layout (pipeline name, status badge, progress, duration).

---

### User Story 2 - Collapse/Expand Running Pipelines Section (Priority: P2)

An operator who already knows what is running wants to focus on completed or failed runs below. They can collapse the running-pipelines section to reclaim vertical space and expand it again when needed.

**Why this priority**: Secondary to visibility — once you have the section, the ability to manage screen real estate is important but not blocking core value.

**Independent Test**: With the running-pipelines section visible and expanded, click the section header toggle. The section collapses (run cards hidden, only header visible). Click again to expand. Works independently of navigation or CTA stories.

**Acceptance Scenarios**:

1. **Given** the running-pipelines section is expanded, **When** the user clicks the section header/toggle control, **Then** the section collapses and the run cards are hidden.
2. **Given** the running-pipelines section is collapsed, **When** the user clicks the section header/toggle control, **Then** the section expands and the run cards become visible.
3. **Given** the user has collapsed the section and refreshes the page, **Then** the section is expanded by default again (collapse state is NOT persisted across page loads — first-load default is always expanded).

---

### User Story 3 - Empty State: No Running Pipelines (Priority: P3)

An operator opens the runs overview when no pipelines are currently running. The running-pipelines section still appears but shows a clear placeholder that communicates the idle state and provides a call-to-action (CTA) to start a pipeline.

**Why this priority**: Required for completeness and avoids a confusing blank section, but the empty state delivers less operational value than the populated section.

**Independent Test**: Load the runs overview when zero pipelines have status `running`. The section renders with an empty-state placeholder message and a CTA element (e.g., a link or button to trigger a pipeline). Can be tested independently by clearing all active runs.

**Acceptance Scenarios**:

1. **Given** no pipeline has status `running`, **When** the user loads the runs overview, **Then** the running-pipelines section is visible and expanded, displaying an empty-state placeholder (not an empty card list).
2. **Given** the empty-state placeholder is shown, **When** the user views it, **Then** a CTA is present that links to a path where the user can initiate a new pipeline run (e.g., the pipelines list or a trigger endpoint).
3. **Given** pipelines transition from running to completed during the session, **When** the last running pipeline finishes, **Then** the section updates to show the empty-state placeholder on the next page reload (no real-time SSE update in v1 — the existing SSE infrastructure targets single run detail pages only, not the runs overview list).

---

### User Story 4 - Navigate to Run Detail from Running Section (Priority: P2)

An operator sees a completed or failed run card (which may briefly appear in the running section during transition) and clicks it to navigate to the run detail page for investigation.

**Why this priority**: Navigation to detail is fundamental to the workflow — operators need to inspect results. Ranked P2 alongside collapse because detail navigation is equally important to usability.

**Independent Test**: Click any run card within the running-pipelines section. Browser navigates to `/runs/{runID}`. Works independently — verifiable with any run card that has a valid run ID.

**Acceptance Scenarios**:

1. **Given** the running-pipelines section shows run cards, **When** the user clicks a run card, **Then** the browser navigates to `/runs/{runID}` (the run detail page for that run).
2. **Given** a run card in the section has status `completed` or `failed` (e.g., visible during a status transition), **When** the user clicks it, **Then** navigation to `/runs/{runID}` succeeds and shows the completed/failed run detail.
3. **Given** the run detail page is loaded, **When** the user navigates back, **Then** they return to the runs overview with the running-pipelines section in its default expanded state.

---

### Edge Cases

- What happens when a running pipeline completes while the page is open and no SSE update mechanism refreshes the section? The stale card remains until the user refreshes (acceptable baseline; real-time behavior is a separate concern).
- How does the system handle a very large number of simultaneously running pipelines (e.g., 50+)? The section shows all running pipelines without artificial truncation, relying on normal scroll. No "see all running" link is needed in v1 — Wave is a single-operator tool with a low expected concurrency ceiling, so unbounded display is acceptable and consistent with how the main runs list works.
- What happens if the running-pipelines section and the main run list below it both show the same runs? Running runs appear in the dedicated section only; the main list below either excludes running status by default or shows all. The spec requires the section to be additive — it surfaces running runs at the top without removing them from the main list unless the design explicitly filters them out.
- What happens when the user applies a pipeline-name filter while running pipelines are displayed? The running-pipelines section should reflect the filter (only show running runs matching the selected pipeline).
- What happens on mobile/narrow viewports? The section must remain usable; card layout adapts consistently with the existing run card responsive behavior.

## Clarifications

### CL-001: Real-time update behavior (resolved)

**Question**: Should the running-pipelines section update in real-time via SSE when the last running pipeline finishes, or only on page reload?

**Resolution**: Page reload only (v1 baseline). The existing SSE infrastructure (`/api/runs/{id}/events`) is scoped to the single run detail page and does not broadcast list-level events to the runs overview. Adding a global "run status changed" SSE topic to the runs overview is out of scope for this feature. The spec edge-case note is updated to reflect this: stale cards persist until the user reloads, which is acceptable baseline behavior.

**Rationale**: Consistent with how the main runs list works today — it is a server-rendered page with no live updates.

---

### CL-002: Maximum count display for 50+ concurrent runs (resolved)

**Question**: Should a maximum count be shown with a "see all running" link when many pipelines run concurrently (e.g., 50+)?

**Resolution**: No truncation or "see all" link. Show all running pipelines with standard scroll. Wave is a single-operator orchestration tool — the realistic ceiling for simultaneous runs is low. Matching the unbounded behavior of the existing main runs list avoids added complexity for a niche scenario.

**Rationale**: Consistent with the main list's unbounded display. The header count badge (FR-009) already communicates total volume at a glance.

---

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The runs overview page MUST display a dedicated "Running Pipelines" section positioned below the filter/search bar and above the main runs list.
- **FR-002**: The running-pipelines section MUST be expanded by default on every page load (no persisted collapse state).
- **FR-003**: The running-pipelines section MUST include a toggle control in its header that collapses and expands the section contents.
- **FR-004**: When at least one pipeline has status `running`, the section MUST display those pipeline run cards using the same run card visual pattern as the main runs list.
- **FR-005**: When no pipelines have status `running`, the section MUST display an empty-state placeholder with a CTA that directs the user toward initiating a new pipeline run.
- **FR-006**: Each run card in the running-pipelines section MUST be a navigable link to `/runs/{runID}`.
- **FR-007**: Clicking a run card with status `completed` or `failed` MUST navigate to the run detail page for that run.
- **FR-008**: The running-pipelines section MUST respect the active pipeline-name filter applied in the filter/search bar (only show running runs matching the current filter).
- **FR-009**: The section header MUST display the label "Running" (or equivalent) and a count of currently running pipelines.
- **FR-010**: The section MUST be accessible: the toggle control MUST have appropriate ARIA attributes (`aria-expanded`, `aria-controls`) and be keyboard-operable.

### Key Entities _(include if feature involves data)_

- **RunSummary (running subset)**: A pipeline run with `Status = "running"`. Key attributes: `RunID`, `PipelineName`, `Status`, `Progress` (%), `StepsCompleted`, `StepsTotal`, `Duration`, `FormattedStartedAt`. Relationship: same entity shown in main runs list; the section is a filtered view, not a separate data type.
- **Running Pipelines Section**: A UI container on the runs overview. Attributes: expanded/collapsed state (transient, not persisted), filtered run set, empty-state vs. populated state. Relationship: appears between filter bar and main `RunSummary` list.
- **Empty State / CTA**: Displayed when the running subset is empty. Attributes: placeholder message text, CTA link destination (pipeline list or run trigger endpoint).

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: The running-pipelines section is visible on the runs overview page on 100% of page loads, regardless of how many pipelines are running.
- **SC-002**: The section is in expanded state on initial page load 100% of the time (zero cases where it loads collapsed).
- **SC-003**: When pipelines are running, 100% of active run cards in the section link correctly to their respective run detail pages (zero broken navigation links).
- **SC-004**: The empty-state placeholder renders correctly (CTA present, no blank/empty section) when zero pipelines are running.
- **SC-005**: The toggle control passes automated accessibility checks: `aria-expanded` reflects actual expanded/collapsed state, and the control is reachable and operable via keyboard navigation.
- **SC-006**: Applying a pipeline-name filter reduces the running-pipelines section to show only matching active runs (filter applies to both the section and the main list simultaneously).
