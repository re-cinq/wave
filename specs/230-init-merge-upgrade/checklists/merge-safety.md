# Merge Safety Checklist: Init Merge & Upgrade Workflow

**Feature**: #230 | **Date**: 2026-03-04
**Focus**: Data integrity, destructive operation prevention, and user safety

---

## Data Integrity

- [ ] CHK101 - Are requirements defined for preserving file permissions and ownership during merge operations? [Completeness]
- [ ] CHK102 - Is it specified whether symbolic links in `.wave/` directories are followed or preserved as-is during comparison and merge? [Completeness]
- [ ] CHK103 - Are requirements defined for handling partial write failures (e.g., disk full after writing 3 of 5 new files)? Should the operation be atomic (all-or-nothing) or best-effort? [Completeness]
- [ ] CHK104 - Is the byte-for-byte comparison in C-5 specified to handle platform-specific line endings (CRLF vs LF) deterministically? [Clarity]
- [ ] CHK105 - Is the behavior defined when a user file has the same content as the default but different file metadata (timestamps, permissions)? Should it be "up to date" or "preserved"? [Clarity]

## Destructive Operation Prevention

- [ ] CHK106 - Is it explicitly stated that `wave init --merge` NEVER deletes existing files, even if they no longer correspond to any embedded default? [Completeness]
- [ ] CHK107 - Is the "abort" path (user answers "N" at confirmation) guaranteed to have zero side effects — including no temporary files left behind? [Completeness]
- [ ] CHK108 - Is it specified whether the change summary computation itself has side effects (e.g., creating directories needed for comparison)? [Clarity]
- [ ] CHK109 - Are the requirements clear that `--force` without `--merge` is the ONLY path that overwrites user files? [Consistency]

## User Safety

- [ ] CHK110 - Is the confirmation prompt wording specified, or just the existence of a prompt? Does it clearly communicate the risk level of proceeding? [Completeness]
- [ ] CHK111 - Is the stderr output for `--merge --force`/`--merge --yes` sufficient for audit trails in CI/CD environments? [Coverage]
- [ ] CHK112 - Are requirements defined for what the user should do if they accidentally ran `--force` instead of `--merge`? Is there recovery guidance? [Coverage]
- [ ] CHK113 - Is the "Already up to date" message defined clearly enough to distinguish from error conditions? [Clarity]
