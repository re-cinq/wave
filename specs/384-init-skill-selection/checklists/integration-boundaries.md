# Integration & Boundary Checklist: Wave Init Interactive Skill Selection

**Feature**: `384-init-skill-selection`
**Date**: 2026-03-14
**Focus**: Integration points, system boundaries, and cross-component requirements

---

## System Integration

- [ ] CHK043 - Is the interaction between `SkillSelectionStep` and `SourceRouter` fully specified — does the step create its own router or receive one via dependency injection? [Completeness]
- [ ] CHK044 - Is the `DirectoryStore` target path (`.wave/skills/`) consistent with how `wave skills install` creates its store, ensuring init-installed and CLI-installed skills coexist? [Consistency]
- [ ] CHK045 - Does the data model specify whether `SkillSelectionStep` should validate that installed skills are loadable (have valid `SKILL.md`) before reporting success? [Completeness]
- [ ] CHK046 - Is the relationship between `WizardResult.Skills` and `Manifest.Skills` type-compatible — both `[]string` with identical semantics? [Consistency]
- [ ] CHK047 - Does the plan address whether `buildManifest()` should emit `skills: []` (empty list) or omit the key entirely when no skills are installed? [Clarity]

## Boundary Conditions

- [ ] CHK048 - Is the maximum length of a skill name bounded in the spec or delegated to adapter validation? [Completeness]
- [ ] CHK049 - Are the valid characters for bare skill names defined (alphanumeric, hyphens, dots)? [Completeness]
- [ ] CHK050 - Is the behavior defined when the ecosystem CLI exists but returns a non-zero exit code for reasons other than "not found"? [Coverage]
- [ ] CHK051 - Does the spec address the case where `wave.yaml` already has a `skills:` key from manual editing — does `buildManifest()` merge or overwrite? [Coverage]

## Cross-Feature Impact

- [ ] CHK052 - Does adding Step 6 affect the existing `onboarding_test.go` assertions about step count or wizard flow completion? [Coverage]
- [ ] CHK053 - Is the impact on `wave init --yes` documented for downstream automation scripts that rely on non-interactive init completing without new prompts? [Completeness]
- [ ] CHK054 - Does the step renumbering (FR-012) account for any external documentation, help text, or error messages that reference "Step N of 5"? [Coverage]
