# Philosopher Output Parsing Quality Checklist

**Feature**: Restore and Stabilize `wave meta` Dynamic Pipeline Generation
**Date**: 2026-03-16
**Scope**: FR-002, FR-012, Edge Cases 1–3

This checklist validates requirements quality for the philosopher output parsing subsystem — the most failure-prone component in the meta pipeline.

---

## Delimiter Protocol

- [ ] CHK101 - Is the exact delimiter syntax specified (e.g., must `--- PIPELINE ---` be on its own line, or can it appear mid-line)? [Clarity]
- [ ] CHK102 - Is whitespace tolerance around delimiters defined (leading/trailing spaces, blank lines between delimiter and content)? [Clarity]
- [ ] CHK103 - Is the behavior defined when delimiters appear more than once in the output (duplicate sections)? [Completeness]
- [ ] CHK104 - Is the ordering requirement specified — must `--- PIPELINE ---` always precede `--- SCHEMAS ---`? [Clarity]

## YAML Parsing

- [ ] CHK105 - Are requirements for YAML validity scoped — must the pipeline YAML be a single document, or can it be multi-document? [Clarity]
- [ ] CHK106 - Is the expected YAML structure specified beyond "pipeline YAML" — required top-level keys, step structure? [Completeness]
- [ ] CHK107 - Is behavior defined when the philosopher emits valid YAML that doesn't represent a valid pipeline (e.g., missing `steps` key)? [Completeness]

## Schema Handling

- [ ] CHK108 - Is the schema file naming convention specified — how are filenames derived from the schemas section? [Completeness]
- [ ] CHK109 - Are the "common JSON schema errors" that FR-012 auto-repairs enumerated, or is the repair scope open-ended? [Clarity]
- [ ] CHK110 - Is the behavior defined when a schema references `$ref` to external schemas — are external references allowed? [Coverage]
- [ ] CHK111 - Is the minimum valid schema defined — is `{"type": "object"}` sufficient, or are additional fields required? [Clarity]

## Error Reporting

- [ ] CHK112 - Does the spec require raw philosopher output to be preserved in error messages for debugging (Edge Case 1)? [Completeness]
- [ ] CHK113 - Is the error message format for circular dependencies specified — should it include the cycle path? [Clarity]
- [ ] CHK114 - Is the distinction between parse errors (malformed YAML) and validation errors (valid YAML, invalid pipeline) clear in error reporting requirements? [Clarity]
