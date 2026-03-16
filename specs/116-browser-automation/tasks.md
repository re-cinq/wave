# Tasks: Browser Automation Capability

**Feature**: #116 — Browser Automation for Personas
**Branch**: `116-browser-automation`
**Generated**: 2026-03-16
**Source**: spec.md, plan.md, data-model.md, research.md

## Phase 1: Setup

- [X] T001 [P1] Add `github.com/chromedp/chromedp` dependency via `go get github.com/chromedp/chromedp` and verify `go.mod` / `go.sum` updated

## Phase 2: Foundational — Core Types & Adapter Registration (Prerequisites for all stories)

- [X] T002 [P1] [P] Define `BrowserCommand` struct with fields: Action, URL, Selector, Value, Format, TimeoutMs, WaitFor in `internal/adapter/browser.go`
- [X] T003 [P1] [P] Define `BrowserResult` struct with fields: Status, Data, Error, DurationMs in `internal/adapter/browser.go`
- [X] T004 [P1] [P] Define `BrowserConfig` struct with fields: Headless, ViewportWidth, ViewportHeight, MaxRedirects, MaxResponseSize, CommandTimeout in `internal/adapter/browser.go`
- [X] T005 [P1] Define `BrowserAdapter` struct implementing `AdapterRunner` interface with a stub `Run` method in `internal/adapter/browser.go`
- [X] T006 [P1] Add `NewBrowserAdapter()` constructor with sensible defaults (headless: true, viewport: 1280x720, max_redirects: 10, max_response_size: 5MB, command_timeout: 30s) in `internal/adapter/browser.go`
- [X] T007 [P1] Add `"browser"` case to `ResolveAdapter` switch in `internal/adapter/opencode.go:157` returning `NewBrowserAdapter()`
- [X] T008 [P1] Implement JSON command parsing in `BrowserAdapter.Run()`: parse `cfg.Prompt` as `[]BrowserCommand`, validate required fields per action type, return structured errors for invalid input in `internal/adapter/browser.go`
- [X] T009 [P1] Implement chromedp allocator setup in `BrowserAdapter.Run()`: create `chromedp.NewExecAllocator` with headless flag, viewport dimensions, user-data-dir in workspace temp, and sandbox-safe Chromium flags in `internal/adapter/browser.go`
- [X] T010 [P1] Implement command dispatch loop: iterate `[]BrowserCommand`, route to per-action handler, collect `[]BrowserResult`, marshal to JSON stdout in `internal/adapter/browser.go`
- [X] T011 [P1] Implement `AdapterResult` construction: wrap JSON stdout, set exit code (0 on success, 1 on any error), populate ResultContent in `internal/adapter/browser.go`

## Phase 3: Story 1 — Navigate and Screenshot (P1)

- [X] T012 [P1] [S1] Implement `executeNavigate` handler: use `chromedp.Navigate(url)`, extract page title via `chromedp.Title`, capture final URL and HTTP status code, return `BrowserResult` with data map in `internal/adapter/browser.go`
- [X] T013 [P1] [S1] Implement `executeScreenshot` handler: use `chromedp.CaptureScreenshot()`, base64-encode PNG, include width/height/format in result data map in `internal/adapter/browser.go`
- [X] T014 [P1] [S1] Implement per-command timeout enforcement: wrap each command execution in timer-based timeout (avoids chromedp context cancellation) using `BrowserCommand.TimeoutMs` (default 30s) in `internal/adapter/browser.go`
- [X] T015 [P1] [S1] Implement `wait_for` support: when `BrowserCommand.WaitFor` is set, execute `chromedp.WaitVisible(selector)` before the main action in `internal/adapter/browser.go`
- [X] T016 [P1] [S1] Write unit tests for navigate and screenshot command parsing and validation in `internal/adapter/browser_test.go`
- [X] T017 [P1] [S1] Write integration test: start `httptest.NewServer` serving a simple HTML page, navigate to it, take screenshot, verify result contains base64 PNG data in `internal/adapter/browser_test.go`

## Phase 4: Story 4 — Sandbox-Compliant Browser Execution (P1)

- [X] T018 [P1] [S4] Implement domain allowlist enforcement via CDP `fetch.Enable`: intercept all network requests, check URL host against `cfg.AllowedDomains`, fail non-matching requests with `fetch.FailRequest(ErrorReasonBlockedByClient)` in `internal/adapter/browser.go`
- [X] T019 [P1] [S4] Implement browser process lifecycle cleanup: use `defer` to cancel chromedp context ensuring browser process termination, verify no orphan processes in `internal/adapter/browser.go`
- [X] T020 [P1] [S4] Implement sandbox flag projection: when `cfg.SandboxEnabled`, add `--disable-extensions`, `--disable-plugins`, `--disable-popup-blocking` to chromedp allocator options in `internal/adapter/browser.go`
- [X] T021 [P1] [S4] Implement max redirect enforcement: track redirect count during navigation, abort with error when exceeding `BrowserConfig.MaxRedirects` in `internal/adapter/browser.go`
- [X] T022 [P1] [S4] Write unit test for domain filtering logic: verify allowed domains pass, blocked domains return error, wildcard patterns match in `internal/adapter/browser_test.go`
- [X] T023 [P1] [S4] Write integration test: start httptest server, configure adapter with `AllowedDomains: ["localhost"]`, verify navigation to localhost succeeds and external domain fails in `internal/adapter/browser_test.go`

## Phase 5: Story 2 — Extract Page Content (P2)

- [X] T024 [P2] [S2] Implement `executeGetText` handler: use `chromedp.Text(selector)` when selector provided, otherwise extract `document.body.innerText` via `chromedp.Evaluate`, return text as string in result data in `internal/adapter/browser.go`
- [X] T025 [P2] [S2] Implement `executeGetHTML` handler: when selector provided use `chromedp.OuterHTML(selector)`, otherwise evaluate `document.documentElement.outerHTML`, return HTML string in result data in `internal/adapter/browser.go`
- [X] T026 [P2] [S2] Implement response size limiting: check text/HTML length against `BrowserConfig.MaxResponseSize`, truncate with warning when exceeded in `internal/adapter/browser.go`
- [X] T027 [P2] [S2] Write unit tests for get_text and get_html command validation (with/without selector, size limiting) in `internal/adapter/browser_test.go`
- [X] T028 [P2] [S2] Write integration test: navigate to httptest page with known content, extract text and HTML, verify expected content matches in `internal/adapter/browser_test.go`

## Phase 6: Story 3 — Form Interaction (P3)

- [X] T029 [P3] [S3] Implement `executeClick` handler: use `chromedp.Click(selector)`, handle element-not-found errors with structured error including the failed selector in `internal/adapter/browser.go`
- [X] T030 [P3] [S3] Implement `executeType` handler: use `chromedp.SendKeys(selector, value)`, handle element-not-found errors with structured error in `internal/adapter/browser.go`
- [X] T031 [P3] [S3] Write unit tests for click and type command validation (missing selector, missing value for type) in `internal/adapter/browser_test.go`
- [X] T032 [P3] [S3] Write integration test: navigate to httptest page with form, type into input, click submit button, verify resulting page state in `internal/adapter/browser_test.go`

## Phase 7: Preflight & Manifest Validation

- [X] T033 [P1] [P] Extend `preflight.Checker` with `CheckBrowserBinary()` method: search for chromium/chromium-browser/google-chrome/google-chrome-stable on PATH, return actionable platform-specific install instructions on failure in `internal/preflight/preflight.go`
- [X] T034 [P1] [P] Wire browser preflight check into pipeline preflight: `CheckBrowserBinary()` available for pipelines declaring browser tool dependency in `internal/preflight/preflight.go`
- [X] T035 [P1] Add manifest validation: existing `validatePersonasListWithFile` already verifies adapter-persona matching — browser adapter follows same pattern
- [X] T036 [P1] Write preflight tests: mock PATH with/without browser binary, verify detection and error messages in `internal/preflight/preflight_test.go`
- [X] T037 [P1] Write manifest validation test: adapter-persona mismatch already covered by existing tests in `internal/manifest/parser_test.go`

## Phase 8: Observability & Event Integration

- [X] T038 [P2] Implement structured event emission: emit `stream_activity` events with `tool_name: "browser.<action>"` and `tool_target: "<url-or-selector>"` for each command execution in `internal/adapter/browser.go`
- [X] T039 [P2] Implement error event emission: emit structured error events for command failures, timeouts, and domain blocks in `internal/adapter/browser.go`
- [X] T040 [P2] Write unit tests for event emission: verify correct event fields for navigate, screenshot, and error scenarios in `internal/adapter/browser_test.go`

## Phase 9: Polish & Cross-Cutting Concerns

- [X] T041 [P1] Implement viewport size limits enforcement: cap maximum viewport dimensions to prevent oversized screenshots (e.g., max 3840x2160) in `internal/adapter/browser.go`
- [X] T042 [P1] Implement browser crash recovery: detect when chromedp context errors indicate a crashed browser, return structured error and ensure process cleanup in `internal/adapter/browser.go`
- [X] T043 [P1] Run `go test -race ./internal/adapter/...` and `go test -race ./internal/preflight/...` — fix any race conditions
- [X] T044 [P1] Run `go vet ./internal/adapter/... ./internal/preflight/...` — fix any lint findings
- [X] T045 [P1] Verify all success criteria: SC-001 (screenshot <10s), SC-002 (domain blocking), SC-003 (no state persistence), SC-004 (preflight detection), SC-005 (structured events), SC-006 (crash recovery), SC-007 (end-to-end integration)

## Dependency Graph

```
T001 → T002-T004 (parallel) → T005-T006 → T007 → T008-T011 (sequential)
T011 → T012-T015 (Phase 3, sequential)
T011 → T018-T021 (Phase 4, can run parallel with Phase 3)
T015 → T024-T026 (Phase 5, after Phase 3)
T026 → T029-T030 (Phase 6, after Phase 5)
T007 → T033-T035 (Phase 7, parallel with Phases 3-6)
T010 → T038-T039 (Phase 8, after command dispatch)
T032,T023,T037 → T041-T045 (Phase 9, after all features)
```
