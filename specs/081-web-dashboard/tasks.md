# Tasks

## Phase 1: Go HTTP Server Infrastructure

- [X] Task 1.1: Create `internal/dashboard/types.go` with API request/response types
- [X] Task 1.2: Create `internal/dashboard/server.go` with HTTP server setup, mux routing, middleware (CORS, content-type), graceful shutdown
- [X] Task 1.3: Create `internal/dashboard/handlers.go` with REST API handlers (runs list, run detail, events, steps, artifacts)
- [X] Task 1.4: Create `internal/dashboard/sse.go` with SSE event broker (client registration, fan-out broadcast, heartbeat)
- [X] Task 1.5: Create `internal/dashboard/embed.go` with `go:embed` directives and static file serving

## Phase 2: CLI Command

- [X] Task 2.1: Create `cmd/wave/commands/serve.go` with `wave serve` cobra command (--port, --bind flags)
- [X] Task 2.2: Register serve command in `cmd/wave/main.go`

## Phase 3: Frontend Build Setup

- [X] Task 3.1: Initialize `web/` directory with Bun, Preact, Vite, TypeScript configuration [P]
- [X] Task 3.2: Configure Vite build to output to `internal/dashboard/static/` with compression [P]

## Phase 4: Frontend Components

- [X] Task 4.1: Create root app component with routing (PipelineList / PipelineDetail views) [P]
- [X] Task 4.2: Create PipelineList component -- table of runs with status badges, filtering, search [P]
- [X] Task 4.3: Create PipelineDetail component -- run info, step list, event log [P]
- [X] Task 4.4: Create StepDetail component -- step progress, persona, duration, contract results [P]
- [X] Task 4.5: Create StatusBadge component -- color-coded status indicators [P]

## Phase 5: Frontend Data Layer

- [X] Task 5.1: Create `useApi` hook for REST API calls with error handling
- [X] Task 5.2: Create `useSSE` hook for real-time event stream with auto-reconnect
- [X] Task 5.3: Wire SSE events into component state for live progress updates

## Phase 6: Styling

- [X] Task 6.1: Create base CSS styles -- layout, typography, color scheme (light theme) [P]
- [X] Task 6.2: Create responsive layout for desktop and tablet viewports [P]

## Phase 7: Testing

- [X] Task 7.1: Write unit tests for `internal/dashboard/server.go` (startup, shutdown, config)
- [X] Task 7.2: Write unit tests for `internal/dashboard/handlers.go` (each endpoint with mock store)
- [X] Task 7.3: Write unit tests for `internal/dashboard/sse.go` (subscribe, unsubscribe, broadcast, heartbeat)
- [X] Task 7.4: Write unit tests for `cmd/wave/commands/serve.go` (flag parsing, validation)
- [X] Task 7.5: Build frontend and verify bundle size < 50 KB gzipped
- [X] Task 7.6: Run `go test ./...` to verify no regressions

## Phase 8: Polish

- [X] Task 8.1: Verify `go:embed` works correctly with pre-built assets
- [X] Task 8.2: Test end-to-end: `wave serve` + concurrent `wave run` shows live updates
- [X] Task 8.3: Verify existing CLI commands are unaffected
