# Implementation Plan: Graph Loops and Conditional Edge Routing

## Objective

Extend Wave's pipeline executor from a strict DAG model to a directed graph walker that supports backward edges (loops), conditional edge routing, and command steps -- enabling native implement-test-fix cycles.

## Approach

**Incremental extension, not rewrite.** The core insight from codebase analysis:

1. **Wave already has composition-level primitives** (`LoopConfig`, `BranchConfig`, `GateConfig` in `types.go`) that operate at the `CompositionExecutor` level. The issue asks for **graph-level** routing at the `DefaultPipelineExecutor` level.

2. **The existing `ExecConfig` already defines `type: command`** but it's only validated in `dryrun.go` -- not yet executed in the main executor. We complete this.

3. **Backward compatibility**: Pipelines without `edges` fields continue using topological sort. The graph walker only activates when edges are present.

### Strategy: Dual-Mode Executor

- **DAG mode** (default): Existing topological sort + `findReadySteps()` loop. Zero changes for existing pipelines.
- **Graph mode**: Activated when any step defines `edges` or `type: conditional`. Uses a graph walker that follows edges, tracks visits, and evaluates conditions.

Detection is at pipeline load time: if any step has `Edges` or `Type == "conditional"`, the pipeline enters graph mode.

## File Mapping

### New Files

| Path | Purpose |
|------|---------|
| `internal/pipeline/graph.go` | `GraphWalker` -- edge-following executor with visit tracking |
| `internal/pipeline/graph_test.go` | Unit tests for graph walker |
| `internal/pipeline/condition.go` | Condition expression parser (`outcome=X`, `context.K=V`) |
| `internal/pipeline/condition_test.go` | Condition parser tests |

### Modified Files

| Path | Change |
|------|--------|
| `internal/pipeline/types.go` | Add `Edges`, `MaxVisits`, `Script`, `Type` fields to `Step`; add `EdgeConfig`, `ConditionExpr` types; add `MaxStepVisits` to `Pipeline` |
| `internal/pipeline/dag.go` | Add `ValidateGraph()` for graph-mode pipelines (allows cycles but validates safety limits); add `IsGraphPipeline()` detection |
| `internal/pipeline/executor.go` | Add `executeGraphPipeline()` entry point; add `executeCommandStep()` for script execution; modify `Execute()` to detect and dispatch graph mode |
| `internal/state/store.go` | Add `SaveStepVisitCount()` / `GetStepVisitCount()` to `StateStore` interface; add `VisitCount` field to `StepStateRecord` |
| `internal/state/sqlite.go` | Implement visit count persistence in SQLite (add column to step_state table) |
| `internal/pipeline/dryrun.go` | Add validation for `edges`, `max_visits`, `type: conditional`, `script` fields |

### Unchanged Files

All existing pipeline YAML files, composition executor, contract validation, adapter infrastructure -- no changes needed.

## Architecture Decisions

### 1. Graph Walker as Separate Code Path

**Decision**: Implement `GraphWalker` as a separate struct in `graph.go` rather than modifying the existing DAG execution loop.

**Rationale**: The DAG loop in `executor.go` (lines 498-581) is tightly coupled to the `completed` map pattern and topological ordering. Modifying it to handle cycles would require fundamental changes to `findReadySteps()`, `skipDependentSteps()`, and the completion counting logic. A separate walker is cleaner and doesn't risk regressions in existing DAG pipelines.

**Integration**: `Execute()` in `executor.go` checks `IsGraphPipeline()` and dispatches to `executeGraphPipeline()` which creates a `GraphWalker` and delegates.

### 2. Condition Expression Grammar

**Decision**: Simple key=value grammar with three namespaces:
- `outcome=success|failure` -- step execution result
- `context.KEY=VALUE` -- shared context from artifacts
- (unconditional) -- no condition = default/fallback edge

**Rationale**: The issue shows only these forms. A full expression language (operators, boolean logic) adds complexity without clear use cases. The grammar can be extended later.

**Parser**: `ParseCondition(expr string) -> ConditionExpr` returns a struct with `{Namespace, Key, Value}`. Evaluation is a simple equality check against the step's execution context.

### 3. Visit Count Storage

**Decision**: Add `visit_count` column to the existing `step_states` table rather than a new table.

**Rationale**: Visit count is inherently per-step-per-pipeline, which maps directly to the existing `step_states` primary key. A new table would require extra joins for resume logic.

### 4. Edge Evaluation Order

**Decision**: Edges are evaluated in YAML order. First matching condition wins. Last edge without condition serves as fallback.

**Rationale**: Matches the issue's examples where unconditional edges appear last as fallbacks. Order-dependent evaluation is intuitive for YAML authors.

### 5. Command Step Execution

**Decision**: Command steps use `os/exec` directly, capturing stdout+stderr. Output is stored as a step artifact and made available to downstream conditions.

**Rationale**: Command steps are explicitly "lightweight script execution without spinning up an LLM." Using the adapter infrastructure would be overkill and create unnecessary dependency.

### 6. Circuit Breaker

**Decision**: Track the last 3 error messages per step. If all 3 are identical (normalized), terminate the loop immediately regardless of `max_visits`.

**Rationale**: Prevents burning API tokens on an unrecoverable error. The normalization strips timestamps and line numbers to compare failure signatures.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Infinite loops from misconfigured `max_visits` | Medium | High | Default `max_visits=10`, graph-level `max_step_visits=50`, circuit breaker |
| Resume semantics for graph pipelines | Medium | Medium | Visit counts persisted in state store; resume reloads visit counts |
| Breaking existing DAG pipelines | Low | Critical | Dual-mode detection; graph mode only activates with `edges` field; comprehensive test coverage |
| Condition expression grammar too limited | Low | Low | Extensible parser design; can add operators later |
| Race conditions in concurrent step visit tracking | Low | Medium | Visit counts are per-step, updated atomically in state store |

## Testing Strategy

### Unit Tests
- **Condition parser**: All expression forms, edge cases, malformed input
- **Graph walker**: Linear graph, diamond graph, simple loop, nested loops, max_visits enforcement, circuit breaker
- **Edge evaluation**: Order-dependent matching, fallback edges, no matching edge (error)
- **Visit tracking**: Increment, max enforcement, resume with existing counts
- **Command step**: Stdout/stderr capture, exit code handling, timeout

### Integration Tests
- **Backward compatibility**: Run existing DAG pipelines through graph-mode detection -- must detect DAG mode
- **End-to-end loop**: implement -> test -> gate -> fix -> test cycle with mock adapter
- **Resume from graph pipeline**: Persist state, resume, verify visit counts restored
- **Mixed pipeline**: DAG steps + graph steps in same pipeline (if supported)

### Regression Tests
- All existing `go test ./...` must pass unchanged
- Existing pipeline YAML files must load and validate without errors
