# State Machine Quality Review: Optional Pipeline Steps

**Feature**: 118-optional-pipeline-steps
**Date**: 2026-02-20
**Focus**: Quality of state machine requirements for the new `failed_optional` and `skipped` states

---

## State Definition Completeness

- [ ] CHK101 - Are all valid source states for transitioning TO `failed_optional` explicitly enumerated (Running only, or also Retrying)? [Completeness]
- [ ] CHK102 - Are all valid transitions OUT of `failed_optional` specified (terminal state — should explicitly state no outgoing transitions)? [Completeness]
- [ ] CHK103 - Is the `skipped` state explicitly defined as a StepState constant in `internal/state/store.go`, or does the spec rely on the display-only constant from `display.StateSkipped`? [Completeness]
- [ ] CHK104 - Are all valid source states for transitioning TO `skipped` enumerated (Pending only, since the step never starts executing)? [Completeness]
- [ ] CHK105 - Does the spec define whether `skipped` is a terminal state (no outgoing transitions) or whether it can transition on resume? [Completeness]

---

## State Persistence Clarity

- [ ] CHK106 - Is it specified whether `SaveStepState` should accept `"skipped"` as a valid StepState, or is `"skipped"` only tracked in the in-memory `execution.States` map? [Clarity]
- [ ] CHK107 - Does the spec define which database columns are populated for a `skipped` step (started_at? completed_at? error_message?)? [Clarity]
- [ ] CHK108 - Is the error_message format for `failed_optional` steps defined — should it contain the original adapter error, or a wrapper message? [Clarity]
- [ ] CHK109 - Are `GetStepStates` and `GetStepState` query behaviors for `"failed_optional"` specified (do they return it in the same result set as other states)? [Clarity]

---

## State Machine Consistency

- [ ] CHK110 - Does the state machine in data-model.md show `Pending -> Skipped` as a direct transition, matching the behavior described in tasks T015? [Consistency]
- [ ] CHK111 - Is the Retrying -> FailedOptional transition shown in the state diagram, given that FR-011 specifies retries are exhausted before marking failed_optional? [Consistency]
- [ ] CHK112 - Do all packages that define state constants (pipeline/types.go, state/store.go, event/emitter.go, display/types.go) agree on the exact string value `"failed_optional"`? [Consistency]
- [ ] CHK113 - Is the `GetStatus` switch statement update (T007) consistent with the state values used in the execution loop (T011)? [Consistency]

---

## State Machine Coverage

- [ ] CHK114 - Does the spec address what happens if `SaveStepState` is called with `StateFailedOptional` for a step that was never in `StateRunning` (defensive programming edge case)? [Coverage]
- [ ] CHK115 - Is there a requirement for state store queries that filter by state — do existing queries like "get all failed steps" need updating to include or exclude `failed_optional`? [Coverage]
- [ ] CHK116 - Does the spec address state machine behavior for cancelled pipelines — if a pipeline is cancelled while an optional step is running, what state does it get? [Coverage]
