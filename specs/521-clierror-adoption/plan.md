# Implementation Plan: CLIError Adoption

## Objective

Replace `fmt.Errorf()` calls with structured `CLIError` returns across 12 CLI commands to provide consistent, machine-parseable error output with actionable suggestions.

## Approach

1. **Add new error codes** to `errors.go` for categories not yet covered (state DB, run not found, migration, dataset, validation)
2. **Systematically update each command** to use `NewCLIError()` for user-facing errors, following the patterns already established in `run.go`, `resume.go`, and `compose.go`
3. **Preserve internal errors** — only wrap errors at the user-facing boundary; internal helper functions that feed into user-visible returns should propagate raw errors that the caller wraps

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/errors.go` | modify | Add new error code constants |
| `cmd/wave/commands/do.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/meta.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/validate.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/clean.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/status.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/logs.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/cancel.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/chat.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/postmortem.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/doctor.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/bench.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |
| `cmd/wave/commands/migrate.go` | modify | Replace `fmt.Errorf` with `NewCLIError` |

## Architecture Decisions

1. **New error codes**: Add `CodeStateDBError`, `CodeRunNotFound`, `CodeMigrationFailed`, `CodeDatasetError`, `CodeValidationFailed` to cover all 12 commands' error categories
2. **Boundary principle**: Only user-facing `RunE` function returns and their direct helpers get `CLIError`. Internal library errors remain `fmt.Errorf` and get wrapped at the boundary
3. **Suggestion quality**: Every `CLIError` must include a non-empty suggestion string pointing the user to the next action
4. **cancel.go special case**: The cancel command already has structured `CancelResult` output. We convert the `fmt.Errorf` calls in `outputCancelResult` to `CLIError` while preserving the existing JSON output
5. **checkOnboarding**: The `do.go` helper `checkOnboarding()` already returns a `CLIError` — no change needed there

## Error Code Mapping

| Code | Used by | When |
|------|---------|------|
| `CodeManifestMissing` | do, meta, validate, chat | Manifest file not found |
| `CodeManifestInvalid` | do, meta, validate, chat | YAML parse failure |
| `CodeOnboardingRequired` | do | `checkOnboarding()` fails |
| `CodeInvalidArgs` | clean, bench, doctor, migrate | Missing/invalid flags or args |
| `CodeStateDBError` (NEW) | status, logs, cancel, chat, postmortem | State DB open/query failure |
| `CodeRunNotFound` (NEW) | cancel, chat, postmortem, status, logs | Run ID not found |
| `CodeMigrationFailed` (NEW) | migrate | Migration runner or execution failure |
| `CodeDatasetError` (NEW) | bench | Dataset load/parse failure |
| `CodeValidationFailed` (NEW) | validate | Manifest or pipeline validation failure |
| `CodePipelineNotFound` | chat | Pipeline definition not found |
| `CodeInternalError` | all | Unexpected/unclassified errors |

## Risks

1. **Test assertions on error strings**: Some tests may check `err.Error()` string content. The `CLIError.Error()` method returns `Message`, so as long as we keep the message text similar, tests should pass. Tests checking for `*CLIError` type will need updating.
2. **cancel.go dual output**: The cancel command already serializes `CancelResult` JSON. Converting to `CLIError` for non-success cases changes the JSON structure in `--format json` mode. We should use `CLIError` for actual error returns but keep `CancelResult` for the success-path output.

## Testing Strategy

1. Run `go test ./cmd/wave/commands/...` after each batch of changes
2. Run `go test ./...` after all changes
3. Verify that existing test assertions still pass — most tests use cobra's `Execute()` which captures `RunE` errors
4. No new test files needed — the existing test coverage validates the error paths
