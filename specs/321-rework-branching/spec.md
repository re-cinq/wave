# audit: partial — failure recovery with retry and rework (#287)

**Issue**: [#321](https://github.com/re-cinq/wave/issues/321)
**Repository**: re-cinq/wave
**Author**: nextlevelshit
**State**: OPEN
**Complexity**: complex

## Audit Finding: Partial

**Source**: [#287 — feat(pipeline): failure recovery with retry and rework](https://github.com/re-cinq/wave/issues/287)

### Category
**Partial** — Retry logic exists but rework branching is missing.

### Evidence
- `internal/pipeline/executor.go:460-598` retry logic with prompt adaptation
- `on_failure` supports: fail, skip, continue, retry
- `internal/pipeline/resume.go:137` loads failure context from prior attempts
- Partial artifacts preserved in state DB

### Remediation
Implement rework branching: allow `on_failure` to specify an alternative step/pipeline path. Enhance resume to carry richer failure context.

---

## Current State Analysis

### Existing Retry Infrastructure
- **RetryConfig** (`types.go:75-81`): `max_attempts`, `backoff`, `base_delay`, `adapt_prompt`, `on_failure`
- **on_failure policies**: `fail` (default), `skip`, `continue` — all terminal; no branching
- **AttemptContext** (`types.go:119-126`): carries `Attempt`, `MaxAttempts`, `PriorError`, `FailureClass`, `PriorStdout`
- **StepAttemptRecord** (`state/types.go:155-169`): persists `ErrorMessage`, `FailureClass`, `StdoutTail`
- **Resume** (`resume.go`): `loadResumeState` loads `FailureContexts` from prior attempts

### Existing Composition Infrastructure
- **BranchConfig** (`types.go:323-327`): conditional branching based on template expressions — evaluates `on` condition and selects from `cases` map
- **CompositionExecutor** (`composition.go`): full iterate/branch/gate/loop/aggregate execution
- **SubPipelineLoader**: loads and runs child pipelines by name

### Gap
When all retry attempts are exhausted, the only options are: stop (`fail`), skip the step (`skip`), or log failure and continue (`continue`). There is no way to redirect execution to an alternative step or sub-pipeline for rework — e.g., "if implementation fails, run a diagnostic step and re-attempt."

## Acceptance Criteria

1. **New `on_failure` policy: `rework`** — When all retry attempts are exhausted, instead of terminating, execute an alternative step or sub-pipeline path
2. **YAML schema for rework declaration** — `on_failure: rework` with a `rework` config block specifying the target step or pipeline
3. **Richer failure context** — The rework target receives structured failure context including: error message, failure class, stdout tail, attempt history, and partial artifacts from the failed step
4. **State persistence** — Rework transitions are recorded in the state DB (step attempts, rework target metadata)
5. **DAG validation** — Rework targets are validated at pipeline load time (target step/pipeline must exist)
6. **Resume compatibility** — Resuming a pipeline that entered rework mode works correctly
7. **Event emission** — Rework transitions emit observable progress events
8. **Rework depth limit** — Configurable limit to prevent infinite rework loops (default: 1)
9. **Tests** — Unit tests for rework branching, integration tests for end-to-end rework flow
