# Task Breakdown: Web-Based Pipeline Operations Dashboard

**Branch**: `085-web-operations-dashboard` | **Date**: 2026-02-13
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Phase 1: Setup & Build Tag Infrastructure

- [ ] T001 [P1] [US3] Create `internal/webui/` package with build-tag-gated server skeleton, `embed.go` with `go:embed` directives for `static/` and `templates/` directories, and `types.go` with all API response types from data-model.md (`RunListResponse`, `RunSummary`, `RunDetailResponse`, `StepDetail`, `DAGData`, `DAGNode`, `DAGEdge`, `PaginationCursor`). Files: `internal/webui/server.go`, `internal/webui/embed.go`, `internal/webui/types.go`
- [ ] T002 [P1] [US3] Create `wave serve` CLI command with `--port` (default 8080) and `--bind` (default 127.0.0.1) flags, plus build-tag stub that prints error when compiled without `webui` tag. Files: `cmd/wave/commands/serve.go`, `cmd/wave/commands/serve_stub.go`
- [ ] T003 [P1] [US3] Add `NewReadOnlyStateStore` constructor in `internal/state/readonly.go` that opens SQLite with `?mode=ro`, `PRAGMA query_only=ON`, `MaxOpenConns(10)`, and WAL mode — skipping migrations. File: `internal/state/readonly.go`

## Phase 2: Core Dashboard — Monitor Pipeline Runs (US1)

- [ ] T004 [P1] [US1] Create base HTML layout template (`templates/layout.html`) with navigation, CSS, and JS script tags, plus `templates/runs.html` for the pipeline run list page with status badges, timing, pagination controls, and status filter UI. Create `static/style.css` with responsive dashboard styles. Files: `internal/webui/templates/layout.html`, `internal/webui/templates/runs.html`, `internal/webui/static/style.css`
- [ ] T005 [P1] [US1] Implement `pagination.go` with cursor encode/decode (`base64(JSON{t,id})`), and `handlers_runs.go` with `GET /api/runs` (JSON, cursor-paginated, filterable by status/pipeline/time) and `GET /runs` (HTML run list page). Implement `routes.go` to register all routes on `http.ServeMux`. Files: `internal/webui/pagination.go`, `internal/webui/handlers_runs.go`, `internal/webui/routes.go`
- [ ] T006 [P1] [US1] Create `templates/run_detail.html` and partials (`partials/step_card.html`, `partials/run_row.html`) for the run detail view showing steps, events timeline, and error messages. Implement `GET /api/runs/{id}` and `GET /runs/{id}` handlers in `handlers_runs.go`. Files: `internal/webui/templates/run_detail.html`, `internal/webui/templates/partials/step_card.html`, `internal/webui/templates/partials/run_row.html`

## Phase 3: Real-Time Progress via SSE (US2)

- [ ] T007 [P] [P1] [US2] Implement SSE broker (`sse_broker.go`) with channel-per-client fan-out pattern implementing `event.ProgressEmitter` interface: client registration/deregistration, non-blocking broadcast, disconnect detection via `r.Context().Done()`. Implement SSE handler (`handlers_sse.go`) for `GET /api/runs/{id}/events` with `text/event-stream` content type using `http.Flusher`. File: `internal/webui/sse_broker.go`, `internal/webui/handlers_sse.go`
- [ ] T008 [P] [P1] [US2] Create `static/app.js` with run list auto-refresh and `static/sse.js` with `EventSource` client including auto-reconnect within 5 seconds (NFR-004), DOM update on progress events, and status badge transitions. Files: `internal/webui/static/app.js`, `internal/webui/static/sse.js`

## Phase 4: Security & Middleware

- [ ] T009 [P] [P1] [US3] Implement `auth.go` with bearer token middleware (applied only when bind is not localhost), token from `--token` flag / `WAVE_SERVE_TOKEN` env / auto-generated random 32-byte hex. Implement `middleware.go` with CORS (same-origin for localhost), security headers (X-Frame-Options, X-Content-Type-Options, CSP), and request logging. Wire `--token` flag into `serve.go` command. Files: `internal/webui/auth.go`, `internal/webui/middleware.go`

## Phase 5: Execution Control (US4)

- [ ] T010 [P2] [US4] Implement `handlers_control.go` with `POST /api/pipelines/{name}/start` (load pipeline from manifest, create run, execute in goroutine), `POST /api/runs/{id}/cancel` (call `RequestCancellation`), and `POST /api/runs/{id}/retry` (read original run params, create new run). Add input validation and error responses. File: `internal/webui/handlers_control.go`
- [ ] T011 [P2] [US4] Add pipeline start form UI to the dashboard — pipeline selector dropdown populated from manifest, input textarea, and start button. Add stop/retry buttons to run detail view. Handle form submissions with vanilla JS `fetch()`. Files: `internal/webui/templates/runs.html` (update), `internal/webui/templates/run_detail.html` (update), `internal/webui/static/app.js` (update)

## Phase 6: DAG Visualization (US5)

- [ ] T012 [P2] [US5] Implement `dag.go` with topological sort and Sugiyama-style layer assignment for pipeline step dependencies, computing (x, y) coordinates for each node. Implement `dag_svg.go` with SVG rendering via Go template: nodes as rounded `<rect>` with status-colored fills, edges as bezier `<path>` elements. Create `templates/partials/dag_svg.html` template. Files: `internal/webui/dag.go`, `internal/webui/dag_svg.go`, `internal/webui/templates/partials/dag_svg.html`
- [ ] T013 [P] [P2] [US5] Create `static/dag.js` with SVG interaction handlers: hover tooltips showing step details (persona, duration, tokens), click-to-navigate to step detail, and status-color legend. File: `internal/webui/static/dag.js`

## Phase 7: Artifact Browsing & Personas (US6, US7)

- [ ] T014 [P] [P3] [US6] Implement `redact.go` with credential pattern matching (AWS keys, OpenAI/Anthropic keys, GitHub PATs, inline passwords, Bearer tokens) applied before rendering artifact content. File: `internal/webui/redact.go`
- [ ] T015 [P] [P3] [US6] Implement `handlers_artifacts.go` with `GET /runs/{id}/artifacts/{step}/{name}` serving artifact content with path traversal prevention (validate against workspace root), size truncation for files >1 MB, syntax highlighting hints, and credential redaction. Create `templates/partials/artifact_viewer.html`. Files: `internal/webui/handlers_artifacts.go`, `internal/webui/templates/partials/artifact_viewer.html`
- [ ] T016 [P] [P3] [US7] Implement `handlers_personas.go` with `GET /personas` (HTML) and `GET /api/personas` (JSON) listing all personas from manifest with description, adapter, model, and permission rules (allowed/denied tools). Create `templates/personas.html`. Files: `internal/webui/handlers_personas.go`, `internal/webui/templates/personas.html`

## Phase 8: Testing & Polish

- [ ] T017 [P1] Implement unit tests for core packages: `readonly.go` (read-only store), `pagination.go` (cursor encode/decode/edge cases), `redact.go` (credential patterns), `dag.go` (topological sort, layout), `auth.go` (token validation, localhost bypass). Files: `internal/state/readonly_test.go`, `internal/webui/pagination_test.go`, `internal/webui/redact_test.go`, `internal/webui/dag_test.go`, `internal/webui/auth_test.go`
- [ ] T018 [P1] Implement integration tests for HTTP handlers: run list with pagination, run detail, SSE streaming, execution control endpoints, artifact serving with path traversal rejection, persona listing. Verify graceful shutdown on interrupt. File: `internal/webui/handlers_test.go`, `internal/webui/sse_test.go`
- [ ] T019 Verify build tag isolation: ensure `go build ./cmd/wave` (without `-tags webui`) succeeds and produces a binary where `wave serve` prints the expected error stub message. Ensure `go test ./...` passes without the tag. Ensure existing commands work identically. File: `internal/webui/build_tag_test.go`
- [ ] T020 Run `go vet ./...`, `go test -race ./...`, verify JS bundle sizes are under 50 KB, verify no regressions in existing test suite.

---

## Dependency Graph

```
T001 ──┬──> T002 (serve command needs server package)
       ├──> T004 (templates need embed.go)
       └──> T005 (handlers need types, server)
T003 ──────> T005 (handlers need read-only store)
T004 ──────> T006 (detail templates extend layout)
T005 ──┬──> T006 (detail handlers extend run handlers)
       └──> T007 (SSE broker integrates with server)
T007 ──────> T008 (JS client needs SSE endpoint)
T005 ──────> T009 (middleware wraps routes)
T005 ──────> T010 (control handlers extend run routes)
T010 ──────> T011 (UI needs control endpoints)
T006 ──────> T012 (DAG renders in detail view)
T012 ──────> T013 (DAG JS needs SVG template)
T014 ──────> T015 (artifact handler uses redact)
T001 ──────> T016 (persona handler needs server)
T001-T016 ──> T017 (unit tests cover all packages)
T017 ──────> T018 (integration tests after unit tests)
T018 ──────> T019 (build tag test after all code)
T019 ──────> T020 (final verification)
```

## Parallelization Notes

Tasks marked with `[P]` can run in parallel with other tasks in the same phase:
- **Phase 3**: T007 and T008 can proceed in parallel (backend SSE + frontend JS)
- **Phase 4**: T009 can proceed in parallel with Phase 3 work
- **Phase 6**: T013 can proceed in parallel once T012 is done
- **Phase 7**: T014, T015, and T016 are all independent of each other
- **Phase 8**: T017 tests can start as soon as the code they test is written

## Task-to-Story Coverage

| User Story | Tasks | Coverage |
|------------|-------|----------|
| US1 (Monitor Runs) | T004, T005, T006 | Run list, detail, pagination, filtering |
| US2 (Real-Time SSE) | T007, T008 | SSE broker, JS client, auto-reconnect |
| US3 (Server Command) | T001, T002, T003, T009 | Package setup, CLI, read-only store, auth |
| US4 (Execution Control) | T010, T011 | Start/stop/retry API, dashboard UI |
| US5 (DAG Visualization) | T012, T013 | Layout algorithm, SVG render, interactions |
| US6 (Artifact Browsing) | T014, T015 | Credential redaction, artifact viewer |
| US7 (View Personas) | T016 | Persona list page |
| Cross-cutting | T017, T018, T019, T020 | Testing, verification |
