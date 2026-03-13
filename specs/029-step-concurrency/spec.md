# feat: Add concurrency property to spawn multiple agents for the same step

**Issue**: [#29](https://github.com/re-cinq/wave/issues/29)
**Labels**: enhancement, priority: medium
**Author**: nextlevelshit
**State**: OPEN

## Summary

Add a `concurrency` property to pipeline step definitions that allows spawning multiple agent instances to process work in parallel. When `concurrency: N` is set on a step, the executor spawns N identical agent subprocesses — each with its own workspace — and collects their outputs into a merged artifact set.

## Background

Some pipeline steps could benefit from parallel execution, particularly when processing multiple items or performing independent operations. Currently, the executor runs one agent per step. The issue requests Wave-level parallelism where the **executor** spawns N agent processes, as distinct from issue #112 which only tells the agent it *may* spawn subagents internally.

Key distinction:
- **Issue #112** (DONE): `max_concurrent_agents` injects a concurrency hint into CLAUDE.md so the agent knows it can spawn subagents
- **Issue #29** (THIS): `concurrency` makes the executor itself fork N parallel adapter invocations for the same step

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
Default max is `10` (configurable via `wave.yaml` top-level `max_concurrency`). This prevents resource exhaustion from misconfiguration. The existing `Runtime.MaxConcurrentWorkers` field already exists on the manifest and can serve as the global cap.

### Relationship to MaxConcurrentAgents
`concurrency` (executor-level parallelism) is distinct from `max_concurrent_agents` (agent-internal subagent hint). Both can coexist on a step — concurrency controls how many adapter processes Wave spawns, while max_concurrent_agents controls how many subagents each spawned adapter may use.

## Existing Codebase State

- `Step.MaxConcurrentAgents` (types.go:143) — already exists for prompt hints (#112, done)
- `MatrixStrategy.MaxConcurrency` (types.go:269) — parallel matrix item processing (different mechanism)
- `IterateConfig.MaxConcurrent` (types.go:326) — iterate-mode parallelism
- `Runtime.MaxConcurrentWorkers` (manifest/types.go:73) — global worker limit
- `errgroup` is already imported in executor.go and used in sequence.go, composition.go, matrix.go
- `ConcurrencyValidator` (validation.go:289) — workspace-level lock manager (prevents duplicate pipeline runs)

### Comment from #76
> The concurrency isolation concern is addressed by #76 — git worktree workspaces give each concurrent agent its own isolated git checkout, eliminating branch collision.

## Acceptance Criteria

- [ ] `concurrency` property is recognized in step configuration (manifest parsing)
- [ ] Multiple agent instances are spawned via goroutine pool with `errgroup`
- [ ] Each agent gets an isolated workspace
- [ ] Results from concurrent agents are collected into a merged artifact set
- [ ] Fail-fast on any agent failure with clear error attribution
- [ ] Default max concurrency of 10 enforced
- [ ] Documentation updated with concurrency examples
- [ ] Race condition tests pass (`go test -race`)
