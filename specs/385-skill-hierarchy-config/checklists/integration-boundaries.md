# Integration Boundaries Checklist: Hierarchical Skill Configuration

**Feature**: #385 — Skill Hierarchy Config
**Date**: 2026-03-14
**Focus**: Quality of requirements at integration points between skill hierarchy and existing subsystems

---

## Executor Integration

- [ ] CHK026 - Are the requirements for injecting a `skill.Store` dependency into the executor specified (constructor parameter, field on `Executor` struct, or via `ExecutionContext`)? [Completeness]
- [ ] CHK027 - Is it specified whether the executor should fail the step or skip provisioning when `DirectoryStore.Read()` fails for a previously-validated skill? [Completeness]
- [ ] CHK028 - Does the spec define the ordering of skill provisioning relative to existing steps in `buildAdapterRunConfig` (before or after `SkillConfig`-based provisioning)? [Clarity]
- [ ] CHK029 - Are the requirements for SKILL.md file placement in the workspace specified — does it go to `.wave/skills/<name>/SKILL.md` or the workspace root? [Clarity]

## Manifest/Pipeline Loading

- [ ] CHK030 - Is the impact on `manifest.Load()` callers documented — specifically, how many call sites exist and whether they all have access to a `skill.Store`? [Coverage]
- [ ] CHK031 - Does the spec address whether `ValidateWithFile` signature changes or whether a separate validation step is added (plan says separate function but no FR codifies this)? [Consistency]
- [ ] CHK032 - Is it clear whether pipeline skill validation should block pipeline loading or only produce warnings? [Clarity]

## Preflight System

- [ ] CHK033 - Does the spec define whether name-only skill references (no `SkillConfig`) should trigger any preflight behavior, or is preflight exclusively for `requires.skills` entries? [Completeness]
- [ ] CHK034 - Are requirements specified for what happens when a skill has both a `SkillConfig` entry AND a DirectoryStore entry with conflicting content? [Coverage]

## Backward Compatibility

- [ ] CHK035 - Is there a requirement that existing `wave.yaml` files without any `skills:` fields must parse identically to before (not just "no error" but identical struct output)? [Completeness]
- [ ] CHK036 - Does the spec address whether existing test fixtures and example manifests need to be updated, or whether they implicitly test backward compatibility? [Coverage]
