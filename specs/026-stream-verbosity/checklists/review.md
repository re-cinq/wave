# Requirements Quality Review: Stream Verbosity (026)

**Feature**: Stream Verbosity | **Date**: 2026-02-09 | **Spec**: `specs/026-stream-verbosity/spec.md`

---

## Completeness

- [ ] CHK001 - Does any requirement specify what happens when the ThrottledProgressEmitter holds a pending stream_activity event but no further events arrive (e.g., the step completes or goes idle)? The data-model mentions "the throttle window expires" but the plan rejects timer-based flushing, and the contract defines no Close/Flush lifecycle method. Could the last tool call of a step be silently lost? [Completeness]

- [ ] CHK002 - Is a Close/Shutdown/Flush method specified for ThrottledProgressEmitter to drain pending events when a step or pipeline completes? The ProgressEmitter interface defines only EmitProgress, and no lifecycle management is documented in any artifact. [Completeness]

- [ ] CHK003 - Does FR-006 or its contract specify behavior when stream_activity events arrive from multiple concurrent steps simultaneously? The contract describes "most-recent-wins" coalescing but does not clarify whether a single ThrottledProgressEmitter instance handles all steps (potentially losing interleaved step context) or whether separate instances exist per step. [Completeness]

- [ ] CHK004 - Are truncation or length limits specified for tool targets extracted by the heuristic fallback? Bash commands are truncated to 60 chars, but no limits are stated for file_path, url, pattern, query, or notebook_path targets which could be arbitrarily long. [Completeness]

- [ ] CHK005 - Does any requirement specify what value the Model field should contain when a persona does not explicitly configure a model (i.e., it falls back to the adapter's default model)? Is an empty value acceptable for FR-010, or must a resolved model always be present? [Completeness]

- [ ] CHK006 - Does any requirement address what happens when extractToolTarget receives a valid tool name but malformed (non-JSON-object) input? The contract says "MUST NOT panic on nil/empty input" but does not mention non-object JSON input (raw string, number, array). [Completeness]

- [ ] CHK007 - Is there a requirement covering the scenario where multiple result events appear in a single stream (e.g., retry within a single adapter invocation)? Only "no final-result before process exit" and "final-result indicates error" are documented as edge cases. [Completeness]

- [ ] CHK008 - FR-001, FR-003, and FR-007 are listed as "already implemented" but have NO audit tasks or test-coverage tasks. Should Phase 2 include verification tasks for these, consistent with how FR-002/FR-004/FR-005/FR-008/FR-012 are audited? [Completeness]

- [ ] CHK009 - Edge case 3 (final-result event indicates error) has no explicit task. T004 tests result accumulation for success cases and T005 tests subprocess termination, but neither specifically tests a well-formed "result" event carrying an error indication with exit code propagation. [Completeness]

- [ ] CHK010 - Edge case 5 (terminal resize during rendering) is listed in the spec but has no corresponding task, requirement, or test. Should the plan document whether existing display code already handles this, or whether the new ThrottledProgressEmitter needs resize safety? [Completeness]

- [ ] CHK011 - Edge case 8 (non-streaming adapters continue with buffered output) has no task coverage. Should there be a test confirming that adapters not invoking OnStreamEvent produce no stream_activity events and that the ThrottledProgressEmitter handles receiving zero stream_activity events? [Completeness]

---

## Clarity

- [ ] CHK012 - The tool-target-extraction contract specifies function signature `func extractToolTarget(toolName string, input map[string]json.RawMessage) string` but the actual code and data-model show `func extractToolTarget(toolName string, input json.RawMessage) string`. Which is canonical? Conflicting signatures could cause an implementer to change the function signature unnecessarily. [Clarity]

- [ ] CHK013 - The research.md names the new Event field `AdapterType` with JSON tag `"adapter_type"`, while plan.md, data-model.md, step-start-event contract, and tasks.md all use `Adapter` with JSON tag `"adapter"`. Which is authoritative? Two implementers working from different documents would produce incompatible field names. [Clarity]

- [ ] CHK014 - The data-model says pending events are "forwarded when the next event arrives or the throttle window expires" (implying timer-based expiry), but research.md explicitly rejects timer-based flushing and the plan describes only lazy forwarding. Is the throttle window self-expiring or purely lazy (event-driven)? [Clarity]

- [ ] CHK015 - FR-011 and the progress-event contract state "The field MUST be present in the event schema (not omitempty) for forward compatibility." However, the existing code uses `json:"estimated_time_ms,omitempty"` and task T018 says "consider whether to change." Is removing omitempty a mandatory requirement or optional? The exploratory language in T018 contradicts the mandatory language in the contract. [Clarity]

- [ ] CHK016 - The research.md heuristic priority list includes 8 fields (adding `path` and `description`) while all other artifacts (plan, data-model, contract, tasks) list exactly 6 fields. Which is the authoritative list for the generic heuristic? [Clarity]

- [ ] CHK017 - FR-006 specifies "1 event per second" throttling but does not clarify whether the 1-second window is measured from the last forwarded event (sliding window) or from fixed wall-clock intervals. The data-model implies sliding window; the research mentions "end of each 1-second interval" suggesting fixed intervals. [Clarity]

- [ ] CHK018 - Are Model and Adapter fields populated only on the initial step-start emission (first event with State == "running"), or on every event with State == "running"? The step-start contract specifies "when State == 'running' and emitted at step start" but "running" is a general state used beyond initial emission. [Clarity]

---

## Consistency

- [ ] CHK019 - The research.md heuristic fallback includes 8 fields (file_path, url, pattern, command, query, notebook_path, path, description) while all other artifacts list exactly 6 fields (omitting path and description). This is a direct contradiction between the research output and downstream design documents. [Consistency]

- [ ] CHK020 - The research.md uses field name `AdapterType` / `adapter_type` while plan.md, data-model.md, and contracts all use `Adapter` / `adapter`. These are incompatible field names that would produce different JSON output. [Consistency]

- [ ] CHK021 - The data-model shows ProgressEmitter interface in `internal/event/emitter.go` while the throttled-emitter contract says it implements ProgressEmitter from `internal/display/types.go`. Which package owns the interface? This affects import paths in the new ThrottledProgressEmitter file. [Consistency]

- [ ] CHK022 - The plan.md says "No new dependencies" and "No changes to AdapterRunner interface" as constraints, but no task in tasks.md explicitly verifies either constraint. Should there be a verification step? [Consistency]

- [ ] CHK023 - Task T013 says to apply ThrottledProgressEmitter to "text, quiet, auto/TTY, and auto/non-TTY formats" but QuietProgressDisplay already suppresses all stream_activity events. Wrapping QuietProgressDisplay with throttling is redundant and not addressed by the contract or plan. Is double-suppression intentional? [Consistency]

---

## Coverage

- [ ] CHK024 - The spec lists 7 success criteria (SC-001 through SC-007) but neither the plan nor tasks trace any task to specific success criteria. How will the team verify all success criteria are met at delivery? [Coverage]

- [ ] CHK025 - SC-001 ("tool call visible within 1 second of subprocess emission") is a measurable latency criterion, but no task describes how to instrument or verify this end-to-end latency measurement in automated tests. [Coverage]

- [ ] CHK026 - US3 acceptance scenario 3.1 (concurrent step events carry step ID and persona for disambiguation) is mapped to T009 but T009 does not explicitly test concurrent step interleaving. Is the concurrent disambiguation behavior tested? [Coverage]

- [ ] CHK027 - US4 acceptance scenario 4.2 (basic text mode: events rendered with timestamps at throttled rate) is not explicitly called out in T012's test list. Is non-TTY/basic text mode throttle behavior covered? [Coverage]

- [ ] CHK028 - The step-start-event contract invariant 3 states "Non-streaming adapters still include Model and Adapter fields." No task verifies that non-streaming adapter step-start events populate these fields. [Coverage]

- [ ] CHK029 - Edge case 2 (extremely long lines >10MB) specifies "configurable max buffer" but the existing code hardcodes `10*1024*1024`. No requirement, task, or test addresses making the buffer size configurable. Does the edge case description match reality? [Coverage]

- [ ] CHK030 - SC-002 ("at most 1 tool call event/sec during bursts") is the primary metric for the throttled emitter. T012 tests with short configurable intervals (10ms). Is the actual 1-second behavior ever verified, or only the coalescing logic with accelerated intervals? [Coverage]

---

## Testability

- [ ] CHK031 - The throttled-emitter contract specifies "at most 1 emission per throttleInterval" but defines no measurable tolerance for timing assertions in tests. With configurable short intervals (10ms) for fast tests, what tolerance is acceptable for scheduling jitter? [Testability]

- [ ] CHK032 - FR-011 requires EstimatedTimeMs to be present as 0 in events, but the current code uses `omitempty` which suppresses zero values in JSON. If the tag is not changed, how would a test verify the field is "present" when Go's encoding/json omits it? Testability depends on an unresolved design decision (CHK015). [Testability]

- [ ] CHK033 - T014 requires testing that display formats wrap their progress display with ThrottledProgressEmitter, but CreateEmitter returns an opaque EmitterResult. The task does not specify an assertion strategy (type assertion, mock injection, or behavioral testing). How is internal wrapping verified without exposing implementation details? [Testability]

- [ ] CHK034 - T009 (event bridge test) requires verifying the executor enriches events with pipeline context, but executor tests typically require substantial setup (mock adapters, pipeline configs). No guidance is provided on test setup strategy or fixture design, which could lead to incomplete or fragile tests. [Testability]
