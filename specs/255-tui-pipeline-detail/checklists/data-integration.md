# Data Integration & Provider Quality: TUI Pipeline Detail Right Pane

**Feature**: 255-tui-pipeline-detail | **Date**: 2026-03-06

## Provider Interface Design

- [ ] CHK045 - Does the spec define whether `DetailDataProvider` is a new interface or extends the existing `PipelineDataProvider`? Is the single-responsibility boundary between list-level and detail-level data clearly justified? [Clarity]
- [ ] CHK046 - Are the method signatures (`FetchAvailableDetail`, `FetchFinishedDetail`) defined with enough precision to determine the required state store queries? [Completeness]
- [ ] CHK047 - Does the spec define whether provider methods are synchronous (blocking) or must return via Bubble Tea async commands? Is the async pattern consistent with how `PipelineDataProvider` fetches list data? [Consistency]
- [ ] CHK048 - Is mock injection for testing explicitly required? Does the spec mention that the provider must be an interface (not a concrete struct) to enable test doubles? [Completeness]

## Available Pipeline Data

- [ ] CHK049 - Does the spec define which YAML fields map to the available detail view? Is there a field-by-field mapping from `pipeline.Pipeline` struct to `AvailableDetail`? [Completeness]
- [ ] CHK050 - Does the spec address pipelines with no description, no category, no inputs, or no dependencies? Are all optional fields handled with defined fallback text? [Coverage]
- [ ] CHK051 - Is the "input source and example" in FR-003 traceable to a specific YAML field? Does the pipeline manifest actually have an `input.example` field? [Completeness]
- [ ] CHK052 - Does the spec define how step personas are resolved — from the step's `persona` field, or from a persona lookup in the manifest? What if the persona reference is invalid? [Coverage]

## Finished Pipeline Data

- [ ] CHK053 - Are all state store queries required for the finished detail view enumerated (GetRun, GetPerformanceMetrics, GetArtifacts)? Are there any queries missing (e.g., GetStepStates for skipped steps)? [Completeness]
- [ ] CHK054 - Does the spec define how `FailedStep` is derived — first failed step in execution order, or the step that caused the pipeline to fail (could differ with parallel steps)? [Clarity]
- [ ] CHK055 - Is the duration computation clearly defined — `CompletedAt - StartedAt` for completed, `CancelledAt - StartedAt` for cancelled, and what for failed (which may not have a `CompletedAt`)? [Completeness]
- [ ] CHK056 - Does the spec define the artifact type taxonomy? What values can `ArtifactInfo.Type` take, and are they sourced from the state store or inferred? [Completeness]
- [ ] CHK057 - Is there a requirement for how step results are ordered in the finished detail — by execution order, by step definition order, or alphabetically? [Clarity]

## Async Fetching & State Transitions

- [ ] CHK058 - Does the spec define the state machine for the detail model (idle → loading → loaded/error)? Are all transitions explicitly enumerated? [Completeness]
- [ ] CHK059 - Is there a requirement for cancellation of in-flight fetches when the user selects a different pipeline before the previous fetch completes? [Coverage]
- [ ] CHK060 - Does the spec address the race condition where `DetailDataMsg` arrives for a pipeline that is no longer selected? Should the message be discarded? [Coverage]
