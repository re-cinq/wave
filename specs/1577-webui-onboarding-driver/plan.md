# Implementation Plan — 1577 Webui Onboarding Driver

## 1. Objective

Add a chat-style `/onboard` web UI that drives the existing onboarder agent (`internal/onboarding`) through SSE-streamed conversation and form-answer POST round-trips, with reload survival via `Service.Resume`.

## 2. Approach

- New handler file `internal/webui/handlers_onboard.go` exposing three routes:
  - `GET /onboard` — render chat shell (or `/onboard/{sessionID}` to resume).
  - `GET /onboard/{sessionID}/stream` — SSE: emits conversation events, prompt requests, status, completion.
  - `POST /onboard/{sessionID}/answer` — form-encoded user reply; unblocks the agent's pending prompt.
- Implement an HTTP-bridging `onboarding.UI` that buffers conversation history and exposes channels/condition vars to:
  1. publish agent output as SSE events,
  2. block on `PromptString`/`PromptChoice` until the matching POST arrives.
- Maintain an in-memory `sessionRegistry` (mutex-guarded `map[string]*webOnboardSession`) on the `Server`, mirroring the `activeRuns` pattern.
- Wire the onboarder agent stdout into the bridge via the existing adapter `OnStreamEvent` callback (`internal/adapter/adapter.go:74`).
- Templates under `internal/webui/templates/onboard/` registered via `embed.go` page-template list.

## 3. File Mapping

### Created
- `internal/webui/handlers_onboard.go` — handlers, session registry type, HTTP `UI` bridge.
- `internal/webui/handlers_onboard_test.go` — unit + httptest coverage.
- `internal/webui/templates/onboard/index.html` — chat shell (extends `layout.html`).
- `internal/webui/templates/onboard/_message.html` — SSE-rendered message partial (server-side fragment).
- `internal/webui/templates/onboard/_prompt.html` — form widget for active prompt.

### Modified
- `internal/webui/routes.go` — register the three onboarding routes.
- `internal/webui/embed.go` — add the new templates to `pageTemplates`.
- `internal/webui/server.go` — add `onboardSessions` field + accessors on `Server`.
- `internal/webui/static/style.css` — minimal chat styling (chat bubbles, prompt form, stepper).
- `internal/onboarding/service.go` (only if needed) — surface a constructor that accepts a custom `UI` (verify; baseline likely already does).

### Deleted
None.

## 4. Architecture Decisions

1. **In-memory session registry** — matches existing `activeRuns` precedent (`server.go:78`). Durable persistence is out of scope per issue (1.2 phase, durable comes later).
2. **SSE over WebSocket** — repo already standardised on SSE (`internal/webui/sse.go`); reuse `SSEBroker` patterns + `Last-Event-ID` for backfill.
3. **Form-encoded POSTs** (not JSON) — simpler progressive-enhancement; matches "form-answer POST" wording in the issue and lets non-JS browsers work for the MVP.
4. **HTTP `UI` bridge implementing `onboarding.UI`** — keeps the onboarder Service contract intact; the webui is just another front-end alongside CLI/TUI.
5. **Per-session goroutine** — runs the onboarder Service synchronously; output funnelled through buffered channel to SSE subscribers. Cancelled on session close.
6. **Tailwind deferred** — issue allows fallback to minimal CSS; 1.5c site-wide not landed. Add scoped chat styles to `static/style.css` to avoid blocking on 1.5c.
7. **`sessionID` generation** — random URL-safe string via `crypto/rand` (existing helper in `internal/webui/auth.go` if present; else inline).
8. **Stream events schema** — typed SSE event names: `message`, `prompt`, `status`, `done`, `error`. Each carries JSON payload for forward-compat. Documented in handler comment.
9. **Auth/middleware** — onboarding routes mounted under existing middleware chain. No special bypass; localhost dev is the primary smoke target.

## 5. Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Baseline `Service` is synchronous — long-running onboarder blocks request goroutine. | Run Service on a dedicated goroutine per session; HTTP handlers only read/write through the bridge channels. |
| `Resume(sessionID)` returns same in-memory session — if process restarts, sessions vanish. | Acceptable for 1.2; document limitation. Future durable store is a separate epic. |
| Concurrent prompt + answer races (double-submit). | Bridge tracks a single `pendingPrompt` per session under mutex; reject extra answers with 409. |
| SSE consumers reconnect mid-stream → miss events. | Keep last N (e.g. 200) events in ring buffer; replay on `Last-Event-ID` (existing pattern in `sse.go`). |
| Onboarder writes files in workspace — webui process needs write perms in target repo dir. | Run webui from the project root; document `cwd` requirement in handler doc comment. |
| Browser JS dependency. | SSE works without a JS framework; minimal vanilla JS for SSE → DOM append. No build step. |
| `templates/preview/onboard.html` already exists (1.5c phase A) — naming clash. | Place new templates under `templates/onboard/` (subdir) to avoid collision; preview stays untouched. |

## 6. Testing Strategy

### Unit (`handlers_onboard_test.go`)
- `GET /onboard` → 200, contains chat shell markup.
- `GET /onboard/{id}/stream` → headers `text/event-stream`, emits seeded message.
- `POST /onboard/{id}/answer` → 200, unblocks bridge's `PromptString`.
- 404 on unknown session ID.
- 409 on double answer.
- Bridge: `PromptString` blocks until answer arrives, returns received string; cancelled context releases.

### Integration (`internal/webui/handlers_onboard_integration_test.go`)
- Spin a `httptest.Server`, drive a fake onboarder Service that calls `PromptString` then exits, assert the SSE → POST round-trip completes and Service returns.

### Manual smoke
- Throwaway repo → `wave webui` → browser `/onboard` → step through prompts → confirm `wave.yaml` written.

### CI
- `go test ./internal/webui/... -race` must pass.
- `go vet ./...` + `golangci-lint run` clean.

## 7. Open Questions (deferred to checklist/analyze)

- Exact SSE event JSON schema (proposed in §4.8 — confirm during implementation).
- Whether to render messages server-side (HTML fragments) or client-side (JSON → DOM). Default: server-rendered fragments for simplicity.
- Persona-vs-agent invocation path — verify whether the webui calls the onboarder Service directly or kicks off the `onboard-project` meta-pipeline. Default: Service direct (lighter), pipeline kick-off can come later.
