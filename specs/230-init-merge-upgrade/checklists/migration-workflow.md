# Migration Workflow Checklist: Init Merge & Upgrade Workflow

**Feature**: #230 | **Date**: 2026-03-04
**Focus**: Database migration requirements and the two-step upgrade lifecycle

---

## Migration Requirements

- [ ] CHK201 - Is the `wave migrate validate` command's behavior fully specified — what does "integrity status" mean and what checks are performed? [Clarity]
- [ ] CHK202 - Are requirements defined for migration rollback (`wave migrate down`) in case an upgrade migration fails partway? [Completeness]
- [ ] CHK203 - Is the expected output format of `wave migrate status` specified (table, JSON, plain text)? [Completeness]
- [ ] CHK204 - Is the behavior defined when a migration fails mid-execution — is the database left in a partially migrated state or rolled back? [Completeness]
- [ ] CHK205 - Are requirements defined for what happens when the user skips `wave migrate up` after `wave init --merge`? Will Wave detect the schema mismatch at runtime? [Coverage]

## Upgrade Lifecycle

- [ ] CHK206 - Is the two-step workflow (init --merge → migrate up) the ONLY supported upgrade path, or can users run them in different order? [Clarity]
- [ ] CHK207 - Are requirements defined for upgrading across multiple versions at once (e.g., v0.5 → v0.8) vs sequential upgrades? [Completeness]
- [ ] CHK208 - Is the relationship between binary version, embedded defaults version, and database schema version explicitly defined? [Completeness]
- [ ] CHK209 - Is it specified whether `wave init --merge` should warn if pending migrations exist before proceeding? [Coverage]

## Documentation Quality

- [ ] CHK210 - Does the documentation task (T017) specify expected output examples that match the actual implementation's output format? [Consistency]
- [ ] CHK211 - Are troubleshooting scenarios covered in the upgrade guide for: migration checksum mismatch, stale database, binary version mismatch? [Coverage]
- [ ] CHK212 - Is the documentation structured to serve both first-time upgraders and experienced users (quick reference vs detailed walkthrough)? [Completeness]
