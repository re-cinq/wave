# Requirements Quality Review: 772-task-classifier

## Completeness

- [x] CHK001 - Are all four components from the feature request explicitly covered by functional requirements? (TaskProfile, analyzer, selector, tests) [Completeness]
- [x] CHK002 - Does the spec define default/fallback behavior for every classification dimension (complexity, domain, blast_radius, verification_depth)? [Completeness]
- [x] CHK003 - Are all enum values for each dimension explicitly listed in FR-001? [Completeness]
- [x] CHK004 - Does the routing table in FR-006 cover every combination of input_type, domain, and complexity that can be produced by the analyzer? [Completeness]
- [x] CHK005 - Are edge cases for empty, whitespace, and ambiguous input specified with concrete expected output values? [Completeness]
- [ ] CHK006 - Does FR-006 explicitly state what pipeline is selected for simple/medium refactors (non-architectural)? Currently they fall through to rule (6) impl-issue, but this is implicit rather than stated. [Completeness]
- [x] CHK007 - Is the reuse boundary with internal/suggest clearly specified (which functions/types are reused vs. new)? [Completeness]
- [x] CHK008 - Does the spec define the Reason field content requirements for PipelineConfig (format, minimum detail)? [Completeness]

## Clarity

- [x] CHK009 - Is the function signature for Classify unambiguous? (name collision with suggest.ClassifyInput resolved in CLR-001) [Clarity]
- [x] CHK010 - Is the blast_radius derivation formula concrete enough to implement without interpretation? (base + modifier + clamp defined in data-model.md) [Clarity]
- [x] CHK011 - Is the keyword matching strategy specified clearly enough to produce deterministic results? (keyword lists in plan.md Component 2) [Clarity]
- [x] CHK012 - Is the domain priority ordering unambiguous when multiple domains match? (FR-011 defines explicit ordering) [Clarity]
- [x] CHK013 - Are the PipelineConfig.Name values exact string literals that match real pipeline names? [Clarity]
- [x] CHK014 - Is it clear whether Classify should return an error or always return a valid profile? (Edge cases specify always-valid return) [Clarity]

## Consistency

- [ ] CHK015 - Is the domain priority ordering consistent across all artifacts? spec.md FR-011 says "security > performance > bug > refactor > feature > docs" (omits research), plan.md says "security>performance>bug>refactor>research>docs>feature", data-model.md says "security > performance > bug > refactor > feature > docs > research". Three different orderings. [Consistency]
- [x] CHK016 - Do acceptance scenario expected values match the derivation rules in FR-009/FR-010? (e.g., Story 1 Scenario 1: simple+docs→blast_radius<0.2 matches base 0.1 + modifier -0.1 = 0.0) [Consistency]
- [x] CHK017 - Are the pipeline names in FR-006 consistent with those used in acceptance scenarios? (impl-issue, impl-speckit, ops-pr-review, audit-security, doc-fix, impl-research) [Consistency]
- [x] CHK018 - Does the tasks.md dependency graph match the implementation order specified in plan.md? [Consistency]
- [x] CHK019 - Are the default values for "no keywords" (medium/feature/0.3) consistent between spec edge cases and plan Component 2? [Consistency]
- [x] CHK020 - Do all clarification resolutions (CLR-001 through CLR-005) reflect accurately in the corresponding FRs they claim to update? [Consistency]

## Coverage

- [x] CHK021 - Does every functional requirement (FR-001 through FR-012) map to at least one task in tasks.md? [Coverage]
- [x] CHK022 - Does every success criterion (SC-001 through SC-006) map to at least one task in tasks.md? [Coverage]
- [x] CHK023 - Does every user story acceptance scenario have a corresponding test case specified in the test tasks? [Coverage]
- [x] CHK024 - Are all 5 edge cases from the spec covered by at least one test task? [Coverage]
- [x] CHK025 - Does the test plan cover the full routing table (all 7 rules in FR-006 priority chain)? [Coverage]
- [x] CHK026 - Is there a test task that validates blast_radius boundary values (0.0 and 1.0 clamp behavior)? [Coverage]
- [x] CHK027 - Is there a test task for domain priority ordering when multiple domains match simultaneously? [Coverage]
- [x] CHK028 - Does the integration test task (T007) cover the critical end-to-end paths from Story 4? [Coverage]

## Summary

- **Total items**: 28
- **Passing**: 26
- **Failing**: 2
- **Critical gaps**:
  - CHK006: Simple/medium refactor routing is implicit (falls through to impl-issue) — should be explicitly stated in FR-006
  - CHK015: Domain priority ordering inconsistent across spec.md, plan.md, and data-model.md — `research` placement varies and is omitted from FR-011
