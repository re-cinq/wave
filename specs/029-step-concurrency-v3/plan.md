# Implementation Plan: Step Concurrency

## Objective

Add a `concurrency` field to pipeline step definitions so that Wave can spawn N identical agent instances in parallel for the same step, each with its own isolated workspace, and aggregate their results.

## Approach

The implementation follows the existing `MatrixExecutor` pattern — which already handles fan-out parallelism for matrix strategies — but simplifies it for the case of N identical agents (no item-based fan-out, no input templates). A new `ConcurrencyExecutor` will wrap the same `runStepExecution` call in an `errgroup` pool, create per-agent workspaces, validate contracts independently, and aggregate results.

The key insight is that `MatrixExecutor` already solves most of the hard problems: workspace isolation, result aggregation, and concurrent execution with `errgroup`. The concurrency feature reuses the same structural patterns but with simpler semantics (identical agents, not item-per-agent).

## File Mapping

### New Files
| Path | Purpose |
|------|---------|
| `internal/pipeline/concurrency.go` | `ConcurrencyExecutor` — orchestrates N identical agent instances |
| `internal/pipeline/concurrency_test.go` | Unit and race-condition tests |

### Modified Files
| Path | Action | Changes |
|------|--------|---------|
| `internal/pipeline/types.go` | modify | Add `Concurrency int` field to `Step` struct |
| `internal/pipeline/executor.go` | modify | Route to `ConcurrencyExecutor` when `step.Concurrency > 1` in `executeStep()` |
| `internal/manifest/types.go` | modify | Add `MaxConcurrency int` to `Runtime` struct for global limit |
| `internal/pipeline/dag.go` | modify | Validate `concurrency` field during DAG validation (must be >= 0, <= max) |

### No Changes Needed
| Path | Reason |
|------|--------|
| `internal/state/store.go` | Per-agent state tracking can use existing `SaveStepState` with composite IDs (e.g., `step_id/agent_0`) — no schema changes needed |
| `internal/workspace/workspace.go` | WorkspaceManager already supports creating workspaces with arbitrary step IDs — the concurrency executor uses `step_id/agent_N` paths |
| `internal/contract/` | Contract validation is already per-invocation — each agent gets validated independently via existing `Validate()` calls |
| `internal/adapter/` | Adapter interface is unchanged — each concurrent agent is a separate `Run()` call |

## Architecture Decisions

### 1. ConcurrencyExecutor as a separate type (like MatrixExecutor)
**Decision**: Create `ConcurrencyExecutor` mirroring the `MatrixExecutor` pattern.
**Rationale**: The matrix executor is already proven, well-tested, and handles the same core concerns. Reusing its architectural pattern keeps the codebase consistent.

### 2. Route in `executeStep()` before matrix check
**Decision**: Check `step.Concurrency > 1` in `executeStep()` alongside the existing `step.Strategy != nil && step.Strategy.Type == "matrix"` check.
**Rationale**: Concurrency and matrix are mutually exclusive — a step can use one or the other. If both are set, `concurrency` takes precedence (documented).

### 3. Concurrency field defaults to 0 (disabled)
**Decision**: `Concurrency: 0` or `Concurrency: 1` means single-agent execution (backward compatible). Only values >= 2 trigger the concurrency executor.
**Rationale**: Zero-value Go behavior — all existing pipelines continue to work unchanged.

### 4. Global `max_concurrency` in `Runtime` (not per-pipeline)
**Decision**: Add `MaxConcurrency int` to `manifest.Runtime`. Default 10.
**Rationale**: The issue specifies a top-level manifest field. Per-pipeline limits can be added later without breaking changes.

### 5. Fail-fast semantics via errgroup
**Decision**: Use `errgroup.WithContext()` so the first failure cancels remaining agents.
**Rationale**: The issue explicitly specifies fail-fast behavior. `errgroup` provides exactly this via context cancellation.

### 6. Result aggregation format
**Decision**: Aggregate results into `{ "agent_results": [...], "agent_workspaces": [...], "total_agents": N, "success_count": N, "fail_count": N }` — mirroring the matrix executor's `worker_results` pattern.
**Rationale**: Consistency with existing aggregation format. Downstream steps can use the same patterns to consume results.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Race conditions in workspace path registration | Concurrent workspace writes could corrupt `execution.WorkspacePaths` map | Use `execution.mu` lock (already exists) for all map writes |
| Resource exhaustion from high concurrency | Too many concurrent agents could exhaust memory or API rate limits | Enforce global `MaxConcurrency` cap (default 10), validate at parse time |
| Artifact conflict between agents | Multiple agents writing the same output artifact | Each agent writes to its own workspace; aggregation is explicit |
| Interaction with retry logic | Retrying a concurrent step must re-run all agents | The existing retry loop in `executeStep()` already wraps the dispatch call — retrying calls `executeConcurrentStep()` again with fresh agents |
| Interaction with matrix strategy | User sets both `concurrency` and `strategy.type: matrix` | Document that they are mutually exclusive; validate in DAG validation |

## Testing Strategy

### Unit Tests (`concurrency_test.go`)
1. **Basic execution**: `concurrency: 3` spawns 3 agents with isolated workspaces
2. **Fail-fast**: One agent fails → all others cancelled, step fails
3. **Max concurrency cap**: `concurrency: 20` with `max_concurrency: 5` → capped at 5
4. **Concurrency = 0/1**: Falls through to normal single-agent execution
5. **Result aggregation**: Verify merged artifact format
6. **Workspace isolation**: Each agent gets unique workspace path

### Race Condition Tests
7. **Race detector**: All tests must pass with `-race` flag — critical for goroutine coordination

### Integration Tests
8. **DAG validation**: Step with both `concurrency` and `strategy` set → validation error
9. **YAML parsing**: Verify `concurrency` field round-trips through YAML marshal/unmarshal
