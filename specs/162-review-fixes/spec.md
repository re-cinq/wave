# fix(resume): show prior steps as completed when using --from-step — Code Review Fixes

**Feature Branch**: `162-review-fixes`
**Created**: 2026-02-26
**Status**: Draft
**Source**: [PR #162 Code Review Comment](https://github.com/re-cinq/wave/pull/162#issuecomment-3966468321)

## Summary

Code review on PR #162 identified two issues that need addressing before merge:

1. **Mutex deadlock (HIGH)**: `BubbleTeaProgressDisplay.updateFromEvent` calls `AddStep()` while already holding `btpd.mu.Lock()`, which will deadlock on Go's non-reentrant mutex. Pre-existing bug, but the new synthetic events make it significantly more likely to trigger.

2. **Missing BasicProgressDisplay tests (MEDIUM)**: The `stream_activity` guard was added to `BasicProgressDisplay` but has no test coverage, while the equivalent `BubbleTeaProgressDisplay` guard does.

The review also includes 5 suggested improvements (persona loss on auto-created steps, any-vs-all artifact check, step ID validation, `t.Chdir` usage, glob error logging) and positive observations about the clean scoping, strong test coverage for new logic, and absence of security regressions.

## Labels

None

## User Stories

### User Story 1 — Mutex deadlock fix (Priority: P1)

When a pipeline runs with `--from-step` and synthetic completion events are emitted for prior steps, `BubbleTeaProgressDisplay` must not deadlock when encountering an event for an unknown step ID.

**Why this priority**: A deadlock is a complete hang — the pipeline becomes unresponsive with no error output. This is a correctness and reliability issue.

**Independent Test**: Can be tested by calling `updateFromEvent` with a step ID that does not yet exist in `btpd.steps`, and verifying no deadlock occurs and the step is auto-created.

**Acceptance Scenarios**:

1. **Given** `EmitProgress` holds `btpd.mu`, **When** `updateFromEvent` encounters an unknown step ID, **Then** it creates the step without attempting to re-acquire the mutex.
2. **Given** a pipeline resumed with `--from-step`, **When** synthetic completion events reference steps not yet registered, **Then** those steps are auto-created and marked completed without deadlock.

---

### User Story 2 — BasicProgressDisplay stream_activity guard test coverage (Priority: P2)

The `stream_activity` guard in `BasicProgressDisplay.EmitProgress` (progress.go:647) filters out tool activity events for steps that are not in the "running" state. This guard exists but lacks dedicated test coverage.

**Why this priority**: The equivalent guard in `BubbleTeaProgressDisplay` has test coverage in `bubbletea_progress_resume_test.go`. Parity is needed to prevent regressions.

**Independent Test**: Can be tested with a table-driven test that emits `stream_activity` events for steps in various states (running, completed, not-started) and verifies output is only produced for running steps.

**Acceptance Scenarios**:

1. **Given** a `BasicProgressDisplay` in verbose mode, **When** a `stream_activity` event arrives for a step in "running" state, **Then** tool activity is written to output.
2. **Given** a `BasicProgressDisplay` in verbose mode, **When** a `stream_activity` event arrives for a step in "completed" state, **Then** no tool activity is written to output.
3. **Given** a `BasicProgressDisplay` in verbose mode, **When** a `stream_activity` event arrives for a step not yet started, **Then** no tool activity is written to output.

---

### Edge Cases

- What happens if `updateFromEvent` receives an event with an empty `StepID`? (Already handled — returns early at line 210-212)
- What happens if `AddStep` is called externally while `EmitProgress` is processing? (Addressed by making the internal helper not acquire the mutex)
- What if `stream_activity` arrives for a step that was never registered in `BasicProgressDisplay`? (stepStates map returns zero-value "" which is not "running", so it is correctly dropped)

## Requirements

### Functional Requirements

- **FR-001**: `BubbleTeaProgressDisplay.updateFromEvent` MUST NOT call `AddStep()` (which acquires the mutex). It must use an unlocked helper or inline the step creation logic.
- **FR-002**: The unlocked step-creation path MUST preserve the same behavior as `AddStep`: check existence, create `StepStatus`, append to `stepOrder`.
- **FR-003**: `BasicProgressDisplay.EmitProgress` `stream_activity` guard MUST have table-driven test coverage matching the pattern in `bubbletea_progress_resume_test.go`.
- **FR-004**: All existing tests MUST continue to pass (`go test ./...`).

## Success Criteria

- **SC-001**: `go test -race ./internal/display/...` passes with no data races or deadlocks
- **SC-002**: New test `TestBasicProgressDisplay_StreamActivityGuard` covers running, completed, and not-started states
- **SC-003**: No behavioral change to pipeline execution — only internal refactor and test additions
