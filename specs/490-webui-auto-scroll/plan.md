# Implementation Plan: WebUI Log Auto-Scroll

## Objective

Add auto-scroll tracking to the log viewer so new log lines keep the viewport at the bottom during active streaming, with a "Jump to Bottom" button that appears when the user scrolls away.

## Approach

The log viewer already batches DOM insertions via `flushBatch()`. We add per-section auto-scroll state and a scroll-position check after each batch flush. A floating "Jump to Bottom" button per step section shows/hides based on scroll position.

### Design Decisions

1. **Scroll threshold**: Use a 50px threshold from the bottom to determine "at bottom" — accounts for fractional pixels and minor offsets
2. **Per-section state**: Store `autoScroll: true/false` on each `LogSection` object (already tracked in `this.sections` Map)
3. **Button placement**: Insert the button inside each `.step-log` container (positioned absolutely relative to the step log) — this avoids global button conflicts and keeps per-section semantics
4. **Scroll listener**: Attach a single `scroll` event listener per `.step-log-content` element to update auto-scroll state and button visibility
5. **Search interaction**: When `_scrollToCurrentMatch` fires, temporarily suppress auto-scroll to avoid fighting the search navigation

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/webui/static/log-viewer.js` | modify | Add auto-scroll state to sections, scroll listener, flushBatch auto-scroll, Jump to Bottom wiring |
| `internal/webui/static/style.css` | modify | Add CSS rules for `.jump-to-bottom` button under the existing placeholder comment at line 944 |
| `internal/webui/templates/partials/step_card.html` | modify | Add Jump to Bottom button element inside `.step-log` container |

## Risks

| Risk | Mitigation |
|------|------------|
| Performance with high-frequency log lines | Already mitigated by existing `flushBatch` with `requestAnimationFrame` and 100-line batching; auto-scroll adds one `scrollTop` assignment per frame |
| Scroll event listener overhead | Use passive listener; only check threshold arithmetic (no DOM queries) |
| Conflict with search scroll | Disable auto-scroll when `_scrollToCurrentMatch` is called; re-enable only via Jump to Bottom button or manual scroll to bottom |

## Testing Strategy

This is a frontend-only change with no Go code modifications. Testing:
- Manual verification that `go test ./...` still passes (no Go changes)
- Visual verification of auto-scroll behavior (outside pipeline scope)
- The existing webui handler tests cover template rendering; the new button element should render without breaking template parsing
