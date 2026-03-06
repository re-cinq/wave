# Tasks: TUI Live Output Streaming

**Feature**: #257 — TUI Live Output Streaming
**Branch**: `257-tui-live-output`
**Generated**: 2026-03-06
**Spec**: `specs/257-tui-live-output/spec.md`
**Plan**: `specs/257-tui-live-output/plan.md`

---

## Phase 1: Setup & New Message Types

- [X] T001 [P1] [Setup] Add new message types to `pipeline_messages.go`: `PipelineEventMsg{RunID string, Event event.Event}`, `ElapsedTickMsg{}`, `TransitionTimerMsg{RunID string}`, `LiveOutputActiveMsg{Active bool}` — `internal/tui/pipeline_messages.go`
- [X] T002 [P1] [Setup] Add `stateRunningLive` constant to `DetailPaneState` enum after `stateRunningInfo` — `internal/tui/pipeline_messages.go`

## Phase 2: Foundational — EventBuffer and DisplayFlags

- [X] T003 [P1] [Foundation] Create `live_output.go` with `EventBuffer` struct: fixed-size `[]string` ring buffer with `head`, `count`, `capacity` fields. Implement `NewEventBuffer(capacity int)`, `Append(line string)`, `Lines() []string`, `Len() int` — `internal/tui/live_output.go`
- [X] T004 [P1] [Foundation] Add `DisplayFlags` struct to `live_output.go` with `Verbose`, `Debug`, `OutputOnly` bool fields — `internal/tui/live_output.go`
- [X] T005 [P1] [Foundation] Implement `shouldFormat(evt event.Event, flags DisplayFlags) bool` in `live_output.go`: Default mode shows `started`, `running`, `completed`, `failed`, `contract_validating`; Verbose adds `stream_activity`; Debug adds `step_progress`, `eta_updated`, `compaction_progress`; OutputOnly overrides to show only `completed`/`failed` — `internal/tui/live_output.go`
- [X] T006 [P1] [Foundation] Implement `formatEventLine(evt event.Event) string` in `live_output.go`: format lifecycle events as `[stepID] Starting... (persona: X, model: Y)`, completions as `[stepID] ✓ Completed (Ns)`, stream_activity as `[stepID] ToolName ToolTarget`, debug as `[stepID] ♡ heartbeat (tokens: N/M)`, and respect `NO_COLOR` env var — `internal/tui/live_output.go`
- [X] T007 [P1] [Foundation] Implement `formatErrorBlock(evt event.Event) string` in `live_output.go`: render failure as multi-line block with `✗ Pipeline failed`, step ID, failure reason, remediation, recovery hints — `internal/tui/live_output.go`
- [X] T008 [P1] [Foundation] Implement `formatElapsed(d time.Duration) string` in `live_output.go`: returns `MM:SS` for < 1h, `HH:MM:SS` for >= 1h — `internal/tui/live_output.go`
- [X] T009 [P] [P1] [Foundation] Create `live_output_test.go` with tests: EventBuffer append, capacity overflow, ordering; DisplayFlags shouldFormat for all flag combinations; formatEventLine for each event state; formatErrorBlock; formatElapsed; NO_COLOR support — `internal/tui/live_output_test.go`

## Phase 3: User Story 1 — View Live Output (P1)

- [X] T010 [P1] [US-1] Implement `LiveOutputModel` struct in `live_output.go` with fields: `runID`, `pipelineName`, `width`, `height`, `buffer *EventBuffer`, `viewport viewport.Model`, `autoScroll bool`, `flags DisplayFlags`, `currentStep string`, `stepNumber int`, `totalSteps int`, `model string`, `startedAt time.Time`, `completed bool`, `completionPending bool` — `internal/tui/live_output.go`
- [X] T011 [P1] [US-1] Implement `NewLiveOutputModel(runID, pipelineName string, buffer *EventBuffer, startedAt time.Time, totalSteps int) LiveOutputModel` constructor with `autoScroll: true` default — `internal/tui/live_output.go`
- [X] T012 [P1] [US-1] Implement `LiveOutputModel.SetSize(w, h int)`: allocate 3 lines for header, 2 lines for footer, remainder for viewport. Set `viewport.Width` and `viewport.Height` accordingly — `internal/tui/live_output.go`
- [X] T013 [P1] [US-1] Implement `LiveOutputModel.View()`: render three-part layout — header (pipeline name, "Running (step N/M: stepID)", elapsed time, model), viewport content from buffer, footer (display flags state, auto-scroll indicator) — `internal/tui/live_output.go`
- [X] T014 [P1] [US-1] Implement `LiveOutputModel.Update(msg tea.Msg)` for `PipelineEventMsg`: call `shouldFormat()`, if true call `formatEventLine()`/`formatErrorBlock()`, `buffer.Append()`, update viewport content from `buffer.Lines()`, if `autoScroll` then `viewport.GotoBottom()`. Update `currentStep`/`stepNumber` on `started` events — `internal/tui/live_output.go`
- [X] T015 [P1] [US-1] Add `program *tea.Program` and `buffers map[string]*EventBuffer` fields to `PipelineLauncher`. Implement `SetProgram(p *tea.Program)`, `GetBuffer(runID string) *EventBuffer`, `HasBuffer(runID string) bool`. Initialize `buffers` map in `NewPipelineLauncher()` — `internal/tui/pipeline_launcher.go`
- [X] T016 [P1] [US-1] Add `TUIProgressEmitter` struct to `pipeline_launcher.go` implementing `event.ProgressEmitter` with `program *tea.Program` and `runID string`. `EmitProgress(evt event.Event) error` calls `program.Send(PipelineEventMsg{RunID: runID, Event: evt})` — `internal/tui/pipeline_launcher.go`
- [X] T017 [P1] [US-1] Modify `PipelineLauncher.Launch()`: replace `event.NewNDJSONEmitter()` with `event.NewProgressOnlyEmitter(tuiEmitter)` where `tuiEmitter` is a `TUIProgressEmitter`. Create `EventBuffer` with capacity 1000 and store in `buffers[runID]` — `internal/tui/pipeline_launcher.go`
- [X] T018 [P1] [US-1] Extend `PipelineLauncher.Cleanup()` to also delete the buffer entry from `buffers` map — `internal/tui/pipeline_launcher.go`
- [X] T019 [P1] [US-1] Extend `cursorOnFocusableItem()` in `content.go` to return true for `itemKindRunning` in addition to `itemKindAvailable` and `itemKindFinished` — `internal/tui/content.go`
- [X] T020 [P1] [US-1] Modify Enter key handling in `content.go` for running items: check `launcher.HasBuffer(runID)` — if true, emit `FocusChangedMsg{Pane: FocusPaneRight}` + `LiveOutputActiveMsg{Active: true}` and pass buffer info to detail; if false, emit only `FocusChangedMsg` (existing `stateRunningInfo` behavior) — `internal/tui/content.go`
- [X] T021 [P1] [US-1] Route `PipelineEventMsg` in `content.go` Update: forward to `detail.Update()` — `internal/tui/content.go`
- [X] T022 [P1] [US-1] Add `liveOutput *LiveOutputModel` and `launcher *PipelineLauncher` fields to `PipelineDetailModel`. Handle `PipelineSelectedMsg` for running items with TUI buffer: create `LiveOutputModel`, set `stateRunningLive`. Forward `PipelineEventMsg` to `liveOutput.Update()` if in `stateRunningLive` and RunID matches — `internal/tui/pipeline_detail.go`
- [X] T023 [P1] [US-1] Add `stateRunningLive` rendering to `PipelineDetailModel.View()`: delegate to `liveOutput.View()` — `internal/tui/pipeline_detail.go`
- [X] T024 [P1] [US-1] Handle Esc from `stateRunningLive` in `PipelineDetailModel.Update()`: emit `LiveOutputActiveMsg{Active: false}` + `FocusChangedMsg{Pane: FocusPaneLeft}`, nil out `liveOutput` — `internal/tui/pipeline_detail.go`
- [X] T025 [P1] [US-1] Modify `RunTUI()` in `app.go`: store model locally, after `tea.NewProgram()` call `model.content.launcher.SetProgram(p)` if launcher exists, before `p.Run()` — `internal/tui/app.go`
- [X] T026 [P1] [US-1] Forward `LiveOutputActiveMsg` to status bar in `AppModel.Update()` alongside existing `FocusChangedMsg`, `FormActiveMsg` — `internal/tui/app.go`
- [X] T027 [P] [P1] [US-1] Add tests for `LiveOutputModel`: constructor defaults, SetSize viewport allocation, Update with PipelineEventMsg formats and appends to buffer, View renders header/viewport/footer, step progress tracking updates header — `internal/tui/live_output_test.go`
- [X] T028 [P] [P1] [US-1] Add tests for `PipelineLauncher`: SetProgram, HasBuffer, GetBuffer, TUIProgressEmitter.EmitProgress, buffer cleanup on Cleanup() — `internal/tui/pipeline_launcher_test.go`
- [X] T029 [P] [P1] [US-1] Add tests for content.go: cursorOnFocusableItem includes itemKindRunning, Enter on running TUI-launched pipeline creates stateRunningLive, Enter on running external pipeline keeps stateRunningInfo, PipelineEventMsg routing — `internal/tui/content_test.go`
- [X] T030 [P] [P1] [US-1] Add tests for pipeline_detail.go: stateRunningLive rendering delegates to liveOutput.View(), Esc emits LiveOutputActiveMsg{false}, PipelineEventMsg forwarded to liveOutput — `internal/tui/pipeline_detail_test.go`

## Phase 4: User Story 2 — Toggle Display Flags (P1)

- [X] T031 [P1] [US-2] Implement key handling in `LiveOutputModel.Update()` for `v`, `d`, `o` keys: toggle `flags.Verbose`, `flags.Debug`, `flags.OutputOnly` respectively. Only active when right pane is focused (already gated by detail model forwarding) — `internal/tui/live_output.go`
- [X] T032 [P1] [US-2] Render display flag state in LiveOutputModel footer: `[v] verbose  [ ] debug  [ ] output-only` with active/inactive indicators — `internal/tui/live_output.go`
- [X] T033 [P] [P1] [US-2] Add tests for display flag toggles: v/d/o keypress changes flag state, output-only overrides verbose/debug at format stage, footer reflects current state, flags ignored when left pane focused (verified via content_test.go routing) — `internal/tui/live_output_test.go`

## Phase 5: User Story 3 — Pipeline Completion Transition (P1)

- [X] T034 [P1] [US-3] Implement terminal event handling in `LiveOutputModel.Update()`: on `completed` event, append summary "✓ Pipeline completed in [duration]", set `completed = true`; on `failed` event, append error block, set `completed = true`. If `autoScroll` true, return `tea.Tick(2*time.Second, TransitionTimerMsg{RunID})` cmd; if false, set `completionPending = true` — `internal/tui/live_output.go`
- [X] T035 [P1] [US-3] Handle `TransitionTimerMsg` in `PipelineDetailModel.Update()`: if in `stateRunningLive` and RunID matches, transition to `stateLoading`, fetch finished detail, emit `LiveOutputActiveMsg{Active: false}`, clean up `liveOutput` — `internal/tui/pipeline_detail.go`
- [X] T036 [P1] [US-3] Handle deferred transition: when auto-scroll resumes in `LiveOutputModel` (user scrolls to bottom after completion with `completionPending = true`), start the 2-second `tea.Tick()` transition timer — `internal/tui/live_output.go`
- [X] T037 [P1] [US-3] Cancel transition when user navigates away: on Esc or pipeline change in detail model, if `liveOutput != nil`, don't propagate pending TransitionTimerMsg (verify RunID match in handler) — `internal/tui/pipeline_detail.go`
- [X] T038 [P1] [US-3] Route `TransitionTimerMsg` from content.go to detail model — `internal/tui/content.go`
- [X] T039 [P1] [US-3] On `PipelineLaunchResultMsg` in content.go, trigger data refresh so the pipeline moves from Running to Finished section — `internal/tui/content.go`
- [X] T040 [P] [P1] [US-3] Add tests: completion summary line appended, 2s timer started when autoScroll true, timer deferred when autoScroll false, timer starts on scroll-to-bottom, transition cancelled on navigate away, RunID mismatch ignored — `internal/tui/live_output_test.go`, `internal/tui/pipeline_detail_test.go`

## Phase 6: User Story 4 — Scroll Through Live Output (P2)

- [X] T041 [P2] [US-4] Implement scroll key handling in `LiveOutputModel.Update()`: on ↑/↓/PgUp/PgDn, set `autoScroll = false`, forward to viewport, then check `viewport.AtBottom()` — if true, set `autoScroll = true` — `internal/tui/live_output.go`
- [X] T042 [P2] [US-4] Render auto-scroll paused indicator in footer: "⏸ Scrolling paused — scroll to bottom to resume" when `autoScroll` is false — `internal/tui/live_output.go`
- [X] T043 [P] [P2] [US-4] Add tests: scroll up pauses auto-scroll, new events don't move viewport when paused, scroll to bottom resumes, indicator shown/hidden, buffer/scroll state preserved per-pipeline — `internal/tui/live_output_test.go`

## Phase 7: User Story 5 — Pipeline Failure Display (P2)

- [X] T044 [P2] [US-5] Implement styled error block rendering in `formatErrorBlock()`: red `✗ Pipeline failed` header, failing step `(step ID, persona)`, failure reason from `FailureReason`, remediation from `Remediation`, recovery hints from `RecoveryHints[]` with `→` prefix — `internal/tui/live_output.go`
- [X] T045 [P2] [US-5] Ensure failed event triggers same completion transition flow as completed event (summary → 2s delay → finished detail) — `internal/tui/live_output.go`
- [X] T046 [P] [P2] [US-5] Add tests: error block rendering with all fields populated, error block with missing optional fields, transition fires after failure, left pane shows failed status — `internal/tui/live_output_test.go`

## Phase 8: User Story 6 — Left Pane Elapsed Time (P2)

- [X] T047 [P2] [US-6] Add `tickerActive bool` field to `PipelineListModel`. On `RunningCountMsg` with Count > 0 and `!tickerActive`, start 1-second `tea.Tick()` returning `ElapsedTickMsg{}`, set `tickerActive = true`. On Count == 0, set `tickerActive = false` — `internal/tui/pipeline_list.go`
- [X] T048 [P2] [US-6] Handle `ElapsedTickMsg` in `PipelineListModel.Update()`: if running pipelines exist, re-issue tick cmd (the View already re-renders with updated `time.Since(r.StartedAt)`). If none running, don't re-issue — `internal/tui/pipeline_list.go`
- [X] T049 [P2] [US-6] Update `renderRunningItem()` to use `formatElapsed()` from `live_output.go` for `MM:SS`/`HH:MM:SS` format instead of current `formatDuration()` compact format — `internal/tui/pipeline_list.go`
- [X] T050 [P2] [US-6] Route `ElapsedTickMsg` in content.go to list model — `internal/tui/content.go`
- [X] T051 [P] [P2] [US-6] Add tests: ticker starts when running count > 0, ticker stops when count == 0, elapsed time format MM:SS and HH:MM:SS, ElapsedTickMsg re-issues tick when running — `internal/tui/pipeline_list_test.go`

## Phase 9: Status Bar — Live Output Hints

- [X] T052 [P2] [StatusBar] Add `liveOutputActive bool` field to `StatusBarModel`. Handle `LiveOutputActiveMsg` in `Update()` — `internal/tui/statusbar.go`
- [X] T053 [P2] [StatusBar] Add hint state in `View()`: when `liveOutputActive && focusPane == FocusPaneRight`, show `"v: verbose  d: debug  o: output-only  ↑↓: scroll  Esc: back"` — `internal/tui/statusbar.go`
- [X] T054 [P] [P2] [StatusBar] Add tests: LiveOutputActiveMsg sets state, hint text switches to live output hints, hint text reverts when active=false — `internal/tui/statusbar_test.go`

## Phase 10: Polish & Cross-Cutting

- [X] T055 [P2] [Polish] Handle externally-started running pipelines (C7): when `stateRunningInfo` is shown for a running pipeline without a buffer, update the info message to include "Started externally — live output not available." and show elapsed time — `internal/tui/pipeline_detail.go`
- [X] T056 [P2] [Polish] Handle empty event buffer edge case: when a TUI-launched pipeline is selected before any events arrive, show "Waiting for events..." in the viewport area — `internal/tui/live_output.go`
- [X] T057 [P2] [Polish] Ensure terminal resize (`tea.WindowSizeMsg`) propagates to `LiveOutputModel.SetSize()` via detail model — `internal/tui/pipeline_detail.go`
- [X] T058 [P2] [Polish] Verify `NO_COLOR` support: all styled output in `formatEventLine()`, `formatErrorBlock()`, and `LiveOutputModel.View()` degrades to plain text when `NO_COLOR` is set — `internal/tui/live_output.go`
- [X] T059 [P1] [Polish] Run `go test ./...` and `go test -race ./...` to verify all existing tests pass and no race conditions exist — all test files
- [X] T060 [P] [P2] [Polish] Add integration-level tests: verify app.go forwards LiveOutputActiveMsg, verify content.go routes all new message types correctly end-to-end — `internal/tui/app_test.go`, `internal/tui/content_test.go`
