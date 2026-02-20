# Research: Optional Pipeline Steps

**Feature**: 118-optional-pipeline-steps
**Date**: 2026-02-20

## Decision 1: New Step State Value — `failed_optional`

**Decision**: Introduce `StateFailedOptional StepState = "failed_optional"` as a new constant in both `internal/state/store.go` and `internal/pipeline/types.go`.

**Rationale**: The spec requires distinguishing non-blocking failures from pipeline-halting failures (FR-004). Overloading the existing `"failed"` state with a boolean flag was considered but rejected because:
- Existing state queries across the codebase (dashboard, CLI, event log) filter on exact string matches against `"failed"`. Overloading would silently change their semantics.
- The state store `SaveStepState` uses string-typed `StepState` constants. A new constant follows the established convention (`"pending"`, `"running"`, `"completed"`, `"failed"`, `"retrying"`).
- The display package already defines `StateSkipped = "skipped"` which establishes precedent for states beyond the core five.

**Alternatives Rejected**:
1. *Boolean flag on StepStateRecord*: Would require all consumers to check both `State` and a new `Optional` flag. Increases query complexity and risk of missed filtering.
2. *Separate `optional_failures` table*: Over-engineered for a simple state distinction. Adds joins to every state query.

**Constitution Impact**: Principle 12 limits step states to 5 (`Pending → Running → Completed / Failed / Retrying`). Adding `failed_optional` requires a constitution amendment. Justification: during Rapid Prototype phase, the amendment is lightweight (commit modifying constitution.md with rationale). The new state is a terminal state (like `completed` and `failed`) and does not introduce new transitions — it's an alternative terminal path from `Running`.

---

## Decision 2: Dependency Skipping Strategy — Artifact Injection Based

**Decision**: Skip downstream steps only when they have `memory.inject_artifacts` references to a step in `"failed_optional"` or `"skipped"` state. Steps with ordering-only `dependencies` on a failed optional step are NOT skipped.

**Rationale**: This aligns with the existing `injectArtifacts()` implementation in `executor.go:1030-1084`, which iterates `step.Memory.InjectArtifacts` (not `step.Dependencies`). The `dependencies` field controls topological sort order only — it has no data coupling. Skipping based on artifact injection (the actual data dependency) is more precise and avoids unnecessarily blocking steps that only need ordering guarantees.

**Implementation Approach**: Before calling `executeStep()`, check each artifact injection reference. If any referenced step is in `"failed_optional"` or `"skipped"` state, mark the current step as `"skipped"` and continue. This check is a pre-execution guard in the main loop of `Execute()`.

**Transitive Propagation**: If step C injects artifacts from step B, and step B was skipped (because step A failed as optional), step C is also skipped. This propagation is automatic because step B will be in `"skipped"` state, and the same pre-execution check will catch it for step C.

**Alternatives Rejected**:
1. *Skip based on `dependencies` field*: Would over-skip — many steps use `dependencies` for ordering without injecting artifacts. Would break pipelines where an optional linting step is listed as a dependency for ordering but downstream steps don't need its output.
2. *Require explicit `skip_on_optional_failure` flag*: Adds complexity to YAML schema with minimal benefit. The artifact injection reference already encodes the data dependency.

---

## Decision 3: Pipeline-Level Status — Reuse `"completed"`

**Decision**: When all required steps pass but optional steps fail, the pipeline-level status remains `"completed"`. Optional failure details are captured at step level via `"failed_optional"` state and in summary output.

**Rationale**: Introducing `"completed_with_warnings"` would break:
- Dashboard queries (`WHERE status = 'completed'`)
- CLI status checks (`wave ops status`)
- Event listeners filtering on `"completed"`
- The `GetStatus()` method's state matching in `executor.go:1461-1478`

The pipeline genuinely succeeded from a workflow perspective — all required work was done. Step-level granularity is sufficient for operators.

**Alternatives Rejected**:
1. *New `"completed_with_warnings"` status*: Breaking change to all status consumers with no functional benefit over step-level detail.
2. *Metadata field on PipelineStateRecord*: Adds schema migration and query complexity for information already available from step states.

---

## Decision 4: Contract Validation Skip for Failed Optional Steps

**Decision**: Skip contract validation entirely for optional steps that fail (FR-005). No output exists to validate.

**Rationale**: Contract validation in `runStepExecution()` (executor.go:658-721) runs after the adapter produces output. If the adapter fails, there is no output to validate. Attempting validation on missing/partial output would produce confusing error messages. The step is already marked as `"failed_optional"`, so validation is moot.

**Implementation**: The skip happens naturally — contract validation only runs if `runStepExecution()` succeeds. The change is in `executeStep()` where we intercept the error from `runStepExecution()` and, if the step is optional, mark it `"failed_optional"` instead of propagating the error.

---

## Decision 5: Event Structure — Dual Mechanism

**Decision**: Use both a new event state string `"failed_optional"` and a new `Optional bool` field on `Event` struct for distinguishing optional step events.

**Rationale**: Different consumers need different interfaces:
- NDJSON parsers and dashboard use structured fields (`Optional bool`) for filtering
- Human-readable event streams and log replay use state strings (`"failed_optional"`)
- The `omitempty` JSON tag ensures zero overhead for the 95%+ of events from non-optional steps

**Implementation**: Add `Optional bool` field to `event.Event` with `json:"optional,omitempty"`. Add `StateFailedOptional = "failed_optional"` constant to event package. Set `Optional: true` on all events related to optional steps (running, failed_optional, skipped).

---

## Decision 6: YAML Field Placement — Top-Level on Step

**Decision**: Add `Optional bool` as a top-level field on the `Step` struct with tag `yaml:"optional,omitempty"`.

**Rationale**: Consistent with existing patterns:
- `PipelineMetadata` has top-level `Disabled bool` and `Release bool`
- `ArtifactDef` has top-level `Required bool`
- The `optional` field affects pipeline-level flow control, not handover/exec behavior
- Nesting under `handover` or `exec` would be semantically incorrect

**Validation**: Go's YAML unmarshaling handles bool validation inherently — non-boolean values will cause parse errors. No additional validation code needed for FR-012 beyond what `yaml.v3` already provides.

---

## Decision 7: Retry Behavior for Optional Steps

**Decision**: Optional steps exhaust their configured retries before being marked `"failed_optional"` (FR-011).

**Rationale**: The retry mechanism in `executeStep()` (executor.go:354-405) already handles retries before declaring failure. The only change is: after exhausting retries, instead of propagating the error (which halts the pipeline), check `step.Optional` and either propagate (required) or absorb (optional, mark as `"failed_optional"`).

**Implementation**: The retry loop stays identical. The branching point is after the loop completes with an error — check `step.Optional` to decide between `return lastErr` and `mark failed_optional + return nil`.

---

## Decision 8: Resume Compatibility

**Decision**: Failed optional steps are preserved as `"failed_optional"` during resume and are not re-executed (FR-010).

**Rationale**: The existing `Resume()` method in `executor.go:1378-1431` skips steps where `execution.States[step.ID] == StateCompleted`. Adding `StateFailedOptional` to the skip condition preserves the same pattern — the step already ran, its outcome is recorded, re-running it would not change the pipeline outcome.

The `ResumeManager` in `resume.go` creates a sub-pipeline from `fromStep` onwards. Failed optional steps before `fromStep` are already excluded from the sub-pipeline. Failed optional steps after `fromStep` would need to be re-evaluated, but the simplest approach is to not re-execute them (they already failed).
