# Quality Review Checklist: WebUI Changed-Files Browser with Diff Views

## Completeness

- [ ] CHK001 - Are all API error response formats specified consistently across FR-001, FR-002, FR-004, FR-006, and the edge cases section? [Completeness]
- [ ] CHK002 - Is the HTTP status code defined for each error scenario (branch deleted, empty BranchName, path traversal, run not found, no base branch)? [Completeness]
- [ ] CHK003 - Are the conditions under which DiffSummary returns `available: false` vs an HTTP error status exhaustively enumerated? [Completeness]
- [ ] CHK004 - Is the behavior specified when git is unavailable on the host (not installed or not on PATH)? [Completeness]
- [ ] CHK005 - Is the user flow for navigating from the run list page to the diff view fully described, including entry point and navigation triggers? [Completeness]
- [ ] CHK006 - Are CORS or authentication requirements for the new API endpoints specified, or is it clear they inherit from existing middleware? [Completeness]
- [ ] CHK007 - Is the behavior specified when `computeDiffSummary` returns zero changed files (branch exists but no diff)? [Completeness]
- [ ] CHK008 - Is the file rename scenario fully specified? The edge case says "git rename detection is not required" but FileSummary has a "renamed" status — are the conditions for each value clear? [Completeness]

## Clarity

- [ ] CHK009 - Is the 100KB truncation limit in FR-005 unambiguously defined — does it measure raw diff text bytes, UTF-8 encoded bytes, or line count? [Clarity]
- [ ] CHK010 - Is the "configurable" nature of the size limit (FR-005) specified — where is it configured (manifest, env var, const)? [Clarity]
- [ ] CHK011 - Is the "500 lines" virtualization threshold (FR-010) clearly defined — does it count all lines including context, or only addition/deletion lines? [Clarity]
- [ ] CHK012 - Are the three diff view modes (FR-008) defined with enough detail to distinguish rendering behavior between "unified" and "raw after" (both show the same content in different formats)? [Clarity]
- [ ] CHK013 - Is the "summary bar" in FR-011 specified distinctly from the file list summary in US1 acceptance scenario 1? Are these the same element or different? [Clarity]
- [ ] CHK014 - Is the "note that the run is still active" (in-progress edge case) specified as a concrete UI element or API field? [Clarity]

## Consistency

- [ ] CHK015 - Does the diff-summary-api.json contract schema match the DiffSummary entity definition in data-model.md for all field names and types? [Consistency]
- [ ] CHK016 - Does the file-diff-api.json contract schema match the FileDiff entity definition in data-model.md for all field names and types? [Consistency]
- [ ] CHK017 - Does FR-007 (flat file list sorted alphabetically) align with the file list rendering described in US1 acceptance scenario 2? [Consistency]
- [ ] CHK018 - Are the three diff view modes in FR-008 consistent with the DiffViewMode entity values ("unified", "side-by-side", "raw") and clarification C-002? [Consistency]
- [ ] CHK019 - Does the task dependency graph (tasks.md) correctly reflect the phasing in plan.md (Phase A→B→C→D)? [Consistency]
- [ ] CHK020 - Are the success criteria (SC-001 through SC-007) each traceable to at least one functional requirement? [Consistency]

## Coverage

- [ ] CHK021 - Do acceptance scenarios cover the truncation behavior described in FR-005 (US4 scenario 2 references it, but is the indicator format specified)? [Coverage]
- [ ] CHK022 - Do acceptance scenarios cover the localStorage persistence described in FR-012 (no scenario tests mode retention across page loads)? [Coverage]
- [ ] CHK023 - Do acceptance scenarios cover syntax highlighting for all 10 specified file types, or is coverage limited to the 5 mentioned in US2 scenario 3? [Coverage]
- [ ] CHK024 - Do edge cases cover concurrent API requests (multiple users or rapid file clicks) and their expected behavior? [Coverage]
- [ ] CHK025 - Do edge cases cover the scenario where a file path in the diff contains special characters (spaces, unicode, dots)? [Coverage]
- [ ] CHK026 - Do tasks cover all 13 functional requirements (FR-001 through FR-013) — is each FR traceable to at least one task? [Coverage]
- [ ] CHK027 - Is there a success criterion for accessibility (keyboard navigation, screen reader compatibility) in the diff viewer? [Coverage]
- [ ] CHK028 - Do edge cases address what happens when the repository has submodules that appear in the diff? [Coverage]
