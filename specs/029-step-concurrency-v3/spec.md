# feat: Add concurrency property to spawn multiple agents for the same step

**Issue**: [#29](https://github.com/re-cinq/wave/issues/29)
**Labels**: enhancement, priority: medium
**Author**: nextlevelshit
**Complexity**: complex

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

## Design Decisions

### Work Distribution
Each concurrent agent receives the same step prompt and input artifacts. Work partitioning is the responsibility of the prompt author — Wave spawns N identical agents, not a fan-out/fan-in scheduler. This keeps the runtime simple and avoids introducing a task queue abstraction.

### Result Aggregation
All agent outputs are collected into an array of artifacts. The subsequent step receives the merged artifact set. No automatic deduplication or conflict resolution — downstream steps handle reconciliation.

### Partial Failure Handling
If any agent fails, the step is marked as failed (fail-fast). A future enhancement could add a `concurrency_policy: best-effort` option, but the default must be strict to avoid silent data loss.

### Maximum Concurrency Limit
Default max is `10` (configurable via `wave.yaml` top-level `max_concurrency`). This prevents resource exhaustion from misconfiguration.

### Concurrency Isolation
Per comment on the issue: The concurrency isolation concern is addressed by #76 — git worktree workspaces give each concurrent agent its own isolated git checkout, eliminating branch collision.

## Implementation Notes

- **Executor changes**: `internal/pipeline/executor.go` — the `executeStep()` function currently runs a single subprocess. Needs to be wrapped in a goroutine pool (e.g., `errgroup.Group`) that spawns N agents.
- **Workspace isolation**: Each concurrent agent must get its own ephemeral workspace via `internal/workspace/`. Workspaces are created in parallel.
- **State tracking**: `internal/state/` needs to track per-agent status within a step (e.g., `step_29_agent_0`, `step_29_agent_1`).
- **Contract validation**: Each agent output is validated independently before merging.

## Acceptance Criteria

- [ ] `concurrency` property is recognized in step configuration (manifest parsing)
- [ ] Multiple agent instances are spawned via goroutine pool with `errgroup`
- [ ] Each agent gets an isolated workspace
- [ ] Results from concurrent agents are collected into a merged artifact set
- [ ] Fail-fast on any agent failure with clear error attribution
- [ ] Default max concurrency of 10 enforced
- [ ] Documentation updated with concurrency examples
- [ ] Race condition tests pass (`go test -race`)
