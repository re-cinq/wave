# Implementation Plan: Step Concurrency

## Objective

Add a `concurrency` field to pipeline steps that causes the executor to spawn N parallel adapter invocations for the same step, each in an isolated workspace, with fail-fast semantics and merged artifact output.

## Approach

The implementation threads a new `Concurrency int` field through the pipeline types, adds a parallel execution path in the executor, and collects/merges outputs from all agents. The design follows the existing `MatrixExecutor` pattern (matrix.go) which already solves workspace isolation, errgroup-based parallelism, and result aggregation.

### Execution Flow

1. `executeStep()` detects `step.Concurrency > 1`
2. Delegates to a new `executeConcurrentStep()` method
3. `executeConcurrentStep()` creates N workspaces, injects artifacts into each, spawns N adapter invocations via `errgroup.Group` with `SetLimit(N)`, and collects results
4. Each agent's output artifacts are indexed as `<artifact_name>_<agent_index>` (e.g., `result_0`, `result_1`)
5. A merged summary artifact is also written for downstream consumption
6. Fail-fast: `errgroup` cancels remaining goroutines on first error

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | modify | Add `Concurrency int` field to `Step` struct |
| `internal/pipeline/executor.go` | modify | Add `executeConcurrentStep()` method, wire from `executeStep()` |
| `internal/pipeline/concurrent.go` | create | New file for `ConcurrentExecutor` struct with parallel spawn, workspace creation, result aggregation logic |
| `internal/pipeline/concurrent_test.go` | create | Unit and race tests for concurrent execution |
| `internal/pipeline/executor_test.go` | modify | Add integration test for `concurrency` field flowing through executor |
| `internal/pipeline/types_test.go` | modify | Add test for `EffectiveConcurrency()` method |
| `internal/manifest/types.go` | modify | Add `MaxStepConcurrency int` to `Runtime` struct for global cap |
| `.wave/schemas/wave-pipeline.schema.json` | modify | Add `concurrency` to step properties |
| `.wave/schemas/wave-manifest.schema.json` | modify | Add `max_step_concurrency` to runtime |
| `docs/reference/manifest-schema.md` | modify | Document `concurrency` field |

## Architecture Decisions

### 1. New `Concurrency` Field (not reuse `MaxConcurrentAgents`)

`MaxConcurrentAgents` controls the agent-internal subagent hint (#112). `Concurrency` controls executor-level parallelism. These are orthogonal: a step with `concurrency: 3, max_concurrent_agents: 5` spawns 3 parallel adapter processes, each allowed to use 5 subagents. Adding a separate field avoids semantic overloading.

### 2. Separate `concurrent.go` File

Following the pattern of `matrix.go` for matrix execution, concurrent execution logic goes in its own file. This keeps `executor.go` clean and makes the concurrent executor independently testable.

### 3. Workspace Isolation Strategy

Each concurrent agent gets a workspace created by appending `_agent_<N>` to the step workspace path:
```
.wave/workspaces/<pipeline-id>/<step-id>_agent_0/
.wave/workspaces/<pipeline-id>/<step-id>_agent_1/
.wave/workspaces/<pipeline-id>/<step-id>_agent_2/
```

For worktree workspaces, each agent gets a unique branch: `<branch>-agent-0`, `<branch>-agent-1`, etc. This leverages the existing worktree infrastructure (#76).

### 4. Artifact Merging Strategy

Each agent writes to `<artifact_name>` in its own workspace. After all agents complete, the concurrent executor:
1. Reads each agent's output artifacts
2. For JSON artifacts: wraps all outputs in an array `[agent_0_output, agent_1_output, ...]`
3. For text/markdown: concatenates with agent index headers
4. Writes the merged artifact to the step's primary workspace path

### 5. Global Cap via `Runtime.MaxConcurrentWorkers`

Rather than adding a new top-level field, reuse the existing `Runtime.MaxConcurrentWorkers` as the global cap on step concurrency. If `step.Concurrency > manifest.Runtime.MaxConcurrentWorkers`, cap at the manifest value. Hard maximum of 10 regardless.

### 6. State Tracking

Per-agent state is tracked with suffixed step IDs: `<step_id>_agent_0`, `<step_id>_agent_1`. This integrates with the existing `StepStateRecord` without schema changes. The parent step ID tracks the aggregate state.

### 7. Retry Interaction

Retry applies to the entire concurrent step, not individual agents. If any agent fails, the whole batch is retried on the next attempt. This is simpler than per-agent retry and consistent with fail-fast semantics.

## Risks

| Risk | Mitigation |
|------|------------|
| Resource exhaustion with high concurrency | Hard cap at 10; respect `MaxConcurrentWorkers` from manifest |
| Workspace conflicts with shared worktrees | Each agent gets its own worktree branch (`-agent-N` suffix) |
| API rate limiting with many parallel Claude calls | Bounded by errgroup SetLimit; documented in concurrency guide |
| Artifact naming collisions | Indexed naming scheme (`_agent_N`) prevents collisions |
| Interaction with existing matrix/iterate | `concurrency` is mutually exclusive with `strategy` and `iterate` — validated at parse time |
| Retry complexity | Retry applies to whole concurrent batch, not individual agents |

## Testing Strategy

### Unit Tests (`concurrent_test.go`)
- Table-driven tests with mock adapter:
  - `concurrency: 1` behaves identically to non-concurrent path
  - `concurrency: 3` spawns 3 agents, collects 3 results
  - `concurrency: 15` capped at 10 (or manifest max)
  - One agent fails → entire step fails (fail-fast)
  - All agents succeed → merged artifacts written
  - Mutual exclusion with `strategy` field

### Race Tests
- `go test -race` on all concurrent tests
- Concurrent writes to `execution.ArtifactPaths` use mutex

### Integration Tests (`executor_test.go`)
- `configCapturingAdapter` verifies all N agents get correct config
- End-to-end: step with `concurrency: 2` produces merged artifact

### Validation Tests
- `concurrency: -1` → parse error
- `concurrency: 0` or `concurrency: 1` → normal single-agent execution
- `concurrency` + `strategy` → validation error (mutually exclusive)
