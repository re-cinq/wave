# Tasks: Stream Verbosity (026)

**Branch**: `026-stream-verbosity` | **Generated**: 2026-02-09
**Spec**: `specs/026-stream-verbosity/spec.md` | **Plan**: `specs/026-stream-verbosity/plan.md`

## Overview

5 user stories, 5 phases, 20 tasks. The codebase already implements ~70-75% of the streaming infrastructure (FR-001 through FR-005, FR-007, FR-008, FR-012, FR-013). Tasks target the remaining gaps: display-layer throttling (FR-006), extended tool target extraction (FR-009), step-start metadata (FR-010), ETA field plumbing (FR-011), and comprehensive test coverage for both existing and new code.

---

## Phase 1: Setup & Infrastructure

- [X] T001 [P1] Setup: Verify existing streaming infrastructure compiles and passes tests
  - Run `go test ./internal/adapter/... ./internal/event/... ./internal/pipeline/... ./internal/display/... -race` from project root
  - Confirm all existing tests pass before making any changes
  - File: (project root — no file changes)

---

## Phase 2: US2 — Streaming Adapter Support (P1, foundational prerequisite)

These tasks audit and add test coverage for existing streaming adapter functionality. No production code changes — verification only.

- [X] T002 [P1] [US2] [P] Add table-driven tests for `parseStreamLine()` covering all event types (system, assistant/tool_use, assistant/text, tool_result, result)
  - Test each NDJSON event type produces the correct `StreamEvent` fields
  - Test malformed JSON lines return `(zero, false)` (FR-008)
  - Test extremely long lines (>1MB) are handled without panic
  - File: `internal/adapter/claude_test.go`

- [X] T003 [P1] [US2] [P] Add tests for `OnStreamEvent` callback invocation during stream scanning
  - Verify callback receives `StreamEvent` with correct Type, ToolName, ToolInput for tool_use events
  - Verify callback is NOT invoked for non-tool_use events (tool_result, text, system)
  - Verify callback is NOT invoked when `OnStreamEvent` is nil
  - File: `internal/adapter/claude_test.go`

- [X] T004 [P1] [US2] [P] Add tests for result accumulation from stream completion events (FR-004)
  - Verify `AdapterResult` contains correct `TokensUsed` and `ResultContent` after stream processing
  - Verify result extraction works when the "result" event is preceded by multiple tool_use events
  - File: `internal/adapter/claude_test.go`

- [X] T005 [P1] [US2] Add test for subprocess termination mid-stream (FR-012)
  - Verify adapter returns partial accumulated results and an error when the process exits unexpectedly
  - Verify no panic or hang when the stream is interrupted
  - File: `internal/adapter/claude_test.go`

---

## Phase 3: US1 — Real-Time Tool Call Visibility & US3 — Event Bridge (P1/P2)

### Sub-phase 3A: Extended Tool Target Extraction (FR-009)

- [X] T006 [P1] [US1] Add explicit `case` entries to `extractToolTarget()` for WebFetch, WebSearch, NotebookEdit
  - `WebFetch` → extract `url` field
  - `WebSearch` → extract `query` field
  - `NotebookEdit` → extract `notebook_path` field
  - File: `internal/adapter/claude.go` (lines 420-449, extend the `switch` statement)

- [X] T007 [P1] [US1] Replace empty `default` case in `extractToolTarget()` with generic heuristic fallback
  - Check input JSON fields in priority order: `file_path`, `url`, `pattern`, `command`, `query`, `notebook_path`
  - Return first non-empty match; empty string if none found
  - File: `internal/adapter/claude.go` (line 446-448, replace the `default` case)

- [X] T008 [P1] [US1] [P] Add table-driven tests for `extractToolTarget()` covering all 10 explicit tools + heuristic
  - Test all 7 existing tools (Read, Write, Edit, Glob, Grep, Bash, Task)
  - Test 3 new tools (WebFetch, WebSearch, NotebookEdit)
  - Test generic heuristic: unknown tool with `file_path`, unknown tool with `url`, unknown tool with `query`
  - Test heuristic priority order: unknown tool with both `url` and `file_path` returns `file_path`
  - Test unknown tool with no matching fields returns empty string
  - Test nil/empty input returns empty string without panic
  - Test Bash command truncation at 60 chars with "..." suffix
  - Contract: `specs/026-stream-verbosity/contracts/tool-target-extraction.md`
  - File: `internal/adapter/claude_test.go`

### Sub-phase 3B: Event Bridge Test Coverage (FR-005)

- [X] T009 [P2] [US3] [P] Add test for stream-activity event bridge in executor
  - Verify that `OnStreamEvent` closure in executor emits `Event{State: "stream_activity"}` with correct PipelineID, StepID, Persona, ToolName, ToolTarget
  - Verify that non-tool_use stream events are NOT emitted as stream_activity events
  - Verify that tool_use events with empty ToolName are NOT emitted
  - File: `internal/pipeline/executor_test.go` (or new test file if executor_test.go structure requires it)

---

## Phase 4: US4 — Throttled Display Rendering (P2)

- [X] T010 [P2] [US4] Create `ThrottledProgressEmitter` struct implementing `ProgressEmitter` interface
  - Fields: `inner ProgressEmitter`, `mu sync.Mutex`, `lastStreamActivityTime time.Time`, `pendingStreamActivity *event.Event`, `throttleInterval time.Duration`
  - Constructor: `NewThrottledProgressEmitter(inner ProgressEmitter) *ThrottledProgressEmitter`
  - Constructor with interval: `NewThrottledProgressEmitterWithInterval(inner ProgressEmitter, interval time.Duration) *ThrottledProgressEmitter`
  - Contract: `specs/026-stream-verbosity/contracts/throttled-emitter.md`
  - File: `internal/display/throttled_emitter.go` (new file)

- [X] T011 [P2] [US4] Implement `EmitProgress()` method on `ThrottledProgressEmitter`
  - Non-stream_activity events: forward immediately to inner emitter; flush pending stream_activity event first if one exists
  - stream_activity events: if >= throttleInterval since last emission, forward immediately and update timestamp; otherwise store as pending (most-recent-wins coalescing)
  - First stream_activity event in any window must be emitted immediately (no initial delay)
  - Thread-safe with sync.Mutex
  - Depends on: T010
  - File: `internal/display/throttled_emitter.go`

- [X] T012 [P2] [US4] [P] Add unit tests for `ThrottledProgressEmitter`
  - Test: first stream_activity event passes through immediately
  - Test: events within 1-second window are coalesced (only most recent forwarded at next opportunity)
  - Test: non-stream_activity events pass through immediately regardless of throttle state
  - Test: pending stream_activity event is flushed when non-stream_activity event arrives
  - Test: concurrent access with `-race` flag
  - Test: configurable interval (use short interval like 10ms for fast tests)
  - Test: empty/nil inner emitter handling
  - Depends on: T011
  - Contract: `specs/026-stream-verbosity/contracts/throttled-emitter.md`
  - File: `internal/display/throttled_emitter_test.go` (new file)

- [X] T013 [P2] [US4] Wire `ThrottledProgressEmitter` into `CreateEmitter()` factory
  - Wrap progress displays with `NewThrottledProgressEmitter()` before passing to emitter constructors
  - Apply to: text format (line 59-61), quiet format (line 67-69), auto/TTY format (line 86-93), auto/non-TTY format (line 107-109)
  - Do NOT wrap for JSON format (no progress emitter, NDJSON only — FR-007)
  - Depends on: T010, T011
  - File: `cmd/wave/commands/output.go` (lines 50-113, `CreateEmitter()` and `createAutoEmitter()`)

- [X] T014 [P2] [US4] [P] Add tests for throttle wiring in `CreateEmitter()`
  - Verify text, quiet, and auto formats wrap their progress display with ThrottledProgressEmitter
  - Verify JSON format does NOT use ThrottledProgressEmitter
  - Depends on: T013
  - File: `cmd/wave/commands/output_test.go`

---

## Phase 5: US5 — Step Metadata in Events (P3)

- [X] T015 [P3] [US5] Add `Model` and `Adapter` fields to `Event` struct
  - Add `Model string \`json:"model,omitempty"\`` field
  - Add `Adapter string \`json:"adapter,omitempty"\`` field
  - Place after existing stream event fields (after ToolTarget)
  - Contract: `specs/026-stream-verbosity/contracts/step-start-event.md`
  - File: `internal/event/emitter.go` (lines 11-34, Event struct)

- [X] T016 [P3] [US5] Populate `Model` and `Adapter` fields in step-start event emission
  - Source `Model` from the adapter run config's model field (already available as `persona.Model` or resolved model)
  - Source `Adapter` from the adapter definition's identifier (e.g., `adapterDef.Binary` or normalized name)
  - Depends on: T015
  - File: `internal/pipeline/executor.go` (lines 382-390, the step-start event emission)

- [X] T017 [P3] [US5] [P] Add tests for step-start event metadata
  - Verify step-start events (State == "running") include Model and Adapter fields in NDJSON output
  - Verify fields are omitted when empty (omitempty behavior)
  - Verify non-step-start events do NOT include these fields
  - Depends on: T016
  - Contract: `specs/026-stream-verbosity/contracts/step-start-event.md`
  - File: `internal/event/emitter_test.go`

- [X] T018 [P3] [US5] Verify ETA field plumbing in progress heartbeat events (FR-011)
  - Confirm `EstimatedTimeMs` field is present in heartbeat events emitted by `startProgressTicker()`
  - If zero-value is omitted by `omitempty`, consider whether to change the JSON tag to ensure the field is always present per the progress-event contract
  - Depends on: T015
  - Contract: `specs/026-stream-verbosity/contracts/progress-event.md`
  - File: `internal/pipeline/executor.go` (lines 886-912, startProgressTicker) and `internal/event/emitter.go` (EstimatedTimeMs field tag)

- [X] T019 [P3] [US5] [P] Add tests for ETA field presence in progress events
  - Verify heartbeat events include `estimated_time_ms` field in JSON output (value: 0)
  - Verify the field is present even when value is zero (forward-compatibility requirement)
  - Depends on: T018
  - Contract: `specs/026-stream-verbosity/contracts/progress-event.md`
  - File: `internal/event/emitter_test.go`

---

## Phase 6: Final — Integration Verification & Polish

- [X] T020 [P1] Final: Run full test suite with race detector and verify all tests pass
  - Run `go test ./... -race` from project root
  - Verify no race conditions in new ThrottledProgressEmitter
  - Verify no regressions in existing pipeline execution tests
  - Verify new tests all pass
  - Depends on: all previous tasks
  - File: (project root — no file changes)

---

## Dependency Graph

```
T001 (verify baseline)
  |
  +---> T002, T003, T004, T005 (US2: adapter test coverage, parallelizable)
  |
  +---> T006, T007 (US1: tool target extraction, sequential)
  |       |
  |       +---> T008 (US1: tool target tests, after T006+T007)
  |
  +---> T009 (US3: event bridge tests, parallelizable with above)
  |
  +---> T010 (US4: throttled emitter struct)
          |
          +---> T011 (US4: EmitProgress implementation)
                  |
                  +---> T012 (US4: throttle tests)
                  +---> T013 (US4: wiring in CreateEmitter)
                          |
                          +---> T014 (US4: wiring tests)
  |
  +---> T015 (US5: Event struct fields)
          |
          +---> T016 (US5: populate in executor)
          |       |
          |       +---> T017 (US5: step-start metadata tests)
          |
          +---> T018 (US5: ETA field plumbing)
                  |
                  +---> T019 (US5: ETA tests)

T020 (final: full test suite) — depends on ALL above
```

## Parallel Opportunities

Tasks marked with `[P]` can be executed in parallel with other tasks in their phase:

| Phase | Parallel Tasks | Notes |
|-------|---------------|-------|
| Phase 2 | T002, T003, T004 | Independent test additions, no production code changes |
| Phase 3 | T008, T009 | T008 after T006+T007; T009 independent |
| Phase 4 | T012, T014 | Independent test files |
| Phase 5 | T017, T019 | Independent test additions |

## Requirement Traceability

| Task | FR(s) | User Story |
|------|-------|------------|
| T001 | All | — |
| T002 | FR-002, FR-008 | US2 |
| T003 | FR-005 | US2 |
| T004 | FR-004 | US2 |
| T005 | FR-012 | US2 |
| T006 | FR-009 | US1 |
| T007 | FR-009 | US1 |
| T008 | FR-009 | US1 |
| T009 | FR-005, FR-013 | US3 |
| T010 | FR-006 | US4 |
| T011 | FR-006 | US4 |
| T012 | FR-006 | US4 |
| T013 | FR-006 | US4 |
| T014 | FR-006 | US4 |
| T015 | FR-010 | US5 |
| T016 | FR-010 | US5 |
| T017 | FR-010 | US5 |
| T018 | FR-011 | US5 |
| T019 | FR-011 | US5 |
| T020 | All | — |
