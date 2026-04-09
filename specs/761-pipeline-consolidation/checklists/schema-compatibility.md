# Schema Compatibility Checklist

**Feature**: Pipeline Full Implementation Cycle Consolidation (#761)
**Generated**: 2026-04-09
**Focus**: Contract schema consistency and backward compatibility

This checklist validates the requirements quality around the critical schema evolution
that bridges existing audit pipelines with the new five-dimension audit system.

## Severity Mapping

- [ ] CHK-SC01 - Is the mapping from existing severity levels (critical/high/medium/low/info) to new levels (critical/major/minor/suggestion) documented as a normative requirement, not just a task note? [Completeness]
- [ ] CHK-SC02 - Does the spec define which component owns the mapping (audit pipelines produce old format, gate maps; OR audit pipelines produce new format, breaking existing schema)? [Clarity]
- [ ] CHK-SC03 - Is it specified whether existing audit-security output will be migrated to new severity levels or left on old levels with gate-side mapping? [Completeness]
- [ ] CHK-SC04 - Are the new type enum values (correctness/architecture/test/coverage) additive-only to shared-findings.schema.json, with no removals or renames of existing values? [Consistency]

## Schema Evolution

- [ ] CHK-SC05 - Does the spec explicitly state that existing pipelines (audit-security, audit-dead-code, etc.) MUST continue producing valid output against the modified shared-findings.schema.json? [Coverage]
- [ ] CHK-SC06 - Is the aggregated-findings.schema.json $ref to AuditFinding resolving to the same definition across both old-format and new-format findings? [Consistency]
- [ ] CHK-SC07 - Does the rework-gate-verdict.schema.json reference to previous_iteration_findings use the same finding schema as the aggregated findings? [Consistency]
- [ ] CHK-SC08 - Is the counts_by_severity field in aggregated-findings using old levels (critical/high/medium/low/info) or new levels (critical/major/minor/suggestion), and is this choice documented? [Clarity]

## Contract Handover

- [ ] CHK-SC09 - Are contract validation points specified for every step boundary in the composition pipeline (not just audit outputs)? [Completeness]
- [ ] CHK-SC10 - Is it clear whether sub-pipeline contracts (e.g., impl-issue-core's existing contracts) are validated independently or inherited by the composition? [Clarity]
- [ ] CHK-SC11 - Does the aggregate step's output contract require source_audits metadata (per T009), and is this field present in the formal schema? [Consistency]
