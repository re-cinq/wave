# Implementation Plan: Sub-Pipeline Composition

## Objective

Add a `type: pipeline` step type that allows pipelines to invoke child pipelines with bidirectional artifact flow, lifecycle management (timeout, max_cycles, stop_condition), and parent-child state tracking. This enables workflow nesting — e.g., implement-issue delegating to implement-test-loop for the coding cycle.

## Approach

Wave already has basic sub-pipeline execution via `Step.SubPipeline` and `executeCompositionStep()` in `executor.go`. The existing implementation loads a child pipeline by name and runs it with a fresh child executor, but lacks:

1. **Artifact inject/extract** — no bidirectional artifact flow between parent and child
2. **Lifecycle config** — no timeout, max_cycles, or stop_condition enforcement
3. **State nesting** — child run is independent, not linked to parent
4. **Context merging** — child context changes don't flow back to parent
5. **Workspace sharing** — no `workspace.ref: parent` support for child pipelines

The plan extends the existing infrastructure rather than replacing it. We add a `SubPipelineConfig` struct to the `Step` type, enhance `executeCompositionStep()` to handle the new config, and add parent-child linkage to the state store.

## File Mapping

### New Files
| Path | Purpose |
|------|---------|
| `internal/pipeline/subpipeline.go` | SubPipelineConfig type, artifact inject/extract logic, context merge, lifecycle enforcement |
| `internal/pipeline/subpipeline_test.go` | Unit tests for sub-pipeline composition |

### Modified Files
| Path | Change |
|------|--------|
| `internal/pipeline/types.go` | Add `SubPipelineConfig` struct, `Config` field to `Step` |
| `internal/pipeline/executor.go` | Enhance `executeCompositionStep()` with artifact flow, lifecycle, state nesting |
| `internal/pipeline/composition.go` | Update `CompositionExecutor.executeSubPipeline()` to use new config |
| `internal/pipeline/composition_test.go` | Add tests for new composition features |
| `internal/pipeline/context.go` | Add `MergeFrom()` method for child->parent context merge |
| `internal/pipeline/dryrun.go` | Validate new `config` and `artifacts` fields during dry run |
| `internal/state/store.go` | Add `SetParentRun(childRunID, parentRunID, stepID)` and `GetChildRuns(parentRunID)` methods |
| `internal/state/sqlite.go` | Implement parent-child run linkage in SQLite |
| `internal/state/types.go` | Add `ParentRunID`, `ParentStepID` fields to `RunRecord` |
| `internal/pipeline/validation.go` | Validate sub-pipeline config fields |

## Architecture Decisions

### 1. Extend Step struct vs. new step type
**Decision**: Add `SubPipelineConfig` as a new field on `Step` rather than introducing a separate step type. The existing `SubPipeline` field already establishes the pattern. The new `Config` field holds lifecycle/artifact configuration when the step is a pipeline step.

**Rationale**: Matches existing composition primitives (Gate, Loop, Branch all hang off Step). Avoids a parallel type hierarchy. The `IsCompositionStep()` check already works.

### 2. Artifact inject/extract via config vs. reusing MemoryConfig
**Decision**: Add `artifacts.inject` and `artifacts.extract` as dedicated fields in `SubPipelineConfig` rather than overloading `MemoryConfig.InjectArtifacts`.

**Rationale**: `InjectArtifacts` injects from parent step artifacts into the step's workspace. Sub-pipeline artifact flow is between parent and child *pipeline* executions — conceptually different. Extract (child->parent) has no analog in `MemoryConfig` at all.

### 3. Context merge strategy
**Decision**: Use a key-based merge where child context variables overwrite parent on conflict (last-writer-wins). Template-resolved artifact paths are namespaced by child pipeline name to prevent collisions.

**Rationale**: Simple and predictable. The Fabro model uses diff-merge, but Wave's `PipelineContext` is a flat key-value map — full diff semantics aren't needed.

### 4. State nesting
**Decision**: Add `parent_run_id` and `parent_step_id` columns to the runs table. Child runs link back to parent, enabling hierarchical state queries.

**Rationale**: Minimal schema change. Supports `wave logs <parent-run>` showing child run status inline.

### 5. Workspace sharing
**Decision**: `workspace.ref: parent` on a sub-pipeline step makes the child pipeline's steps inherit the parent step's workspace path. Implemented by passing the workspace path down to the child executor.

**Rationale**: Reuses existing `WorkspaceConfig.Ref` semantics. The child executor just needs the resolved path — no new abstraction needed.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking existing `SubPipeline` usage | Medium | High | All changes are additive — existing `SubPipeline` field and `executeCompositionStep()` behavior preserved when `Config` is nil |
| Artifact path collisions between parent/child | Low | Medium | Namespace extracted artifacts with child pipeline name prefix |
| Circular sub-pipeline references | Medium | High | Add cycle detection during validation — check if pipeline A references B which references A |
| Child timeout not enforced on adapter-level | Low | Medium | Use `context.WithTimeout()` on the child executor's context |
| State store schema migration | Low | Low | Add nullable `parent_run_id` column — no migration needed for existing data |

## Testing Strategy

### Unit Tests (`subpipeline_test.go`)
- `TestSubPipelineConfig_Validate` — config validation (missing fields, invalid timeout format)
- `TestArtifactInject` — parent artifacts copied into child workspace
- `TestArtifactExtract` — child artifacts copied back to parent execution
- `TestContextMerge` — child context variables merge into parent
- `TestLifecycleTimeout` — child pipeline cancelled on timeout
- `TestLifecycleMaxCycles` — max_cycles propagated to child loop config
- `TestStopCondition` — stop condition evaluated against child context
- `TestWorkspaceSharing` — `workspace.ref: parent` passes parent workspace to child
- `TestCircularPipelineDetection` — validation catches A->B->A references

### Integration Tests (`composition_test.go`)
- `TestCompositionExecutor_SubPipelineWithArtifacts` — end-to-end artifact flow
- `TestCompositionExecutor_SubPipelineWithTimeout` — timeout enforcement
- `TestCompositionExecutor_SubPipelineStateNesting` — parent-child state linkage

### Existing Test Preservation
- All existing `TestCompositionExecutor_SubPipeline` tests must continue to pass
- Dry run tests for sub-pipeline validation must be updated
