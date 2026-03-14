# Quality Review Checklist: Wave Init Interactive Skill Selection

**Feature**: `384-init-skill-selection`
**Date**: 2026-03-14
**Artifact Scope**: spec.md, plan.md, tasks.md, data-model.md, research.md

---

## Completeness

- [ ] CHK001 - Are error recovery paths defined for ALL user-facing operations (ecosystem select, skill list, install)? [Completeness]
- [ ] CHK002 - Is the behavior specified when `tessl search ""` returns zero results (empty registry)? [Completeness]
- [ ] CHK003 - Are timeout thresholds defined for subprocess calls (`tessl search`, adapter `Install()`)? [Completeness]
- [ ] CHK004 - Is the ordering of installed skills in `wave.yaml` specified (alphabetical, insertion order, or unspecified)? [Completeness]
- [ ] CHK005 - Is the maximum number of skills a user can select from tessl bounded or unbounded? Are display limits for very long skill lists addressed? [Completeness]
- [ ] CHK006 - Does FR-009 (reconfigure) specify what happens when a previously installed skill is no longer available in the ecosystem registry? [Completeness]
- [ ] CHK007 - Is the behavior defined when `wave init` is run in a project that already has a `.wave/skills/` directory with manually installed skills (not via init)? [Completeness]
- [ ] CHK008 - Are accessibility requirements for the `huh` form interactions specified (keyboard navigation, screen reader compatibility)? [Completeness]
- [ ] CHK009 - Is the expected `tessl search` output format documented or referenced so the parser contract is clear? [Completeness]
- [ ] CHK010 - Does the spec address what happens when the user selects an ecosystem, installs skills, then re-runs `wave init` (not `--reconfigure`) — are existing skills overwritten or preserved? [Completeness]

## Clarity

- [ ] CHK011 - Does FR-002 clearly distinguish the two interaction modes (multi-select vs. confirm/skip) such that an implementer knows exactly when each applies? [Clarity]
- [ ] CHK012 - Is the term "bare names" in FR-005 defined with examples covering edge cases (names with hyphens, dots, slashes)? [Clarity]
- [ ] CHK013 - Does FR-006 define "non-interactive mode" exhaustively — is non-TTY detection separate from `--yes` or equivalent? [Clarity]
- [ ] CHK014 - Is the "install instructions" display in US4/FR-007 specified — is it just text or does it offer to run the install command? [Clarity]
- [ ] CHK015 - Does the spec clearly state whether "Skip" at ecosystem level means "no ecosystem chosen" vs. "ecosystem chosen but no skills selected"? [Clarity]
- [ ] CHK016 - Is the phrase "shown as context" in FR-009/US5 defined — is it informational text, pre-selected options, or something else? [Clarity]
- [ ] CHK017 - Are the success/failure status labels in FR-004 specified (exact wording: "success"/"failed"/"error"/etc.)? [Clarity]

## Consistency

- [ ] CHK018 - Does FR-012 (Step 6 insertion) align with the data-model's flow diagram showing Step 6 between model selection and writeManifest()? [Consistency]
- [ ] CHK019 - Does the plan's "~300 lines new code" estimate align with the 28 tasks in tasks.md — is the scope consistent across artifacts? [Consistency]
- [ ] CHK020 - Does the research decision on `parseTesslSearchOutput()` duplication (RQ-3) align with the task definition in T010 and the data model? [Consistency]
- [ ] CHK021 - Are the ecosystem CLI dependencies in data-model.md consistent with those defined in `internal/skill/source_cli.go`? [Consistency]
- [ ] CHK022 - Does the task dependency graph in tasks.md correctly reflect the sequential constraints implied by the spec (e.g., ecosystem select before skill browse before install)? [Consistency]
- [ ] CHK023 - Does SC-003 ("no `skills:` key in wave.yaml") align with FR-005/FR-013 behavior when skills list is empty? [Consistency]
- [ ] CHK024 - Are the acceptance scenario counts in the spec (US1:4, US2:5, US3:4, US4:3, US5:2) sufficient to cover all functional requirements (FR-001 through FR-013)? [Consistency]

## Coverage

- [ ] CHK025 - Are concurrent `wave init` runs addressed — what happens if two terminals run init simultaneously? [Coverage]
- [ ] CHK026 - Is filesystem permission failure during `.wave/skills/` creation or skill file writing covered as an edge case? [Coverage]
- [ ] CHK027 - Does the test plan (Phase 8 tasks) include integration-level tests for the full ecosystem→browse→install→manifest flow, or only unit tests? [Coverage]
- [ ] CHK028 - Is the behavior when `SourceRouter.Install()` returns partial success (some skills installed, some failed) tested in the acceptance criteria? [Coverage]
- [ ] CHK029 - Are all 5 edge cases in the spec traceable to at least one acceptance scenario or functional requirement? [Coverage]
- [ ] CHK030 - Does the test coverage requirement (SC-009) specify minimum coverage thresholds or just "has tests"? [Coverage]
- [ ] CHK031 - Is the `huh.ErrUserAborted` handling (T026) covered by acceptance criteria, or only by a polish task? [Coverage]
- [ ] CHK032 - Are negative test cases defined for `parseTesslSearchOutput()` — malformed input, binary data, extremely long lines? [Coverage]
