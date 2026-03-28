# ADR-007: Formalize Boundary Between Composition Primitives and Graph Execution

## Status

Accepted

## Context

Wave has two overlapping flow control systems:

**Composition primitives** (pre-epic #589): `iterate`, `branch`, `loop`, `aggregate`, `sub_pipeline`. These are pipeline-level orchestration — they compose entire pipelines or fan out across items.

**Graph execution** (epic #589): `type: conditional`, `type: command`, `edges`, `max_visits`. These are step-level routing — they control flow within a single pipeline's step DAG.

Both systems can express conditional routing and looping, creating confusion about which to use.

## Decision

**Keep both systems and formalize the boundary.**

### Composition = Pipeline-Level Orchestration

Composition primitives operate on **entire pipelines or step groups**:

- `iterate` — parallel fan-out over a list of items, each running a sub-pipeline
- `branch` — route to different pipelines based on a context variable
- `loop` — repeat a pipeline until a condition is met
- `aggregate` — merge results from fan-out
- `sub_pipeline` — invoke a child pipeline

**When to use**: When you need to orchestrate multiple pipeline runs, fan out across items, or compose reusable pipeline units.

### Graph = Step-Level Routing

Graph execution controls **flow between steps within one pipeline**:

- `type: conditional` — route to different next steps based on outcome
- `type: command` — execute shell scripts as steps
- `edges` — define step transitions (including backward edges for loops)
- `max_visits` — loop safety limit

**When to use**: When you need conditional step skipping, retry-then-review loops within a pipeline, or shell command steps alongside agent steps.

### The Boundary

| Concern | Composition | Graph |
|---------|-------------|-------|
| **Scope** | Pipeline-of-pipelines | Steps within one pipeline |
| **Parallelism** | `iterate` with `mode: parallel` | Step concurrency via dependencies |
| **Conditionals** | `branch` routes to different pipelines | `type: conditional` routes to different steps |
| **Loops** | `loop` repeats entire pipelines | Backward `edges` repeat steps |
| **Shell execution** | Not supported | `type: command` |
| **Fan-out/merge** | `iterate` + `aggregate` | Not supported |

### Migration: None Required

Both systems are stable and in active use:
- 6 composition pipelines: audit-quality-loop, ops-epic-runner, impl-review-loop, impl-smart-route, ops-implement-epic, ops-parallel-audit
- 6+ graph pipelines: impl-hotfix, wave-bugfix, ops-debug, test-gen, wave-validate, wave-test-hardening

No pipeline needs to migrate. Authors choose the model that fits their use case.

## Consequences

- Pipeline authors learn one model at a time (most start with graph, use composition for advanced orchestration)
- Validator handles both (already does — `isCompositionStep()` in validate.go)
- WebUI renders both (DAG view for graph, compose indicators for composition steps)
- New features are implemented in the appropriate layer, not both
- Executor maintains two code paths (`executeStep` for DAG, `executeGraphPipeline` for graph) — this is acceptable complexity for the clarity gained

## Alternatives Considered

### Unify to Graph
Could replace `branch` with `type: conditional` and `loop` with backward edges. But `iterate` (parallel fan-out) and `aggregate` have no graph equivalent. Would require adding parallel fan-out to the graph walker — significant complexity for a lateral move.

### Unify to Composition
Could replace graph conditionals with `branch` and backward edges with `loop`. But composition primitives are higher-level and can't express arbitrary DAG shapes. Would lose the expressiveness of conditional edges within a single pipeline.
