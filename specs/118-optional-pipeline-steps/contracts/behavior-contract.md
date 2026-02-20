# Behavior Contracts: Optional Pipeline Steps

## Contract 1: Optional Step Failure Does Not Halt Pipeline

**Requirement**: FR-002, FR-003

**Test**: Given a pipeline with steps [A (required), B (optional), C (required)]:
- When step B fails → pipeline continues, step C executes, pipeline status is "completed"
- When step A fails → pipeline halts immediately (existing behavior preserved)

**Verification**: `go test ./internal/pipeline/ -run TestOptionalStepContinuesPipeline`

---

## Contract 2: Failed Optional State Persisted Correctly

**Requirement**: FR-004

**Test**: Given an optional step that fails:
- Step state in store is `"failed_optional"` (not `"failed"`)
- Step has `completed_at` timestamp set
- Step has `error_message` populated

**Verification**: `go test ./internal/state/ -run TestSaveStepState_FailedOptional`

---

## Contract 3: Contract Validation Skipped for Failed Optional

**Requirement**: FR-005

**Test**: Given an optional step with a contract that would fail validation:
- When the step's adapter execution fails → contract validation is NOT invoked
- Step is marked `"failed_optional"` directly

**Verification**: `go test ./internal/pipeline/ -run TestOptionalStepSkipsContract`

---

## Contract 4: Event Emission for Optional Steps

**Requirement**: FR-006

**Test**: Given an optional step failure:
- Event emitted with `State: "failed_optional"` (not `"failed"`)
- Event has `Optional: true` field set
- Events for optional step's running/progress states also have `Optional: true`

**Verification**: `go test ./internal/pipeline/ -run TestOptionalStepEvents`

---

## Contract 5: Artifact Injection Skipping

**Requirement**: FR-007, FR-009

**Test**: Given step C with `inject_artifacts` from optional step B, and step B failed:
- Step C is marked `"skipped"` (not executed)
- A descriptive message is logged indicating the dependency artifact is unavailable
- Steps that only have `dependencies` (ordering) on step B are NOT skipped

**Verification**: `go test ./internal/pipeline/ -run TestArtifactInjectionSkipping`

---

## Contract 6: Transitive Skip Propagation

**Requirement**: FR-009

**Test**: Given steps A (optional, fails) → B (injects from A) → C (injects from B):
- Step B is skipped (artifact from A unavailable)
- Step C is also skipped (artifact from B unavailable, because B was skipped)

**Verification**: `go test ./internal/pipeline/ -run TestTransitiveSkipPropagation`

---

## Contract 7: Display Distinguishes Optional Failures

**Requirement**: FR-008

**Test**: Given a completed pipeline with optional step failures:
- Summary output shows optional failures with distinct indicator (not same as required failure)
- Pipeline overall status displays as "completed" (not "failed")

**Verification**: `go test ./internal/display/ -run TestOptionalFailureDisplay`

---

## Contract 8: Resume Preserves Optional Step State

**Requirement**: FR-010

**Test**: Given a pipeline run where optional step B failed and required step D failed:
- Resume from step D does not re-execute step B
- Step B's `"failed_optional"` state is preserved in the resumed execution

**Verification**: `go test ./internal/pipeline/ -run TestResumePreservesOptionalState`

---

## Contract 9: Retries Before Failed Optional

**Requirement**: FR-011

**Test**: Given an optional step with `max_retries: 3` that always fails:
- The step is retried 3 times (existing retry mechanism)
- After exhausting retries, step is marked `"failed_optional"` (not `"failed"`)
- Retry events are emitted normally during the retry loop

**Verification**: `go test ./internal/pipeline/ -run TestOptionalStepRetriesExhausted`

---

## Contract 10: YAML Parsing Validation

**Requirement**: FR-012

**Test**: Given YAML with `optional: "not-a-bool"`:
- YAML parsing returns an error
- Given YAML with `optional: true` → parses correctly, `step.Optional == true`
- Given YAML with `optional` field omitted → parses correctly, `step.Optional == false`

**Verification**: `go test ./internal/pipeline/ -run TestStepOptionalYAMLParsing`

---

## Contract 11: All-Optional Pipeline Succeeds

**Requirement**: Edge case

**Test**: Given a pipeline where ALL steps are optional and all fail:
- Pipeline completes with status "completed"
- All steps are marked "failed_optional"

**Verification**: `go test ./internal/pipeline/ -run TestAllOptionalPipelineSucceeds`

---

## Contract 12: Backward Compatibility

**Requirement**: SC-004

**Test**: All existing pipeline tests pass without modification.
- The `optional` field defaults to `false` (Go zero value)
- No existing YAML needs updating
- No existing behavior changes

**Verification**: `go test ./...`
