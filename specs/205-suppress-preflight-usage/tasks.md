# Tasks

## Phase 1: Core Fix

- [X] Task 1.1: Add `cmd.SilenceUsage = true` before the `return runRun(opts, debug)` call in the `RunE` closure of `NewRunCmd()` in `cmd/wave/commands/run.go`. This single line change ensures cobra does not append usage/help text when `runRun` returns an error, while preserving usage text for argument validation errors that happen earlier in the closure.

## Phase 2: Testing

- [X] Task 2.1: Add a unit test in `cmd/wave/commands/run_test.go` that exercises `NewRunCmd()` with a cobra command execution that triggers a preflight-like error path, and verifies the command's `SilenceUsage` field is set to `true` after the RunE handler executes with a `runRun` error.
- [X] Task 2.2: Run `go test ./...` to verify all existing tests pass with the change.
- [X] Task 2.3: Run `go test -race ./...` to verify no race conditions.

## Phase 3: Validation

- [X] Task 3.1: Verify that argument validation errors (e.g., invalid output format) still show usage text by confirming `SilenceUsage` is not set before the `runRun` call.
- [X] Task 3.2: Review the error output flow end-to-end: `executor.Execute` → preflight error → `runRun` → `RunE` → cobra error handling — confirm no usage text is printed.
