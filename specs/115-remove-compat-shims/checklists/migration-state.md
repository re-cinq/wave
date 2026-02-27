# Migration & State Requirements Quality: Remove Backwards-Compatibility Shims

**Feature**: 115-remove-compat-shims
**Date**: 2026-02-20
**Scope**: US3 (Migration Down Paths), US4 (Legacy State Store Fallback), FR-005, FR-006, FR-015

This checklist focuses on the migration and state subsystem because it has the highest
interaction complexity: environment variables, CLI commands, file deletion, checksum
validation, and database initialization all intersect.

---

## Checksum & Validation Impact

- [ ] MIG001 - Are the requirements clear about whether migration checksums are computed from `Up` SQL only, `Up + Down`, or the full `Migration` struct? (Emptying `Down` will change checksums if Down is included.) [Completeness]
- [ ] MIG002 - Is there a requirement specifying what happens when an existing database has checksums computed with old (non-empty) `Down` values and the new code computes different checksums? [Completeness]
- [ ] MIG003 - Does the spec or plan define whether `WAVE_SKIP_MIGRATION_VALIDATION=true` should be mentioned in error messages or migration notes to help users with checksum mismatches? [Completeness]
- [ ] MIG004 - Is there a task to update the `ComputeChecksum()` function if it includes `Down` SQL in its computation? [Coverage]

---

## CLI Command Behavior

- [ ] MIG005 - Are the requirements for the `wave migrate down` help text specific enough to write the text without ambiguity? (C-005 says "update help text" but doesn't specify the wording.) [Clarity]
- [ ] MIG006 - Does the spec define what exit code `wave migrate down` should return when rollback is unsupported? (Non-zero is implied but not stated.) [Completeness]
- [ ] MIG007 - Is the removal of the confirmation prompt in `wave migrate down` explicitly required by a FR, or is it only in the plan/tasks? (Only appears in research.md and tasks, not in spec FR list.) [Consistency]

---

## Environment Variable Handling

- [ ] MIG008 - Does the spec define whether `WAVE_MIGRATION_ENABLED=true` (explicit true) should still be accepted silently, or should any use of the variable trigger a warning? [Completeness]
- [ ] MIG009 - Are the requirements clear about where in the startup sequence the `WAVE_MIGRATION_ENABLED=false` error should be raised? (Before or after database connection?) [Clarity]
- [ ] MIG010 - Is there a requirement to update `WAVE_MIGRATION_ENABLED` documentation in `CLAUDE.md` or `docs/migrations.md` to reflect that the variable no longer has a meaningful false-path? [Coverage]

---

## File Deletion Safety

- [ ] MIG011 - Is there a requirement to verify that `schema.sql` is not referenced by any CI scripts, Makefiles, or Dockerfiles before deletion? [Coverage]
- [ ] MIG012 - Does the spec define the order of operations for `schema.sql` deletion vs `go:embed` removal? (Build will fail if file is deleted before embed is removed, or vice versa — must be atomic in the same commit.) [Clarity]

---

## Interaction Between US3 and US4

- [ ] MIG013 - Are the dependencies between US3 (Down path removal) and US4 (schema.sql removal) explicitly documented? (Both modify the `state` package — are there ordering constraints beyond "same phase"?) [Completeness]
- [ ] MIG014 - Is there a requirement for the intermediate verification step T025 to test both US3 and US4 changes together, or could they be verified independently? [Clarity]
