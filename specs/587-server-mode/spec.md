# feat: server mode with REST API and SSE event streaming

**Issue**: [re-cinq/wave#587](https://github.com/re-cinq/wave/issues/587)
**Labels**: enhancement
**Author**: nextlevelshit
**State**: OPEN

## Context

Wave currently operates as a CLI tool with an existing `wave serve` command that provides a web dashboard with REST API and SSE streaming. This issue upgrades server mode to a production-grade service with:

- **Run scheduling** via FIFO queue with configurable concurrency limit
- **JWT authentication** (replacing simple bearer token)
- **Optional mTLS** for machine-to-machine access
- **HTTP gate resolution** endpoint for interactive pipeline gates
- **Manifest-driven server configuration** via `server:` section in `wave.yaml`
- **TLS support** for encrypted transport

## Current State (Already Implemented)

The existing `internal/webui/` package provides:

| Feature | Status | Location |
|---------|--------|----------|
| `wave serve` CLI command | Exists | `cmd/wave/commands/serve.go` |
| REST API (runs, pipelines, personas, contracts, skills) | Exists | `internal/webui/routes.go` |
| SSE event streaming (`/api/runs/{id}/events`) | Exists | `internal/webui/sse.go`, `sse_broker.go` |
| Pipeline start/cancel/retry/resume | Exists | `internal/webui/handlers_control.go` |
| Bearer token auth | Exists | `internal/webui/middleware.go`, `auth.go` |
| Security headers | Exists | `internal/webui/auth.go` |
| Web dashboard with templates | Exists | `internal/webui/templates/`, `static/` |
| Health endpoint | Exists | `internal/webui/handlers_health.go` |

## Gaps to Fill

### 1. Unified Run Submission (`POST /api/runs`)

Currently, runs are submitted via `POST /api/pipelines/{name}/start`. The issue specifies `POST /api/runs` with pipeline name in the request body, which is a more RESTful pattern.

### 2. Run Log Endpoint (`GET /api/runs/:id/logs`)

A dedicated endpoint returning plain-text or structured logs for a run, distinct from the SSE event stream and the paginated step-events endpoint.

### 3. HTTP Gate Resolution (`POST /api/runs/:id/gate/:step`)

Currently, approval gates either auto-approve or wait for context cancellation/timeout. No mechanism exists for external HTTP clients to resolve a gate. The gate executor in `internal/pipeline/gate.go` needs a channel-based resolution mechanism that the HTTP handler can trigger.

### 4. Run Queue with Concurrency Scheduling

Currently `launchPipelineExecution` starts runs immediately in goroutines. Need a FIFO queue with configurable `--max-concurrent` limit (default: 5) that queues excess runs and dequeues them as slots free up.

### 5. JWT Authentication

Replace simple bearer token with JWT-based auth:
- Token signing with configurable secret (`WAVE_JWT_SECRET`)
- Token validation middleware
- Demo mode (`auth.mode: none`) for local development

### 6. TLS and mTLS Support

- TLS termination with cert/key configuration
- Optional mutual TLS for machine-to-machine authentication
- Configurable via manifest `server.tls` section

### 7. Server Configuration in Manifest

Add `server:` section to manifest types:

```yaml
server:
  bind: "127.0.0.1:8080"
  max_concurrent: 5
  auth:
    mode: jwt              # jwt | mtls | none
    jwt_secret: "${WAVE_JWT_SECRET}"
  tls:
    enabled: false
    cert: ""
    key: ""
```

## Acceptance Criteria

- [ ] `wave serve --max-concurrent N` limits concurrent pipeline runs via FIFO queue
- [ ] `POST /api/runs` creates and queues a new pipeline run
- [ ] `GET /api/runs/:id/logs` returns run logs
- [ ] `POST /api/runs/:id/gate/:step` resolves a waiting gate
- [ ] JWT auth mode validates tokens when `server.auth.mode: jwt`
- [ ] mTLS mode validates client certificates when `server.auth.mode: mtls`
- [ ] `auth.mode: none` allows unauthenticated access
- [ ] TLS enabled via `server.tls.enabled: true` with cert/key paths
- [ ] Server configuration is loaded from manifest `server:` section
- [ ] CLI flags override manifest configuration
- [ ] Graceful shutdown cancels active runs and drains queue
- [ ] All existing endpoints continue to work unchanged
- [ ] All new code has unit tests
- [ ] `go test -race ./...` passes
