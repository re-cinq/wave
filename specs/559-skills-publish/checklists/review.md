# Requirements Quality Review Checklist

**Feature**: #559 Skills Publish
**Generated**: 2026-03-24
**Artifacts**: spec.md, plan.md, tasks.md, research.md, data-model.md

---

## Completeness

- [ ] CHK001 - Are all 5 user stories (audit, publish, batch, integrity, validation) accompanied by acceptance scenarios that cover both happy path and failure path? [Completeness]
- [ ] CHK002 - Does the spec define the full CLI surface area: all subcommands, flags, positional args, and their interactions (e.g., --all vs positional name mutual exclusion)? [Completeness]
- [ ] CHK003 - Are output format requirements specified for both --format json and --format table for all three subcommands (audit, publish, verify)? [Completeness]
- [ ] CHK004 - Is the authentication model for registry publishing fully described (how tessl obtains credentials, what happens when auth is missing/expired)? [Completeness]
- [ ] CHK005 - Are all lockfile lifecycle scenarios specified: creation on first publish, update on subsequent publishes, behavior when lockfile is read-only or in a read-only filesystem? [Completeness]
- [ ] CHK006 - Is the behavior of --registry flag defined when the specified registry is unknown or misconfigured? [Completeness]
- [ ] CHK007 - Does the spec define what "version" means for a published skill (is it the digest, a semver, a timestamp, or something else)? FR-005 mentions "version" in the lockfile but PublishRecord has no Version field. [Completeness]
- [ ] CHK008 - Are requirements for the `skills/` distributable directory fully specified: when it gets updated, whether it's auto-generated or manually curated, and how it stays in sync with `.claude/skills/`? [Completeness]
- [ ] CHK009 - Is the tessl publish command's expected input format documented (does it expect a directory, a file, specific file structure)? [Completeness]
- [ ] CHK010 - Are concurrency requirements specified for batch publish (sequential vs parallel, rate limiting)? [Completeness]

## Clarity

- [ ] CHK011 - Is the distinction between "validation" (FR-003, agentskills.io spec) and "classification" (FR-001, wave-specific detection) clearly delineated in the spec so implementers don't conflate them? [Clarity]
- [ ] CHK012 - Are the classification thresholds (0 = standalone, 1-10 = both, >10 = wave-specific) justified and documented as configurable or hardcoded? [Clarity]
- [ ] CHK013 - Is it clear whether `wave skills publish <name>` publishes from the skill store (`.wave/skills/` or `~/.claude/skills/`) or from the `skills/` distributable directory? [Clarity]
- [ ] CHK014 - Is the relationship between FR-016 (distributable directory) and FR-002 (publish command) unambiguous — does publish read from `skills/` or does it populate `skills/`? [Clarity]
- [ ] CHK015 - Is the "force" flag semantics clear: does --force only override wave-specific warnings, or does it also override validation errors, lockfile corruption, and other safeguards? [Clarity]
- [ ] CHK016 - Is the term "resource files" consistently defined across all artifacts (scripts/, references/, assets/ subdirectories)? [Clarity]
- [ ] CHK017 - Is it clear what "installed skill" means in US1 context — does audit cover only project-local skills, user-global skills, or both? [Clarity]

## Consistency

- [ ] CHK018 - Does the plan's file layout (classify.go, digest.go, lockfile.go, publish.go in internal/skill/) align with the task assignments in tasks.md (T002-T005)? [Consistency]
- [ ] CHK019 - Are the error codes in the plan (B1: skill_publish_failed, skill_validation_failed, skill_already_exists) consistent with those in tasks.md T001? [Consistency]
- [ ] CHK020 - Does the data model's PublishRecord struct match the lockfile schema in research.md R4? [Consistency]
- [ ] CHK021 - Are the 13 skill names and their classifications consistent between research.md R1 evidence table and tasks.md T020 eligible skills list? [Consistency]
- [ ] CHK022 - Does tasks.md T020 list 12 eligible skills (13 total minus wave) consistently with the research.md classification table (9 standalone + 3 both = 12)? [Consistency]
- [ ] CHK023 - Is the lockfile path consistently `.wave/skills.lock` across spec (FR-005), plan (A3), data-model, and tasks (T004, T009)? [Consistency]
- [ ] CHK024 - Does the spec's FR-005 mention "version" as a lockfile field, but the data model's PublishRecord struct omits a version field — is this inconsistency addressed? [Consistency]
- [ ] CHK025 - Are the ValidationReport fields (Errors, Warnings as []ValidationIssue) consistent between data-model.md and the task description T005? [Consistency]

## Coverage

- [ ] CHK026 - Do edge cases cover all external failure modes: network timeout, DNS resolution failure, registry rate limiting, TLS certificate errors? [Coverage]
- [ ] CHK027 - Is there a requirement for what happens when the tessl CLI binary is not installed or not in PATH? [Coverage]
- [ ] CHK028 - Are permission/access error scenarios covered: user lacks write permission to .wave/ directory, lockfile is owned by another user? [Coverage]
- [ ] CHK029 - Is the behavior specified for publishing a skill whose name is already taken on the registry by a different author (edge case 3 mentions it, but is the CLI flow defined)? [Coverage]
- [ ] CHK030 - Are rollback requirements specified: if tessl publish succeeds but lockfile write fails, what is the recovery path? [Coverage]
- [ ] CHK031 - Is there a requirement for publish operation idempotency across different machines (same skill published from two developers' machines)? [Coverage]
- [ ] CHK032 - Are signal handling requirements specified (what happens if user Ctrl+C during publish — is the lockfile left in a consistent state)? [Coverage]
- [ ] CHK033 - Does the spec address the case where a skill has been removed from the registry but still exists in the local lockfile? [Coverage]
- [ ] CHK034 - Is there a requirement for how verify behaves when the lockfile references a registry that is no longer configured? [Coverage]
- [ ] CHK035 - Are backwards compatibility requirements specified for the lockfile format (version field migration strategy)? [Coverage]
