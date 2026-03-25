# Tasks

## Phase 1: Backend — Raise Event Limits

- [X] Task 1.1: Increase event limit in `handleRunDetailPage` from 100 to 5000
- [X] Task 1.2: Increase event limit in `handleAPIRunDetail` from 100 to 5000

## Phase 2: Backend — Paginated Step Events Endpoint

- [X] Task 2.1: Add `StepEventsResponse` type to `types.go`
- [X] Task 2.2: Create `handlers_events.go` with `handleAPIStepEvents` handler supporting `step`, `offset`, `limit` query params
- [X] Task 2.3: Register `GET /api/runs/{id}/step-events` route in `routes.go`

## Phase 3: Frontend — Event De-duplication

- [X] Task 3.1: Track seen event IDs in a `Set` in `LogViewer` to prevent duplicate rendering [P]
- [X] Task 3.2: Apply de-dup check in `addLine()` — skip if event ID already seen [P]

## Phase 4: Frontend — Repeat Collapsing

- [X] Task 4.1: Detect consecutive identical `stream_activity` messages in `addLine()`
- [X] Task 4.2: Create "repeated N times" summary DOM element with counter badge
- [X] Task 4.3: Update counter on subsequent identical messages instead of adding new lines
- [X] Task 4.4: Add CSS styles for `.log-repeated` summary row

## Phase 5: Frontend — Auto-scroll to Active Step

- [X] Task 5.1: On `onStepStateChange('running')`, scroll the step card into viewport [P]
- [X] Task 5.2: Add user-dismissable auto-scroll (disable if user manually scrolled page) [P]
- [X] Task 5.3: Add `.step-active` CSS highlight for the currently running step [P]

## Phase 6: Frontend — Search Filtering

- [X] Task 6.1: Add "Filter" toggle button next to the search bar
- [X] Task 6.2: When filter mode is active, hide non-matching log lines (set `display: none`) [P]
- [X] Task 6.3: Show match count in filter mode [P]
- [X] Task 6.4: Add CSS for filter-active state on the toggle button [P]

## Phase 7: Step Card Header Improvements

- [X] Task 7.1: Add status icon (spinner/checkmark/X) to step card header [P]
- [X] Task 7.2: Ensure duration badge is visible in collapsed state [P]
- [X] Task 7.3: Add chevron rotation animation on expand/collapse [P]

## Phase 8: Polish & Validation

- [X] Task 8.1: Verify no duplicate events in rendered view with both SSE and API fetch
- [X] Task 8.2: Test with terminal run (completed/failed) — historical log load
- [X] Task 8.3: Test with live run — SSE stream + auto-scroll
- [X] Task 8.4: Verify search/filter works across all sections
- [X] Task 8.5: Run `go test ./internal/webui/...` to confirm no regressions
