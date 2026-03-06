# Data Model: TUI Live Output Streaming

**Date**: 2026-03-06
**Feature**: #257 — TUI Live Output Streaming

## New Types

### PipelineEventMsg

Message type bridging executor events into the Bubble Tea event loop.
Emitted by the TUI progress emitter callback inside the executor goroutine.

```go
// PipelineEventMsg carries an executor event for a specific pipeline run.
// Delivered via program.Send() from the progress emitter callback.
type PipelineEventMsg struct {
    RunID string
    Event event.Event
}
```

**Flow**: `ProgressEmitter.EmitProgress()` → `program.Send(PipelineEventMsg{...})` → `ContentModel.Update()` → route to `LiveOutputModel` by RunID.

### LiveOutputModel

UI model for the live output view rendered in the right pane when a TUI-launched running pipeline is focused. Owns the event buffer, viewport, auto-scroll state, display flags, and completion transition.

```go
// LiveOutputModel renders real-time pipeline output in the right pane.
type LiveOutputModel struct {
    runID        string           // Pipeline run identifier
    pipelineName string           // Pipeline display name
    width        int
    height       int

    // Event buffer (ring buffer of formatted lines)
    buffer       *EventBuffer

    // Viewport for scrollable content
    viewport     viewport.Model

    // Auto-scroll state
    autoScroll   bool

    // Display flag toggles
    flags        DisplayFlags

    // Step progress tracking (for header)
    currentStep  string           // Current step ID
    stepNumber   int              // 1-based current step number
    totalSteps   int              // Total steps in pipeline
    model        string           // Model name (e.g., "opus")
    startedAt    time.Time        // Pipeline start time

    // Completion state
    completed        bool         // Pipeline has emitted terminal event
    completionPending bool        // Waiting for auto-scroll to resume before starting timer
    transitionRunID  string       // RunID for which transition timer is active
}
```

**Lifecycle**: Created when a TUI-launched running pipeline is focused. One instance per active live output view. Destroyed when the pipeline transitions to Finished.

### EventBuffer

Ring buffer of pre-formatted display lines with fixed capacity.

```go
// EventBuffer is a bounded ring buffer of formatted display lines.
type EventBuffer struct {
    lines    []string
    capacity int
    head     int  // Index of oldest entry
    count    int  // Number of entries currently in buffer
}
```

**Methods**:
- `NewEventBuffer(capacity int) *EventBuffer`
- `Append(line string)` — adds a line, drops oldest if at capacity
- `Lines() []string` — returns all lines in order (oldest to newest)
- `Len() int` — returns current line count

**Lifecycle**: Created per-pipeline at launch time. Cleaned up when pipeline transitions to Finished.

### DisplayFlags

Tracks the current display flag toggle state for event filtering.

```go
// DisplayFlags tracks which event categories are visible in the live output.
type DisplayFlags struct {
    Verbose    bool  // Show stream_activity events (tool calls)
    Debug      bool  // Show progress heartbeats, token counts, ETA, compaction
    OutputOnly bool  // Show only completed/failed events (overrides Verbose & Debug)
}
```

**Lifecycle**: Owned by `LiveOutputModel`. Toggled by user key presses. Persisted per-pipeline buffer.

### ElapsedTickMsg

Tick message for 1-second elapsed time updates in the left pane.

```go
// ElapsedTickMsg drives elapsed time updates for running pipelines in the left pane.
type ElapsedTickMsg struct{}
```

**Flow**: `tea.Tick(1s)` → `PipelineListModel.Update()` → re-render running items.

### TransitionTimerMsg

Timer message for the 2-second post-completion transition delay.

```go
// TransitionTimerMsg signals that the completion transition delay has elapsed.
type TransitionTimerMsg struct {
    RunID string  // Identifies which pipeline's transition to execute
}
```

**Flow**: `tea.Tick(2s)` → `PipelineDetailModel.Update()` → transition to `stateFinishedDetail`.

### LiveOutputActiveMsg

Status bar hint switching signal, following the `FormActiveMsg` pattern.

```go
// LiveOutputActiveMsg signals the status bar to switch to live output hints.
type LiveOutputActiveMsg struct {
    Active bool
}
```

**Flow**: `PipelineDetailModel` → `AppModel.Update()` → `StatusBarModel.Update()`.

### TUIProgressEmitter

Adapter implementing `event.ProgressEmitter` that bridges events into the TUI.

```go
// TUIProgressEmitter implements event.ProgressEmitter to bridge executor events
// into the Bubble Tea event loop via program.Send().
type TUIProgressEmitter struct {
    program *tea.Program
    runID   string
}
```

**Methods**:
- `EmitProgress(evt event.Event) error` — wraps event in `PipelineEventMsg` and calls `program.Send()`.

**Lifecycle**: Created per-launch by `PipelineLauncher.Launch()`. Lives for the duration of the executor goroutine.

## Modified Types

### PipelineLauncher

Add `*tea.Program` reference and event buffer management.

```go
type PipelineLauncher struct {
    deps      LaunchDependencies
    cancelFns map[string]context.CancelFunc
    buffers   map[string]*EventBuffer      // NEW: per-pipeline event buffers
    program   *tea.Program                 // NEW: for program.Send() in emitter
    mu        sync.Mutex
}
```

**New Methods**:
- `SetProgram(p *tea.Program)` — called after `tea.NewProgram()` returns
- `GetBuffer(runID string) *EventBuffer` — returns buffer for a pipeline (nil for external pipelines)
- `HasBuffer(runID string) bool` — checks if pipeline was TUI-launched

### DetailPaneState (extended)

Add new state for live output view.

```go
const (
    stateEmpty           DetailPaneState = iota
    stateLoading
    stateAvailableDetail
    stateFinishedDetail
    stateRunningInfo                           // Existing: external running pipeline info
    stateRunningLive                           // NEW: TUI-launched running pipeline live output
    stateConfiguring
    stateLaunching
    stateError
)
```

### PipelineDetailModel

Add live output model and state for live streaming.

```go
type PipelineDetailModel struct {
    // ... existing fields ...

    // Live output state (NEW)
    liveOutput *LiveOutputModel  // Active live output model (nil when not in stateRunningLive)
}
```

### PipelineListModel

Add elapsed time ticker management.

```go
type PipelineListModel struct {
    // ... existing fields ...

    // Elapsed time ticker (NEW)
    tickerActive bool  // Whether the 1-second ticker is running
}
```

### ContentModel

Add `PipelineEventMsg` routing and running pipeline focus logic.

**New message routing**:
- `PipelineEventMsg` → route to `PipelineDetailModel` if the selected pipeline matches the run ID
- Enter on `itemKindRunning` → check `launcher.HasBuffer(runID)` to determine `stateRunningLive` vs `stateRunningInfo`

### StatusBarModel

Add `liveOutputActive` field for live output hint state.

```go
type StatusBarModel struct {
    width            int
    contextLabel     string
    focusPane        FocusPane
    formActive       bool
    liveOutputActive bool  // NEW: live output mode active
}
```

### AppModel

Forward `LiveOutputActiveMsg` to status bar (same pattern as `FormActiveMsg`).

### RunTUI()

Set `*tea.Program` reference on launcher between `tea.NewProgram()` and `p.Run()`.

## State Transitions

```
Left pane focused, running item selected (TUI-launched)
    ↓ Enter
ContentModel: check launcher.HasBuffer(runID)
    → true: focus right, create LiveOutputModel, set stateRunningLive
    → emit LiveOutputActiveMsg{Active: true}
PipelineDetailModel: stateRunningLive (live output rendering)
    ↓ Esc
PipelineDetailModel: stateRunningLive → destroy LiveOutputModel
    → emit LiveOutputActiveMsg{Active: false}
    → focus left

Left pane focused, running item selected (external)
    ↓ Enter
ContentModel: check launcher.HasBuffer(runID)
    → false: focus right, set stateRunningInfo (existing behavior)

stateRunningLive, terminal event received
    ↓ PipelineEventMsg with state="completed"/"failed"
LiveOutputModel: append summary/error block, set completed=true
    ↓ auto-scroll enabled → start 2s timer
    ↓ auto-scroll paused → defer timer (completionPending=true)
    ↓ TransitionTimerMsg
PipelineDetailModel: stateRunningLive → stateFinishedDetail
    → emit LiveOutputActiveMsg{Active: false}
    → clean up event buffer

stateRunningLive, user navigates away
    ↓ Esc or select different pipeline
LiveOutputModel: cancel pending transition, preserve buffer state
    → emit LiveOutputActiveMsg{Active: false}
    → focus left (buffer persists for return)
```

## Event Format Specification

### Default Mode (lifecycle events only)

```
[specify] Starting... (persona: navigator, model: opus)
[specify] ✓ Completed (42s)
[plan] Starting... (persona: craftsman, model: opus)
[plan] Contract validation: PASSED
[plan] ✓ Completed (1m23s)
✓ Pipeline completed in 5m 23s
```

### Verbose Mode (adds tool activity)

```
[specify] Starting... (persona: navigator, model: opus)
[specify] Read .wave/artifacts/spec.md
[specify] Write specs/257/plan.md
[specify] Bash go test ./...
[specify] ✓ Completed (42s)
```

### Debug Mode (adds internal events)

```
[specify] Starting... (persona: navigator, model: opus)
[specify] ♡ heartbeat (tokens: 1234/200000)
[specify] ETA: ~2m remaining
[specify] ✓ Completed (42s)
```

### Error Block (on failure)

```
✗ Pipeline failed
  Step: plan (craftsman)
  Reason: context_exhaustion
  Remediation: Consider splitting this step into smaller tasks
  Recovery hints:
    → wave run my-pipeline --from-step plan
    → wave run my-pipeline --model opus
```
