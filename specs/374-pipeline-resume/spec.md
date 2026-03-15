# feat(pipeline): implement reliable resume for failed pipeline steps

**Issue**: [#374](https://github.com/re-cinq/wave/issues/374)
**Labels**: documentation, enhancement, pipeline
**Author**: nextlevelshit
**Complexity**: complex

## Problem

Resuming a failed pipeline step is not properly wired or documented. When a pipeline step fails, the resume path is unclear to users and may not function correctly.

## Current Behavior

- `wave run --from-step <step>` exists and delegates to `ResumeWithValidation` → `ResumeManager.ResumeFromStep`
- `PhaseSkipValidator` only validates `prototype` pipeline phases — non-prototype pipelines get no meaningful validation
- `GetRecommendedResumePoint` is hardcoded to prototype pipeline only
- `executeResumedPipeline` wraps step failures with `FormatPhaseFailureError` (returns `fmt.Errorf`) instead of `StepError` — the CLI's `errors.As(execErr, &stepErr)` never matches, so recovery hints lack the step ID and the resume command
- `FormatPhaseFailureError` contains prototype-specific troubleshooting guidance (spec/docs/dummy/implement phases) that is misleading for other pipelines
- Recovery hints in `recovery.BuildRecoveryBlock` work correctly when `StepError` is provided, but the resume path never provides it
- `FailureClass` is not set when recording failed step attempts in executor.go, so failure context on resume is less useful
- Documentation exists at `docs/guides/state-resumption.md` but is incomplete: missing `--force`, `--run`, `--exclude` with `--from-step`; links to non-existent `/reference/cli#wave-resume`

## Expected Behavior

- Users can resume a failed pipeline from the last successful step using `wave run --from-step <step>`
- Resume correctly restores workspace state, re-injects artifacts from completed steps, and skips already-completed steps
- CLI provides clear error messages with resume instructions when a step fails
- Documentation covers resume workflow with examples

## Acceptance Criteria

- [ ] `wave run --from-step <step> <pipeline>` successfully resumes from a failed step
- [ ] Artifacts from completed steps are correctly injected into the resumed step
- [ ] Workspace state is properly restored on resume
- [ ] Clear error message with resume command is shown when a step fails (both initial run and resumed run)
- [ ] Resume workflow is documented with examples
- [ ] Tests cover resume from each failure mode (contract failure, adapter crash, missing artifact)

## Key Files

- `internal/pipeline/resume.go` — `ResumeManager` and subpipeline creation
- `internal/pipeline/executor.go` — main execution loop, `StepError` wrapping
- `internal/pipeline/validation.go` — `ErrorMessageProvider`, `PhaseSkipValidator`
- `internal/pipeline/errors.go` — `StepError` type
- `internal/recovery/recovery.go` — `BuildRecoveryBlock`, recovery hints
- `cmd/wave/commands/run.go` — `--from-step` flag handling, recovery hint display
- `internal/state/` — SQLite persistence for step status
- `docs/guides/state-resumption.md` — resume documentation
