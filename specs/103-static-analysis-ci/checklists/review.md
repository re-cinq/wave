# Requirements Quality Review Checklist

**Feature**: 103-static-analysis-ci — Static Analysis for Unused/Redundant Go Code
**Date**: 2026-02-18
**Artifacts Reviewed**: spec.md, plan.md, tasks.md, research.md, data-model.md, contracts/

---

## Completeness

- [ ] CHK001 - Are all linters listed in FR-003 traced to at least one acceptance scenario that would exercise them? [Completeness]
- [ ] CHK002 - Does the spec define what "new or modified code" means for `only-new-issues` mode (diff algorithm, merge-base selection)? [Completeness]
- [ ] CHK003 - Is there a requirement specifying the Go version constraint for golangci-lint v2 compatibility (minimum Go version required by the linter itself)? [Completeness]
- [ ] CHK004 - Does the spec address what happens when the `golangci-lint-action` cache is invalidated or unavailable (e.g., first run after cache expiry)? [Completeness]
- [ ] CHK005 - Are the permissions required by the CI workflow job defined (e.g., `contents: read`, `checks: write` for PR annotations)? [Completeness]
- [ ] CHK006 - Does the spec define the expected output format for CI lint results (inline PR annotations vs. workflow summary vs. check run)? [Completeness]
- [ ] CHK007 - Is there a requirement specifying what `golangci-lint run ./...` exits with when violations are found (exit code behavior)? [Completeness]
- [ ] CHK008 - Does the spec address whether the lint workflow should run on draft PRs? [Completeness]

## Clarity

- [ ] CHK009 - FR-006 specifies "v9 or later" for the action, but research.md notes the latest is v7 — is the minimum bound clearly documented as aspirational vs. hard requirement? [Clarity]
- [ ] CHK010 - FR-007 specifies "v2.9 or later" for golangci-lint, but research.md references v2.1.6 — are the version floor values internally consistent across artifacts? [Clarity]
- [ ] CHK011 - Is the distinction between "incremental mode" in CI (`only-new-issues`) and full-scan mode locally (`golangci-lint run ./...`) explicitly stated in a single location? [Clarity]
- [ ] CHK012 - Does FR-009 clearly distinguish between "no generated files exist today" and "an exclusion mechanism is available for future use"? [Clarity]
- [ ] CHK013 - Is the term "standard preset" defined with its exact linter membership, or does it rely on external documentation? [Clarity]
- [ ] CHK014 - Does C-005 clearly resolve whether FR-006/FR-007 minimum bounds are hard gates or guidance for the implementer? [Clarity]

## Consistency

- [ ] CHK015 - Are the linters listed in FR-002 (standard preset members) consistent with the golangci-lint v2 documentation for that preset? [Consistency]
- [ ] CHK016 - Is the `actions/checkout` version (v4) in the data model consistent with what the plan and tasks specify? [Consistency]
- [ ] CHK017 - Does the task list (T004-T006) cover all structural requirements from the lint-workflow contract? [Consistency]
- [ ] CHK018 - Is FR-010's command (`golangci-lint run ./...`) consistent with the Makefile contract's `contains` check? [Consistency]
- [ ] CHK019 - Are the exclusion presets in FR-013 (`std-error-handling`, `comments`) consistent with the golangci-config contract's `contains` check? [Consistency]
- [ ] CHK020 - Does the task dependency graph in tasks.md match the entity dependency graph in data-model.md? [Consistency]
- [ ] CHK021 - Is the `go-version-file: go.mod` value consistent across the data model, contract, and task descriptions? [Consistency]

## Coverage

- [ ] CHK022 - Does every functional requirement (FR-001 through FR-015) have at least one task that implements it? [Coverage]
- [ ] CHK023 - Does every success criterion (SC-001 through SC-009) have at least one validation task or acceptance scenario? [Coverage]
- [ ] CHK024 - Are all edge cases (6 total) addressed by either a requirement, a clarification, or an explicit out-of-scope declaration? [Coverage]
- [ ] CHK025 - Does every user story have acceptance scenarios that are independently testable without requiring other stories to be implemented first? [Coverage]
- [ ] CHK026 - Is there a contract validation defined for each file that will be created or modified? [Coverage]
- [ ] CHK027 - Does the task-to-FR traceability matrix in tasks.md account for all 15 functional requirements without gaps? [Coverage]
- [ ] CHK028 - Are negative test scenarios defined (e.g., what should NOT happen) for critical requirements like FR-014 (revive exclusion)? [Coverage]
