# refactor: route CLI state reads through internal/state.StateStore (logs, decisions, clean, list)

**Issue:** [#1275](https://github.com/re-cinq/wave/issues/1275)
**Repository:** re-cinq/wave
**Labels:** `scope-audit`
**State:** OPEN
**Author:** nextlevelshit

## Context

From wave-scope-audit run `wave-scope-audit-20260422-223006-df0b`.

ADR-007 requires `internal/state.StateStore` be the only path to the SQLite database. Several CLI commands (logs, decisions, clean, list) currently bypass the StateStore and use `database/sql` directly, violating the boundary and preventing consistent read semantics, migrations, and instrumentation.

Scope governance rule: *"No raw `database/sql` imports outside `internal/state`. CLI commands, webui, and tui must consume `StateStore` methods."*

Related to #1159 (splitting StateStore into domain-scoped interfaces). This issue is the routing change that makes that split meaningful.

## Acceptance Criteria

- [ ] Identify every `database/sql` import outside `internal/state` (expected: `cmd/` logs, decisions, clean, list subcommands).
- [ ] Add read methods on `StateStore` (or one of its domain-scoped interfaces) that cover the queries those commands need.
- [ ] Replace the raw SQL in those commands with `StateStore` calls.
- [ ] `go vet`/`go build`/`go test` pass, and a repo-wide grep confirms no `database/sql` imports outside `internal/state`.
- [ ] Add a CI lint (or TODO issue) that fails on new `database/sql` imports outside `internal/state`.

## Confirmed Bypass Sites (2026-04-27)

`grep` against the working tree (matches ADR-007 status block):

| File | Reason |
|------|--------|
| `cmd/wave/commands/logs.go` | `sql.Open("sqlite", ...)`, 7 helper funcs taking `*sql.DB`, includes follow-mode polling and perf aggregation queries |
| `cmd/wave/commands/decisions.go` | `sql.Open`, `queryDecisions()` with category filter not exposed in `StateStore.GetDecisions` |
| `cmd/wave/commands/clean.go` | `sql.Open`, `getWorkspacesWithStatus()` queries `pipeline_run`/`pipeline_state` for distinct pipeline names by status |
| `cmd/wave/commands/list.go` | `sql.Open`, `collectRunsFromDB()` duplicates `StateStore.ListRuns` logic |
| `internal/webui/server.go` | `sql.Open`, `backfillRunTokens()` runs raw `UPDATE ... SUM(tokens_used)` migration |
| `cmd/wave/commands/logs_test.go`, `decisions_test.go`, `list_test.go`, `status_test.go` | Test fixtures use `sql.Open` to seed rows directly |

## Reference

- ADR-007 — Consolidate Database Access Through StateStore Interface (`docs/adr/007-consolidate-database-access-through-statestore-interface.md`)
- ADR-003 — Layered architecture (Presentation must not import infrastructure-specific drivers)
- Issue #1159 — Split StateStore into domain-scoped interfaces (downstream)
- Issue URL: https://github.com/re-cinq/wave/issues/1275
