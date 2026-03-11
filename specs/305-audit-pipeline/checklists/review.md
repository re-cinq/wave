# Requirements Quality Review — Closed-Issue/PR Audit Pipeline (#305)

**Feature**: `wave-audit` pipeline | **Date**: 2026-03-11 | **Spec**: [spec.md](../spec.md)

## Completeness

- [ ] CHK001 - Are all five fidelity categories (verified, partial, regressed, obsolete, unverifiable) defined with unambiguous classification criteria that prevent overlap between categories? [Completeness]
- [ ] CHK002 - Does the spec define what happens when a single inventory item could match multiple fidelity categories (e.g., partially implemented AND partially regressed)? [Completeness]
- [ ] CHK003 - Are the input requirements for `wave run wave-audit` fully specified — what happens when no CLI input is provided vs. invalid input vs. malformed scope expressions? [Completeness]
- [ ] CHK004 - Does the spec define the maximum inventory size the pipeline is expected to handle, and what happens when the inventory exceeds adapter context limits? [Completeness]
- [ ] CHK005 - Are persona selection requirements documented for all four steps, including the rationale for P5 deviation and the read-only constraints for analysis steps? [Completeness]
- [ ] CHK006 - Does FR-011 specify how "not planned" close reason detection works across different GitHub close reason values (e.g., `NOT_PLANNED`, `not_planned`, locale variations)? [Completeness]
- [ ] CHK007 - Are timeout requirements specified per-step, or only at the pipeline level (SC-007 says 90 minutes per step — is this configurable or fixed)? [Completeness]
- [ ] CHK008 - Does the spec define the expected behavior when the `gh` CLI is not installed or not authenticated? [Completeness]

## Clarity

- [ ] CHK009 - Is the distinction between "regressed" and "obsolete" clearly defined — what criteria determine whether removed code is a regression vs. intentional deprecation? [Clarity]
- [ ] CHK010 - Is "static analysis only" (C4) sufficiently precise — does it include or exclude `git log` execution, `gh` CLI queries in the audit step, or file content hashing? [Clarity]
- [ ] CHK011 - Is the scope parsing syntax unambiguous — can "last 30 days" be confused with a label filter, and is the parsing precedence documented? [Clarity]
- [ ] CHK012 - Does SC-002 (90% accuracy for "verified" classifications) define how accuracy is measured — against manual spot-checks, automated verification, or a ground-truth dataset? [Clarity]
- [ ] CHK013 - Is the term "acceptance criteria" defined consistently — does it refer only to checklist items in the issue body, or also to requirements stated in prose? [Clarity]
- [ ] CHK014 - Does the spec clarify what "supporting evidence" means for each fidelity category — are minimum evidence requirements defined per category? [Clarity]

## Consistency

- [ ] CHK015 - Does the 4-step pipeline decomposition (C2) align with all 15 functional requirements — is every FR addressed by at least one step? [Consistency]
- [ ] CHK016 - Are the contract schema field names consistent with the data model entity definitions (e.g., `item_number` vs `number`, `item_type` vs `type`)? [Consistency]
- [ ] CHK017 - Does the inventory schema's `close_reason` field accept the same values referenced in FR-011 and the data model (completed, not_planned, merged)? [Consistency]
- [ ] CHK018 - Are the user stories' acceptance scenarios testable against the success criteria — does each SC map to at least one acceptance scenario? [Consistency]
- [ ] CHK019 - Does the plan's "no Go code changes" claim align with the testing requirement (T017) — can `go test ./...` validate YAML-only additions without new test code? [Consistency]
- [ ] CHK020 - Is the `unverifiable` category consistently handled — FR-007 lists 5 categories, but does the priority ordering in the data model correctly exclude `verified` and `obsolete` from `prioritized_actions`? [Consistency]

## Coverage

- [ ] CHK021 - Are error scenarios specified for each pipeline step — what happens when `collect-inventory` fails, and how does it affect downstream steps? [Coverage]
- [ ] CHK022 - Does the spec address the case where a repository has zero closed issues and zero merged PRs? [Coverage]
- [ ] CHK023 - Are concurrent execution scenarios addressed — what happens if two `wave-audit` runs execute simultaneously on the same repository? [Coverage]
- [ ] CHK024 - Does the spec define behavior for private repositories, forks, or repositories where the `gh` CLI user lacks read access to some issues? [Coverage]
- [ ] CHK025 - Are all seven edge cases listed in the spec cross-referenced to specific functional requirements or handling instructions in step prompts? [Coverage]
- [ ] CHK026 - Does the spec address how issues with only PR references (no direct code changes) are verified — e.g., documentation-only issues or process changes? [Coverage]
- [ ] CHK027 - Is the resume scenario (US4) tested for each step boundary — not just inventory→audit, but also audit→triage and triage→publish? [Coverage]
- [ ] CHK028 - Does the spec define the expected output when ALL items are classified as "verified" — is the prioritized_actions array empty, and is the publish step skipped entirely? [Coverage]
