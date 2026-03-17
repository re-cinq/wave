# Feature Specification: Polish Log Streaming UX

**Feature Branch**: `464-log-streaming-ux`
**Created**: 2026-03-17
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/464
**Parent Issue**: #455
**Dependency**: #461 (run detail view provides step container)

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Auto-Scroll Log Following (Priority: P1)

A user monitoring a running pipeline wants the log viewer to automatically scroll to show new output as it arrives, similar to `tail -f`. When they scroll up to inspect earlier output, auto-scroll pauses so they can read without the view jumping. A "Jump to bottom" button appears so they can resume following.

**Why this priority**: This is the foundational UX for real-time log monitoring. Without auto-scroll, users must manually scroll to see new output, making the streaming infrastructure pointless from a usability perspective.

**Independent Test**: Can be fully tested by starting a pipeline run, observing that new log lines appear at the bottom automatically, scrolling up, confirming auto-scroll pauses, and clicking "Jump to bottom" to resume.

**Acceptance Scenarios**:

1. **Given** a running pipeline step producing log output, **When** the user opens the run detail view and the log viewer is scrolled to the bottom, **Then** new log lines appear and the view scrolls down automatically to keep the latest line visible.
2. **Given** auto-scroll is active, **When** the user scrolls up by any amount, **Then** auto-scroll pauses and a "Jump to bottom" button becomes visible.
3. **Given** auto-scroll is paused, **When** the user clicks "Jump to bottom", **Then** the view scrolls to the latest log line and auto-scroll resumes.
4. **Given** auto-scroll is paused, **When** the user manually scrolls back to the bottom, **Then** auto-scroll resumes and the "Jump to bottom" button disappears.

---

### User Story 2 - Collapsible Log Sections per Step (Priority: P1)

A user viewing a multi-step pipeline run wants to expand and collapse log output for each step independently, similar to GitHub Actions job groups. This lets them focus on the step they care about without being overwhelmed by output from all steps.

**Why this priority**: Equally critical to auto-scroll — without collapsible sections, a multi-step pipeline produces an unmanageable wall of interleaved log output. This is the structural backbone of the log viewer.

**Independent Test**: Can be tested by running a multi-step pipeline and verifying each step has a collapsible header that expands/collapses its log content independently.

**Acceptance Scenarios**:

1. **Given** a pipeline run with multiple steps, **When** the user views the run detail page, **Then** each step's log output is displayed in a collapsible section with the step name, status, and duration visible in the header.
2. **Given** a step section is collapsed, **When** the user clicks the section header, **Then** the log output expands and becomes visible.
3. **Given** a step is currently running, **When** the user views the run detail page, **Then** the running step's section is expanded by default and other completed steps are collapsed.
4. **Given** a step has failed, **When** the user views the run detail page, **Then** the failed step's section is expanded by default.

---

### User Story 3 - Log Line Timestamps and Line Numbers (Priority: P2)

A user debugging a pipeline failure wants each log line to display a timestamp and line number so they can correlate events with other systems and reference specific lines when discussing issues.

**Why this priority**: Essential for debugging but not for basic monitoring. Timestamps and line numbers are metadata overlays that enhance an already-functional log viewer.

**Independent Test**: Can be tested by viewing any pipeline's log output and verifying each line has a visible timestamp and sequential line number in a gutter column.

**Acceptance Scenarios**:

1. **Given** a step is producing log output, **When** the user views the step's log section, **Then** each log line displays a sequential line number in a left gutter and a timestamp.
2. **Given** log output with timestamps, **When** the user views the timestamps, **Then** they are displayed in a consistent, human-readable format (e.g., `HH:MM:SS` or `HH:MM:SS.mmm`).
3. **Given** a step with many log lines, **When** the line numbers column is displayed, **Then** the gutter width accommodates the number of digits without layout shifts.

---

### User Story 4 - ANSI Color Rendering (Priority: P2)

A user viewing log output from tools that produce colored terminal output (test runners, linters, compilers) wants ANSI color codes rendered as styled text rather than displayed as raw escape sequences.

**Why this priority**: Raw ANSI codes make logs unreadable. Rendering them improves readability significantly, but the log viewer is still usable without it (just ugly).

**Independent Test**: Can be tested by running a pipeline step that produces ANSI-colored output and verifying colors appear correctly in the browser.

**Acceptance Scenarios**:

1. **Given** log output containing ANSI color escape sequences, **When** the user views the log, **Then** the text is rendered with the corresponding colors/styles applied via CSS.
2. **Given** log output with unsupported or malformed ANSI codes, **When** the user views the log, **Then** the escape sequences are stripped cleanly rather than displayed as raw characters.
3. **Given** log output with nested or overlapping ANSI codes (e.g., bold + color), **When** rendered, **Then** styles are applied correctly without visual artifacts.

---

### User Story 5 - Search and Filter Within Logs (Priority: P2)

A user looking for a specific error message or keyword in a large log output wants to search and filter log lines without leaving the page. The search operates client-side on already-streamed content.

**Why this priority**: Valuable for large logs but users can use browser Ctrl+F as a basic alternative. This provides a better UX with match highlighting and line-level filtering.

**Independent Test**: Can be tested by opening a completed pipeline's logs, typing a search term, and verifying matching lines are highlighted and/or non-matching lines are filtered out.

**Acceptance Scenarios**:

1. **Given** the log viewer is displaying output, **When** the user types a search term in the search input, **Then** matching lines are highlighted and the view scrolls to the first match.
2. **Given** an active search with multiple matches, **When** the user presses next/previous navigation controls, **Then** the view cycles through matches sequentially.
3. **Given** an active search, **When** the user clears the search input, **Then** all highlighting is removed and the full log is displayed.
4. **Given** log output is still streaming, **When** a new line arrives that matches the search term, **Then** the match count updates and the new match is included in navigation.

---

### User Story 6 - Large Log Performance (Priority: P2)

A user viewing a step that produced 10,000+ log lines wants the log viewer to remain responsive — no freezing, no input lag, no dropped frames during scrolling.

**Why this priority**: Performance is invisible when it works but completely breaks the experience when it doesn't. Large log volumes are common in CI-like pipeline output.

**Independent Test**: Can be tested by loading a run with 10k+ log lines and verifying smooth scrolling, responsive search, and no browser tab crashes.

**Acceptance Scenarios**:

1. **Given** a step with 10,000+ log lines, **When** the user scrolls through the log viewer, **Then** scrolling is smooth with no visible jank or frame drops.
2. **Given** a step producing high-volume output (hundreds of lines per second), **When** the user is viewing the step, **Then** the browser remains responsive and the UI does not freeze.
3. **Given** a large log is loaded, **When** the user uses search, **Then** search results appear within 500ms.

---

### User Story 7 - Log Download and Copy (Priority: P3)

A user wants to export the raw log content for a step — to share with a colleague, attach to a bug report, or analyze with external tools.

**Why this priority**: Nice-to-have convenience feature. Users can work around this by copying from the browser, but a dedicated button provides clean raw output.

**Independent Test**: Can be tested by clicking the download button on any step's log section and verifying the downloaded file contains the raw log content.

**Acceptance Scenarios**:

1. **Given** a step with log output, **When** the user clicks the download/copy button, **Then** the raw log content (without HTML formatting) is available for download as a text file or copied to clipboard.
2. **Given** log output with ANSI codes, **When** the user downloads the raw log, **Then** the file contains the original ANSI codes (not the HTML-rendered version).

---

### User Story 8 - SSE Reconnection Indicator (Priority: P3)

A user whose browser temporarily lost the SSE connection wants a visual indicator that reconnection is in progress, so they know log output may have a gap and is being recovered.

**Why this priority**: The SSE broker already handles reconnection with `Last-Event-ID` backfill. This story adds a thin UX layer on top of existing infrastructure.

**Independent Test**: Can be tested by simulating a network interruption (e.g., toggling airplane mode briefly) and verifying a "Reconnecting..." banner appears and disappears after reconnection.

**Acceptance Scenarios**:

1. **Given** an active SSE connection, **When** the connection drops, **Then** a "Reconnecting..." banner appears at the top of the log viewer.
2. **Given** the reconnection banner is shown, **When** the SSE connection is re-established, **Then** the banner disappears and missed events are backfilled.
3. **Given** the reconnection banner is shown, **When** reconnection fails after multiple retries, **Then** the banner updates to indicate the connection has been lost with a manual retry option.

---

### Edge Cases

- What happens when a step produces zero log output? → The collapsible section displays "No output" placeholder text.
- What happens when the SSE connection is lost permanently (server shutdown)? → The reconnection indicator transitions to a "Connection lost" state with a manual refresh option.
- What happens when log output contains very long lines (1000+ characters)? → Lines soft-wrap within the log viewer using CSS `word-break: break-all` and `white-space: pre-wrap`, keeping line numbers visible in the gutter (see C3).
- What happens when the user opens a completed (non-running) pipeline? → All log output is loaded at once, auto-scroll is disabled, all step sections are collapsed except failed steps.
- What happens when multiple steps run concurrently? → Each step has its own independent collapsible section; log output is not interleaved across sections.
- What happens when a search term matches thousands of lines? → Match count is displayed but only the viewport vicinity is highlighted to maintain performance.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST auto-scroll the log viewer when the user's scroll position is at or near the bottom of the log output.
- **FR-002**: System MUST pause auto-scroll when the user scrolls up and display a "Jump to bottom" button.
- **FR-003**: System MUST render each pipeline step's log output in an independently collapsible section.
- **FR-004**: System MUST expand running and failed step sections by default and collapse completed step sections.
- **FR-005**: System MUST display a sequential line number and timestamp for each log line.
- **FR-006**: System MUST convert ANSI color/style escape sequences to styled HTML for display, and strip any unsupported or malformed sequences.
- **FR-007**: System MUST provide a client-side search input that highlights matching log lines and supports next/previous navigation.
- **FR-008**: System MUST maintain smooth scrolling and responsive UI with 10,000+ log lines per step.
- **FR-009**: System MUST provide a download button that exports raw log content (with original ANSI codes) as a text file.
- **FR-010**: System MUST provide a copy-to-clipboard button for log content.
- **FR-011**: System MUST display a "Reconnecting..." banner when the SSE connection drops and hide it upon successful reconnection.
- **FR-012**: System MUST NOT modify the existing SSE streaming backend or broker logic (client-side changes only).
- **FR-013**: System MUST NOT modify log persistence or storage mechanisms.
- **FR-014**: System MUST redact credentials in displayed log output (existing redaction pipeline applies).

### Key Entities

- **LogLine**: A single line of log output associated with a step. Attributes: line number, timestamp, raw content (including ANSI codes), step ID.
- **LogSection**: A collapsible container for a step's log output. Attributes: step ID, step name, status, expanded/collapsed state, collection of log lines.
- **SearchState**: Client-side search context. Attributes: query string, match indices, current match index, total match count.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: New log lines appear in the viewer within 1 second of being emitted by the SSE stream when auto-scroll is active.
- **SC-002**: The log viewer renders 10,000 log lines with no visible scroll jank (no frame drops below 30fps).
- **SC-003**: Search results across 10,000 lines return within 500ms of the user stopping typing.
- **SC-004**: Users can expand/collapse any step section in under 100ms (no layout recalculation lag).
- **SC-005**: ANSI color codes from standard 16-color and 256-color palettes are rendered correctly with zero raw escape sequences visible.
- **SC-006**: SSE reconnection banner appears within 3 seconds of connection loss and disappears within 1 second of reconnection.
- **SC-007**: All log viewer features work correctly in both light and dark themes (existing theme toggle).

## Clarifications

The following ambiguities were identified during spec review and resolved based on codebase analysis.

### C1: What constitutes "log output" in the web UI?

**Ambiguity**: The spec refers to "log output" and "log lines" throughout, but the current SSE infrastructure streams structured `stream_activity` events (tool name + target) and lifecycle events — not raw subprocess stdout/stderr lines. The `EventSource` in `sse.js` listens for `started`, `running`, `completed`, `failed`, `step_progress`, `stream_activity`, and `eta_updated` event types. There is no `log_line` event type.

**Resolution**: "Log output" means `stream_activity` events rendered as structured log entries. Each `stream_activity` SSE event (containing `tool_name`, `tool_target`, and `message` fields from `event.Event`) is treated as one log line. This aligns with the existing adapter stream parsing in `internal/adapter/claude.go` which emits `StreamEvent` records for each tool use. Raw subprocess stdout/stderr is not surfaced — the spec scope is the `stream_activity` event stream that already flows through the SSE broker.

**Rationale**: FR-012 mandates no backend changes. The `stream_activity` events already carry per-step tool activity data through the SSE broker. Surfacing raw stdout would require new adapter plumbing and backend changes, violating the client-side-only constraint.

### C2: Timestamp source for log lines

**Ambiguity**: FR-005 requires "a timestamp for each log line" but doesn't specify whether timestamps come from the SSE event's `timestamp` field (set by the emitter at event creation time) or should be generated client-side at event receipt time.

**Resolution**: Use the `timestamp` field from each SSE event's JSON payload (`event.Event.Timestamp`). This is set server-side when the event is emitted and represents when the tool activity actually occurred. Display format: `HH:MM:SS.mmm` (millisecond precision) to allow correlation with adapter logs.

**Rationale**: Server-side timestamps are more accurate than client receipt time (which includes network latency and batching delays). The `event.Event` struct already includes a `Timestamp time.Time` field populated at emit time.

### C3: Long line handling strategy

**Ambiguity**: The edge case section says lines 1000+ characters should "wrap within the log viewer or are horizontally scrollable" — both options are listed without a decision.

**Resolution**: Long lines soft-wrap within the log viewer container using CSS `word-break: break-all` and `white-space: pre-wrap`. No horizontal scrollbar within individual log sections. This preserves the ability to visually scan line numbers in the gutter without horizontal scrolling hiding them.

**Rationale**: Horizontal scrolling in nested containers (log section inside step card inside page) creates confusing UX with multiple scroll axes. Soft-wrapping keeps line numbers visible and is the standard approach in CI log viewers (GitHub Actions, GitLab CI).

### C4: Search behavior — highlight vs filter

**Ambiguity**: User Story 5 intro says "search and filter log lines" and mentions "line-level filtering", but the acceptance scenarios only describe highlighting and match navigation (scroll to match, next/previous). These are different UX patterns: highlight keeps all lines visible with matches emphasized; filter hides non-matching lines.

**Resolution**: Search uses **highlight-only** mode — all log lines remain visible, matching lines receive a highlight background, and next/previous navigation scrolls through matches. No line-level filtering (hiding non-matching lines). A future iteration could add a filter toggle.

**Rationale**: Highlight-only preserves log context around matches, which is critical for debugging. Filtering removes surrounding context that helps users understand the sequence of events leading to an error. Highlight-only is also simpler to implement with virtual scrolling since it doesn't change the DOM structure.

### C5: Performance strategy for 10,000+ log lines

**Ambiguity**: FR-008 and SC-002 require smooth performance at 10k+ lines but don't specify the implementation technique. Virtual scrolling (rendering only visible rows), chunked DOM insertion, and CSS containment are all viable approaches with different trade-off profiles.

**Resolution**: Use **CSS containment** (`contain: content` on each log line element) combined with **chunked DOM insertion** (batch incoming events in 100ms frames using `requestAnimationFrame`). Do NOT use virtual scrolling in v1 — it adds significant complexity for search highlighting, copy-to-clipboard, and ANSI rendering, and 10k DOM nodes with CSS containment is well within modern browser capabilities. If profiling reveals issues, virtual scrolling can be added as an optimization in a subsequent iteration.

**Rationale**: CSS containment tells the browser each log line's layout is independent, enabling paint-only updates without full reflow. Chunked insertion prevents long-running JS tasks from blocking the main thread. This approach keeps implementation simple while meeting the 30fps scroll target for 10k lines. GitHub Actions and Jenkins both use full DOM rendering for similar log volumes.
