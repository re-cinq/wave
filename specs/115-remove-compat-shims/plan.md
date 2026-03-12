# Implementation Plan: Remove Backwards-Compatibility Shims

## 1. Objective

Remove all backward-compatibility shims, legacy fallbacks, and deprecated code paths from the Wave codebase. The project is pre-v1.0.0 and explicitly states backward compatibility is not a constraint during the prototype phase.

## 2. Approach

Systematically audit each identified compat shim, determine whether it serves current functionality or is purely legacy, and remove those that are purely legacy. Each removal is validated by running `go test ./...` to ensure no regressions.

The work is grouped by package to minimize cognitive overhead and ensure no cross-cutting dependencies are missed.

## 3. File Mapping

### Files to Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/state/migration_definitions.go` | modify | Remove all migration `Down` fields — forward-only migrations in prototype phase |
| `internal/contract/contract.go` | modify | Remove deprecated `StrictMode` field from `ContractConfig` |
| `internal/contract/jsonschema.go` | modify | Remove `StrictMode` fallback logic, use only `MustPass` |
| `internal/contract/typescript.go` | modify | Remove `IsTypeScriptAvailable()` compat wrapper, inline into callers or replace with `CheckTypeScriptAvailability()` |
| `internal/contract/json_cleaner.go` | modify | Remove `extractJSONFromTextLegacy` function and its fallback call path |
| `internal/pipeline/executor.go:468-476` | modify | Remove `Handover.MaxRetries` / `Handover.Contract.MaxRetries` fallback for retry logic |
| `internal/pipeline/executor.go:1024` | modify | Remove `StrictMode` mapping from handover contract |
| `internal/pipeline/executor.go:2404` | modify | Clean up "legacy state store" comment |
| `internal/pipeline/types.go:192` | modify | Update `WorkspaceConfig.Type` comment — remove "legacy directory" wording |
| `internal/pipeline/types.go:233-238` | modify | Remove `MaxRetries` from `HandoverConfig` |
| `internal/pipeline/context.go:103-107` | modify | Remove legacy template variables (`pipeline_id`, `pipeline_name`, `step_id` without `pipeline_context.` prefix) |
| `internal/pipeline/meta.go:578-630` | modify | Remove `extractYAMLLegacy` function and its fallback call path |
| `internal/pipeline/resume.go:254` | modify | Remove exact-name directory fallback for legacy workspaces |
| `internal/display/types.go:270-272` | modify | Remove "backward compat" comment from global tool activity fields |
| `internal/display/bubbletea_progress.go:377` | modify | Remove "backward compat" comment |
| `internal/contract/testsuite.go:101` | modify | Update comment — "backward compatible" → just describe default behavior |
| `internal/doctor/doctor.go:190` | modify | Remove "legacy" wording from Wave project detection |
| `internal/worktree/worktree.go:92` | modify | Update comment — "legacy behavior" → "default: branch from HEAD" |
| `cmd/wave/commands/output.go:107` | modify | Remove "backward compatibility" comment |
| `cmd/wave/commands/list.go:825-885` | modify | Remove `collectRunsFromPipelineState` legacy table reader |
| `cmd/wave/commands/list.go:991` | modify | Update comment — "legacy workspace" → describe the fallback |

### Test Files to Update

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/executor_test.go:3423-3456` | modify | Remove `TestExecuteStep_RetryConfig_BackwardCompat` test |
| `internal/pipeline/meta_test.go:445-447` | modify | Remove `extractYAMLLegacy` test references |
| `internal/pipeline/context_test.go:41-44` | modify | Remove `legacy_variables` test case |
| `internal/contract/contract_test.go` | modify | Replace `StrictMode` with `MustPass` in test configs |
| `internal/contract/typescript_test.go:317-327` | modify | Remove `TestIsTypeScriptAvailable` compat test, update callers |

### Files Confirmed Clean (no changes needed)

| File | Reason |
|------|--------|
| `internal/manifest/` | No deprecated field aliases or fallback parsing found |
| `internal/workspace/` | No compat shims found |

## 4. Architecture Decisions

1. **Migration `Down` fields**: Remove entirely. In prototype phase, we only migrate forward. If a rollback is ever needed, it should be a new forward migration that undoes the change. The `Migration` struct's `Down` field stays (it's part of the type) but all values become empty strings.

2. **`StrictMode` → `MustPass`**: `StrictMode` is marked deprecated in favor of `MustPass`. Remove `StrictMode` field entirely and update all references to use `MustPass`. This is a clean rename — no semantic change.

3. **`Handover.MaxRetries` removal**: The new `RetryConfig` on steps is the canonical retry configuration. Remove the fallback that reads `Handover.MaxRetries` and `Handover.Contract.MaxRetries`. Any YAML files still using the old field will simply have it ignored.

4. **Legacy template variables**: The `pipeline_id`, `pipeline_name`, `step_id` shortcuts (without `pipeline_context.` prefix) are legacy. Remove them — the canonical form is `{{pipeline_context.pipeline_id}}`.

5. **`extractYAMLLegacy`**: The meta-pipeline output format now uses `--- PIPELINE ---` / `--- SCHEMAS ---` markers. The old code-block fallback is no longer needed.

6. **`collectRunsFromPipelineState`**: Determine if callers still reference this. If the `pipeline_run` table (v2+) is the sole source of truth, remove the legacy `pipeline_state` reader.

7. **`IsTypeScriptAvailable()`**: Simple wrapper around `CheckTypeScriptAvailability()`. Replace all callers with the richer function or a direct `available, _ := CheckTypeScriptAvailability()` pattern.

8. **`extractJSONFromTextLegacy`**: The progressive JSON recovery system (`RecoverJSON`) is the current path. The legacy bracket-matching fallback can be removed — if recovery fails, that's a hard error.

## 5. Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Removing `Handover.MaxRetries` breaks existing YAML pipelines | Medium | Audit all `.wave/pipelines/*.yaml` for `max_retries` usage under `handover:` |
| Removing legacy template variables breaks pipeline prompts | Medium | Grep all `.wave/` YAML/MD files for `{{pipeline_id}}` etc. |
| Removing `collectRunsFromPipelineState` breaks `wave list` for old databases | Low | The v2 `pipeline_run` table should be the canonical source; old databases would need re-migration anyway |
| Removing `extractYAMLLegacy` breaks meta-pipeline output parsing | Low | Current meta-pipelines use the new marker format |
| Removing migration `Down` blocks prevents future rollbacks | Low | Forward-only migrations are standard practice; rollbacks are new forward migrations |

## 6. Testing Strategy

1. **After each file group change**: Run `go test ./...` to catch regressions immediately
2. **After all changes**: Run `go test -race ./...` for race condition detection
3. **After all changes**: Run `go vet ./...` for static analysis
4. **Grep validation**: Confirm no remaining "backward compat" / "legacy" references in source code (excluding docs, specs, and the issue spec itself)
5. **Build validation**: Ensure `go build ./...` succeeds
