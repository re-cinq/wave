# Data Model: Stream Verbosity

**Feature Branch**: `026-stream-verbosity`
**Phase**: 1 (Data Model)
**Spec**: `specs/026-stream-verbosity/spec.md`

This document defines the entities, fields, relationships, and data flow for the stream
verbosity feature. Entities are split into two categories: existing entities already in
the codebase (documented as-is from source), and new or modified entities required to
close the gaps identified in the spec (FR-006, FR-009 extension, FR-010, FR-011).

---

## Existing Entities

### StreamEvent

**Location**: `internal/adapter/adapter.go` (lines 19-27)
**Purpose**: Raw parsed event from an AI subprocess's NDJSON stream output. This is the
adapter-layer data type representing a single line of real-time subprocess output after
JSON parsing.

```go
type StreamEvent struct {
    Type      string // "tool_use", "tool_result", "text", "result", "system"
    ToolName  string // e.g. "Read", "Write", "Bash", "Glob", "Grep"
    ToolInput string // summary of input (file path, command, pattern)
    Content   string // text content or result summary
    TokensIn  int    // cumulative input tokens
    TokensOut int    // cumulative output tokens
}
```

**Field Descriptions**:

| Field       | Type   | Description |
|-------------|--------|-------------|
| `Type`      | string | Event category from the NDJSON stream. One of: `"tool_use"` (tool invocation), `"tool_result"` (tool output, currently skipped), `"text"` (assistant text block), `"result"` (final completion event), `"system"` (initialization). |
| `ToolName`  | string | Name of the tool being invoked. Populated only when `Type == "tool_use"`. Examples: `"Read"`, `"Write"`, `"Edit"`, `"Bash"`, `"Glob"`, `"Grep"`, `"Task"`. |
| `ToolInput` | string | Human-readable summary of the tool's primary input. For file tools this is the file path; for Bash, a truncated command; for search tools, the pattern. Extracted by `extractToolTarget()`. |
| `Content`   | string | Text content from assistant text blocks or result summaries. Truncated to 80 characters for text events. |
| `TokensIn`  | int    | Cumulative input token count reported by the subprocess. Populated for `"result"` events. |
| `TokensOut` | int    | Cumulative output token count reported by the subprocess. Populated for `"result"` events. |

**Lifecycle**: Created by `parseStreamLine()` during line-by-line NDJSON scanning in
`ClaudeAdapter.Run()`. Delivered to the pipeline layer via the `OnStreamEvent` callback.
Discarded after callback invocation (not persisted).

---

### Event

**Location**: `internal/event/emitter.go` (lines 11-34)
**Purpose**: Pipeline-level structured event used for both programmatic output (NDJSON to
stdout) and human-readable display (stderr). This is the consumer-facing data type that
carries enriched context about pipeline and step activity.

```go
type Event struct {
    Timestamp  time.Time `json:"timestamp"`
    PipelineID string    `json:"pipeline_id"`
    StepID     string    `json:"step_id,omitempty"`
    State      string    `json:"state"`
    DurationMs int64     `json:"duration_ms,omitempty"`
    Message    string    `json:"message,omitempty"`
    Persona    string    `json:"persona,omitempty"`
    Artifacts  []string  `json:"artifacts,omitempty"`
    TokensUsed int       `json:"tokens_used,omitempty"`

    // Progress tracking fields
    Progress        int     `json:"progress,omitempty"`
    CurrentAction   string  `json:"current_action,omitempty"`
    TotalSteps      int     `json:"total_steps,omitempty"`
    CompletedSteps  int     `json:"completed_steps,omitempty"`
    EstimatedTimeMs int64   `json:"estimated_time_ms,omitempty"`
    ValidationPhase string  `json:"validation_phase,omitempty"`
    CompactionStats *string `json:"compaction_stats,omitempty"`

    // Stream event fields (real-time tool activity)
    ToolName   string `json:"tool_name,omitempty"`
    ToolTarget string `json:"tool_target,omitempty"`
}
```

**Field Descriptions**:

| Field              | Type      | Description |
|--------------------|-----------|-------------|
| `Timestamp`        | time.Time | When the event was created. |
| `PipelineID`       | string    | Identifier of the executing pipeline. |
| `StepID`           | string    | Identifier of the executing step. Empty for pipeline-level events. |
| `State`            | string    | Event state. See state constants below. |
| `DurationMs`       | int64     | Duration of the completed operation in milliseconds. |
| `Message`          | string    | Human-readable description of the event. |
| `Persona`          | string    | Name of the persona executing the step. |
| `Artifacts`        | []string  | List of artifact paths produced by the step. |
| `TokensUsed`       | int       | Total tokens consumed by the step. |
| `Progress`         | int       | Step progress percentage (0-100). |
| `CurrentAction`    | string    | Description of the current action being performed. |
| `TotalSteps`       | int       | Total number of steps in the pipeline. |
| `CompletedSteps`   | int       | Number of steps completed so far. |
| `EstimatedTimeMs`  | int64     | Estimated time remaining in milliseconds. Currently always 0 (FR-011). |
| `ValidationPhase`  | string    | Current contract validation phase. |
| `CompactionStats`  | *string   | JSON-encoded compaction statistics. |
| `ToolName`         | string    | Tool being used. Populated for `stream_activity` events. Examples: `"Read"`, `"Write"`, `"Bash"`. |
| `ToolTarget`       | string    | Primary target of the tool. Populated for `stream_activity` events. Examples: file path, command, search pattern. |

**State Constants** (defined in `internal/event/emitter.go`, lines 37-51):

| Constant                    | Value                  | Description |
|-----------------------------|------------------------|-------------|
| `StateStarted`              | `"started"`            | Pipeline or step has started. |
| `StateRunning`              | `"running"`            | Pipeline or step is actively executing. |
| `StateCompleted`            | `"completed"`          | Pipeline or step finished successfully. |
| `StateFailed`               | `"failed"`             | Pipeline or step failed. |
| `StateRetrying`             | `"retrying"`           | Step is being retried after failure. |
| `StateStepProgress`         | `"step_progress"`      | Step progress update with percentage. |
| `StateETAUpdated`           | `"eta_updated"`        | Estimated time remaining updated. |
| `StateContractValidating`   | `"contract_validating"`| Contract validation in progress. |
| `StateCompactionProgress`   | `"compaction_progress"`| Context compaction in progress. |
| `StateStreamActivity`       | `"stream_activity"`    | Real-time tool activity from the AI subprocess. |

**Lifecycle**: Created by the pipeline executor when step lifecycle events occur or when
the `OnStreamEvent` callback fires for `tool_use` events. Emitted through
`NDJSONEmitter.Emit()`, which writes to stdout (NDJSON) and optionally forwards to a
`ProgressEmitter` (stderr display).

---

### AdapterRunConfig

**Location**: `internal/adapter/adapter.go` (lines 29-47)
**Purpose**: Configuration passed to an adapter's `Run()` method. Contains all parameters
for subprocess execution, including the streaming callback that bridges the adapter layer
to the pipeline event system.

```go
type AdapterRunConfig struct {
    Adapter       string
    Persona       string
    WorkspacePath string
    Prompt        string
    SystemPrompt  string
    Timeout       time.Duration
    Env           []string
    Temperature   float64
    AllowedTools  []string
    DenyTools     []string
    OutputFormat  string
    Debug         bool
    Model         string

    // OnStreamEvent is called for each real-time event during Claude Code execution.
    // If nil, streaming events are silently ignored.
    OnStreamEvent func(StreamEvent)
}
```

**Key Field for Stream Verbosity**:

| Field           | Type                  | Description |
|-----------------|-----------------------|-------------|
| `OnStreamEvent` | `func(StreamEvent)`   | Callback invoked by the adapter for each parsed stream event. Set by the pipeline executor as a closure that filters for `tool_use` events, enriches them with pipeline context (pipeline ID, step ID, persona), and emits them as `Event{State: "stream_activity"}`. If nil, no streaming events are emitted. Non-streaming adapters (e.g., `ProcessGroupRunner`) never invoke this callback. |
| `Model`         | string                | Model identifier (e.g., `"opus"`, `"sonnet"`, `"claude-opus-4-5-20251101"`). Used by FR-010 for step-start metadata. |
| `Adapter`       | string                | Adapter type identifier (e.g., `"claude"`). Used by FR-010 for step-start metadata. |

---

### AdapterResult

**Location**: `internal/adapter/adapter.go` (lines 49-55)
**Purpose**: Return value from an adapter execution. Contains the final accumulated result
after stream processing completes.

```go
type AdapterResult struct {
    ExitCode      int
    Stdout        io.Reader
    TokensUsed    int
    Artifacts     []string
    ResultContent string
}
```

| Field           | Type      | Description |
|-----------------|-----------|-------------|
| `ExitCode`      | int       | Subprocess exit code. 0 for success. |
| `Stdout`        | io.Reader | Raw stdout content from the subprocess. |
| `TokensUsed`    | int       | Total tokens consumed (input + output). Extracted from the `"result"` stream event. |
| `Artifacts`     | []string  | Artifact paths extracted from output. |
| `ResultContent` | string    | Extracted final result content. For `stream-json`, this comes from the `"result"` event's `result` field after markdown/JSON correction. |

---

### NDJSONEmitter

**Location**: `internal/event/emitter.go` (lines 57-62)
**Purpose**: Primary event emitter. Writes events as NDJSON to stdout and optionally
forwards them to a `ProgressEmitter` for human-readable display on stderr.

```go
type NDJSONEmitter struct {
    encoder         *json.Encoder
    suppressJSON    bool
    mu              sync.Mutex
    progressEmitter ProgressEmitter
}
```

| Field              | Type             | Description |
|--------------------|------------------|-------------|
| `encoder`          | *json.Encoder    | JSON encoder writing to stdout. |
| `suppressJSON`     | bool             | When true, suppresses JSON output to stdout (progress-only mode). |
| `mu`               | sync.Mutex       | Protects concurrent Emit() calls. |
| `progressEmitter`  | ProgressEmitter  | Optional display-layer emitter for stderr rendering. This is where the ThrottledProgressEmitter is wired in. |

**Behavior**: `Emit()` always forwards to `progressEmitter` first (if set), then writes
NDJSON to stdout (unless `suppressJSON` is true). All events are written to stdout
unthrottled, satisfying FR-007.

---

### ProgressEmitter (interface)

**Location**: `internal/event/emitter.go` (lines 64-68)
**Purpose**: Interface for display-layer consumers that render human-readable progress on
stderr.

```go
type ProgressEmitter interface {
    EmitProgress(event Event) error
}
```

Existing implementations:
- `ProgressDisplay` -- full TTY dashboard with spinners, progress bars, and step tracking
- `BubbleTeaProgressDisplay` -- BubbleTea-based interactive terminal UI
- `BasicProgressDisplay` -- simple text output for non-TTY environments
- `QuietProgressDisplay` -- minimal output (pipeline-level completion/failure only)

---

### PipelineContext

**Location**: `internal/display/types.go` (lines 190-238)
**Purpose**: Comprehensive context object passed to display renderers. Aggregates pipeline
state, step tracking, timing, and tool activity into a single structure for rendering.

```go
type PipelineContext struct {
    // Project metadata
    ManifestPath  string
    PipelineName  string
    WorkspacePath string

    // Step tracking
    TotalSteps     int
    CurrentStepNum int
    CompletedSteps int
    FailedSteps    int
    SkippedSteps   int

    // Progress calculation
    OverallProgress int
    EstimatedTimeMs int64

    // Current execution state
    CurrentStepID   string
    CurrentPersona  string
    CurrentAction   string
    CurrentStepName string

    // Timing information
    PipelineStartTime int64
    CurrentStepStart  int64
    AverageStepTimeMs int64
    ElapsedTimeMs     int64

    // Step status mapping
    StepStatuses map[string]ProgressState
    StepOrder    []string

    // Step durations in milliseconds
    StepDurations map[string]int64

    // Deliverables by step
    DeliverablesByStep map[string][]string

    // Tool activity (verbose mode)
    LastToolName   string
    LastToolTarget string

    // Additional context
    Message string
    Error   string
}
```

**Key Fields for Stream Verbosity**:

| Field            | Type   | Description |
|------------------|--------|-------------|
| `LastToolName`   | string | Most recent tool name from a `stream_activity` event. Used by verbose display renderers. |
| `LastToolTarget` | string | Most recent tool target from a `stream_activity` event. Used by verbose display renderers. |
| `EstimatedTimeMs`| int64  | ETA in milliseconds. Currently always 0 (FR-011 schema plumbing). |

---

## New/Modified Entities

### 1. Event -- New Fields (FR-010)

**Modification to**: `internal/event/emitter.go`, Event struct
**Requirement**: FR-010 (model name and adapter type in step-start events)

Add two new fields to the existing `Event` struct:

```go
type Event struct {
    // ... existing fields ...

    // Step metadata (FR-010: model and adapter type in step-start events)
    Model   string `json:"model,omitempty"`   // Model name (e.g., "claude-sonnet-4-20250514")
    Adapter string `json:"adapter,omitempty"` // Adapter type (e.g., "claude")
}
```

| Field     | Type   | When Populated | Description |
|-----------|--------|----------------|-------------|
| `Model`   | string | `State == "running"` (step start) | The model identifier used for this step. Sourced from `AdapterRunConfig.Model`. Examples: `"opus"`, `"sonnet"`, `"claude-opus-4-5-20251101"`. |
| `Adapter` | string | `State == "running"` (step start) | The adapter type executing this step. Sourced from `AdapterRunConfig.Adapter`. Examples: `"claude"`, `"opencode"`. |

**Impact**: These fields use `omitempty`, so they are omitted from NDJSON output for
events that do not carry metadata (stream_activity, step_progress, etc.). No breaking
change to existing consumers.

---

### 2. ThrottledProgressEmitter (FR-006) -- NEW

**Location**: `internal/display/throttled_emitter.go` (new file)
**Requirement**: FR-006 (1-second throttling of stream_activity for human display)
**Implements**: `ProgressEmitter` interface from `internal/event/emitter.go`

```go
type ThrottledProgressEmitter struct {
    inner                  ProgressEmitter
    mu                     sync.Mutex
    lastStreamActivityTime time.Time
    pendingStreamActivity  *event.Event
    throttleInterval       time.Duration  // default: 1 * time.Second
}
```

| Field                     | Type             | Description |
|---------------------------|------------------|-------------|
| `inner`                   | ProgressEmitter  | The wrapped display emitter (BubbleTeaProgressDisplay, BasicProgressDisplay, etc.). All events ultimately pass through to this emitter. |
| `mu`                      | sync.Mutex       | Protects concurrent access to throttling state. |
| `lastStreamActivityTime`  | time.Time        | Timestamp of the last `stream_activity` event forwarded to the inner emitter. Used for 1-second sliding window enforcement. |
| `pendingStreamActivity`   | *event.Event     | Most recent `stream_activity` event received but not yet forwarded (coalesced). On the next event or tick that crosses the throttle boundary, this event is forwarded to the inner emitter. |
| `throttleInterval`        | time.Duration    | Minimum time between forwarded `stream_activity` events. Default: 1 second. |

**Behavior**:

- `EmitProgress(event)` receives all events from the `NDJSONEmitter`.
- For events where `State != "stream_activity"`: forward immediately to `inner.EmitProgress()`.
- For `stream_activity` events:
  - If `>= throttleInterval` has elapsed since `lastStreamActivityTime`: forward immediately, update `lastStreamActivityTime`.
  - Otherwise: store in `pendingStreamActivity` (most-recent-wins coalescing). The pending event is forwarded when the next non-throttled event arrives or the throttle window expires.
- This ensures that rapid tool call bursts (e.g., 10+ Grep calls in 1 second) produce at most 1 display update per second while showing the most recent tool call.

**Constructor**:

```go
func NewThrottledProgressEmitter(inner ProgressEmitter) *ThrottledProgressEmitter
func NewThrottledProgressEmitterWithInterval(inner ProgressEmitter, interval time.Duration) *ThrottledProgressEmitter
```

**Relationship to existing architecture**: The `ThrottledProgressEmitter` is wired as the
`progressEmitter` field on `NDJSONEmitter`. The call chain becomes:

```
NDJSONEmitter.Emit(event)
    |
    +---> stdout (NDJSON, unthrottled, all events) ... FR-007
    |
    +---> ThrottledProgressEmitter.EmitProgress(event)
              |
              +---> [stream_activity: coalesce at 1-sec]
              +---> [all other states: pass through immediately]
              |
              +---> inner.EmitProgress(event)  (actual display renderer)
```

---

### 3. Extended Tool Target Extraction (FR-009 Extension)

**Modification to**: `extractToolTarget()` in `internal/adapter/claude.go`
**Requirement**: FR-009 (extended tool mappings + generic heuristic fallback)

This is not a new struct but a behavioral change to an existing function. The current
implementation handles 7 tools. The extension adds 3 explicit mappings and a generic
heuristic fallback.

**Current tool-to-target mapping** (already implemented):

| Tool   | Target Field  | Example Target |
|--------|---------------|----------------|
| Read   | `file_path`   | `/home/user/main.go` |
| Write  | `file_path`   | `/home/user/output.json` |
| Edit   | `file_path`   | `/home/user/config.yaml` |
| Glob   | `pattern`     | `**/*.go` |
| Grep   | `pattern`     | `OnStreamEvent` |
| Bash   | `command`     | `go test ./...` (truncated to 60 chars) |
| Task   | `description` | `Analyze the test results` |

**New explicit mappings** (to be added):

| Tool         | Target Field    | Example Target |
|--------------|-----------------|----------------|
| WebFetch     | `url`           | `https://example.com/api` |
| WebSearch    | `query`         | `Go stream-json parsing 2026` |
| NotebookEdit | `notebook_path` | `/home/user/analysis.ipynb` |

**Generic heuristic fallback** (for any unrecognized tool name):

When the tool name does not match any explicit mapping, check the tool's input JSON for
well-known field names in the following priority order. Return the first match:

| Priority | Field Name      | Rationale |
|----------|-----------------|-----------|
| 1        | `file_path`     | Most common target for file-oriented tools. |
| 2        | `url`           | Web-oriented tools. |
| 3        | `pattern`       | Search-oriented tools. |
| 4        | `command`        | Execution-oriented tools. |
| 5        | `query`         | Query-oriented tools. |
| 6        | `notebook_path` | Notebook-oriented tools. |

If no heuristic match is found, return an empty string (the tool name alone is still
displayed by the event's `ToolName` field).

**Updated function signature** (unchanged):

```go
func extractToolTarget(toolName string, input json.RawMessage) string
```

---

## Entity Relationships

### Overview Diagram

```
Claude subprocess (NDJSON on stdout)
    |
    v
bufio.Scanner (line-by-line, 10MB max buffer)
    |
    v
parseStreamLine() ---> StreamEvent (adapter layer, raw parsed event)
    |
    v
extractToolTarget() (tool name -> primary target string)
    |
    v
OnStreamEvent callback (AdapterRunConfig field)
    |
    |   [closure in pipeline executor filters for tool_use,
    |    enriches with pipeline context]
    |
    v
event.Event{State: "stream_activity", ToolName, ToolTarget, PipelineID, StepID, Persona}
    |
    v
NDJSONEmitter.Emit()
    |
    +---> json.Encoder --> stdout (NDJSON, unthrottled)       ... FR-007
    |
    +---> ThrottledProgressEmitter.EmitProgress()             ... FR-006
              |
              +---> [stream_activity: 1-sec coalescing, most-recent-wins]
              +---> [all other states: immediate passthrough]
              |
              v
         ProgressEmitter.EmitProgress()  (inner display renderer)
              |
              +---> BubbleTeaProgressDisplay  (interactive TTY)
              +---> ProgressDisplay           (TTY dashboard with spinners)
              +---> BasicProgressDisplay      (non-TTY text output)
              +---> QuietProgressDisplay      (pipeline-level only, suppresses stream_activity)
```

### Entity Ownership

| Entity                      | Package           | Owner Layer   |
|-----------------------------|-------------------|---------------|
| StreamEvent                 | internal/adapter  | Adapter       |
| AdapterRunConfig            | internal/adapter  | Adapter       |
| AdapterResult               | internal/adapter  | Adapter       |
| Event                       | internal/event    | Pipeline      |
| NDJSONEmitter               | internal/event    | Pipeline      |
| ProgressEmitter (interface) | internal/event    | Pipeline      |
| ThrottledProgressEmitter    | internal/display  | Display       |
| PipelineContext             | internal/display  | Display       |
| BubbleTeaProgressDisplay    | internal/display  | Display       |
| BasicProgressDisplay        | internal/display  | Display       |
| QuietProgressDisplay        | internal/display  | Display       |

### Cross-Layer Dependencies

```
adapter layer                pipeline layer               display layer
--------------               ----------------             ---------------
StreamEvent       -------->  Event                ------> PipelineContext
AdapterRunConfig  (callback) NDJSONEmitter        ------> ThrottledProgressEmitter
AdapterResult                ProgressEmitter(if)  <------ (implementations)
```

- The **adapter layer** has no dependency on the pipeline or display layers. It exposes
  `StreamEvent` and the `OnStreamEvent` callback slot.
- The **pipeline layer** depends on the adapter layer (imports `StreamEvent`) and defines
  the `ProgressEmitter` interface that the display layer implements.
- The **display layer** depends on the pipeline layer (imports `Event`, implements
  `ProgressEmitter`). It has no dependency on the adapter layer.

---

## Data Flow (End-to-End)

### Step 1: Subprocess Output

The Claude subprocess is started with `--output-format stream-json`. It writes NDJSON
lines to stdout as it executes. Each line is a JSON object with a `type` field.

### Step 2: Line-by-Line Scanning

`ClaudeAdapter.Run()` reads stdout through a `bufio.Scanner` with a 10MB maximum line
buffer. Each line is processed as it arrives (real-time, not buffered until process exit).

### Step 3: Stream Event Parsing

`parseStreamLine()` parses each JSON line and produces a `StreamEvent`:
- `"system"` type: returns `StreamEvent{Type: "system"}`.
- `"assistant"` type: delegates to `parseAssistantEvent()`, which inspects `message.content`
  blocks for `tool_use` and `text` entries.
- `"tool_result"` type: returns false (skipped; the preceding `tool_use` already reported).
- `"result"` type: extracts token counts into `StreamEvent{Type: "result"}`.

### Step 4: Tool Target Extraction

For `tool_use` events, `extractToolTarget()` maps the tool name to its primary input
field. The extracted target is stored in `StreamEvent.ToolInput`.

### Step 5: Stream Event Callback (Event Bridge)

The `OnStreamEvent` callback (a closure set by the pipeline executor) is invoked:
- **Filter**: Only `tool_use` events with a non-empty `ToolName` pass through.
- **Enrich**: The closure adds `PipelineID`, `StepID`, and `Persona` from the executing
  step context.
- **Emit**: Creates an `Event{State: "stream_activity"}` and passes it to
  `NDJSONEmitter.Emit()`.

```go
// From internal/pipeline/executor.go
OnStreamEvent: func(evt adapter.StreamEvent) {
    if evt.Type == "tool_use" && evt.ToolName != "" {
        e.emit(event.Event{
            Timestamp:  time.Now(),
            PipelineID: pipelineID,
            StepID:     step.ID,
            State:      event.StateStreamActivity,
            Persona:    step.Persona,
            ToolName:   evt.ToolName,
            ToolTarget: evt.ToolInput,
        })
    }
},
```

### Step 6: Dual-Stream Emission

`NDJSONEmitter.Emit()` performs two actions under a mutex:

1. **Progress emitter** (stderr): If a `ProgressEmitter` is configured, forwards the
   event to `progressEmitter.EmitProgress()`. With the new `ThrottledProgressEmitter`,
   this applies 1-second coalescing to `stream_activity` events.
2. **NDJSON encoder** (stdout): Unless `suppressJSON` is true, encodes the event as a
   single JSON line to stdout. All events are written immediately with no throttling
   (FR-007).

### Step 7: Throttled Display Rendering

The `ThrottledProgressEmitter` receives the event and decides:
- **Non-stream_activity events** (started, running, completed, failed, step_progress):
  forwarded immediately to the inner `ProgressEmitter`. If a pending `stream_activity`
  event exists, it is flushed first.
- **stream_activity events**: if the throttle window (1 second) has elapsed, forward
  immediately and reset the timer. Otherwise, store as pending (overwriting any previous
  pending event).

### Step 8: Display Rendering

The inner `ProgressEmitter` implementation renders the event to stderr:
- **BubbleTeaProgressDisplay**: Updates the `PipelineContext` with `LastToolName` and
  `LastToolTarget`, then sends a BubbleTea model update for re-rendering.
- **BasicProgressDisplay**: If verbose mode is enabled, prints a timestamped line with
  tool name and target (truncated to 60 characters).
- **QuietProgressDisplay**: Ignores all `stream_activity` events.

### Step 9: Result Accumulation

After the subprocess exits, `ClaudeAdapter.parseOutput()` scans the accumulated stdout
buffer for the `"result"` event, extracting token counts and the final result content.
This is returned in `AdapterResult` for artifact writing and contract validation. The
streaming process does not affect result extraction (FR-004 backward compatibility).

---

## Requirement Traceability

| Requirement | Entity / Mechanism | Status |
|-------------|-------------------|--------|
| FR-001 | `ClaudeAdapter.buildArgs()`: `--output-format stream-json` | Existing |
| FR-002 | `ClaudeAdapter.Run()`: `bufio.Scanner` line-by-line | Existing |
| FR-003 | `parseStreamLine()` + `extractToolTarget()` | Existing |
| FR-004 | `ClaudeAdapter.parseOutput()` extracts `"result"` event | Existing |
| FR-005 | `OnStreamEvent` closure in executor emits `stream_activity` | Existing |
| FR-006 | `ThrottledProgressEmitter` (new) | **New** |
| FR-007 | `NDJSONEmitter.Emit()` writes all events to stdout | Existing |
| FR-008 | `parseStreamLine()` returns false for malformed lines | Existing |
| FR-009 | `extractToolTarget()` explicit mappings | Existing (7 tools) |
| FR-009 ext | `extractToolTarget()` + WebFetch, WebSearch, NotebookEdit + heuristic | **New** |
| FR-010 | `Event.Model`, `Event.Adapter` fields | **New** |
| FR-011 | `Event.EstimatedTimeMs` field (always 0 initially) | Existing (schema), **New** (plumbing) |
| FR-012 | `ClaudeAdapter.Run()` context cancellation + process group kill | Existing |
| FR-013 | `Event.StepID`, `Event.Persona` in stream_activity events | Existing |
