# Research: Browser Automation Capability

**Feature**: #116 — Browser Automation for Personas
**Date**: 2026-03-16

## Decision 1: CDP Client Library — chromedp vs rod

**Decision**: Use `chromedp` (github.com/chromedp/chromedp)

**Rationale**:
- chromedp is the most mature Go CDP library (6+ years, 10k+ GitHub stars)
- First-party maintained by the Chrome DevTools Protocol team contributors
- Provides high-level actions (Navigate, Click, SendKeys, Screenshot, etc.) that map 1:1 to our BrowserCommand types
- Built-in support for headless mode, viewport configuration, and timeouts
- Context-based lifecycle management aligns with Wave's `context.Context` pattern
- chromedp.NewExecAllocator allows passing Chromium flags for sandbox control
- Network interception via `fetch.Enable` / `network.Enable` for domain allowlist enforcement

**Alternatives Rejected**:
- **rod** (github.com/go-rod/rod): Simpler API but auto-downloads browser binaries by default (violates Principle 1: single binary, no runtime dependencies). While configurable, rod's download-first philosophy creates friction. Less mature CDP protocol coverage for network interception.
- **playwright-go**: Wraps Node.js Playwright — violates the "no Node.js" constraint (FR-016).
- **selenium/webdriver**: Requires a separate WebDriver binary (geckodriver/chromedriver) — adds a runtime dependency beyond just having Chrome/Chromium installed.

## Decision 2: Command Protocol Design

**Decision**: JSON array in `AdapterRunConfig.Prompt` field

**Rationale**:
- The `AdapterRunner.Run` interface accepts `AdapterRunConfig` where `Prompt` is a free-text string
- For the browser adapter, `Prompt` carries a JSON array of `BrowserCommand` objects
- The adapter parses JSON, executes commands sequentially, returns JSON array of `BrowserResult` objects via stdout
- This avoids modifying the `AdapterRunner` interface (which would touch all adapters)
- The executor constructs the JSON array from step configuration

**Protocol**:
```json
// Input (Prompt field)
[
  {"action": "navigate", "url": "http://localhost:3000", "timeout_ms": 30000},
  {"action": "screenshot", "format": "png"},
  {"action": "get_text", "selector": "#content"}
]

// Output (stdout)
[
  {"status": "success", "data": {"title": "My App", "url": "http://localhost:3000", "status_code": 200}, "duration_ms": 1200},
  {"status": "success", "data": {"image": "base64...", "format": "png", "width": 1280, "height": 720}, "duration_ms": 350},
  {"status": "success", "data": {"text": "Hello World"}, "duration_ms": 15}
]
```

## Decision 3: Domain Allowlist Enforcement

**Decision**: Network-level interception via CDP `fetch.Enable`

**Rationale**:
- Edge Case EC-005 requires blocking sub-resource requests, not just navigation
- chromedp supports `fetch.Enable` which intercepts ALL network requests before they leave the browser
- Each request is checked against the persona's `allowed_domains` list
- Non-matching requests are failed with `network.ErrorReasonBlockedByClient`
- This is more secure than just checking navigation URLs, which wouldn't catch XHR/fetch/image/script loads

## Decision 4: Browser Process Lifecycle

**Decision**: One browser process per step, killed on step completion

**Rationale**:
- Constitution Principle 4 (Fresh Memory) and FR-006 require no state persistence across step boundaries
- chromedp.NewExecAllocator creates a fresh browser process
- On step completion (success or failure), `chromedp.Cancel` terminates the browser
- The adapter's `Run` method uses `defer` to ensure cleanup even on panics
- A watchdog goroutine monitors for orphan processes using the browser PID

## Decision 5: Chromium Detection in Preflight

**Decision**: Extend `preflight.Checker` with browser-specific checks

**Rationale**:
- FR-009 requires preflight detection of browser binary
- The existing `Checker.CheckTools` method already does `exec.LookPath` for tools
- For browser steps, the preflight system checks for `chromium`, `chromium-browser`, `google-chrome`, or `google-chrome-stable` on PATH
- The preflight error message includes platform-specific installation instructions
- This follows the established pattern — no new mechanism needed

## Decision 6: Sandbox Compliance

**Decision**: Use chromedp.ExecAllocator flags to mirror Wave's sandbox

**Rationale**:
- FR-013 requires browser process isolation matching `runtime.sandbox` config
- chromedp supports `--no-sandbox` (ironically needed inside containers), `--disable-gpu`, `--disable-dev-shm-usage`
- When Wave sandbox is enabled, the browser launches with: `--disable-extensions`, `--disable-plugins`, `--disable-popup-blocking`, `--disable-translate`
- User data dir is set to a temp directory within the workspace, ensuring no persistent state
- Network domain enforcement via CDP `fetch.Enable` (Decision 3) provides the network-level sandbox

## Decision 7: Integration with Existing Adapter Pattern

**Decision**: Add `"browser"` case to `ResolveAdapter` switch in `internal/adapter/opencode.go`

**Rationale**:
- `ResolveAdapter` is the central adapter factory (claude, opencode, default)
- Adding `case "browser": return NewBrowserAdapter()` follows the exact same pattern
- The `BrowserAdapter` struct implements `AdapterRunner` interface
- No changes to the interface or executor needed — the browser adapter receives commands via `cfg.Prompt` like all other adapters
- Manifest declares `browser` adapter in `adapters:` section; personas reference it via `adapter: browser`

## Open Questions (None)

All technical decisions from the spec's [NEEDS CLARIFICATION] markers were resolved during the clarification phase:
- C1: Go-native CDP client → chromedp (this research confirms)
- C2: `get_html` defaults to full page → confirmed
- C3: Use existing `adapter` field → confirmed (no new `capabilities` map)
- C4: JSON protocol via Prompt field → confirmed
- C5: System dependency with preflight check → confirmed
