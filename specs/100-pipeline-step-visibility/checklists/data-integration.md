# Data Model & Integration Requirements Quality Checklist

**Feature**: Pipeline Step Visibility in Default Run Mode
**Spec**: `specs/100-pipeline-step-visibility/spec.md`
**Date**: 2026-02-14
**Focus**: Data model changes, API contracts, and cross-component integration

## Data Model Completeness

- [ ] CHK201 - Is the `StepPersonas` field's initialization behavior defined? Should it be initialized as an empty map or left nil when no personas are configured? [Completeness]
- [ ] CHK202 - Does the spec define what `StepDurations` should contain for failed steps? The data-model says "completed and failed show final duration" but no FR explicitly requires failed step duration display. [Completeness]
- [ ] CHK203 - Is the `CreatePipelineContext()` API change (accepting `stepPersonas` parameter) backward-compatible with existing callers, or are all call sites enumerated? [Completeness]
- [ ] CHK204 - Does the spec define the lifecycle of `StepPersonas` entries — are they set once at registration and immutable, or can persona assignments change? [Completeness]
- [ ] CHK205 - Is the relationship between `CurrentPersona` (existing field for running step) and `StepPersonas[currentStepID]` defined? Should they always be consistent? [Completeness]

## Integration Clarity

- [ ] CHK206 - Is the contract between `toPipelineContext()` and the renderer clearly defined — specifically, which fields the renderer may assume are always populated vs. optionally populated? [Clarity]
- [ ] CHK207 - Is the thread-safety claim (existing mutex is sufficient) validated by identifying all code paths that write to `StepPersonas`? [Clarity]
- [ ] CHK208 - Is the `AddStep()` call site in `output.go` documented as the single population point for persona data, or could other code paths also set personas? [Clarity]

## Cross-Artifact Consistency

- [ ] CHK209 - Do the tasks (T001-T029) cover all six changes described in the plan (Changes 1-6), with no plan change left unimplemented? [Consistency]
- [ ] CHK210 - Does the task dependency ordering (Phase 1 before Phase 2/3, Phase 4 after 2/3) match the plan's stated dependencies? [Consistency]
- [ ] CHK211 - Are the file paths referenced in tasks.md consistent with those in plan.md and data-model.md? [Consistency]
- [ ] CHK212 - Does the test plan in tasks.md (T016-T026) cover all rows in the plan's Test Plan table? [Consistency]
- [ ] CHK213 - Is the `StepPersonas` field placement in the struct ("after StepDurations") consistent between T001, the plan (Change 1), and the data-model document? [Consistency]

## Test Coverage Adequacy

- [ ] CHK214 - Are there test requirements validating that `StepPersonas` is correctly populated through the full data flow (AddStep → toPipelineContext → renderer)? [Coverage]
- [ ] CHK215 - Is there a test requirement for the `ProgressDisplay` path's `toPipelineContext()` (not just BubbleTea), since T024 only tests the BubbleTea path? [Coverage]
- [ ] CHK216 - Does the test plan include negative tests (e.g., step not in `StepPersonas` map, step not in `StepStatuses` but in `StepOrder`)? [Coverage]
- [ ] CHK217 - Is there a test requirement that validates existing display tests still pass (SC-006) beyond just running the test suite (T027)? [Coverage]
