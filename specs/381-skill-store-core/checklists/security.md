# Security Requirements Quality: Skill Store Core

**Feature**: #381 — Skill Store Core
**Date**: 2026-03-13
**Focus**: Path traversal, input validation, and filesystem security requirements

## Path Traversal Prevention

- [ ] CHK-S01 - Does the spec define path traversal prevention as a pre-filesystem check (i.e., name validation rejects before any `os.Stat` or `filepath.Join`)? [Security-Completeness]
- [ ] CHK-S02 - Is the defense-in-depth strategy (regex + `filepath.Join` containment check) described in research.md also reflected in spec.md requirements? [Security-Consistency]
- [ ] CHK-S03 - Does the spec cover path traversal via the `SourcePath` field? (Can a caller construct a Skill with a malicious SourcePath and pass it to Write?) [Security-Coverage]
- [ ] CHK-S04 - Are all four CRUD operations listed in FR-009 as requiring path traversal prevention, and do acceptance scenarios exist for each? [Security-Coverage]

## Input Validation Boundaries

- [ ] CHK-S05 - Is it clear that name validation via regex is the SOLE defense against path traversal (per RQ-3), or are additional filesystem containment checks also required? [Security-Clarity]
- [ ] CHK-S06 - Does the spec address what happens when `metadata` map keys or values contain control characters, null bytes, or excessively long strings? [Security-Coverage]
- [ ] CHK-S07 - Is the `Value` field of `ParseError` described as "sanitized for security" — does the spec define what sanitization means here? (e.g., truncation, character escaping) [Security-Clarity]
- [ ] CHK-S08 - Does the spec define validation for the `license` field, or is it accepted as any arbitrary string with no length constraint? [Security-Completeness]

## Filesystem Safety

- [ ] CHK-S09 - Does FR-009 or any requirement specify that `os.RemoveAll` in Delete must verify the target is within a known source root before executing? [Security-Completeness]
- [ ] CHK-S10 - Is the file permission mode for created directories (0755) and files (0644) specified in the spec or only in the plan? [Security-Consistency]
