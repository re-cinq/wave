# feat: server mode with REST API and SSE event streaming

**Issue**: [re-cinq/wave#587](https://github.com/re-cinq/wave/issues/587)
**Labels**: enhancement
**Author**: nextlevelshit
**State**: OPEN

## Context

Wave currently operates as a CLI tool. Adding a server mode enables team workflows, webhook-triggered pipelines, and persistent monitoring. Fabro's `fabro serve` provides a reference architecture with HTTP API, SSE event streaming, concurrent scheduling, and JWT/mTLS auth.

## Current State Analysis

The majority of server mode infrastructure **already exists** in `internal/webui/`:

| Feature | Status | Location |
|---------|--------|----------|
| `wave serve` CLI command | Complete | `cmd/wave/commands/serve.go` |
| REST API (runs CRUD, pipelines, personas, etc.) | Complete | `internal/webui/routes.go` |
| `POST /api/runs` (submit run) | Complete | `internal/webui/handlers_control.go` |
| `GET /api/runs/:id` (run detail) | Complete | `internal/webui/handlers_runs.go` |
| `GET /api/runs/:id/logs` (run logs) | Complete | `internal/webui/handlers_control.go` |
| SSE event streaming with reconnection backfill | Complete | `internal/webui/handlers_sse.go`, `sse_broker.go` |
| Pipeline start/cancel/retry/resume | Complete | `internal/webui/handlers_control.go` |
| FIFO scheduler with concurrency limit | Complete | `internal/webui/scheduler.go` |
| Bearer token auth middleware | Complete | `internal/webui/middleware.go` |
| JWT auth middleware + validation | Complete | `internal/webui/jwt.go`, `middleware.go` |
| mTLS support | Complete | `internal/webui/server.go` (TLS listener config) |
| Auth mode enum (none/bearer/jwt/mtls) | Complete | `internal/webui/server.go` |
| GateRegistry + WebUIGateHandler | Complete | `internal/webui/gate_handler.go` |
| Gate approve HTTP endpoint | Complete | `internal/webui/handlers_control.go` |
| Manifest `server:` config | Complete | `internal/manifest/types.go` |
| Graceful shutdown with run cancellation | Complete | `internal/webui/server.go` |
| Security headers | Complete | `internal/webui/auth.go` |
| Health endpoint | Complete | `internal/webui/handlers_health.go` |
| Web dashboard (HTML templates) | Complete | `internal/webui/templates/` |

## Bugs and Gaps Found

### Bug 1: Gate tests don't compile (CRITICAL)

`internal/webui/handlers_gate_test.go` was written against an older `GateRegistry` API:
- Calls `g.Register(runID, stepID)` with 2 args — actual signature requires 3: `(runID, stepID string, gate *pipeline.GateConfig)`
- Calls `g.Resolve(runID, stepID)` returning `bool` — actual signature is `(runID string, decision *pipeline.GateDecision)` returning `error`
- References `srv.handleResolveGate` — method doesn't exist (actual is `handleGateApprove`)
- References `g.Cleanup(runID, stepID)` — actual is `g.Remove(runID)`

This **breaks the entire webui package** — `go test ./internal/webui/` fails at compilation.

### Bug 2: `gateRegistry` field never initialized (CRITICAL)

Server struct has two gate-related fields:
- `gates *GateRegistry` — initialized in `NewServer` as `gates: NewGateRegistry()`
- `gateRegistry *GateRegistry` — **never initialized** (nil)

The `handleGateApprove` handler uses `s.gateRegistry` (nil), not `s.gates` (initialized). This means gate approval always returns "gate registry not initialized" (503).

### Bug 3: Gate handler not wired into executor

`launchPipelineExecution` creates the executor without `pipeline.WithGateHandler(...)`. Even though `WebUIGateHandler` exists, it's never used — pipelines started from the server have no gate handler, so gate steps will hang or fail.

## Acceptance Criteria

1. All webui tests compile and pass (`go test -race ./internal/webui/...`)
2. Gate approval endpoint is functional end-to-end (gateRegistry initialized, handler wired to executor)
3. `go test -race ./...` passes across the entire project
4. All REST API endpoints from the issue are functional
