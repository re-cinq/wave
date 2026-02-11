# Init Command Flags & Modes Requirements Checklist

**Feature**: Release-Gated Pipeline Embedding
**Domain**: `wave init` flag interactions and mode behaviors
**Date**: 2026-02-11

---

## Flag Combinations

- [ ] CHK201 - Does the spec define behavior for all flag combinations? Specifically: `init` alone, `--all`, `--merge`, `--force`, `--all --merge`, `--all --force`, `--merge --force`, `--all --merge --force`. [Completeness]
- [ ] CHK202 - Is the interaction between `--force` and release filtering specified? Does `--force` overwrite with release-filtered content, or does it bypass filtering? [Completeness]
- [ ] CHK203 - Does CLR-004 (merge respects release filtering) have a corresponding functional requirement, or is it only captured as a clarification? Should FR-002 or a new FR explicitly cover merge mode? [Consistency]
- [ ] CHK204 - Is the `--all` flag behavior during merge mode fully specified? The edge case says "both flags compose naturally" — but does the spec define what "naturally" means with sufficient precision? [Clarity]

## Display & UX

- [ ] CHK205 - Does CLR-005 (filtered counts in display) specify whether the output should indicate that filtering occurred? Should the user see "Initialized 3 of 18 pipelines (release-only)" or just "Initialized 3 pipelines"? [Completeness]
- [ ] CHK206 - Does the spec define the warning format for zero release pipelines (FR-011)? Is the warning text specified, or left to implementation discretion? [Completeness]
- [ ] CHK207 - Does SC-007 (help text for `--all`) specify the expected help string content, or only that it must exist? [Completeness]
- [ ] CHK208 - Does the spec address whether `wave init --all` should display a notice that non-release pipelines were included, to differentiate from a normal init? [Coverage]

## Existing Behavior Preservation

- [ ] CHK209 - Does the spec define whether existing `wave init` behavior (without `--all`) changes for users who already have Wave installed? Is this a breaking change for current users who expect all pipelines? [Coverage]
- [ ] CHK210 - Are acceptance scenarios for `--merge` mode (US4, edge cases) complete enough to prevent regressions in the existing merge logic? [Coverage]
- [ ] CHK211 - Does SC-004 (existing tests pass) conflict with FR-010 (existing tests must be updated)? If tests are updated, they are no longer "existing" — is the intent clear that current behavior should be preserved where not explicitly changed? [Consistency]
