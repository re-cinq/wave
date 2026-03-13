# feat(pipeline): support stacked worktrees for dependent matrix child pipelines

> GitHub Issue: https://github.com/re-cinq/wave/issues/220
> Labels: enhancement, pipeline
> Author: nextlevelshit

## Summary

When `gh-implement-epic` runs child pipelines with dependency tiers, each child pipeline currently creates its worktree from the same base branch (e.g., `main`). This means tier 1 items don't have access to tier 0's code changes, even though they depend on them.

## Problem

Given a dependency chain: `#206 → #207 → #208`

- #206 branches from `main`, implements its changes, creates a PR
- #207 branches from `main` (not from #206's branch), so it doesn't see #206's code
- #207 may duplicate work, create conflicts, or fail because expected code doesn't exist

## Proposed Solution

Add a `stacked` execution mode to the matrix strategy where each tier's child pipelines branch from the previous tier's output branch instead of the base branch:

```yaml
strategy:
  type: matrix
  child_pipeline: gh-implement
  dependency_key: "dependencies"
  stacked: true  # New field
```

When `stacked: true`:
1. Tier 0 branches from `main` (or configured base)
2. After tier 0 completes, extract the branch name from its PR result
3. Tier 1 branches from tier 0's branch (or a merge of multiple tier 0 branches)
4. Continue for subsequent tiers

### Design Considerations

- **Single parent in tier**: Straightforward — branch from parent's branch
- **Multiple parents in tier**: May need to merge parent branches into a temporary integration branch, or use the "latest" parent branch as base
- **Failure handling**: If a parent tier fails, dependent tiers should still be skipped (existing behavior)
- **Alternative**: Instead of stacking branches, merge each tier's PR before starting the next tier (slower but cleaner git history)

## Acceptance Criteria

- [ ] New `stacked` boolean field in matrix strategy configuration is parsed and validated
- [ ] When `stacked: true`, tier N+1 child pipelines receive the branch name(s) from tier N's completed child pipeline(s) as their base branch
- [ ] Single-parent case: child pipeline branches from parent's output branch
- [ ] Multi-parent case: parent branches are merged into a temporary integration branch used as the base for the child pipeline
- [ ] If a parent tier fails, dependent tiers are skipped (existing behavior preserved)
- [ ] When `stacked: false` or omitted, existing behavior is unchanged (all tiers branch from base)
- [ ] Unit tests cover single-parent stacking, multi-parent merging, and failure propagation
- [ ] Integration test demonstrates a 2-tier chain where tier 1 sees tier 0's code changes

## Current Behavior

All child pipelines branch from the same base. Dependencies only control execution ordering — "don't start until the prior pipeline finishes" — but don't propagate code changes between tiers.

## Context

Discovered during the first live run of `gh-implement-epic` against #184. The current model works when subissues touch independent parts of the codebase, but breaks down for truly sequential implementation chains.

## Related

- Part of the `gh-implement-epic` pipeline (#184)
- Matrix dependency tiers: `internal/pipeline/matrix.go` (`tieredExecution`)
- Child pipeline invocation: `internal/pipeline/matrix.go` (`childPipelineWorker`)
