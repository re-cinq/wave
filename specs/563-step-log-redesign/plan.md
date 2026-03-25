# Implementation Plan — Step Log View Redesign (#563)

## 1. Objective

Bring the web UI step log viewer to GitHub Actions parity by raising event limits, collapsing repetitive log lines, adding duration badges to step headers, auto-scrolling to the active step, and wiring up the existing search infrastructure to filter log lines.

## 2. Approach

The implementation splits into two layers:

**Backend**: Increase the hard-coded event limit from 100 to 5000 for the run detail page and API endpoint. Add a dedicated paginated events API endpoint (`GET /api/runs/{id}/step-events`) that supports `step`, `offset`, and `limit` query parameters for lazy-loading step logs.

**Frontend**: Enhance `log-viewer.js` to collapse consecutive identical `stream_activity` events into "repeated N times" summary rows. Add auto-scroll-to-active-step behavior on step state transitions. Improve the step card header to surface duration badges and status icons more prominently. Wire the existing search UI to also support line-level filtering (hide non-matching lines).

## 3. File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/webui/handlers_runs.go` | modify | Raise event limit from 100 to 5000 in `handleRunDetailPage` and `handleAPIRunDetail` |
| `internal/webui/handlers_events.go` | create | New handler for `GET /api/runs/{id}/step-events` with pagination params |
| `internal/webui/routes.go` | modify | Register the new `/api/runs/{id}/step-events` route |
| `internal/webui/types.go` | modify | Add `StepEventsResponse` type for the paginated events endpoint |
| `internal/webui/static/log-viewer.js` | modify | Add repeat-collapsing, auto-scroll-to-active-step, line filtering |
| `internal/webui/static/sse.js` | modify | Auto-scroll to active step card on step state transition |
| `internal/webui/templates/partials/step_card.html` | modify | Add status icon and chevron indicator to header |
| `internal/webui/static/style.css` | modify | Add styles for repeated-events row, active-step highlight, search-filter mode |

## 4. Architecture Decisions

### AD-1: Raise limit vs. pagination
Raise the default limit to 5000 for both the HTML page and JSON API. For truly long-running pipelines (>5000 events per step), add a separate paginated endpoint that the frontend can call on scroll-up. This avoids breaking the existing SSE flow while supporting large logs.

### AD-2: Repeat collapsing in frontend, not backend
Collapsing consecutive identical `stream_activity` events is a view concern. The backend stores all events; the frontend detects consecutive identical messages during `addLine()` and renders a "repeated N times" indicator. This preserves full audit trail in the DB.

### AD-3: De-duplication via event ID tracking
The log viewer already receives events from both SSE and API fetch (on page load for terminal runs). Track event IDs in a `Set` to prevent duplicate rendering. The `EventSummary.ID` field (int64 from the DB) serves as the dedup key.

### AD-4: Auto-scroll targets step cards, not individual lines
When a step transitions to "running", scroll the step card into view (not individual log lines). Within the step, the existing per-section `autoScroll` flag handles scrolling to the latest line.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| 5000 events may cause page lag on slow devices | Frontend already uses `requestAnimationFrame` batch rendering with 100-line-per-frame cap; repeat collapsing reduces DOM nodes further |
| Paginated endpoint may introduce complexity | Keep it simple: offset/limit SQL query, no cursor needed since events are append-only |
| Repeat-collapsing may hide important state changes | Only collapse consecutive `stream_activity` events with identical messages; never collapse state transitions |
| De-dup Set grows unbounded for very long runs | Cap the set at 10000 entries; older entries won't re-appear from SSE |

## 6. Testing Strategy

- **Backend**: Unit test for new paginated events handler with mock state store, verifying offset/limit/step filtering
- **Backend**: Verify existing `handlers_test.go` still passes after limit change
- **Frontend**: Manual verification of repeat-collapsing, auto-scroll, search filtering, and dedup behavior
- **Integration**: Test with a live pipeline run to confirm SSE + API fetch dedup works correctly
