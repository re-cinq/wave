# Tasks

## Phase 1: Add New Error Codes
- [X] Task 1.1: Add new error code constants to `errors.go` — `CodeStateDBError`, `CodeRunNotFound`, `CodeMigrationFailed`, `CodeDatasetError`, `CodeValidationFailed`

## Phase 2: Core Implementation — Manifest-Loading Commands
- [X] Task 2.1: Adopt CLIError in `do.go` — manifest missing/invalid, pipeline generation, execution errors [P]
- [X] Task 2.2: Adopt CLIError in `meta.go` — manifest missing/invalid, pipeline generation/execution, save errors [P]
- [X] Task 2.3: Adopt CLIError in `validate.go` — manifest read/parse, structure/reference/pipeline validation errors [P]
- [X] Task 2.4: Adopt CLIError in `chat.go` — state DB, manifest, run not found, pipeline load, workspace, step/artifact not found [P]

## Phase 3: Core Implementation — State DB Commands
- [X] Task 3.1: Adopt CLIError in `status.go` — state DB open, JSON marshal errors [P]
- [X] Task 3.2: Adopt CLIError in `logs.go` — state DB open, query errors, duration parse errors [P]
- [X] Task 3.3: Adopt CLIError in `cancel.go` — state DB, run not found, not running, cancellation failures [P]
- [X] Task 3.4: Adopt CLIError in `postmortem.go` — state DB missing, run not found, wrong status errors [P]

## Phase 4: Core Implementation — Remaining Commands
- [X] Task 4.1: Adopt CLIError in `clean.go` — missing flags, invalid status, duration parse, listing errors [P]
- [X] Task 4.2: Adopt CLIError in `doctor.go` — flag dependency errors, scan/check/optimize failures [P]
- [X] Task 4.3: Adopt CLIError in `bench.go` — missing flags, mode validation, dataset load, benchmark execution errors [P]
- [X] Task 4.4: Adopt CLIError in `migrate.go` — migration runner creation, version parse, execution failures [P]

## Phase 3: Testing & Validation
- [X] Task 5.1: Run `go test ./cmd/wave/commands/...` to verify all command tests pass
- [X] Task 5.2: Run `go test ./...` for full test suite validation
- [X] Task 5.3: Run `go vet ./...` for static analysis
