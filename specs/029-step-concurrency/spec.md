# feat: Add concurrency property to spawn multiple agents for the same step

**Issue**: [#29](https://github.com/re-cinq/wave/issues/29)
**Labels**: enhancement
**Author**: nextlevelshit

## Summary

Add a `concurrency` property to pipeline step definitions that allows spawning multiple agent instances to process work in parallel.

## Motivation

Some pipeline steps could benefit from parallel execution, particularly when processing multiple items or performing independent operations. This would improve pipeline throughput and reduce overall execution time.

## Proposed Solution

Add a `concurrency` field to step configuration in `wave.yaml`:

```yaml
steps:
  - name: process-items
    persona: worker
    concurrency: 3  # Spawn 3 agents in parallel
```

## Acceptance Criteria

- [ ] `concurrency` property is recognized in step configuration
- [ ] Multiple agent instances can be spawned for a single step
- [ ] Results from concurrent agents are properly collected and merged
- [ ] Error handling works correctly with concurrent agents
- [ ] Documentation updated with concurrency examples

## Technical Considerations

- How should work be distributed among concurrent agents?
- How should results be aggregated from multiple agents?
- What happens if one agent fails while others succeed?
- Should there be a maximum concurrency limit?

## Design Decisions

### Work Distribution

Each concurrent agent receives the **same prompt and input** but operates in its own isolated workspace (worktree or directory). This is a "replicated worker" pattern, not a "partitioned work" pattern. The existing `MatrixStrategy` already handles the partitioned-work case (fan-out over distinct items). Step-level concurrency addresses the case where you want N identical agents working the same problem independently (e.g., generating multiple candidate solutions, running the same analysis with different temperatures, or simply parallelizing identical workloads).

### Result Aggregation

Results from all concurrent agents are collected into an array. The step result contains:
- `worker_results`: Array of individual worker outputs
- `worker_workspaces`: Array of workspace paths
- `total_workers`: Number of workers spawned
- `success_count` / `fail_count`: Success/failure tallies

This mirrors the existing `MatrixExecutor.aggregateResults` pattern.

### Partial Failure Semantics

Use **fail-fast** by default (matching the existing `errgroup` behavior in `MatrixExecutor`). When any concurrent agent fails, the context is cancelled and remaining agents are stopped. The step is marked as failed.

### Maximum Concurrency Limit

Respect the existing `runtime.max_concurrent_workers` from the manifest as a global cap. If a step's `concurrency` exceeds this limit, it is clamped to the runtime maximum. Default maximum: 10 if not configured.

### Workspace Isolation

Each concurrent agent gets its own isolated workspace. The comment on issue #29 confirms that git worktree workspaces (#76) solve the branch collision problem. Each concurrent agent workspace is created as `.wave/workspaces/<pipeline>/<step>/worker_<index>/`.

### Relationship to MatrixStrategy

`concurrency` and `strategy.matrix` are **mutually exclusive** on the same step. If both are specified, validation fails with a clear error. `MatrixStrategy.MaxConcurrency` controls parallelism *within* a matrix fan-out; `Step.Concurrency` controls how many identical agents run the same step.

## Related

- Wave pipeline execution system
- Agent subprocess management
- Issue #76: Git worktree workspaces for concurrent agent isolation
- Issue #64: Branch collision bug (fixed by worktree workspaces)
