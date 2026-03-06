# Feature Specification: TUI Live Output Streaming

**Feature Branch**: `257-tui-live-output`  
**Created**: 2026-03-06  
**Status**: Draft  
**Issue**: [#257](https://github.com/re-cinq/wave/issues/257) (part 6 of 10, parent: [#251](https://github.com/re-cinq/wave/issues/251))  
**Input**: Connect the pipeline executor's event system (`internal/event/`) to the TUI right pane so that running pipelines display real-time progress output. When a running pipeline is selected, the right pane shows live verbose output with toggleable display flags. Auto-scroll follows new output. Manual scrolling pauses auto-scroll. When a pipeline completes, the view transitions to the finished detail.

## Clarifications

The following ambiguities were identified and resolved during specification refinement:

### C1: Event delivery mechanism — channel-based subscription vs callback injection

**Ambiguity**: The executor emits events via `EventEmitter.Emit()` which writes NDJSON to stdout and optionally calls a `ProgressEmitter` callback. The TUI runs a Bubble Tea event loop that processes `tea.Msg` types. The issue says "connect event system to TUI" but doesn't specify the bridging mechanism — channel-based pub/sub, callback-to-message adapter, or polling.

**Resolution**: Use a callback-to-message adapter. When `PipelineLauncher.Launch()` constructs the executor, it wraps the emitter with a TUI-aware callback that converts each `event.Event` into a `PipelineEventMsg` and sends it to the Bubble Tea program via `program.Send()`. This is the same non-blocking pattern used by `display.SendUpdate()` in the existing progress display. The `tea.Program` reference is stored on the `PipelineLauncher` at initialization (set via `SetProgram(*tea.Program)` after `tea.NewProgram()` returns). No channel or subscription mechanism is needed — the executor already has the `ProgressEmitter` callback interface, and `program.Send()` is thread-safe.

### C2: Event buffer management — unbounded growth vs fixed-size ring buffer

**Ambiguity**: A pipeline run can emit thousands of events (1 Hz heartbeat + per-tool-call stream_activity). Storing all events for viewport rendering would consume unbounded memory. The spec needs to define retention behavior.

**Resolution**: Use a fixed-size ring buffer of rendered output lines (not raw events). Each incoming event is formatted into one or more display lines and appended to the buffer. The buffer capacity is 1000 lines — sufficient for scrollback (a typical pipeline produces ~200-500 visible events) without unbounded growth. When the buffer exceeds capacity, the oldest lines are dropped. The viewport renders from this buffer. This matches how terminal emulators handle scrollback — users see the most recent output and can scroll up through history.

### C3: Auto-scroll vs manual scroll behavior

**Ambiguity**: The issue says "auto-scroll follows new output by default" and "manual scroll pauses auto-scroll" but doesn't specify how auto-scroll resumes. Does the user need to press a key to re-engage auto-scroll, or does scrolling to the bottom automatically re-engage it?

**Resolution**: Auto-scroll is a boolean flag, initially `true`. When the user presses ↑/↓ or Page Up/Page Down while the right pane is focused, auto-scroll is set to `false`. Auto-scroll re-engages when the user scrolls to the bottom of the buffer (the viewport is at the last line). This matches the behavior of terminal emulators and chat applications (scroll up to read history, scroll to bottom to resume following). Additionally, a visual indicator is shown when auto-scroll is paused (e.g., "⏸ Scrolling paused — scroll to bottom to resume" at the bottom of the viewport).

### C4: Display flag toggles — filtering vs verbosity levels

**Ambiguity**: The issue lists `v` (verbose), `d` (debug), and `o` (output-only) toggle keys but doesn't define what each mode shows or hides. The existing CLI has `--verbose` (shows tool activity) and `--debug` (shows internal details). "Output-only" is not defined in the existing codebase.

**Resolution**: The display flags control which event types are rendered in the live output viewport:

- **Default mode** (no flags): Shows step lifecycle events (`started`, `running`, `completed`, `failed`), contract validation results, and step progress summaries. Each step shows a single-line status update.
- **Verbose mode** (`v` toggle): Adds `stream_activity` events showing per-tool-call detail (e.g., `[plan] Read .wave/artifacts/spec.md`, `[plan] Write plan.md`). This is equivalent to `wave run -v`.
- **Debug mode** (`d` toggle): Adds internal events: progress ticker heartbeats, ETA updates, token counts per step, compaction progress, and adapter metadata. Useful for diagnosing slow steps or token exhaustion.
- **Output-only mode** (`o` toggle): Shows only step `completed`/`failed` events with their final artifacts — suppresses all intermediate progress. Useful for seeing just the results without noise.

Flags are independent toggles (not mutually exclusive). When `o` is active, it overrides `v` and `d` (output-only takes precedence). The current flag state is displayed at the bottom of the live output area.

### C5: Pipeline completion transition — immediate swap vs animated

**Ambiguity**: The issue says "when pipeline completes, right pane transitions to finished pipeline detail view." It's unclear whether this is an instant swap or a gradual transition, and whether the user should be notified.

**Resolution**: When the executor emits a terminal event (`completed` or `failed`), the live output view appends a final summary line (e.g., "✓ Pipeline completed in 5m 23s" or "✗ Pipeline failed at step 'plan'") and then, after a 2-second delay, transitions the right pane to the finished detail view (the same view implemented in #255). This gives the user time to read the final status before the view changes. The 2-second delay is skipped if the user navigates away (selects a different pipeline) before it fires. The pipeline also moves from the Running section to the Finished section in the left pane.

### C6: Multiple running pipelines — which one streams to the right pane

**Ambiguity**: The TUI supports launching multiple pipelines (each with its own executor goroutine). When multiple pipelines are running, the right pane can only show one at a time. The spec needs to define which pipeline's events are displayed.

**Resolution**: The right pane shows events for the currently selected running pipeline in the left pane. When the user moves the cursor to a different running pipeline, the right pane switches to that pipeline's event stream. Each running pipeline maintains its own event buffer (ring buffer of rendered lines) independently. Switching between running pipelines preserves each buffer's content and scroll position — the user can switch back and resume reading from where they left off. Event buffers are cleaned up when the pipeline transitions to Finished.

### C7: Running pipelines started from outside the TUI

**Ambiguity**: Running pipelines may appear in the list because they were started from the CLI in another terminal. The TUI has no executor reference for these — it only knows about them via state store polling. The right pane can't show live events for externally-started pipelines.

**Resolution**: When a running pipeline started from outside the TUI is selected, the right pane shows the existing informational message (from #255 C3): pipeline name, "Running" status, elapsed time, and an additional note: "Started externally — live output not available." The `v`, `d`, `o` toggle keys are inactive for externally-started pipelines. Only TUI-launched pipelines (those with an event buffer in the launcher) receive live streaming. This is documented as a known limitation.

### C8: Elapsed time updates in the left pane

**Ambiguity**: The issue says "Running pipeline elapsed time updates in left pane in real-time." The current left pane displays `StartedAt time.Time` for running pipelines but doesn't continuously update the elapsed time.

**Resolution**: A 1-second ticker drives elapsed time updates in the left pane. The pipeline list component handles this tick by re-rendering running pipeline items with updated elapsed time (calculated from the pipeline's start time). The ticker is started when there is at least one running pipeline and stopped when there are none. This follows the same tick pattern used by the header bar's logo animation ticker. The elapsed time is formatted as `MM:SS` for runs under an hour and `HH:MM:SS` for longer runs.

### C9: Status line showing current step in the live output header

**Ambiguity**: The issue says "Pipeline status line shows current step (e.g., 'Running (step 3/6: plan)')". It's unclear whether this is a separate header within the right pane or integrated into the scrollable content.

**Resolution**: The live output right pane has a fixed (non-scrollable) header area at the top showing: pipeline name, status with current step (e.g., "Running (step 3/6: plan)"), elapsed time, and model name. Below the header is the scrollable viewport containing the event log. Below the viewport is a fixed footer showing the current display flags and auto-scroll status. This three-part layout (header + viewport + footer) ensures the pipeline status is always visible regardless of scroll position.

### C10: Error display on pipeline failure

**Ambiguity**: The issue says "when pipeline fails, right pane shows error details with actionable messages." The executor emits `FailureReason`, `Remediation`, and `RecoveryHints` in the `failed` event. The spec needs to define how these are rendered.

**Resolution**: When a pipeline emits a `failed` event, the live output appends a styled error block:
- A red `✗ Pipeline failed` header
- The step that failed (step ID and persona)
- The failure reason (e.g., "context_exhaustion", "timeout", "general_error")
- The remediation text (e.g., "Consider splitting this step into smaller tasks")
- Recovery hints (actionable suggestions from the executor)

This error block replaces the need for the user to scroll through logs to find what went wrong. After the 2-second delay (C5), the right pane transitions to the finished detail view which also shows the error.

### C11: NDJSON stdout suppression in TUI mode

**Ambiguity**: The `PipelineLauncher.Launch()` method currently creates an `event.NewNDJSONEmitter()` which writes NDJSON to stdout. However, the TUI owns stdout — Bubble Tea renders the entire UI to stdout. Writing raw NDJSON to stdout during a pipeline run would corrupt the TUI display, producing garbled output.

**Resolution**: When launching a pipeline from the TUI, use `event.NewProgressOnlyEmitter(tuiProgressEmitter)` instead of `event.NewNDJSONEmitter()`. This suppresses all NDJSON output to stdout while routing events through the `ProgressEmitter` callback interface. The TUI's progress emitter implementation converts each `event.Event` into a `PipelineEventMsg` and delivers it via `program.Send()`, which is the bridging mechanism described in C1. This uses the existing `suppressJSON: true` flag in `NDJSONEmitter` — no new emitter infrastructure is needed.

### C12: Enter key behavior on running pipelines

**Ambiguity**: The existing `cursorOnFocusableItem()` method in `content.go` returns `true` only for `itemKindAvailable` and `itemKindFinished`, explicitly excluding `itemKindRunning`. Pressing Enter on a running pipeline is currently a no-op. However, FR-009 requires display flag toggle keys to be active "when the right pane is focused and showing live output," which implies the right pane must be focusable for running pipelines. Without the ability to focus the right pane, users cannot scroll through live output or toggle display flags.

**Resolution**: Extend `cursorOnFocusableItem()` to include `itemKindRunning`. When Enter is pressed on a running pipeline, focus transitions to the right pane (same as available/finished items). If the pipeline is TUI-launched (has an event buffer), the right pane shows the live output view (`stateRunningLive`) with scrollable viewport and display flag toggles active. If the pipeline was started externally (no event buffer), the right pane shows the informational message (`stateRunningInfo`) per C7 — the right pane is still focusable for consistency but toggle keys are inactive. In both cases, Esc returns focus to the left pane. This follows the same focus pattern established by #255 and #256 for available and finished pipelines.

### C13: Display flag effect on existing buffer lines

**Ambiguity**: The buffer stores formatted display lines (C2), and display flags control which event types are formatted into the buffer (C4). User Story 2, scenario 2 says "stream_activity events are no longer shown" but also "Existing verbose lines remain in the buffer." If the buffer stores pre-formatted lines, hiding previously-added lines would require a separate filtering mechanism or storing raw events alongside formatted lines.

**Resolution**: Display flags act as a filter at the **formatting/append stage only**. When an event arrives and verbose mode is off, `stream_activity` events are simply not formatted into the buffer — they are discarded. Lines already present in the buffer are always visible regardless of subsequent flag changes. The phrase "no longer shown" in US-2 scenario 2 refers exclusively to **new events** arriving after the toggle — existing verbose lines remain visible in the buffer and continue to be rendered in the viewport. This keeps the buffer implementation simple (append-only ring buffer of strings) without needing raw event storage or retroactive filtering.

### C14: Status bar hints for live output mode

**Ambiguity**: The `StatusBarModel` currently shows three hint variants based on focus pane and form state. The live output feature introduces new key bindings (`v`, `d`, `o` for display flag toggles) that the status bar should advertise when the right pane is focused on a live output view. Without updated hints, users won't discover the toggle shortcuts.

**Resolution**: Add a new status bar hint state for live output. When the right pane is focused and showing live output (`stateRunningLive`), the status bar displays: `"v: verbose  d: debug  o: output-only  ↑↓: scroll  Esc: back"`. This requires a new message type (e.g., `LiveOutputActiveMsg{Active bool}`) sent when the detail pane enters/exits the `stateRunningLive` state with right-pane focus, so the status bar can switch hint text. The existing `FocusChangedMsg` is insufficient because it doesn't carry pane-state information. This follows the same pattern as `FormActiveMsg` which tells the status bar to switch to form-specific hints.

### C15: Completion transition deferred while user is scrolling

**Ambiguity**: The 2-second completion transition timer (C5) fires regardless of user interaction with the live output viewport. If the user has manually scrolled up to review earlier output (auto-scroll paused) when the pipeline completes, the transition would abruptly replace the live output with the finished detail view, interrupting their reading of the error block or event history.

**Resolution**: The transition timer is deferred while auto-scroll is paused. When a pipeline emits a terminal event while the user is scrolled up (auto-scroll paused), the completion summary line is appended to the buffer but the 2-second transition timer is **not started**. When the user subsequently scrolls to the bottom (auto-scroll resumes), the transition timer starts from that point. This ensures users are never interrupted while reviewing output. If the user navigates away before scrolling to the bottom, the transition is cancelled entirely — the pipeline moves to Finished in the left pane regardless, and the user can view the finished detail by selecting it there.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - View Live Output for a Running Pipeline (Priority: P1)

A developer launches a pipeline from the TUI (via #256's launch flow). The pipeline appears in the Running section of the left pane. The developer selects the running pipeline and presses Enter to focus the right pane. The right pane transitions from the placeholder to a live output view showing a fixed header with the pipeline name, status ("Running (step 1/6: specify)"), and elapsed time. Below the header, a scrollable viewport shows event lines as they arrive: step starts, tool activity (in verbose mode), step completions. The output auto-scrolls to follow new lines. At the bottom, a footer shows the current display flags.

**Why this priority**: This is the core feature of the entire issue — without live streaming, running pipelines remain opaque in the TUI and users must rely on CLI output in another terminal.

**Independent Test**: Can be tested by launching a pipeline (mock adapter), selecting it in the Running section, pressing Enter to focus the right pane, and verifying the right pane renders a live output view with events appearing as the executor emits them. Verify the header shows step progress and the viewport auto-scrolls.

**Acceptance Scenarios**:

1. **Given** a pipeline has been launched from the TUI, **When** the user selects it in the Running section and presses Enter, **Then** the right pane is focused and shows a live output view with a header displaying pipeline name, "Running (step N/M: stepID)", and elapsed time.
2. **Given** the live output view is active, **When** the executor emits a step `started` event, **Then** a new line appears in the viewport (e.g., "[specify] Starting...").
3. **Given** the live output view is active, **When** the executor emits a step `completed` event, **Then** a completion line appears (e.g., "✓ [specify] Completed (42s)") and the header updates to show the next step.
4. **Given** the live output view is active with auto-scroll enabled, **When** new events arrive, **Then** the viewport scrolls to show the latest line.
5. **Given** the live output view is active, **When** the footer renders, **Then** it shows the current display flag state (e.g., "Flags: [v] verbose  [d] debug  [o] output-only").

---

### User Story 2 - Toggle Display Flags (Priority: P1)

A developer watching a running pipeline's live output presses `v` to enable verbose mode. The viewport begins showing tool-level activity (file reads, writes, bash commands) in addition to step lifecycle events. The developer presses `v` again to toggle it off, returning to default mode. Similarly, `d` toggles debug information and `o` toggles output-only mode. The footer updates to reflect the current flag state.

**Why this priority**: Display flags are explicitly called out in the issue's acceptance criteria. They let users control the signal-to-noise ratio of the live output — essential for both quick monitoring (default mode) and debugging (verbose/debug).

**Independent Test**: Can be tested by starting a live output view, pressing each toggle key, and verifying that the rendered event lines include/exclude the appropriate event types. Verify the footer reflects the current state.

**Acceptance Scenarios**:

1. **Given** the live output view is active in default mode, **When** the user presses `v`, **Then** verbose mode is enabled and `stream_activity` events (tool calls) appear in the viewport. The footer shows verbose as active.
2. **Given** verbose mode is active, **When** the user presses `v` again, **Then** verbose mode is disabled and new `stream_activity` events are no longer formatted into the buffer. Existing verbose lines already in the buffer remain visible.
3. **Given** the live output view is active, **When** the user presses `d`, **Then** debug mode is enabled and internal events (heartbeats, token counts, ETA) appear.
4. **Given** the live output view is active, **When** the user presses `o`, **Then** output-only mode is enabled and only step completion/failure events are shown. `v` and `d` flags are visually overridden.
5. **Given** output-only mode is active, **When** the user presses `o` again, **Then** output-only mode is disabled and the previous flag state (`v`, `d`) resumes.

---

### User Story 3 - Pipeline Completion Transition (Priority: P1)

A developer is watching a running pipeline's live output. The pipeline completes all steps successfully. The live output shows a final summary line ("✓ Pipeline completed in 5m 23s"). After a brief pause, the right pane transitions to the finished detail view showing the execution summary with step results, artifacts, and action hints. In the left pane, the pipeline moves from the Running section to the Finished section.

**Why this priority**: Completion transition is critical for the user experience — without it, a completed pipeline would remain showing stale live output or abruptly disappear. The transition provides continuity from monitoring to post-run inspection.

**Independent Test**: Can be tested by running a pipeline to completion (mock adapter) and verifying: (a) the final summary line appears in the live output, (b) after a brief delay the right pane transitions to the finished detail view, (c) the left pane moves the pipeline to the Finished section.

**Acceptance Scenarios**:

1. **Given** a running pipeline is selected and live output is active, **When** the executor emits a `completed` event, **Then** a summary line "✓ Pipeline completed in [duration]" appears in the live output viewport.
2. **Given** the completion summary has appeared and auto-scroll is enabled, **When** 2 seconds elapse, **Then** the right pane transitions to the finished detail view (as defined in #255).
3. **Given** the pipeline has completed, **When** the left pane updates, **Then** the pipeline moves from the Running section to the Finished section with "completed" status.
4. **Given** the pipeline has completed, **When** the user navigates away before the 2-second transition, **Then** the transition timer is cancelled and the right pane shows whatever pipeline the user selected.
5. **Given** the pipeline has completed while auto-scroll is paused, **When** the user scrolls to the bottom, **Then** the 2-second transition timer starts from that point.

---

### User Story 4 - Scroll Through Live Output (Priority: P2)

A developer watching a running pipeline's live output scrolls up using ↑ or Page Up to review earlier events. Auto-scroll pauses, and a "Scrolling paused" indicator appears. New events continue to arrive and are appended to the buffer but the viewport stays at the user's scroll position. The developer scrolls back down to the bottom, and auto-scroll re-engages, resuming real-time following.

**Why this priority**: Scrolling is important for reviewing what happened in earlier steps while the pipeline continues, but most users will rely on auto-scroll for the common case of monitoring progress.

**Independent Test**: Can be tested by starting a live output view, pressing ↑ to scroll up, verifying auto-scroll pauses (indicator shown), then scrolling to the bottom and verifying auto-scroll resumes.

**Acceptance Scenarios**:

1. **Given** the live output view is active with auto-scroll enabled, **When** the user presses ↑, **Then** the viewport scrolls up by one line and auto-scroll is paused.
2. **Given** auto-scroll is paused, **When** new events arrive, **Then** the buffer grows but the viewport remains at the user's scroll position. The "Scrolling paused" indicator is visible.
3. **Given** auto-scroll is paused, **When** the user scrolls to the bottom of the buffer, **Then** auto-scroll re-engages and the "Scrolling paused" indicator disappears.
4. **Given** auto-scroll is paused, **When** the user navigates to a different pipeline and then back, **Then** the scroll position and auto-scroll state are preserved.

---

### User Story 5 - Pipeline Failure Display (Priority: P2)

A developer is watching a running pipeline that encounters an error. The pipeline fails at step "plan". The live output shows an error block with a red header, the failing step name, the failure reason, and actionable remediation suggestions from the executor's recovery hints. After a brief pause, the right pane transitions to the finished detail view showing the failure summary.

**Why this priority**: Error display is important for diagnosing failures quickly, but most pipelines complete successfully, making this secondary to the happy-path streaming.

**Independent Test**: Can be tested by running a pipeline that fails (mock adapter returning error) and verifying the error block renders with failure reason, remediation, and recovery hints.

**Acceptance Scenarios**:

1. **Given** a running pipeline is selected, **When** the executor emits a `failed` event, **Then** the live output appends an error block with: failure header, failing step ID, failure reason, and remediation text.
2. **Given** the error block has appeared, **When** 2 seconds elapse, **Then** the right pane transitions to the finished detail view showing the failure summary.
3. **Given** the executor emits recovery hints, **When** the error block renders, **Then** recovery hints are displayed as actionable suggestions.
4. **Given** a running pipeline fails, **When** the left pane updates, **Then** the pipeline moves from Running to Finished with "failed" status.

---

### User Story 6 - Left Pane Elapsed Time Updates (Priority: P2)

A developer has multiple running pipelines visible in the left pane. Each running pipeline's elapsed time display updates every second, showing the duration since the pipeline started. When all running pipelines complete, the ticker stops.

**Why this priority**: Elapsed time updates provide useful at-a-glance monitoring without needing to select each pipeline, but the live output view (US-1) is the primary monitoring mechanism.

**Independent Test**: Can be tested by launching a pipeline (mock adapter with delay) and verifying the elapsed time in the left pane increments each second. Verify the ticker stops when no running pipelines remain.

**Acceptance Scenarios**:

1. **Given** one or more pipelines are running, **When** 1 second elapses, **Then** each running pipeline's elapsed time display in the left pane updates (e.g., "00:03" → "00:04").
2. **Given** all running pipelines have completed, **When** the next tick fires, **Then** the ticker stops (no more elapsed time update messages).
3. **Given** a running pipeline shows "01:23" and completes, **When** it moves to the Finished section, **Then** it shows the final duration (not a ticking time).

---

### Edge Cases

- What happens when the ring buffer fills up (1000 lines) while the user is scrolled to the top? The oldest lines are dropped, which may shift the viewport's absolute position. The viewport should adjust its offset to maintain the user's relative position in the buffer as closely as possible.
- What happens when a pipeline emits events faster than the TUI can render? Events queue in the UI framework's internal message queue. The viewport batches pending events on each render cycle rather than rendering per-event.
- What happens when the user switches between two running pipelines rapidly? Each pipeline's buffer and scroll state are preserved independently. Switching is a model state change (which buffer to render), not a data refetch.
- What happens when the terminal is resized during live output? The viewport re-renders at the new dimensions. Long lines may wrap or truncate depending on the new width. Scroll position is preserved relative to the buffer.
- What happens when a pipeline has zero events (just started)? The viewport shows an empty state with a "Waiting for events..." message. The header still shows "Running (step 0/N)" status.
- What happens when the user presses `v`/`d`/`o` while the left pane is focused? The toggle keys are only active when the right pane is focused and showing live output. They are ignored otherwise, following the established focus-gating pattern.
- What happens when the live output transition timer fires but the pipeline's finished detail data hasn't loaded yet? The right pane transitions to a loading state (same as #255's loading state) while the detail data is fetched asynchronously.
- What happens when a pipeline was started externally and then the TUI is started? The running pipeline appears in the list but has no event buffer. Selecting it shows the informational message (C7), not live output.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: When a TUI-launched running pipeline is selected in the left pane, the right pane MUST transition to a live output view consisting of three sections: a fixed header (pipeline name, status with current step, elapsed time, model), a scrollable viewport (event log), and a fixed footer (display flags, auto-scroll status).
- **FR-002**: The executor's events MUST be delivered to the TUI event loop in real time via the existing progress emitter callback interface, using `event.NewProgressOnlyEmitter()` to suppress NDJSON stdout output (which would corrupt the TUI display). Each executor event MUST be converted into a UI message carrying the run ID and full event data, delivered without blocking the executor.
- **FR-003**: Each TUI-launched running pipeline MUST maintain its own bounded event buffer of formatted display lines (capacity: 1000 lines). When the buffer exceeds capacity, the oldest entries MUST be discarded. Buffers MUST be keyed by run ID and cleaned up when the pipeline transitions to Finished.
- **FR-004**: The live output viewport MUST auto-scroll to follow new events by default. When the user scrolls up (↑, Page Up), auto-scroll MUST pause. Auto-scroll MUST resume when the user scrolls to the bottom of the buffer.
- **FR-005**: A visual indicator MUST be displayed when auto-scroll is paused (e.g., "⏸ Scrolling paused — scroll to bottom to resume").
- **FR-006**: The `v` key MUST toggle verbose mode on/off. When enabled, stream activity events (tool calls with tool name and target) MUST be included in the rendered output. When disabled, only step lifecycle events are shown.
- **FR-007**: The `d` key MUST toggle debug mode on/off. When enabled, internal events (progress heartbeats, token counts, ETA updates, compaction progress) MUST be included in the rendered output.
- **FR-008**: The `o` key MUST toggle output-only mode on/off. When enabled, only step `completed` and `failed` events MUST be shown. Output-only mode MUST override verbose and debug modes.
- **FR-009**: The display flag toggle keys (`v`, `d`, `o`) MUST only be active when the right pane is focused and showing live output for a TUI-launched pipeline. They MUST be ignored in all other contexts.
- **FR-010**: The live output header MUST display the current step progress in the format "Running (step N/M: stepID)" where N is the current step number (1-based), M is total steps, and stepID is the currently executing step's identifier. This MUST update as each step starts.
- **FR-011**: The live output header MUST display a continuously updating elapsed time since pipeline start, formatted as `MM:SS` for runs under an hour and `HH:MM:SS` for longer runs.
- **FR-012**: When the executor emits a terminal event (`completed` or `failed`), the live output MUST append a summary line with the final status and total duration.
- **FR-013**: After a terminal event, the right pane MUST transition to the finished detail view (from #255) after a 2-second delay. The delay MUST be cancelled if the user navigates to a different pipeline before it fires. The delay MUST be deferred while auto-scroll is paused — the timer starts only when the user scrolls to the bottom (auto-scroll resumes).
- **FR-014**: When a pipeline fails, the live output MUST display an error block containing: the failing step ID, the failure reason, the remediation text, and any recovery hints from the event.
- **FR-015**: The left pane MUST display a continuously updating elapsed time for each running pipeline, driven by a 1-second ticker. The ticker MUST start when at least one pipeline is running and stop when none are running.
- **FR-016**: Switching between running pipelines in the left pane MUST preserve each pipeline's event buffer, scroll position, and auto-scroll state independently.
- **FR-017**: Running pipelines started from outside the TUI (no event buffer in the launcher) MUST display an informational message in the right pane when selected: pipeline name, "Running" status, elapsed time, and "Started externally — live output not available."
- **FR-018**: Each event message delivered to the TUI MUST carry a run identifier and the event data so that the live output component can route events to the correct pipeline's buffer.
- **FR-019**: Event formatting MUST respect `NO_COLOR` — all styled output (colors, bold, indicators) MUST degrade to plain text when `NO_COLOR` is set.
- **FR-020**: The live output footer MUST show the current display flag state with visual indicators for active/inactive flags (e.g., `[v] verbose  [ ] debug  [ ] output-only`).
- **FR-021**: The live output view MUST handle terminal resizing by re-rendering the viewport at the new dimensions, preserving scroll position and auto-scroll state.
- **FR-022**: Pressing Enter on a running pipeline in the left pane MUST focus the right pane, enabling scroll navigation and display flag toggles. This extends `cursorOnFocusableItem()` to include `itemKindRunning`. Esc MUST return focus to the left pane.
- **FR-023**: The status bar MUST display context-appropriate hints when the right pane is focused and showing live output: `"v: verbose  d: debug  o: output-only  ↑↓: scroll  Esc: back"`. A `LiveOutputActiveMsg` MUST signal the status bar to switch hint text, following the same pattern as `FormActiveMsg`.
- **FR-024**: Display flag toggles MUST act as filters at the formatting stage only. Toggling a flag off MUST prevent new events of that type from being formatted into the buffer, but MUST NOT remove or hide lines already present in the buffer.

### Key Entities

- **PipelineEventMsg**: Message type carrying the run ID and the full event data. Emitted by the progress emitter callback injected into the executor. Routed through the UI event loop to the appropriate pipeline's event buffer.
- **LiveOutputModel**: UI model responsible for rendering the live output view for a single running pipeline. Owns the event buffer (ring buffer), viewport, auto-scroll state, and display flag toggles. Managed by the detail pane as a new state (`stateRunningLive`).
- **EventBuffer**: Ring buffer of formatted display lines, capacity 1000. Keyed by run ID on the launcher or a dedicated buffer manager. Each buffer maintains its own write position, line count, and provides methods to append formatted lines and retrieve a window of lines for viewport rendering.
- **DisplayFlags**: Struct tracking the current toggle state: Verbose, Debug, and OutputOnly booleans. Determines which event types are formatted and appended to the buffer. Owned by each live output model instance.
- **ElapsedTickMsg**: Message emitted by a 1-second ticker to drive elapsed time updates in the left pane. The ticker is managed by the list or content component — started when running pipeline count is greater than zero, stopped when it reaches zero.
- **TransitionTimerMsg**: Message emitted after the 2-second post-completion delay to trigger the right pane transition from live output to finished detail view. Carries the run ID to identify which pipeline's transition to execute.
- **LiveOutputActiveMsg**: Message signaling the status bar that the right pane is showing live output (`Active: true`) or has left that state (`Active: false`). Used to switch status bar hints to show display flag shortcuts.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Selecting a TUI-launched running pipeline displays a live output view with events appearing within one render cycle of emission — verified by unit tests with mock executor emitting events and asserting viewport content updates.
- **SC-002**: Display flag toggles (`v`, `d`, `o`) change the visible event types immediately on keypress — verified by unit tests toggling each flag and checking rendered output includes/excludes the appropriate event categories.
- **SC-003**: Auto-scroll pauses on manual scroll and resumes when the viewport reaches the bottom — verified by unit tests simulating scroll input and checking the auto-scroll indicator state.
- **SC-004**: Pipeline completion triggers a summary line followed by a 2-second delayed transition to the finished detail view — verified by tests with mock executor and timer assertions.
- **SC-005**: Pipeline failure renders an error block with step ID, failure reason, remediation, and recovery hints — verified by tests with mock executor emitting a `failed` event with all error fields populated.
- **SC-006**: Elapsed time in the left pane updates every second for running pipelines — verified by tests asserting elapsed tick handling increments displayed time.
- **SC-007**: Switching between running pipelines preserves each pipeline's event buffer and scroll position — verified by tests with two concurrent mock pipelines and interleaved selection.
- **SC-008**: All existing TUI tests continue to pass after integration — the live output feature does not break existing pipeline list, detail, header, status bar, or launch flow components.
- **SC-009**: The live output view renders correctly at terminal widths from 80 to 300 columns and heights from 24 to 100 rows — verified by rendering tests at boundary dimensions.
- **SC-010**: Display flag keys are ignored when the left pane is focused — verified by tests sending `v`/`d`/`o` key events with left pane focus and asserting no state change.
