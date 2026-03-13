# Failure Recovery with Retry and Rework (#321)

**Issue**: [#321 — audit: partial — failure recovery with retry and rework](https://github.com/re-cinq/wave/issues/321)
**Source**: [#287 — feat(pipeline): failure recovery with retry and rework](https://github.com/re-cinq/wave/issues/287)
**Category**: Partial — Retry logic exists but rework branching is missing

## Background

The pipeline executor currently supports four `on_failure` actions for `RetryConfig`:
- `fail` — halt the pipeline (default)
- `skip` — skip the step and mark as skipped
- `continue` — mark as failed but continue pipeline execution
- (implicit retry via `max_attempts > 1`)

However, there is no mechanism to redirect execution to an alternative step or sub-pipeline when a step exhausts its retry attempts. This "rework branching" capability would allow pipeline authors to define fallback paths — for example, running a simpler implementation strategy when a complex one fails, or routing to a human review step on repeated failures.

## Evidence from Codebase

- `internal/pipeline/executor.go:682-733` — `on_failure` switch handles `skip`, `continue`, and `fail` only
- `internal/pipeline/types.go:74-81` — `RetryConfig` struct with `OnFailure` field
- `internal/pipeline/resume.go:137` — `loadResumeState` loads `FailureContexts` from prior attempts
- `internal/pipeline/types.go:119-126` — `AttemptContext` struct with limited failure context fields
- `internal/pipeline/composition.go:264-298` — `executeBranch` handles conditional branching for composition steps (different mechanism)

## Requirements

### R1: Rework Branching via `on_failure: rework`

Add a new `on_failure` action `rework` to `RetryConfig` that redirects execution to an alternative step within the same pipeline when all retry attempts are exhausted.

**YAML syntax**:
```yaml
retry:
  max_attempts: 3
  on_failure: rework
  rework_step: fallback-step-id
```

When `on_failure: rework` is triggered:
1. The failed step is marked as `StateFailed`
2. The failure context (error, stdout tail, attempt count, failure class) is carried forward
3. The rework target step is executed with the failure context injected
4. The rework step's output replaces the failed step's output for downstream dependency resolution
5. If the rework step itself fails, normal `on_failure` semantics apply to the rework step

### R2: Enhanced Failure Context

Extend `AttemptContext` with richer failure metadata to give rework steps better information:

- `ContractErrors`: structured contract validation errors (not just string)
- `StepDuration`: how long the step ran before failing
- `ArtifactPaths`: paths to any partial artifacts the failed step produced
- `FailedStepID`: the ID of the step that triggered rework (useful when rework step serves multiple failed steps)

### R3: Resume Support for Rework

Enhance `ResumeManager.loadResumeState` to track rework transitions so that:
- Pipeline state correctly reflects that a rework step was executed
- Resumed pipelines can skip rework steps that already completed
- Failure context from the original step is available during resume

### R4: Pipeline Schema Update

Update `wave-pipeline.schema.json` to:
- Add `rework` to the `on_failure` enum for RetryConfig
- Add `rework_step` field to RetryConfig
- Document the rework branching behavior

### R5: DAG Validation

Extend DAG validation to verify that:
- `rework_step` references an existing step ID in the pipeline
- Rework targets don't create cycles in the dependency graph
- Rework targets are not upstream dependencies of the failing step

## Acceptance Criteria

- [ ] `on_failure: rework` with `rework_step` redirects to the specified step after retry exhaustion
- [ ] Failure context is injected into the rework step's `AttemptContexts`
- [ ] Rework step's output is available to downstream steps via the original step's artifact keys
- [ ] DAG validation catches invalid rework targets (cycles, missing steps)
- [ ] Pipeline schema validates `rework_step` when `on_failure: rework`
- [ ] Resume correctly handles pipelines with rework transitions
- [ ] Unit tests cover: rework trigger, failure context propagation, DAG validation, resume
- [ ] Existing `on_failure` behaviors (fail, skip, continue) remain unchanged

## Labels

None

## Metadata

- **Author**: nextlevelshit
- **State**: OPEN
- **Complexity**: complex
