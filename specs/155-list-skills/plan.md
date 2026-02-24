# Implementation Plan: `wave list skills`

## Objective

Add a `skills` subcommand to `wave list` that displays declared skills from `wave.yaml`, their installation status (via running each skill's `check` command), and which pipelines require them. Support both table and JSON output formats, consistent with the existing `adapters`, `personas`, `pipelines`, `contracts`, and `runs` subcommands.

## Approach

Follow the established patterns in `cmd/wave/commands/list.go`:

1. **Data collection** via a `collectSkills()` function (mirrors `collectAdapters`, `collectPersonas`, `collectContracts`)
2. **Table rendering** via a `listSkillsTable()` function (mirrors `listAdaptersTable`, `listContractsTable`)
3. **JSON output** by adding a `Skills` field to `ListOutput` and populating it in the JSON branch of `runList()`
4. **Pipeline cross-referencing** by scanning pipeline YAML files for `requires.skills` arrays (similar to how `collectContracts` scans for `contract.schema_path`)

The manifest is currently parsed via a local `manifestData2` struct in `list.go`. We'll extend this struct to include a `Skills` field mapping to `SkillConfig`-equivalent structs, keeping the same pattern.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/list.go` | modify | Add `SkillInfo` struct, `collectSkills()`, `listSkillsTable()`, extend `manifestData2`, `ListOutput`, `ValidArgs`, `runList()`, and help text |
| `cmd/wave/commands/list_test.go` | modify | Add tests for skills table format, JSON format, empty skills, pipeline cross-referencing, and sorted output |

## Architecture Decisions

### AD-1: Extend `manifestData2` vs. use `manifest.Load`

**Decision**: Extend `manifestData2` with a `Skills` field.

**Rationale**: The existing code uses `manifestData2` for all manifest-sourced data in the list command. Switching to `manifest.Load` would introduce validation (e.g., checking that persona prompt files exist) that can fail in the list context where we just want to display what's configured. Staying consistent with the existing pattern avoids unnecessary complexity.

### AD-2: Check command execution

**Decision**: Use `exec.Command("sh", "-c", skill.Check)` to run check commands, matching the pattern in `internal/preflight/preflight.go`.

**Rationale**: Check commands are shell command strings (e.g., `specify --version`), so they need shell interpretation. The `sh -c` pattern is already established in the codebase via `preflight.Checker.runShellCommand`.

### AD-3: Pipeline cross-referencing approach

**Decision**: Scan pipeline YAML files and parse `requires.skills` arrays.

**Rationale**: This is consistent with how `collectContracts` scans pipelines for `contract.schema_path`. The pipeline files already use the `Requires` struct with `Skills []string` (see `internal/pipeline/types.go:22-25`).

### AD-4: SkillInfo struct design

**Decision**: Include `name`, `check`, `install`, `installed` (bool), and `used_by` (pipeline names array) fields.

**Rationale**: Covers all information specified in the issue. The `used_by` field is a simple string array (just pipeline names) rather than a complex struct, since skills are referenced at the pipeline level, not the step level.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Check command hangs/slow | Low | Medium | Check commands are expected to be fast (e.g., `--version`). No timeout needed since this is user-initiated CLI. |
| Pipeline YAML parsing diverges from `pipeline.Requires` struct | Low | Low | Parse only the `requires.skills` field from pipeline YAML, using a minimal struct. |
| Test flakiness from `exec.LookPath` or shell commands | Medium | Low | Tests use controlled manifests with known check commands (e.g., `true`/`false`). |

## Testing Strategy

### Unit Tests (in `list_test.go`)

1. **TestListCmd_Skills_TableFormat** — Skills listed with correct headers
2. **TestListCmd_Skills_ShowsCheckCommand** — Check command displayed per skill
3. **TestListCmd_Skills_ShowsInstallCommand** — Install command displayed per skill
4. **TestListCmd_Skills_ShowsStatus** — Installed vs missing status based on check command result
5. **TestListCmd_Skills_ShowsPipelineUsage** — Pipeline names shown when they require the skill
6. **TestListCmd_Skills_NoSkillsDefined** — "(none defined)" when no skills in manifest
7. **TestListCmd_Skills_JSONFormat** — Valid JSON output with `skills` field
8. **TestListCmd_Skills_SortedAlphabetically** — Skills listed in alphabetical order
9. **TestListCmd_Skills_InListAll** — Skills appear in `wave list` (no filter) output
10. **TestListCmd_FilterOptions** — Extend existing table-driven test to include `skills` filter
