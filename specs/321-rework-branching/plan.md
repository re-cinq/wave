# Implementation Plan: Rework Branching (#321)

## 1. Objective

Extend the pipeline executor's failure recovery to support rework branching — a new `on_failure: rework` policy that redirects execution to an alternative step or sub-pipeline when all retry attempts are exhausted, carrying rich failure context forward.

## 2. Approach

### Strategy: Extend existing retry/on_failure infrastructure

The rework branching capability builds on two existing systems:
1. **RetryConfig.OnFailure** — add `"rework"` as a new policy alongside `fail`, `skip`, `continue`
2. **Composition primitives** — leverage the existing `SubPipelineLoader` and `CompositionExecutor` patterns for executing rework targets

### YAML Schema

```yaml
steps:
  - id: implement
    persona: craftsman
    retry:
      max_attempts: 3
      adapt_prompt: true
      on_failure: rework
      rework:
        target_step: diagnose     # Alternative: target_pipeline: fix-pipeline
        max_rework_depth: 1       # Default: 1 (prevents infinite loops)
        inject_failure_context: true  # Default: true
```

The `rework` block lives inside `RetryConfig` since it only applies when `on_failure: rework`. Two mutually exclusive targeting modes:
- `target_step` — redirect to another step in the same pipeline
- `target_pipeline` — redirect to a named sub-pipeline

### Failure Context Enrichment

A new `ReworkContext` struct extends `AttemptContext` with:
- Full attempt history (all attempts, not just last)
- Partial artifact paths from the failed step
- Original step configuration metadata
- Rework depth counter

This context is injected into the rework target's workspace as `.wave/artifacts/rework_context` (JSON), making it available to the rework step's persona via artifact injection.

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | modify | Add `ReworkConfig` struct, add to `RetryConfig`; add `StateReworking` constant; add `ReworkContext` struct |
| `internal/pipeline/executor.go` | modify | Add `"rework"` case in on_failure switch; implement `executeRework` method; inject rework context |
| `internal/pipeline/resume.go` | modify | Load rework state on resume; handle rework-in-progress pipelines |
| `internal/pipeline/dag.go` | modify | Validate rework target steps exist in DAG validation |
| `internal/pipeline/errors.go` | modify | Add rework-specific error type |
| `internal/state/store.go` | modify | Add rework tracking methods to `StateStore` interface |
| `internal/state/types.go` | modify | Add `ReworkRecord` type for rework state persistence |
| `internal/state/migrations.go` | modify | Add migration for rework tracking table |
| `internal/state/migration_definitions.go` | modify | Define rework migration SQL |
| `internal/event/types.go` | modify | Add rework event states (`StateReworking`, `StateReworkCompleted`, `StateReworkFailed`) |
| `internal/pipeline/executor_test.go` | modify | Add rework branching unit tests |
| `internal/pipeline/dag_test.go` | modify | Add rework target validation tests |
| `internal/pipeline/resume_test.go` | modify | Add rework resume tests |

## 4. Architecture Decisions

### AD-1: Rework config nested in RetryConfig
**Decision**: `ReworkConfig` is a sub-struct of `RetryConfig`, not a peer field on `Step`.
**Rationale**: Rework only applies when retries are exhausted (i.e., it's a retry escalation policy). Nesting communicates this relationship and keeps the Step struct flat.

### AD-2: Target step vs target pipeline — mutually exclusive
**Decision**: `target_step` and `target_pipeline` are mutually exclusive fields.
**Rationale**: Mirrors the existing `ArtifactRef` pattern where `Step` and `Pipeline` are mutually exclusive (see `types.go:181-186`). Validated at DAG load time.

### AD-3: Rework depth limit default = 1
**Decision**: Default `max_rework_depth: 1` prevents infinite rework loops. A rework target step that itself has `on_failure: rework` will be blocked by depth limit.
**Rationale**: Without a depth limit, a chain of rework targets could loop forever. Default of 1 means "try rework once, then fall through to fail."

### AD-4: Rework context as artifact injection
**Decision**: Failure context is serialized to `.wave/artifacts/rework_context` as JSON and injected like any other artifact.
**Rationale**: Reuses existing artifact injection infrastructure. The rework target step's persona reads it like any other input artifact — no special handling needed in the adapter layer.

### AD-5: Rework state = "reworking"
**Decision**: Add `StateReworking = "reworking"` alongside existing `StateRetrying`, `StateFailed`, etc.
**Rationale**: Observable state transitions need a distinct state for rework. This appears in events, state DB, and TUI display.

### AD-6: No circular rework references
**Decision**: DAG validation rejects cycles in rework targets (A reworks to B, B reworks to A).
**Rationale**: Prevents infinite loops at pipeline load time rather than at runtime.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Rework depth limit bypass via chained target_pipeline | Low | High | Track rework depth across sub-pipeline boundaries; store in execution context |
| Resume from mid-rework loses original step context | Medium | Medium | Persist rework chain in state DB; `loadResumeState` reconstructs context |
| DAG validation false positives when rework target has complex deps | Low | Low | Rework target validation is opt-in (only when `on_failure: rework`) |
| Rework target step not ready (deps not met) | Medium | Medium | When reworking to a same-pipeline step, mark deps as satisfied from original execution |

## 6. Testing Strategy

### Unit Tests
- `RetryConfig` with `on_failure: rework` parses correctly
- `ReworkConfig` validation (mutually exclusive `target_step`/`target_pipeline`)
- DAG validation catches missing rework target steps
- DAG validation catches circular rework references
- Rework depth tracking and limit enforcement
- `ReworkContext` serialization/deserialization

### Integration Tests
- Step fails all retries → rework target step executes
- Rework target receives correct failure context
- Rework target succeeds → pipeline continues
- Rework target fails → pipeline fails (depth limit 1)
- Rework to sub-pipeline executes correctly
- Resume from rework-in-progress state works
- State DB records rework transitions

### Edge Cases
- Rework with `max_rework_depth: 0` → acts like `fail`
- Rework target step also fails → respects its own on_failure policy
- Rework with `adapt_prompt: true` → combines retry adaptation with rework context
- Concurrent steps where one enters rework
