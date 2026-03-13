# Requirements Quality Review: Skill Store Core

**Feature**: #381 — Skill Store Core
**Date**: 2026-03-13
**Artifacts Reviewed**: spec.md, plan.md, tasks.md, data-model.md, research.md, contracts/

## Completeness

- [ ] CHK001 - Are all CRUD operations (Read, Write, List, Delete) covered by at least one user story with acceptance scenarios? [Completeness]
- [ ] CHK002 - Does the spec define what happens when Write is called with a skill whose `name` differs from the existing directory's `name`? (e.g., rename scenario or update-with-changed-name) [Completeness]
- [ ] CHK003 - Are filesystem permission errors (e.g., read-only directory on Write, unreadable SKILL.md on Read) explicitly addressed in the error taxonomy? [Completeness]
- [ ] CHK004 - Is the behavior for concurrent access to the store defined? (The contract says "NOT thread-safe" — is this documented in the spec or only in the contract artifact?) [Completeness]
- [ ] CHK005 - Does FR-007 specify how source directory precedence is configured or only the default ordering? (Is precedence hardcoded or configurable?) [Completeness]
- [ ] CHK006 - Is the maximum depth of skill directory scanning defined? (Only immediate subdirectories, or recursive?) [Completeness]
- [ ] CHK007 - Are the optional resource subdirectory names (`scripts/`, `references/`, `assets/`) exhaustively defined, or can arbitrary subdirectories be treated as resources? [Completeness]
- [ ] CHK008 - Does the spec define the behavior when a SKILL.md file has duplicate YAML keys (e.g., two `name` fields)? [Completeness]

## Clarity

- [ ] CHK009 - Is the `allowed-tools` format unambiguous? Does the spec explicitly state whether quoted and unquoted YAML strings are both accepted? [Clarity]
- [ ] CHK010 - Is the frontmatter delimiter format precisely defined? (Must it be exactly `---\n` or does `--- \n` with trailing whitespace also qualify?) [Clarity]
- [ ] CHK011 - Is "semantically equivalent" in SC-002 (round-trip fidelity) defined precisely enough to test? (e.g., is YAML key ordering significant? trailing newlines?) [Clarity]
- [ ] CHK012 - Is the relationship between `ParseError`, `DiscoveryError`, and `SkillError` clear enough that an implementer can determine which error type each operation returns without consulting the contract artifacts? [Clarity]
- [ ] CHK013 - Does FR-004 (name must match parent directory) clearly specify whether this applies during Write operations (enforcing directory name from `skill.Name`) or only during Read/List? [Clarity]
- [ ] CHK014 - Is "non-empty" for description precisely defined? (Does a string of only whitespace count as empty?) [Clarity]

## Consistency

- [ ] CHK015 - Are the error type names consistent across spec.md, plan.md, data-model.md, and contracts? (spec says "typed errors for: not-found, parse-failure, validation-failure" — do all artifacts use `ParseError` consistently?) [Consistency]
- [ ] CHK016 - Does the data-model.md show `errors.go` as a separate file while plan.md puts error types in `store.go`? Is the file structure consistent across artifacts? [Consistency]
- [ ] CHK017 - Is the number of existing SKILL.md files consistent? (SC-001 says "13 existing" — does this match the actual count, and is it referenced consistently?) [Consistency]
- [ ] CHK018 - Does the task ordering in tasks.md respect the dependency graph defined in plan.md? (Are `[P]` parallel markers consistent with prerequisite constraints?) [Consistency]
- [ ] CHK019 - Does the `DiscoveryError.Unwrap()` returning `nil` (data-model.md line 68) conflict with standard Go error unwrapping patterns? Is this consistent with how callers are expected to inspect errors? [Consistency]

## Coverage

- [ ] CHK020 - Are symlink-related behaviors covered in acceptance scenarios, not just edge cases? (Edge case section mentions symlinks are followed — is this testable?) [Coverage]
- [ ] CHK021 - Is the behavior for very large SKILL.md files defined? (e.g., a 10MB markdown body — is there a size limit?) [Coverage]
- [ ] CHK022 - Are Unicode skill names addressed? The regex `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` rejects them, but is this explicitly stated as intentional? [Coverage]
- [ ] CHK023 - Is the behavior defined when the highest-precedence source directory exists but is not writable, and Write is called? [Coverage]
- [ ] CHK024 - Are empty `metadata` maps (`metadata: {}`) and null metadata values (`metadata: null`) distinguished in the spec? [Coverage]
- [ ] CHK025 - Does the spec cover the case where `allowed-tools` contains duplicate tool names? (e.g., `"Read Read Write"`) [Coverage]
- [ ] CHK026 - Is the Delete operation's behavior when multiple sources contain the same skill name fully specified? (Delete from first match only vs. all sources) [Coverage]
