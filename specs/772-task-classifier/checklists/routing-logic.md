# Routing Logic Quality Checklist: 772-task-classifier

Validates the classification and routing requirements are well-specified for the domain-specific logic of this feature.

## Classification Specification Quality

- [x] CHK-R01 - Are keyword lists for each domain specified with enough examples to implement without guessing? [Completeness]
- [x] CHK-R02 - Are keyword lists for each complexity level specified with enough examples? [Completeness]
- [x] CHK-R03 - Is the keyword matching strategy defined (substring, word boundary, regex)? Plan specifies keyword presence in lowercased text. [Clarity]
- [x] CHK-R04 - Is the blast_radius base value for each complexity level explicitly defined? (simple=0.1, medium=0.3, complex=0.6, architectural=0.8) [Clarity]
- [x] CHK-R05 - Is the blast_radius domain modifier for each domain explicitly defined? (security=+0.2, performance=+0.1, docs=-0.1) [Clarity]
- [ ] CHK-R06 - Are blast_radius domain modifiers defined for ALL domains? bug, refactor, feature, and research have no explicit modifier — are they +0.0? [Completeness]
- [x] CHK-R07 - Is the verification_depth derivation a complete mapping? (all 4 complexity values map to exactly one depth) [Completeness]

## Routing Table Specification Quality

- [x] CHK-R08 - Is the routing priority chain in FR-006 total? (every valid TaskProfile maps to exactly one pipeline) [Completeness]
- [x] CHK-R09 - Are the routing rules mutually exclusive at each priority level? (no ambiguous overlaps) [Clarity]
- [x] CHK-R10 - Does the routing table handle the "performance" domain? (It falls through to complexity-based rules 6/7 — is this intentional or missing?) [Completeness]
- [x] CHK-R11 - Is the PR URL short-circuit documented as taking precedence over ALL other signals? (FR-006 rule (a) is first) [Clarity]
- [x] CHK-R12 - Does FR-006 specify behavior when Domain=refactor AND Complexity∈{simple,medium}? (Falls to rule 6 → impl-issue) [Clarity]

## Input Analysis Specification Quality

- [x] CHK-R13 - Is it specified how input and issueBody are combined for analysis? (concatenation, priority, or independent analysis) [Clarity]
- [x] CHK-R14 - Is it specified whether input_type detection happens before or after keyword analysis? (FR-003: before) [Clarity]
- [x] CHK-R15 - Does the spec define what happens when issueBody is empty string vs. not provided? [Completeness]
- [x] CHK-R16 - Is it clear that PR URL detection comes from suggest.ClassifyInput, not from keyword matching? [Clarity]

## Summary

- **Total items**: 16
- **Passing**: 15
- **Failing**: 1
- **Critical gaps**:
  - CHK-R06: Blast radius domain modifiers not explicitly defined for bug, refactor, feature, research domains — implementer must assume +0.0
