# Compatibility Requirements Quality: Skill Store Core

**Feature**: #381 — Skill Store Core
**Date**: 2026-03-13
**Focus**: Legacy system coexistence, naming collisions, and migration path

## Legacy Coexistence

- [ ] CHK-C01 - Does FR-010 define "coexist" precisely enough to verify? (Is it sufficient that existing tests pass, or must the two systems be explicitly decoupled at the import/type level?) [Compatibility-Clarity]
- [ ] CHK-C02 - Is the relationship between `SkillConfig.CommandsGlob` (legacy) and `Skill.AllowedTools` (new) documented? Could consumers confuse these two tool-permission mechanisms? [Compatibility-Completeness]
- [ ] CHK-C03 - Does the spec address potential type name collisions in the `internal/skill/` package? (e.g., both systems might eventually need a `List` function — is namespace partitioning defined?) [Compatibility-Coverage]
- [ ] CHK-C04 - Is SC-006 ("legacy provisioning tests pass unchanged") testable without running the full test suite? Does the spec identify which specific tests constitute the legacy boundary? [Compatibility-Clarity]

## Agent Skills Specification Conformance

- [ ] CHK-C05 - Does the spec reference a specific version or URL for the "Agent Skills Specification" format, or could the format definition drift without notice? [Compatibility-Completeness]
- [ ] CHK-C06 - Are there SKILL.md fields defined in the Agent Skills Specification that are intentionally excluded from this implementation? If so, is this documented? [Compatibility-Coverage]
- [ ] CHK-C07 - Does the spec define forward-compatibility behavior? (What happens when a SKILL.md contains unknown frontmatter fields not in the schema — are they silently ignored, preserved in round-trip, or rejected?) [Compatibility-Completeness]

## Migration Path

- [ ] CHK-C08 - Does the spec define any migration hooks or adapter pattern for transitioning from `Provisioner`/`SkillConfig` to `Store`/`Skill`? Or is migration explicitly out of scope? [Compatibility-Completeness]
- [ ] CHK-C09 - Is it clear whether the new `Store` will eventually replace the legacy `Provisioner`, or are they permanently parallel systems? [Compatibility-Clarity]
