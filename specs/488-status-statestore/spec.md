# audit: regressed — status command direct SQLite access (#2)

**Issue**: [#488](https://github.com/re-cinq/wave/issues/488)
**Labels**: audit
**Author**: nextlevelshit
**Source**: #2 — Refactor Status Command to Use StateStore Interface

## Problem

`cmd/wave/commands/status.go` bypasses the `StateStore` interface and accesses SQLite directly:

- Imports `database/sql` and `modernc.org/sqlite`
- Contains raw SQL helper functions: `queryRun`, `queryRunningRuns`, `queryRecentRuns`, `queryRunsInternal`, `queryRunsInternalWithArgs`, `scanRuns`
- `runStatus()` opens `sql.Open("sqlite", dbPath)` directly instead of using `state.NewStateStore()`

## Acceptance Criteria

1. `status.go` uses `state.NewStateStore()` to obtain a `StateStore` instance
2. `showRunDetails` calls `store.GetRun(runID)` instead of `queryRun(db, runID)`
3. `showRunningRuns` calls `store.GetRunningRuns()` instead of `queryRunningRuns(db)`
4. `showAllRuns` calls `store.ListRuns(opts)` instead of `queryRecentRuns(db, limit)`
5. All raw SQL helper functions removed: `queryRun`, `queryRunningRuns`, `queryRecentRuns`, `queryRunsInternal`, `queryRunsInternalWithArgs`, `scanRuns`
6. `database/sql` and `modernc.org/sqlite` imports removed from `status.go`
7. `RunRecord` fields mapped to `StatusRunInfo` for display formatting
8. `go test ./...` passes
9. `go vet ./...` clean
