# Implementation Plan: Remove Backwards-Compatibility Shims

## Objective

Remove all code paths, fields, functions, and comments that exist solely for backward compatibility in Wave's prototype-phase codebase. This reduces accidental complexity and makes the codebase easier to reason about.

## Approach

Work through the codebase package-by-package, evaluating each discovered shim. For each:
1. Determine if it's a true compat shim (removable) vs. functional code labeled as "legacy" (keep but relabel)
2. Remove or consolidate the code
3. Update tests that exercise the compat path
4. Run `go test ./...` to verify no regressions

Categorize shims into three groups:
- **Remove**: Pure compat code with no current consumers
- **Consolidate**: Dual-path code where one path can be eliminated
- **Relabel**: Code that works correctly but is misleadingly labeled "legacy"

## File Mapping

### Remove

| File | Action | What |
|------|--------|------|
| `internal/contract/contract.go` | modify | Remove `StrictMode` field, replace all usages with `MustPass` |
| `internal/contract/jsonschema.go` | modify | Replace `StrictMode` references with `MustPass` |
| `internal/contract/typescript.go` | modify | Remove `IsTypeScriptAvailable()` wrapper, inline to `CheckTypeScriptAvailability()` |
| `internal/contract/typescript_test.go` | modify | Update test for removed `IsTypeScriptAvailable` |
| `internal/contract/contract_test.go` | modify | Replace `StrictMode` with `MustPass` in tests |
| `internal/pipeline/executor.go:569-576` | modify | Remove handover MaxRetries fallback, use only `Retry.EffectiveMaxAttempts()` |
| `internal/pipeline/executor.go:1142` | modify | Remove `StrictMode` mapping in contract config construction |
| `internal/pipeline/executor_test.go:3423-3454` | modify | Remove `TestExecuteStep_RetryConfig_BackwardCompat` test |
| `internal/state/migration_definitions.go` | modify | Remove `Down` SQL from all migrations (keep Up only) |

### Consolidate

| File | Action | What |
|------|--------|------|
| `internal/pipeline/meta.go` | modify | Remove `extractYAMLLegacy()` fallback; require `--- PIPELINE ---` marker |
| `internal/pipeline/meta_test.go` | modify | Remove tests for legacy YAML extraction |
| `internal/contract/json_cleaner.go` | modify | Remove `extractJSONFromTextLegacy()` fallback; return error when recovery fails |
| `internal/state/store.go` | modify | Remove old schema.sql fallback path; always use migration system |
| `cmd/wave/commands/list.go` | modify | Remove `collectRunsFromPipelineState()` and its fallback call |
| `cmd/wave/commands/output.go` | modify | Remove direct flag reading fallback in `GetOutputConfig` |

### Relabel / Minor Cleanup

| File | Action | What |
|------|--------|------|
| `internal/display/types.go` | modify | Remove "backward compat" comment; fields are still functional |
| `internal/display/bubbletea_progress.go` | modify | Remove "backward compat" comment; logic is still correct |
| `internal/pipeline/context.go:103` | modify | Remove "legacy" comment; these are standard template variables |
| `internal/pipeline/resume.go:254` | modify | Remove "legacy" comment; this is valid fallback behavior |
| `internal/tui/pipeline_list.go:28` | modify | Remove compat comment from `itemKindAvailable` |
| `internal/worktree/worktree.go:92` | modify | Remove "legacy behavior" comment; this is default behavior |
| `internal/doctor/doctor.go:190` | modify | Change message from "Wave project detected (legacy)" to "Wave project detected" |
| `internal/pipeline/types.go:199` | modify | Remove "legacy directory" from comment |
| `cmd/wave/commands/list.go:991` | modify | Remove "legacy workspace" from comment |
| `internal/contract/testsuite.go:101` | modify | Remove "backward compatible" from comment |
| `internal/contract/jsonschema.go:170` | modify | Remove "backward compatibility" from comment |

## Architecture Decisions

1. **StrictMode → MustPass migration**: `StrictMode` is deprecated in favor of `MustPass`. Since they serve the same purpose, replace all `StrictMode` usage with `MustPass` directly. This is a field rename, not a behavior change.

2. **Migration Down paths**: Remove all `Down` SQL. In a pre-v1.0.0 prototype, rollback migrations add complexity with no benefit. If schema needs to change, a new Up migration is written.

3. **extractYAMLLegacy removal**: The meta-pipeline now uses `--- PIPELINE ---` / `--- SCHEMAS ---` markers. The old code-block extraction format is dead code.

4. **json_cleaner legacy fallback**: The progressive recovery system in `json_cleaner.go` is the primary path. The legacy brace-matching fallback adds a second code path. If progressive recovery fails, returning an error is cleaner than falling through to a less capable algorithm.

5. **State store schema.sql fallback**: The migration system is the standard path. The old `schema.sql` direct-read path is dead code when migrations are enabled.

6. **collectRunsFromPipelineState**: The old `pipeline_state` table is superseded by `pipeline_run`. The fallback only triggers on query failure of the new table, which would indicate a broken database rather than a schema mismatch worth handling.

7. **Handover MaxRetries**: The new `RetryConfig` with `max_attempts` is the intended mechanism. The `Handover.MaxRetries` and `Handover.Contract.MaxRetries` fallback is pure compat. However, note that existing pipeline YAML files may still use `handover.max_retries` — need to verify no active pipelines depend on this.

## Risks

| Risk | Mitigation |
|------|-----------|
| Active pipeline YAML files use `handover.max_retries` | Grep all `.wave/pipelines/` and `internal/defaults/pipelines/` for `max_retries` usage; migrate to `retry.max_attempts` before removing |
| Migration Down removal could complicate future development | Pre-v1.0.0, Down paths are not needed. New Up migrations handle schema changes |
| `extractYAMLLegacy` may still be hit by some meta-pipeline output | Verify all meta-pipeline personas output the new `--- PIPELINE ---` format |
| StrictMode removal breaks test configs | Comprehensive test update in same PR |
| collectRunsFromPipelineState removal breaks `wave list` for old databases | Old databases without `pipeline_run` table should trigger migration on next `wave run`, not rely on compat query |

## Testing Strategy

1. **Per-change validation**: Run `go test ./...` after each package modification
2. **Race detection**: Final `go test -race ./...` pass
3. **Static analysis**: `go vet ./...` must pass cleanly
4. **Pipeline YAML audit**: Grep all pipeline definitions for removed field names
5. **Remove dedicated compat tests**: `TestExecuteStep_RetryConfig_BackwardCompat`, `TestIsTypeScriptAvailable` (wrapper test)
6. **Update remaining tests**: Replace `StrictMode` with `MustPass` in all test configurations
