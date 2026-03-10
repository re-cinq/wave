# Artifact Flow Quality Checklist: Pipeline Composition UI (#261)

**Feature**: `261-tui-compose-ui` | **Date**: 2026-03-07
**Scope**: Quality validation of artifact matching, compatibility validation, and flow visualization requirements

## Matching Algorithm Specification

- [ ] CHK101 - Is the matching granularity defined — does name-only matching produce false positives when two pipelines output artifacts with the same name but different semantics? [Completeness]
- [ ] CHK102 - Is the directionality of matching specified — can only pipeline N feed pipeline N+1, or is there a mechanism for non-adjacent artifact references? [Clarity]
- [ ] CHK103 - Are the `ArtifactRef.As` alias semantics accounted for — does the matching check the original artifact name, the alias, or both? [Completeness]
- [ ] CHK104 - Is the behavior defined for pipelines where the last step has no `OutputArtifacts` but intermediate steps do? [Completeness]
- [ ] CHK105 - Is the matching behavior specified for case sensitivity — are artifact names case-sensitive matches? [Clarity]
- [ ] CHK106 - Is the `ArtifactRef.Step` field accounted for — the existing intra-pipeline injection uses step+artifact to identify a source, but cross-pipeline matching ignores the step field? [Consistency]

## Validation Status Rules

- [ ] CHK107 - Are the status escalation rules clearly defined — does a single CompatibilityError at any boundary override all CompatibilityWarning statuses? [Clarity]
- [ ] CHK108 - Is the `IsReady()` predicate behavior consistent with FR-008 — both Valid and Warning allow starting, but Error requires confirmation? [Consistency]
- [ ] CHK109 - Are diagnostic message formats specified with enough structure for programmatic consumption (JSON output for CLI) vs human-readable display? [Completeness]
- [ ] CHK110 - Is the distinction between "unmatched output" (informational) and "missing input" (warning/error) clearly defined in the requirements, not just the data model? [Clarity]

## Visualization Requirements

- [ ] CHK111 - Are the full-mode (≥120 cols) and compact-mode (<120 cols) rendering requirements defined as spec-level requirements or only in research/plan? [Completeness]
- [ ] CHK112 - Is the viewport scroll behavior for the artifact flow visualization specified — which keys scroll, what is the scroll step? [Completeness]
- [ ] CHK113 - Is the content of each pipeline box in the visualization defined — does it show step names, artifact names, artifact types, or just pipeline names? [Clarity]
- [ ] CHK114 - Is the behavior specified for sequences with many boundaries (5+ pipelines) that exceed the viewport height? [Coverage]
- [ ] CHK115 - Are color/styling requirements specified in terms of semantic meaning rather than specific terminal escape codes (allowing theme flexibility)? [Clarity]
