# Quality Review Checklist: Wave Skills CLI

**Feature**: `wave skills` CLI — list/search/install/remove/sync
**Spec**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md) | **Tasks**: [tasks.md](../tasks.md)
**Generated**: 2026-03-14

---

## Completeness

- [ ] CHK001 - Are all 5 subcommands (list, search, install, remove, sync) fully specified with acceptance scenarios? [Completeness]
- [ ] CHK002 - Does each functional requirement (FR-001 through FR-015) have at least one acceptance scenario that exercises it? [Completeness]
- [ ] CHK003 - Are all 3 new error codes (skill_not_found, skill_source_error, skill_dependency_missing) mapped to specific triggering conditions? [Completeness]
- [ ] CHK004 - Are output schemas defined for all 5 subcommands in both table and JSON format? [Completeness]
- [ ] CHK005 - Does the spec define behavior for all 7 recognized source prefixes (tessl:, bmad:, openspec:, speckit:, github:, file:, https://)? [Completeness]
- [ ] CHK006 - Are all 7 edge cases in the spec traceable to at least one acceptance scenario or functional requirement? [Completeness]
- [ ] CHK007 - Does the plan specify the file location and registration path for every new code artifact? [Completeness]
- [ ] CHK008 - Are success criteria (SC-001 through SC-008) each verifiable by at least one planned test case? [Completeness]

## Clarity

- [ ] CHK009 - Is the distinction between `wave skills list` (DirectoryStore SKILL.md) and `wave list skills` (manifest SkillConfig) clearly documented and unambiguous? [Clarity]
- [ ] CHK010 - Are the table column names and ordering explicitly specified for each subcommand's table output? [Clarity]
- [ ] CHK011 - Is the confirmation prompt text for `wave skills remove` specified verbatim (exact string)? [Clarity]
- [ ] CHK012 - Are the error message templates for unrecognized prefixes and missing dependencies specified with sufficient detail to implement? [Clarity]
- [ ] CHK013 - Is the precedence model (project .wave/skills/ > user .claude/skills/) clearly stated with numeric precedence values? [Clarity]
- [ ] CHK014 - Are the `--format` flag valid values (table, json) and default (table) explicitly stated for each subcommand? [Clarity]
- [ ] CHK015 - Is the empty-state behavior for each subcommand (no skills, no results, no dependencies) defined with specific output text? [Clarity]

## Consistency

- [ ] CHK016 - Does the plan's `newSkillStore()` precedence (project=2, user=1) match the spec's stated precedence model (project takes precedence over user)? [Consistency]
- [ ] CHK017 - Do all data model struct fields in data-model.md match the JSON field names referenced in acceptance scenarios? [Consistency]
- [ ] CHK018 - Does the plan's testing strategy cover all success criteria from the spec? [Consistency]
- [ ] CHK019 - Are the function signatures in the plan (runSkillsList, runSkillsInstall, etc.) consistent with each subcommand's required parameters? [Consistency]
- [ ] CHK020 - Does the task list (T001-T037) cover every implementation item described in the plan without gaps or duplication? [Consistency]
- [ ] CHK021 - Are the error code constant names in plan.md/data-model.md consistent with the naming pattern in the existing errors.go? [Consistency]
- [ ] CHK022 - Does the plan's `classifySkillError()` mapping cover all error scenarios described in the spec's acceptance scenarios? [Consistency]
- [ ] CHK023 - Is the `promptConfirm` pattern referenced in the plan consistent with its actual signature in doctor.go? [Consistency]

## Coverage

- [ ] CHK024 - Are concurrent access scenarios addressed (e.g., concurrent install of the same skill)? [Coverage]
- [ ] CHK025 - Is behavior specified for when the skill store directories (.wave/skills/, .claude/skills/) do not exist at read time? [Coverage]
- [ ] CHK026 - Are permission/filesystem errors (read-only directory, missing permissions) addressed in the spec or edge cases? [Coverage]
- [ ] CHK027 - Does the spec address what happens when `--format json` encounters partial failures (e.g., list with some malformed SKILL.md files)? [Coverage]
- [ ] CHK028 - Are non-interactive/scripted usage patterns addressed beyond just `--yes` on remove (e.g., piped input, CI environments)? [Coverage]
- [ ] CHK029 - Is the behavior for duplicate skill names across precedence levels fully defined for all subcommands, not just list? [Coverage]
- [ ] CHK030 - Does the spec define behavior when install succeeds but produces warnings (e.g., skill overwrites existing)? [Coverage]
- [ ] CHK031 - Are network failure scenarios addressed for search and sync (tessl CLI available but network unreachable)? [Coverage]
- [ ] CHK032 - Is the `--help` output content specified for both the parent command and each subcommand? [Coverage]
- [ ] CHK033 - Does the testing strategy include tests for the global flag precedence (--json > --quiet > --output > local --format)? [Coverage]
