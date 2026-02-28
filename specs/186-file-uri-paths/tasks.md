# Tasks

## Phase 1: Core Utility

- [X] Task 1.1: Create `internal/pathfmt/fileuri.go` with `FileURI(path string) string` function
  - Prefix absolute paths (starting with `/`) with `file://`
  - Skip paths that already contain a URI scheme (`://`)
  - Skip relative paths and empty strings
  - Return the original string for non-absolute paths
  - Note: placed in `internal/pathfmt` instead of `internal/display` to avoid import cycle with `internal/deliverable`
- [X] Task 1.2: Create `internal/pathfmt/fileuri_test.go` with table-driven tests
  - Absolute path `/home/user/file.json` → `file:///home/user/file.json`
  - Relative path `.wave/workspaces/...` → unchanged
  - Already prefixed `file:///path` → unchanged
  - URL `https://github.com/...` → unchanged
  - Empty string → empty string
  - Path with spaces `/path/with spaces/file` → `file:///path/with spaces/file`
  - Root path `/` → `file:///`

## Phase 2: Contract Validation Paths

- [X] Task 2.1: Modify `internal/contract/validation_error_formatter.go` — wrap `artifactPath` with `pathfmt.FileURI()` in `FormatJSONSchemaError`
- [X] Task 2.2: Modify `internal/contract/jsonschema.go` — wrap `artifactPath` with `pathfmt.FileURI()` in error detail strings [P]
- [X] Task 2.3: Verify no import cycle — resolved by placing `FileURI` in `internal/pathfmt` package [P]

## Phase 3: Recovery Hint Paths

- [X] Task 3.1: Modify `internal/recovery/recovery.go` — apply `pathfmt.FileURI()` to the workspace path in the `ls` command hint
- [X] Task 3.2: Update `tests/preflight_recovery_test.go` — update double-slash check to allow `://` in URI schemes

## Phase 4: Outcome & Deliverable Display Paths

- [X] Task 4.1: Modify `internal/display/outcome.go` — wrap `outcome.WorkspacePath` with `pathfmt.FileURI()` in `GenerateNextSteps` [P]
- [X] Task 4.2: Modify `internal/deliverable/types.go` — wrap `absPath` with `pathfmt.FileURI()` in `Deliverable.String()` [P]
- [X] Task 4.3: Modify `internal/display/progress.go` — wrap artifact `path` with `pathfmt.FileURI()` in handover line [P]
- [X] Task 4.4: Modify `internal/display/bubbletea_model.go` — wrap artifact `path` with `pathfmt.FileURI()` in handover line [P]

## Phase 5: Test Updates & Validation

- [X] Task 5.1: Existing tests in contract, recovery, display, deliverable pass without modification (assertions use `strings.Contains` patterns that match with `file://` prefix)
- [X] Task 5.2: Updated `tests/preflight_recovery_test.go` — fixed double-slash detection to allow URI scheme `://`
- [X] Task 5.3: Run `go test -race ./...` — all tests pass
- [X] Task 5.4: Run `go vet ./...` — no static analysis issues
