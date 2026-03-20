# Tasks

## Phase 1: Core Refactor

- [X] Task 1.1: Update imports — replace `database/sql` and `modernc.org/sqlite` with `github.com/recinq/wave/internal/state`
- [X] Task 1.2: Add `runRecordToStatusInfo(r *state.RunRecord) StatusRunInfo` conversion function
- [X] Task 1.3: Refactor `runStatus()` — replace `sql.Open()` with `state.NewStateStore()`, update function signatures to pass `state.StateStore` instead of `*sql.DB`
- [X] Task 1.4: Refactor `showRunDetails()` — use `store.GetRun()` + conversion
- [X] Task 1.5: Refactor `showRunningRuns()` — use `store.GetRunningRuns()` + conversion
- [X] Task 1.6: Refactor `showAllRuns()` — use `store.ListRuns()` + conversion

## Phase 2: Cleanup

- [X] Task 2.1: Delete `queryRun`, `queryRunningRuns`, `queryRecentRuns`, `queryRunsInternal`, `queryRunsInternalWithArgs`, `scanRuns` functions (lines 291-446)

## Phase 3: Validation

- [X] Task 3.1: Run `go build ./cmd/wave/...` to verify compilation
- [X] Task 3.2: Run `go vet ./...` for static analysis
- [X] Task 3.3: Run `go test ./...` for full test suite
