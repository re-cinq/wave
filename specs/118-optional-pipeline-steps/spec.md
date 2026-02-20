# Feature Specification: Optional Pipeline Steps

**Feature Branch**: `118-optional-pipeline-steps`
**Created**: 2026-02-20
**Status**: Draft
**Input**: [GitHub Issue #118](https://github.com/re-cinq/wave/issues/118) — Add possibility to mark pipeline steps as optional

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Non-Critical Step Continues Pipeline on Failure (Priority: P1)

As a pipeline author, I want to mark individual steps as optional so that when those steps fail, the pipeline continues executing subsequent steps instead of halting entirely.

**Why this priority**: This is the core value proposition. Without this, pipelines with non-critical steps (notifications, linting, staging deploys) halt on any failure, making them fragile for real-world workflows.

**Independent Test**: Can be fully tested by creating a pipeline with an optional step that returns an error, and verifying the pipeline continues to the next step and completes successfully.

**Acceptance Scenarios**:

1. **Given** a pipeline with step B marked `optional: true` between required steps A and C, **When** step B fails during execution, **Then** the pipeline logs the failure for step B, marks it with a non-blocking failed status, and proceeds to execute step C.
2. **Given** a pipeline with step B marked `optional: true` between required steps A and C, **When** step B succeeds, **Then** the pipeline continues normally as if step B were required — its artifacts are available for downstream injection.
3. **Given** a pipeline where all steps are required (default), **When** any step fails, **Then** the pipeline halts immediately (existing behavior preserved).

---

### User Story 2 - Pipeline Summary Distinguishes Optional Failures (Priority: P2)

As a pipeline operator, I want the pipeline summary output to clearly distinguish between required step failures (which stopped the pipeline) and optional step failures (which did not), so I can quickly understand what happened and what needs attention.

**Why this priority**: Without clear reporting, operators cannot tell whether a pipeline "succeeded with warnings" or had real failures. This visibility is essential for operational use.

**Independent Test**: Can be tested by running a pipeline with both a failing optional step and a succeeding required step, then inspecting the summary output and state records.

**Acceptance Scenarios**:

1. **Given** a completed pipeline where an optional step failed, **When** viewing the pipeline summary, **Then** the failed optional step is displayed with a distinct status (not the same as a pipeline-halting failure) and the overall pipeline status reflects success.
2. **Given** a completed pipeline where a required step failed, **When** viewing the pipeline summary, **Then** the pipeline shows as failed and the required step failure is clearly indicated as the cause.
3. **Given** a pipeline state store, **When** querying step states for a run that included optional failures, **Then** the state records indicate which failures were optional vs. required.

---

### User Story 3 - Optional Step Dependency Handling (Priority: P2)

As a pipeline author, I want to use optional steps within dependency chains so that if an optional step fails or is skipped, downstream steps that depend on it handle the missing artifacts gracefully rather than crashing.

**Why this priority**: Real pipelines have dependencies between steps. Without graceful dependency handling for optional steps, the feature is limited to leaf-node steps only, which severely restricts its usefulness.

**Independent Test**: Can be tested by creating a pipeline where step C depends on optional step B, step B fails, and verifying step C either skips (if it depends on optional output) or runs with a clear indication that the dependency artifact is unavailable.

**Acceptance Scenarios**:

1. **Given** step C depends on optional step B and step B fails, **When** step C attempts to inject artifacts from step B, **Then** step C is skipped with a clear message indicating the dependency artifact is unavailable, and the pipeline continues.
2. **Given** step C depends on optional step B and step B succeeds, **When** the pipeline executes step C, **Then** artifacts from step B are injected normally and step C runs as expected.
3. **Given** step C has no dependency on optional step B and step B fails, **When** the pipeline reaches step C, **Then** step C executes normally without any impact from step B's failure.

---

### User Story 4 - Resume Pipeline with Optional Step History (Priority: P3)

As a pipeline operator, I want the resume functionality to correctly handle previously-failed optional steps so that resuming a pipeline from a later step does not re-execute or block on optional steps that already ran.

**Why this priority**: Resume is a key Wave capability. It must work correctly with optional steps, but this is a lower priority because it extends an existing feature rather than enabling a new workflow.

**Independent Test**: Can be tested by running a pipeline where an optional step fails, then resuming from a step after it, and verifying the resume skips the already-executed optional step.

**Acceptance Scenarios**:

1. **Given** a pipeline run where optional step B failed and the pipeline continued to step D which also failed (for an unrelated reason), **When** the operator resumes from step D, **Then** the pipeline resumes from step D without re-executing optional step B and its failed status is preserved.

---

### Edge Cases

- What happens when **all steps** in a pipeline are marked optional? The pipeline should succeed (vacuously) even if every step fails, since no required step failed.
- What happens when an optional step has **retry configuration**? The step should exhaust its retries before being considered failed-optional. Retries still apply to optional steps.
- What happens when an optional step is the **last step** in the pipeline? The pipeline completes successfully even if the last optional step fails.
- What happens when a **required step depends on a failed optional step's artifacts**? The required step should fail with a clear error indicating its dependency (an optional step) did not produce the expected artifact.
- What happens when a pipeline is configured with `optional: false` (explicit default)? Behavior is identical to omitting the field entirely — the step is required.
- What happens during **contract validation** for a failed optional step? Contract validation is skipped for steps that failed with optional status, since there is no output to validate.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST support an `optional` boolean field as a top-level field on the `Step` struct (`yaml:"optional,omitempty"`), defaulting to `false` when omitted.
- **FR-002**: System MUST continue pipeline execution past a failed optional step, proceeding to the next step in topological order.
- **FR-003**: System MUST preserve existing behavior for required steps — a failed required step halts the pipeline.
- **FR-004**: System MUST record the execution state of failed optional steps as `"failed_optional"` (a new `StepState` constant) in the pipeline state store, distinct from the existing `"failed"` state used for required steps.
- **FR-005**: System MUST skip contract validation for optional steps that fail, since there is no output to validate.
- **FR-006**: System MUST emit progress events that distinguish optional step failures from required step failures, using both a `"failed_optional"` event state and an `Optional bool` field on the Event struct.
- **FR-007**: System MUST handle artifact injection gracefully when the source step is optional and failed — downstream steps that have `memory.inject_artifacts` references to a `"failed_optional"` or `"skipped"` step are themselves skipped. Steps that only reference a failed optional step via `dependencies` (ordering only, without artifact injection) are NOT skipped.
- **FR-008**: System MUST display optional step failures distinctly from required step failures in pipeline summary output.
- **FR-009**: System MUST support optional steps within dependency chains — a step with artifact injection from a failed optional step is itself skipped (with `"skipped"` state), regardless of whether it is required or optional. This skipping propagates transitively: if step C injects artifacts from step B, and step B was skipped due to step A failing, step C is also skipped.
- **FR-010**: System MUST correctly handle optional step state during pipeline resume — failed optional steps are not re-executed on resume.
- **FR-011**: System MUST apply retry configuration to optional steps before declaring them failed-optional — retries are exhausted first.
- **FR-012**: System MUST validate the `optional` field during manifest parsing and reject invalid values (non-boolean).

### Key Entities

- **Step** (`internal/pipeline/types.go:Step`): A unit of pipeline execution. Extended with a top-level `Optional bool` field (`yaml:"optional,omitempty"`, default `false`) that controls whether its failure halts the pipeline. Key attributes: id, persona, dependencies, optional flag, handover config.
- **Step State** (`internal/state/store.go:StepState`): The persisted execution state of a step within a pipeline run. Extended with new constant `StateFailedOptional StepState = "failed_optional"` to distinguish non-blocking failures. The existing `display.StateSkipped = "skipped"` is used for dependency-skipped downstream steps. Key attributes: step ID, state, error message.
- **Event** (`internal/event/emitter.go:Event`): Progress event struct. Extended with `Optional bool` field (`json:"optional,omitempty"`) and new state constant `StateFailedOptional = "failed_optional"`. Key attributes: pipeline ID, step ID, state, optional flag.
- **Pipeline Summary**: The aggregated result of a pipeline execution. Uses existing `"completed"` status when all required steps pass. Extended to separately report required failures (pipeline-halting) and optional failures (informational) in display output. Key attributes: overall status, step states, failure categorization.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A pipeline with a failing optional step and a succeeding required step completes with overall success status.
- **SC-002**: A pipeline with a failing required step halts immediately, regardless of any optional steps — existing behavior is preserved with zero regressions.
- **SC-003**: Pipeline summary output contains distinct indicators for optional vs. required step failures in 100% of runs that include optional step failures.
- **SC-004**: All existing pipeline tests pass without modification (backward compatibility), since the default value of the optional field is `false`.
- **SC-005**: Artifact injection from a failed optional step to a downstream step results in the downstream step being skipped with a descriptive message, not an unhandled error.
- **SC-006**: Pipeline resume correctly preserves optional step failure state and does not re-execute failed optional steps.

## Clarifications _(resolved during specification refinement)_

### CLR-001: Non-blocking failure state value

**Question**: What string value should be used for the distinct non-blocking failure status referenced in FR-004?

**Resolution**: Use `"failed_optional"` as the new `StepState` constant. This follows the existing codebase convention of lowercase snake_case state strings (e.g., `"pending"`, `"running"`, `"completed"`, `"failed"`, `"retrying"` in `internal/state/store.go`). The display package already defines `StateSkipped = "skipped"` which will be used for dependency-skipped downstream steps.

**Rationale**: A dedicated state value (rather than overloading `"failed"` with a flag) ensures backward-compatible state queries — existing consumers that filter on `"failed"` will not accidentally match optional failures.

### CLR-002: Dependency skipping granularity for mixed dependencies

**Question**: When a step depends on both a failed optional step (for artifact A) and a successful required step (for artifact B), should the step be skipped entirely or only when it has an unsatisfied artifact injection?

**Resolution**: A step is skipped only if it has an **artifact injection reference** (`memory.inject_artifacts`) pointing to a step that is in `"failed_optional"` or `"skipped"` state. If a step lists multiple dependencies but only injects artifacts from successful ones, it runs normally. Steps whose `dependencies` field references a failed optional step (for ordering only, without artifact injection) are NOT skipped — they proceed in execution order.

**Rationale**: This aligns with how `injectArtifacts()` in `internal/pipeline/executor.go` works — it iterates `step.Memory.InjectArtifacts` references, not the `dependencies` field. The `dependencies` field controls topological ordering only. Skipping based on artifact injection (the actual data coupling) is more precise and avoids unnecessarily skipping steps that only have an ordering dependency.

### CLR-003: Pipeline overall status when optional steps fail

**Question**: Should the pipeline use a new status value (e.g., `"completed_with_warnings"`) or reuse `"completed"` when optional steps fail but all required steps succeed?

**Resolution**: Reuse the existing `"completed"` state for the pipeline-level status. Optional failure details are captured at the step level via the `"failed_optional"` state and in the pipeline summary output.

**Rationale**: Introducing a new pipeline-level state would break existing consumers (dashboard queries, event listeners, CLI status checks) that match on `"completed"`. The pipeline genuinely succeeded from a workflow perspective. The step-level `"failed_optional"` state provides sufficient granularity for operators who need to inspect optional failures.

### CLR-004: YAML field placement for `optional`

**Question**: Where exactly in the YAML schema should the `optional` field be placed — as a top-level Step field or nested under a sub-config?

**Resolution**: Add `Optional bool` as a **top-level field on the `Step` struct** in `internal/pipeline/types.go`, with YAML tag `yaml:"optional,omitempty"`. It defaults to `false` (zero value for bool).

**Rationale**: This is consistent with other step-level behavioral flags in the codebase. The `PipelineMetadata` struct uses top-level `Disabled bool` and `Release bool`. The `ArtifactDef` struct uses top-level `Required bool`. Nesting under `handover` or `exec` would be semantically incorrect since `optional` affects pipeline-level flow control, not handover or execution behavior.

### CLR-005: Event structure for optional failure distinction

**Question**: How should progress events distinguish optional step failures — via a new event state string, a new field on the Event struct, or both?

**Resolution**: Use **both** mechanisms:
1. A new event state string `"failed_optional"` for step failure events when the step is optional.
2. A new boolean field `Optional bool` on the `Event` struct (`json:"optional,omitempty"`) that is set to `true` on all events related to optional steps (start, progress, failed_optional, skipped).

**Rationale**: The dual approach serves different consumers. The state string provides human-readable event streams and is consistent with existing state-based event routing (e.g., `"retrying"`, `"contract_failed"`). The boolean field enables structured filtering in programmatic consumers (dashboard, NDJSON parsers) without parsing state strings. The `omitempty` tag ensures zero overhead for non-optional steps.
