# Work Items

## Phase 1: Setup
- [ ] 1.1: Verify branch `1168-backfill-tokens-migration` is checked out and clean.
- [ ] 1.2: Confirm v23 is highest existing migration; v24 is next.

## Phase 2: Core Implementation
- [ ] 2.1: Append `Migration{Version: 24, Description: "Backfill pipeline_run.total_tokens from event_log for legacy runs", Up: <UPDATE SQL>, Down: ""}` to `GetAllMigrations()` in `internal/state/migration_definitions.go`. Use the same SQL currently in `backfillRunTokens` (UPDATE `pipeline_run` SET `total_tokens` = SUM(event_log.tokens_used) WHERE `total_tokens = 0` AND status IN ('completed','failed','cancelled')).
- [ ] 2.2: Remove the `backfillRunTokens(cfg.DBPath)` call from `NewServer` in `internal/webui/server.go` (around line 101).
- [ ] 2.3: Delete the `backfillRunTokens` function body in `internal/webui/server.go` (lines 215-242).
- [ ] 2.4: Remove now-unused imports (`database/sql`, `_ "modernc.org/sqlite"` if only used by the backfill) from `internal/webui/server.go`. Run `goimports`.

## Phase 3: Testing
- [ ] 3.1: Add a test in `internal/state/migrations_test.go` that seeds a DB with mixed `pipeline_run` rows (zero-token completed + non-zero completed) and `event_log` entries, applies through v24, and asserts only the zero-token rows are backfilled.
- [ ] 3.2: Run `go test -race ./internal/state/... ./internal/webui/...` and confirm green.
- [ ] 3.3 [P]: Run `go build ./...` to confirm no orphaned imports.
- [ ] 3.4 [P]: Run `golangci-lint run ./internal/state/... ./internal/webui/...`.

## Phase 4: Polish
- [ ] 4.1: Update ADR-007 reference (`docs/adr/007-consolidate-database-access-through-statestore-interface.md`) note where it describes relocating `backfillRunTokens`, to reflect that it became migration v24 rather than a relocated function.
- [ ] 4.2: Final sanity: grep repo for any remaining `backfillRunTokens` references; only ADR mentions should remain (now updated).
- [ ] 4.3: Commit with conventional message: `refactor(webui): move token backfill into versioned migration v24` and open PR linking #1168.
