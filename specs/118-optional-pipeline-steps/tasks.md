# Implementation Tasks: Optional Pipeline Steps

**Feature**: 118-optional-pipeline-steps
**Generated**: 2026-02-20
**Source**: spec.md, plan.md, data-model.md, research.md, contracts/

---

## Phase 1: Setup & Constants

These tasks add the foundational type changes and constants that all subsequent phases depend on.

- [X] T001 P1 US1 Add `Optional bool` field to `Step` struct in `internal/pipeline/types.go` — insert `Optional bool \`yaml:"optional,omitempty"\`` after `Dependencies` field, before `Memory` field
- [X] T002 P1 US1 Add `StateFailedOptional` constant to pipeline types in `internal/pipeline/types.go` — add `StateFailedOptional = "failed_optional"` to the existing `const` block alongside `StatePending`, `StateRunning`, etc.
- [X] T003 [P] P1 US1 Add `StateFailedOptional StepState` constant to state store in `internal/state/store.go` — add `StateFailedOptional StepState = "failed_optional"` to the existing `const` block (line 23-29)
- [X] T004 [P] P1 US2 Add `Optional bool` field and `StateFailedOptional` constant to event package in `internal/event/emitter.go` — add `Optional bool \`json:"optional,omitempty"\`` field to `Event` struct (after `Adapter` field), and add `StateFailedOptional = "failed_optional"` to the event state constants block
- [X] T005 [P] P1 US2 Add `StateFailedOptional ProgressState` constant to display types in `internal/display/types.go` — add `StateFailedOptional ProgressState = "failed_optional"` to the `ProgressState` const block (after `StateCancelled`)

---

## Phase 2: Foundational — State Persistence & Validation

These tasks update the state store to correctly persist the new state, and ensure YAML parsing validates the `optional` field. They must complete before execution logic changes.

- [X] T006 P1 US1 Update `SaveStepState` completedAt logic in `internal/state/store.go` (line 279) — change `if state == StateCompleted || state == StateFailed` to also include `state == StateFailedOptional` so that `completed_at` is set for failed optional steps
- [X] T007 [P] P1 US1 Update `GetStatus` step state mapping in `internal/pipeline/executor.go` (line 1469-1477) — add a `case StateFailedOptional:` (using the string value `"failed_optional"`) to the switch that populates `FailedOptionalSteps` on `PipelineStatus`
- [X] T008 [P] P1 US2 Add `FailedOptionalSteps []string` and `SkippedSteps []string` fields to `PipelineStatus` struct in `internal/pipeline/executor.go` (line 34-43) — add after `FailedSteps` field
- [X] T009 P1 US1 Write unit test for YAML parsing of `optional` field in `internal/pipeline/types_test.go` — test `optional: true`, `optional: false`, field omitted (defaults to false), and `optional: "not-a-bool"` (should error). Maps to behavior contract 10 (FR-012)

---

## Phase 3: Core Execution — US1 (Non-Critical Step Continues Pipeline on Failure)

This is the highest-priority user story. These tasks modify the main execution loop to handle optional step failures without halting.

- [X] T010 P1 US1 Modify `executeStep` in `internal/pipeline/executor.go` (line 330-405) — after the retry loop exhausts retries (line 387-390), check `step.Optional`: if true, set `execution.States[step.ID] = StateFailedOptional`, save state as `state.StateFailedOptional` with error message, and `return nil` instead of returning the error. Required steps retain existing behavior (`return lastErr`)
- [X] T011 P1 US1 Modify the main execution loop in `Execute()` in `internal/pipeline/executor.go` (line 274-305) — after `executeStep` returns nil, check `execution.States[step.ID]`: if it is `StateFailedOptional`, append to `execution.Status.FailedOptionalSteps` instead of `execution.Status.CompletedSteps`. If the step completed normally, append to `CompletedSteps` as before
- [X] T012 P1 US1 Emit `failed_optional` event in `executeStep` when an optional step fails in `internal/pipeline/executor.go` — emit `event.Event` with `State: event.StateFailedOptional`, `Optional: true`, step ID, error message, and persona. This replaces the pipeline-level failure event for optional steps
- [X] T013 P1 US1 Set `Optional: true` on all events emitted for optional steps in `internal/pipeline/executor.go` — in `runStepExecution` (line 433+), when emitting "running", "step_progress", and "stream_activity" events for a step, set `Optional: step.Optional` on the event. Pass `step.Optional` through the `OnStreamEvent` callback closure
- [X] T014 P1 US1 Write unit tests for optional step failure continuing pipeline in `internal/pipeline/executor_test.go` — test: (a) pipeline with optional step B between required A and C, B fails, pipeline completes; (b) all required steps, existing failure behavior preserved; (c) optional step succeeds, artifacts available downstream. Maps to behavior contracts 1, 3, 12

---

## Phase 4: Dependency Skipping — US3 (Optional Step Dependency Handling)

- [X] T015 P2 US3 Add pre-execution artifact injection check in `Execute()` loop in `internal/pipeline/executor.go` (before calling `executeStep` at line 275) — before executing each step, iterate `step.Memory.InjectArtifacts`: if any referenced step's state in `execution.States` is `StateFailedOptional` or `"skipped"` (display.StateSkipped), mark the current step as `"skipped"` in `execution.States`, save state via store as `"skipped"`, emit a skipped event with descriptive message, append to `execution.Status.SkippedSteps`, and `continue` to next step. Steps with only `Dependencies` (ordering) references to failed optional steps are NOT skipped
- [X] T016 P2 US3 Write unit tests for artifact injection skipping in `internal/pipeline/executor_test.go` — test: (a) step C with `inject_artifacts` from failed optional step B → C is skipped; (b) step C with only `dependencies` on failed optional B → C runs; (c) transitive skipping: A(optional, fails) → B(injects from A, skipped) → C(injects from B, skipped). Maps to behavior contracts 5, 6

---

## Phase 5: Display & Events — US2 (Pipeline Summary Distinguishes Optional Failures)

- [X] T017 [P] P2 US2 Add `FormatState` and `GetStateIcon` cases for `StateFailedOptional` in `internal/display/capability.go` (line 359-392) — add case for `StateFailedOptional`: use `Warning` color (yellow/orange) for text and a distinct icon (e.g., `⚠` or `!` for ASCII). This distinguishes optional failures visually from required failures (red ✗) and successes (green ✓)
- [X] T018 P2 US2 Handle `"failed_optional"` event state in `BubbleTeaProgressDisplay.processEvent` in `internal/display/bubbletea_progress.go` (after line 243 skipped case) — add `case "failed_optional":` that sets `step.State = StateFailedOptional`, clears current step, and captures duration. Also update `toPipelineContext` counting logic (line 274-288) to count `StateFailedOptional` steps separately and report them in the context
- [X] T019 P2 US2 Render `StateFailedOptional` in bubbletea model View in `internal/display/bubbletea_model.go` (after line 344 StateSkipped case) — add rendering for `StateFailedOptional`: use a distinct icon (e.g., `⚠`) and color (yellow, lipgloss Color "11") with step ID, persona, and duration. This visually separates optional failures from pipeline-halting failures
- [X] T020 [P] P2 US2 Update pipeline completion summary in executor to report optional failures in `internal/pipeline/executor.go` — in the completion event emission (line 316-322), include count of failed optional steps and skipped steps in the completion message (e.g., "5 steps completed, 1 optional failure, 1 skipped")
- [X] T021 P2 US2 Write unit tests for display of optional failures in `internal/display/bubbletea_progress_test.go` or `internal/display/capability_test.go` — test: `FormatState(StateFailedOptional)` returns warning-colored text; `GetStateIcon(StateFailedOptional)` returns distinct icon; bubbletea model renders failed_optional differently from failed. Maps to behavior contract 7

---

## Phase 6: Resume Compatibility — US4 (Resume Pipeline with Optional Step History)

- [X] T022 P3 US4 Update `Resume()` in `internal/pipeline/executor.go` (line 1411) — change skip condition from `execution.States[step.ID] != StateCompleted` to also skip `StateFailedOptional`: `execution.States[step.ID] != StateCompleted && execution.States[step.ID] != StateFailedOptional`
- [X] T023 P3 US4 Update `loadResumeState` in `internal/pipeline/resume.go` (line 244) — when loading prior step state, in addition to marking steps as `StateCompleted`, check the state store for steps with `"failed_optional"` state and preserve them as `StateFailedOptional` in the resume state (do not re-execute them). If no state store is available, check for workspace existence as before
- [X] T024 P3 US4 Update `executeResumedPipeline` in `internal/pipeline/resume.go` (line 338-364) — in the step execution loop, add the same pre-execution artifact injection skip check as T015 (check if step's `inject_artifacts` references point to `StateFailedOptional` or `"skipped"` steps), and update the CompletedSteps/FailedOptionalSteps tracking
- [X] T025 P3 US4 Write unit tests for resume with optional step state in `internal/pipeline/resume_test.go` — test: (a) resume from step D skips previously failed-optional step B; (b) step B's `"failed_optional"` state is preserved in resumed execution. Maps to behavior contracts 8, 9

---

## Phase 7: Edge Cases & Comprehensive Testing

- [X] T026 [P] P1 US1 Write test for all-optional pipeline succeeding in `internal/pipeline/executor_test.go` — pipeline where ALL steps are `optional: true` and all fail → pipeline status is `"completed"`, all steps marked `"failed_optional"`. Maps to behavior contract 11
- [X] T027 [P] P1 US1 Write test for optional step with retries in `internal/pipeline/executor_test.go` — optional step with `max_retries: 3` that always fails → retries 3 times, then marked `"failed_optional"` (not `"failed"`). Maps to behavior contract 9
- [X] T028 [P] P2 US3 Write test for required step depending on failed optional step's artifacts in `internal/pipeline/executor_test.go` — required step C has `inject_artifacts` from failed optional step B → step C is skipped with descriptive error. Covers edge case from spec
- [X] T029 [P] P1 US1 Write test for optional step as last step in pipeline in `internal/pipeline/executor_test.go` — optional last step fails → pipeline completes successfully. Covers edge case from spec
- [X] T030 [P] P1 US1 Write test for `optional: false` explicit default in `internal/pipeline/executor_test.go` — step with `optional: false` behaves identically to step without `optional` field. Covers edge case from spec
- [X] T031 P1 US1 Write test for contract validation skipped on failed optional step in `internal/pipeline/executor_test.go` — optional step with contract that would fail → step adapter fails, contract validation NOT invoked, step marked `"failed_optional"`. Maps to behavior contract 3

---

## Phase 8: Integration & Backward Compatibility

- [X] T032 P1 US1 Run full test suite `go test ./...` to verify backward compatibility — all existing tests must pass without modification since `Optional` defaults to `false`. Maps to behavior contract 12 and SC-004
- [X] T033 [P] P1 US2 Write test for event emission with optional fields in `internal/pipeline/executor_test.go` or `internal/event/emitter_test.go` — verify events for optional steps have `Optional: true` and `State: "failed_optional"`, events for non-optional steps have `Optional` omitted (zero value). Maps to behavior contract 4
- [X] T034 P1 US1 Write test for state store persistence of `"failed_optional"` state in `internal/state/store_test.go` — save step state as `StateFailedOptional`, query it back, verify state is `"failed_optional"` and `completed_at` is set. Maps to behavior contract 2

---

## Summary

| Phase | Task Count | User Stories | Parallelizable |
|-------|-----------|-------------|----------------|
| 1: Setup & Constants | 5 | US1, US2 | T003, T004, T005 |
| 2: Foundational | 4 | US1, US2 | T007, T008 |
| 3: Core Execution (US1) | 5 | US1 | — |
| 4: Dependency Skipping (US3) | 2 | US3 | — |
| 5: Display & Events (US2) | 5 | US2 | T017, T020 |
| 6: Resume (US4) | 4 | US4 | — |
| 7: Edge Cases | 6 | US1, US3 | T026-T030 |
| 8: Integration | 3 | US1, US2 | T033 |
| **Total** | **34** | | **12** |
