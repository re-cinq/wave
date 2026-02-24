# feat(cli): add `wave list skills` subcommand

**Feature Branch**: `155-list-skills`
**Issue**: [#155](https://github.com/re-cinq/wave/issues/155)
**Labels**: enhancement
**Status**: Draft

## Context

The `wave list` command supports `adapters`, `runs`, `pipelines`, `personas`, and `contracts` — but not `skills`.

With the migration from dead `skill_mounts` to the active `skills` config map, users can now declare skills in `wave.yaml` with `install`, `check`, and `init` commands. However, there's no CLI way to inspect declared skills and their status.

## Proposed Behavior

```
wave list skills
```

Should display:
- Skill name
- Check command and whether it passes (installed vs missing)
- Install command (if configured)
- Which pipelines require it (via `requires.skills`)

JSON output via `--format json` should also be supported, consistent with other `wave list` subcommands.

## Implementation Notes

- Add `skills` to `ValidArgs` in `list.go`
- Read `skills` from manifest (add to `manifestData2` struct or use `manifest.Load`)
- Run each skill's `check` command to determine availability
- Cross-reference with pipeline `requires.skills` blocks for usage info
- Follow existing patterns from `listAdaptersTable` / `collectAdapters`

## User Scenarios & Testing

### User Story 1 - List skills in table format (Priority: P1)

A developer wants to see which skills are declared in their `wave.yaml` and whether they are installed.

**Why this priority**: Core functionality — the primary use case for the command.

**Independent Test**: Run `wave list skills` with skills declared in `wave.yaml` and verify the table output contains skill names, check status, and install commands.

**Acceptance Scenarios**:

1. **Given** a `wave.yaml` with skills declared, **When** the user runs `wave list skills`, **Then** the output shows each skill name, check command, installed/missing status, and install command.
2. **Given** a `wave.yaml` with no skills declared, **When** the user runs `wave list skills`, **Then** the output shows a "(none defined)" message.
3. **Given** no `wave.yaml` file, **When** the user runs `wave list skills`, **Then** the output shows a "manifest not found" message.

---

### User Story 2 - List skills in JSON format (Priority: P1)

A developer or CI system wants machine-readable output of skill status.

**Why this priority**: JSON output is required for consistency with other `wave list` subcommands.

**Independent Test**: Run `wave list skills --format json` and verify the output is valid JSON containing skill data.

**Acceptance Scenarios**:

1. **Given** skills are declared, **When** the user runs `wave list skills --format json`, **Then** valid JSON is output with skill names, check status, install commands, and pipeline usage.
2. **Given** no filter is used, **When** the user runs `wave list --format json`, **Then** the JSON output includes a `skills` field alongside other categories.

---

### User Story 3 - Show pipeline usage (Priority: P2)

A developer wants to know which pipelines require each skill.

**Why this priority**: Cross-referencing adds context but is supplementary to the core listing.

**Independent Test**: Run `wave list skills` with a pipeline that declares `requires.skills` and verify the output shows which pipelines use each skill.

**Acceptance Scenarios**:

1. **Given** a pipeline with `requires.skills: [speckit]`, **When** the user runs `wave list skills`, **Then** the skill `speckit` output includes the pipeline name in its usage list.

---

### Edge Cases

- Skill with no `check` command should never occur (validation requires it), but if encountered, show "unchecked" status.
- Skill whose `check` command fails should show as "missing" / not installed.
- Skill whose `check` command hangs — existing shell execution timeout should apply.

## Requirements

### Functional Requirements

- **FR-001**: System MUST add `"skills"` to `ValidArgs` in the list command.
- **FR-002**: System MUST read the `skills` map from `wave.yaml` (via `manifestData2` or `manifest.Load`).
- **FR-003**: System MUST run each skill's `check` command to determine availability status.
- **FR-004**: System MUST display skill name, check command, status (installed/missing), and install command in table format.
- **FR-005**: System MUST cross-reference pipeline `requires.skills` blocks to show which pipelines use each skill.
- **FR-006**: System MUST support `--format json` output consistent with other list subcommands.
- **FR-007**: System MUST include skills in the "list all" (no filter) output.

### Key Entities

- **SkillInfo**: Represents a skill's name, check command, install command, installed status, and pipeline usage — the JSON/table output structure.
- **SkillConfig** (existing): The manifest type at `internal/manifest/types.go:161` that declares install, init, check, and commands_glob.

## Success Criteria

- **SC-001**: `wave list skills` displays all declared skills with their status.
- **SC-002**: `wave list skills --format json` produces valid JSON matching the output schema.
- **SC-003**: Pipeline cross-referencing correctly identifies which pipelines require each skill.
- **SC-004**: All existing `list` tests continue to pass.
- **SC-005**: New tests cover table format, JSON format, empty skills, and pipeline cross-referencing.
