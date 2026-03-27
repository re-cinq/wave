# feat: graph loops and conditional edge routing for implement-test-fix cycles

**Issue**: https://github.com/re-cinq/wave/issues/577
**Labels**: enhancement
**Author**: nextlevelshit
**Complexity**: complex

## Context

Wave's current pipeline executor uses a strict DAG (directed acyclic graph) model with topological sort. Steps can only depend on prior steps, and `on_failure` actions are limited to `fail`/`skip`/`continue`/`rework`/`retry`. This cannot express cycles like implement-test-fix loops.

The goal is to extend Wave's executor to support directed graphs with backward edges, conditional routing, and command steps while preserving contract validation, persona isolation, and fresh memory guarantees.

## Design Goals

Combine graph expressiveness (loops via backward edges with `max_visits` safety, conditional edge routing where agents influence routing via response directives) with Wave's existing contract validation and persona isolation.

### Conditional Edges

Steps can route based on outcomes and context:

```yaml
steps:
  - name: gate
    type: conditional
    depends_on: [test]
    edges:
      - target: finalize
        condition: "outcome=success"
      - target: fix
        # unconditional fallback
```

### Loop Support

Steps can reference earlier steps as targets, creating cycles:

```yaml
  - name: fix
    depends_on: [gate]
    max_visits: 5
    edges:
      - target: test    # backward edge -- creates loop
```

The executor handles cycles by:
1. Replacing topological sort with a graph walker that follows edges
2. Tracking visit counts per step
3. Enforcing `max_visits` limits (default: 10, graph-level `max_step_visits`)

### Command Steps

New step type for shell script execution:

```yaml
  - name: validate
    type: command
    script: "go test ./... 2>&1 || true"
    depends_on: [implement]
```

Command steps run scripts and capture stdout/stderr into context without using adapters.

### Context-Based Routing

Conditions can reference context values:

```yaml
edges:
  - target: exit
    condition: "context.tests_passed=true"
  - target: fix
    condition: "context.tests_passed=false"
```

## What Wave Keeps

- Contract validation at step boundaries, even in loops
- Fresh memory -- each loop iteration starts fresh
- Persona isolation -- fix step uses persona with appropriate permissions
- YAML pipelines -- conditions are YAML-native

## What Wave Gains

- Implement-test-fix loops natively supported
- Agent-influenced routing -- steps can set `preferred_next` in output
- Conditional branching based on outcomes without hardcoded `on_failure`
- Command steps -- lightweight script execution without LLM

## Safety Mechanisms

- `max_visits` per step (default 10)
- `max_step_visits` graph-level limit (default 50)
- Circuit breaker: if same failure signature repeats 3x, terminate

## Acceptance Criteria

1. Existing DAG pipelines (no edges defined) work unchanged -- backward compatible
2. Steps with `edges` field route to named targets based on conditions
3. Backward edges (target referencing earlier step) create loops
4. `max_visits` enforced per step; exceeded limit terminates with error
5. `type: conditional` steps evaluate conditions without adapter execution
6. `type: command` steps execute shell scripts and capture output
7. Condition expressions support `outcome=success/failure` and `context.KEY=VALUE`
8. Visit counts persisted in state store for resume support
9. Contract validation runs at each step boundary including loop iterations
10. All existing tests pass unchanged
