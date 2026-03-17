# Research: Polish Log Streaming UX

**Feature**: #464 — Polish Log Streaming UX
**Date**: 2026-03-17

## Current State Analysis

### SSE Event Pipeline

The event flow is: `adapter/claude.go` emits `StreamEvent` → `event.Event` (with `State: "stream_activity"`, `ToolName`, `ToolTarget`, `Message`) → `SSEBroker.Emit()` → `handleSSE()` HTTP handler → client `sse.js` `EventSource`.

**Key finding**: `stream_activity` events are already received by the client but **silently discarded**. In `sse.js:204`:
```js
if (data.message && type !== 'step_progress' && type !== 'stream_activity') {
    appendEventToTimeline(data, type);
}
```

This means the entire SSE transport for log-like data is already working end-to-end. The gap is purely in the **client-side rendering** of these events.

### Event Payload Structure

From `internal/event/emitter.go`, `stream_activity` events carry:
- `timestamp` (time.Time) — server-side emission time
- `pipeline_id` (string) — run ID for filtering
- `step_id` (string) — which step produced this activity
- `tool_name` (string) — e.g., "Read", "Write", "Bash", "Grep"
- `tool_target` (string) — file path, command, pattern
- `message` (string) — human-readable activity description
- `tokens_used` (int) — running token count

### Current UI Architecture

- **No framework**: Plain vanilla JS, no React/Vue/Svelte
- **No build step**: Static files in `internal/webui/static/` embedded via `embed.go`
- **Server-rendered HTML**: Go templates in `internal/webui/templates/`
- **CSS variables**: Full dark/light theme support via CSS custom properties
- **Step cards**: Rendered in `step_card.html` partial — show status, progress, tokens, artifacts
- **Run detail layout**: Sidebar (DAG graph) + main column (steps list + events timeline)

### SSE Reconnection

The backend already supports `Last-Event-ID` based backfill in `handlers_sse.go:38-57`. The client-side `EventSource` has `onerror` handler that falls back to polling. The `retry: 3000` directive is sent on connection start.

### Dependency: #461 (Step Container)

The spec states dependency on #461 for the step container. However, the **existing step card** (`step_card.html`) already provides the container structure needed. The collapsible sections can be built by extending the existing step card with a toggle on the `.step-body` element.

## Technology Decisions

### Decision 1: ANSI Color Rendering

**Decision**: Use a lightweight inline ANSI-to-HTML converter (custom ~80 LOC function).

**Rationale**: No npm/bundler is available. The standard 16 + 256 color palette plus bold/italic/underline covers 99% of CLI tool output. A self-contained function avoids external dependencies and keeps the single-binary constraint satisfied.

**Alternatives rejected**:
- **ansi_up.js** (npm library): Requires a build step or CDN load; violates single-binary + no-runtime-deps constraint
- **Server-side conversion**: Violates FR-012 (no backend changes)

### Decision 2: Performance Strategy for 10k+ Lines

**Decision**: CSS containment (`contain: content`) + chunked DOM insertion via `requestAnimationFrame` batching (100ms frames).

**Rationale**: Per C5 in the spec, this avoids the complexity of virtual scrolling while meeting the 30fps scroll target. CSS containment tells the browser each log line's layout is independent, enabling paint-only updates. 10k DOM nodes with containment is well within modern browser capabilities (GitHub Actions uses the same approach).

**Alternatives rejected**:
- **Virtual scrolling**: Adds significant complexity for search highlighting, ANSI rendering, copy-to-clipboard, and line number alignment. Deferred to v2 if profiling shows need.
- **Web Workers for parsing**: Overkill for string matching; the main thread can handle 10k substring searches in <100ms.

### Decision 3: Search Implementation

**Decision**: Client-side highlight-only search with debounced input (300ms), match counter, and Ctrl+G / Shift+Ctrl+G navigation.

**Rationale**: Per C4, highlight-only preserves log context around matches. Debouncing prevents search-per-keystroke on large logs. Match navigation uses keyboard shortcuts familiar to developers (same as browser find).

### Decision 4: Auto-Scroll Mechanism

**Decision**: Use `IntersectionObserver` on a sentinel element at the bottom of the log container.

**Rationale**: More performant than scroll event listeners. When the sentinel is visible, auto-scroll is active. When the user scrolls up and the sentinel leaves the viewport, auto-scroll pauses. The "Jump to bottom" button calls `scrollIntoView()` on the sentinel.

**Alternatives rejected**:
- **Scroll event + threshold**: Requires `passive: true` listener and scroll position math; less clean and has edge cases with varying line heights.
- **MutationObserver**: Only triggers on DOM changes, doesn't detect manual scroll direction.

### Decision 5: Collapsible Step Sections

**Decision**: Extend existing `.step-card` with a `details`/`summary`-like toggle pattern using CSS `max-height` transitions and a click handler on `.step-header`.

**Rationale**: The step card structure already has `.step-header` (clickable) and `.step-body` (content). Adding a log output section below the body and toggling visibility via a CSS class is the minimal change. Default expanded for running/failed steps, collapsed for completed.

### Decision 6: File Organization

**Decision**: Create a new `log-viewer.js` file for all log viewer logic, loaded only on the run detail page.

**Rationale**: `sse.js` is already 300 LOC. The log viewer adds ~500-700 LOC of functionality (ANSI parsing, search, auto-scroll, collapsible sections). A separate file maintains readability and only loads on pages that need it. No bundler needed — just a `<script>` tag in `run_detail.html`.
