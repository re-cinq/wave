# Implementation Plan: Browser Automation Capability

**Branch**: `116-browser-automation` | **Date**: 2026-03-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/116-browser-automation/spec.md`

## Summary

Add a `browser` adapter to Wave that enables personas to automate Chromium via the Chrome DevTools Protocol (CDP). The adapter uses `chromedp` (Go-native CDP client) compiled into the Wave binary, requiring only a system-installed Chromium at runtime. Commands (navigate, screenshot, get_text, get_html, click, type) are passed as JSON via the existing `Prompt` field and results returned via stdout. Domain allowlist enforcement, sandbox compliance, and clean process lifecycle are enforced per Wave's security model.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `github.com/chromedp/chromedp` (new), existing Wave packages
**Storage**: N/A (stateless adapter — no persistence)
**Testing**: `go test -race ./...` (existing test infrastructure)
**Target Platform**: Linux (primary), macOS (secondary)
**Project Type**: Single Go binary (internal package addition)
**Performance Goals**: Screenshot of simple page in <10 seconds (SC-001)
**Constraints**: Single binary (no Node.js), system Chromium only, sandbox compliant
**Scale/Scope**: 1 new package (`internal/adapter/browser.go`), extensions to 3 existing packages

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | chromedp compiles into Go binary. Only requires system Chromium (same pattern as `claude` CLI). |
| P2: Manifest as SSOT | PASS | Browser adapter declared in `wave.yaml` adapters section. Persona references via `adapter: browser`. |
| P3: Persona-Scoped Boundaries | PASS | Domain allowlist from persona sandbox config enforced at network level. |
| P4: Fresh Memory | PASS | Browser process killed after each step. No state persistence. |
| P5: Navigator-First | N/A | Browser steps are execution steps, not navigation. |
| P6: Contracts at Handover | PASS | Browser adapter returns structured JSON; standard contract validation applies. |
| P7: Relay | N/A | Browser adapter doesn't use LLM context — no compaction needed. |
| P8: Ephemeral Workspaces | PASS | Browser user data dir inside workspace temp directory. |
| P9: Credentials Never Touch Disk | PASS | No credentials involved in browser automation. |
| P10: Observable Progress | PASS | All commands emit structured events via EventEmitter. |
| P11: Bounded Recursion | N/A | Browser adapter is a leaf execution — no recursion. |
| P12: Minimal State Machine | PASS | Uses standard step state machine (Pending → Running → Completed/Failed). |
| P13: Test Ownership | PASS | Full test suite required. Table-driven tests for all command types. |

## Project Structure

### Documentation (this feature)

```
specs/116-browser-automation/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: technology decisions
├── data-model.md        # Phase 1: entity definitions
└── tasks.md             # Phase 2 output (not created by plan)
```

### Source Code (repository root)

```
internal/adapter/
├── browser.go           # BrowserAdapter, BrowserCommand, BrowserResult, BrowserConfig
├── browser_test.go      # Unit tests (command parsing, domain filtering, config)
├── opencode.go          # ResolveAdapter extension (add "browser" case)
└── ...                  # Existing adapter files (unchanged)

internal/preflight/
├── preflight.go         # Extended with browser binary detection
└── preflight_test.go    # Tests for browser preflight checks

internal/manifest/
├── types.go             # No changes needed (Persona.Adapter already supports arbitrary strings)
└── validation.go        # Add browser adapter validation (adapter field matches)
```

**Structure Decision**: All new code lives in `internal/adapter/browser.go` (single file, ~400-500 lines). The existing adapter pattern is extended, not replaced. Preflight gains a browser-specific check method. No new packages needed.

## Implementation Phases

### Phase A: Core Types & Adapter Skeleton

1. Add `github.com/chromedp/chromedp` dependency to `go.mod`
2. Create `internal/adapter/browser.go` with:
   - `BrowserCommand` struct (action, url, selector, value, format, timeout_ms, wait_for)
   - `BrowserResult` struct (status, data, error, duration_ms)
   - `BrowserConfig` struct (headless, viewport, max_redirects, max_response_size, command_timeout)
   - `BrowserAdapter` struct implementing `AdapterRunner`
   - `NewBrowserAdapter()` constructor
3. Add `"browser"` case to `ResolveAdapter` in `internal/adapter/opencode.go`

### Phase B: Command Execution

1. Implement `BrowserAdapter.Run()`:
   - Parse `cfg.Prompt` as `[]BrowserCommand`
   - Create chromedp allocator with sandbox-aware flags
   - Set up domain filtering via CDP network interception
   - Execute commands sequentially with per-command timeouts
   - Return `[]BrowserResult` as JSON via stdout
2. Implement individual command handlers:
   - `executeNavigate` — open URL, return title/url/status_code
   - `executeScreenshot` — capture viewport as base64 PNG
   - `executeGetText` — extract visible text (optional selector)
   - `executeGetHTML` — extract HTML (full page or selector)
   - `executeClick` — click element by CSS selector
   - `executeType` — type text into input by CSS selector
3. Implement `wait_for` support (CSS selector wait before command)

### Phase C: Security & Sandbox

1. Domain allowlist enforcement via CDP `fetch.Enable`:
   - Intercept all network requests
   - Check request URL domain against `cfg.AllowedDomains`
   - Block non-matching requests with `fetch.FailRequest`
2. Browser process lifecycle:
   - Launch in workspace temp directory (no persistent state)
   - Kill process group on step completion via existing `killProcessGroup`
   - Watchdog goroutine to detect orphan processes
3. Sandbox flag projection:
   - When `cfg.SandboxEnabled`, add `--disable-extensions`, `--disable-plugins` to Chromium flags
   - Set `--user-data-dir` to workspace-scoped temp directory

### Phase D: Preflight & Validation

1. Extend `preflight.Checker` with `CheckBrowserBinary()`:
   - Search for chromium/chrome binaries on PATH
   - Return actionable install instructions per platform
2. Add manifest validation for browser adapter:
   - Verify persona with `adapter: browser` references a declared browser adapter
   - Validate browser adapter `binary` field matches available browsers

### Phase E: Testing

1. Unit tests (`internal/adapter/browser_test.go`):
   - Command parsing (valid/invalid JSON)
   - Domain filtering logic
   - Config defaults
   - Sandbox flag generation
2. Integration test (requires Chromium installed):
   - Navigate to `httptest.NewServer`, extract text, take screenshot
   - Domain blocking verification
   - Timeout and error handling
3. Preflight tests (`internal/preflight/preflight_test.go`):
   - Browser binary detection
   - Missing binary error messages

## Complexity Tracking

_No constitution violations — no entries needed._
