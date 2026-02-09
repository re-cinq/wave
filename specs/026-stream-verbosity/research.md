# Research: Stream Verbosity (026)

**Branch**: `026-stream-verbosity` | **Date**: 2026-02-09 | **Phase**: 0

## Overview

Wave's streaming adapter (`ClaudeAdapter`) already implements ~70-75% of the stream verbosity feature. The adapter uses `--output-format stream-json`, parses NDJSON line-by-line via `bufio.Scanner`, extracts tool invocations, and emits `stream_activity` events through the event bridge wired in `executor.go`. The remaining work targets four gaps: display-layer throttling, extended tool target extraction, step metadata fields, and ETA schema plumbing.

This research documents design decisions for each gap, grounded in the existing codebase.

---

## 1. Throttling Strategy

### Decision

Introduce a `ThrottledProgressEmitter` wrapper in `internal/display/` that implements the existing `event.ProgressEmitter` interface. It decorates an inner `ProgressEmitter` and applies per-event-type throttling:

- `stream_activity` events: coalesced using a 1-second sliding window with most-recent-wins semantics. When a new `stream_activity` event arrives within the window, it replaces the pending event. At the end of each 1-second interval, the most recent event is forwarded to the inner emitter.
- All other event types (`started`, `running`, `completed`, `failed`, `step_progress`, `contract_validating`, etc.): passed through immediately with no throttling.

The `NDJSONEmitter` continues to write all events to stdout without any throttling, preserving FR-007.

### Rationale

- **Clean separation of concerns.** Throttling is a human-display concern, not an event-emission concern. The existing architecture already separates programmatic output (`NDJSONEmitter` to stdout) from display output (`ProgressEmitter` to stderr). The throttle wrapper slots into the display path without touching the programmatic path.
- **Existing 200ms refresh rate is unrelated.** `ProgressDisplay.refreshRate` (line 280 of `progress.go`) governs overall rendering frequency -- how often the terminal is repainted. The throttle wrapper governs how often individual `stream_activity` events reach the display layer at all. These are orthogonal concerns and must remain separate.
- **Correct integration point.** The wrapper is injected where `NewNDJSONEmitterWithProgress` or `NewProgressOnlyEmitter` receives a `ProgressEmitter` argument. The caller passes `ThrottledProgressEmitter(innerDisplay)` instead of `innerDisplay` directly.

### Alternatives Rejected

| Alternative | Reason Rejected |
|---|---|
| Timer-based goroutine flushing on a tick | Adds concurrency complexity (goroutine lifecycle, timer cleanup). A sliding window with lazy forwarding on the next event or a deferred flush achieves the same result with simpler control flow. |
| Channel-based event buffering | Over-engineered. A single `sync.Mutex` protecting the pending event and last-emit timestamp is sufficient. Channel-based designs introduce backpressure and shutdown concerns. |
| Throttling inside `NDJSONEmitter.Emit()` | Violates FR-007. Programmatic NDJSON output must be unthrottled. The dual-stream design explicitly separates these paths. |
| Modifying `ProgressDisplay.render()` throttle | The 200ms render throttle is for terminal repaint frequency. Adding event-type filtering there conflates two concerns and would not work for `BasicProgressDisplay` or `BubbleTeaProgressDisplay`. |

---

## 2. Tool Target Extraction Heuristic

### Decision

Extend the `extractToolTarget` function in `internal/adapter/claude.go` with three additions:

1. **Explicit mappings for known tools not yet handled:**
   - `WebFetch` -> `url` field
   - `WebSearch` -> `query` field
   - `NotebookEdit` -> `notebook_path` field

2. **Generic heuristic fallback for unrecognized tools.** After the explicit `switch` cases, check the tool's input JSON for well-known field names in priority order:
   ```
   file_path > url > pattern > command > query > notebook_path > path > description
   ```
   Return the first non-empty match. This covers future tools that follow common parameter naming conventions without requiring code changes.

3. **Unrecognized tool with no heuristic match.** Return empty string (the event is still emitted with the tool name alone, per the spec's edge case for unrecognized tools).

### Rationale

- **Maximizes display value.** WebFetch, WebSearch, and NotebookEdit are all tools available in Claude Code's standard tool set (visible in this agent's own tool list). Adding explicit mappings ensures operators see meaningful targets immediately.
- **Forward-compatible.** The generic heuristic means new tools added upstream (e.g., TodoWrite, Skill) will display targets if they use standard field names, without requiring a Wave release.
- **The existing code structure supports this cleanly.** The `extractToolTarget` function is a single `switch` with a `default` case that currently returns empty string. Adding three cases and a heuristic fallback in the `default` branch is a minimal, self-contained change.

### Alternatives Rejected

| Alternative | Reason Rejected |
|---|---|
| Only explicit mappings, no heuristic | Breaks every time a new tool is added upstream. Wave would need a code release for each new Claude Code tool. |
| Regex-based input parsing | Fragile. Tool inputs are structured JSON, not free text. JSON field lookup is both simpler and more reliable. |
| Extracting all fields as a concatenated summary | Too noisy for display. The point is a single, meaningful target per tool invocation. |

---

## 3. Event Schema Extension (Step Metadata)

### Decision

Add two new string fields to the `event.Event` struct in `internal/event/emitter.go`:

```go
Model       string `json:"model,omitempty"`        // Model used for this step
AdapterType string `json:"adapter_type,omitempty"` // Adapter type (e.g., "claude", "opencode")
```

Populate these fields in the executor's step-start event (the `"running"` state emission at step begin). The model value comes from `persona.Model` (already available in `AdapterRunConfig.Model`) and the adapter type comes from `adapterDef.Binary` or a normalized adapter name.

### Rationale

- **Minimal schema change.** Two optional fields with `omitempty` tags. Old consumers that do not expect these fields will ignore them in JSON parsing. Existing event processing code (`EmitProgress`, display handlers) does not need modification -- they simply do not read these fields unless updated.
- **Data is already available.** The executor already has `persona.Model` and `adapterDef.Binary` in scope when it emits the step-start event (see `executor.go` lines 424-450). No new data fetching or plumbing is required.
- **Consistent with existing field patterns.** The Event struct already has optional fields like `Persona`, `ToolName`, `ToolTarget` with `omitempty`. Model and AdapterType follow the same pattern.

### Alternatives Rejected

| Alternative | Reason Rejected |
|---|---|
| Separate metadata event type (e.g., `StateStepMetadata`) | Over-engineered. Multiplies event types without adding value. Consumers would need to correlate metadata events with step-start events. Embedding in the existing step-start event is simpler and self-contained. |
| Embedding full adapter config object | Leaks implementation details (temperature, timeout, env vars) into the event stream. The operator needs model and adapter type for monitoring, not the full config. |
| Adding to `PipelineContext` only, not `Event` | Would make metadata invisible to NDJSON consumers. The spec requires metadata in events for both display and programmatic output. |

---

## 4. ETA Field Approach

### Decision

The `EstimatedTimeMs` field already exists in `event.Event` (line 27 of `emitter.go`) and `PipelineContext` (line 207 of `types.go`). The `ProgressCalculator` in `progress.go` already implements `CalculateETA` using average step duration. For the initial implementation:

- Ensure progress heartbeat events include `EstimatedTimeMs` (currently always zero because no historical data feeds the calculator).
- Do **not** build a duration history store or add `expected_duration` YAML fields yet.
- Document the forward-compatible path: future iteration adds either a SQLite-backed duration history table (recording actual step durations on completion) or an `expected_duration` field in `wave.yaml` step/persona definitions, or both.

### Rationale

- **P3 priority, disproportionate scope.** FR-011 is part of User Story 5 (P3 -- "polish that improves observability"). Building a full duration history store requires new SQLite migrations, data collection on step completion, and statistical estimation logic. This is an entire feature in its own right.
- **Schema is already forward-compatible.** The `EstimatedTimeMs` field exists in both `Event` and `PipelineContext`. The `ProgressCalculator.CalculateETA` method exists and works correctly when fed average step times. The only missing piece is the data source, which can be added incrementally.
- **Zero is honest.** Per acceptance scenario 5.3: "the estimated time field is zero or omitted (not a fabricated estimate)." Hardcoded guesses would be misleading and erode operator trust.

### Alternatives Rejected

| Alternative | Reason Rejected |
|---|---|
| Full SQLite duration history store now | Scope creep. Requires a new migration, step-completion hooks to record durations, and statistical aggregation. Not justified for a P3 feature in the initial implementation. |
| Hardcoded estimates per persona | Inaccurate and misleading. Step durations vary wildly based on prompt complexity, model choice, and external factors. Fixed estimates would be worse than no estimate. |
| Skip the field entirely | The field already exists. Ensuring it is present (as zero) in heartbeat events costs nothing and maintains forward compatibility for future iterations. |

---

## 5. Testing Strategy

### Decision

Unit tests for each new component, following Go table-driven test conventions:

1. **ThrottledProgressEmitter tests** (`internal/display/throttled_emitter_test.go`):
   - Verify `stream_activity` events are coalesced within 1-second windows (most-recent-wins).
   - Verify non-`stream_activity` events pass through immediately with no delay.
   - Verify concurrent event emission is safe (test with `-race`).
   - Verify the first `stream_activity` event in a window is forwarded immediately (no unnecessary delay for sparse events).

2. **Extended `extractToolTarget` tests** (`internal/adapter/claude_test.go`):
   - Table-driven tests covering WebFetch, WebSearch, NotebookEdit with their respective fields.
   - Tests for the generic heuristic fallback with unknown tool names and standard field names.
   - Tests for unknown tools with no recognizable fields (returns empty string).

3. **Step metadata tests** (`internal/pipeline/executor_test.go` or `internal/event/emitter_test.go`):
   - Verify step-start events include Model and AdapterType fields.
   - Verify fields are omitted (empty) when not set (`omitempty` behavior).

4. **ETA field tests**:
   - Existing `ProgressCalculator` tests already cover ETA calculation logic.
   - Add a test verifying heartbeat events include `EstimatedTimeMs: 0` when no historical data is available.

### Rationale

- **Per constitution Principle 13**: "Tests are guardrails." Per CLAUDE.md: "Every failing test is the concern of the worker who caused it."
- **Table-driven tests** are the Go convention and provide clear coverage of input/output combinations.
- **Race detector** is required per CLAUDE.md testing requirements (`go test -race ./...`).
- **Unit tests over integration tests** for these components. The ThrottledProgressEmitter, extractToolTarget, and event schema changes are all unit-testable in isolation without subprocess execution.

### Alternatives Rejected

| Alternative | Reason Rejected |
|---|---|
| Integration-only tests (run full pipeline, check output) | Too slow, too many variables, hard to isolate failures. A throttling bug should not require a full pipeline execution to detect. |
| Skip tests for P3 features (ETA plumbing) | Violates test ownership principle. Even a zero-value field should have a test confirming it is present and correctly typed. |
| Mock-heavy tests with interface indirection | The components being tested are concrete and self-contained. Mocking is appropriate for the inner `ProgressEmitter` in throttle tests but unnecessary for `extractToolTarget` or event struct changes. |

---

## 6. Thread Safety

### Decision

`ThrottledProgressEmitter` uses `sync.Mutex` to protect its internal state (pending event, last-emit timestamp). All public methods acquire the lock before reading or writing shared state.

### Rationale

- **Concurrent pipeline steps.** Wave supports concurrent step execution. Multiple goroutines can emit `stream_activity` events simultaneously, all flowing through the same `ThrottledProgressEmitter` instance.
- **Existing pattern in codebase.** `NDJSONEmitter` (line 100-103 of `emitter.go`), `ProgressDisplay` (line 233 of `progress.go`), `BasicProgressDisplay` (line 523 of `progress.go`), and `BubbleTeaProgressDisplay` all use `sync.Mutex` for the same reason. The throttle wrapper follows the established convention.
- **Mutex is sufficient.** The critical section is small: check timestamp, update pending event, optionally forward to inner emitter. No long-running operations occur under the lock.

### Alternatives Rejected

| Alternative | Reason Rejected |
|---|---|
| Atomic operations only (`sync/atomic`) | Insufficient for multi-field updates. The throttle needs to atomically check-and-update both the pending event and the last-emit timestamp. Atomic operations work for single values, not composite state. |
| Single-goroutine event loop with channel intake | Adds lifecycle complexity (goroutine startup, shutdown, drain). Potential deadlocks if the channel blocks. The mutex approach is simpler and proven in the existing codebase. |
| No synchronization (caller responsibility) | Violates the principle of least surprise. The existing emitter types are all goroutine-safe. A wrapper that silently breaks under concurrency would be a footgun. |

---

## Implementation Scope Summary

| FR | Status | Work Required |
|---|---|---|
| FR-001 through FR-005 | Done | Audit + test coverage |
| FR-006 | New | `ThrottledProgressEmitter` wrapper |
| FR-007 | Done | No change (NDJSONEmitter already unthrottled) |
| FR-008 | Done | Audit + test coverage |
| FR-009 | Partial | Extend `extractToolTarget` with 3 tools + generic heuristic |
| FR-010 | New | Add Model/AdapterType fields to Event, populate in executor |
| FR-011 | Partial | Verify zero-value plumbing in heartbeat events |
| FR-012 | Done | Audit + test coverage |
| FR-013 | Done | Audit + test coverage |

### Files to Modify

| File | Change |
|---|---|
| `internal/event/emitter.go` | Add `Model`, `AdapterType` fields to `Event` struct |
| `internal/adapter/claude.go` | Extend `extractToolTarget` with WebFetch, WebSearch, NotebookEdit + heuristic fallback |
| `internal/pipeline/executor.go` | Populate Model and AdapterType in step-start event |
| `internal/display/throttled_emitter.go` | New file: `ThrottledProgressEmitter` implementation |
| `internal/display/throttled_emitter_test.go` | New file: throttle unit tests |
| `internal/adapter/claude_test.go` | Add table-driven tests for extended tool target extraction |

### Files Unchanged

| File | Reason |
|---|---|
| `internal/adapter/adapter.go` | `StreamEvent` and `AdapterRunConfig` already correct |
| `internal/display/progress.go` | Throttling is handled by wrapper, not by modifying existing displays |
| `internal/display/bubbletea_progress.go` | Already handles `stream_activity` events in verbose mode |
