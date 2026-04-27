# Implementation Plan — Issue #1275

## 1. Objective

Eliminate every `database/sql` import outside `internal/state/` by routing the four bypassing CLI commands (`logs`, `decisions`, `clean`, `list`) and the webui `backfillRunTokens` migration through the `StateStore` interface, then enforce the boundary with a `depguard` rule.

## 2. Approach

Follow ADR-007 Option 2 (Consolidate into Existing StateStore Interface). The interface already covers most read paths; add a small set of focused methods to fill the gaps, then rewrite the call sites to depend only on `StateStore`. Keep behavioural parity (follow-mode polling cadence, color output, JSON shape) — this is a routing refactor, not a feature change.

Concretely:

1. **Extend types** in `internal/state/types.go`:
   - `EventQueryOptions` gains `SinceUnix int64`, `TailLimit int`, `OrderDesc bool` (the existing `AfterID` already supports follow-mode).
   - New `DecisionQueryOptions{ StepID, Category string }`.
   - New `EventAggregateStats{ TotalEvents int; TotalTokens int; AvgDurationMs, MinDurationMs, MaxDurationMs float64 }`.

2. **Add methods** on `StateStore` (`internal/state/store.go`, with read-only fallback in `internal/state/readonly.go` already automatic since both share `*stateStore`):
   - `GetMostRecentRunID() (string, error)` — replaces `getMostRecentRunID(db)` in `logs.go`/`decisions.go`.
   - `RunExists(runID string) (bool, error)` — replaces `runExists(db, runID)`.
   - `GetRunStatus(runID string) (string, error)` — replaces `getRunStatus(db, runID)` (used in follow loop).
   - Extend `GetEvents` to honour the new fields on `EventQueryOptions` (Since, TailLimit, OrderDesc) so `queryLogs`/`queryNewLogs` collapse into one call site.
   - `GetEventAggregateStats(runID string) (*EventAggregateStats, error)` — replaces `renderPerformanceSummary`'s aggregate query.
   - Replace `GetDecisions` filtering: introduce `GetDecisionsFiltered(runID string, opts DecisionQueryOptions) ([]*DecisionRecord, error)` (keep existing `GetDecisions` and `GetDecisionsByStep` to avoid breaking other callers, mark them as thin wrappers).
   - `ListPipelineNamesByStatus(status string) ([]string, error)` — replaces `getWorkspacesWithStatus` in `clean.go` (the existing fallback to `pipeline_state` becomes part of the implementation).
   - `BackfillRunTokens() (int64, error)` — replaces `webui.backfillRunTokens`. Lives on `StateStore` (read-write only); webui calls it via `rwStore`.

3. **Migrate call sites**:
   - `cmd/wave/commands/logs.go`: drop the `database/sql` import and the `_ "modernc.org/sqlite"` blank import. Open via `state.NewReadOnlyStateStore(".agents/state.db")`. Map `LogsOptions` → `EventQueryOptions`. Reuse `state.LogRecord` directly or keep `LogsEntry` as a presentation DTO. Follow mode keeps its 500ms ticker but uses `store.GetEvents(runID, EventQueryOptions{AfterID: lastID})`. `runLogsTrace` similarly opens via `NewReadOnlyStateStore` for the lookup.
   - `cmd/wave/commands/decisions.go`: use `state.NewReadOnlyStateStore`, call `GetDecisionsFiltered`. Keep the printing layer untouched.
   - `cmd/wave/commands/clean.go`: use `state.NewReadOnlyStateStore`, call `ListPipelineNamesByStatus`.
   - `cmd/wave/commands/list.go`: replace `collectRunsFromDB` with a call to `state.NewReadOnlyStateStore` + `store.ListRuns(state.ListRunsOptions{...})`. Map results to `RunInfo`.
   - `internal/webui/server.go`: `backfillRunTokens` becomes `s.rwStore.BackfillRunTokens()` and the function (along with `database/sql` and `_ "modernc.org/sqlite"` imports) is deleted.

4. **Migrate tests**: rewrite `logs_test.go`, `decisions_test.go`, `list_test.go`, `status_test.go` (only the test imports `database/sql`) to seed fixtures via `state.NewStateStore` + the existing write methods (`CreateRun`, `LogEvent`, `RecordDecision`, etc.). Where direct ID control is needed, use `RecordStepAttempt` / `LogEvent` ordering since IDs are auto-increment in arrival order.

5. **Add depguard rule** to `.golangci.yml`:
   ```yaml
   no-direct-sql:
     files:
       - "**/*.go"
     deny:
       - pkg: "database/sql"
         desc: "ADR-007: only internal/state may import database/sql"
   ```
   Whitelist `internal/state/**` via depguard's `files` exclusion (use a separate rule, since depguard v2 matches by file pattern only; the standard pattern is to scope the rule's `files:` to everything *except* `internal/state/**`).

## 3. File Mapping

### Modified
- `internal/state/types.go` — add `DecisionQueryOptions`, `EventAggregateStats`; extend `EventQueryOptions`.
- `internal/state/store.go` — add new interface methods + impls (~150 LOC additions).
- `internal/state/store_test.go` — unit tests for new methods.
- `cmd/wave/commands/logs.go` — drop `database/sql`, route through StateStore (~150 LOC delta, mostly removals).
- `cmd/wave/commands/logs_test.go` — re-seed via StateStore.
- `cmd/wave/commands/decisions.go` — drop `database/sql`, route through StateStore.
- `cmd/wave/commands/decisions_test.go` — re-seed via StateStore.
- `cmd/wave/commands/clean.go` — drop `database/sql`, route through StateStore.
- `cmd/wave/commands/list.go` — drop `database/sql`, route through StateStore.
- `cmd/wave/commands/list_test.go` — re-seed via StateStore.
- `cmd/wave/commands/status_test.go` — re-seed via StateStore (only the test file imports `database/sql`).
- `internal/webui/server.go` — drop `backfillRunTokens` raw SQL, call `BackfillRunTokens()`.
- `.golangci.yml` — add `no-direct-sql` depguard rule.
- `docs/adr/007-consolidate-database-access-through-statestore-interface.md` — flip status from "Proposed (not started)" to "Accepted (implemented)" with the resolved bypass list.

### Created
- (none required — all changes land in existing files)

### Deleted
- (none)

## 4. Architecture Decisions

- **Reuse the existing `StateStore` interface** rather than introduce per-domain sub-interfaces. ADR-007 explicitly defers segregation to ADR-002. The +8 method growth is acceptable and documented in the ADR.
- **Read-only commands use `NewReadOnlyStateStore`** to keep the existing `?mode=ro` + `query_only=ON` defense-in-depth.
- **Keep `LogsEntry`/`DecisionEntry` as presentation DTOs** in the `commands` package; do not leak `state.LogRecord` field names into JSON output (output schema is part of the public CLI surface).
- **`BackfillRunTokens` is idempotent** and only touches finalized runs — same semantics as today, just moved into `internal/state`.
- **`depguard` rule is the enforcement mechanism**, not a docs-only TODO. Acceptance criterion #5 says "CI lint (or TODO issue)"; we choose the lint.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Behavioural drift in `logs --follow` (timing, missed events) | Keep ticker cadence and `AfterID` semantics identical; add a regression test that exercises follow over a polling tick. |
| Performance regression: each CLI invocation now runs migrations on connect | Use `NewReadOnlyStateStore` for read-only commands (skips migrations entirely). `clean.go` and `list.go` are already read-only conceptually. |
| Test fixture rewrite churn | Keep helper struct (`logsTestHelper`) but replace its body to use `state.NewStateStore` + write methods. Unit-test contracts unchanged. |
| `depguard` rule blocks legitimate use in `internal/state` | Rule scope explicitly excludes `internal/state/**`; verified via running `go run ./cmd/wave` then `golangci-lint run`. |
| Webui callers of `backfillRunTokens` lose log message | Keep equivalent log line in caller (`s.NewServer`) when the helper returns >0 rows. |
| `JSON output shape` change for `wave logs --format json` etc. | Snapshot tests / table-driven assertions in existing `*_test.go` will catch any drift. |

## 6. Testing Strategy

- **Unit (state package)**: new tests in `internal/state/store_test.go` for each added method (`GetMostRecentRunID`, `RunExists`, `GetRunStatus`, `GetEventAggregateStats`, `GetDecisionsFiltered`, `ListPipelineNamesByStatus`, `BackfillRunTokens`, extended `EventQueryOptions` filters). Cover empty DB, single row, multi-row, error cases.
- **Unit (commands)**: existing `*_test.go` files keep their public assertions but seed fixtures via `state.StateStore` write methods. Add a regression test that opens via the CLI `runLogs` path against an isolated tmp dir.
- **Lint**: run `golangci-lint run ./...` to confirm the new `depguard` rule fires on a deliberately added test file (and only there), then remove it.
- **End-to-end**: run `wave logs`, `wave decisions`, `wave clean --status failed`, `wave list runs` against a populated `.agents/state.db` from a real pipeline run; diff text output before/after to confirm parity.
- **CI**: `go vet ./...`, `go build ./...`, `go test -race ./...`, `golangci-lint run ./...` all pass.
- **Boundary verification**: `grep -R "database/sql" --include='*.go' . | grep -v '^./internal/state/'` returns no matches.
