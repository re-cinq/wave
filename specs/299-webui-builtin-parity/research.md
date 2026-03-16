# Research: WebUI Built-in Parity

## R-001: Build Tag Removal Strategy

**Decision**: Remove `//go:build webui` from all 22 Go files in `internal/webui/` and from `cmd/wave/commands/serve.go`. Delete `cmd/wave/commands/serve_stub.go`.

**Rationale**: The spec (C1) explicitly requires unconditional compilation. The `types.go` file already compiles unconditionally (no build tag), confirming the package can exist without the tag. All 22 tagged files are in `internal/webui/` and 1 in `cmd/wave/commands/serve.go`. The stub exists solely as a fallback when the tag is absent — it becomes dead code.

**Alternatives Rejected**:
- Keep an opt-out tag (`//go:build !nowebui`): Adds complexity for no practical benefit. Wave is "single static binary" per constitution.
- Conditional `embed.FS`: Would require splitting embed directives across tagged/untagged files — fragile.

**Impact**: Binary size increase from embedded assets (HTML/CSS/JS). Currently ~50KB of static files. Well under the 2MB SC-002 threshold.

**Files requiring build tag removal** (22 files):
- `internal/webui/auth.go`
- `internal/webui/auth_test.go`
- `internal/webui/dag.go`
- `internal/webui/dag_test.go`
- `internal/webui/embed.go`
- `internal/webui/handlers_artifacts.go`
- `internal/webui/handlers_control.go`
- `internal/webui/handlers_personas.go`
- `internal/webui/handlers_pipelines.go`
- `internal/webui/handlers_runs.go`
- `internal/webui/handlers_sse.go`
- `internal/webui/handlers_test.go`
- `internal/webui/middleware.go`
- `internal/webui/pagination.go`
- `internal/webui/pagination_test.go`
- `internal/webui/redact.go`
- `internal/webui/redact_test.go`
- `internal/webui/routes.go`
- `internal/webui/server.go`
- `internal/webui/sse.go`
- `internal/webui/sse_broker.go`
- `cmd/wave/commands/serve.go`

**File to delete**: `cmd/wave/commands/serve_stub.go`

## R-002: Retry Handler Execution Gap

**Decision**: Extract shared pipeline execution logic from `handleStartPipeline` into a `launchPipelineExecution(runID, pipelineName, input)` method, then call it from both `handleStartPipeline` and `handleRetryRun`.

**Rationale**: The current `handleRetryRun` (lines 207-248 in `handlers_control.go`) creates a new DB record but never launches execution. This violates User Story 2, Acceptance Scenario 3 ("execution begins"). The execution setup code in `handleStartPipeline` (lines 58-158) handles adapter resolution, emitter creation, audit logging, executor creation, and goroutine launch — all of which must be replicated for retry.

**Alternatives Rejected**:
- Duplicate the execution code in handleRetryRun: DRY violation, maintenance burden.
- Call handleStartPipeline internally: Wrong abstraction — the retry handler has different request parsing and validation.

## R-003: Resume-from-Step API

**Decision**: Add `POST /api/runs/{id}/resume` endpoint with body `{"from_step": "<step-id>", "force": false}`.

**Rationale**: The `ResumeManager` in `internal/pipeline/resume.go` already implements full resume logic (`ResumeFromStep`). The webui needs a thin HTTP handler that:
1. Validates the run exists and is in failed/cancelled state
2. Loads the pipeline YAML
3. Creates a new executor with `ResumeManager`
4. Calls `ResumeManager.ResumeFromStep()` with the provided step ID

The response returns a new `run_id` (matching `StartPipelineResponse` shape). The UI adds a "Resume from…" dropdown on the run detail page for failed/cancelled runs.

**Alternatives Rejected**:
- Reuse the retry endpoint with a query param: Conflates two distinct operations with different semantics (retry = full restart; resume = partial re-execution from a point).

## R-004: SSE Last-Event-ID Backfill

**Decision**: Add `id:` field to SSE events using the database event row ID. On reconnection, read `Last-Event-ID` header and backfill from the state store.

**Rationale**: The current SSE handler (`handlers_sse.go`) doesn't include event IDs and doesn't handle reconnection backfill. The `LogRecord` in `state/store.go` already has an `ID int64` field (auto-increment), and `GetEvents` supports `EventQueryOptions` which can be extended for `AfterID` filtering. The browser's built-in `EventSource` automatically sends `Last-Event-ID` on reconnect.

**Implementation**:
1. Modify `SSEBroker.EmitProgress` to include `ID` in `SSEEvent`
2. In `handleSSE`, parse `Last-Event-ID` header → query `GetEvents` with `AfterID` filter → send missed events before subscribing to live stream
3. Add `AfterID int64` field to `EventQueryOptions`

## R-005: Fallback Polling Mechanism (FR-005)

**Decision**: Add `GET /api/runs/{id}/poll` endpoint returning the current run state as JSON, to be used when SSE is unavailable.

**Rationale**: FR-005 requires a fallback when SSE connections fail. The simplest approach is a polling endpoint that returns the same data as `handleAPIRunDetail`. The frontend JavaScript can detect SSE failure and fall back to periodic polling (e.g., every 3 seconds). This requires minimal server-side work since `handleAPIRunDetail` already exists — the frontend just needs the fallback logic.

**Alternative**: WebSocket fallback — rejected as overkill; polling is sufficient for dashboard use cases.

## R-006: Template & Frontend Enhancements

**Decision**: Enhance existing Go HTML templates with additional UI components for resume, DAG interaction, and responsive layout. No JavaScript build step.

**Rationale**: The existing template system (layout + partials + per-page clones) is well-structured. New UI features (resume dropdown, DAG tooltips, keyboard navigation) can be implemented with:
- Additional Go template partials (`resume_dialog.html`, `dag_tooltip.html`)
- Vanilla JavaScript in `static/app.js` and `static/dag.js`
- CSS media queries in `static/style.css` for responsive layout
- ARIA attributes on interactive elements for accessibility

This preserves the "no JS build step" architecture and keeps everything embeddable via `//go:embed`.
