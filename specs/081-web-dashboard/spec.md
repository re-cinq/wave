# Add web-based pipeline monitoring dashboard with minimal JavaScript footprint

**Issue**: [#81](https://github.com/re-cinq/wave/issues/81)
**Labels**: enhancement, frontend, needs-design
**Author**: nextlevelshit
**State**: OPEN

## Summary

Add a lightweight web UI for monitoring and managing Wave pipeline executions from a browser, using the smallest practical JavaScript footprint.

## Motivation

Wave currently operates exclusively via CLI. A web dashboard would allow operators to monitor pipeline status, view execution history, and inspect step outputs without terminal access -- useful for team visibility and remote monitoring.

## Functional Requirements

- [ ] Display a list of pipeline runs with status (pending, running, completed, failed)
- [ ] Show real-time progress updates for active pipeline executions
- [ ] View step-level details: persona, duration, contract validation results
- [ ] Browse workspace artifacts and step outputs
- [ ] Filter and search pipeline history

## Non-Functional Requirements

- [ ] Minimal JavaScript bundle size (target: < 50 KB gzipped)
- [ ] No Node.js runtime dependency -- UI assets embedded in the Go binary via `go:embed`
- [ ] Single binary deployment preserved (no separate frontend server)
- [ ] Responsive layout for desktop and tablet browsers
- [ ] Accessible without authentication on localhost (optional auth for remote access)

## Design Constraints

- **Integration**: Served by an HTTP handler within the existing Wave binary
- **Data source**: Read from the SQLite state database and filesystem workspaces
- **Live updates**: WebSocket or SSE for real-time pipeline progress
- **Embedding**: Static assets compiled into the binary using `go:embed`

## Framework Considerations

Original candidates from this issue:
- **Svelte** -- Small bundle output, compiler-based, good fit for embedded UIs
- **Vue** -- Mature ecosystem, moderate bundle size
- **Preact** -- React-compatible API at ~3 KB, strong candidate for minimal JS
- **Vanilla JS / Web Components** -- Zero framework overhead, highest control

> **Note**: Vite is a build tool, not a UI framework. It can be used with any of the above options for development and bundling.

Recommendation from research comment: Preact with Vite bundling, SSE for real-time updates.

## Scope

### In Scope (Phase 1)

- Read-only dashboard: pipeline list, status, step details
- Embedded HTTP server with `go:embed` assets
- WebSocket/SSE for live progress updates
- `wave serve` standalone command

### Out of Scope (Future)

- Pipeline execution control (start, stop, retry) from the UI (author requested for future)
- Configuration editing through the UI
- Multi-user authentication and authorization
- Mobile-optimized layouts
- Persona management UI
- DAG visualization

## Acceptance Criteria

1. `wave serve` starts an HTTP server exposing the dashboard on a configurable port
2. Dashboard displays all pipeline runs from the state database
3. Active pipeline progress updates in real-time without page refresh
4. Total JavaScript bundle is under 50 KB gzipped
5. All assets are embedded -- no external CDN or runtime dependencies
6. Existing CLI functionality is unaffected

## Open Questions

- Should the web UI be opt-in (behind a build tag) or always included in the binary?
- What is the acceptable binary size increase from embedded assets?
- Should the dashboard expose the same structured events used by the CLI display?

## Research Notes (from issue comments)

- Use Bun instead of npm/Node.js for build tooling
- Preact recommended as framework (~3 KB)
- SSE preferred over WebSocket for real-time updates (simpler, auto-reconnect)
- go:embed with pre-compression for static assets
- Go 1.22+ ServeMux for HTTP routing
- Separate read/write SQLite pools for concurrent access
- Localhost-only binding by default for security
