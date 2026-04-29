# Work Items — 1577 Webui Onboarding Driver

## Phase 1: Setup
- [X] Item 1.1: Verify `internal/onboarding.Service` constructor accepts a custom `UI` — confirmed `StartOptions.UI` already plumbed through `BaselineService`. No service.go change required.
- [X] Item 1.2: Add `onboardSessions` map + mutex to `Server` in `internal/webui/server.go` (mirror `activeRuns`). New `serverOnboard` group with sessions + factory hook.
- [X] Item 1.3: Sketch SSE event schema (`message`, `prompt`, `status`, `done`, `error`) in a top-of-file doc comment for `handlers_onboard.go`.

## Phase 2: Core Implementation
- [X] Item 2.1: Implement HTTP `UI` bridge type (`webOnboardUI`) in `handlers_onboard.go` — buffered channels, `pendingPrompt` slot under mutex, ring-buffer of past events.
- [X] Item 2.2: Implement session registry helpers — `createOnboardSession`, `getOnboardSession`, `closeOnboardSession`.
- [X] Item 2.3: Implement `handleOnboardPage` (`GET /onboard` + `GET /onboard/{id}`) — render chat shell, create-or-resume session.
- [X] Item 2.4: Implement `handleOnboardStream` (`GET /onboard/{id}/stream`) — SSE loop, `Last-Event-ID` backfill from ring buffer.
- [X] Item 2.5: Implement `handleOnboardAnswer` (`POST /onboard/{id}/answer`) — validate pending prompt, deliver answer.
- [X] Item 2.6: Wire onboarder Service goroutine launched on session create; route `Notify` callbacks into the bridge (Service stdout currently wired through Notify; native `OnStreamEvent` adapter wiring deferred until 1.3 lands a streaming-capable Service).
- [X] Item 2.7: Register routes in `internal/webui/routes.go`.
- [X] Item 2.8: Author templates `templates/onboard/index.html`, `_message.html`, `_prompt.html`.
- [X] Item 2.9: Register templates in `internal/webui/embed.go` (page + onboard partials walker).
- [X] Item 2.10: Add minimal chat CSS to `internal/webui/static/style.css`.

## Phase 3: Testing
- [X] Item 3.1: Unit tests for `handleOnboardPage` (200 + body assertions).
- [X] Item 3.2: Unit tests for `handleOnboardStream` (SSE headers, replay via `Last-Event-ID`).
- [X] Item 3.3: Unit tests for `handleOnboardAnswer` (success, 404, 409 double answer, 400 prompt-id mismatch).
- [X] Item 3.4: Bridge unit tests — `PromptString` blocking + ctx cancel.
- [X] Item 3.5: Integration test `internal/webui/handlers_onboard_integration_test.go` — full SSE → POST round trip with fake Service.
- [X] Item 3.6: `go test ./... -race` clean.

## Phase 4: Polish
- [ ] Item 4.1: Manual browser smoke on a throwaway repo — verify `wave.yaml` materialises after flow. Capture trace in PR description. **Deferred to PR review** — test infrastructure built; manual smoke happens at merge gate.
- [X] Item 4.2: `golangci-lint run` + `go vet ./...` clean.
- [X] Item 4.3: Update `docs/guides/web-dashboard.md` with `/onboard` paragraph.
- [ ] Item 4.4: PR description: link issue #1577, list acceptance-criteria checkmarks, attach smoke evidence. **Done at PR open time.**
