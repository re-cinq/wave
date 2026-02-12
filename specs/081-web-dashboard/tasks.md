# Tasks

## Phase 1: Go HTTP Server Infrastructure

- [ ] Task 1.1: Create `internal/dashboard/types.go` with API request/response types
- [ ] Task 1.2: Create `internal/dashboard/server.go` with HTTP server setup, mux routing, middleware (CORS, content-type), graceful shutdown
- [ ] Task 1.3: Create `internal/dashboard/handlers.go` with REST API handlers (runs list, run detail, events, steps, artifacts)
- [ ] Task 1.4: Create `internal/dashboard/sse.go` with SSE event broker (client registration, fan-out broadcast, heartbeat)
- [ ] Task 1.5: Create `internal/dashboard/embed.go` with `go:embed` directives and static file serving

## Phase 2: CLI Command

- [ ] Task 2.1: Create `cmd/wave/commands/serve.go` with `wave serve` cobra command (--port, --bind flags)
- [ ] Task 2.2: Register serve command in `cmd/wave/main.go`

## Phase 3: Frontend Build Setup

- [ ] Task 3.1: Initialize `web/` directory with Bun, Preact, Vite, TypeScript configuration [P]
- [ ] Task 3.2: Configure Vite build to output to `internal/dashboard/static/` with compression [P]

## Phase 4: Frontend Components

- [ ] Task 4.1: Create root app component with routing (PipelineList / PipelineDetail views) [P]
- [ ] Task 4.2: Create PipelineList component -- table of runs with status badges, filtering, search [P]
- [ ] Task 4.3: Create PipelineDetail component -- run info, step list, event log [P]
- [ ] Task 4.4: Create StepDetail component -- step progress, persona, duration, contract results [P]
- [ ] Task 4.5: Create StatusBadge component -- color-coded status indicators [P]

## Phase 5: Frontend Data Layer

- [ ] Task 5.1: Create `useApi` hook for REST API calls with error handling
- [ ] Task 5.2: Create `useSSE` hook for real-time event stream with auto-reconnect
- [ ] Task 5.3: Wire SSE events into component state for live progress updates

## Phase 6: Styling

- [ ] Task 6.1: Create base CSS styles -- layout, typography, color scheme (light theme) [P]
- [ ] Task 6.2: Create responsive layout for desktop and tablet viewports [P]

## Phase 7: Testing

- [ ] Task 7.1: Write unit tests for `internal/dashboard/server.go` (startup, shutdown, config)
- [ ] Task 7.2: Write unit tests for `internal/dashboard/handlers.go` (each endpoint with mock store)
- [ ] Task 7.3: Write unit tests for `internal/dashboard/sse.go` (subscribe, unsubscribe, broadcast, heartbeat)
- [ ] Task 7.4: Write unit tests for `cmd/wave/commands/serve.go` (flag parsing, validation)
- [ ] Task 7.5: Build frontend and verify bundle size < 50 KB gzipped
- [ ] Task 7.6: Run `go test ./...` to verify no regressions

## Phase 8: Polish

- [ ] Task 8.1: Verify `go:embed` works correctly with pre-built assets
- [ ] Task 8.2: Test end-to-end: `wave serve` + concurrent `wave run` shows live updates
- [ ] Task 8.3: Verify existing CLI commands are unaffected
