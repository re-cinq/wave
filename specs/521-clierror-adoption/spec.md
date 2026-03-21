# fix(cli): adopt CLIError across all 12 commands missing structured errors

**Issue**: [#521](https://github.com/re-cinq/wave/issues/521)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

## Description

The following 12 commands need CLIError adoption for structured error handling:

1. do
2. meta
3. validate
4. clean
5. status
6. logs
7. cancel
8. chat
9. postmortem
10. doctor
11. bench
12. migrate

These commands currently lack consistent error handling using the CLIError struct defined in the codebase. Adopting CLIError will improve error reporting consistency and user experience.

## Current State

- `CLIError` is defined in `cmd/wave/commands/errors.go` with `NewCLIError(code, message, suggestion)` constructor
- Error codes are constants: `CodePipelineNotFound`, `CodeManifestMissing`, `CodeManifestInvalid`, `CodeContractViolation`, `CodeFlagConflict`, `CodeOnboardingRequired`, `CodePreflightFailed`, `CodeInternalError`, `CodeSecurityViolation`, `CodeSkillNotFound`, `CodeSkillSourceError`, `CodeSkillDependencyMissing`, `CodeInvalidArgs`
- Commands already using CLIError: `run.go`, `resume.go`, `compose.go`, `skills.go`, `agent.go`, `output.go`
- All 12 target commands use `fmt.Errorf()` for errors that should be `CLIError`

## Acceptance Criteria

- [ ] All 12 listed commands return `*CLIError` for user-facing errors (manifest missing/invalid, invalid args, run not found, state DB errors, etc.)
- [ ] New error code constants added where needed (e.g., state DB, run not found, migration, dataset)
- [ ] Existing behavior preserved — only error types change, not program flow
- [ ] All existing tests pass
- [ ] Error messages include actionable suggestions
