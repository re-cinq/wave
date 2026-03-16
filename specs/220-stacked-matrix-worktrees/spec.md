# Feature Specification: Stacked Worktrees for Dependent Matrix Child Pipelines

**Feature Branch**: `220-stacked-matrix-worktrees`
**Created**: 2026-03-16
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/220

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Sequential Dependency Chain Branching (Priority: P1)

A pipeline author configures an epic implementation pipeline where child issues have sequential dependencies (e.g., #206 → #207 → #208). Each dependent child pipeline needs access to the code changes produced by its predecessor so it can build upon them rather than duplicating work or encountering missing code.

**Why this priority**: This is the core value proposition. Without stacked branching, dependent matrix items operate in isolation and cannot see each other's code changes, making sequential implementation chains fundamentally broken.

**Independent Test**: Can be fully tested by running a 2-tier matrix pipeline with `stacked: true` where tier 1 creates a file and tier 2 reads that file. Tier 2 succeeds only if it can see tier 1's output.

**Acceptance Scenarios**:

1. **Given** a matrix strategy with `stacked: true` and items A (tier 0) and B (tier 1, depends on A), **When** item A completes and produces code changes on its branch, **Then** item B's child pipeline receives item A's branch as its base branch instead of the pipeline's default base.
2. **Given** a matrix strategy with `stacked: true` and a linear chain A → B → C, **When** all items complete successfully, **Then** each tier's child pipeline branches from the previous tier's output branch.
3. **Given** a matrix strategy with `stacked: true`, **When** a tier 0 item fails, **Then** all items in tier 1+ that depend on the failed item are skipped (existing behavior preserved).

---

### User Story 2 - Multi-Parent Branch Merging (Priority: P2)

A pipeline author has a diamond dependency pattern where item D depends on both items B and C (which both depend on A). Item D needs to see code changes from both B and C. The system creates a temporary integration branch that merges both parent branches so item D has a complete view of all upstream changes.

**Why this priority**: Diamond and fan-in dependency patterns are common in real epics. Without multi-parent merging, only linear chains would work with stacking.

**Independent Test**: Can be tested with a 3-tier matrix: tier 0 has item A, tier 1 has items B and C (both depend on A), tier 2 has item D (depends on B and C). Verify that D's workspace contains changes from both B and C.

**Acceptance Scenarios**:

1. **Given** item D depends on items B and C which both completed successfully, **When** D's child pipeline starts, **Then** the system creates an integration branch that merges B's and C's branches.
2. **Given** item D depends on items B and C where B's and C's changes conflict, **When** the merge fails, **Then** D is marked as failed with a clear error message indicating the merge conflict, and the integration branch is cleaned up.

---

### User Story 3 - Backward Compatibility (Priority: P1)

A pipeline author has existing matrix pipelines without the `stacked` field. These pipelines continue to work exactly as before — all child pipelines branch from the same base, and dependency tiers only control execution ordering.

**Why this priority**: Existing pipeline definitions must not break. This is a non-negotiable compatibility requirement.

**Independent Test**: Run any existing matrix pipeline YAML that lacks the `stacked` field and verify behavior is identical to current behavior.

**Acceptance Scenarios**:

1. **Given** a matrix strategy without the `stacked` field, **When** tiered execution runs, **Then** all child pipelines branch from the configured base (same as current behavior).
2. **Given** a matrix strategy with `stacked: false`, **When** tiered execution runs, **Then** behavior is identical to omitting the field.

---

### User Story 4 - Stacked Mode with Direct Worker Execution (Priority: P3)

A pipeline author uses `stacked: true` with direct step execution (no `child_pipeline` configured). The stacking applies to worker workspaces instead — each tier's workers start from a workspace that includes the previous tier's file changes.

**Why this priority**: While child pipelines are the primary use case, consistency requires that stacking also works for direct worker execution to avoid surprising behavioral gaps.

**Independent Test**: Configure a matrix step with `stacked: true` but no `child_pipeline`, where tier 0 workers create files and tier 1 workers read those files.

**Acceptance Scenarios**:

1. **Given** a matrix step with `stacked: true` and no `child_pipeline`, **When** tier 1 workers execute, **Then** their workspaces contain the files created by tier 0 workers.

---

### Edge Cases

- What happens when `stacked: true` is used without `dependency_key`? The field is ignored since there are no tiers — all items run in parallel from the same base. No error is raised.
- What happens when a tier has multiple items and `stacked: true`? Items within the same tier run in parallel from the same base (the previous tier's output). They do not stack against each other.
- What happens when the parent branch has been deleted before the child tier starts? The system reports a clear error and skips the dependent item rather than silently falling back to the default base.
- What happens when `stacked: true` is used with `max_concurrency: 1` and a single linear chain? Each item runs sequentially, branching from the previous item's branch — the simplest stacking case.
- What happens when a parent item succeeds but produces no commits? The child tier uses the parent's branch as base, which is identical to the original base. This is correct behavior — no special handling needed.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The matrix strategy configuration MUST accept an optional `stacked` boolean field that defaults to `false`.
- **FR-002**: When `stacked: true` and `dependency_key` is set, each tier's child pipelines MUST receive the previous tier's output branch as their base branch instead of the pipeline's default base.
- **FR-003**: When an item has a single parent dependency, the child pipeline MUST branch from that parent's output branch.
- **FR-004**: When an item has multiple parent dependencies (from different items in the previous tier), the system MUST create a temporary integration branch by merging all parent branches.
- **FR-005**: When a merge conflict occurs during integration branch creation (FR-004), the dependent item MUST be marked as failed with a descriptive error message.
- **FR-006**: When `stacked: false` or the field is omitted, the existing behavior MUST be preserved — all items branch from the configured base.
- **FR-007**: The system MUST extract the output branch name from completed child pipeline results to pass to dependent tiers.
- **FR-008**: When `stacked: true` is used without `dependency_key`, the field MUST be silently ignored (no error).
- **FR-009**: Integration branches created for multi-parent merging MUST be cleaned up after the dependent item completes (success or failure).
- **FR-010**: Progress events MUST include stacking-related information (e.g., which base branch is being used for each tier).

### Key Entities

- **MatrixStrategy**: Extended with a `Stacked` boolean field controlling whether tiers propagate branch context.
- **MatrixResult**: Extended to include the output branch name from a completed child pipeline, enabling downstream tiers to use it as their base.
- **Integration Branch**: A temporary git branch created by merging multiple parent branches when a child item depends on more than one parent. Named deterministically (e.g., `integration/<pipeline-id>/<item-id>`).
- **Tier Context**: The accumulated branch state passed between tiers during stacked execution — maps item IDs to their output branch names.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A 2-tier linear dependency chain with `stacked: true` completes successfully, with tier 1's workspace containing tier 0's code changes.
- **SC-002**: A diamond dependency pattern (A → B, C → D) with `stacked: true` completes successfully, with D's workspace containing changes from both B and C.
- **SC-003**: All existing matrix pipeline tests pass without modification when `stacked` is not configured.
- **SC-004**: Merge conflicts during integration branch creation produce actionable error messages that identify the conflicting parent branches.
- **SC-005**: Integration branches are not present in the repository after pipeline completion (cleaned up).
- **SC-006**: Unit test coverage for single-parent stacking, multi-parent merging, merge conflict handling, and failure propagation across stacked tiers.

## Clarifications

The following ambiguities were identified during spec review and resolved based on codebase analysis.

### Q1: How does the parent pipeline extract the output branch name from a completed child pipeline?

**Ambiguity**: FR-007 requires extracting the output branch from child pipeline results, but `NewChildExecutor()` creates independent execution state — the parent `MatrixExecutor` cannot access the child executor's `WorktreePaths` after execution completes.

**Resolution**: Extend `MatrixResult` with an `OutputBranch string` field. The `childPipelineWorker` function must capture the child executor's worktree branch name after `childExecutor.Execute()` returns and store it in the result. The child executor's `WorktreePaths` map is populated during execution (see `executor.go:1717`) — after `Execute()` returns, the parent can read the child executor's pipeline execution state to extract the branch. This is an internal implementation detail — no YAML schema change needed.

**Rationale**: This is the minimal change that fits the existing architecture. The child executor already tracks its worktree branches; we just need to surface the information back through `MatrixResult`.

### Q2: Are integration branches local-only or do they need to be pushed to the remote?

**Ambiguity**: FR-004 says "create a temporary integration branch" and FR-009 says "cleaned up after completion," but doesn't specify scope. The `worktree.Manager` operates on local branches only.

**Resolution**: Integration branches are **local-only**. They are created via `git merge` in the local repository and used as the base for the child pipeline's worktree. They are never pushed to a remote. Cleanup (FR-009) means deleting the local branch and removing the worktree via `worktree.Manager.Remove()` followed by `git branch -D`.

**Rationale**: Pushing integration branches would create remote pollution and require cleanup on failure. The child pipeline creates its own feature branch from the integration base — that feature branch is what gets pushed. The integration branch is purely a local staging mechanism.

### Q3: How does stacking work for direct worker execution (no `child_pipeline`)?

**Ambiguity**: User Story 4 says workers should see previous tier's file changes, but `createWorkerWorkspace()` creates isolated empty directories. There's no mechanism to propagate file state between tiers for non-child-pipeline workers.

**Resolution**: For direct worker execution with `stacked: true`, each tier's workers receive a **copy of the previous tier's first completed worker's workspace**. After all workers in tier N complete, the system identifies the workspace path from the first successful result in that tier and uses it as the template for tier N+1 workers. Files are copied into the new worker workspace before execution begins. If a tier has multiple successful workers that modified different files, only the first worker's state propagates (consistent with the non-stacked behavior where workers are independent). This is P3 priority and can be deferred to a follow-up if the child pipeline path (P1) is delivered first.

**Rationale**: Direct workers use basic directory workspaces, not git worktrees, so branch-based stacking doesn't apply. File copying is the simplest mechanism that achieves the stated goal. Using the first worker's output avoids the complexity of merging file trees from parallel workers.

### Q4: In a tier with partial failures and `stacked: true`, do independent downstream items still run?

**Ambiguity**: If tier 1 has items B and C, B succeeds but C fails, and tier 2 has item D that depends only on B (not C) — does D run? The spec only states "all items in tier 1+ that depend on the failed item are skipped" (US1-AS3).

**Resolution**: Yes, D runs. The existing `shouldSkipItem()` logic correctly handles this — it only skips an item if one of its **direct dependencies** failed. Items with no dependency on the failed item are unaffected. This behavior is preserved with stacking: D receives B's output branch as its base. No spec change needed — the existing behavior is correct and the spec language already supports this reading.

**Rationale**: This matches the current tiered execution behavior (`shouldSkipItem` in `matrix.go:900`). Stacking should not change failure propagation semantics — it only changes branch resolution.

### Q5: What is the "output branch" when a child pipeline uses workspace type other than "worktree"?

**Ambiguity**: FR-002 says child pipelines receive "the previous tier's output branch as their base branch." But a child pipeline might not use worktree-type workspaces, meaning it has no output branch to propagate.

**Resolution**: `stacked: true` **requires** that child pipelines use worktree-type workspaces (i.e., at least one step in the child pipeline has `workspace.type: worktree`). If a child pipeline completes without creating any worktree branch, the system treats the item as having no output branch and logs a warning. Downstream items that depend on it receive the pipeline's default base branch instead (graceful fallback). A validation warning (not error) is emitted during pipeline loading if `stacked: true` is set with a `child_pipeline` whose steps don't use worktree workspaces, since this configuration is likely a misconfiguration.

**Rationale**: Failing hard would be overly strict for a configuration that might work correctly in some cases (e.g., the child pipeline modifies shared state through other means). A warning provides actionable feedback without blocking execution.
