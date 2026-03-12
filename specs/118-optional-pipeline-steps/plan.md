# Implementation Plan: Optional Pipeline Steps

## 1. Objective

Add an `optional: true` field to pipeline step configuration so that when an optional step fails, the pipeline logs the failure and continues execution instead of halting. Dependent steps of a failed optional step are skipped.

## 2. Approach

The implementation builds on the existing `retry.on_failure` mechanism in the executor. The key insight is that the executor already handles `on_failure: "continue"` and `on_failure: "skip"` — we add `Optional bool` as a first-class field on `Step` that influences the executor's failure handling without requiring users to understand the retry subsystem.

**Strategy**: Add the `Optional` field to the Step struct, then modify the executor to check `step.Optional` when a step fails (after all retry attempts). If optional, treat it as `on_failure: "continue"`. Additionally, modify `findReadySteps` to skip steps whose dependencies include a failed optional step.

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | modify | Add `Optional bool` field to `Step` struct |
| `internal/pipeline/executor.go` | modify | Handle optional step failure in `executeStep`, modify `findReadySteps` to skip steps with failed optional deps, update completion logic to track optional failures separately |
| `internal/pipeline/executor_test.go` | modify | Add tests for optional step behavior |
| `internal/display/types.go` | modify | Add `OptionalFailedSteps int` to display context |
| `internal/display/dashboard.go` | modify | Display optional failures distinctly in summary |
| `internal/display/progress.go` | modify | Track optional failures in progress computation |
| `internal/display/bubbletea_model.go` | modify | Show optional failures in TUI status line |
| `internal/pipeline/dag.go` | no change | DAG validation doesn't need changes — optional is an execution concern |
| `internal/state/store.go` | no change | Existing `StateFailed` and `StateSkipped` states suffice |

## 4. Architecture Decisions

### 4.1 First-class field vs. retry sugar

**Decision**: `Optional` is a first-class `bool` field on `Step`, not sugar over `retry.on_failure`.

**Rationale**:
- Clearer YAML ergonomics (`optional: true` vs. `retry: { on_failure: continue }`)
- Allows `optional` + `retry` to coexist (e.g., optional step with 3 retries before giving up)
- The field's semantics are "this step's failure is non-blocking" — orthogonal to retry policy

### 4.2 Precedence: `optional` vs. `retry.on_failure`

When both are set:
- `retry.on_failure` takes explicit precedence if set to a non-empty value
- `optional: true` acts as a fallback — if `retry.on_failure` is empty (or unset), optional steps default to `on_failure: "continue"`

This avoids surprises: a user who explicitly sets `retry.on_failure: "fail"` on an optional step gets the explicit behavior.

### 4.3 Dependency propagation for failed optional steps

When an optional step fails:
- Its state becomes `StateFailed` (not `StateSkipped`) to accurately reflect what happened
- Steps that depend on it are marked `StateSkipped` — they can't run without artifacts from the failed step
- The `findReadySteps` function checks for failed/skipped dependencies and skips them transitively
- The `completed` map treats failed-optional and skipped steps as "done" to avoid deadlock

### 4.4 Pipeline-level success/failure

A pipeline succeeds if all non-optional steps complete. Optional step failures don't affect the pipeline exit code. The `PipelineStatus.FailedSteps` field continues to track all failures, but a new helper distinguishes blocking vs. non-blocking failures.

## 5. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing `retry.on_failure` behavior | High | `optional` only applies when `retry.on_failure` is unset; explicit config always wins |
| Deadlock in DAG when optional step fails | High | Failed optional steps marked as "completed" in the DAG traversal set |
| Missing artifacts for dependent steps | Medium | Dependent steps are skipped before execution, not during — no partial artifact state |
| Display output confusion | Low | Clearly label optional failures differently from blocking failures |

## 6. Testing Strategy

### Unit Tests

1. **Basic optional step failure**: Optional step fails → pipeline continues → next step runs
2. **Optional step success**: Optional step succeeds → behaves identically to required step
3. **Default behavior preserved**: Step without `optional` field → pipeline halts on failure (regression test)
4. **Optional with retry**: Optional step with `max_attempts: 3` → retries, then continues on exhaustion
5. **Dependency skip on optional failure**: Step B depends on optional step A → A fails → B is skipped → pipeline succeeds
6. **Transitive skip**: C depends on B depends on optional A → A fails → B and C skipped
7. **Mixed dependencies**: C depends on B (required) and A (optional) → A fails → C skipped (missing dep)
8. **Precedence**: `optional: true` + `retry.on_failure: "fail"` → step failure halts pipeline (explicit wins)
9. **Pipeline status**: Optional failures appear in `FailedSteps` but pipeline status is `completed`
10. **Display output**: Verify display context correctly counts optional vs. required failures

### Integration Tests

- YAML round-trip: Parse `optional: true` from YAML, verify struct field, serialize back
- End-to-end: Execute a multi-step pipeline with optional step failure, verify final state
