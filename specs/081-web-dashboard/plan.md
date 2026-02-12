# Implementation Plan: Web Dashboard (#81)

## Objective

Add a `wave serve` command that starts an embedded HTTP server serving a read-only web dashboard for monitoring pipeline executions. The dashboard reads from the existing SQLite state database and provides real-time progress updates via Server-Sent Events (SSE).

## Approach

Build the feature in layers:

1. **Go HTTP server** (`internal/dashboard/`) -- REST API endpoints + SSE stream + static file serving via `go:embed`
2. **Frontend** (`web/`) -- Preact SPA built with Vite/Bun, producing a minimal JS bundle embedded at compile time
3. **CLI command** (`cmd/wave/commands/serve.go`) -- `wave serve` cobra command wiring up the server
4. **Event bridge** -- Subscribe to the existing `event.EventEmitter` system to push real-time updates to SSE clients

The Go server uses the standard library `net/http` (Go 1.22+ enhanced ServeMux) with no additional HTTP framework dependencies. The frontend is Preact (~3 KB) bundled by Vite, targeting < 50 KB gzipped total.

## File Mapping

### New Files

| Path | Purpose |
|------|---------|
| `internal/dashboard/server.go` | HTTP server setup, mux configuration, middleware |
| `internal/dashboard/server_test.go` | Server unit tests |
| `internal/dashboard/handlers.go` | REST API handler functions |
| `internal/dashboard/handlers_test.go` | Handler unit tests |
| `internal/dashboard/sse.go` | SSE event broker -- fan-out to connected clients |
| `internal/dashboard/sse_test.go` | SSE broker tests |
| `internal/dashboard/embed.go` | `go:embed` directives for static assets |
| `internal/dashboard/types.go` | API request/response types |
| `cmd/wave/commands/serve.go` | `wave serve` cobra command |
| `cmd/wave/commands/serve_test.go` | Serve command tests |
| `web/` | Frontend source directory (Preact + Vite) |
| `web/package.json` | Bun/npm dependencies |
| `web/vite.config.ts` | Vite build configuration |
| `web/index.html` | Entry HTML |
| `web/src/app.tsx` | Root Preact component |
| `web/src/components/PipelineList.tsx` | Pipeline run list view |
| `web/src/components/PipelineDetail.tsx` | Single pipeline run detail view |
| `web/src/components/StepDetail.tsx` | Step-level detail panel |
| `web/src/components/StatusBadge.tsx` | Status indicator component |
| `web/src/hooks/useSSE.ts` | SSE connection hook |
| `web/src/hooks/useApi.ts` | REST API fetch hook |
| `web/src/styles/` | CSS styles |

### Modified Files

| Path | Change |
|------|--------|
| `cmd/wave/main.go` | Add `rootCmd.AddCommand(commands.NewServeCmd())` |
| `internal/state/store.go` | Add read-only query methods needed by dashboard API (if not already covered) |

### No Changes Required

| Path | Reason |
|------|--------|
| `internal/event/emitter.go` | Already has `EventEmitter` interface -- dashboard SSE broker implements `ProgressEmitter` |
| `internal/state/types.go` | Existing types (`RunRecord`, `LogRecord`, `ArtifactRecord`, `StepProgressRecord`) cover dashboard needs |
| `internal/state/schema.sql` | Existing schema has all required tables |

## Architecture Decisions

### AD-1: SSE over WebSocket
SSE is simpler (HTTP/1.1 compatible, automatic reconnection, unidirectional), aligns with the read-only Phase 1 scope, and avoids a WebSocket dependency. The existing `event.Event` struct serializes cleanly to SSE `data:` fields as JSON.

### AD-2: Preact + Vite + Bun
Preact gives a React-compatible API at ~3 KB. Vite produces optimized bundles. Bun replaces Node.js for build-time tooling per author feedback. The built assets are committed to `internal/dashboard/static/` so `go build` works without Bun/Node.js at compile time.

### AD-3: Embed pre-built assets
The `web/` directory builds to `internal/dashboard/static/`. A `//go:embed static/*` directive in `embed.go` makes assets available at runtime. This preserves the single-binary constraint. Developers who modify the frontend run `bun run build` to regenerate assets before `go build`.

### AD-4: Read-only SQLite access
The dashboard opens the SQLite database in read-only mode (`?mode=ro`) with WAL journal mode, allowing concurrent reads while the pipeline executor writes. This avoids any locking contention.

### AD-5: Localhost-only by default
The server binds to `127.0.0.1` by default. A `--bind` flag allows overriding to `0.0.0.0` for remote access. No authentication is implemented in Phase 1.

### AD-6: Reuse existing state package
The dashboard API reuses `internal/state.StateStore` methods (`ListRuns`, `GetRun`, `GetEvents`, `GetArtifacts`, `GetAllStepProgress`, `GetPipelineProgress`) rather than writing raw SQL. This keeps the data access layer consistent.

## REST API Design

```
GET  /api/runs                 -- List pipeline runs (with ?status=, ?pipeline=, ?limit= filters)
GET  /api/runs/:id             -- Get single run details
GET  /api/runs/:id/events      -- Get events for a run
GET  /api/runs/:id/steps       -- Get step progress for a run
GET  /api/runs/:id/artifacts   -- Get artifacts for a run
GET  /api/events/stream        -- SSE stream for real-time updates
GET  /                         -- Serve dashboard SPA (index.html)
GET  /static/*                 -- Serve embedded static assets
```

## Risks

| Risk | Mitigation |
|------|-----------|
| Bundle size exceeds 50 KB target | Preact is ~3 KB, and Vite tree-shakes aggressively. Monitor with `vite-plugin-compression`. Drop to vanilla JS if needed. |
| Binary size increase from embedded assets | Pre-compress with gzip/brotli, expect < 100 KB increase. Acceptable for the functionality gained. |
| SQLite locking under concurrent read/write | Use `?mode=ro` for dashboard reads. WAL mode allows concurrent readers. Separate connection pool. |
| SSE connection scaling | Single-process fan-out is sufficient for monitoring use case (< 10 concurrent viewers expected). |
| Frontend build toolchain complexity | Pre-built assets committed to repo. `go build` works standalone. Bun only needed for frontend development. |
| `needs-design` label -- design not finalized | Phase 1 is read-only dashboard with clear requirements. Defer design-heavy features (DAG viz, execution control) to future phases. |

## Testing Strategy

### Unit Tests
- `internal/dashboard/server_test.go` -- server startup, shutdown, configuration
- `internal/dashboard/handlers_test.go` -- each API endpoint with mock store
- `internal/dashboard/sse_test.go` -- SSE broker subscribe/unsubscribe/broadcast
- `cmd/wave/commands/serve_test.go` -- command flag parsing, validation

### Integration Tests
- Start server, hit API endpoints, verify JSON responses match state DB
- SSE connection receives events when pipeline state changes
- Embedded static assets are served correctly with proper content types

### Frontend Tests
- Build output verification: bundle size < 50 KB gzipped
- Accessibility: semantic HTML, keyboard navigation
- Component rendering with mock data

### Manual Verification
- `wave serve` starts successfully, dashboard loads in browser
- Real-time updates visible when running a pipeline concurrently
- Filter and search functionality works
