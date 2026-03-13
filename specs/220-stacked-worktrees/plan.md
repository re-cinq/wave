# Implementation Plan: Stacked Worktrees for Dependent Matrix Child Pipelines

## Objective

Enable matrix child pipelines in tiered execution mode to propagate code changes between dependency tiers. When `stacked: true` is configured, tier N+1 child pipelines will branch from tier N's output branches instead of all branching from the same base.

## Approach

The implementation centers on the `tieredExecution` method in `internal/pipeline/matrix.go`. After each tier completes, we collect the output branch names from successful child pipeline executions and make them available as base branches for the next tier's items.

### Key Insight

The child pipeline (`gh-implement`) already creates worktrees using `workspace.type: worktree` with `branch: "{{ pipeline_id }}"` and `base: main`. The stacking mechanism needs to override the `base` parameter for child pipeline executions in dependent tiers, so they branch from a parent's output branch instead of `main`.

### Strategy: Override child pipeline workspace base at runtime

Rather than modifying child pipeline YAML definitions, we'll pass the stacked base branch through the child pipeline executor context. The `childPipelineWorker` already creates a fresh `NewChildExecutor()` per item. We can extend this to set a `baseBranchOverride` on the child executor that `createStepWorkspace` will use when resolving the `base` field for worktree workspaces.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | **modify** | Add `Stacked bool` field to `MatrixStrategy` struct |
| `.wave/schemas/wave-pipeline.schema.json` | **modify** | Add `stacked` property to `MatrixStrategy` definition |
| `internal/pipeline/matrix.go` | **modify** | Core changes: (1) `tieredExecution` collects output branches per tier, (2) `childPipelineWorker` accepts base branch override, (3) new merge logic for multi-parent stacking |
| `internal/pipeline/executor.go` | **modify** | Add `baseBranchOverride` field to `DefaultPipelineExecutor`, use it in `createStepWorkspace` when resolving `base` for worktree workspaces |
| `internal/pipeline/matrix_test.go` | **modify** | Add tests for stacked single-parent, multi-parent merge, and failure propagation |
| `.wave/pipelines/gh-implement-epic.yaml` | **modify** | Add `stacked: true` to the implement-subissues step's strategy |

## Architecture Decisions

### 1. Branch name extraction from child pipeline results

After a child pipeline completes, we need the branch name it created. The child executor's `PipelineExecution` stores `WorktreePaths` keyed by branch name. The `childPipelineWorker` currently discards the child executor state after completion. We need to capture the branch name from the child execution and include it in the `MatrixResult`.

**Decision**: Add a `BranchName string` field to `MatrixResult`. After child pipeline execution, extract the branch from the child executor's state and populate it.

### 2. Multi-parent merge strategy

When an item in tier N+1 depends on multiple items from tier N (each with different branches), we need to create a temporary integration branch that merges all parent branches.

**Decision**: Use `git merge` to create a temporary integration branch. The branch name follows the pattern `wave/stacked/<pipeline_id>/tier-<N>-merge-<hash>`. If the merge conflicts, the tier fails with a clear error.

### 3. Base branch override mechanism

Rather than deeply threading the base branch through templates, we add a `baseBranchOverride` field to `DefaultPipelineExecutor`. When set, `createStepWorkspace` uses it as the `base` for worktree creation instead of the value from the step's YAML config.

**Decision**: The override only applies when the workspace `base` field is non-empty (i.e., the step intended to use a base). Steps without `base` configured are unaffected.

### 4. Backward compatibility

When `stacked` is `false` or omitted, the behavior is completely unchanged. The stacking logic is gated behind `strategy.Stacked` check in `tieredExecution`.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Multi-parent merge conflicts | Medium | High | Fail the tier with clear error message; user can restructure dependencies |
| Branch name not available in child result | Low | High | The child pipeline always creates a worktree; we extract from the executor's WorktreePaths |
| Stale worktree references after merge | Low | Medium | Use unique branch names for integration branches; clean up on pipeline completion |
| Child pipeline workspace base override doesn't propagate to all steps | Low | High | Only the first step that creates the worktree matters; subsequent steps reuse via branch key |

## Testing Strategy

### Unit Tests
1. **Stacked field parsing**: Verify `MatrixStrategy.Stacked` is correctly parsed from YAML
2. **Single-parent stacking**: Two tiers, tier 1 item depends on tier 0 item. Verify tier 1's child executor receives tier 0's branch as base
3. **Multi-parent stacking**: Tier 1 item depends on two tier 0 items. Verify integration branch is created from merging both parent branches
4. **Failure propagation**: Tier 0 item fails, dependent tier 1 items are skipped (existing behavior preserved with stacking enabled)
5. **No stacking (default)**: When `stacked` is false/omitted, all items branch from original base

### Integration Tests
1. **Two-tier stacked chain**: Use mock adapter to simulate a child pipeline that creates a worktree and makes a commit. Verify tier 1 sees tier 0's commit in its worktree.
