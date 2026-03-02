# Implementation Plan: Suppress Preflight Usage Text

## Objective

When `wave run` fails due to a preflight validation error (missing tools or skills), suppress cobra's automatic usage/help text output so the user only sees the error message and recovery hints.

## Approach

The root cause is that cobra automatically prints usage text when a `RunE` function returns an error. The fix is to set `cmd.SilenceUsage = true` inside the `RunE` handler **before** returning the error, but only when the error originates from a preflight check (or more broadly, from `runRun` — since all errors from `runRun` already include recovery hints and don't benefit from usage text).

There are two valid approaches:

### Option A: Blanket `SilenceUsage` for `runRun` errors (Recommended)

Set `cmd.SilenceUsage = true` before calling `runRun()`. This means any error from pipeline execution (preflight, contract validation, runtime) will not print usage. This is the right behavior because:
- Pipeline execution errors are never caused by wrong CLI flags
- Recovery hints already provide actionable guidance
- Usage text is only helpful for argument/flag errors (handled before `runRun`)

### Option B: Conditional `SilenceUsage` for preflight errors only

Have `runRun` return a sentinel/typed error, check for it in the `RunE` closure, and only set `SilenceUsage` for preflight errors. This is more surgical but adds complexity for no real benefit — non-preflight execution errors also don't benefit from usage text.

**Decision**: Option A. Set `cmd.SilenceUsage = true` before the `runRun()` call in the `RunE` handler. Argument validation errors (missing pipeline name, invalid output format) happen before this point and will still show usage text.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/run.go` | modify | Add `cmd.SilenceUsage = true` before `runRun()` call |
| `cmd/wave/commands/run_test.go` | modify | Add test verifying usage is suppressed for preflight errors |

## Architecture Decisions

### AD-1: Scope of usage suppression

**Decision**: Suppress usage for all errors returned by `runRun()`, not just preflight errors.

**Rationale**: Once the CLI has accepted valid arguments and is executing the pipeline, any error is an execution error — not a "you used the command wrong" error. Cobra's usage text is designed for the latter case. The recovery hints system already provides context-specific guidance for execution failures.

### AD-2: Placement of SilenceUsage

**Decision**: Set `cmd.SilenceUsage = true` immediately before the `return runRun(opts, debug)` line in the `RunE` closure.

**Rationale**: This ensures that argument validation errors (missing pipeline name, invalid output format) that happen earlier in the `RunE` function still show usage text, while all pipeline execution errors do not.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Suppressing usage for errors that benefit from it | Low | Low | All pre-`runRun` validation errors still show usage; `runRun` errors already have recovery hints |
| Breaking existing error output expectations | Low | Low | Existing tests verify error content, not usage text presence |

## Testing Strategy

1. **Unit test**: Verify that `NewRunCmd()` execution with a preflight failure does not produce usage text in output
2. **Verify existing tests**: Ensure all existing tests in `cmd/wave/commands/run_test.go` and `tests/preflight_recovery_test.go` continue to pass
3. **Manual verification**: Run `wave run` with a missing tool to confirm clean error output
