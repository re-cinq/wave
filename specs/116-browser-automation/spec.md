# Feature Specification: Browser Automation Capability for Personas

**Feature Branch**: `116-browser-automation`
**Created**: 2026-03-16
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/116

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Navigate and Screenshot (Priority: P1)

As a pipeline author, I want a persona to navigate to a URL and capture a screenshot, so that I can visually verify application state as part of an automated pipeline.

**Why this priority**: This is the foundational browser automation action — all other interactions build on the ability to open a page and capture its rendered output.

**Independent Test**: Can be fully tested by configuring a step with the `browser` adapter, navigating to a known URL, and verifying a screenshot artifact is produced.

**Acceptance Scenarios**:

1. **Given** a pipeline step configured with the `browser` adapter and a `navigate` command targeting `http://localhost:3000`, **When** the step executes, **Then** the browser opens the URL and returns a success status with page metadata (title, URL, status code).
2. **Given** a pipeline step that has navigated to a URL, **When** a `screenshot` command is issued, **Then** a base64-encoded PNG image is returned as a structured artifact.
3. **Given** a pipeline step targeting a URL that returns HTTP 500, **When** the `navigate` command executes, **Then** the adapter reports the HTTP status code and still captures the rendered page state.

---

### User Story 2 - Extract Page Content (Priority: P2)

As a persona in a QA pipeline, I want to extract rendered text and DOM content from a web page, so that I can analyze application output without relying on screenshots alone.

**Why this priority**: Text extraction enables programmatic analysis by downstream personas, making it more actionable than screenshots alone.

**Independent Test**: Can be tested by navigating to a page with known content and verifying the extracted text matches expected output.

**Acceptance Scenarios**:

1. **Given** a browser has navigated to a page, **When** a `get_text` command is issued, **Then** the visible text content of the page is returned as a string artifact.
2. **Given** a browser has navigated to a page, **When** a `get_html` command is issued with a CSS selector, **Then** the outer HTML of matching elements is returned.
3. **Given** a page with dynamically loaded content, **When** `get_text` is issued after a configurable wait condition, **Then** the dynamically rendered content is included in the result.

---

### User Story 3 - Form Interaction (Priority: P3)

As a QA persona, I want to click elements and type into form fields, so that I can perform end-to-end interaction testing within a pipeline.

**Why this priority**: Interactive testing is a high-value use case but depends on navigation and content extraction working first.

**Independent Test**: Can be tested by navigating to a form page, filling fields, submitting, and verifying the resulting page state.

**Acceptance Scenarios**:

1. **Given** a browser on a page with an input field, **When** a `type` command is issued with a CSS selector and text value, **Then** the text is entered into the matching input element.
2. **Given** a browser on a page with a button, **When** a `click` command is issued with a CSS selector, **Then** the element is clicked and any resulting navigation or DOM change occurs.
3. **Given** a `click` command targeting a non-existent selector, **When** the command executes, **Then** the adapter returns a structured error with the selector that failed to match.

---

### User Story 4 - Sandbox-Compliant Browser Execution (Priority: P1)

As a Wave operator, I want browser automation to respect Wave's sandbox and permission model, so that personas cannot access unauthorized URLs or leak data across step boundaries.

**Why this priority**: Security compliance is a hard requirement — browser automation without sandbox enforcement would violate Wave's security model.

**Independent Test**: Can be tested by configuring a persona with a restricted domain allowlist and verifying that navigation to non-allowed domains is blocked.

**Acceptance Scenarios**:

1. **Given** a persona with `allowed_domains: ["localhost"]`, **When** the browser adapter attempts to navigate to `https://external-site.com`, **Then** the navigation is blocked and an error is returned.
2. **Given** a browser step completes, **When** the next step begins, **Then** no browser state (cookies, localStorage, session data) persists from the previous step.
3. **Given** a step that references the browser adapter but the persona's `adapter` field is not `"browser"`, **When** manifest validation runs, **Then** validation fails with a clear error indicating adapter mismatch.

---

### Edge Cases

- What happens when the browser process crashes or hangs mid-command? The adapter MUST enforce a per-command timeout and terminate the browser process, returning an error artifact.
- What happens when a page triggers infinite redirects? The adapter MUST enforce a maximum redirect count (configurable, default 10) and abort with an error.
- What happens when a screenshot is requested on a page with a very large viewport? The adapter MUST enforce maximum viewport dimensions and image size limits.
- What happens when the browser binary is not installed? The preflight system MUST detect the missing dependency and report it before pipeline execution begins.
- What happens when the page loads content from domains not in the allowlist? Network requests to non-allowed domains MUST be blocked at the browser level, not just at navigation.
- What happens when a `get_text` command returns megabytes of text? The adapter MUST enforce a maximum response size and truncate with a warning.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST provide a `browser` adapter that implements the `AdapterRunner` interface and can be resolved by name via `ResolveAdapter`.
- **FR-002**: The browser adapter MUST support the following commands: `navigate`, `screenshot`, `get_text`, `click`, `type`.
- **FR-003**: The browser adapter MUST return structured results (command status, output data, error details) that downstream personas can parse.
- **FR-004**: Screenshots MUST be returned as base64-encoded PNG images within a structured artifact envelope.
- **FR-005**: The browser adapter MUST enforce the persona's `allowed_domains` list, blocking both navigation and sub-resource requests to non-allowed domains.
- **FR-006**: The browser adapter MUST terminate the browser process after each step completes — no browser state may persist across step boundaries.
- **FR-007**: The browser adapter MUST enforce a configurable per-command timeout (default: 30 seconds) and a per-step timeout (inherited from the step configuration).
- **FR-008**: Pipeline steps using the browser adapter MUST reference a persona whose `adapter` field is set to `"browser"`. Steps referencing the browser adapter with a persona configured for a different adapter MUST fail validation during manifest loading.
- **FR-009**: The preflight system MUST check for the browser binary (and any required dependencies like Chromium) and report missing dependencies before pipeline execution.
- **FR-010**: The browser adapter MUST support a configurable viewport size (default: 1280x720).
- **FR-011**: The browser adapter MUST support an optional CSS selector parameter for `get_text` and `get_html` commands to scope extraction to specific elements.
- **FR-012**: The browser adapter MUST log all commands and their outcomes via Wave's structured event system for observability.
- **FR-013**: The browser adapter MUST respect the `runtime.sandbox` configuration — when sandbox is enabled, the browser process MUST be launched with equivalent isolation constraints.
- **FR-014**: The adapter MUST support a `wait_for` option on commands that accept a CSS selector or a timeout duration, to handle dynamically rendered content.
- **FR-015**: The browser adapter MUST support a `get_html` command that returns the full page HTML (`document.documentElement.outerHTML`) by default when no CSS selector is provided. When a CSS selector is provided, it returns the `outerHTML` of matching elements.
- **FR-016**: The browser adapter MUST use a Go-native Chrome DevTools Protocol (CDP) client library (chromedp or rod) compiled into the Wave binary, requiring only a system-installed Chromium/Chrome browser at runtime. No Node.js or other runtime dependency is permitted.
- **FR-018**: The browser adapter MUST receive commands via a structured JSON protocol through the `Prompt` field of `AdapterRunConfig`. Each prompt contains a JSON array of `BrowserCommand` objects that the adapter executes sequentially, returning a JSON array of `BrowserResult` objects via stdout.
- **FR-019**: The browser capability MUST be declared using the existing `Persona.Adapter` field set to `"browser"` in the manifest. The `ResolveAdapter` function MUST be extended with a `"browser"` case. A new `capabilities` map field on `Persona` is NOT required — adapter selection already gates access.
- **FR-017**: The browser adapter MUST support headless execution by default, with an optional `headless: false` configuration for debugging purposes.

### Key Entities

- **BrowserCommand**: Represents a single browser action (navigate, screenshot, get_text, get_html, click, type) with parameters (URL, selector, value, timeout, viewport).
- **BrowserResult**: The structured response from a browser command containing status (success/error), output data (base64 image, extracted text, HTML), timing information, and error details.
- **BrowserAdapter**: The `AdapterRunner` implementation that manages the Chromium process lifecycle via CDP, command dispatch, and result collection. Resolved by name `"browser"` in `ResolveAdapter`.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A pipeline step using the `browser` adapter can navigate to a URL and return a screenshot artifact in under 10 seconds for a simple page.
- **SC-002**: The browser adapter correctly blocks navigation to domains not in the persona's `allowed_domains` list, with zero false negatives.
- **SC-003**: No browser state (cookies, localStorage, process) persists after a step completes — verified by inspecting the system process table and filesystem.
- **SC-004**: The preflight check correctly identifies when the browser binary is missing and provides an actionable error message with installation instructions.
- **SC-005**: All browser commands are logged as structured events, and the event stream includes command type, duration, and success/failure status.
- **SC-006**: The adapter handles browser crashes gracefully — the step fails with a clear error, and no orphan browser processes remain.
- **SC-007**: Integration test demonstrates a persona navigating to a local HTTP server, extracting text content, and returning it as an artifact that a downstream step can consume.

## Clarifications

The following ambiguities were identified and resolved during specification refinement.

### C1: Browser Engine Choice (FR-016)

**Question**: Should the browser adapter use a Go-native CDP client (chromedp/rod) compiled into the Wave binary, or a separate Node.js-based adapter (Playwright/Puppeteer)?

**Resolution**: Go-native CDP client (chromedp or rod).

**Rationale**: Wave's critical constraint #1 is "single static binary — no runtime dependencies except adapter binaries." Node.js would introduce a large runtime dependency. The existing adapter pattern (`ResolveAdapter`) already supports Go-native implementations (ClaudeAdapter, OpenCodeAdapter). A Go CDP library compiles directly into the binary and only requires a system Chromium installation, which the preflight system can detect.

### C2: `get_html` Default Behavior (FR-015)

**Question**: Should `get_html` return the full page HTML by default, or require a CSS selector?

**Resolution**: Return full page HTML by default when no selector is provided.

**Rationale**: FR-011 already specifies the selector as "optional" for both `get_text` and `get_html`. Returning `document.documentElement.outerHTML` when no selector is given is the standard browser API behavior and provides maximum utility for downstream personas that need the full DOM.

### C3: Capability Gating Mechanism (FR-008)

**Question**: Should a new `capabilities` map be added to the `Persona` struct to gate browser access, or should the existing `adapter` field suffice?

**Resolution**: Use the existing `Persona.Adapter` field set to `"browser"`. No new `capabilities` field needed.

**Rationale**: The current `Persona` struct (`internal/manifest/types.go:45`) has an `Adapter` field that already determines which adapter a persona uses. Adding a separate `capabilities` map would create redundant gating — if a persona's adapter is `"browser"`, it uses the browser; if not, it doesn't. The `ResolveAdapter` switch already enforces this pattern. Manifest validation can catch mismatches at load time.

### C4: Command Dispatch Protocol (FR-018, new)

**Question**: How does the browser adapter receive commands, given that `AdapterRunner.Run` accepts a free-text `Prompt` string?

**Resolution**: Commands are passed as a structured JSON array in the `Prompt` field. The adapter parses the JSON into `BrowserCommand` objects and returns results as a JSON array via stdout.

**Rationale**: The existing `ProcessGroupRunner` already uses `Prompt` as a generic input string. For the browser adapter, this field carries structured JSON rather than natural language. This avoids modifying the `AdapterRunner` interface while providing a well-defined protocol. The executor constructs the JSON command array from the step configuration.

### C5: Chromium Dependency Management (FR-009)

**Question**: How should Chromium be provided — bundled, auto-downloaded, or required as a system dependency?

**Resolution**: Required as a system dependency, detected by the preflight system.

**Rationale**: Bundling or auto-downloading Chromium (~150MB+) would violate the single-binary philosophy and introduce supply chain risk. The preflight system (`internal/preflight/preflight.go`) already checks for external dependencies before pipeline execution. Adding a Chromium check follows the established pattern. The preflight error message should include installation instructions for common platforms.
