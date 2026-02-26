# Tasks

## Phase 1: Fix Mutex Deadlock

- [X] Task 1.1: Extract `addStepLocked` helper from `AddStep` in `bubbletea_progress.go`
  - Create unexported method `addStepLocked(stepID, stepName, persona string)` containing the step-creation logic (existence check, `StepStatus` creation, `stepOrder` append)
  - Modify `AddStep` to acquire the lock and delegate to `addStepLocked`
  - File: `internal/display/bubbletea_progress.go`

- [X] Task 1.2: Replace `AddStep` call in `updateFromEvent` with `addStepLocked`
  - Change line 216 from `btpd.AddStep(evt.StepID, evt.StepID, "")` to `btpd.addStepLocked(evt.StepID, evt.StepID, "")`
  - File: `internal/display/bubbletea_progress.go`

## Phase 2: Add Missing Test Coverage

- [X] Task 2.1: Add `TestBasicProgressDisplay_StreamActivityGuard` table-driven test [P]
  - Create test in `progress_test.go` with 3 sub-cases: running (output expected), completed (no output), not-started (no output)
  - Use `bytes.Buffer` writer and verbose mode
  - Pre-set `stepStates` map entries before emitting `stream_activity` events
  - Assert on buffer content (non-empty vs empty)
  - File: `internal/display/progress_test.go`

## Phase 3: Validation

- [X] Task 3.1: Run existing test suite
  - Run `go test ./internal/display/...` and verify all tests pass
  - Run `go test -race ./internal/display/...` to validate no race conditions

- [X] Task 3.2: Run full project test suite
  - Run `go test ./...` to verify no regressions across the project
