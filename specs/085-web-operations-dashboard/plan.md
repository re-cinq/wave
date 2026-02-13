# Implementation Plan: Web-Based Pipeline Operations Dashboard

**Branch**: `085-web-operations-dashboard` | **Date**: 2026-02-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/085-web-operations-dashboard/spec.md`

## Summary

Add a `wave serve` command that starts an embedded HTTP server providing a browser-based pipeline operations dashboard. The dashboard displays pipeline runs, real-time progress via SSE, step-level details with DAG visualization, artifact browsing, and execution control (start/stop/retry). All frontend assets are embedded in the Go binary via `go:embed`, using Go `html/template` for server-side rendering and vanilla JavaScript for interactivity. The feature is gated behind a `webui` build tag so it adds zero overhead when not compiled in. A read-only SQLite connection serves dashboard queries concurrently alongside the pipeline executor's write connection via WAL mode.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `net/http` stdlib (Go 1.22+ enhanced `ServeMux`), `html/template`, `go:embed`, `modernc.org/sqlite` (existing)
**Storage**: SQLite (existing `.wave/state.db`) — read-only connection for dashboard, read-write for execution control
**Testing**: `go test` with table-driven tests, `-race` flag required
**Target Platform**: Linux/macOS/Windows (single binary, same as existing Wave)
**Project Type**: Single Go project with embedded web frontend (server-side rendered)
**Performance Goals**: API responses <200ms for 1000 runs (NFR-005), SSE latency <2s (SC-003), JS <50 KB gzipped (NFR-001)
**Constraints**: Binary size increase <200 KB (NFR-002), no external CDN/runtime dependencies (FR-013), no Node.js build toolchain (C-003)
**Scale/Scope**: Single operator, 1-5 concurrent browser clients, up to 1000 pipeline runs

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | All assets embedded via `go:embed`. No new runtime dependencies. Build tag `webui` for opt-in. |
| P2: Manifest as SSOT | PASS | Dashboard reads personas and pipelines from manifest. No separate config. |
| P3: Persona-Scoped Execution | PASS | Dashboard executes pipelines through existing executor, which enforces persona boundaries. |
| P4: Fresh Memory at Step Boundaries | PASS | Dashboard does not alter pipeline execution model. Steps still get fresh context. |
| P5: Navigator-First | PASS | Pipeline execution initiated from dashboard follows same pipeline definitions. |
| P6: Contracts at Every Handover | PASS | Dashboard-triggered runs go through the same executor with contract validation. |
| P7: Relay via Summarizer | N/A | Dashboard does not interact with relay/compaction. |
| P8: Ephemeral Workspaces | PASS | Dashboard artifact browsing is read-only. Workspace lifecycle unchanged. |
| P9: Credentials Never Touch Disk | PASS | Bearer tokens via env var or flag only, never persisted. Artifact display redacts credentials. |
| P10: Observable Progress | PASS | Dashboard is a new consumer of existing progress events via SSE. Enhances observability. |
| P11: Bounded Recursion | N/A | Dashboard does not introduce new recursion paths. |
| P12: Minimal Step State Machine | PASS | No new step states. Dashboard reads existing 5 states. |
| P13: Test Ownership | PASS | All new code will have comprehensive tests. Existing tests must continue to pass. |

**Constitution Check: PASS — no violations.**

## Project Structure

### Documentation (this feature)

```
specs/085-web-operations-dashboard/
├── plan.md              # This file
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 entity definitions
├── contracts/           # Phase 1 API contracts
│   ├── api-runs-list.json
│   ├── api-run-detail.json
│   ├── api-execution-control.json
│   ├── api-sse-events.json
│   ├── api-artifacts.json
│   └── api-personas.json
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/
├── state/
│   └── readonly.go          # NewReadOnlyStateStore constructor
│
└── webui/                   # NEW PACKAGE — all dashboard code
    ├── server.go            # go:build webui — HTTP server, lifecycle, config
    ├── routes.go            # go:build webui — Route registration
    ├── handlers_runs.go     # go:build webui — Run list and detail handlers
    ├── handlers_control.go  # go:build webui — Start/cancel/retry handlers
    ├── handlers_artifacts.go # go:build webui — Artifact content handlers
    ├── handlers_personas.go # go:build webui — Persona list handler
    ├── handlers_sse.go      # go:build webui — SSE stream handler
    ├── sse_broker.go        # go:build webui — SSE client management and broadcast
    ├── auth.go              # go:build webui — Bearer token middleware
    ├── middleware.go        # go:build webui — CORS, security headers, logging
    ├── dag.go               # go:build webui — DAG layout computation
    ├── dag_svg.go           # go:build webui — SVG rendering for DAG
    ├── redact.go            # go:build webui — Credential redaction for artifacts
    ├── pagination.go        # go:build webui — Cursor encode/decode
    ├── types.go             # go:build webui — API response types
    ├── embed.go             # go:build webui — go:embed directives
    ├── static/              # Embedded static assets
    │   ├── app.js           # Main application JS (<5 KB)
    │   ├── sse.js           # SSE client with auto-reconnect (<2 KB)
    │   ├── dag.js           # DAG interaction (hover, click) (<3 KB)
    │   └── style.css        # Dashboard styles (<10 KB)
    └── templates/           # Embedded HTML templates
        ├── layout.html      # Base layout with nav, head, scripts
        ├── runs.html        # Pipeline run list page
        ├── run_detail.html  # Run detail with steps, DAG, events
        ├── personas.html    # Persona list page
        └── partials/
            ├── run_row.html     # Single run row (reusable)
            ├── step_card.html   # Step detail card
            ├── dag_svg.html     # DAG SVG template
            └── artifact_viewer.html # Artifact content display

cmd/wave/commands/
├── serve.go             # go:build webui — Real serve command
└── serve_stub.go        # go:build !webui — Error stub

tests/
└── webui/               # Dashboard-specific tests
    ├── server_test.go
    ├── handlers_test.go
    ├── sse_test.go
    ├── auth_test.go
    ├── dag_test.go
    ├── redact_test.go
    └── pagination_test.go
```

**Structure Decision**: Single Go project with a new `internal/webui/` package containing all dashboard code. This follows the existing pattern of `internal/<feature>/` packages (e.g., `internal/display/`, `internal/tui/`). The `webui` build tag gates the entire package. Test files in `tests/webui/` follow the existing `tests/` convention for integration tests, while unit tests live alongside source files.

## Key Design Decisions

### D-001: HTTP Server (R-001)
Standard library `net/http` with Go 1.22+ `ServeMux` for method-based routing. No third-party router dependency.

### D-002: SSE Architecture (R-002)
Broker/hub pattern implementing `event.ProgressEmitter` interface. The SSE broker registers with the pipeline executor's event emitter and fans out events to connected browser clients via channels. Each client gets a dedicated goroutine and channel. Disconnection detected via `r.Context().Done()`.

### D-003: Database Access (R-003)
New `NewReadOnlyStateStore(dbPath)` in `internal/state/readonly.go` opens the database with `?mode=ro`, `PRAGMA query_only=ON`, and `MaxOpenConns(10)` for concurrent HTTP handlers. Write operations for execution control use a separate `NewStateStore` connection.

### D-004: Build Tag (R-004)
`//go:build webui` on all files in `internal/webui/` and `cmd/wave/commands/serve.go`. Stub file `serve_stub.go` with `//go:build !webui` prints error message. Build command: `go build -tags webui ./cmd/wave`.

### D-005: DAG Visualization (R-005)
Server-side SVG generation via Go templates. Topological sort for layer assignment, simple grid positioning. Nodes are SVG `<rect>` elements with status-colored fills. Edges are SVG `<path>` elements with bezier curves. Interactive features (hover tooltips, click navigation) via minimal JS event handlers on SVG elements.

### D-006: Authentication (R-006)
Bearer token middleware applied only when bind address is not localhost. Token from `--token` flag, `WAVE_SERVE_TOKEN` env, or auto-generated. Displayed at startup on stderr.

### D-007: Pagination (R-009)
Cursor-based using `(started_at, run_id)` composite key. Cursor encoded as base64 JSON. Extends existing `ListRuns` query with `WHERE (started_at < ? OR (started_at = ? AND run_id < ?))`.

### D-008: Credential Redaction (R-008)
Regex-based pattern matching for common credential formats (AWS keys, API tokens, passwords). Applied before rendering artifact content. Builds on patterns from `internal/security/sanitize.go`.

### D-009: Execution Control Architecture
Pipeline start/cancel/retry from the dashboard use the same code paths as CLI commands:
- **Start**: Load pipeline definition, create `PipelineExecutor`, call `Execute()` in a goroutine. Register run in state DB.
- **Cancel**: Call `RequestCancellation()` on the state store. The executor's cancellation polling detects this.
- **Retry**: Read original run's pipeline name and input, create a new run with same parameters.

The server holds references to the manifest, pipeline loader, adapter resolver, and state store — same components used by `cmd/wave/commands/run.go`.

## Complexity Tracking

_No constitution violations. No complexity justifications needed._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none) | — | — |
