# Data Model: Web-Based Pipeline Operations Dashboard

**Branch**: `085-web-operations-dashboard` | **Date**: 2026-02-13

## Overview

The dashboard data model is entirely read-oriented. It reads from the existing SQLite state database (`internal/state/`) and the manifest configuration (`internal/manifest/`). No new database tables are required — the existing schema already contains all necessary entities.

## Existing Entities (Read-Only from State DB)

### RunRecord (pipeline_run table)

The primary entity displayed in the dashboard. Already defined in `internal/state/types.go`.

| Field | Type | Source Column | Dashboard Use |
|-------|------|---------------|---------------|
| RunID | string | run_id (PK) | Unique identifier, URL parameter |
| PipelineName | string | pipeline_name | Display name, filtering |
| Status | string | status | Status badge, filtering |
| Input | string | input | Display in detail view |
| CurrentStep | string | current_step | Active step indicator |
| TotalTokens | int | total_tokens | Token consumption display |
| StartedAt | time.Time | started_at | Timestamp display, cursor pagination |
| CompletedAt | *time.Time | completed_at | Duration calculation |
| CancelledAt | *time.Time | cancelled_at | Cancellation indicator |
| ErrorMessage | string | error_message | Error display in detail view |
| Tags | []string | tags_json | Tag badges, filtering |

**Statuses**: `pending`, `running`, `completed`, `failed`, `cancelled`

### LogRecord (event_log table)

Step-level events within a pipeline run. Already defined in `internal/state/types.go`.

| Field | Type | Source Column | Dashboard Use |
|-------|------|---------------|---------------|
| ID | int64 | id (PK) | Ordering |
| RunID | string | run_id (FK) | Parent association |
| Timestamp | time.Time | timestamp | Timeline display |
| StepID | string | step_id | Step grouping |
| State | string | state | Status indicator |
| Persona | string | persona | Persona badge |
| Message | string | message | Event description |
| TokensUsed | int | tokens_used | Token display |
| DurationMs | int64 | duration_ms | Duration display |

### ArtifactRecord (artifact table)

Step output artifacts. Already defined in `internal/state/types.go`.

| Field | Type | Source Column | Dashboard Use |
|-------|------|---------------|---------------|
| ID | int64 | id (PK) | Unique identifier |
| RunID | string | run_id (FK) | Parent association |
| StepID | string | step_id | Step grouping |
| Name | string | name | Display name |
| Path | string | path | File access (validated) |
| Type | string | type | File type icon |
| SizeBytes | int64 | size_bytes | Size display |
| CreatedAt | time.Time | created_at | Timestamp |

### StepProgressRecord (step_progress table)

Real-time step progress. Already defined in `internal/state/types.go`.

| Field | Type | Source Column | Dashboard Use |
|-------|------|---------------|---------------|
| StepID | string | step_id (PK) | Step identification |
| RunID | string | run_id (FK) | Parent association |
| Persona | string | persona | Persona display |
| State | string | state | Status indicator |
| Progress | int | progress | Progress bar (0-100) |
| CurrentAction | string | current_action | Activity label |
| Message | string | message | Status message |
| StartedAt | *time.Time | started_at | Duration calculation |
| UpdatedAt | time.Time | updated_at | Staleness detection |
| EstimatedCompletionMs | int64 | estimated_completion_ms | ETA display |
| TokensUsed | int | tokens_used | Token count |

### PipelineProgressRecord (pipeline_progress table)

Aggregated pipeline-level progress. Already defined in `internal/state/types.go`.

| Field | Type | Source Column | Dashboard Use |
|-------|------|---------------|---------------|
| RunID | string | run_id (PK) | Parent association |
| TotalSteps | int | total_steps | Progress denominator |
| CompletedSteps | int | completed_steps | Progress numerator |
| CurrentStepIndex | int | current_step_index | Active step highlighting |
| OverallProgress | int | overall_progress | Overall progress bar |
| EstimatedCompletionMs | int64 | estimated_completion_ms | Pipeline ETA |
| UpdatedAt | time.Time | updated_at | Staleness detection |

### ArtifactMetadataRecord (artifact_metadata table)

Extended artifact metadata for display. Already defined in `internal/state/types.go`.

| Field | Type | Source Column | Dashboard Use |
|-------|------|---------------|---------------|
| ArtifactID | int64 | artifact_id (PK/FK) | Join to artifact |
| RunID | string | run_id | Association |
| StepID | string | step_id | Association |
| PreviewText | string | preview_text | Inline preview |
| MimeType | string | mime_type | Content type handling |
| Encoding | string | encoding | Character encoding |
| MetadataJSON | string | metadata_json | Extended metadata |
| IndexedAt | time.Time | indexed_at | Freshness |

### CancellationRecord (cancellation table)

Cancellation request tracking. Used for the "Stop" action in the dashboard.

| Field | Type | Source Column | Dashboard Use |
|-------|------|---------------|---------------|
| RunID | string | run_id (PK) | Target run |
| RequestedAt | time.Time | requested_at | Timestamp |
| Force | bool | force | Force cancel flag |

## Entities from Manifest (Read-Only)

### Persona (manifest.Persona)

Read from `wave.yaml` at server startup. Defined in `internal/manifest/types.go`.

| Field | Type | Dashboard Use |
|-------|------|---------------|
| Adapter | string | Adapter badge |
| Description | string | Description text |
| SystemPromptFile | string | System prompt path |
| Temperature | float64 | Config display |
| Model | string | Model badge |
| Permissions.AllowedTools | []string | Permission list |
| Permissions.Deny | []string | Deny list |

### Pipeline (pipeline.Pipeline)

Read from `.wave/pipelines/` at server startup. Defined in `internal/pipeline/types.go`.

| Field | Type | Dashboard Use |
|-------|------|---------------|
| Metadata.Name | string | Pipeline identifier |
| Metadata.Description | string | Description text |
| Steps | []Step | DAG construction, start form |
| Input | InputConfig | Input form generation |

## New Types (Dashboard-Specific)

### API Response Types

These are JSON serialization types for the REST API. Defined in the new `internal/webui/` package.

#### RunListResponse

```go
type RunListResponse struct {
    Runs       []RunSummary `json:"runs"`
    NextCursor string       `json:"next_cursor,omitempty"`
    HasMore    bool         `json:"has_more"`
}
```

#### RunSummary

```go
type RunSummary struct {
    RunID        string     `json:"run_id"`
    PipelineName string     `json:"pipeline_name"`
    Status       string     `json:"status"`
    CurrentStep  string     `json:"current_step,omitempty"`
    TotalTokens  int        `json:"total_tokens"`
    StartedAt    time.Time  `json:"started_at"`
    CompletedAt  *time.Time `json:"completed_at,omitempty"`
    Duration     string     `json:"duration,omitempty"`
    Tags         []string   `json:"tags,omitempty"`
    Progress     int        `json:"progress,omitempty"`
    ErrorMessage string     `json:"error_message,omitempty"`
}
```

#### RunDetailResponse

```go
type RunDetailResponse struct {
    Run       RunSummary           `json:"run"`
    Steps     []StepDetail         `json:"steps"`
    Events    []EventSummary       `json:"events"`
    Artifacts []ArtifactSummary    `json:"artifacts"`
    DAG       *DAGData             `json:"dag,omitempty"`
}
```

#### StepDetail

```go
type StepDetail struct {
    StepID      string     `json:"step_id"`
    Persona     string     `json:"persona"`
    State       string     `json:"state"`
    Progress    int        `json:"progress"`
    Action      string     `json:"current_action,omitempty"`
    StartedAt   *time.Time `json:"started_at,omitempty"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
    Duration    string     `json:"duration,omitempty"`
    TokensUsed  int        `json:"tokens_used"`
    Error       string     `json:"error,omitempty"`
    Artifacts   []ArtifactSummary `json:"artifacts,omitempty"`
}
```

#### DAGData

```go
type DAGData struct {
    Nodes []DAGNode `json:"nodes"`
    Edges []DAGEdge `json:"edges"`
}

type DAGNode struct {
    ID       string `json:"id"`
    Label    string `json:"label"`
    Persona  string `json:"persona"`
    Status   string `json:"status"`
    Progress int    `json:"progress"`
    X        int    `json:"x"`
    Y        int    `json:"y"`
}

type DAGEdge struct {
    From string `json:"from"`
    To   string `json:"to"`
}
```

#### PaginationCursor

```go
type PaginationCursor struct {
    Timestamp int64  `json:"t"`
    RunID     string `json:"id"`
}
```

### SSE Event Types

SSE events are serialized from the existing `event.Event` struct. The SSE endpoint serializes each event as:

```
event: <state>
data: <JSON-encoded event.Event>

```

Event types map directly from `event.Event.State`:
- `started` — New run or step started
- `running` — Step is actively executing
- `completed` — Step or run completed
- `failed` — Step or run failed
- `step_progress` — Progress update (percentage, action)
- `stream_activity` — Tool usage activity
- `eta_updated` — ETA recalculation

## Entity Relationships

```
Pipeline (manifest) ──1:N──> Step (manifest)
                                │
                                ├──> DAGNode (computed at render time)
                                │
RunRecord (state DB) ──1:N──> LogRecord (state DB)
    │                 ──1:N──> ArtifactRecord (state DB)
    │                 ──1:N──> StepProgressRecord (state DB)
    │                 ──1:1──> PipelineProgressRecord (state DB)
    │                 ──0:1──> CancellationRecord (state DB)
    │
    └── ArtifactRecord ──1:0..1──> ArtifactMetadataRecord (state DB)

Persona (manifest) ──referenced by──> Step.Persona
                   ──referenced by──> StepProgressRecord.Persona
                   ──referenced by──> LogRecord.Persona
```

## Database Access Pattern

The dashboard opens the state database as **read-only** (`NewReadOnlyStateStore`). All mutations (start, cancel, retry) go through the existing `StateStore` interface on a separate read-write connection.

### Read Path (dashboard queries):
- `ListRuns` with cursor pagination → Run list page
- `GetRun` → Run detail page
- `GetEvents` → Event timeline
- `GetArtifacts` → Artifact list
- `GetAllStepProgress` → Step progress display
- `GetPipelineProgress` → Overall progress bar

### Write Path (execution control):
- `CreateRun` → Start pipeline (via separate RW store)
- `RequestCancellation` → Stop pipeline
- `CreateRun` (with same input) → Retry pipeline

The read-only and read-write connections coexist via SQLite WAL mode.
