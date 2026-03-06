# Research: TUI Live Output Streaming

**Date**: 2026-03-06
**Feature**: #257 — TUI Live Output Streaming

## R1: Event Delivery — Callback-to-Message Adapter

**Decision**: Use the existing `ProgressEmitter` callback interface with `event.NewProgressOnlyEmitter()` to bridge executor events into Bubble Tea's message loop via `program.Send()`.

**Rationale**: The executor already calls `emitter.Emit(event)` at every lifecycle point — step started/completed/failed, stream_activity (tool calls), progress heartbeats, contract validation, and compaction stats. The `NDJSONEmitter` supports a `ProgressEmitter` callback that receives every event. `NewProgressOnlyEmitter(pe)` sets `suppressJSON: true`, preventing NDJSON output to stdout (which would corrupt the TUI display since Bubble Tea owns stdout).

The TUI's progress emitter implementation converts each `event.Event` into a `PipelineEventMsg{RunID, Event}` and delivers it via `program.Send()`, which is thread-safe and non-blocking. This requires the `PipelineLauncher` to hold a `*tea.Program` reference, set after `tea.NewProgram()` returns via `SetProgram()`.

**Key implementation points**:
- `PipelineLauncher.Launch()` currently creates `event.NewNDJSONEmitter()` (line 87 of `pipeline_launcher.go`). Change to `event.NewProgressOnlyEmitter(tuiEmitter)`.
- The `tuiEmitter` implements `event.ProgressEmitter` with `EmitProgress(event.Event) error` that calls `l.program.Send(PipelineEventMsg{...})`.
- `program.Send()` is safe to call from goroutines — documented in Bubble Tea's API.

**Alternatives Rejected**:
- Channel-based pub/sub: Adds unnecessary complexity; the callback interface is already there.
- Polling state store: 5-second refresh interval is too slow for live streaming.
- NDJSON stdout + parse: Would corrupt TUI display; Bubble Tea owns stdout.

## R2: Event Buffer — Ring Buffer of Formatted Lines

**Decision**: Use a simple ring buffer of pre-formatted strings (capacity 1000 lines) per running pipeline.

**Rationale**: Events arrive as `event.Event` structs. The buffer stores formatted display lines (not raw events) because:
1. Display flags (verbose/debug/output-only) filter at the formatting stage — events that don't match the current flags are not added to the buffer.
2. Once in the buffer, lines are always visible regardless of subsequent flag changes (per C13/FR-024).
3. A ring buffer of strings is simpler than storing raw events + retroactive filtering.

Buffer implementation: a fixed-size `[]string` slice with write position and count. When count exceeds capacity, oldest entries are overwritten (circular). Methods: `Append(line string)`, `Lines(offset, limit int) []string`, `Len() int`.

Each running pipeline's buffer is keyed by run ID on the `PipelineLauncher`. Buffers are cleaned up when the pipeline transitions to Finished.

**Alternatives Rejected**:
- Unbounded slice: Memory grows without limit for long-running pipelines.
- Store raw events + reformat on flag change: More complex, and C13 says existing lines always remain visible.

## R3: Auto-scroll with Pause/Resume via Viewport

**Decision**: Use `charmbracelet/bubbles/viewport.Model` for the scrollable content area with an `autoScroll bool` flag.

**Rationale**: The `viewport.Model` from Bubble Tea's `bubbles` package handles scrolling, key bindings (up, down, PgUp, PgDn), and content rendering. The live output model wraps it with auto-scroll logic:
- `autoScroll` starts as `true`.
- On each new event (buffer append), if `autoScroll` is true, call `viewport.GotoBottom()`.
- When the user presses scroll keys (detected via `tea.KeyMsg` before forwarding to viewport), set `autoScroll = false`.
- After forwarding scroll keys to the viewport, check if `viewport.AtBottom()` returns true — if so, set `autoScroll = true`.

The `viewport.Model` is already used in `PipelineDetailModel` for finished detail scrolling, so this follows the established pattern.

**Alternatives Rejected**:
- Custom scroll implementation: Reinventing what viewport already provides.
- Always auto-scroll: Users need to scroll up to review earlier output.

## R4: Display Flag Filtering at Format Stage

**Decision**: Three independent boolean flags (`Verbose`, `Debug`, `OutputOnly`) with format-time filtering.

**Rationale**: When an event arrives:
1. Check the event's `State` field against the current display flags.
2. If the event matches the active filter criteria, format it into one or more display lines and append to the buffer.
3. If it doesn't match, discard it silently.

Flag logic per spec (C4):
- **Default mode** (no flags): Shows `started`, `running`, `completed`, `failed`, `contract_validating` events.
- **Verbose (`v`)**: Adds `stream_activity` events (tool calls with tool name and target).
- **Debug (`d`)**: Adds `step_progress` (heartbeats), `eta_updated`, `compaction_progress` events and token counts.
- **Output-only (`o`)**: Only `completed` and `failed` events. Overrides `v` and `d`.

Flags are independent toggles (not radio buttons). Output-only takes precedence when active.

**Alternatives Rejected**:
- Retroactive filtering (re-render buffer on flag change): Per C13, existing lines remain visible.
- Mutually exclusive modes: Spec says flags are independent toggles.

## R5: Completion Transition Timer with Deferred Start

**Decision**: Use `tea.Tick()` for the 2-second delay, with deferred start when auto-scroll is paused.

**Rationale**: When a terminal event (`completed`/`failed`) arrives:
1. Append the summary/error block to the buffer.
2. If `autoScroll` is true, start a 2-second `tea.Tick()` that returns `TransitionTimerMsg{RunID}`.
3. If `autoScroll` is false (user scrolled up), record `completionPending = true` but don't start the timer.
4. When auto-scroll resumes (user scrolls to bottom), start the timer.
5. If the user navigates away (selects different pipeline), cancel the transition entirely.

`TransitionTimerMsg` carries the run ID so the handler can verify the pipeline is still selected before transitioning.

**Alternatives Rejected**:
- Immediate transition: Interrupts users reading output.
- No transition: Users have to manually navigate to finished detail.

## R6: Elapsed Time Ticker for Left Pane

**Decision**: Use a 1-second `tea.Tick()` returning `ElapsedTickMsg`, managed by the pipeline list model.

**Rationale**: The header bar already uses `LogoTickMsg` for logo animation with a ticker pattern. The elapsed time ticker follows the same approach:
- Start ticker when running pipeline count > 0.
- Stop ticker (don't re-issue tick cmd) when count == 0.
- On each tick, re-render running pipeline items with updated elapsed time (calculated from `StartedAt`).

The `formatDuration()` function in `pipeline_list.go` already formats durations. The spec requires `MM:SS` / `HH:MM:SS` format — need a new `formatElapsed()` function for this specific format since `formatDuration()` uses compact format (`Nm`/`Nh`).

**Alternatives Rejected**:
- Dedicated goroutine: `tea.Tick()` is the idiomatic Bubble Tea approach.
- Reuse pipeline refresh timer (5s): Too slow for 1-second elapsed time updates.

## R7: tea.Program Reference for Send()

**Decision**: Store `*tea.Program` reference on `PipelineLauncher` via `SetProgram()`.

**Rationale**: `tea.NewProgram()` returns a `*tea.Program` that exposes `Send(msg tea.Msg)`. The launcher needs this to deliver `PipelineEventMsg` from the executor's progress emitter callback. Since `tea.NewProgram()` is called in `RunTUI()`, the program reference is set on the launcher after creation:

```go
p := tea.NewProgram(model, tea.WithAltScreen())
if model.content.launcher != nil {
    model.content.launcher.SetProgram(p)
}
_, err := p.Run()
```

**Alternatives Rejected**:
- Passing program through LaunchDependencies: Program doesn't exist until after `NewAppModel()`.
- Using channels instead of program.Send(): More complex, less idiomatic.

## R8: Status Bar Hint Switching for Live Output

**Decision**: Add `LiveOutputActiveMsg{Active bool}` message type, following the `FormActiveMsg` pattern.

**Rationale**: The status bar currently has three hint states: left pane focused, right pane focused, and form active. Live output needs a fourth hint state showing `v`/`d`/`o`/scroll keys. The existing `FormActiveMsg` pattern is the correct approach — emit `LiveOutputActiveMsg{Active: true}` when entering `stateRunningLive` with focus, emit `Active: false` when leaving.

**Alternatives Rejected**:
- Extending `FocusChangedMsg` with state info: Breaks the single-responsibility of that message.
- Status bar querying content state: Violates Bubble Tea's message-passing architecture.
