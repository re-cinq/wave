# Implementation Plan: Reliable Pipeline Resume

## 1. Objective

Fix the resume error propagation path so that step failures during resumed pipelines produce `StepError` (enabling recovery hints with correct resume commands), generalize the validation/error messaging to work for all pipelines (not just `prototype`), record `FailureClass` on step attempts for better retry context, and update documentation to cover the full resume workflow.

## 2. Approach

The resume infrastructure is ~90% complete. The core bugs are:

1. **`executeResumedPipeline` returns `FormatPhaseFailureError` instead of `StepError`** — this breaks the CLI's recovery hint extraction. Fix: return `&StepError{StepID: step.ID, Err: err}` like `Execute()` does.

2. **`FormatPhaseFailureError` is prototype-specific** — it contains hardcoded guidance for spec/docs/dummy/implement phases. Since `executeResumedPipeline` uses it for ALL pipelines, this is misleading. Fix: remove the `FormatPhaseFailureError` call from the resume path entirely (the CLI's `recovery.BuildRecoveryBlock` already generates appropriate hints).

3. **`PhaseSkipValidator.ValidatePhaseSequence` only validates `prototype`** — for non-prototype pipelines it either silently passes or errors incorrectly. Fix: generalize to validate that all dependency steps' workspaces exist (or skip validation for non-prototype pipelines since `--force` already provides an escape hatch).

4. **`FailureClass` not recorded in step attempts** — `executeStep` records failed attempts but doesn't classify the error. Fix: use `recovery.ClassifyError()` to set `FailureClass` before recording.

5. **Documentation gaps** — update `docs/guides/state-resumption.md` with `--force`, `--run`, `--exclude` flags and fix broken link.

## 3. File Mapping

| File | Action | Change |
|------|--------|--------|
| `internal/pipeline/resume.go` | modify | Return `StepError` from `executeResumedPipeline`, remove `FormatPhaseFailureError` wrapper, emit failed event |
| `internal/pipeline/executor.go` | modify | Record `FailureClass` on failed step attempts |
| `internal/pipeline/validation.go` | modify | Generalize `PhaseSkipValidator` for non-prototype pipelines |
| `internal/pipeline/resume_test.go` | modify | Add tests for `StepError` propagation, non-prototype pipeline resume |
| `internal/recovery/recovery_test.go` | modify | Add test verifying recovery hints work with resume-originated errors |
| `docs/guides/state-resumption.md` | modify | Add `--force`, `--run`, `--exclude` docs, fix broken link |

## 4. Architecture Decisions

- **Keep `FormatPhaseFailureError` for validation errors only** — it's fine for phase prerequisite failures (where the error IS about phases). Remove it from step execution failures where it wraps runtime errors.
- **Don't change the `Execute()` path** — it already correctly returns `StepError`. Only fix the `executeResumedPipeline` path to match.
- **Generalize `PhaseSkipValidator` minimally** — for non-prototype pipelines, validate that dependency workspace dirs exist rather than checking prototype-specific phase completion. This gives useful validation without prototype assumptions.
- **Use existing `recovery.ClassifyError()`** — don't invent a new classification system; wire the existing one into step attempt recording.

## 5. Risks

| Risk | Mitigation |
|------|-----------|
| Changing error types could break downstream consumers | `StepError` wraps the original error with `Unwrap()`, so `errors.As/Is` chains still work |
| `PhaseSkipValidator` changes could reject valid resumes | Keep `--force` as escape hatch; add tests for common pipeline shapes |
| `FormatPhaseFailureError` removal could lose useful guidance | Recovery hints from `recovery.BuildRecoveryBlock` provide equivalent (better) guidance |
| Circular import adding `recovery` to `pipeline` for `ClassifyError` | The `executor.go` already imports `security`, `contract`, `preflight` — `recovery` has no `pipeline` dependency so no cycle |

## 6. Testing Strategy

- **Unit tests** (`resume_test.go`):
  - `TestExecuteResumedPipeline_ReturnsStepError` — verify `errors.As(*StepError)` works on resume failure
  - `TestResumeNonPrototypePipeline` — verify resume works for arbitrary pipeline names
  - `TestResumeWithFailureClassInContext` — verify `FailureClass` is populated on resume
- **Unit tests** (`recovery_test.go`):
  - `TestRecoveryBlockFromStepError` — verify recovery block includes resume command
- **Existing tests** — run `go test ./...` to ensure no regressions
- **Manual verification** — run a pipeline with `--mock`, force a failure, verify resume command is shown correctly
