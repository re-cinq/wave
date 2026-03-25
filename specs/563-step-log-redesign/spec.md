# feat(webui): step log view redesign — GitHub Actions parity

**Issue**: [#563](https://github.com/re-cinq/wave/issues/563)
**Labels**: enhancement, ux, frontend
**Author**: nextlevelshit
**Parent**: Extracted from #550 — Feature 3

## Problem

The step log view works but lacks GitHub Actions polish: no collapsing of repetitive events, no duration badges on section headers, event history capped at 100 lines.

## Changes Required

### Backend
- Increase event limit in `handleRunDetailPage` from 100 to 1000+ (or paginate)
- Consider a dedicated `/api/runs/{id}/events?step=<id>&offset=<n>&limit=<n>` endpoint for lazy loading

### Frontend (`log-viewer.js` + `step_card.html`)
- Add collapsible section headers per step with duration badge and status icon
- Collapse consecutive identical `stream_activity` events into "repeated N times" summary
- Auto-scroll to the currently active step during live runs (SSE already provides step transitions)
- Add keyword search/filter within the log panel (foundation exists in `log-viewer.js`)
- Expandable error details (currently truncates at 200 chars with show/hide)

### Stretch
- Lazy-load older events on scroll-up for long-running pipelines
- Syntax highlighting for code blocks in log output

## Acceptance Criteria

- [ ] Step sections are collapsible with duration badges
- [ ] Repetitive consecutive events are collapsed with count
- [ ] Auto-scroll follows the active step during live execution
- [ ] Keyword search filters log lines
- [ ] Historical logs show more than 100 events
- [ ] No duplicate event lines in the rendered view
