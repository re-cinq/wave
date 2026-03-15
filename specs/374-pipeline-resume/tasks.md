# Tasks

## Phase 1: Fix Error Propagation (Critical Path)

- [X] Task 1.1: Return `StepError` from `executeResumedPipeline` instead of `FormatPhaseFailureError`
  - In `internal/pipeline/resume.go:453-455`, replace `return r.errors.FormatPhaseFailureError(step.ID, err, pipelineName)` with `return &StepError{StepID: step.ID, Err: err}`
  - Also emit a "failed" event (matching `Execute()` behavior) before returning
  - Update pipeline status to `StateFailed` and persist to state store
- [X] Task 1.2: Record `FailureClass` on failed step attempts in `executor.go`
  - In `internal/pipeline/executor.go` around line 722, use `recovery.ClassifyError(err)` to populate `FailureClass` in the `StepAttemptRecord`
  - Add import for `recovery` package

## Phase 2: Generalize Validation

- [X] Task 2.1: Update `PhaseSkipValidator` for non-prototype pipelines
  - In `internal/pipeline/validation.go`, modify `ValidatePhaseSequence` to handle arbitrary pipeline names by checking dependency workspace existence instead of prototype-specific phase names
  - Keep prototype-specific validation as-is for backward compatibility
- [X] Task 2.2: Update `GetRecommendedResumePoint` to work for non-prototype pipelines [P]
  - In `internal/pipeline/resume.go:542-560`, generalize to find first incomplete step for any pipeline (not just prototype)
  - Fall back to first step if no workspace state exists

## Phase 3: Testing

- [X] Task 3.1: Add `StepError` propagation test for resume path [P]
  - In `internal/pipeline/resume_test.go`, add `TestExecuteResumedPipeline_ReturnsStepError` that verifies `errors.As(*StepError)` extracts the correct step ID when a resumed step fails
- [X] Task 3.2: Add non-prototype pipeline resume test [P]
  - In `internal/pipeline/resume_test.go`, add test that resumes a pipeline with a non-prototype name and verifies it works correctly
- [X] Task 3.3: Add `FailureClass` recording test [P]
  - Verify that `FailureClass` is set on `StepAttemptRecord` when a step fails with a classified error (contract, security, runtime)
- [X] Task 3.4: Run full test suite
  - `go test ./...` — verify no regressions across all packages

## Phase 4: Documentation

- [X] Task 4.1: Update `docs/guides/state-resumption.md`
  - Add `--force` flag documentation (skips phase validation and stale artifact checks)
  - Add `--run <run-id>` flag documentation (target specific prior run for artifact resolution)
  - Add `--exclude` with `--from-step` combination documentation
  - Fix broken link to `/reference/cli#wave-resume`
  - Add failure mode examples (contract failure, adapter crash, missing artifact)
- [X] Task 4.2: Final validation
  - Review all changes for consistency
  - Verify recovery hint output format matches expectations
  - Ensure `go vet ./...` passes
