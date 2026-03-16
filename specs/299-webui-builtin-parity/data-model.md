# Data Model: WebUI Built-in Parity

## Existing Entities (No Changes Required)

### RunRecord (state/store.go)
Already models pipeline runs with all needed fields: `RunID`, `PipelineName`, `Status`, `Input`, `CurrentStep`, `TotalTokens`, `StartedAt`, `CompletedAt`, `ErrorMessage`, `Tags`.

### LogRecord (state/store.go)
Already models events with `ID` (auto-increment), `RunID`, `StepID`, `State`, `Persona`, `Message`, `TokensUsed`, `DurationMs`, `Timestamp`. The `ID` field is critical for SSE `Last-Event-ID` backfill.

### ArtifactRecord (state/store.go)
Already models artifacts with `ID`, `RunID`, `StepID`, `Name`, `Path`, `Type`, `SizeBytes`.

### StepStateRecord (state/store.go)
Already models step state with `StepID`, `PipelineID`, `State`, `RetryCount`, `StartedAt`, `CompletedAt`, `WorkspacePath`, `ErrorMessage`.

## New/Modified Types

### ResumeRunRequest (webui/types.go — NEW)
```go
type ResumeRunRequest struct {
    FromStep string `json:"from_step"`
    Force    bool   `json:"force"`
}
```
Request body for `POST /api/runs/{id}/resume`.

### ResumeRunResponse (webui/types.go — NEW)
```go
type ResumeRunResponse struct {
    RunID         string    `json:"run_id"`
    OriginalRunID string    `json:"original_run_id"`
    PipelineName  string    `json:"pipeline_name"`
    FromStep      string    `json:"from_step"`
    Status        string    `json:"status"`
    StartedAt     time.Time `json:"started_at"`
}
```
Response from the resume endpoint.

### EventQueryOptions (state/store.go — MODIFIED)
```go
type EventQueryOptions struct {
    Limit   int
    AfterID int64  // NEW: for SSE Last-Event-ID backfill
}
```
Add `AfterID` field to support querying events after a specific ID for SSE reconnection backfill.

### SSEEvent (webui/sse.go — EXISTING, usage change)
The `ID` field already exists on `SSEEvent` but is not populated. The broker must set it from the event's database row ID.

## API Endpoints

### Existing (unchanged)
| Method | Path | Handler |
|--------|------|---------|
| GET | `/api/runs` | `handleAPIRuns` |
| GET | `/api/runs/{id}` | `handleAPIRunDetail` |
| POST | `/api/pipelines/{name}/start` | `handleStartPipeline` |
| POST | `/api/runs/{id}/cancel` | `handleCancelRun` |
| POST | `/api/runs/{id}/retry` | `handleRetryRun` |
| GET | `/api/personas` | `handleAPIPersonas` |
| GET | `/api/pipelines` | `handleAPIPipelines` |
| GET | `/api/runs/{id}/artifacts/{step}/{name}` | `handleArtifact` |
| GET | `/api/runs/{id}/events` | `handleSSE` |

### New
| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| POST | `/api/runs/{id}/resume` | `handleResumeRun` | Resume from failed step (FR-007) |

### Modified
| Endpoint | Change |
|----------|--------|
| `POST /api/runs/{id}/retry` | Now launches actual pipeline execution (R-002) |
| `GET /api/runs/{id}/events` | SSE events include `id:` field; supports `Last-Event-ID` backfill (R-004) |

## Template Structure

### Existing Templates (enhanced)
- `templates/layout.html` — Add responsive meta viewport, ARIA landmarks, keyboard nav JS
- `templates/runs.html` — Already has start form; no major changes
- `templates/run_detail.html` — Add resume dropdown, DAG tooltips, step log streaming
- `templates/personas.html` — Add permission summary display
- `templates/pipelines.html` — Add DAG preview per pipeline

### Existing Partials (enhanced)
- `templates/partials/dag_svg.html` — Add tooltip containers, ARIA labels, click handlers
- `templates/partials/step_card.html` — Add log streaming toggle, artifact viewer link
- `templates/partials/artifact_viewer.html` — Add credential redaction indicator, truncation notice

### New Partials
- `templates/partials/resume_dialog.html` — Step picker for resume-from-step

## Static Assets

### Existing (enhanced)
- `static/style.css` — Add responsive breakpoints (768px, 1024px, 1920px), focus indicators, DAG tooltip styles
- `static/app.js` — Add keyboard navigation, polling fallback logic
- `static/sse.js` — Add Last-Event-ID support, reconnection with backfill
- `static/dag.js` — Add hover tooltips, click-to-inspect, scroll/zoom for large DAGs

## Database Changes

No schema changes required. The existing tables (`runs`, `event_log`, `artifacts`, `step_state`, `progress_snapshots`) already support all needed queries. The only change is adding `AfterID` filtering to `GetEvents` query logic.
