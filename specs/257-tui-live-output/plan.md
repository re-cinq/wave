# Implementation Plan: TUI Live Output Streaming

**Branch**: `257-tui-live-output` | **Date**: 2026-03-06 | **Spec**: `specs/257-tui-live-output/spec.md`
**Input**: Feature specification from `/specs/257-tui-live-output/spec.md`

## Summary

Connect the pipeline executor's event system to the TUI right pane for real-time output streaming. When a TUI-launched running pipeline is selected and focused, the right pane shows a three-part layout: fixed header (pipeline name, step progress, elapsed time, model), scrollable viewport (event log from a 1000-line ring buffer), and fixed footer (display flags, auto-scroll status). Events bridge from the executor to the TUI via `event.NewProgressOnlyEmitter()` + `program.Send()`. Display flag toggles (`v`/`d`/`o`) filter events at the formatting stage. Auto-scroll follows new output by default; manual scrolling pauses it. Pipeline completion triggers a 2-second delayed transition to the finished detail view. The left pane shows a continuously updating elapsed time ticker for running pipelines.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `charmbracelet/bubbletea` v1.3.10, `charmbracelet/bubbles/viewport` (existing)
**Storage**: SQLite via `internal/state` (existing — run status updates on completion)
**Testing**: `go test` with `testify/assert`, `testify/require`
**Target Platform**: Linux/macOS terminal (80–300 columns, 24–100 rows)
**Project Type**: Single Go binary — changes in `internal/tui/` and `cmd/wave/`
**Constraints**: No new external dependencies; must not break existing tests (`go test ./...`)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new runtime dependencies. Uses existing bubbletea, bubbles/viewport. |
| P2: Manifest as SSOT | ✅ Pass | Pipeline data flows from manifest via existing provider. |
| P3: Persona-Scoped Execution | ✅ Pass | Executor runs full pipeline with persona scoping — TUI only observes events. |
| P4: Fresh Memory at Step Boundary | ✅ Pass | TUI is a display layer — doesn't affect step execution context. |
| P5: Navigator-First Architecture | ✅ Pass | Executor runs the full pipeline DAG including navigator steps. |
| P6: Contracts at Every Handover | ✅ Pass | Executor handles contract validation — TUI displays results but doesn't bypass. |
| P7: Relay via Dedicated Summarizer | N/A | TUI component, no context compaction. |
| P8: Ephemeral Workspaces | ✅ Pass | Executor creates workspaces as normal — TUI doesn't interfere. |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling in TUI layer. Executor inherits env vars. |
| P10: Observable Progress | ✅ Pass | Enhances observability by routing executor events to the TUI display. |
| P11: Bounded Recursion | ✅ Pass | Executor enforces bounds — TUI doesn't modify executor behavior. |
| P12: Minimal Step State Machine | ✅ Pass | Uses existing step state machine via executor. |
| P13: Test Ownership | ✅ Pass | All new code will have tests; existing tests updated for modified signatures. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/257-tui-live-output/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/tui/
├── app.go                     # MODIFY — forward LiveOutputActiveMsg to status bar, SetProgram()
├── app_test.go                # MODIFY — test LiveOutputActiveMsg forwarding
├── content.go                 # MODIFY — route PipelineEventMsg, Enter on running items, extend cursorOnFocusableItem()
├── content_test.go            # MODIFY — test running pipeline focus, event routing
├── pipeline_detail.go         # MODIFY — add stateRunningLive, LiveOutputModel integration, transition timer
├── pipeline_detail_test.go    # MODIFY — test live output view rendering, flag toggles, transition
├── pipeline_list.go           # MODIFY — add elapsed time ticker (ElapsedTickMsg), formatElapsed()
├── pipeline_list_test.go      # MODIFY — test elapsed time ticker start/stop, format
├── pipeline_launcher.go       # MODIFY — add SetProgram(), buffers map, TUIProgressEmitter, NewProgressOnlyEmitter
├── pipeline_launcher_test.go  # MODIFY — test emitter creation, buffer management
├── pipeline_messages.go       # MODIFY — add PipelineEventMsg, ElapsedTickMsg, TransitionTimerMsg, LiveOutputActiveMsg
├── live_output.go             # NEW — LiveOutputModel, EventBuffer, DisplayFlags, event formatting
├── live_output_test.go        # NEW — buffer tests, flag filtering, auto-scroll, viewport rendering
├── statusbar.go               # MODIFY — add liveOutputActive state, live output hints
├── statusbar_test.go          # MODIFY — test LiveOutputActiveMsg handling, hint text
├── header_messages.go         # READ-ONLY — FocusPane, FocusChangedMsg reused

cmd/wave/commands/
├── run.go                     # READ-ONLY — reference for executor construction patterns
```

**Structure Decision**: One new file (`live_output.go`) contains the LiveOutputModel, EventBuffer, DisplayFlags, and event formatting logic. This isolates the new feature's complexity from existing components while keeping it in the `tui` package for shared type access.

## Implementation Approach

### Phase 1: Foundation — New Types and Messages

**Files**: `pipeline_messages.go`, `live_output.go`

1. Add new message types to `pipeline_messages.go`:
   - `PipelineEventMsg{RunID string, Event event.Event}`
   - `ElapsedTickMsg{}`
   - `TransitionTimerMsg{RunID string}`
   - `LiveOutputActiveMsg{Active bool}`

2. Add `stateRunningLive` to `DetailPaneState` constants.

3. Create `live_output.go` with core types:
   - `EventBuffer` struct with `NewEventBuffer()`, `Append()`, `Lines()`, `Len()`
   - `DisplayFlags` struct with `Verbose`, `Debug`, `OutputOnly` booleans
   - `LiveOutputModel` struct with constructor, buffer, viewport, flags, step tracking
   - `formatEvent()` function mapping `event.Event` to display line(s) based on flags
   - `formatElapsed()` function producing `MM:SS` / `HH:MM:SS` format

### Phase 2: EventBuffer and Event Formatting

**Files**: `live_output.go`, `live_output_test.go`

1. Implement `EventBuffer`:
   - Ring buffer using a fixed-size `[]string` with head pointer and count
   - `Append()` overwrites oldest entry when at capacity
   - `Lines()` returns entries in chronological order
   - Test: append beyond capacity, verify oldest dropped, verify order

2. Implement event formatting with display flag awareness:
   - `shouldFormat(evt event.Event, flags DisplayFlags) bool` — filter logic
   - `formatEventLine(evt event.Event) string` — produces display line
   - Default mode: `started`, `running`, `completed`, `failed`, `contract_validating`
   - Verbose: adds `stream_activity`
   - Debug: adds `step_progress`, `eta_updated`, `compaction_progress`
   - OutputOnly: only `completed`, `failed` (overrides verbose/debug)
   - Error block formatting for `failed` events with remediation/recovery hints
   - Respect `NO_COLOR` env var by checking `os.Getenv("NO_COLOR")`

### Phase 3: LiveOutputModel — Viewport and Auto-scroll

**Files**: `live_output.go`, `live_output_test.go`

1. Implement `LiveOutputModel`:
   - Constructor: `NewLiveOutputModel(runID, name string, buffer *EventBuffer, startedAt time.Time, totalSteps int) LiveOutputModel`
   - `SetSize(w, h int)` — allocate header (3 lines), footer (2 lines), viewport gets remainder
   - `Update(msg tea.Msg)` — handle keys, PipelineEventMsg, TransitionTimerMsg
   - `View()` — render header + viewport + footer

2. Implement auto-scroll:
   - On `PipelineEventMsg`: format event, append to buffer, update viewport content, if `autoScroll` then `viewport.GotoBottom()`
   - On scroll keys (↑, ↓, PgUp, PgDn): set `autoScroll = false`, forward to viewport, check `viewport.AtBottom()` to re-engage
   - Render auto-scroll indicator in footer when paused

3. Implement display flag toggles:
   - `v` toggles `flags.Verbose`, `d` toggles `flags.Debug`, `o` toggles `flags.OutputOnly`
   - Footer renders current flag state: `[v] verbose  [ ] debug  [ ] output-only`

4. Implement step progress tracking:
   - On `started` events with `StepID`: update `currentStep`, `stepNumber`
   - Header renders: "Running (step N/M: stepID)"
   - Header elapsed time updates on each render (computed from `startedAt`)

### Phase 4: Completion Transition

**Files**: `live_output.go`, `live_output_test.go`

1. Implement terminal event handling:
   - On `completed` event: append summary line ("✓ Pipeline completed in [duration]"), set `completed = true`
   - On `failed` event: append error block (step ID, reason, remediation, recovery hints), set `completed = true`
   - If `autoScroll` is true: start 2-second `tea.Tick()` returning `TransitionTimerMsg{RunID}`
   - If `autoScroll` is false: set `completionPending = true`

2. Implement deferred transition:
   - When auto-scroll resumes (user scrolls to bottom after completion): start the 2-second timer
   - On `TransitionTimerMsg`: verify RunID matches, return cmd to transition to finished detail

### Phase 5: PipelineLauncher — Event Bridge and Buffer Management

**Files**: `pipeline_launcher.go`, `pipeline_launcher_test.go`

1. Add `*tea.Program` reference:
   - Add `program *tea.Program` field
   - Add `SetProgram(p *tea.Program)` method
   - Add `buffers map[string]*EventBuffer` field (initialized in constructor)

2. Implement `TUIProgressEmitter`:
   - Struct: `program *tea.Program`, `runID string`
   - `EmitProgress(evt event.Event) error` — `program.Send(PipelineEventMsg{RunID: runID, Event: evt})`, return nil

3. Modify `Launch()`:
   - Replace `event.NewNDJSONEmitter()` with `event.NewProgressOnlyEmitter(tuiEmitter)`
   - Create `EventBuffer` with capacity 1000, store in `buffers[runID]`

4. Add buffer lifecycle methods:
   - `GetBuffer(runID string) *EventBuffer`
   - `HasBuffer(runID string) bool`
   - Extend `Cleanup()` to delete buffer

### Phase 6: Content Model Integration — Running Pipeline Focus

**Files**: `content.go`, `content_test.go`

1. Extend `cursorOnFocusableItem()` to include `itemKindRunning`:
   - Return true for `itemKindAvailable`, `itemKindFinished`, AND `itemKindRunning`

2. Modify Enter handling for running items:
   - Check `launcher.HasBuffer(runID)`:
     - `true`: emit `FocusChangedMsg` + `LiveOutputActiveMsg{Active: true}`, create LiveOutputModel in detail
     - `false`: emit `FocusChangedMsg` only (existing `stateRunningInfo` behavior)

3. Route `PipelineEventMsg`:
   - Forward to `PipelineDetailModel` which routes to `LiveOutputModel` if RunID matches

4. Handle `PipelineLaunchResultMsg` for completion:
   - After launcher cleanup, trigger list refresh for pipeline movement from Running to Finished

5. Route `ElapsedTickMsg` to list model.

6. Route `TransitionTimerMsg` to detail model.

7. Handle `LiveOutputActiveMsg` emission on Esc from live output.

### Phase 7: Pipeline Detail — stateRunningLive Integration

**Files**: `pipeline_detail.go`, `pipeline_detail_test.go`

1. Add `liveOutput *LiveOutputModel` field.

2. Handle `PipelineSelectedMsg` for running items with TUI buffer:
   - Create `LiveOutputModel` with buffer from launcher
   - Set `stateRunningLive`

3. Handle `PipelineEventMsg`:
   - Forward to `liveOutput.Update()` if in `stateRunningLive` and RunID matches

4. Handle `TransitionTimerMsg`:
   - If in `stateRunningLive` and RunID matches:
     - Transition to `stateLoading`, fetch finished detail
     - Emit `LiveOutputActiveMsg{Active: false}`
     - Clean up `liveOutput`

5. Handle Esc from `stateRunningLive`:
   - Emit `LiveOutputActiveMsg{Active: false}`
   - Emit `FocusChangedMsg{Pane: FocusPaneLeft}`

6. Render `stateRunningLive`: delegate to `liveOutput.View()`

### Phase 8: Elapsed Time Ticker

**Files**: `pipeline_list.go`, `pipeline_list_test.go`

1. Add `tickerActive bool` field.

2. Start ticker when `RunningCountMsg.Count > 0` (if not already active).

3. Stop ticker (don't re-issue) when count == 0.

4. Handle `ElapsedTickMsg`:
   - Simply return — the `View()` method re-renders with updated elapsed time (computed from `time.Since(r.StartedAt)`)
   - Re-issue tick cmd if running pipelines exist

5. Update `renderRunningItem()` to use `formatElapsed()` for `MM:SS`/`HH:MM:SS` format.

### Phase 9: Status Bar — Live Output Hints

**Files**: `statusbar.go`, `statusbar_test.go`

1. Add `liveOutputActive bool` field.

2. Handle `LiveOutputActiveMsg` in `Update()`.

3. Add hint state in `View()`:
   - When `liveOutputActive && focusPane == FocusPaneRight`:
     - Show: `"v: verbose  d: debug  o: output-only  ↑↓: scroll  Esc: back"`

### Phase 10: App Model — Message Forwarding and Program Reference

**Files**: `app.go`, `app_test.go`

1. Forward `LiveOutputActiveMsg` to status bar (alongside existing `FocusChangedMsg`, `FormActiveMsg`).

2. Modify `RunTUI()`:
   - After `tea.NewProgram()`, before `p.Run()`:
     - Call `model.content.launcher.SetProgram(p)` if launcher exists
   - This requires storing the model locally (currently passed inline)

### Phase 11: Test Suite

**Files**: All `*_test.go` files

1. `live_output_test.go` (NEW):
   - EventBuffer: append, capacity overflow, ordering
   - DisplayFlags: shouldFormat() for each flag combination
   - Event formatting: lifecycle, stream_activity, error block
   - Auto-scroll: pause on scroll, resume on bottom
   - Completion transition: timer start, deferred start, cancel on navigate
   - NO_COLOR support

2. `content_test.go`:
   - cursorOnFocusableItem() includes itemKindRunning
   - Enter on running TUI-launched pipeline focuses right pane with stateRunningLive
   - Enter on running external pipeline focuses right pane with stateRunningInfo
   - PipelineEventMsg routing

3. `pipeline_detail_test.go`:
   - stateRunningLive rendering
   - TransitionTimerMsg handling
   - Esc from live output

4. `pipeline_list_test.go`:
   - Elapsed tick start/stop
   - formatElapsed() format verification

5. `statusbar_test.go`:
   - LiveOutputActiveMsg hint switching

6. `pipeline_launcher_test.go`:
   - SetProgram(), HasBuffer(), GetBuffer()
   - Buffer cleanup on pipeline finish

## Complexity Tracking

_No constitution violations. No complexity tracking entries needed._
