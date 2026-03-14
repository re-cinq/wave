# Requirements Quality Review: Hierarchical Skill Configuration

**Feature**: #385 — Skill Hierarchy Config
**Date**: 2026-03-14
**Artifacts Reviewed**: spec.md, plan.md, tasks.md, research.md, data-model.md

---

## Completeness

- [ ] CHK001 - Does the spec define the exact YAML schema for `skills:` at all three scopes (key name, type, required/optional, default value)? [Completeness]
- [ ] CHK002 - Is the error message format for validation failures specified (e.g., template string, required fields in the message)? [Completeness]
- [ ] CHK003 - Does the spec define what happens when `ResolveSkills` is called with personas that have no `Skills` field vs personas not found in the manifest? [Completeness]
- [ ] CHK004 - Is the provisioning behavior for resource files from DirectoryStore fully specified (target paths, overwrite policy, file permissions)? [Completeness]
- [ ] CHK005 - Does the spec define the interaction between skill provisioning and workspace lifecycle (when provisioning runs relative to workspace creation and artifact injection)? [Completeness]
- [ ] CHK006 - Is there a requirement specifying the behavior when the DirectoryStore contains valid `SKILL.md` but the skill's `ResourcePaths` reference files that don't exist? [Completeness]
- [ ] CHK007 - Does the spec address whether skill resolution results are logged or emitted as observable events (per P10 constitution principle)? [Completeness]

## Clarity

- [ ] CHK008 - Is the meaning of "precedence" in the merge clearly defined — does it mean override, first-wins, or something else for string-only references where all values are equal? [Clarity]
- [ ] CHK009 - Is it clear whether `ValidateSkillRefs` should accept a `Store` interface or a concrete `DirectoryStore` — and is the interface contract specified? [Clarity]
- [ ] CHK010 - Is it unambiguous how `requires.skills` keys are extracted — specifically, does `SkillNames()` exist on the `Requires` struct or does it need to be created? [Clarity]
- [ ] CHK011 - Is the scope label format for error messages formally specified (e.g., always `"global"`, `"persona:<name>"`, `"pipeline:<name>"`)? [Clarity]
- [ ] CHK012 - Does the spec clarify whether pipeline skill validation happens once at load time or on every execution (important for dynamically modified pipeline files)? [Clarity]

## Consistency

- [ ] CHK013 - Does the plan's `ValidateManifestSkills` function signature in T008 match the data-model's `ValidateSkillRefs` signature, particularly the `personas` parameter type? [Consistency]
- [ ] CHK014 - Is there consistency between the spec (FR-007 says "at load time") and the plan (Phase C says both manifest and pipeline load paths) about WHEN validation occurs? [Consistency]
- [ ] CHK015 - Does the plan's approach to passing a `skill.Store` to the manifest `Load()` function align with the existing `Load()` signature and callers (no orphaned call sites)? [Consistency]
- [ ] CHK016 - Is the tasks dependency graph consistent with the plan's phase ordering (e.g., T013 depends on T001-T003 via struct fields but this dependency isn't shown)? [Consistency]
- [ ] CHK017 - Do the acceptance scenarios in US3 (pipeline scope) use the same deterministic output ordering as the resolution function specifies (alphabetical sort)? [Consistency]
- [ ] CHK018 - Is the existing `Provisioner.Provision()` interaction consistent between FR-013 (which describes DirectoryStore provisioning) and the plan's T015 (which separates command file vs content provisioning)? [Consistency]

## Coverage

- [ ] CHK019 - Are performance implications addressed — what happens when dozens of skills are declared across scopes and each requires a `store.Read()` call during validation? [Coverage]
- [ ] CHK020 - Is concurrent access to the DirectoryStore considered — can two pipeline steps validating simultaneously cause race conditions? [Coverage]
- [ ] CHK021 - Does the spec cover the case where a skill is removed from `.wave/skills/` between manifest validation and step execution (time-of-check vs time-of-use)? [Coverage]
- [ ] CHK022 - Are error reporting requirements covered for multi-scope aggregation — specifically, is the order of errors in the aggregated list deterministic? [Coverage]
- [ ] CHK023 - Is the behavior specified when a pipeline's `skills:` list contains duplicate entries within the same scope (e.g., `skills: ["foo", "foo"]`)? [Coverage]
- [ ] CHK024 - Does the spec address how skill resolution interacts with pipeline resume — are resolved skills recalculated on resume or cached from the original run? [Coverage]
- [ ] CHK025 - Is there coverage for the case where `requires.skills` contains a skill name that fails DirectoryStore validation — should `SkillConfig` entries also be validated against the store? [Coverage]
