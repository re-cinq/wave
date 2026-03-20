# audit: partial — webui log streaming auto-scroll (#464)

**Issue**: [#490](https://github.com/re-cinq/wave/issues/490)
**Labels**: audit
**Author**: nextlevelshit
**Source**: #464 — feat(webui): polish log streaming UX with auto-scroll, search, and collapsible sections

## Problem

The webui log viewer has search and collapsible sections implemented, but auto-scroll is incomplete:

- `log-viewer.js` has no auto-scroll logic — new log lines append to the DOM but the viewport does not follow
- `style.css:944` has a CSS comment placeholder for "jump to bottom button" but no actual rules
- `run_detail.html` has no Jump to Bottom button element in the DOM
- The `flushBatch` method appends lines without scrolling

## Evidence

- `internal/webui/static/log-viewer.js:94` — search input referenced
- `internal/webui/static/style.css:851` — collapsible step log sections CSS
- `internal/webui/static/style.css:944` — jump to bottom button CSS defined (empty)
- `internal/webui/templates/run_detail.html:103` — search UI elements present

## Acceptance Criteria

1. **Auto-scroll on new lines**: When a step log section is expanded and the user is at or near the bottom, new log lines should auto-scroll the viewport to keep the latest output visible
2. **Pause on manual scroll-up**: If the user scrolls up to read earlier output, auto-scroll pauses — new lines still append but the viewport stays put
3. **Jump to Bottom button**: A floating button appears when the user has scrolled away from the bottom; clicking it scrolls to the latest line and re-enables auto-scroll
4. **Button hides at bottom**: When the user is at the bottom (either by scrolling manually or via the button), the button hides
5. **Per-section behavior**: Auto-scroll state is tracked per step log section, not globally
6. **No interference with search**: Auto-scroll should not conflict with the existing search/highlight scroll behavior (search scroll takes priority)

## Scope

- `internal/webui/static/log-viewer.js` — add auto-scroll tracking and Jump to Bottom wiring
- `internal/webui/static/style.css` — add CSS for the Jump to Bottom button
- `internal/webui/templates/run_detail.html` or `templates/partials/step_card.html` — add Jump to Bottom button element (if not already present)
