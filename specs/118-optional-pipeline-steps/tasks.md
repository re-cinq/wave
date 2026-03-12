# Tasks

## Phase 1: Data Model

- [X] Task 1.1: Add `Optional bool` field to `Step` struct in `internal/pipeline/types.go` with `yaml:"optional,omitempty"` tag
- [X] Task 1.2: Add `IsOptional()` helper method on `Step` that returns `s.Optional` (for future extension, e.g. conditional optional)

## Phase 2: Core Executor Logic

- [X] Task 2.1: Modify `executeStep` in `internal/pipeline/executor.go` to check `step.Optional` when all retry attempts are exhausted and `retry.on_failure` is unset — treat as `on_failure: "continue"` (mark step failed, emit event, return nil)
- [X] Task 2.2: Modify `findReadySteps` to detect steps whose dependencies include a failed or skipped step and exclude them from the ready set
- [X] Task 2.3: Update the main execution loop in `Execute` to handle the case where `executeStepBatch` returns nil but some steps in the batch were marked failed/skipped (optional step failures) — add these to `completed` map to avoid deadlock, and track them in `FailedSteps` without setting pipeline state to failed
- [X] Task 2.4: Update pipeline completion logic: pipeline status is `completed` (not `failed`) when only optional steps failed — add helper `hasRequiredFailures(execution)` to distinguish

## Phase 3: Display and Reporting

- [X] Task 3.1: Add `OptionalFailedSteps int` field to `DisplayContext` in `internal/display/types.go` [P]
- [X] Task 3.2: Update `internal/display/dashboard.go` to show optional failures distinctly (e.g. "2 pass, 1 optional-fail") [P]
- [X] Task 3.3: Update `internal/display/progress.go` to count optional failures separately when computing display context [P]
- [X] Task 3.4: Update `internal/display/bubbletea_model.go` to render optional failure count in status line [P]

## Phase 4: Testing

- [X] Task 4.1: Test — optional step fails, pipeline continues to next independent step
- [X] Task 4.2: Test — optional step succeeds, pipeline behaves normally
- [X] Task 4.3: Test — step without `optional` field fails, pipeline halts (regression)
- [X] Task 4.4: Test — optional step with retries: retries all attempts, then continues
- [X] Task 4.5: Test — dependent step is skipped when optional dependency fails [P]
- [X] Task 4.6: Test — transitive skip: C depends on B depends on optional A, A fails, both B and C skipped [P]
- [X] Task 4.7: Test — `optional: true` with `retry.on_failure: "fail"` — explicit on_failure wins
- [X] Task 4.8: Test — pipeline status is `completed` when only optional steps fail
- [X] Task 4.9: Test — YAML parsing round-trip for `optional: true` field [P]

## Phase 5: Validation

- [X] Task 5.1: Run `go test ./...` and fix any failures
- [X] Task 5.2: Run `go vet ./...` to check for issues
- [X] Task 5.3: Verify existing `retry.on_failure` tests still pass (regression check)
