# Implementation Plan: Sub-Pipeline Composition

## Objective

Add a `SubPipelineConfig` struct to the `Step` type that enables pipeline steps to invoke child pipelines with bidirectional artifact flow, lifecycle management (timeout, max_cycles, stop_condition), context merging, and parent-child state tracking. This builds on the existing `SubPipeline` field and `executeCompositionStep()` infrastructure.

## Approach

Wave already has basic sub-pipeline execution:
- `Step.SubPipeline` (types.go:293) references a child pipeline by name
- `executeCompositionStep()` (executor.go:3951) loads the child pipeline, creates a child executor, and runs it
- `CompositionExecutor.executeSubPipeline()` (composition.go:438) delegates to `runSubPipeline()` via SequenceExecutor

The plan extends this infrastructure by:
1. Adding a `SubPipelineConfig` struct for lifecycle/artifact configuration
2. Enhancing `executeCompositionStep()` to inject parent artifacts before child execution and extract child artifacts after
3. Adding `MergeFrom()` to `PipelineContext` for child-to-parent context variable propagation
4. Adding `parent_run_id`/`parent_step_id` to the state store for hierarchical run tracking
5. Adding circular composition detection to prevent A->B->A deadlocks
6. Preserving backward compatibility -- when `Config` is nil, existing behavior is unchanged

## File Mapping

### New Files
| Path | Purpose |
|------|---------|
| `internal/pipeline/subpipeline.go` | `SubPipelineConfig` type, artifact inject/extract logic, context merge, lifecycle enforcement, circular reference detection |
| `internal/pipeline/subpipeline_test.go` | Unit tests for all sub-pipeline composition functionality |

### Modified Files
| Path | Change | Lines Affected |
|------|--------|----------------|
| `internal/pipeline/types.go` | Add `SubPipelineConfig` struct; add `Config *SubPipelineConfig` field to `Step` | ~293-300 (near SubPipeline field) |
| `internal/pipeline/executor.go` | Enhance `executeCompositionStep()`: artifact inject before child execution, artifact extract + context merge after, lifecycle timeout via `context.WithTimeout()`, parent-child state linkage | ~3951-4048 |
| `internal/pipeline/context.go` | Add `MergeFrom(child *PipelineContext, namespace string)` method | New method |
| `internal/pipeline/composition.go` | Update `executeSubPipeline()` and `runSubPipeline()` to use `SubPipelineConfig` when present (timeout enforcement, artifact passing) | ~438-478 |
| `internal/state/types.go` | Add `ParentRunID`, `ParentStepID` fields to `RunRecord` | ~6-20 |
| `internal/state/store.go` | Add `SetParentRun(childRunID, parentRunID, stepID)` and `GetChildRuns(parentRunID)` to `StateStore` interface | Interface definition |
| `internal/state/sqlite.go` | Implement parent-child linkage: ALTER TABLE runs, new methods | Schema + methods |
| `internal/pipeline/dag.go` | Add circular sub-pipeline reference detection during DAG validation | New validation function |
| `internal/pipeline/dryrun.go` | Validate sub-pipeline config fields during dry run | Add validation case |

## Architecture Decisions

### 1. Extend Step struct with Config field
**Decision**: Add `Config *SubPipelineConfig` as a new field on `Step` alongside the existing `SubPipeline` field.

**Rationale**: Matches existing composition primitives (Gate, Loop, Branch all hang off Step). When `Config` is nil, existing behavior is preserved. The `IsCompositionStep()` check already works for `SubPipeline != ""`.

### 2. Separate artifact config from MemoryConfig
**Decision**: `artifacts.inject` and `artifacts.extract` live in `SubPipelineConfig`, not in `MemoryConfig.InjectArtifacts`.

**Rationale**: `InjectArtifacts` is for step-to-step artifact injection within a pipeline. Sub-pipeline artifact flow is between parent and child *pipeline executions* -- conceptually different. Extract (child->parent) has no analog in `MemoryConfig`.

### 3. Last-writer-wins context merge
**Decision**: Child context variables overwrite parent on key collision. Artifact paths from child are namespaced with child pipeline name to prevent collisions.

**Rationale**: Simple, predictable, and sufficient. Wave's `PipelineContext` is a flat key-value map. Full diff-merge semantics (Fabro-style) would add complexity without clear benefit.

### 4. State nesting via parent_run_id
**Decision**: Add nullable `parent_run_id` and `parent_step_id` columns to the runs table. Child runs link back to parent.

**Rationale**: Minimal schema change. Supports `wave logs <parent-run>` showing child run status. No migration needed -- new nullable columns with default NULL.

### 5. Workspace sharing via ref: parent
**Decision**: `workspace.ref: parent` on a sub-pipeline step passes the parent step's resolved workspace path to the child executor.

**Rationale**: Reuses existing `WorkspaceConfig.Ref` semantics. The child executor receives the path as an option -- no new abstraction needed.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking existing `SubPipeline` usage | Medium | High | All changes additive -- existing behavior preserved when `Config` is nil. Run full test suite. |
| Artifact path collisions parent/child | Low | Medium | Namespace extracted artifacts with child pipeline name prefix |
| Circular sub-pipeline references | Medium | High | Cycle detection in DAG validation -- build graph of pipeline->sub-pipeline references, detect cycles |
| Child timeout not enforced at adapter level | Low | Medium | Use `context.WithTimeout()` on child executor context -- propagates to adapter calls |
| State store schema migration | Low | Low | Nullable columns with default NULL -- backward compatible, no migration for existing data |
| Concurrent access to PipelineContext during merge | Low | Medium | `PipelineContext` already uses `sync.Mutex` -- `MergeFrom()` will acquire lock |

## Testing Strategy

### Unit Tests (`subpipeline_test.go`)
- `TestSubPipelineConfig_Validate` -- invalid timeout format, missing pipeline name, valid configs
- `TestArtifactInject` -- parent artifacts copied into child workspace correctly
- `TestArtifactExtract` -- child artifacts registered in parent execution state
- `TestContextMerge` -- child context variables merge into parent, namespace isolation
- `TestLifecycleTimeout` -- child execution cancelled on timeout expiry
- `TestLifecycleMaxCycles` -- max_cycles propagated to child loop configuration
- `TestStopCondition` -- stop condition template evaluation against child context
- `TestWorkspaceSharing` -- `workspace.ref: parent` passes correct path
- `TestCircularPipelineDetection` -- validation catches A->B->A cycles

### Integration Tests (in `composition_test.go`)
- `TestCompositionExecutor_SubPipelineWithArtifacts` -- end-to-end artifact flow
- `TestCompositionExecutor_SubPipelineWithTimeout` -- timeout enforcement
- `TestCompositionExecutor_SubPipelineStateNesting` -- parent-child run linkage

### Backward Compatibility
- All existing `TestCompositionExecutor_SubPipeline*` tests must continue passing
- Existing pipelines with bare `SubPipeline` field (no `Config`) work identically
