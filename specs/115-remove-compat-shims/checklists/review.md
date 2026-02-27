# Requirements Quality Review: Remove Backwards-Compatibility Shims

**Feature**: 115-remove-compat-shims
**Date**: 2026-02-20
**Artifacts Reviewed**: spec.md, plan.md, tasks.md, research.md, data-model.md

---

## Completeness — Are all necessary requirements captured?

- [ ] CHK001 - Does the spec define the expected behavior when a YAML pipeline config contains a `strictMode` key after the field is removed from the Go struct? (The edge case mentions it should "fail with a clear validation error" but no FR explicitly requires adding a validation error — the data-model notes it will be "silently ignored" which contradicts the edge case.) [Completeness]
- [ ] CHK002 - Does FR-003 specify which valid JSON extraction scenarios the recovery parser must handle, or is there a reference to existing test coverage that defines the boundary? [Completeness]
- [ ] CHK003 - Does FR-004 specify what error message or error type is returned when `--- PIPELINE ---` markers are absent, or just that "an error" is returned? [Completeness]
- [ ] CHK004 - Is there a requirement for updating migration checksums after Down SQL is emptied? (Research Area 4 notes "Migration checksums will change" and the data-model confirms this, but no FR or task addresses recalculating or handling checksum validation failures.) [Completeness]
- [ ] CHK005 - Does the spec address what happens to in-progress or partially-applied migrations when Down paths become empty? (Edge case for `wave migrate down` is covered, but not for a mid-upgrade failure recovery scenario.) [Completeness]
- [ ] CHK006 - Is there a requirement for the exact error message text when `WAVE_MIGRATION_ENABLED=false` is set? (C-003 resolution specifies the message, but no FR captures the specific wording as a testable requirement.) [Completeness]
- [ ] CHK007 - Does the spec capture whether the `schema.sql` deletion should be a separate commit or can be combined with the `store.go` changes? (Task T024 is separate from T021-T023 but no ordering constraint is documented.) [Completeness]

---

## Clarity — Are requirements unambiguous and testable?

- [ ] CHK008 - Is the phrase "recovery parser" in FR-003 and acceptance scenario 2.1 defined or referenced clearly enough for an implementer to know which code path this refers to? [Clarity]
- [ ] CHK009 - Is the acceptance scenario 2.3 ("returns a clear error instead of silently falling back") specific enough about what "clear error" means — error type, message content, or just non-nil error? [Clarity]
- [ ] CHK010 - Is the wording of FR-005 ("set all Down SQL fields to empty strings") unambiguous about whether the empty string means `""` vs removing the content but keeping the field assignment (e.g., `Down: "",` vs omitting `Down:`)? [Clarity]
- [ ] CHK011 - Does FR-008 ("remove or update all source code comments that reference backwards-compatibility for removed functionality") provide clear criteria for distinguishing comments about removed functionality from comments about preserved functionality? [Clarity]
- [ ] CHK012 - Is FR-015 ("update the wave migrate down CLI command to return a clear error") now superseded by C-005's resolution that the existing `MigrateDown()` error is sufficient? The FR and clarification seem to diverge on whether CLI command changes are needed. [Clarity]

---

## Consistency — Do requirements, plan, and tasks agree with each other?

- [ ] CHK013 - The edge case says `strictMode` in pipeline YAML "should fail with a clear validation error" but the data-model says the key "will be silently ignored." Which behavior is intended? The spec and data-model are contradictory. [Consistency]
- [ ] CHK014 - FR-015 requires updating the `wave migrate down` CLI command, but C-005 states the existing `MigrateDown()` error is sufficient and only help text needs updating. Are the FR and clarification aligned? [Consistency]
- [ ] CHK015 - The plan's risk assessment mentions "Migration checksums will change" as medium likelihood/medium impact, but no task in tasks.md addresses checksum recalculation or `WAVE_SKIP_MIGRATION_VALIDATION`. Is this a missing task or an accepted risk? [Consistency]
- [ ] CHK016 - Task T006 references `internal/contract/jsonschema.go:333` for a "StrictMode comment" update, but research.md maps this same file:line. Are the line numbers still accurate after T005 removes lines 266-267? [Consistency]
- [ ] CHK017 - The spec says all pipeline YAML configs must "continue to function without modification" (SC-007), but if the `strictMode` key becomes unrecognized, any YAML using it would silently lose that setting. Is there a task to verify no pipeline YAML uses `strictMode`? [Consistency]

---

## Coverage — Are edge cases, risks, and non-functional concerns addressed?

- [ ] CHK018 - Is there a requirement or task to verify that no external callers (tests, integration scripts) reference `extractJSONFromTextLegacy` or `extractYAMLLegacy` by name? [Coverage]
- [ ] CHK019 - Does the spec address the behavior when the recovery JSON parser encounters input that the legacy parser could handle but the recovery parser cannot? (Is there a defined "gap" between the two parsers' capabilities?) [Coverage]
- [ ] CHK020 - Is there a task to verify that `go build` succeeds after `schema.sql` deletion (to catch any lingering `go:embed` references in other files or build tags)? [Coverage]
- [ ] CHK021 - Does the spec define rollback strategy if the combined changes cause unexpected test failures? (The plan says "move fast" but with ~15 files changed, is there a checkpoint or incremental verification approach?) [Coverage]
- [ ] CHK022 - Is there a requirement to verify that `go generate ./...` (if used) still succeeds after the removals? [Coverage]
- [ ] CHK023 - Does SC-001 ("Zero references to backwards compat...in Go source files that describe removed functionality") provide sufficient guidance on how to distinguish stale references from valid ones? (What about comments describing the removal itself?) [Coverage]
- [ ] CHK024 - Is there a task to check that the `resume.go` workspace lookup removal doesn't break any test fixtures that create directories without hash suffixes? [Coverage]
- [ ] CHK025 - Does the spec address whether `WAVE_MIGRATION_ENABLED` should be removed from documentation (CLAUDE.md, docs/migrations.md) after the legacy path is removed, or only the runtime behavior? [Coverage]
