# feat: sub-pipeline composition for workflow nesting

**Issue**: [re-cinq/wave#585](https://github.com/re-cinq/wave/issues/585)
**Labels**: enhancement
**Author**: nextlevelshit
**Complexity**: complex

## Context

Fabro supports **sub-workflows** via `house` shape nodes -- a parent workflow can invoke a child workflow with bidirectional context flow. The child gets a clone of parent context, executes independently, and modifications merge back via diff. The parent can configure max cycles, poll intervals, and stop conditions.

Wave's `meta.go` supports dynamic pipeline generation but not runtime composition -- you can't have a pipeline step that invokes another pipeline with full bidirectional artifact/context flow.

### Current Codebase State

Wave already has partial sub-pipeline infrastructure:
- **`Step.SubPipeline`** field (`types.go:293`) with YAML tag `pipeline`
- **`Step.SubInput`** field (`types.go:294`) for child input templating
- **`IsCompositionStep()`** (`types.go:484`) checks for SubPipeline, Iterate, Branch, Gate, Loop, Aggregate
- **`executeCompositionStep()`** in `executor.go:3951` -- loads child pipeline from `.wave/pipelines/`, creates child executor, executes, marks step complete/failed
- **`CompositionExecutor.executeSubPipeline()`** in `composition.go:438` -- delegates to `runSubPipeline()` which uses `SequenceExecutor`
- **`runSubPipeline()`** in `composition.go:447` -- loads pipeline, executes via SequenceExecutor, stores terminal step output in template context

**What's missing**:
1. No artifact inject config (parent -> child artifact copying)
2. No artifact extract config (child -> parent artifact copying)
3. No context variable merging between parent and child
4. No lifecycle management (timeout, stop_condition, max_cycles)
5. No state nesting (child run not linked to parent)
6. No workspace sharing (`ref: parent`)

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

1. Add `SubPipelineConfig` struct with lifecycle and artifact fields
2. Enhance executor's `executeCompositionStep()` with artifact inject/extract, lifecycle, state nesting
3. Add `MergeFrom()` to `PipelineContext` for bidirectional context flow
4. Add parent-child linkage to state store (nullable `parent_run_id`, `parent_step_id` columns)
5. Add circular composition detection to DAG validation
6. Preserve backward compatibility with existing `SubPipeline` usage

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
- [ ] Circular sub-pipeline references are detected and rejected at validation time
- [ ] All new functionality has unit tests with >80% coverage

## Research Sources

- Fabro sub-workflows: `house` shape with `stack.child_workflow`, `manager.max_cycles`, `manager.stop_condition`
- Fabro context flow: bidirectional context clone + diff merge
