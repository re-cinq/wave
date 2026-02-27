# Feature Specification: Remove Backwards-Compatibility Shims

**Feature Branch**: `115-remove-compat-shims`
**Created**: 2026-02-20
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/115

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Remove Deprecated Contract Field (Priority: P1)

As a Wave developer, I want the deprecated `StrictMode` field removed from `ContractConfig` so that there is a single clear field (`MustPass`) controlling validation strictness, reducing confusion and eliminating dual-path logic.

**Why this priority**: `StrictMode` is explicitly marked `// Deprecated: use MustPass instead` and its continued presence creates dual-path logic throughout the contract validation system (`internal/contract/jsonschema.go`, `typescript.go`). This is the most clear-cut backwards-compat shim in the codebase and its removal simplifies the most code paths.

**Independent Test**: Can be fully tested by verifying that all contract validation tests pass using only `MustPass` and that no YAML/JSON configs in the project reference `strictMode`.

**Acceptance Scenarios**:

1. **Given** the `ContractConfig` struct, **When** a developer inspects it, **Then** there is no `StrictMode` field — only `MustPass` controls validation strictness.
2. **Given** a pipeline step with `must_pass: true` in its contract config, **When** validation runs, **Then** validation behaves identically to the previous `StrictMode: true` behavior.
3. **Given** existing tests reference `StrictMode`, **When** tests are updated, **Then** all tests pass using `MustPass` instead, with no behavior change.

---

### User Story 2 - Collapse Legacy JSON/YAML Extraction Fallbacks (Priority: P1)

As a Wave developer, I want the legacy fallback parsers removed so that there is a single, well-defined extraction path for JSON and YAML output, reducing complexity and eliminating dead fallback code.

**Why this priority**: Two distinct legacy fallback functions exist: `extractJSONFromTextLegacy` in `internal/contract/json_cleaner.go` and `extractYAMLLegacy` in `internal/pipeline/meta.go`. Both are retained solely for backward compatibility with old output formats. Collapsing to the new extraction paths eliminates dual-path logic.

**Independent Test**: Can be tested by verifying that the JSON recovery parser and the new `--- PIPELINE ---` / `--- SCHEMAS ---` section-based extraction handle all valid inputs, and that the legacy functions are no longer called.

**Acceptance Scenarios**:

1. **Given** AI output wrapped in markdown code blocks, **When** JSON extraction runs, **Then** the recovery parser handles it without falling back to `extractJSONFromTextLegacy`.
2. **Given** meta-pipeline output with `--- PIPELINE ---` sections, **When** YAML extraction runs, **Then** it extracts correctly without `extractYAMLLegacy`.
3. **Given** meta-pipeline output without section markers, **When** YAML extraction runs, **Then** it returns a clear error instead of silently falling back to a legacy parser.

---

### User Story 3 - Remove Migration Down Paths (Priority: P2)

As a Wave developer, I want unnecessary migration `Down` SQL removed from `migration_definitions.go` so that the migration system is simpler and does not carry dead rollback code for a pre-v1.0 prototype where schema rollback is not a supported use case.

**Why this priority**: All 6 migrations carry `Down` SQL. For a prototype that explicitly states backward compatibility is not a constraint, these `Down` paths add maintenance burden and create a false sense of rollback safety. Removing them simplifies the migration system.

**Independent Test**: Can be tested by verifying that migration `Up` paths still apply correctly, that the `wave migrate down` command gracefully reports that rollback is unsupported, and that all migration tests pass.

**Acceptance Scenarios**:

1. **Given** the migration definitions, **When** a developer inspects them, **Then** all `Down` fields are empty strings (`""`), and the `Down` field remains on the `Migration` struct for clarity.
2. **Given** a user runs `wave migrate down`, **When** the command executes, **Then** it returns a clear error message stating that rollback is not supported in the prototype phase.
3. **Given** the migration system, **When** `wave migrate up` runs, **Then** all migrations apply successfully as before.

---

### User Story 4 - Remove Legacy State Store Fallback (Priority: P2)

As a Wave developer, I want the legacy `schema.sql` initialization path removed from `state/store.go` so that the migration system is the only way to initialize the database, eliminating the dual-path conditional.

**Why this priority**: `store.go` contains a fallback to the old `schema.sql` file when migrations are disabled. Since migrations are enabled by default and the old schema path is a compat shim, removing it simplifies database initialization.

**Independent Test**: Can be tested by verifying database initialization works exclusively through the migration system and that `WAVE_MIGRATION_ENABLED=false` no longer silently falls back to the old schema.

**Acceptance Scenarios**:

1. **Given** Wave startup with default configuration, **When** the database initializes, **Then** it uses the migration system exclusively.
2. **Given** Wave startup with `WAVE_MIGRATION_ENABLED=false`, **When** the database initializes, **Then** it returns a clear error stating that the legacy schema initialization path has been removed and suggesting the user remove the `WAVE_MIGRATION_ENABLED=false` setting.
3. **Given** the `schema.sql` file and its `go:embed` directive in `store.go`, **When** the cleanup is complete, **Then** both the file and the embed directive (including the `"embed"` import) are deleted.

---

### User Story 5 - Clean Up Legacy Comments and Labels (Priority: P3)

As a Wave developer, I want all source code comments referencing "legacy", "backward compatible", or "deprecated" that describe removed shims to be cleaned up, so the codebase does not carry misleading documentation about non-existent compatibility paths.

**Why this priority**: After the functional shims are removed, stale comments reduce code clarity. This is lower priority because comments do not affect runtime behavior, but they do affect maintainability.

**Independent Test**: Can be tested by running a grep for "legacy", "backward compat", "deprecated" in Go source files and confirming that no remaining references describe removed functionality.

**Acceptance Scenarios**:

1. **Given** the completed cleanup, **When** a developer searches for `backward compat` in Go source, **Then** no results reference removed functionality (documentation files excluded).
2. **Given** YAML type fields using "legacy" terminology (e.g., `empty for legacy directory`), **When** cleanup is complete, **Then** comments describe current behavior only.
3. **Given** the `pipeline/executor.go:1454` comment `// Not tracked in legacy state store`, **When** cleanup is complete, **Then** the comment is removed or updated to reflect the current state store design.

---

### User Story 6 - Collapse Legacy Workspace Directory Lookup (Priority: P3)

As a Wave developer, I want the legacy exact-name directory lookup in `pipeline/resume.go` removed so that workspace resolution uses only the current naming convention (`<name>-<timestamp>-<hash>`).

**Why this priority**: The fallback at `resume.go:187-189` checks for directories without hash suffixes (a legacy naming convention). Since all current workspaces use the new convention, this is dead code.

**Independent Test**: Can be tested by verifying that pipeline resume still locates workspaces using the hash-suffixed naming convention.

**Acceptance Scenarios**:

1. **Given** a pipeline with existing workspace `pipeline-name-20260101-abc123`, **When** resume runs, **Then** it finds the workspace correctly.
2. **Given** a pipeline with no matching workspace, **When** resume runs, **Then** it reports that no workspace was found (does not silently fall back to legacy lookup).

---

### Edge Cases

- What happens if a pipeline YAML still references `strictMode` instead of `must_pass`? The system should fail with a clear validation error pointing the user to use `must_pass`.
- What happens if `wave migrate down` is called after Down paths are removed? The command should return a clear, actionable error message.
- What happens if the old `schema.sql` is referenced by external tooling? Out of scope per the issue — no external consumers during prototype phase.
- What happens if meta-pipeline output lacks both the new section markers AND valid YAML? The extraction should return a structured error, not silently produce empty output.
- What happens if `WAVE_MIGRATION_ENABLED` environment variable is set to `false` after the legacy path is removed? The system MUST return a clear error stating the legacy schema path has been removed (see C-003).
- What happens if test fixtures or test helpers use `StrictMode`? All test references must be migrated to `MustPass` before the field is removed.
- The "legacy template variables" in `pipeline/context.go` (`pipeline_id`, `pipeline_name`, `step_id`) are NOT compat shims — they are the primary format actively used by all pipeline YAML files. Only the misleading "legacy" comment should be updated; the variables themselves must be preserved.
- The `StrictMode` field in `internal/security/config.go` (sanitization strictness) is a completely separate concept from `contract.ContractConfig.StrictMode` and is NOT part of this cleanup (see C-001).

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST remove the `StrictMode` field from `ContractConfig` and consolidate all strict-mode logic to use `MustPass` exclusively. This includes updating `executor.go:675` (remove `StrictMode` assignment), `executor.go:700` (change to `MustPass`), `jsonschema.go:266-267` and `jsonschema.go:346` (replace `StrictMode` with `MustPass`), and `typescript.go:25` (replace `StrictMode` with `MustPass`). See C-001.
- **FR-002**: System MUST remove the `IsTypeScriptAvailable()` backward-compatibility wrapper from `internal/contract/typescript.go` and update all callers to use `CheckTypeScriptAvailability()` directly.
- **FR-003**: System MUST remove `extractJSONFromTextLegacy` from `internal/contract/json_cleaner.go` and ensure the recovery parser handles all valid JSON extraction scenarios.
- **FR-004**: System MUST remove `extractYAMLLegacy` from `internal/pipeline/meta.go` and require the section-marker format (`--- PIPELINE ---` / `--- SCHEMAS ---`) for meta-pipeline output.
- **FR-005**: System MUST set all `Down` SQL fields to empty strings in `internal/state/migration_definitions.go`, keeping the `Down` field on the `Migration` struct (see C-002).
- **FR-006**: System MUST remove the legacy `schema.sql` initialization fallback from `internal/state/store.go`, delete the `schema.sql` file, remove the `go:embed` directive and `schemaFS` variable, and remove the `"embed"` import (see C-004).
- **FR-007**: System MUST remove the legacy exact-name directory lookup from `internal/pipeline/resume.go`.
- **FR-008**: System MUST remove or update all source code comments that reference backwards-compatibility for removed functionality.
- **FR-009**: System MUST update the `WorkspaceConfig.Type` YAML tag comment from `empty for legacy directory` to reflect that empty means the default workspace type.
- **FR-010**: System MUST update the `worktree.go` comment from `legacy behavior` to describe it as the default branch-from-HEAD behavior.
- **FR-011**: System MUST update the `pipeline/context.go` comment from `Handle legacy template variables` to accurately describe these as the primary short-form template variables.
- **FR-012**: System MUST update the `pipeline/executor.go` comment `// Not tracked in legacy state store` to reflect the current state store design.
- **FR-013**: System MUST ensure `go test -race ./...` passes after all removals.
- **FR-014**: System MUST ensure `go vet ./...` reports no issues after all removals.
- **FR-015**: System MUST update the `wave migrate down` CLI command to return a clear error if Down paths are removed, rather than silently doing nothing.

### Key Entities

- **ContractConfig**: Configuration struct for output validation. Currently has both `StrictMode` (deprecated) and `MustPass` fields. After cleanup, only `MustPass` remains.
- **Migration**: Schema migration definition struct in `internal/state/`. Currently has `Up` and `Down` SQL fields. After cleanup, `Down` field remains on the struct but all values are empty strings (see C-002).
- **PipelineGenerationResult**: Result of meta-pipeline generation in `internal/pipeline/meta.go`. Currently has legacy YAML extraction fallback. After cleanup, only section-marker extraction is supported.
- **StateStore**: Database initialization in `internal/state/store.go`. Currently has dual-path initialization (migration vs legacy schema.sql). After cleanup, only the migration path exists.
- **WorkspaceConfig**: Pipeline workspace configuration in `internal/pipeline/types.go`. Currently has `Type` field with "legacy directory" terminology. After cleanup, terminology reflects current behavior.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Zero references to "backwards compat", "backward compat", or "deprecated" remain in Go source files that describe removed functionality (documentation files excluded).
- **SC-002**: `go test -race ./...` passes with zero failures after all removals.
- **SC-003**: `go vet ./...` reports zero issues.
- **SC-004**: The total lines of Go source code is reduced (net deletion of compat-shim code).
- **SC-005**: No dual-path conditional logic remains for old-vs-new format handling in the affected packages (`contract`, `pipeline`, `state`).
- **SC-006**: The PR diff shows net negative lines changed (more deletions than additions), confirming complexity reduction.
- **SC-007**: All existing pipeline YAML configurations in `.wave/pipelines/` and `internal/defaults/pipelines/` continue to function without modification (this is internal cleanup, not behavioral change for current consumers).

## Clarifications _(resolved)_

### C-001: StrictMode removal scope in executor.go

**Question**: The `StrictMode` field in `ContractConfig` is referenced in `executor.go:675` (where it's set to the same value as `MustPass`) and `executor.go:700` (where it's checked for soft-failure logic). Should the executor's soft-failure check at line 700 switch to using `MustPass`, or does the dual-path logic need restructuring?

**Resolution**: Replace all `StrictMode` references in `executor.go` with `MustPass`. At line 675, remove the `StrictMode: step.Handover.Contract.MustPass` assignment entirely. At line 700, change `contractCfg.StrictMode` to `contractCfg.MustPass`. The same applies to `jsonschema.go:266-267` (the fallback `if !cfg.MustPass && cfg.StrictMode` becomes unnecessary — just use `cfg.MustPass`) and `jsonschema.go:346` (`if !cfg.StrictMode` becomes `if !cfg.MustPass`). The `typescript.go:25` check `if cfg.StrictMode` should become `if cfg.MustPass`. Note: `security.StrictMode` in `internal/security/config.go` is a completely separate concept (sanitization strictness) and is NOT part of this cleanup.

**Rationale**: Since `executor.go:675` already sets `StrictMode` to the value of `MustPass`, the two fields are semantically identical in practice. Consolidating to `MustPass` is a direct substitution with no behavioral change.

### C-002: Migration Down field — empty strings vs struct field removal

**Question**: Acceptance scenario 3.1 offers two options: set `Down` fields to empty strings, or remove the `Down` field from the `Migration` struct entirely. Which approach should be taken?

**Resolution**: Keep the `Down` field on the `Migration` struct but set all `Down` values to empty strings (`""`) in `migration_definitions.go`. The `MigrateDown` function in `migrations.go:269-271` already checks for empty `Down` and returns `fmt.Errorf("migration %d has no rollback script")`, which naturally satisfies FR-015 without requiring changes to the CLI command itself.

**Rationale**: Keeping the struct field preserves the type definition's clarity about what migrations can theoretically contain, avoids cascading changes to `RollbackMigration()` and `MigrateDown()` method signatures, and the existing empty-check at `migrations.go:269` already provides the required error message. This is the minimal-change approach consistent with the prototype's "move fast" philosophy.

### C-003: WAVE_MIGRATION_ENABLED=false behavior after legacy path removal

**Question**: When `WAVE_MIGRATION_ENABLED=false` is set after the legacy `schema.sql` fallback is removed, should the system ignore the flag and always use migrations, or return a deprecation error?

**Resolution**: Return a clear error message stating that the legacy schema initialization path has been removed and that migrations are now the only supported method. The error should suggest removing the `WAVE_MIGRATION_ENABLED=false` setting.

**Rationale**: Silently ignoring a configuration flag violates the principle of least surprise. A clear error helps operators understand that their configuration is outdated and needs updating. Since this is prototype-phase software with no backward-compatibility guarantee, failing loudly is the safer default.

### C-004: schema.sql file and go:embed directive disposal

**Question**: The spec says to remove the `schema.sql` fallback but doesn't address the `go:embed` directive in `store.go:16-17` that embeds `schema.sql`. Should the `schema.sql` file and its embed directive both be deleted?

**Resolution**: Yes, delete both the `schema.sql` file and the `go:embed` directive (`var schemaFS embed.FS`) from `store.go`. Also remove the `"embed"` import. The file serves no purpose once the legacy initialization path is removed.

**Rationale**: Leaving an orphaned embedded file creates confusion. The `go:embed` directive would cause a compile error if the file were deleted without removing the directive, so both must be removed together. This is explicitly covered by acceptance scenario 4.3 ("the file is deleted if it serves no other purpose").

### C-005: wave migrate down CLI command behavior after Down removal

**Question**: FR-015 says to update the `wave migrate down` CLI command to return a clear error. But if we keep the `Down` field as empty strings (per C-002), the existing `MigrateDown()` function already returns an error ("migration N has no rollback script"). Does the CLI command itself need additional changes?

**Resolution**: The existing error from `MigrateDown()` is sufficient for the runtime behavior. However, the `wave migrate down` CLI command's help text (`--long` description) should be updated to state that rollback is not supported in the prototype phase, so users get the message before even attempting the operation. The confirmation prompt can be removed since rollback will always fail.

**Rationale**: Updating the help text provides a better user experience by communicating the limitation upfront rather than after a confirmation prompt. The underlying error from `MigrateDown()` serves as the safety net if the help text is ignored.
