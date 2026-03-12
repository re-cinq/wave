# Contract Schema Quality Review — Closed-Issue/PR Audit Pipeline (#305)

**Feature**: `wave-audit` pipeline | **Date**: 2026-03-11

This checklist validates the quality of the four JSON Schema contracts that govern inter-step data handoffs.

## Schema Completeness

- [ ] CHK101 - Does `audit-inventory.schema.json` require the `summary` object (with `total_issues`, `total_prs`, `excluded_not_planned`) or is it optional — and is this intentional given FR-004 requires a "structured JSON inventory"? [Completeness]
- [ ] CHK102 - Does `audit-findings.schema.json` define the `unmet_criteria` and `remediation` fields as required for `partial`/`regressed` categories per FR-009, or are they globally optional? [Completeness]
- [ ] CHK103 - Does `audit-triage-report.schema.json` enforce that `prioritized_actions` items reference only non-verified/non-obsolete findings, or is this a prompt-level concern only? [Completeness]
- [ ] CHK104 - Does `audit-publish-result.schema.json` define the `issues_created` array as required when `success=true`, or can a successful publish have zero issues created? [Completeness]
- [ ] CHK105 - Are `additionalProperties` constraints defined in all four schemas to prevent uncontrolled schema drift? [Completeness]

## Schema-to-Spec Alignment

- [ ] CHK106 - Does the inventory item schema include all fields from the data model's `InventoryItem` entity — specifically `body`, `labels`, `linked_prs`, `linked_commits`, and `acceptance_criteria`? [Consistency]
- [ ] CHK107 - Does the findings schema's `category` enum match the five fidelity categories listed in FR-007 exactly (verified, partial, regressed, obsolete, unverifiable)? [Consistency]
- [ ] CHK108 - Does the triage report schema's `metadata.repository` pattern (`^[^/]+/[^/]+$`) match the inventory schema's `scope.repository` pattern for consistency? [Consistency]
- [ ] CHK109 - Does the publish result schema's `category` enum (`partial`, `regressed`) correctly reflect which categories produce publishable findings per FR-015? [Consistency]
- [ ] CHK110 - Are the `timestamp` fields using consistent format (`date-time`) across all four schemas? [Consistency]

## Schema Robustness

- [ ] CHK111 - Do the schemas define minimum array lengths where empty arrays would indicate a problem (e.g., `findings` in the triage report should it allow empty)? [Coverage]
- [ ] CHK112 - Are string fields that should never be empty protected with `minLength: 1` constraints (titles, descriptions, remediation text)? [Coverage]
- [ ] CHK113 - Does the inventory schema validate URL format for the `url` field, or is any string accepted? [Coverage]
- [ ] CHK114 - Are integer fields constrained with appropriate `minimum` values to prevent negative numbers (e.g., `item_number >= 1`, `total_audited >= 0`)? [Coverage]
- [ ] CHK115 - Does the error object in `audit-publish-result.schema.json` have a clearly defined relationship with the `success` field — is the error required when `success=false`? [Coverage]
