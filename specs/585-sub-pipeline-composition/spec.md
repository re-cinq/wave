# feat: sub-pipeline composition for workflow nesting

**Issue**: https://github.com/re-cinq/wave/issues/585
**Labels**: enhancement
**Author**: nextlevelshit
**Complexity**: complex

## Context

Fabro supports **sub-workflows** via `house` shape nodes — a parent workflow can invoke a child workflow with bidirectional context flow. The child gets a clone of parent context, executes independently, and modifications merge back via diff. The parent can configure max cycles, poll intervals, and stop conditions.

Wave's `meta.go` supports dynamic pipeline generation but not runtime composition — you can't have a pipeline step that invokes another pipeline.

## Design

### Nested Pipeline Steps

```yaml
steps:
  - name: plan
    persona: navigator

  - name: implement-and-test
    type: pipeline
    pipeline: implement-test-loop    # reference another pipeline
    depends_on: [plan]
    config:
      max_cycles: 50                  # max iterations in child
      timeout: 3600s                  # hard timeout
      stop_condition: "context.tests_pass=true"
    artifacts:
      inject: [plan]                  # pass parent artifacts to child
      extract: [implementation]        # pull child artifacts back to parent
```

### Context Flow

1. **Parent -> Child**: Artifacts listed in `inject` are copied into child's `.wave/artifacts/`
2. **Child -> Parent**: Artifacts listed in `extract` are copied back after child completes
3. **Context variables**: Child's context updates merge into parent context

### Use Cases

- **implement-issue** delegates to **implement-test-loop** for the implement->test->fix cycle
- **speckit-flow** delegates to **implement** for the actual coding step
- **audit-dual** runs **audit-security** and **audit-dx** as child pipelines

### Workspace Handling

Child pipeline gets its own workspace (worktree) by default. Can share parent workspace via:

```yaml
  - name: implement-and-test
    type: pipeline
    pipeline: implement-test-loop
    workspace:
      ref: parent                     # share parent's workspace
```

## Implementation Scope

1. Add `type: pipeline` step type to manifest schema
2. Sub-pipeline executor — launches child pipeline within parent run
3. Artifact injection/extraction between parent and child
4. Context merging
5. Lifecycle management (timeout, stop condition, max cycles)
6. State tracking — child run is a sub-entry in parent run state

## Acceptance Criteria

- [ ] A step with `type: pipeline` and `pipeline: <name>` loads and executes the referenced child pipeline
- [ ] Artifacts listed in `artifacts.inject` are copied into the child pipeline's workspace before execution
- [ ] Artifacts listed in `artifacts.extract` are copied back to the parent's execution state after child completion
- [ ] `config.timeout` enforces a hard timeout on child pipeline execution
- [ ] `config.max_cycles` limits child pipeline iterations (when child has a loop)
- [ ] `config.stop_condition` evaluates a template expression for early termination
- [ ] `workspace.ref: parent` shares the parent step's workspace with the child
- [ ] Child pipeline state is tracked as a sub-entry under the parent run
- [ ] Child pipeline failures propagate correctly (respecting parent step's `optional` flag)
- [ ] Context variables from child execution merge back into parent context
- [ ] Existing sub-pipeline execution via `SubPipeline` field continues to work unchanged
- [ ] All new functionality has unit tests with >80% coverage

## Research Sources

- Fabro sub-workflows: `house` shape with `stack.child_workflow`, `manager.max_cycles`, `manager.stop_condition`
- Fabro context flow: bidirectional context clone + diff merge
