# audit: partial — backward-compat shims remaining (#115)

**Issue**: [#500](https://github.com/re-cinq/wave/issues/500)
**Source**: [#115](https://github.com/re-cinq/wave/issues/115) — chore: remove backwards-compatibility shims and reduce accidental complexity
**Category**: audit — partial remediation
**Labels**: audit
**Author**: nextlevelshit

## Summary

Wave-audit pipeline detected residual backward-compatibility comments and infrastructure from the prototype era. While most shims have been removed, several remain:

1. `internal/pipeline/validation.go:32` — comment "Prototype-specific validation (backward compatible)" on code that handles `impl-prototype`/`prototype` pipeline names
2. `internal/pipeline/resume.go:567` — comment "Prototype-specific logic (backward compatible)" on code that handles prototype resume points
3. `internal/state/migrations.go` — Down migration infrastructure (`RollbackMigration`, `MigrateDown`, `Migration.Down` field) exists but all 10 migration definitions have `Down: ""`
4. `internal/pipeline/deprecated.go` — `ResolveDeprecatedName` taxonomy mappings (intentional UX shim, leave as-is)

## Acceptance Criteria

- [ ] Remove "backward compatible" parenthetical from comment at `validation.go:32`
- [ ] Remove "backward compatible" parenthetical from comment at `resume.go:567`
- [ ] Review Down migration paths — decide whether to keep or remove the infrastructure
- [ ] `deprecated.go` left unchanged (intentional UX shim)
- [ ] All tests pass (`go test ./...`)
