# Tasks: Guided Workflow Orchestrator TUI

**Feature Branch**: `248-guided-workflow-tui`
**Generated**: 2026-03-16
**Spec**: `specs/248-guided-workflow-tui/spec.md`
**Plan**: `specs/248-guided-workflow-tui/plan.md`

---

## Phase 1: Setup & Foundation

- [X] T001 [P1] [US6] Add `GuidedState` enum type to `internal/tui/views.go` with four constants: `GuidedStateHealth`, `GuidedStateProposals`, `GuidedStateFleet`, `GuidedStateAttached`
- [X] T002 [P1] [US6] Add `GuidedMode bool` field to `LaunchDependencies` struct in `internal/tui/pipeline_messages.go`
- [X] T003 [P1] [US1] Add `guidedMode`, `guidedState`, and `healthDone` fields to `ContentModel` struct in `internal/tui/content.go`
- [X] T004 [P1] [US1] Modify `NewContentModel()` in `internal/tui/content.go` to accept `GuidedMode` from `LaunchDependencies` and when true: set initial `guidedState` to `GuidedStateHealth`, set `currentView` to `ViewHealth`, eagerly initialize `healthList` model

---

## Phase 2: Foundational — Health Phase Completion (US1: Health-First Startup)

- [X] T005 [P1] [US1] Add `HealthPhaseCompleteMsg` message type to `internal/tui/pipeline_messages.go` with `AllPassed bool` and `Summary string` fields
- [X] T006 [P1] [US1] Add `totalChecks` and `completedCount` fields to `HealthListModel` in `internal/tui/health_list.go`; initialize `totalChecks` from `len(provider.CheckNames())` in constructor
- [X] T007 [P1] [US1] Modify `HealthListModel.Update()` in `internal/tui/health_list.go` to increment `completedCount` on each `HealthCheckResultMsg` and emit `HealthPhaseCompleteMsg` when `completedCount >= totalChecks`
- [X] T008 [P1] [US1] Add `allPassed()` and `buildSummary()` helper methods to `HealthListModel` in `internal/tui/health_list.go` for constructing the completion message
- [X] T009 [P1] [US1] Handle `HealthPhaseCompleteMsg` in `ContentModel.Update()` in `internal/tui/content.go`: when `guidedMode` is true, transition `guidedState` to `GuidedStateProposals`, initialize suggest list/detail models, set `currentView` to `ViewSuggest`, emit `ViewChangedMsg`
- [X] T010 [P1] [US6] Set `GuidedMode: true` in the root command's `RunE` handler in `cmd/wave/main.go` (line ~60) when constructing `LaunchDependencies`
- [X] T011 [P1] [US1] Modify `ContentModel.Init()` in `internal/tui/content.go` to return `healthList.Init()` commands when `guidedMode` is true (start health checks immediately on startup)
- [X] T012 [P1] [US1] Write tests for health phase completion detection and auto-transition in `internal/tui/health_list_test.go` — verify `HealthPhaseCompleteMsg` emitted after all checks complete

---

## Phase 3: Tab Navigation in Guided Mode (US5: View State Machine)

- [X] T013 [P1] [US5] Modify Tab key handling in `ContentModel.Update()` in `internal/tui/content.go`: when `guidedMode` is true, toggle between `ViewSuggest` (Proposals) and `ViewPipelines` (Fleet) instead of cycling through 8 views
- [X] T014 [P1] [US5] Handle Tab during `GuidedStateHealth` in `ContentModel.Update()` — allow skipping to fleet view (`GuidedStateFleet`) with `currentView = ViewPipelines`
- [X] T015 [P] [P2] [US5] Add `guidedMode bool` field to `StatusBarModel` in `internal/tui/statusbar.go` and update hint text to show guided-mode-specific keybinding hints (Tab: proposals/fleet)
- [X] T016 [P] [P2] [US5] Wire `guidedMode` from `ContentModel` to `StatusBarModel` via `ViewChangedMsg` or initialization in `internal/tui/app.go`
- [X] T017 [P1] [US5] Write tests for guided-mode Tab toggling in `internal/tui/content_test.go` — verify Tab toggles between Proposals and Fleet, and Tab during health skips to fleet

---

## Phase 4: Proposals View Enhancements (US2: Pipeline Proposal Selection)

- [X] T018 [P1] [US2] Add `healthSummary string` field to `SuggestListModel` in `internal/tui/suggest_list.go` and render it as a header line in `View()` (e.g., "12 open issues, 3 PRs awaiting review, all deps OK")
- [X] T019 [P1] [US2] Add `skipped map[int]bool` field to `SuggestListModel` in `internal/tui/suggest_list.go`; handle `s` key to toggle skip state on focused proposal; render skipped proposals as dimmed
- [X] T020 [P1] [US2] Add `inputOverlay *textinput.Model` and `overlayTarget int` fields to `SuggestListModel` in `internal/tui/suggest_list.go`; handle `m` key to activate overlay prefilled with proposal input; handle Enter to confirm and Esc to cancel overlay
- [X] T021 [P1] [US2] Add `IsInputActive() bool` method to `SuggestListModel` returning `m.inputOverlay != nil` in `internal/tui/suggest_list.go`; wire into `ContentModel.IsInputActive()` to prevent `q` quit during input
- [X] T022 [P1] [US2] Add DAG preview rendering to `SuggestDetailModel.View()` in `internal/tui/suggest_detail.go` — for "sequence" type show `A → B → C` with artifact arrows; for "parallel" type show parallel columns
- [X] T023 [P2] [US2] Enhance empty state in `SuggestListModel.View()` in `internal/tui/suggest_list.go` — show "No pipeline recommendations" with hint for `n` (manual launch) and Tab (fleet view)
- [X] T024 [P1] [US2] Write tests for skip/modify keybindings in `internal/tui/suggest_list_test.go` — verify `s` toggles skip, `m` opens input overlay, Enter confirms modification
- [X] T025 [P1] [US2] Write tests for DAG preview rendering in `internal/tui/suggest_detail_test.go` — verify correct rendering for single, sequence, and parallel proposal types

---

## Phase 5: Fleet View Enhancements (US4: Fleet View with Archive Separation)

- [X] T026 [P2] [US4] Add `itemKindArchiveDivider` constant to `itemKind` enum in `internal/tui/pipeline_list.go`
- [X] T027 [P2] [US4] Modify `buildNavigableItems()` in `internal/tui/pipeline_list.go` to insert an archive divider `navigableItem` between the running and finished sections when `showArchiveDivider` is true
- [X] T028 [P2] [US4] Add `showArchiveDivider bool` field to `PipelineListModel` in `internal/tui/pipeline_list.go`; set to `true` when `guidedMode` is active (passed via construction or message)
- [X] T029 [P2] [US4] Render the archive divider in `PipelineListModel.View()` in `internal/tui/pipeline_list.go` — horizontal rule with "Archive" label, non-selectable (skip during cursor navigation)
- [X] T030 [P2] [US4] Add `sequenceGroups map[string][]string` field to `PipelineListModel` in `internal/tui/pipeline_list.go`; parse `compose:` prefix from `RunningPipeline` names to group sequence-linked runs visually
- [X] T031 [P] [P2] [US4] Render sequence grouping indicators in running section of `PipelineListModel.View()` — show pending (◌), running (●), completed (✓) for sequence members
- [X] T032 [P2] [US4] Write tests for archive divider insertion and sequence grouping in `internal/tui/pipeline_list_test.go`

---

## Phase 6: Multi-Select and Batch Launch (US3: Multi-Select and Batch Launch)

- [X] T033 [P2] [US3] Modify `SuggestListModel.Update()` in `internal/tui/suggest_list.go` to exclude skipped proposals from multi-select batch — when Enter is pressed with selections, filter out `skipped` indices before emitting `SuggestComposeMsg`
- [X] T034 [P2] [US3] Update selected count display in `SuggestListModel.View()` header in `internal/tui/suggest_list.go` — show "N proposals — M selected" when multi-select is active
- [X] T035 [P2] [US3] Update `SuggestDetailModel.View()` in `internal/tui/suggest_detail.go` to show combined execution plan when `multiSelected` contains 2+ proposals — list all pipelines with execution mode
- [X] T036 [P2] [US3] Write tests for batch launch with skip exclusion in `internal/tui/suggest_list_test.go`

---

## Phase 7: Launch Integration & View Transitions (US5 continued)

- [X] T037 [P1] [US5] Handle `SuggestLaunchMsg` in `ContentModel.Update()` in `internal/tui/content.go` — when `guidedMode`: launch pipeline via `PipelineLauncher`, transition `guidedState` to `GuidedStateFleet`, switch `currentView` to `ViewPipelines`
- [X] T038 [P1] [US5] Handle `SuggestComposeMsg` in `ContentModel.Update()` in `internal/tui/content.go` — when `guidedMode`: call `PipelineLauncher.LaunchSequence()` for multi-select batch, transition to fleet view
- [X] T039 [P2] [US5] Track `guidedState` transitions for attach/detach in `ContentModel.Update()` in `internal/tui/content.go` — set `GuidedStateAttached` when entering live output, restore `GuidedStateFleet` on Esc detach
- [X] T040 [P1] [US5] Write integration test for full guided flow cycle in `internal/tui/content_test.go` — Health → Proposals → Launch → Fleet → Attach → Detach → Proposals

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T041 [P1] [US6] Verify `wave run <pipeline>` does NOT activate guided mode in `cmd/wave/commands/run.go` — ensure `GuidedMode` is only set in root command, not in `run` subcommand
- [X] T042 [P] [P2] Update `StatusBarModel.View()` in `internal/tui/statusbar.go` to render context-appropriate hints per `guidedState` — Health: "Tab: skip to fleet", Proposals: "Enter: launch · Space: select · s: skip · m: modify · Tab: fleet", Fleet: "Enter: attach · Tab: proposals"
- [X] T043 [P] [P2] Handle terminal resize (`tea.WindowSizeMsg`) propagation to all guided-mode child models in `internal/tui/app.go` — ensure health, suggest, and pipeline models receive size updates
- [X] T044 [P2] Add health check timeout handling in `HealthListModel.Update()` in `internal/tui/health_list.go` — after 30s, show warning on hung checks and allow auto-transition with partial results
- [X] T045 [P1] Run `go test -race ./...` to verify no regressions (SC-003)
- [X] T046 [P1] Verify 80x24 minimum terminal rendering for all guided views (SC-009)
