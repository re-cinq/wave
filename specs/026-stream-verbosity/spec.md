# Feature Specification: Stream Verbosity

**Feature Branch**: `026-stream-verbosity`
**Created**: 2026-02-09
**Status**: Draft (Clarified)
**Input**: User description: "Stream verbosity: Replace Claude Code --output-format json with stream-json for real-time NDJSON streaming of tool calls during pipeline execution. Add streaming adapter support, event bridge for tool call events, and throttled display rendering."

## Implementation Scope

This feature describes the **desired end state** for stream verbosity, not a greenfield implementation. The existing codebase already provides approximately 70-75% of the described functionality (streaming adapter with `stream-json`, NDJSON parsing, tool target extraction, event bridge wiring). Implementation MUST audit the existing code against each requirement and only build what is missing, rather than reimplementing existing working infrastructure.

**Already implemented** (audit and add test coverage):
- FR-001 through FR-005: Streaming adapter, NDJSON parsing, tool extraction, result accumulation, event bridge
- FR-008: Malformed line handling
- FR-009: Tool target extraction for Read, Write, Edit, Glob, Grep, Bash, Task
- FR-012: Subprocess termination handling
- FR-013: Step context in events

**Requires new implementation**:
- FR-006: Per-tool-activity throttling at 1-second rate with event coalescing in the display layer
- FR-009 extension: Tool target extraction for additional tool types (WebFetch, WebSearch, NotebookEdit) and generic heuristic fallback
- FR-010: Model and adapter type fields in step-start events
- FR-011: ETA field plumbing in progress events (populated with zero initially)

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Real-Time Tool Call Visibility (Priority: P1)

As a Wave user running a multi-step pipeline, I want to see what the AI agent is doing in real time (which files it reads, what commands it runs, what it writes) so that I no longer stare at a silent terminal for minutes with no indication of progress.

**Why this priority**: The core problem is user experience during long-running pipeline steps. Steps can run for 270+ seconds with zero visible activity. This story directly eliminates the "silent terminal" problem by surfacing tool call events as they happen.

**Independent Test**: Can be fully tested by running any pipeline step that uses multiple tools (e.g., a step that reads files, runs tests, and writes output) and verifying that each tool invocation appears in the terminal output within 1 second of being parsed from the subprocess stream.

**Acceptance Scenarios**:

1. **Given** a pipeline step is executing in a TTY terminal with default settings, **When** the AI agent invokes any recognized tool, **Then** a line appears on stderr showing the tool name and its primary target (file path, command, or pattern) within 1 second of the event being parsed.
2. **Given** a pipeline step is executing in programmatic output mode, **When** the AI agent invokes a tool, **Then** the tool call appears as a tool-activity event in the NDJSON output stream with tool name and tool target fields, with no throttling applied.
3. **Given** a pipeline step is executing and the AI agent makes 10 tool calls within 1 second, **When** those events reach the human-readable display, **Then** at most 1 tool call event is rendered per second, showing the most recent tool call.

---

### User Story 2 - Streaming Adapter Support (Priority: P1)

As a pipeline operator, I want Wave's adapter to use real-time streaming output instead of buffered single-result output so that tool call data is available in real time rather than only after the entire step completes.

**Why this priority**: This is a prerequisite for all other stories. Without switching from buffered output to streaming NDJSON, no real-time events are available.

**Independent Test**: Can be tested by running a single pipeline step and verifying that (a) the adapter requests streaming output from the underlying CLI tool, (b) stdout is processed line-by-line as NDJSON, and (c) the final result is still correctly extracted for artifact writing and contract validation.

**Acceptance Scenarios**:

1. **Given** an adapter that supports streaming is configured, **When** it builds the subprocess invocation, **Then** the adapter requests the streaming output format from the underlying CLI tool.
2. **Given** the adapter is executing a prompt, **When** the subprocess emits newline-delimited events on stdout, **Then** each event is parsed individually in real time (not buffered until process exit).
3. **Given** the adapter is processing a stream, **When** a final-result event is received, **Then** the output content, token usage, and cost data are extracted and returned in the adapter result for downstream processing.
4. **Given** the adapter is processing a stream, **When** a malformed line is encountered, **Then** the line is skipped without terminating the stream or the pipeline step.

---

### User Story 3 - Event Bridge for Tool Calls (Priority: P2)

As a pipeline developer building custom display integrations, I want tool call events from the AI subprocess to be emitted as structured events through Wave's event system so that I can build monitoring, logging, and visualization tools on top of the event stream.

**Why this priority**: The event bridge connects the adapter layer to the event/display layer. It enables both the built-in display (Story 1) and external integrations.

**Independent Test**: Can be tested by registering an event listener, running a pipeline step, and verifying that `stream_activity` events are emitted with correct tool name, tool target, pipeline ID, and step ID fields for each tool call.

**Acceptance Scenarios**:

1. **Given** a stream callback is configured for a pipeline step, **When** a tool invocation event is received from the stream, **Then** a tool-activity event is emitted containing the tool name (e.g., "Read", "Bash", "Write") and the tool's primary target.
2. **Given** the event bridge is active, **When** a tool-activity event is emitted, **Then** the event includes the correct pipeline ID, step ID, and persona fields from the executing step context.
3. **Given** the event bridge is active, **When** non-tool-invocation stream events arrive (e.g., initialization, text content, or tool results), **Then** they are not emitted as tool-activity events (they are handled separately or ignored).

---

### User Story 4 - Throttled Display Rendering (Priority: P2)

As a user watching pipeline execution in my terminal, I want tool call updates to be displayed at a readable pace (not flooding the screen) so that I can follow what the agent is doing without visual noise.

**Why this priority**: Without throttling, rapid tool call bursts (e.g., 10+ Grep calls in a second) would flood the terminal. Throttling ensures readability while maintaining real-time feedback.

**Independent Test**: Can be tested by simulating a burst of 20 stream_activity events within 1 second and verifying that the display renderer outputs at most 1-2 lines during that period, showing only the most recent event.

**Acceptance Scenarios**:

1. **Given** the display is in interactive TTY mode, **When** multiple tool call events arrive within the same 1-second window, **Then** only the most recent event is rendered at the next display tick (event coalescing).
2. **Given** the display is in basic text mode (non-TTY or explicit text output mode), **When** tool call events arrive, **Then** events are rendered with timestamps at the throttled rate of at most 1 per second.
3. **Given** the display is in quiet mode, **When** tool call events arrive, **Then** no tool call events are displayed (only final completion/failure).
4. **Given** tool call events are being throttled for human display, **When** the same events are emitted to the programmatic output stream, **Then** all events appear unthrottled (throttling applies only to the human-readable display).

---

### User Story 5 - Step Metadata in Events (Priority: P3)

As a pipeline operator monitoring execution, I want step-start events to include the model name and adapter type, and progress events to include estimated time remaining, so that I have full context about what is running and when it will finish.

**Why this priority**: Polish that improves observability. Not required for core streaming functionality but enhances the user experience for operators monitoring pipelines.

**Independent Test**: Can be tested by running a pipeline and checking that step-start events include model and adapter fields, and that progress events include an ETA field when historical duration data is available.

**Acceptance Scenarios**:

1. **Given** a pipeline step starts executing, **When** the "running" event is emitted, **Then** the event includes the model name and adapter type used for the step.
2. **Given** a pipeline step is in progress, **When** a progress heartbeat event is emitted, **Then** the event schema includes an estimated time remaining field. For initial implementation, this field is zero (no estimate available). When configured or historical duration data becomes available in a future iteration, this field reflects the calculated estimate.
3. **Given** no configured or historical duration data is available for a step, **When** a progress event is emitted, **Then** the estimated time field is zero or omitted (not a fabricated estimate).

---

### Edge Cases

- What happens when the AI subprocess exits unexpectedly mid-stream (crash, timeout, signal)?
  - The adapter MUST detect the broken pipe, stop scanning, and return whatever partial result was accumulated along with the error.
- What happens when the stream contains extremely long lines (>10MB)?
  - The NDJSON scanner MUST have a configurable maximum line buffer and skip lines that exceed it without crashing.
- What happens when the adapter receives a final-result event indicating an error?
  - The adapter MUST propagate the error message and non-zero exit code to the pipeline executor.
- What happens when no final-result event is received before the process exits?
  - The adapter MUST return an error indicating the stream was incomplete, along with any partial output accumulated.
- What happens when the display terminal is resized during rendering?
  - The display MUST adapt to the new terminal dimensions without corrupting output.
- What happens when programmatic output mode is used with streaming?
  - All tool-activity events MUST appear in the programmatic output at full fidelity (no throttling).
- What happens when multiple pipeline steps stream concurrently?
  - Each tool-activity event MUST include its step ID and persona so consumers can distinguish between interleaved events from different steps.
- What happens when the adapter does not support streaming (non-Claude adapters)?
  - Adapters that do not support streaming MUST continue to work with buffered output. Tool-activity events are simply not emitted for those adapters.
- What happens when the stream contains an unrecognized tool name?
  - The system MUST emit a tool-activity event with the raw tool name and no target, rather than silently dropping the event.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: Adapters that support streaming MUST request the real-time streaming output format from the underlying AI CLI tool, rather than the buffered single-result format.
- **FR-002**: System MUST parse subprocess stdout as newline-delimited JSON (NDJSON), processing each line individually as it arrives (not buffering until process completion).
- **FR-003**: System MUST extract tool invocation information (tool name and primary target) from stream events that represent tool calls.
- **FR-004**: System MUST accumulate the final result from the stream's completion event, preserving backward compatibility with artifact writing and contract validation.
- **FR-005**: System MUST emit a structured tool-activity event through the event system for each tool invocation, including pipeline ID, step ID, persona, tool name, and tool target.
- **FR-006**: System MUST throttle human-readable display of tool-activity events to a target maximum of 1 event per second, using event coalescing (most-recent-wins). Under degraded conditions (high system load), the system SHOULD maintain this rate on a best-effort basis. Throttling MUST be implemented in the display layer (as a wrapper around the existing ProgressEmitter) — the NDJSONEmitter continues to write all events to stdout immediately, and the ThrottledProgressEmitter coalesces stream_activity events at a 1-second interval while passing all other event types through immediately. This is separate from the existing display refresh rate (which governs overall rendering frequency).
- **FR-007**: System MUST NOT throttle tool-activity events in the programmatic output stream — all events are emitted at full fidelity for machine consumers.
- **FR-008**: System MUST handle malformed lines in the stream gracefully by skipping them without terminating the pipeline step.
- **FR-009**: System MUST extract the primary target for each known tool type: file path for file-oriented tools (Read, Write, Edit, NotebookEdit), search pattern for search tools (Glob, Grep), URL for web tools (WebFetch, WebSearch), truncated command summary for shell execution tools (Bash), and task description for orchestration tools (Task). For unrecognized tool types, the system MUST apply a generic heuristic that checks common input field names (file_path, url, pattern, command, query, notebook_path) in priority order before falling back to tool name alone. This ensures new tools added to upstream CLI tools display meaningful targets without requiring code changes.
- **FR-010**: System MUST include model name and adapter type in step-start events.
- **FR-011**: System MUST include an estimated time remaining field in progress heartbeat events. For initial implementation, this field MUST be present in the event schema but set to zero (indicating no estimate available). A future iteration MAY populate this field using an `expected_duration` YAML configuration field on steps/personas and/or a SQLite-backed duration history store that records actual step durations on completion. The event schema MUST be forward-compatible with both approaches.
- **FR-012**: System MUST handle subprocess termination mid-stream by returning accumulated partial results and an appropriate error.
- **FR-013**: When multiple pipeline steps execute concurrently, tool-activity events from each step MUST include sufficient context (step ID, persona) for consumers to distinguish between them.

### Key Entities

- **Adapter Stream Event**: A single parsed event from an AI subprocess's NDJSON stream. Contains event type (initialization, tool invocation, tool result, text content, or final result), tool name, tool input/target, text content, and token counts. This is the adapter-layer data type that represents raw subprocess output.
- **Tool-Activity Event**: The pipeline-level event emitted through the event system when a tool invocation is detected in the stream. Enriched with pipeline context (pipeline ID, step ID, persona). This is the consumer-facing data type used by display renderers and external integrations.
- **Stream Event Bridge**: The mechanism that connects the adapter layer to the pipeline event system. Implemented as the `OnStreamEvent` callback on `AdapterRunConfig` — a closure set by the pipeline executor that filters raw stream events (accepting only `tool_use` events with non-empty tool names), enriches them with pipeline context (pipeline ID, step ID, persona), and emits them as tool-activity events. Adapters that support streaming call this callback during line-by-line parsing; adapters that do not support streaming (e.g., OpenCode, ProcessGroupRunner) simply do not call it, and no tool-activity events are emitted. No changes to the `AdapterRunner` interface are required.
- **Throttled Display Consumer**: A display-layer wrapper around the existing `ProgressEmitter` interface. Receives all events but applies per-event-type throttling: stream_activity events are coalesced using a 1-second sliding window (most-recent-wins), while all other event types (step-start, step-complete, progress) pass through immediately. This component is distinct from the overall display refresh rate (which governs rendering frequency, currently 200ms/5 FPS). The NDJSONEmitter bypasses this consumer entirely, writing all events unthrottled to stdout.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: During pipeline step execution, tool call activity is visible to the user within 1 second of the event being emitted by the subprocess on stdout (measured from Wave parsing the event to rendering it on the display).
- **SC-002**: Human-readable terminal output displays at most 1 tool call event per second during rapid tool call bursts, preventing display flooding.
- **SC-003**: NDJSON programmatic output contains all tool call events at full fidelity with zero throttling or loss.
- **SC-004**: All existing pipeline execution tests continue to pass after the output format change (backward compatibility of final results).
- **SC-005**: Malformed stream lines do not cause pipeline step failures — the system processes all valid events and skips invalid ones.
- **SC-006**: Step-start events contain model and adapter metadata, verifiable in both NDJSON and display output.
- **SC-007**: The adapter correctly handles subprocess crashes mid-stream by returning partial results and an error, rather than hanging or panicking.

## Clarifications _(resolved)_

The following ambiguities were identified during specification review and resolved autonomously based on codebase analysis and industry best practices.

### C-001: Implementation Scope — Incremental vs. Greenfield (High Priority)

**Ambiguity**: The spec describes streaming adapter functionality as if it were entirely new, but the codebase already implements ~70-75% of the described requirements (streaming with `stream-json`, NDJSON parsing, tool target extraction, event bridge wiring).

**Resolution**: Treat the spec as describing the desired end state. Implementers MUST audit existing code against each FR and only build gaps: display-layer throttling (FR-006), extended tool target extraction (FR-009), step-start metadata (FR-010), and ETA field plumbing (FR-011). Core adapter infrastructure (FR-001 through FR-005, FR-008, FR-012, FR-013) is already working and needs only test coverage additions.

**Rationale**: Reimplementing working infrastructure would introduce regression risk and waste effort. The existing `ClaudeAdapter` already uses `--output-format stream-json`, parses NDJSON via `bufio.Scanner`, calls `OnStreamEvent` for tool_use events, and the executor already emits `stream_activity` events with pipeline context. Added the Implementation Scope section to make this explicit.

### C-002: Throttling Architecture — Display Layer Ownership (High Priority)

**Ambiguity**: The spec required 1-event-per-second throttling with event coalescing but did not specify where in the architecture throttling lives or how it integrates with the existing dual-stream output (NDJSONEmitter writes to both stdout and ProgressEmitter).

**Resolution**: Throttling is owned by a new `ThrottledProgressEmitter` wrapper in the display layer. The NDJSONEmitter continues writing all events to stdout immediately (satisfying FR-007). The ThrottledProgressEmitter wraps the existing ProgressEmitter, coalescing only stream_activity events at a 1-second interval while passing all other event types through immediately. This is separate from the existing 200ms/5 FPS display refresh rate which governs overall rendering frequency.

**Rationale**: Throttling is explicitly a human-display concern (FR-006 says "human-readable display", FR-007 says programmatic output is unthrottled). Placing it in the display layer maintains clean separation of concerns and matches the spec's "Throttled Display Consumer" entity description. Updated FR-006 and Key Entities to document this.

### C-003: Tool Target Extraction — Extended Mappings + Generic Heuristic (Medium Priority)

**Ambiguity**: FR-009 specified extraction for "known tool types" but did not define which tools are "known" vs "unrecognized." The existing implementation handles Read, Write, Edit, Glob, Grep, Bash, and Task, but upstream CLI tools include additional tools (WebFetch, WebSearch, NotebookEdit, TodoWrite, Skill) with obvious primary targets.

**Resolution**: Extend explicit mappings to cover commonly used tools not yet handled (WebFetch → url, WebSearch → query, NotebookEdit → notebook_path), then implement a generic heuristic fallback that checks well-known input field names (file_path, url, pattern, command, query, notebook_path) in priority order for any unrecognized tool. Fall back to tool name alone only when no heuristic match is found.

**Rationale**: The generic heuristic approach maximizes display value for operators while remaining resilient to new tools added upstream — no code changes required when new tools follow common parameter naming conventions. Updated FR-009 to document the extended tool list and heuristic strategy.

### C-004: ETA Data Source — Schema Plumbing Now, Data Later (Medium Priority)

**Ambiguity**: FR-011 required "estimated time remaining" from "historical or configured duration data" but did not specify the data source, storage mechanism, or fallback behavior. No duration history store exists in the current codebase.

**Resolution**: Initial implementation adds the ETA field to the event schema and progress heartbeat events but always sets it to zero (matching acceptance scenario 5.3). A future iteration can add an `expected_duration` YAML field on steps/personas and/or a SQLite-backed duration history store. The event schema is designed to be forward-compatible with both approaches.

**Rationale**: User Story 5 is labeled P3 ("Polish that improves observability. Not required for core streaming functionality"). Implementing full ETA calculation would require a new persistence layer and historical data collection — disproportionate scope for a polish feature. Schema plumbing now maintains forward-compatibility without scope creep. Updated FR-011 to document phased approach.

### C-005: Streaming Adapter Interface — Keep Existing Callback Pattern (Low Priority)

**Ambiguity**: The spec introduces a "Stream Event Bridge" entity and discusses non-streaming adapters, but does not clarify whether the `AdapterRunner` interface needs modification to declare streaming capability, or whether a new abstraction (channel, interface) should replace the existing callback pattern.

**Resolution**: Keep the existing `OnStreamEvent` callback on `AdapterRunConfig`. No interface changes needed. The callback-based bridge already handles both streaming adapters (Claude, which calls the callback during parsing) and non-streaming adapters (OpenCode, ProcessGroupRunner, which simply do not call it). The "Stream Event Bridge" is implemented as the closure in the executor that enriches raw StreamEvents with pipeline context.

**Rationale**: The current pattern already works correctly for all adapter types without requiring capability declarations or interface changes. Adding `SupportsStreaming()` methods or switching to channels would add complexity without solving an actual problem. Updated Key Entities to document the callback-based implementation.
