# Implementation Plan

## Objective

Remove residual "backward compatible" comments from pipeline validation and resume code, and review whether the Down migration infrastructure should be kept or removed.

## Approach

This is a minimal cleanup: two comment edits and one architectural decision about migration rollback infrastructure.

### Comment Removal (validation.go, resume.go)

The comments mark prototype-specific code paths as "backward compatible" — but these aren't shims, they're the actual implementation for prototype pipelines. The parenthetical is misleading. Remove the "(backward compatible)" text from both comments while keeping the descriptive part.

### Down Migration Review (migrations.go)

**Decision: Keep the Down migration infrastructure.**

Rationale:
- The `Migration` struct's `Down` field, `RollbackMigration()`, `MigrateDown()`, and `wave migrate down` CLI command form standard migration infrastructure
- All 10 current migrations have `Down: ""` — this is expected since SQLite `ALTER TABLE` and `CREATE TABLE` operations are hard to reverse safely
- The infrastructure isn't a "backward-compatibility shim" — it's forward-looking capability for future migrations that might need rollback
- The `wave migrate down` CLI command is a documented user-facing feature
- Removing it would require deleting the CLI command, test coverage, and struct field — high churn for no functional benefit

No changes needed for migrations.go.

## File Mapping

| File | Action | Change |
|------|--------|--------|
| `internal/pipeline/validation.go` | modify | Remove "(backward compatible)" from line 32 comment |
| `internal/pipeline/resume.go` | modify | Remove "(backward compatible)" from line 567 comment |

## Architecture Decisions

- **Keep deprecated.go**: Intentional UX shim for pipeline name resolution — not a backward-compat artifact to remove
- **Keep Down migration infrastructure**: Standard migration pattern, not a backward-compat shim

## Risks

- **None significant** — this is a comment-only change with no behavioral impact

## Testing Strategy

- Run `go test ./...` to confirm no regressions
- No new tests needed — comments don't affect behavior
