# Research: Remove Backwards-Compatibility Shims

**Feature**: 115-remove-compat-shims
**Date**: 2026-02-20
**Status**: Complete

## Phase 0 — Unknowns & Research

### Zero NEEDS CLARIFICATION markers

The spec was self-validated with 3 iterations and has zero unresolved unknowns. All 5 clarifications (C-001 through C-005) were resolved during spec authoring. The research below consolidates the codebase investigation findings needed for implementation planning.

---

## Research Area 1: StrictMode Removal (FR-001, FR-002)

### Decision: Direct substitution of `StrictMode` → `MustPass`

**Rationale**: `executor.go:675` already sets `StrictMode = step.Handover.Contract.MustPass`, proving the fields are semantically identical. All downstream consumers use the same boolean for the same purpose.

**Affected Files**:
| File | Line(s) | Current Code | Action |
|------|---------|-------------|--------|
| `internal/contract/contract.go:18` | 18 | `StrictMode bool \`json:"strictMode,omitempty"\`` | Remove field |
| `internal/pipeline/executor.go:675` | 675 | `StrictMode: step.Handover.Contract.MustPass,` | Remove line |
| `internal/pipeline/executor.go:700` | 700 | `if contractCfg.StrictMode {` | Change to `contractCfg.MustPass` |
| `internal/contract/jsonschema.go:266-267` | 266-267 | `if !cfg.MustPass && cfg.StrictMode { mustPass = cfg.StrictMode }` | Remove the `if` block, `mustPass` already set from `cfg.MustPass` at L265 |
| `internal/contract/jsonschema.go:333` | 333 | Comment: `If StrictMode is false` | Update comment |
| `internal/contract/jsonschema.go:346` | 346 | `if !cfg.StrictMode {` | Change to `if !cfg.MustPass {` |
| `internal/contract/typescript.go:25` | 25 | `if cfg.StrictMode {` | Change to `if cfg.MustPass {` |

**Test Files Requiring Updates**:
| File | References |
|------|-----------|
| `internal/contract/contract_test.go:214,246,260,269` | `StrictMode: true/false` → `MustPass: true/false` |
| `internal/contract/typescript_test.go:29,38,117,130,248` | `StrictMode: true/false` → `MustPass: true/false` |

**NOT in scope**: `internal/security/config.go:27-28` (`StrictMode` on `SanitizationConfig`) and `internal/pipeline/executor_schema_test.go:242,719` (`securityConfig.Sanitization.StrictMode`) — these are completely separate concepts.

**Alternatives Rejected**:
- Keep `StrictMode` as an alias — adds complexity, contradicts prototype "move fast" ethos
- Rename `MustPass` to `StrictMode` — `MustPass` is the newer, clearer name per convention

---

## Research Area 2: IsTypeScriptAvailable Wrapper Removal (FR-002)

### Decision: Remove wrapper, update all callers to use `CheckTypeScriptAvailability()`

**Rationale**: `IsTypeScriptAvailable()` at `typescript.go:95-100` is explicitly marked "kept for backward compatibility" and simply wraps `CheckTypeScriptAvailability()`, discarding the version return value.

**Callers to update**:
| File | Line | Current Call | Replacement |
|------|------|-------------|-------------|
| `internal/contract/contract_test.go:252` | 252 | `IsTypeScriptAvailable()` | `available, _ := CheckTypeScriptAvailability(); if !available {` |
| `internal/contract/contract_test.go:273` | 273 | `IsTypeScriptAvailable()` | Same pattern |
| `internal/contract/typescript_test.go:46` | 46 | `IsTypeScriptAvailable()` | Same pattern |
| `internal/contract/typescript_test.go:100` | 100 | `IsTypeScriptAvailable()` | Already has `tscAvailable := IsTypeScriptAvailable()` |
| `internal/contract/typescript_test.go:188` | 188 | `IsTypeScriptAvailable()` | Same pattern |
| `internal/contract/typescript_test.go:317-327` | 317 | `TestIsTypeScriptAvailable` test | Delete entire test |

**Alternatives Rejected**:
- Keep the wrapper — violates "no compat shims" goal, adds dead code

---

## Research Area 3: Legacy JSON/YAML Extraction Fallback Removal (FR-003, FR-004)

### Decision: Remove `extractJSONFromTextLegacy` and `extractYAMLLegacy`

**JSON Legacy (FR-003)**:
- File: `internal/contract/json_cleaner.go:83-148`
- Called from: `json_cleaner.go:80` (fallback when recovery parser fails)
- Action: Remove `extractJSONFromTextLegacy` method. Modify `ExtractJSONFromText` to return the recovery parser error instead of falling back.

**YAML Legacy (FR-004)**:
- File: `internal/pipeline/meta.go:604-630`
- Called from: `meta.go:579` (fallback when `--- PIPELINE ---` marker not found)
- Action: Remove `extractYAMLLegacy` function. Change fallback at `meta.go:577-580` to return an error with a clear message about the required `--- PIPELINE ---` / `--- SCHEMAS ---` format.

**Alternatives Rejected**:
- Keep legacy as optional — contradicts the spec's requirement for a single extraction path
- Log warning and use legacy — still maintains dual paths

---

## Research Area 4: Migration Down Path Removal (FR-005, FR-015)

### Decision: Set all `Down` SQL fields to empty strings, update CLI help text

**Rationale**: Per C-002, the `Down` field stays on the `Migration` struct but all values become `""`. The existing check at `migrations.go:269` (`if fullMigration.Down == "" { return fmt.Errorf(...) }`) already handles this naturally.

**Affected Files**:
| File | Action |
|------|--------|
| `internal/state/migration_definitions.go` | Set all 6 `Down:` values to `""` |
| `cmd/wave/commands/migrate.go:72-114` | Update `Long` description, remove confirmation prompt |

**CLI Command Changes**:
- Update `--long` description to state rollback is not supported in prototype phase
- Remove the confirmation prompt (lines 96-104) since rollback always fails anyway
- Let the existing `MigrateDown()` error provide the runtime failure message

**Alternatives Rejected**:
- Remove `Down` field from struct — cascading changes to `RollbackMigration()` and `MigrateDown()` signatures, higher churn for no functional benefit
- Return error in CLI before calling MigrateDown — the existing error from `migrations.go:270` is clear enough

---

## Research Area 5: Legacy State Store Fallback Removal (FR-006)

### Decision: Remove schema.sql fallback, error on `WAVE_MIGRATION_ENABLED=false`

**Affected Files**:
| File | Action |
|------|--------|
| `internal/state/store.go:6` | Remove `"embed"` import |
| `internal/state/store.go:15-17` | Remove `go:embed schema.sql` directive and `schemaFS` variable |
| `internal/state/store.go:150-165` | Replace `if/else` with: always use migrations, error if `ShouldUseMigrations()` is false |
| `internal/state/schema.sql` | Delete file |
| `internal/state/migration_config.go:59-62` | Update `ShouldUseMigrations()` — or change callsite in store.go to return error if false |

**Error Message** (when `WAVE_MIGRATION_ENABLED=false`):
```
legacy schema initialization has been removed; migrations are now the only supported method — remove the WAVE_MIGRATION_ENABLED=false setting
```

**Alternatives Rejected**:
- Silently ignore the flag — violates principle of least surprise
- Keep schema.sql as a backup — contradicts the cleanup goal

---

## Research Area 6: Legacy Workspace Directory Lookup (FR-007)

### Decision: Remove exact-name directory fallback in resume.go

**Affected Code**: `internal/pipeline/resume.go:187-189`
```go
if info, err := os.Stat(filepath.Join(wsRoot, p.Metadata.Name)); err == nil && info.IsDir() {
    runDirs = append([]string{filepath.Join(wsRoot, p.Metadata.Name)}, runDirs...)
}
```

**Rationale**: All current workspaces use the `<name>-<timestamp>-<hash>` convention. The exact-name check is for a legacy format that no longer exists.

**Alternatives Rejected**:
- Keep for "just in case" — dead code serves no purpose in prototype

---

## Research Area 7: Legacy Comment Cleanup (FR-008 through FR-012)

### Decision: Update or remove all identified legacy comments

| File | Line | Current Comment | Action |
|------|------|----------------|--------|
| `internal/pipeline/types.go:77` | 77 | `empty for legacy directory` | Change to `empty for default directory workspace` |
| `internal/worktree/worktree.go:92` | 92 | `New branch from HEAD (legacy behavior)` | Change to `New branch from HEAD (default)` |
| `internal/pipeline/context.go:75` | 75 | `Handle legacy template variables` | Change to `Short-form template variables (primary format used by pipeline YAML)` |
| `internal/pipeline/executor.go:1454` | 1454 | `// Not tracked in legacy state store` | Change to `// Not available from pipeline_state record` |
| `internal/contract/jsonschema.go:333` | 333 | Comment mentioning `StrictMode` | Update to reference `MustPass` |
| `internal/contract/typescript.go:95-97` | 95-97 | `IsTypeScriptAvailable` and "backward compatibility" comment | Entire function removed per FR-002 |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Test breakage from StrictMode removal | Medium | Low | All test references mapped; mechanical find-replace |
| External YAML configs using `strictMode` key | Low | Medium | `json:"strictMode,omitempty"` tag removal means the field is silently ignored in JSON/YAML — no runtime error. Spec edge case says this should error, but YAML configs use `must_pass` key not `strictMode` |
| Migration checksums change when Down is emptied | Medium | Medium | `WAVE_SKIP_MIGRATION_VALIDATION=true` is already available; alternatively, recalculate checksums based on Up SQL only |
| Meta-pipeline output missing section markers | Low | Low | This is a deliberate behavioral change — the error message will guide users to the new format |
| `schema.sql` referenced by tests | Low | Low | Grep shows no test references to `schema.sql` directly |

## Dependency Order

The implementation should follow this order to minimize cascading failures:

1. **P1: StrictMode field removal** (FR-001) — touches most files but is mechanical
2. **P1: IsTypeScriptAvailable wrapper** (FR-002) — simple removal, test updates
3. **P1: JSON legacy fallback** (FR-003) — isolated to json_cleaner.go
4. **P1: YAML legacy fallback** (FR-004) — isolated to meta.go
5. **P2: Migration Down paths** (FR-005, FR-015) — isolated to state package + migrate command
6. **P2: schema.sql fallback** (FR-006) — isolated to state package
7. **P3: Legacy workspace lookup** (FR-007) — isolated to resume.go
8. **P3: Comment cleanup** (FR-008 through FR-012) — pure documentation changes
9. **Verification: `go test -race ./...` and `go vet ./...`** (FR-013, FR-014)
