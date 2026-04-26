# Implementation Plan — #1168

## Objective

Remove the data-mutation side effect from `webui.NewServer` by promoting `backfillRunTokens` to a versioned schema migration. Server construction must become side-effect-free.

## Approach

The existing `backfillRunTokens` logic is a single idempotent SQL `UPDATE` (guarded by `WHERE total_tokens = 0`). The state package already has a versioned migration framework (`internal/state/migrations.go`, `migration_definitions.go`) where each `Migration` is identified by an integer `Version` and carries an `Up` SQL string. Pending migrations are applied automatically when `state.NewStateStore` opens the DB.

Strategy:
1. Append a new `Migration{Version: 24}` to `GetAllMigrations()` containing the same `UPDATE pipeline_run ...` SQL the constructor runs today.
2. Delete `backfillRunTokens` (function + call site) from `internal/webui/server.go`.
3. Migration framework handles idempotency — `schema_migrations` row records that v24 ran, so it never runs twice. The `WHERE total_tokens = 0` clause keeps the SQL safe even if re-applied manually.

## File Mapping

**Modify:**
- `internal/state/migration_definitions.go` — append migration v24 with the backfill SQL.
- `internal/webui/server.go` — remove `backfillRunTokens(cfg.DBPath)` call (line 101) and the function body (lines 215-242). Drop `database/sql` import if no longer used; check.

**Test (new):**
- `internal/state/migrations_test.go` — extend with a test asserting v24 applies the backfill correctly (zero-token completed runs get summed from `event_log`; non-zero runs untouched).

**No changes:**
- `internal/state/migration_runner.go` — registration is automatic via `GetAllMigrations()`.
- `internal/state/store.go` — already runs `MigrateUp` on every open.

## Architecture Decisions

- **SQL-only migration (no Go hook).** The framework's `Migration.Up` is a SQL string. The current logic is pure SQL, so no extension to support Go-callback migrations is needed. ADR-007 anticipated relocating `backfillRunTokens` into the state package; expressing it as a versioned `Migration` is the cleanest realization.
- **Version 24.** Highest existing version is 23. New migration is appended.
- **No `Down` script.** The backfill is forward-only (recomputes derived data). Other backfill-style migrations in this file also have `Down: ""`.
- **Idempotency comes from two layers:** `schema_migrations` table prevents re-execution; the `WHERE total_tokens = 0` guard keeps SQL safe in any rerun scenario.

## Risks

- **Risk:** Existing deployments where v1-23 are already applied but the backfill never ran (e.g. fresh installs that opened the DB through a path that skipped `backfillRunTokens`). **Mitigation:** v24 will run as a pending migration on next `NewStateStore` open — same effect as the constructor call.
- **Risk:** Existing deployments where the backfill *did* run via the constructor; `pipeline_run` rows already have correct `total_tokens`. **Mitigation:** the SQL is filtered by `total_tokens = 0`, so already-correct rows are untouched. v24 effectively becomes a no-op `UPDATE 0 rows`.
- **Risk:** Removing `database/sql` import may break the build if other code in `server.go` still uses it. **Mitigation:** verify with `goimports` / `go build` after edit.

## Testing Strategy

- Unit test in `internal/state/migrations_test.go`: open a fresh DB, apply migrations through v23, seed `pipeline_run` (one row with `total_tokens=0` and matching `event_log` rows totalling N; one row with `total_tokens=5` and event_log summing to 99 — must remain 5). Apply v24, assert backfill happened only on the zero row.
- Existing webui tests must still pass with the constructor no longer mutating the DB.
- Run `go test -race ./internal/state/... ./internal/webui/...`.
