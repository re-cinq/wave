# Tasks: Guided TUI Orchestrator

**Feature Branch**: `248-guided-tui-orchestrator`
**Generated**: 2026-03-16
**Spec**: `specs/248-guided-tui-orchestrator/spec.md`
**Plan**: `specs/248-guided-tui-orchestrator/plan.md`

---

## Phase 1: Setup

- [X] T001 [P1] Create `internal/tui/guided_flow.go` — Define `GuidedFlowPhase` enum (`GuidedPhaseHealth`, `GuidedPhaseProposals`, `GuidedPhaseFleet`, `GuidedPhaseAttached`) and `GuidedFlowState` struct with fields: `Phase`, `HealthComplete`, `HasErrors`, `UserConfirmed`, `TransitionTimer`. Add phase transition methods: `TransitionToProposals()`, `TransitionToFleet()`, `TransitionToAttached()`, `DetachToFleet()`. Add helper `IsGuided() bool` and `TabTarget() ViewType` that returns the toggle destination based on current phase.

- [X] T002 [P1] Create `internal/tui/guided_messages.go` — Define message types: `HealthAllCompleteMsg{HasErrors bool}`, `HealthTransitionMsg{}`, `HealthContinueMsg{}`, `SuggestModifyMsg{Pipeline SuggestProposedPipeline}`. These are the Bubble Tea messages that drive guided flow state transitions.

- [X] T003 [P1] [P] Create `internal/tui/guided_flow_test.go` — Table-driven tests for `GuidedFlowState` transitions: Health→Proposals (no errors), Health→Proposals (with errors + UserConfirmed), Proposals→Fleet (on launch), Fleet→Attached (on Enter), Attached→Fleet (on Esc). Test `TabTarget()` returns correct view for each phase. Test that transitions are idempotent.

---

## Phase 2: Foundational — Health Completion Tracking (Story 1 prerequisite)

- [X] T004 [P1] [Story1] Modify `internal/tui/health_list.go` — Add completion tracking to `HealthListModel.Update()`: after processing each `HealthCheckResultMsg`, count checks where `Status != HealthCheckChecking`. When `completedCount == len(m.checks)`, return a cmd that emits `HealthAllCompleteMsg{HasErrors: hasErrors}` where `hasErrors` is true if any check has `Status == HealthCheckErr`.

- [X] T005 [P1] [Story1] [P] Add tests for health completion tracking in `internal/tui/health_list.go` (new test file or extend existing) — Test: all checks OK → emits `HealthAllCompleteMsg{HasErrors: false}`. Test: one check error → emits `HealthAllCompleteMsg{HasErrors: true}`. Test: partial completion → no message emitted. Test: re-run (`r` key) resets completion tracking.

---

## Phase 3: Story 1 — Health-First Startup (P1)

- [X] T006 [P1] [Story1] Modify `internal/tui/app.go` — Add `Guided bool` field to `AppModel`. Update `NewAppModel()` to accept a `guided bool` parameter and pass it through to `NewContentModel()`. When `guided` is true, set it on the `ContentModel`.

- [X] T007 [P1] [Story1] Modify `internal/tui/content.go` — Add `guidedFlow *GuidedFlowState` field to `ContentModel`. Update `NewContentModel()` to accept and store the guided flag. When guided, set `guidedFlow = &GuidedFlowState{Phase: GuidedPhaseHealth}`.

- [X] T008 [P1] [Story1] Modify `internal/tui/content.go` (`Init()`) — When `guidedFlow != nil`, override startup: set `currentView = ViewHealth`, lazy-create `healthList` and `healthDetail` models, and return their `Init()` commands (starts async health checks). When `guidedFlow == nil`, preserve existing behavior unchanged.

- [X] T009 [P1] [Story1] Modify `internal/tui/content.go` (`Update()`) — Add handler for `HealthAllCompleteMsg`: if no errors, set `guidedFlow.TransitionTimer = true` and return `tea.Tick(1*time.Second, func(time.Time) tea.Msg { return HealthTransitionMsg{} })`. If errors, stay on health view and set `guidedFlow.HasErrors = true`. Add handler for `HealthTransitionMsg`: call `guidedFlow.TransitionToProposals()`, switch `currentView` to `ViewSuggest`, lazy-create suggest models, return init commands. Add handler for `HealthContinueMsg`: set `guidedFlow.UserConfirmed = true`, start transition timer.

- [X] T010 [P1] [Story1] Modify `internal/tui/health_detail.go` — When displayed in guided mode and health errors exist (`HealthAllCompleteMsg` received with errors), append a prompt section: "Some checks failed. Press y to continue or q to quit." Handle `y` key → emit `HealthContinueMsg{}`.

- [X] T011 [P1] [Story1] Modify `cmd/wave/main.go` — In the `shouldLaunchTUI` path (line ~59), set `deps.Guided = true` (or pass it as a separate parameter). The `LaunchDependencies` struct needs a new `Guided bool` field. This ensures `wave` (no subcommand) activates guided mode while `wave run` does not.

- [X] T012 [P1] [Story1] [P] Add guided startup tests in `internal/tui/content_test.go` — Test: guided mode starts at `ViewHealth`. Test: non-guided mode starts at `ViewPipelines` (regression). Test: `HealthAllCompleteMsg{HasErrors: false}` triggers timer. Test: `HealthTransitionMsg` switches to `ViewSuggest`. Test: `HealthAllCompleteMsg{HasErrors: true}` does NOT auto-transition. Test: `HealthContinueMsg` after errors triggers transition timer.

---

## Phase 4: Story 2 — Pipeline Proposal Selection (P1)

- [X] T013 [P1] [Story2] Modify `internal/tui/suggest_list.go` — Add `s` key handler: when pressed, remove the proposal at `m.cursor` from `m.proposals`, adjust cursor and selected map, emit updated selection. This implements "skip/dismiss" (FR-007).

- [X] T014 [P1] [Story2] Modify `internal/tui/suggest_list.go` — Add `m` key handler: when pressed on a valid proposal, emit `SuggestModifyMsg{Pipeline: m.proposals[m.cursor]}` to request input modification before launch (FR-007).

- [X] T015 [P1] [Story2] Modify `internal/tui/content.go` — Add handler for `SuggestModifyMsg`: show an input editor overlay. Reuse the existing `textinput.Model` pattern from `PipelineDetailModel` configuring form. Pre-populate with the proposal's `Input` field. On submit, launch with modified input via `SuggestLaunchMsg`. On cancel (Esc), return to suggest list.

- [X] T016 [P1] [Story2] [P] Add tests for suggest list key handlers in `internal/tui/suggest_list_test.go` — Test: `s` key removes proposal from list. Test: `s` on last proposal → empty list shows "No suggestions available". Test: `m` key emits `SuggestModifyMsg` with correct pipeline. Test: multi-select + Enter still emits `SuggestComposeMsg`. Test: cursor adjustment after dismiss.

---

## Phase 5: Story 7 — Non-Regression Guard (P1)

- [X] T017 [P1] [Story7] Add regression tests in `internal/tui/content_test.go` — Test: when `guidedFlow == nil`, Tab cycles through all 8 views via `cycleView()`. Test: when `guidedFlow == nil`, Init() starts at `ViewPipelines`. Test: `SuggestLaunchMsg` handling is unchanged (already switches to ViewPipelines and launches). These tests MUST pass with zero changes to pipeline execution code.

- [X] T018 [P1] [Story7] Add regression test in `cmd/wave/commands/run_test.go` or `cmd/wave/main_test.go` — Verify that `wave run <pipeline>` does NOT set `Guided = true`. Verify that `shouldLaunchTUI` returns false when a subcommand is provided.

---

## Phase 6: Story 5 — View Toggle Navigation (P2)

- [X] T019 [P2] [Story5] Modify `internal/tui/content.go` (Tab handler) — When `guidedFlow != nil`, replace `cycleView()` with guided toggle: if `currentView == ViewSuggest`, switch to `ViewPipelines`; if `currentView == ViewPipelines`, switch to `ViewSuggest`. Block Tab when in attached live output mode (`guidedFlow.Phase == GuidedPhaseAttached`). Shift+Tab reverses the toggle. Lazy-create target view models as needed.

- [X] T020 [P2] [Story5] Modify `internal/tui/content.go` — Add number key `1`–`8` direct-jump navigation. In the `tea.KeyMsg` handler, when `msg.String()` is `"1"` through `"8"`, map to the corresponding `ViewType` (1→ViewPipelines, 2→ViewPersonas, ..., 8→ViewSuggest). Call the appropriate lazy-init + focus logic (extract from `cycleView()` into a `setView(v ViewType) tea.Cmd` helper). Only activate when `focus == FocusPaneLeft` and no input/form is active.

- [X] T021 [P2] [Story5] [P] Add tab navigation tests in `internal/tui/content_test.go` — Test: guided Tab from Suggest → Pipelines. Test: guided Tab from Pipelines → Suggest. Test: guided Tab during attached → no-op. Test: Shift+Tab reverses. Test: number key `3` → ViewContracts in guided mode. Test: non-guided Tab still cycles all 8 views.

---

## Phase 7: Story 3 — DAG Preview (P2)

- [X] T022 [P2] [Story3] Create `internal/tui/suggest_dag.go` — Implement `RenderDAG(proposal SuggestProposedPipeline) string` function. For sequence proposals (`Type == "sequence"`), render: `[pipeline-a] ──→ [pipeline-b] ──→ [pipeline-c]` with box-drawing characters, one step per line with vertical arrows. For parallel proposals (`Type == "parallel"`), render stacked: `┌ [pipeline-a]` / `├ [pipeline-b]` / `└ [pipeline-c]` with `(concurrent)` label. For single proposals, return empty string (no DAG needed).

- [X] T023 [P2] [Story3] Modify `internal/tui/suggest_detail.go` — In `View()`, after rendering proposal details, if the selected proposal has `Type == "sequence"` or `Type == "parallel"`, call `RenderDAG()` and append the result below the existing detail text. For multi-selected proposals, render a combined execution plan showing staging.

- [X] T024 [P2] [Story3] [P] Create `internal/tui/suggest_dag_test.go` — Table-driven tests: single proposal → empty string. Sequence with 2 pipelines → correct arrow layout. Sequence with 3 pipelines → multi-step arrows. Parallel with 2 pipelines → grouped layout. Mixed multi-select (sequence + single) → staged layout. Empty sequence list → graceful fallback.

---

## Phase 8: Story 4 — Fleet Monitoring with Archive Separation (P2)

- [X] T025 [P2] [Story4] Modify `internal/tui/pipeline_list.go` — Add `itemKindDivider` to the `itemKind` enum. Add `guided bool` field to `PipelineListModel`. When `guided == true`, modify `buildNavigableItems()` to use archive layout: running runs first (all pipelines, flat), then a divider item with label "─── Archive ───", then completed/failed runs. The divider is a non-selectable navigable item (cursor skips it).

- [X] T026 [P2] [Story4] Modify `internal/tui/pipeline_list.go` (`View()`) — Render `itemKindDivider` items as a horizontal rule line using the full pane width. Style with dimmed foreground. Ensure cursor navigation skips divider items (adjust `handleKeyMsg` up/down to skip over dividers).

- [X] T027 [P2] [Story4] Modify `internal/tui/pipeline_provider.go` — Add `SequenceGroup string` field to both `RunningPipeline` and `FinishedPipeline` structs. This will be populated from the compose group run ID stored in the state store.

- [X] T028 [P2] [Story4] Modify `internal/tui/pipeline_list.go` — When `guided == true` and runs have `SequenceGroup` set, visually group them using tree connectors (`├─`, `└─`) under a group header showing the sequence name. Group headers are `itemKindPipelineName` items.

- [X] T029 [P2] [Story4] Modify `internal/tui/content.go` — When entering live output (`Enter` on a running pipeline in guided mode), set `guidedFlow.Phase = GuidedPhaseAttached`. When exiting live output (`Esc`), set `guidedFlow.Phase = GuidedPhaseFleet`. This enables Tab blocking during attachment.

- [X] T030 [P2] [Story4] [P] Add archive divider tests in `internal/tui/pipeline_list_test.go` — Test: guided mode with running + finished → divider between them. Test: guided mode all running → no divider. Test: guided mode all finished → no divider (or divider at top). Test: cursor skips divider items. Test: non-guided mode → no divider (existing tree layout). Test: sequence grouping renders tree connectors.

---

## Phase 9: Story 6 — Sequence/Parallel Execution Wiring (P3)

- [X] T031 [P3] [Story6] Modify `internal/tui/content.go` — Enhance `SuggestLaunchMsg` handler: check `Pipeline.Type`. If `"single"`, use existing launch path. If `"sequence"`, call `launcher.LaunchSequence()` with the proposal's `Sequence` pipeline names. If `"parallel"`, call `launcher.LaunchParallel()` with the pipeline names. Update `guidedFlow.Phase` to `GuidedPhaseFleet` after launch.

- [X] T032 [P3] [Story6] Modify `internal/tui/content.go` — Enhance `SuggestComposeMsg` handler: for multi-selected proposals, determine execution strategy. Group sequences together, launch parallels concurrently. Switch to `ViewPipelines` and update `guidedFlow.Phase` to `GuidedPhaseFleet`.

- [X] T033 [P3] [Story6] [P] Add sequence launch tests — Test: `SuggestLaunchMsg` with `Type="sequence"` calls `LaunchSequence`. Test: `SuggestLaunchMsg` with `Type="parallel"` calls `LaunchParallel`. Test: `SuggestLaunchMsg` with `Type="single"` uses existing path. Test: `SuggestComposeMsg` with mixed types handles each correctly.

---

## Phase 10: Polish & Cross-Cutting

- [X] T034 [P2] [P] Modify `internal/tui/statusbar.go` — Add guided-mode hint branches. When `guidedMode` is active: Health view → "Tab: skip to proposals  r: re-run  q: quit". Proposals view → "Enter: launch  Space: select  m: modify  s: skip  Tab: fleet". Fleet view → "Enter: attach  Tab: proposals  c: cancel". Attached → "Esc: detach". Add `guidedMode bool` field to `StatusBarModel` and wire from `ContentModel`.

- [X] T035 [P2] [P] Add statusbar guided mode tests in `internal/tui/statusbar_test.go` — Test: guided health view shows correct hints. Test: guided proposals view shows correct hints. Test: guided fleet view shows correct hints. Test: non-guided mode hints are unchanged.

- [X] T036 [P1] Run `go test -race ./...` — Verify all existing tests pass. Verify all new tests pass. Zero test regressions. This is the final gate before the feature is considered complete.

- [X] T037 [P2] [P] Modify `internal/tui/content.go` — Handle edge case: zero proposals. When `SuggestDataMsg` arrives with empty proposals list in guided mode, display "No proposals available — run pipelines manually with `wave run`" message in the suggest list view (already handled by existing empty-state rendering in `suggest_list.go`, verify it works in guided mode context).

- [X] T038 [P2] [P] Handle terminal resize in guided mode — Verify `tea.WindowSizeMsg` propagates correctly to all views in guided mode. The existing `SetSize()` cascade in `content.go` should handle this, but verify no rendering corruption when resizing during health phase or proposals phase. Add a test if needed.
