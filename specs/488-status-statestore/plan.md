# Implementation Plan

## Objective

Refactor `cmd/wave/commands/status.go` to use the `StateStore` interface instead of direct SQLite access, aligning it with the pattern used by all other commands (artifacts, compose, postmortem, chat, cancel, etc.).

## Approach

Single-file refactor of `status.go`:

1. Replace `sql.Open("sqlite", dbPath)` with `state.NewStateStore(dbPath)`
2. Replace `queryRun(db, runID)` with `store.GetRun(runID)` + conversion to `StatusRunInfo`
3. Replace `queryRunningRuns(db)` with `store.GetRunningRuns()` + conversion
4. Replace `queryRecentRuns(db, limit)` with `store.ListRuns(state.ListRunsOptions{Limit: limit})` + conversion
5. Delete all raw SQL helper functions (~160 lines)
6. Add a `runRecordToStatusInfo` conversion function

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/status.go` | modify | Replace direct SQL with StateStore calls |

## Architecture Decisions

- **Conversion function**: `RunRecord` has `time.Time` fields while `StatusRunInfo` uses formatted strings. A `runRecordToStatusInfo()` helper converts between them, keeping display logic in the commands package.
- **Error message compatibility**: `StateStore.GetRun()` already returns `"run not found: <id>"` errors, matching the existing `queryRun` behavior — no change in error handling needed.
- **GetRunningRuns() behavior**: The StateStore's `GetRunningRuns()` includes both `status='running'` and recent `status='pending'` runs (within 5 minutes). The current raw SQL only checks `status='running'`. This is a minor behavior improvement — pending runs that just started will now also appear.

## Risks

| Risk | Mitigation |
|------|------------|
| Field mapping mismatch between `RunRecord` and `StatusRunInfo` | Verify all fields map correctly; `RunRecord` has all needed fields |
| Different SQL query semantics | `GetRunningRuns()` includes pending runs — acceptable improvement |

## Testing Strategy

- Existing tests (if any) should continue to pass
- `go test ./cmd/wave/commands/...` to verify no compilation errors
- `go vet ./...` for static analysis
- Manual verification that `wave status` output format is unchanged
