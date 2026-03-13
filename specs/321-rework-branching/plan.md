# Implementation Plan: Rework Branching (#321)

## 1. Objective

Implement `on_failure: rework` in the pipeline executor, allowing a step to redirect execution to an alternative "rework" step when all retry attempts are exhausted. Also enhance the failure context carried between retry attempts and into rework steps with richer metadata.

## 2. Approach

The feature is scoped to **single-step rework targets within the same pipeline**. This avoids the complexity of sub-pipeline rework paths while covering the primary use case: fallback strategies for failing steps.

The implementation follows the existing `on_failure` pattern in `executor.go`, adding `rework` as a fourth case alongside `fail`, `skip`, and `continue`. The rework step is a regular pipeline step that gets executed inline when the original step exhausts retries, with failure context injected via the existing `AttemptContexts` mechanism.

### Key Design Decisions

1. **Rework step is an existing step in the pipeline DAG**, not a dynamically-created step. Pipeline authors declare the rework target step normally and reference it via `rework_step`. This keeps DAG validation simple and reuses all existing step execution machinery.

2. **Rework step replaces the failed step for downstream dependency resolution.** After rework completes, its workspace path and artifact paths are registered under the original failed step's ID, so downstream steps see the rework output as if it came from the original step.

3. **`rework_step` field on `RetryConfig`**, not on `Step` directly, because rework is semantically tied to retry exhaustion. The field is only meaningful when `on_failure: rework`.

4. **Enhanced `AttemptContext` is additive** — new fields are added alongside existing ones. No breaking changes to the struct.

5. **Rework step is NOT re-executed if it was already completed** during a resumed pipeline run. The resume manager checks for rework completion in state.

## 3. File Mapping

### Modified Files

| File | Changes |
|------|---------|
| `internal/pipeline/types.go` | Add `ReworkStep` field to `RetryConfig`; add `ContractErrors`, `StepDuration`, `ArtifactPaths`, `FailedStepID` fields to `AttemptContext`; add `StateReworking` constant |
| `internal/pipeline/executor.go` | Add `case "rework"` to on_failure switch; implement `executeReworkStep` method; emit rework events |
| `internal/pipeline/resume.go` | Track rework transitions in `ResumeState`; handle rework step completion during resume |
| `internal/pipeline/validation.go` | Add rework step validation to `DAGValidator` (target exists, no cycles, not upstream) |
| `.wave/schemas/wave-pipeline.schema.json` | Add `rework` to on_failure enum; add `rework_step` property to RetryConfig (if RetryConfig is defined, or to Step) |
| `internal/pipeline/retry_test.go` | Tests for new `RetryConfig` fields |
| `internal/pipeline/executor_test.go` | Tests for rework execution, failure context propagation, artifact replacement |
| `docs/reference/pipeline-schema.md` | Document `on_failure: rework` and `rework_step` |

### Files NOT Changed

- `internal/pipeline/composition.go` — Composition branching is a separate mechanism; rework branching operates at the executor level
- `internal/state/store.go` — Existing state store already tracks step states and attempts; `StateReworking` uses the same persistence as other states
- `internal/adapter/` — No adapter changes needed; rework step uses normal adapter execution

## 4. Architecture Decisions

### AD1: Rework as Inline Step Execution

The rework step is executed inline in `executeStep()` after the on_failure switch triggers. This is simpler than modifying the DAG walk in `ExecutePipeline()` because:
- No DAG modification at runtime
- The rework step runs in the same goroutine/batch context
- Failure context is immediately available
- Artifact path replacement is localized

### AD2: Rework Step Must Not Have Dependencies on the Failed Step

If step A has `rework_step: B`, then B must NOT depend on A (directly or transitively). This prevents cycles and ensures B can run independently. DAG validation enforces this.

### AD3: Rework Step Can Appear Elsewhere in the Pipeline

A rework step is a normal step. It can be the target of multiple `rework_step` references and can also be scheduled normally by the DAG. If it runs via rework, it's marked completed and the DAG skips it when it would normally execute.

### AD4: No Sub-Pipeline Rework (Deferred)

The issue mentions "alternative step/pipeline path." For v1, only same-pipeline step targets are supported. Sub-pipeline rework would require extending the composition executor and is deferred to a follow-up issue.

## 5. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Rework step creates dependency cycle | Pipeline fails to validate | DAG validation checks for transitive cycles through rework targets |
| Rework step fails — double failure | Confusion about pipeline state | Rework step follows its own on_failure policy; if it also fails, the pipeline state reflects the rework step's failure |
| Artifact path collision when rework replaces original | Wrong artifacts consumed downstream | Explicit artifact path replacement under mutex; well-tested |
| Resume doesn't know about rework transition | Step re-executed unnecessarily | Track rework transitions in state DB via a new `rework_of` field in step state |
| Schema backward compatibility | Old schemas reject new fields | `rework_step` is optional; `rework` is added to existing enum; no breaking changes |

## 6. Testing Strategy

### Unit Tests

- **RetryConfig validation**: `rework_step` required when `on_failure: rework`; empty/missing target errors
- **DAG validation**: rework target exists, no cycles, not upstream of failing step
- **Executor rework trigger**: mock adapter returns error, verify rework step executes after retry exhaustion
- **Failure context propagation**: verify enhanced `AttemptContext` fields are populated and injected into rework step
- **Artifact replacement**: verify downstream steps see rework step's artifacts under original step's keys
- **Resume with rework**: verify resume manager correctly handles completed rework steps
- **Existing on_failure unchanged**: verify `fail`, `skip`, `continue` behavior is not affected

### Integration Tests

- End-to-end pipeline with `on_failure: rework` using mock adapter
- Pipeline where rework step also fails (cascading failure)
- Resume from step after rework has occurred

### Regression Tests

- All existing `TestExecuteStep_RetryConfig_OnFailure*` tests pass unchanged
- All existing `TestOptionalStep_*` tests pass unchanged
