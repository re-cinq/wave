# Tasks: Polish Log Streaming UX

**Feature**: #464 — Polish Log Streaming UX
**Branch**: `464-log-streaming-ux`
**Generated**: 2026-03-17

## Phase 1: Setup — Project Scaffolding

- [X] T001 [P1] Create `internal/webui/static/log-viewer.js` with LogViewer class skeleton: constructor initializing `sections` (Map), `search` (SearchState), `connection` (ConnectionState), `batchBuffer` (array), `batchTimer` (null). Export singleton `window.logViewer`. Include `init(stepCards)` method stub that bootstraps LogSection objects from existing `.step-card` DOM elements on page load.
- [X] T002 [P1] Add `<script src="/static/log-viewer.js"></script>` to `internal/webui/templates/run_detail.html` before `sse.js` script tag. Add `logViewer.init()` call in the inline `<script>` block after the page loads (after the existing `connectSSE` call).

## Phase 2: Foundational — Core Data Flow (blocking all rendering)

- [X] T003 [P1] [US2] Modify `internal/webui/templates/partials/step_card.html` — add a `.step-log` container div inside `.step-body` at the end (after artifacts section): `<div class="step-log" id="log-{{.StepID}}" data-step-id="{{.StepID}}"><div class="step-log-content"></div><div class="step-log-sentinel"></div></div>`. Add a chevron indicator span (`▶`/`▼`) to `.step-header` for collapse toggle.
- [X] T004 [P1] [US2] In `log-viewer.js`, implement `LogSection` object factory: `createSection(stepId, stepName, status, element)` returning `{stepId, stepName, status, expanded, lines:[], lineCount:0, element, autoScroll:true}`. Set `expanded = true` when status is `"running"` or `"failed"`, `false` otherwise. Wire `init()` to create a LogSection for each `.step-card` on the page, storing in `this.sections`.
- [X] T005 [P1] [US2] In `log-viewer.js`, implement `toggleSection(stepId)` method: toggle `.step-collapsed` CSS class on the `.step-log` element, update chevron indicator text, update `section.expanded` state. Add click handler on `.step-header` elements in `init()`.
- [X] T006 [P1] In `log-viewer.js`, implement `addLine(stepId, eventData)` method: create a LogLine object `{lineNumber, timestamp, stepId, toolName, toolTarget, message, rawContent, htmlContent:null, element:null}` from the SSE event data. Assign sequential `lineNumber` per section. Push to `section.lines[]`, increment `section.lineCount`. Buffer the LogLine in `batchBuffer[]`.
- [X] T007 [P1] In `log-viewer.js`, implement `flushBatch()` method: use `requestAnimationFrame` to process `batchBuffer[]` in chunks. For each LogLine, create a `.log-line` div with `contain: content` style, append it to the matching `.step-log-content` element. Clear the buffer after flushing. Schedule via `requestAnimationFrame` with a 100ms batch window using `setTimeout`.
- [X] T008 [P1] Modify `internal/webui/static/sse.js` `handleSSEEvent()` function — add routing for `stream_activity` events to the log viewer: `if (type === 'stream_activity' && window.logViewer) { window.logViewer.addLine(data.step_id, data); }`. Place this before the existing filter at line 204.

## Phase 3: US1 — Auto-Scroll with Pause/Resume (P1)

- [X] T009 [P1] [US1] In `log-viewer.js`, implement auto-scroll using `IntersectionObserver` on `.step-log-sentinel` elements. In `init()`, create one observer per section. When sentinel is visible, set `section.autoScroll = true`. When sentinel leaves viewport (user scrolled up), set `section.autoScroll = false` and show "Jump to bottom" button.
- [X] T010 [P1] [US1] In `log-viewer.js`, create "Jump to bottom" button element per section (appended to `.step-log`). On click, call `sentinel.scrollIntoView({behavior: 'smooth'})` and re-enable `section.autoScroll`. Hide button when sentinel re-enters viewport.
- [X] T011 [P1] [US1] In `flushBatch()`, after appending new DOM elements, check `section.autoScroll` — if true, call `sentinel.scrollIntoView({behavior: 'instant'})` to keep the view pinned to the bottom.

## Phase 4: US2 — Collapsible Log Sections (P1, continued)

- [X] T012 [P1] [US2] Add CSS styles in `internal/webui/static/style.css` for collapsible sections: `.step-log` (overflow hidden, transition), `.step-collapsed .step-log` (max-height: 0, overflow: hidden), `.step-log-chevron` (inline indicator), `.step-header` (cursor: pointer). Support both light and dark themes using existing CSS custom properties.
- [X] T013 [P1] [US2] In `log-viewer.js`, update `updateStepCardState()` integration: when `sse.js` calls `updateStepCardState()` and a step transitions to `running` or `failed`, auto-expand that section. When step completes, collapse it. Modify `handleSSEEvent` in `sse.js` to call `window.logViewer.onStepStateChange(stepId, newState)` after updating the card.
- [X] T014 [P] [US2] Handle zero-output steps: in `flushBatch()` or on step completion, if a section has `lineCount === 0`, insert a `<div class="log-empty">No output</div>` placeholder inside `.step-log-content`.

## Phase 5: US3 — Timestamps and Line Numbers (P2)

- [X] T015 [P] [US3] In `log-viewer.js`, update the DOM creation in `flushBatch()` to render each `.log-line` with: `<span class="log-gutter">{lineNumber}</span><span class="log-time">{HH:MM:SS.mmm}</span><span class="log-tool">{toolName}</span><span class="log-content">{message}</span>`. Parse `timestamp` from event data (ISO 8601) and format as `HH:MM:SS.mmm`.
- [X] T016 [P] [US3] Add CSS in `style.css` for log line layout: `.log-line` (display: flex, font-family: monospace, font-size: 13px), `.log-gutter` (min-width: 4ch, text-align: right, color: dimmed, user-select: none), `.log-time` (min-width: 13ch, color: dimmed), `.log-tool` (badge pill style, min-width: 5ch), `.log-content` (flex: 1, white-space: pre-wrap, word-break: break-all). Ensure gutter accommodates up to 5-digit line numbers.

## Phase 6: US4 — ANSI Color Rendering (P2)

- [X] T017 [P] [US4] In `log-viewer.js`, implement `ansiToHtml(rawText)` function (~80 LOC): single-pass regex parser using `/\x1b\[([0-9;]*)m/g`. Support SGR codes: reset (0), bold (1), italic (3), underline (4), strikethrough (9). Standard 16 colors: fg (30-37, 90-97), bg (40-47, 100-107). 256-color: `38;5;N` fg, `48;5;N` bg. Track active styles in a state object, emit `<span style="...">` for each styled segment. Strip unsupported/malformed sequences. HTML-escape all text content.
- [X] T018 [P] [US4] Add CSS in `style.css` for ANSI color classes: `.ansi-bold`, `.ansi-italic`, `.ansi-underline`, `.ansi-strikethrough`, and color classes for the 16 standard colors (`.ansi-fg-{0-15}`, `.ansi-bg-{0-15}`). Use theme-aware colors that work in both light and dark modes.
- [X] T019 [US4] In `log-viewer.js`, integrate `ansiToHtml()` into the `addLine()` method: set `logLine.htmlContent = ansiToHtml(logLine.rawContent)`. Update `flushBatch()` to use `htmlContent` for the `.log-content` span's `innerHTML` instead of `textContent`.

## Phase 7: US5 — Search with Match Navigation (P2)

- [X] T020 [US5] In `internal/webui/templates/run_detail.html`, add a search bar UI in the Steps card header area: `<div class="log-search"><input type="text" id="log-search-input" placeholder="Search logs..." aria-label="Search log output"><span id="log-search-count"></span><button id="log-search-prev" title="Previous match (Shift+Ctrl+G)">&#x25B2;</button><button id="log-search-next" title="Next match (Ctrl+G)">&#x25BC;</button><button id="log-search-clear" title="Clear search">&#x2715;</button></div>`.
- [X] T021 [US5] In `log-viewer.js`, implement `SearchState` object and `search(query)` method: debounce 300ms, case-insensitive substring match across all lines in expanded sections. Build `matches[]` array of `{stepId, lineIndex, charStart, charEnd}`. Update `#log-search-count` with "N of M matches". Store `currentIndex = 0`.
- [X] T022 [US5] In `log-viewer.js`, implement search highlighting: for each match, wrap the matched substring in the `.log-content` span with `<mark class="search-match">`. For the current match, add `.search-current` class. Scroll the current match into view.
- [X] T023 [US5] In `log-viewer.js`, implement `nextMatch()` and `prevMatch()` methods: increment/decrement `currentIndex`, update `.search-current` class, scroll to new current match. Wire to next/prev buttons and keyboard shortcuts (Ctrl+G / Shift+Ctrl+G). Wire clear button to `clearSearch()` which removes all `<mark>` elements and resets SearchState.
- [X] T024 [US5] In `log-viewer.js`, update `addLine()` to check new lines against active search query. If match found, add to `matches[]` and update match count display.

## Phase 8: US6 — Large Log Performance (P2)

- [X] T025 [P] [US6] In `log-viewer.js`, optimize `flushBatch()`: batch DOM insertions using `DocumentFragment`. Limit batch size to 100 lines per `requestAnimationFrame` frame. If buffer exceeds 100, schedule another rAF for the remainder. This prevents long-running JS tasks from blocking the main thread.
- [X] T026 [P] [US6] Add CSS in `style.css`: `.log-line { contain: content; }` for layout isolation. Add `.step-log-content { contain: layout style; }` for the container. These tell the browser each line's layout is independent, enabling paint-only updates.
- [X] T027 [US6] In `log-viewer.js`, optimize search for large logs: limit highlighting to lines within ±50 lines of the viewport (use `IntersectionObserver` or scroll position check). Keep full match index in memory but only apply `<mark>` DOM changes for visible lines. Update marks on scroll.

## Phase 9: US7 — Log Download and Copy (P3)

- [X] T028 [P] [US7] In `log-viewer.js`, implement `downloadLog(stepId)` method: collect `section.lines.map(l => l.rawContent).join('\n')`, create a `Blob` with type `text/plain`, generate an object URL, create a temporary `<a>` element with `download` attribute set to `{stepId}.log`, click it, then revoke the URL.
- [X] T029 [P] [US7] In `log-viewer.js`, implement `copyLog(stepId)` method: collect raw content same as download, use `navigator.clipboard.writeText()` with fallback to `document.execCommand('copy')` via a temporary `<textarea>`. Show brief "Copied!" feedback on the button.
- [X] T030 [US7] In `internal/webui/templates/partials/step_card.html`, add download and copy buttons to `.step-header`: `<button class="btn-icon" onclick="window.logViewer.downloadLog('{{.StepID}}')" title="Download log">⬇</button><button class="btn-icon" onclick="window.logViewer.copyLog('{{.StepID}}')" title="Copy log">📋</button>`. Style as small icon buttons in `style.css`.

## Phase 10: US8 — SSE Reconnection Indicator (P3)

- [X] T031 [US8] In `internal/webui/templates/run_detail.html`, add a reconnection banner element above the steps list: `<div id="sse-reconnect-banner" class="reconnect-banner" hidden><span class="reconnect-message">Reconnecting...</span><button id="reconnect-retry" hidden onclick="window.logViewer.reconnect()">Retry</button></div>`.
- [X] T032 [US8] In `log-viewer.js`, implement `ConnectionState` management: `{status: "connected", retryCount: 0, lastEventId: null}`. Add methods `onDisconnect()` (show banner, increment retryCount, show retry button after 5 failures) and `onReconnect()` (hide banner, reset retryCount).
- [X] T033 [US8] Modify `internal/webui/static/sse.js` — enhance `onerror` handler: call `window.logViewer.onDisconnect()` when SSE connection drops. Enhance `onopen` handler: call `window.logViewer.onReconnect()` when connection re-establishes. Track retry count in the error handler.
- [X] T034 [US8] Add CSS in `style.css` for `.reconnect-banner`: fixed position above steps, yellow/warning background, centered text, smooth slide-down transition. `.reconnect-banner.disconnected` state with red background and retry button visible.

## Phase 11: Polish & Cross-Cutting Concerns

- [X] T035 [P] Add CSS in `style.css` for `.log-search` bar: flexbox layout, input styling consistent with existing form elements, button styling matching existing `.btn` pattern, match counter styling. Position within the Steps card header.
- [X] T036 [P] Add CSS in `style.css` for "Jump to bottom" button: `.jump-to-bottom` — fixed position within the log section, rounded pill shape, semi-transparent background, hover effect, fade-in/out transition.
- [X] T037 [P] Add CSS for search match highlighting: `mark.search-match` (background: yellow/theme-aware), `mark.search-current` (background: orange/theme-aware, outline). Ensure contrast works in both light and dark themes.
- [X] T038 Theme compatibility: verify all new CSS uses existing CSS custom properties (`--bg-primary`, `--text-primary`, `--border-color`, etc.) from `style.css`. Add any missing theme variables needed for log viewer elements (gutter color, ANSI palette overrides for dark mode).
- [X] T039 In `log-viewer.js`, handle the `updatePageFromAPI` polling rebuild in `sse.js`: the polling function at line 88-91 calls `stepsList.innerHTML = ''` which destroys log sections. Update `createStepCard()` in `sse.js` to include the `.step-log` container, and after rebuilding step cards, call `window.logViewer.reattach()` to re-bind LogSection objects to the new DOM elements without losing accumulated log lines.
- [X] T040 Edge case: handle completed (non-running) pipeline page load — all log output is loaded at once from the API events endpoint, auto-scroll is disabled, all sections collapsed except failed steps. In `init()`, if `runStatus` is terminal, fetch historical events via `/api/runs/{runID}` and populate sections from the `events` array.
- [X] T041 Verify `go test ./...` passes — no Go source files are modified (only templates, JS, CSS), but template compilation tests in `internal/webui/` must still pass with the modified templates.
