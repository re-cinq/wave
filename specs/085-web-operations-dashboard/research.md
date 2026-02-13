# Research: Web-Based Pipeline Operations Dashboard

**Branch**: `085-web-operations-dashboard` | **Date**: 2026-02-13

## Phase 0 — Outline & Research

### Unknowns Extracted from Spec

1. **HTTP server framework choice** — Use `net/http` stdlib or a router library?
2. **SSE implementation** — How to bridge the existing `event.EventEmitter` to SSE streams?
3. **Read-only SQLite connection** — How to create `NewReadOnlyStateStore` alongside the existing writer?
4. **Build tag gating** — How to structure `//go:build webui` across multiple files?
5. **DAG visualization** — How to render pipeline DAGs with vanilla JS under 50 KB?
6. **Authentication mechanism** — Bearer token lifecycle for non-localhost binding.
7. **Template organization** — How to structure `go:embed` for HTML templates + static assets.

---

### R-001: HTTP Server and Routing

**Decision**: Use Go standard library `net/http` with `http.ServeMux` (Go 1.22+ enhanced routing).

**Rationale**: Go 1.22+ `ServeMux` supports method-based routing (`GET /api/runs`) and path parameters (`GET /api/runs/{id}`), eliminating the need for a third-party router. This aligns with Constitution Principle 1 (single binary, minimal dependencies) and avoids adding a new dependency like `chi` or `gorilla/mux`. The existing project already uses only `cobra` and `yaml.v3` as external dependencies.

**Alternatives Rejected**:
- `chi` router: Adds a dependency for routing patterns that stdlib now covers.
- `gorilla/mux`: Archived project, not appropriate for new code.
- `gin`/`echo`: Full frameworks — far too heavy for a single-binary tool.

---

### R-002: Server-Sent Events (SSE) Implementation

**Decision**: Implement SSE using stdlib `net/http` with `http.Flusher` interface. Create a broker/hub pattern that bridges the existing `event.EventEmitter` to connected SSE clients.

**Rationale**: SSE is simpler than WebSockets for server-to-client push and natively supported by browsers via `EventSource`. The existing `event.Event` struct already contains all fields needed for progress updates (progress, step state, persona, tokens, etc.). The `NDJSONEmitter` pattern can be extended with a new `SSEBroadcaster` that implements the `ProgressEmitter` interface and fans out events to all connected HTTP clients.

**Architecture**:
```
Pipeline Executor → EventEmitter.Emit(event) → SSEBroadcaster.EmitProgress(event)
                                                      ↓
                                               Connected SSE clients
                                               (fan-out to all browsers)
```

The SSE broker manages client connections with a channel-per-client pattern:
- New client connects → register channel
- Client disconnects → deregister channel (detect via `r.Context().Done()`)
- Event arrives → broadcast to all registered channels (non-blocking send)

**Alternatives Rejected**:
- WebSockets: Bidirectional communication not needed; SSE is simpler, supports auto-reconnect natively, and doesn't require a JS library.
- Polling: Higher latency, more server load, worse user experience.
- Third-party SSE library: Unnecessary given the simplicity of the SSE protocol.

---

### R-003: Read-Only SQLite Connection

**Decision**: Add a `NewReadOnlyStateStore` constructor to `internal/state/` that opens the database with `?mode=ro` URI parameter and `PRAGMA query_only=ON`. Use higher `MaxOpenConns` (e.g., 10) to support concurrent HTTP handlers.

**Rationale**: The existing `NewStateStore` uses `SetMaxOpenConns(1)` because it's the single writer for pipeline execution. The dashboard needs concurrent read access from multiple HTTP handlers without contending with the executor's write lock. SQLite WAL mode (already enabled in `store.go:129`) supports unlimited concurrent readers alongside a single writer. The `?mode=ro` parameter ensures the dashboard connection cannot accidentally write to the database.

**Implementation**:
```go
func NewReadOnlyStateStore(dbPath string) (StateStore, error) {
    db, err := sql.Open("sqlite", dbPath+"?mode=ro")
    // SetMaxOpenConns(10) for concurrent HTTP handlers
    // PRAGMA query_only=ON as defense-in-depth
    // PRAGMA journal_mode=WAL (required for concurrent reads)
    // Skip migration initialization (read-only)
}
```

The `StateStore` interface already contains all the read methods the dashboard needs (`ListRuns`, `GetRun`, `GetEvents`, `GetArtifacts`, `GetAllStepProgress`, `GetPipelineProgress`). Write methods will return errors on the read-only connection.

**Alternatives Rejected**:
- Sharing the same connection: Would require `MaxOpenConns > 1` on the writer, changing its locking behavior.
- Separate read interface: Unnecessary complexity; the existing `StateStore` interface works.

---

### R-004: Build Tag Gating (`//go:build webui`)

**Decision**: Use a `webui` build tag with file-pair pattern: `serve.go` (with tag) contains the real implementation, `serve_stub.go` (without tag) contains the error stub.

**Rationale**: This is the standard Go pattern for optional features. Files guarded by `//go:build webui` are only compiled when `go build -tags webui` is used. The stub file prints a clear error message when `wave serve` is run on a binary built without the tag. This approach is used by projects like CockroachDB and Hugo for optional features.

**File Structure**:
```
internal/webui/
├── server.go          // go:build webui — HTTP server, routes, middleware
├── sse.go             // go:build webui — SSE broker implementation
├── handlers.go        // go:build webui — API handlers
├── templates.go       // go:build webui — Template loading via go:embed
├── auth.go            // go:build webui — Bearer token auth middleware
├── static/            // Embedded static assets (JS, CSS)
│   ├── app.js
│   ├── sse.js
│   ├── dag.js
│   └── style.css
└── templates/         // Embedded HTML templates
    ├── layout.html
    ├── runs.html
    ├── run_detail.html
    ├── personas.html
    └── partials/
        ├── run_row.html
        ├── step_card.html
        └── dag.html

cmd/wave/commands/
├── serve.go           // go:build webui — Real serve command
└── serve_stub.go      // go:build !webui — Error stub
```

**Alternatives Rejected**:
- Runtime feature flag: Would embed assets in all builds, violating FR-014 and NFR-002.
- Plugin system: Overcomplicated for a single optional feature.

---

### R-005: DAG Visualization

**Decision**: Server-side SVG generation using Go's `html/template` package. The pipeline step graph is computed on the server and rendered as an inline SVG in the HTML response.

**Rationale**: The spec requires DAG rendering (FR-010) under a 50 KB JS budget (NFR-001). Server-side SVG generation requires zero JavaScript for the static graph. Interactive features (hover for status, click for details) can be achieved with minimal JS using SVG event handlers. Pipeline DAGs are typically small (3-10 nodes), so the SVG is lightweight.

**Layout Algorithm**: Topological sort → layer assignment (Sugiyama-style) → simple left-to-right or top-to-bottom positioning. Each node is a rounded rectangle with status color. Edges are SVG `<path>` elements with simple bezier curves.

**Alternatives Rejected**:
- D3.js: 80+ KB minified, exceeds the entire JS budget.
- Mermaid.js: 300+ KB, designed for general-purpose diagrams.
- Cytoscape.js: 200+ KB, graph-specific but too large.
- Canvas-based: Requires more JS than SVG, no native interactivity.

---

### R-006: Authentication for Non-Localhost

**Decision**: Static bearer token via `--token` flag or `WAVE_SERVE_TOKEN` env var. Auto-generate a random 32-byte hex token at startup if none provided. Apply auth middleware only when `--bind` is not `127.0.0.1`/`localhost`.

**Rationale**: Per C-001 in the spec, Wave is a single-operator tool. The codebase has no existing auth infrastructure (`internal/security/` contains input sanitization and path validation, not authentication). A bearer token provides protection against casual LAN access without introducing session management, user databases, or OAuth complexity.

**Token Lifecycle**:
1. Server starts with `--bind 0.0.0.0`
2. If `--token` flag or `WAVE_SERVE_TOKEN` env not set → generate random token
3. Print token to stderr: `Dashboard token: <token>` (similar to Jupyter Notebook)
4. All API endpoints require `Authorization: Bearer <token>` header
5. HTML pages include token in a meta tag for JS to pick up (or use cookie)

**Alternatives Rejected**:
- mTLS: Complex setup, not appropriate for a development tool.
- OAuth: Requires external provider, way too complex.
- Basic auth: Less secure than bearer tokens, prompts browser dialogs.

---

### R-007: Frontend Template Organization

**Decision**: Go `html/template` with a base layout template and partial templates. Static assets (JS, CSS) embedded via `go:embed`. Templates use server-side rendering for initial page load; HTMX-style patterns considered but rejected in favor of vanilla JS + SSE for real-time updates.

**Rationale**: Per FR-013 and C-003, the frontend uses server-side rendering with Go templates and vanilla JS. The existing `go:embed` pattern in `internal/defaults/embed.go` provides the precedent. All templates and static files are embedded in the binary at compile time, requiring zero external requests (SC-007).

**Page Structure**:
- `/` — Dashboard home, redirects to `/runs`
- `/runs` — Pipeline run list with pagination, filtering, status indicators
- `/runs/{id}` — Run detail view with step list, DAG, events timeline
- `/runs/{id}/artifacts/{step}/{name}` — Artifact viewer
- `/personas` — Persona list and detail
- `/api/runs` — JSON API for run list (used by SSE update logic)
- `/api/runs/{id}` — JSON API for run detail
- `/api/runs/{id}/events` — SSE endpoint for real-time updates
- `/api/pipelines/{name}/start` — POST: Start pipeline
- `/api/runs/{id}/cancel` — POST: Cancel run
- `/api/runs/{id}/retry` — POST: Retry failed run
- `/static/*` — Embedded static assets

**Alternatives Rejected**:
- React/Vue/Svelte SPA: Requires Node.js build toolchain, conflicts with single-binary philosophy.
- HTMX: Adds a 14 KB dependency; vanilla JS + SSE achieves the same with less.
- Web Components: More JS complexity than needed for this scope.

---

### R-008: Credential Redaction for Artifact Display

**Decision**: Reuse the existing `security.InputSanitizer.removeSuspiciousContent()` pattern from `internal/security/sanitize.go` and add a credential-pattern regex matcher for artifact content display.

**Rationale**: The spec requires credential redaction (SR-005, FR-016) when displaying artifacts. The existing security package already has sanitization infrastructure. A dedicated `RedactCredentials(content string) string` function can be added that matches patterns like API keys, tokens, passwords, and AWS/GCP credential formats.

**Patterns to Redact**:
- `AKIA[A-Z0-9]{16}` (AWS access keys)
- `[a-zA-Z0-9/+=]{40}` adjacent to "secret" (AWS secret keys)
- `sk-[a-zA-Z0-9]{48}` (OpenAI/Anthropic keys)
- `ghp_[a-zA-Z0-9]{36}` (GitHub PATs)
- `password\s*[:=]\s*\S+` (inline passwords)
- Generic `Bearer [a-zA-Z0-9._-]+` patterns

---

### R-009: Cursor-Based Pagination

**Decision**: Cursor-based pagination using composite key `(started_at, run_id)`. Encode cursor as base64 JSON `{"t": <unix_timestamp>, "id": "<run_id>"}`. API accepts `?cursor=<encoded>&limit=25`.

**Rationale**: Per C-005 in the spec, cursor-based pagination provides stable results when new runs are created. The existing `ListRuns` query already sorts by `started_at DESC`. Adding a `WHERE started_at < ? OR (started_at = ? AND run_id < ?)` clause with the cursor values implements keyset pagination efficiently.

**Alternatives Rejected**:
- Offset-based: Unstable when new rows are inserted, causes duplicate/skipped items.
- Page number: Same issues as offset-based.
