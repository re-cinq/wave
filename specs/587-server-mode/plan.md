# Implementation Plan: Server Mode with REST API and SSE Event Streaming

## Objective

Upgrade Wave's existing `wave serve` web dashboard into a production-grade server mode with FIFO run scheduling, JWT/mTLS authentication, HTTP gate resolution, TLS support, and manifest-driven configuration.

## Approach

This is an **incremental enhancement** of the existing `internal/webui/` package, not a greenfield build. The strategy is:

1. **Add run scheduler** — extract immediate goroutine launch into a FIFO queue with concurrency control
2. **Add new API endpoints** — `POST /api/runs`, `GET /api/runs/:id/logs`, `POST /api/runs/:id/gate/:step`
3. **Replace auth** — swap bearer token middleware with a pluggable auth system (JWT, mTLS, none)
4. **Add TLS** — configure TLS termination in `Server.Start()`
5. **Add manifest config** — `server:` section in manifest types, merge with CLI flags

Each layer is independently testable and builds on the previous one.

## File Mapping

### New Files

| Path | Purpose |
|------|---------|
| `internal/webui/scheduler.go` | FIFO run queue with configurable concurrency limit |
| `internal/webui/scheduler_test.go` | Scheduler unit tests |
| `internal/webui/handlers_gate.go` | HTTP gate resolution handler |
| `internal/webui/handlers_gate_test.go` | Gate handler tests |
| `internal/webui/jwt.go` | JWT token generation and validation |
| `internal/webui/jwt_test.go` | JWT tests |

### Modified Files

| Path | Changes |
|------|---------|
| `internal/webui/server.go` | Add scheduler field, TLS config, mTLS setup, `ServerConfig` expansion |
| `internal/webui/routes.go` | Register new endpoints: `POST /api/runs`, `GET /api/runs/:id/logs`, `POST /api/runs/:id/gate/:step` |
| `internal/webui/middleware.go` | Refactor `authMiddleware` to support JWT/mTLS/none modes |
| `internal/webui/auth.go` | Add JWT validation, mTLS client cert validation, auth mode types |
| `internal/webui/handlers_control.go` | Refactor `handleStartPipeline` to use scheduler, add `handleSubmitRun`, `handleRunLogs` |
| `internal/webui/types.go` | Add request/response types for new endpoints |
| `internal/manifest/types.go` | Add `Server` config struct to `Manifest` |
| `cmd/wave/commands/serve.go` | Add `--max-concurrent`, `--tls-cert`, `--tls-key`, `--auth-mode` flags; merge with manifest config |
| `internal/pipeline/gate.go` | Add channel-based resolution for external gate triggers |

## Architecture Decisions

### 1. Scheduler lives in `internal/webui/`

The run scheduler is HTTP-server-specific (queuing API-submitted runs). It does not belong in `internal/pipeline/` which handles step-level execution. The scheduler wraps `launchPipelineExecution` calls.

### 2. Channel-based gate resolution

The `GateExecutor.executeApproval()` currently blocks on context cancellation or timeout. To support HTTP resolution, add a `gateChannels map[string]chan struct{}` on the server, keyed by `{runID}:{stepID}`. The gate executor receives a channel, and the HTTP handler closes it to unblock the gate.

### 3. JWT over session tokens

JWT is stateless and fits Wave's architecture (no session store needed). The server validates `HS256`-signed tokens using a configurable secret. Token issuance is out of scope for this PR — users generate tokens externally or via a future `wave token create` command.

### 4. Auth mode enum, not boolean

Replace the current `requiresAuth()` bool check with an `AuthMode` enum (`none`, `bearer`, `jwt`, `mtls`). The `bearer` mode preserves backward compatibility with the existing token auth. Default for localhost is `none`; default for non-localhost is `bearer` (preserving existing behavior).

### 5. Manifest config merges with CLI flags

CLI flags take precedence over manifest `server:` config. This follows the standard pattern: env vars < manifest < CLI flags.

### 6. TLS is additive, not required

TLS is disabled by default. When enabled, `Server.Start()` calls `httpServer.ServeTLS()` instead of `httpServer.Serve()`. mTLS is an extension of TLS that additionally requires `tls.RequireAndVerifyClientCert`.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Gate resolution race condition | Gate resolves twice or misses signal | Use `sync.Once` for channel close; check gate state before resolution |
| JWT secret management | Secret leak or missing config | Require `WAVE_JWT_SECRET` env var; fail fast if JWT mode without secret |
| Scheduler deadlock | Queued runs never execute | Bounded channel with timeout; always release slot in `defer`; test concurrent drain |
| mTLS certificate complexity | Users struggle with cert setup | Good error messages; `auth.mode: none` as easy fallback; skip mTLS in first iteration if needed |
| Breaking existing auth | Existing `--token` users lose access | `bearer` mode as backward-compatible default |

## Testing Strategy

### Unit Tests
- **Scheduler**: concurrent enqueue/dequeue, max capacity, cancellation, shutdown drain
- **JWT**: token generation, validation, expiry, invalid signature, missing claims
- **Gate handler**: resolve waiting gate, resolve non-existent gate (404), double-resolve (idempotent)
- **Auth middleware**: each mode (none/bearer/jwt/mtls) with valid/invalid credentials
- **Manifest parsing**: server section with all fields, partial fields, defaults

### Integration Tests
- **End-to-end server**: start server, submit run via `POST /api/runs`, observe via SSE, verify scheduling
- **Gate flow**: start pipeline with gate step, resolve via HTTP, verify pipeline continues

### Existing Tests
- `go test -race ./...` must pass — verify all existing webui tests still pass with refactored auth
