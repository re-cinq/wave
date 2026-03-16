# Research: Stacked Worktrees for Dependent Matrix Child Pipelines

**Feature**: #220 | **Date**: 2026-03-16

## Research Areas

### R1: Branch Propagation Between Child Pipeline Tiers

**Decision**: Extend `MatrixResult` with an `OutputBranch` field and capture it from the child executor's `WorktreePaths` map after `Execute()` returns.

**Rationale**: The child executor already tracks its worktree branches in `execution.WorktreePaths` (keyed by resolved branch name). After `childExecutor.Execute()` returns, the parent matrix executor can inspect the child executor's `PipelineExecution` to extract the branch. This requires `NewChildExecutor` to expose its execution state, or the `childPipelineWorker` function to capture the branch from the child executor's internal state.

**Alternatives Rejected**:
- **Artifact-based signaling** (child writes branch name to a JSON file): Adds I/O overhead, requires schema, and is fragile if the child pipeline doesn't have the right output step.
- **Environment variable injection**: Violates fresh-memory principle (Principle 4) and creates hidden coupling.
- **Git branch name convention**: Deterministic naming (`wave/<parent-pipeline>/<item-id>`) would work but removes flexibility for child pipelines to name their own branches.

**Implementation Detail**: The `childPipelineWorker` in `matrix.go:932` creates a `NewChildExecutor()` and calls `Execute()`. After `Execute()` returns, iterate over the child executor's `PipelineExecution.WorktreePaths` to find the branch. The child executor's `pipelines` map (private) contains the execution — we need to either expose it via a method or capture the execution during `Execute()`.

**Approach**: Add a `LastExecution() *PipelineExecution` method to `DefaultPipelineExecutor` that returns the most recent pipeline execution. This is minimal and enables the parent to read the child's worktree branch.

### R2: Multi-Parent Integration Branch Merging

**Decision**: Create local-only integration branches using `git merge` when a child item depends on multiple parents from different previous tiers.

**Rationale**: Git's native merge capability handles combining branches cleanly. Integration branches are named deterministically (`integration/<pipeline-id>/<item-id>`) for traceability and cleanup.

**Alternatives Rejected**:
- **Cherry-pick based merging**: Would require tracking individual commits, breaks when commits depend on each other, and doesn't handle file renames.
- **Octopus merge**: Git supports multi-parent merges natively. However, using sequential two-way merges is simpler to implement and provides better conflict reporting (we know which pair of branches conflicted).
- **Rebase-based approach**: Non-linear history makes cleanup harder and doesn't work well with parallel branches.

**Implementation Detail**: In the `tieredExecution` function, when building the base branch for a stacked tier item:
1. If single parent: use the parent's output branch directly as the `base` for the worktree.
2. If multiple parents: create an integration branch from the first parent, then sequentially merge each additional parent. On conflict, fail the item with a descriptive error.
3. Register the integration branch for cleanup in a `cleanupBranches` map.

### R3: Backward Compatibility for Non-Stacked Pipelines

**Decision**: Default `Stacked` to `false`. When not set or `false`, the existing `tieredExecution` code path is unchanged — all items branch from the same configured base.

**Rationale**: The `stacked` field only affects how the base branch is resolved for each tier. The tier computation, concurrency control, skip logic, and result aggregation are completely independent of stacking.

**Alternatives Rejected**:
- **Separate code path**: Duplicating `tieredExecution` for stacked mode would create maintenance burden. Better to add a conditional branch within the existing flow.

### R4: Exposing Child Executor State

**Decision**: Add `LastExecution()` method to `DefaultPipelineExecutor` and capture the output branch in the `childPipelineWorker` closure.

**Rationale**: The child executor's `pipelines` map is private. Rather than exposing the full map, a focused `LastExecution()` method provides exactly what's needed without leaking internal state.

**Implementation Detail**: The `Execute()` method stores the execution in `e.pipelines[pipelineID]`. A `LastExecution()` method returns the most recently created execution by tracking it in a new `lastExecution` field set during `Execute()`.

### R5: Integration Branch Cleanup

**Decision**: Clean up integration branches after the dependent tier completes (success or failure). Use `worktree.Manager.Remove()` followed by `git branch -D`.

**Rationale**: Integration branches are local staging mechanisms. They should not persist after use. Cleanup after each tier ensures no accumulation. If cleanup fails, log a warning but don't fail the pipeline.

### R6: Stacked Mode Without dependency_key

**Decision**: When `stacked: true` is set but `dependency_key` is empty, silently ignore stacking (FR-008). No error, no warning — it's a valid but no-op configuration.

**Rationale**: Without tiers, all items run in parallel from the same base. Stacking has no effect because there's no previous tier to stack from.

### R7: Direct Worker Stacking (P3 — Deferred)

**Decision**: Defer to follow-up. Document the approach (file copying from previous tier's first successful worker) but do not implement in this PR.

**Rationale**: P1 (child pipeline stacking) and P2 (multi-parent merging) are the core value. Direct worker stacking is P3 and involves a different mechanism (file system copying vs. git branches) that adds complexity without blocking the primary use cases.
