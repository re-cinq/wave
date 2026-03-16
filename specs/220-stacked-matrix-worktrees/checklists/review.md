# Requirements Quality Review Checklist

**Feature**: Stacked Worktrees for Dependent Matrix Child Pipelines (#220)
**Date**: 2026-03-16

## Completeness

- [ ] CHK001 - Are all transitions between stacked tier states fully specified (tier 0 → tier 1 → … → tier N), including what data flows at each boundary? [Completeness]
- [ ] CHK002 - Is the behavior specified when `stacked: true` is combined with `max_concurrency` limiting parallelism within a tier? [Completeness]
- [ ] CHK003 - Does the spec define what happens when a child pipeline produces multiple worktree branches (multi-step pipelines with different worktree workspaces)? [Completeness]
- [ ] CHK004 - Is the error reporting format for merge conflicts (FR-005) specified precisely enough for implementation — what fields, structure, or message template? [Completeness]
- [ ] CHK005 - Are progress event payloads (FR-010) defined with sufficient detail — event names, fields, and when they fire? [Completeness]
- [ ] CHK006 - Is the cleanup timing for integration branches (FR-009) specified — after each tier or after all tiers complete? [Completeness]
- [ ] CHK007 - Does the spec address the case where the parent pipeline itself is resumed — how is `TierContext` restored from persisted state? [Completeness]
- [ ] CHK008 - Is there a requirement covering the naming convention for integration branches, or is it left as an implementation detail? [Completeness]

## Clarity

- [ ] CHK009 - Is the term "output branch" unambiguously defined — does it mean the worktree branch, the feature branch pushed to remote, or the branch name in git? [Clarity]
- [ ] CHK010 - Is "first worktree branch found" (plan Phase B) deterministic when a child pipeline has multiple steps with worktree workspaces? [Clarity]
- [ ] CHK011 - Is the distinction between "base branch" (what the child pipeline forks from) and "output branch" (what the child pipeline produces) consistently used throughout? [Clarity]
- [ ] CHK012 - Does "all items branch from the configured base" (FR-006) clearly identify which configuration field provides that base? [Clarity]
- [ ] CHK013 - Is the phrase "silently ignored" (FR-008) specific enough — does it mean no log output, no event, or just no error? [Clarity]

## Consistency

- [ ] CHK014 - Are the user story acceptance scenarios consistent with the functional requirements — does every scenario trace to at least one FR? [Consistency]
- [ ] CHK015 - Is the clarification Q5 resolution (graceful fallback to default base) consistent with edge case 3 (clear error when parent branch deleted)? These describe similar but different failure modes. [Consistency]
- [ ] CHK016 - Is the P3 deferral (User Story 4 — direct worker stacking) consistently marked as out-of-scope across spec, plan, and tasks? [Consistency]
- [ ] CHK017 - Are the success criteria (SC-001 through SC-006) traceable 1:1 to test tasks in tasks.md? [Consistency]
- [ ] CHK018 - Does the plan's Phase E "selected option B" (closure wrapping) align with the tasks — is there a task that implements this specific mechanism? [Consistency]

## Coverage

- [ ] CHK019 - Is there a requirement covering observability of stacked execution in `wave logs` or `wave list runs` output? [Coverage]
- [ ] CHK020 - Are security implications addressed — can a malicious branch name in `OutputBranch` cause command injection in `git merge` or `git checkout` calls? [Coverage]
- [ ] CHK021 - Is there a requirement for what happens when the repository's working directory is dirty when integration branch creation starts? [Coverage]
- [ ] CHK022 - Does the spec cover concurrent pipeline runs that might create integration branches with colliding names (same pipeline ID and item ID)? [Coverage]
- [ ] CHK023 - Is resource cleanup specified for the case where the pipeline process is killed (SIGKILL) mid-integration-branch-creation? [Coverage]
- [ ] CHK024 - Are there requirements for documenting the `stacked` field in user-facing documentation (YAML reference, pipeline authoring guide)? [Coverage]
- [ ] CHK025 - Does the spec address performance implications — how many sequential git merges are acceptable before the approach degrades? [Coverage]
