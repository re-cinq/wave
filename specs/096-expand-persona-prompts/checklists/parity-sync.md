# Parity & Sync Quality Checklist: 096-expand-persona-prompts

> Validates that parity and synchronization requirements between .wave/personas/
> and internal/defaults/personas/ are specified with sufficient precision and coverage.

## Parity Definition

- [ ] CHK201 - Is the meaning of "byte-identical" explicitly defined (content identical, not just semantically equivalent)? [Clarity]
- [ ] CHK202 - Does the requirement specify which directory is the canonical source of truth (.wave/personas/) and which is the sync target? [Completeness]
- [ ] CHK203 - Is the sync direction explicitly specified (one-way from .wave/ to internal/defaults/, not bidirectional)? [Clarity]
- [ ] CHK204 - Does the spec address the file set: must both directories contain exactly the same set of filenames, or only parity for overlapping files? [Completeness]

## Validation Approach

- [ ] CHK205 - Is the validation command (`diff -r`) specified as a concrete, executable step in the plan? [Completeness]
- [ ] CHK206 - Does the spec define the expected exit behavior of the diff command (exit code 0 = pass)? [Clarity]
- [ ] CHK207 - Is there a requirement to validate parity after the FR-008 fixes land (i.e., sync must happen after content edits, not before)? [Consistency]

## Embedding & Build Impact

- [ ] CHK208 - Does the spec confirm that updating .md files in internal/defaults/personas/ changes embedded content at compile time without modifying .go files? [Completeness]
- [ ] CHK209 - Is the relationship between `//go:embed` and the persona files documented clearly enough that an implementer understands they don't need to touch embed.go? [Clarity]
- [ ] CHK210 - Does the spec address whether `wave init` behavior needs verification after the sync (i.e., new projects get expanded personas)? [Coverage]

## Risk & Edge Cases

- [ ] CHK211 - Does the spec address what happens if files exist in one directory but not the other? [Coverage]
- [ ] CHK212 - Is there a requirement preventing the addition or removal of persona files as part of this change? [Completeness]
- [ ] CHK213 - Does the spec address file encoding requirements (UTF-8, line endings) to prevent false parity failures? [Coverage]
