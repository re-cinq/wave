# Requirements Quality Review Checklist

**Feature**: Skill Dependency Installation in Pipeline Steps
**Branch**: `102-skill-deps-pipeline`
**Date**: 2026-02-14

This checklist validates the quality of the requirements in `spec.md`, `plan.md`, and `tasks.md` — not the implementation itself. Each item is a "unit test for requirements."

---

## Completeness

- [ ] CHK001 - Are all five user stories traceable to at least one functional requirement (FR-001 through FR-012)? [Completeness]
- [ ] CHK002 - Does the spec define what happens when `requires` is omitted entirely from a pipeline YAML? (Is it optional? Does the preflight phase still run?) [Completeness]
- [ ] CHK003 - Does the spec define what happens when `requires.skills` is an empty list `[]` vs. omitted? [Completeness]
- [ ] CHK004 - Are the concrete install/check/init commands for all three target skills (Speckit, BMAD, OpenSpec) documented with enough specificity to implement without guesswork? [Completeness]
- [ ] CHK005 - Does the spec define the ordering of preflight checks — are tools checked before skills, or is the order unspecified? [Completeness]
- [ ] CHK006 - Does the spec define whether preflight fails fast on the first dependency failure or collects all failures before reporting? [Completeness]
- [ ] CHK007 - Is the provisioning chain's staging directory path (`.wave-skill-commands/.claude/commands/`) explicitly documented as a requirement or left as an implementation detail? [Completeness]
- [ ] CHK008 - Does FR-008 clearly distinguish the three `install` optionality scenarios: (a) install defined and succeeds, (b) install defined and fails, (c) install not defined? [Completeness]
- [ ] CHK009 - Are all edge cases (6 documented) linked to specific functional requirements or acceptance scenarios that exercise them? [Completeness]
- [ ] CHK010 - Does the spec address what happens when `commands_glob` matches zero files for an installed skill? (Edge case 1 covers it for check-passes case, but is the non-warning path clear?) [Completeness]

## Clarity

- [ ] CHK011 - Is the distinction between "skill" (lifecycle-managed external tool) and "tool" (PATH-checked binary) unambiguous throughout the spec? [Clarity]
- [ ] CHK012 - Does the spec clearly define what "installed" means for each of the three target skills — is it the check command passing, or something else? [Clarity]
- [ ] CHK013 - Is the `emitter func(name, kind, message string)` callback contract clear enough for a developer to implement the executor wiring without consulting the plan? [Clarity]
- [ ] CHK014 - Does the spec clarify what "descriptive error" (FR-009) means — is there a defined error format, or is free-form text acceptable? [Clarity]
- [ ] CHK015 - Is the relationship between `Checker.Run()`, `CheckTools()`, and `CheckSkills()` unambiguous — does `Run()` call both, or is it a separate method? [Clarity]
- [ ] CHK016 - Does the spec clearly define who calls the `init` command — the Checker during preflight, or the Provisioner during workspace setup? [Clarity]
- [ ] CHK017 - Are the terms "preflight phase," "preflight checks," and "preflight validation" used consistently, or do they refer to different things? [Clarity]

## Consistency

- [ ] CHK018 - Is the `commands_glob` resolution root (repo root per C5) consistent with how the Provisioner's `repoRoot` parameter is documented in data-model.md? [Consistency]
- [ ] CHK019 - Does the task list (T001-T021) cover all 7 changes enumerated in plan.md (Changes 1-7)? [Consistency]
- [ ] CHK020 - Are the event message strings in tasks.md T003/T004 consistent with the event states contract in `contracts/event-states.go`? [Consistency]
- [ ] CHK021 - Does the plan's claim of "no new packages or files needed" align with the tasks — are any tasks creating new files not mentioned in the plan? [Consistency]
- [ ] CHK022 - Is the `StatePreflight` constant name used consistently across spec.md (FR-010, C4), plan.md (Change 1), data-model.md, and contracts/event-states.go? [Consistency]
- [ ] CHK023 - Does the task dependency graph in tasks.md match the dependency ordering described in plan.md (e.g., T001 before T002, T006 before T008)? [Consistency]
- [ ] CHK024 - Are line number references in tasks.md (e.g., "executor.go:512", "emitter.go line 55-69") consistent with the line numbers cited in research.md and plan.md? [Consistency]

## Coverage

- [ ] CHK025 - Does the test plan (T011-T018) cover all 6 edge cases listed in the spec? [Coverage]
- [ ] CHK026 - Are there acceptance scenarios for the negative path where a skill's `install` command is missing (FR-008 optionality)? [Coverage]
- [ ] CHK027 - Does the task list include verification that existing preflight tests still pass after the `NewChecker` signature change (backward compatibility of variadic opts)? [Coverage]
- [ ] CHK028 - Is there a task or test covering the interaction between tool checks and skill checks when both fail simultaneously? [Coverage]
- [ ] CHK029 - Does the plan address the performance constraint SC-006 (<500ms overhead) with a concrete measurement or test strategy? [Coverage]
- [ ] CHK030 - Are there requirements or tests for the adapter's `copySkillCommands()` behavior when the staging directory is empty or doesn't exist? [Coverage]
- [ ] CHK031 - Does the spec define success/failure behavior when the same skill is declared in `requires.skills` more than once (duplicate entries)? [Coverage]
