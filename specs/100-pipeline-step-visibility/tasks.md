# Tasks: Pipeline Step Visibility in Default Run Mode

**Feature Branch**: `100-pipeline-step-visibility`
**Generated**: 2026-02-13
**Spec**: `specs/100-pipeline-step-visibility/spec.md`
**Plan**: `specs/100-pipeline-step-visibility/plan.md`

## Phase 1: Data Model Foundation

These tasks add the `StepPersonas` field and wire it through both rendering paths. All subsequent phases depend on this.

- [X] T001 [P1] [US2] Add `StepPersonas map[string]string` field to `PipelineContext` struct in `internal/display/types.go:192`. Add the field after `StepDurations` (line 227), following the same comment style as `StepDurations`. The field maps stepID to persona name.

- [X] T002 [P1] [US2] Populate `StepPersonas` in `BubbleTeaProgressDisplay.toPipelineContext()` in `internal/display/bubbletea_progress.go:268`. Build a `stepPersonas` map by iterating `btpd.steps` and extracting each `step.Persona`. Include it in the returned `PipelineContext` struct literal (after `StepDurations` at line 359).

- [X] T003 [P1] [US2] Populate `StepPersonas` in `ProgressDisplay.toPipelineContext()` in `internal/display/progress.go:436`. Build a `stepPersonas` map by iterating `pd.steps` and extracting each `step.Persona`. Include it in the returned `PipelineContext` struct literal (after line 488 where `StepStatuses` is set).

- [X] T004 [P1] [US2] Update `CreatePipelineContext()` in `internal/display/progress.go:638` to accept a `stepPersonas map[string]string` parameter and store it in the returned `PipelineContext`. Update all call sites (search for `CreatePipelineContext(` across the repo).

- [X] T005 [P1] [US2] Add `StepPersonas` field test to `internal/display/types_test.go`. In the existing `TestPipelineContext_Structure` test (line 340), add `StepPersonas` to the struct literal and verify it is populated correctly. Also add a standalone test `TestPipelineContext_StepPersonas` that creates a context with multiple step personas and validates the mapping.

## Phase 2: BubbleTea All-Step Rendering (US1 + US2 + US3)

These tasks rewrite the primary rendering path. Depends on Phase 1 (T001-T002).

- [X] T006 [P1] [US1] Rewrite `renderCurrentStep()` in `internal/display/bubbletea_model.go:247` to iterate ALL steps in `m.ctx.StepOrder` instead of only collecting completed + running steps. For each stepID, read `m.ctx.StepStatuses[stepID]` and render based on state. Remove the separate `completedSteps` and `currentStep` variables. Replace the blank line separator between completed and running steps with a continuous list. Keep the existing deliverable tree rendering for completed steps and tool activity rendering for the running step.

- [X] T007 [P1] [US1] Within the rewritten `renderCurrentStep()` in `internal/display/bubbletea_model.go`, add rendering for `StateNotStarted` steps. Render as `○ stepID (persona)` using lipgloss muted color (Color("244")). Use `m.ctx.StepPersonas[stepID]` for the persona name. Only show the persona in parentheses if the persona string is non-empty.

- [X] T008 [P1] [US2] Within the rewritten `renderCurrentStep()` in `internal/display/bubbletea_model.go`, update completed step rendering (currently line 275) to include persona name. Change from `✓ stepID (duration)` to `✓ stepID (persona) (duration)`. Use `m.ctx.StepPersonas[stepID]` for the persona name. Maintain the existing bright cyan color (Color("12")).

- [X] T009 [P2] [US3] Within the rewritten `renderCurrentStep()` in `internal/display/bubbletea_model.go`, ensure the running step shows live elapsed time. This is the existing behavior at line 309-311 — preserve it. Also ensure completed steps show final static duration from `m.ctx.StepDurations[stepID]` (existing behavior at line 270-274). No changes needed if the rewrite preserves these patterns.

- [X] T010 [P3] [US4] Within the rewritten `renderCurrentStep()` in `internal/display/bubbletea_model.go`, add rendering for `StateSkipped` steps. Render as `— stepID (persona)` using lipgloss muted color (Color("244")). Use the em dash character `—` (U+2014), not a hyphen. No timing information for skipped steps.

- [X] T011 [P3] [US5] Within the rewritten `renderCurrentStep()` in `internal/display/bubbletea_model.go`, add rendering for `StateFailed` steps. Render as `✗ stepID (persona) (duration)` using lipgloss red color (Color("9")). Show final duration from `m.ctx.StepDurations[stepID]` if available. Use `✗` (U+2717) for the cross mark.

- [X] T012 [P3] [US5] Within the rewritten `renderCurrentStep()` in `internal/display/bubbletea_model.go`, add rendering for `StateCancelled` steps. Render as `⊛ stepID (persona)` using lipgloss warning/yellow color (Color("11")). Use `⊛` (U+229B) for the circled asterisk indicator. No timing information for cancelled steps.

## Phase 3: Dashboard All-Step Rendering (US1 + US2)

These tasks update the secondary Dashboard rendering path. Depends on Phase 1 (T001, T003). Can be done in parallel [P] with Phase 2.

- [X] T013 [P1] [US1] Rewrite `renderStepStatusPanel()` in `internal/display/dashboard.go:155` to iterate ALL steps using `ctx.StepOrder` (deterministic ordering) instead of iterating `ctx.StepStatuses` (random map order). For each stepID in `ctx.StepOrder`, look up state from `ctx.StepStatuses[stepID]`, persona from `ctx.StepPersonas[stepID]`, and render: `icon stepID (persona)`. Remove the `break` at line 179 that stops after the first running step. Remove the `stepNum` counter variable. Keep the pulsating effect only for the running step.

- [X] T014 [P1] [US1] Update `getStatusIcon()` in `internal/display/dashboard.go:290` to use spec-mandated indicators. Change `StateSkipped` from `"-"` to `"—"` (em dash, U+2014). Change `StateCancelled` from `"X"` to `"⊛"` (circled asterisk, U+229B). Other states remain unchanged.

- [X] T015 [P1] [US2] In the rewritten `renderStepStatusPanel()` in `internal/display/dashboard.go`, add persona display for non-running steps. Currently only the running step shows persona via `ctx.CurrentPersona`. Use `ctx.StepPersonas[stepID]` for all steps. Show duration for completed and failed steps using `ctx.StepDurations[stepID]`.

## Phase 4: Comprehensive Tests

Depends on Phases 2 and 3. Tests validate all functional requirements.

- [X] T016 [P1] [US1] Add test `TestRenderCurrentStep_AllStepsVisible` to `internal/display/bubbletea_model_test.go`. Create a `PipelineContext` with 4 steps in `StepOrder` with states: completed, running, not_started, not_started. Call `renderCurrentStep()` on a `ProgressModel` and verify the output string contains all 4 step IDs. Validates FR-001.

- [X] T017 [P1] [US2] Add test `TestRenderCurrentStep_PersonaDisplayed` to `internal/display/bubbletea_model_test.go`. Create a `PipelineContext` with `StepPersonas` mapping 3 steps to distinct personas. Verify the rendered output contains all 3 persona names in parentheses. Validates FR-002.

- [X] T018 [P1] [US1] Add test `TestRenderCurrentStep_StepOrderPreserved` to `internal/display/bubbletea_model_test.go`. Create a `PipelineContext` with `StepOrder: ["alpha", "beta", "gamma"]`. Verify "alpha" appears before "beta" and "beta" appears before "gamma" in the rendered output using `strings.Index`. Validates FR-008.

- [X] T019 [P1] [US1] Add test `TestRenderCurrentStep_AllSixStates` to `internal/display/bubbletea_model_test.go`. Create a `PipelineContext` with 6 steps, each in a different state (not_started, running, completed, failed, skipped, cancelled). Verify the output contains all 6 indicator characters: `○`, spinner char, `✓`, `✗`, `—`, `⊛`. Validates FR-003 through FR-007 and SC-004.

- [X] T020 [P1] [US1] Add test `TestRenderCurrentStep_SingleSpinner` to `internal/display/bubbletea_model_test.go`. Create a `PipelineContext` with 4 steps where exactly one is running. Verify the braille spinner characters appear exactly once in the output. Validates FR-012.

- [X] T021 [P2] [US3] Add test `TestRenderCurrentStep_RunningShowsElapsedTime` to `internal/display/bubbletea_model_test.go`. Create a `PipelineContext` with one running step and `CurrentStepStart` set to a known past time. Verify the rendered output contains a duration string (e.g., contains "s" indicating seconds). Validates FR-010.

- [X] T022 [P2] [US3] Add test `TestRenderCurrentStep_CompletedShowsFinalDuration` to `internal/display/bubbletea_model_test.go`. Create a `PipelineContext` with one completed step and `StepDurations` set to 23500ms. Verify the rendered output contains "23.5s". Validates FR-010.

- [X] T023 [P1] [US1] Add test `TestRenderCurrentStep_SingleStepPipeline` to `internal/display/bubbletea_model_test.go`. Create a `PipelineContext` with 1 step (running). Verify the output renders without error and contains the step ID. Validates edge case: single-step pipeline.

- [X] T024 [P1] [US2] Add test `TestBubbleTeaToPipelineContext_StepPersonas` to `internal/display/progress_test.go`. Verify that `BubbleTeaProgressDisplay.toPipelineContext()` (via `AddStep` then state update) populates `StepPersonas` correctly. Call `AddStep("s1", "step-1", "navigator")` and `AddStep("s2", "step-2", "implementer")`, then verify the resulting `PipelineContext.StepPersonas` has both entries. Validates C-4.

- [X] T025 [P1] [US1] Add test `TestDashboard_RenderStepStatusPanel_AllSteps` to `internal/display/dashboard_test.go`. Create a `PipelineContext` with 3 steps in `StepOrder` with different states and `StepPersonas`. Call `renderStepStatusPanel()` and verify output contains all 3 step IDs and their personas. Validates FR-001, FR-008 for dashboard path.

- [X] T026 [P1] [US1] Add test `TestDashboard_GetStatusIcon_SpecIndicators` to `internal/display/dashboard_test.go`. Verify `getStatusIcon()` returns the correct character for each of the 6 states: `○` for not_started, `>` for running, checkmark for completed, crossmark for failed, `—` for skipped, `⊛` for cancelled. Validates indicator consistency.

## Phase 5: Integration & Polish

Final validation. Depends on all previous phases.

- [X] T027 [P1] Run `go test ./internal/display/...` to verify all new and existing display tests pass. Fix any compilation errors or test failures. Validates SC-006.

- [X] T028 [P1] Run `go test -race ./...` to verify no race conditions in the full test suite. The `toPipelineContext()` methods already run under mutex, but verify no new data races are introduced. Validates test ownership requirement.

- [X] T029 [P1] Run `go vet ./internal/display/...` to verify no static analysis warnings in the modified code. Fix any issues found.
