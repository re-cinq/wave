# chore: remove backwards-compatibility shims and reduce accidental complexity

**Issue**: [re-cinq/wave#115](https://github.com/re-cinq/wave/issues/115)
**Labels**: chore, refactor, tech-debt
**Author**: nextlevelshit
**State**: OPEN

## Summary

Wave is in active prototype development. Per `CLAUDE.md`: *"Backward compatibility is NOT a constraint during prototype phase. We move fast and let tests catch regressions."* This issue tracks the removal of all backwards-compatibility shims and associated accidental complexity before they accumulate further.

## Background

Backwards compatibility concerns in this codebase may appear in several forms:
- **Config/manifest schema**: old field names or deprecated YAML keys still supported
- **Database migrations**: migration `Down` paths that exist only to preserve old schema shapes
- **API contracts**: output schemas that include deprecated fields for consumer compatibility
- **Code paths**: conditional logic that handles both old and new formats simultaneously

Since we are pre-v1.0.0, none of these need to be preserved.

## Tasks

- [ ] Search codebase for references to "backwards compat", "backward compat", "deprecated", "legacy" and evaluate each
- [ ] Search for dual-path conditional logic (e.g. `if oldFormat ... else newFormat`) and collapse to the new path
- [ ] Review `internal/state/migration_definitions.go` — remove `Down` SQL that only exists for backwards compat (not genuine rollback safety)
- [ ] Review `internal/manifest/` for deprecated field aliases or fallback parsing
- [ ] Review `internal/workspace/` and `internal/pipeline/` for compat shims
- [ ] Remove any renamed variables kept only for compat (e.g. `oldField`, `legacyX`)
- [ ] Run `go test ./...` after each removal to confirm no regressions

## Acceptance Criteria

- [ ] All code paths that exist solely for backwards compatibility are removed
- [ ] No remaining references to "backwards compat" in source comments or code (documentation excluded)
- [ ] `go test -race ./...` passes
- [ ] `go vet ./...` reports no issues
- [ ] PR description links back to this issue and lists specific packages changed

## Out of Scope

- Removing functionality used by current consumers or tests (that is a separate refactor)
- Changing public API behaviour — this is internal cleanup only
- Post-v1.0.0 compatibility commitments (tracked separately)

## References

- `CLAUDE.md` — "Backward compatibility is NOT a constraint during prototype phase"
- `internal/state/migration_definitions.go` — migration system
- `internal/manifest/` — config loading
