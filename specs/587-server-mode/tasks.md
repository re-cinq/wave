# Tasks

## Phase 1: Run Scheduler

- [ ] Task 1.1: Create `internal/webui/scheduler.go` — FIFO run queue with configurable concurrency limit. Struct with bounded channel semaphore, enqueue method that blocks when full, dequeue on slot release, shutdown drain with context cancellation.
- [ ] Task 1.2: Write `internal/webui/scheduler_test.go` — concurrent enqueue/dequeue, max capacity enforcement, cancellation propagation, graceful shutdown drain, slot release on run completion.
- [ ] Task 1.3: Integrate scheduler into `internal/webui/server.go` — add scheduler field to `Server`, initialize in `NewServer`, shutdown in `Start()`. Refactor `launchPipelineExecution` to submit through scheduler instead of direct goroutine.
- [ ] Task 1.4: Add `--max-concurrent` flag to `cmd/wave/commands/serve.go`, pass through `ServerConfig`.

## Phase 2: New API Endpoints

- [ ] Task 2.1: Add `POST /api/runs` handler in `internal/webui/handlers_control.go` — accept `{pipeline: "name", input: "..."}` body, validate pipeline exists, submit via scheduler, return run ID and status. [P]
- [ ] Task 2.2: Add `GET /api/runs/:id/logs` handler in `internal/webui/handlers_control.go` — query events from state store, format as structured log output with timestamps. [P]
- [ ] Task 2.3: Register new routes in `internal/webui/routes.go`.
- [ ] Task 2.4: Add request/response types for new endpoints in `internal/webui/types.go`.

## Phase 3: HTTP Gate Resolution

- [ ] Task 3.1: Add channel-based gate resolution to `internal/pipeline/gate.go` — add `ResolveCh chan struct{}` field or accept a channel parameter in `executeApproval`, select on it alongside timeout/context.
- [ ] Task 3.2: Add gate channel registry to `internal/webui/server.go` — `gateChannels map[string]chan struct{}` keyed by `runID:stepID`, with register/resolve/cleanup methods.
- [ ] Task 3.3: Create `internal/webui/handlers_gate.go` — `POST /api/runs/:id/gate/:step` handler that looks up gate channel and closes it, returning 200 on success or 404 if no waiting gate.
- [ ] Task 3.4: Write `internal/webui/handlers_gate_test.go` — resolve waiting gate, resolve non-existent gate, double-resolve idempotency.
- [ ] Task 3.5: Register gate route in `internal/webui/routes.go`.

## Phase 4: Manifest Server Configuration

- [ ] Task 4.1: Add `Server` config struct to `internal/manifest/types.go` — bind, max_concurrent, auth (mode, jwt_secret), tls (enabled, cert, key) fields.
- [ ] Task 4.2: Add `Server *ServerConfig` field to `Manifest` struct, update YAML tags.
- [ ] Task 4.3: Write manifest parsing tests for server section — full config, partial config, defaults, env var expansion for jwt_secret.
- [ ] Task 4.4: Update `cmd/wave/commands/serve.go` to merge manifest server config with CLI flags (CLI flags take precedence).

## Phase 5: JWT Authentication

- [ ] Task 5.1: Create `internal/webui/jwt.go` — JWT validation with HS256 signing, claims parsing (sub, exp, iat), `ValidateJWT(tokenString, secret)` function. [P]
- [ ] Task 5.2: Write `internal/webui/jwt_test.go` — valid token, expired token, invalid signature, missing claims, malformed token. [P]
- [ ] Task 5.3: Refactor `internal/webui/middleware.go` — replace `authMiddleware` with `authMiddlewareForMode(mode)` that dispatches to bearer/jwt/none handlers. Add `AuthMode` type with constants.
- [ ] Task 5.4: Update `internal/webui/auth.go` — add `AuthMode` enum (`none`, `bearer`, `jwt`, `mtls`), JWT validation call, auth mode resolution from config.
- [ ] Task 5.5: Update `internal/webui/server.go` — pass auth mode and JWT secret to middleware, update `ServerConfig` to include auth fields.

## Phase 6: TLS and mTLS

- [ ] Task 6.1: Add TLS configuration to `internal/webui/server.go` — when TLS enabled, configure `tls.Config` and call `httpServer.ServeTLS()` instead of `httpServer.Serve()`. [P]
- [ ] Task 6.2: Add mTLS support — when auth mode is `mtls`, set `tls.Config.ClientAuth = tls.RequireAndVerifyClientCert` and load CA cert pool. [P]
- [ ] Task 6.3: Add `--tls-cert`, `--tls-key`, `--auth-mode` flags to `cmd/wave/commands/serve.go`.
- [ ] Task 6.4: Write TLS/mTLS tests — server starts with TLS, rejects without client cert in mTLS mode.

## Phase 7: Graceful Shutdown Enhancement

- [ ] Task 7.1: Enhance `Server.Start()` shutdown — cancel all active runs via `activeRuns` map, drain scheduler queue, wait for in-flight runs with timeout.
- [ ] Task 7.2: Test graceful shutdown — verify active runs receive cancellation, queued runs are drained, server exits cleanly.

## Phase 8: Validation and Polish

- [ ] Task 8.1: Run `go test -race ./...` — fix any race conditions or test failures.
- [ ] Task 8.2: Run `golangci-lint run ./...` — fix lint issues.
- [ ] Task 8.3: Verify backward compatibility — existing `wave serve` with `--token` flag must work unchanged.
- [ ] Task 8.4: Verify all existing webui tests pass with refactored auth middleware.
