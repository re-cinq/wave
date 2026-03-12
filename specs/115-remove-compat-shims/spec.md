# chore: remove backwards-compatibility shims and reduce accidental complexity

**Issue**: [#115](https://github.com/re-cinq/wave/issues/115)
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

## Discovered Compat Shims (Codebase Analysis)

The following backward-compatibility shims were identified through codebase analysis:

### 1. `internal/contract/contract.go` — Deprecated `StrictMode` field
- `StrictMode bool` marked `// Deprecated: use MustPass instead` on `ContractConfig`
- Still actively used throughout `jsonschema.go`, `typescript.go`, `security/sanitize.go`
- Executor maps `step.Handover.Contract.MustPass` → `StrictMode` at line 1142

### 2. `internal/contract/typescript.go` — `IsTypeScriptAvailable()` wrapper
- Line 95-99: Wrapper kept "for backward compatibility" around `CheckTypeScriptAvailability()`
- Used in test files only (`contract_test.go`, `typescript_test.go`)

### 3. `internal/pipeline/executor.go` — Legacy handover retry fallback
- Lines 569-576: Falls back to `Handover.MaxRetries` / `Handover.Contract.MaxRetries` when `Retry.MaxAttempts` is not set
- New `RetryConfig` is the intended mechanism; legacy `max_retries` fields are compat shims

### 4. `internal/pipeline/meta.go` — `extractYAMLLegacy()` fallback
- Lines 578-580, 604+: Falls back to old output format parsing when `--- PIPELINE ---` marker is missing
- Function `extractYAMLLegacy` extracts YAML from code blocks (old meta-pipeline format)

### 5. `internal/contract/json_cleaner.go` — `extractJSONFromTextLegacy()` fallback
- Lines 80, 83-84: Fallback to original brace-matching extraction when progressive recovery fails

### 6. `internal/state/store.go` — Old schema.sql fallback
- Lines 159-174: Falls back to reading `schema.sql` file when `ShouldUseMigrations()` returns false
- Dual-path initialization (migration system vs. old schema)

### 7. `internal/state/migration_definitions.go` — `Down` SQL in migrations
- All 9 migrations have `Down` paths for rolling back
- Some Down paths (especially migrations 6-8 for ALTER TABLE) are complex table-recreation workarounds

### 8. `cmd/wave/commands/output.go` — `GetOutputConfig` fallback
- Lines 107-118: Falls back to direct flag reading when `ResolvedFlags` not in context

### 9. `cmd/wave/commands/list.go` — `collectRunsFromPipelineState` legacy fallback
- Lines 782-783, 825+: Falls back to old `pipeline_state` table when new `pipeline_run` query fails

### 10. `internal/display/types.go` — Global tool activity fallback
- Line 271-273: `LastToolName`/`LastToolTarget` kept as "global fallback for backward compat"

### 11. `internal/display/bubbletea_progress.go` — "Primary" step compat
- Line 383: First running step selected as "primary" for backward compat

### 12. `internal/pipeline/context.go` — Legacy template variables
- Line 103: `pipeline_id`, `pipeline_name`, `step_id` as "legacy template variables"

### 13. `internal/pipeline/resume.go` — Legacy workspace directory lookup
- Line 254: Checks for exact-name directory without hash suffix (legacy format)

### 14. `internal/tui/pipeline_list.go` — `itemKindAvailable` compat constant
- Line 28: `itemKindAvailable` kept "for PipelineSelectedMsg.Kind compatibility"

### 15. `internal/worktree/worktree.go` — "Legacy behavior" comment
- Line 92: Comment labels default branch-from-HEAD as "legacy behavior" (actually still correct behavior)

### 16. `internal/doctor/doctor.go` — "Legacy" project detection message
- Line 190: Message says "Wave project detected (legacy)" for wave.yaml-based projects

### 17. `internal/pipeline/types.go` — Workspace type comment
- Line 199: Comment mentions "empty for legacy directory" workspace type

### 18. `cmd/wave/commands/list.go` — Legacy workspace name fallback
- Line 991: Comment says "legacy workspace without run ID suffix"
