# Tasks

## Phase 1: Backend Filter Passthrough

- [X] Task 1.1: Update `handleRunsPage` in `handlers_runs.go` to read `pipeline` query param and pass it to `ListRunsOptions`, and include current filter state (status, pipeline) in template data so templates can show active filters

## Phase 2: Status Indicators & Row Enrichment

- [X] Task 2.1: Update `run_row.html` — add Unicode status icons (✓ for completed, ● animated for running, ✕ for failed, ○ for cancelled, ◌ for pending) before badge text; make row clickable with `data-href` attribute [P]
- [X] Task 2.2: Add status icon CSS — `.status-icon` class with color mapping, `.status-icon-running` with pulse animation; add `.run-row-clickable` cursor pointer and hover highlight [P]
- [X] Task 2.3: Add relative timestamp JS — `relativeTime(isoString)` function that converts to "Xs ago", "Xm ago", "Xh ago", "Xd ago"; update `run_row.html` to use `<time>` element with `datetime` attr and relative display [P]

## Phase 3: Filter Controls

- [X] Task 3.1: Update `runs.html` — add pipeline dropdown (populated from `.Pipelines`), date range `<input type="date">` for "since" filter; wrap in `.filters` container [P]
- [X] Task 3.2: Add active-filter indication — show `.filters-active` bar with "Filtering by: status, pipeline" text and "Clear filters" button when any filter is active [P]
- [X] Task 3.3: Update `filterRuns()` in `app.js` to coordinate all three filters (status, pipeline, since) into URL params; add `clearFilters()` function [P]

## Phase 4: Sorting Controls

- [X] Task 4.1: Update `runs.html` `<thead>` — make Status, Pipeline, Started, Duration columns sortable with `data-sort` attribute and sort-direction indicator (▲/▼)
- [X] Task 4.2: Add client-side `sortTable(column, direction)` in `app.js` — sort DOM rows by text content for status/pipeline, by `data-timestamp`/`data-duration` for time columns; persist sort in URL param `sort=column&dir=asc|desc`
- [X] Task 4.3: Add sort header CSS — `.sortable` cursor, `.sort-indicator` arrow styling, `.sort-active` highlight

## Phase 5: Pagination & Empty State Polish

- [X] Task 5.1: Polish pagination — style "Load More" as a wider centered button with count hint ("Load more runs..."); add subtle separator line above [P]
- [X] Task 5.2: Update empty state — differentiate between "no runs at all" (show getting-started message) and "no runs matching filters" (show "No runs found matching your filters" with clear-filters link) [P]

## Phase 6: Row Hover & Click Affordances

- [X] Task 6.1: Add row click handler in `app.js` — `document.querySelectorAll('.run-row[data-href]')` click navigates to detail page; exclude clicks on `<a>` elements within the row
- [X] Task 6.2: Add hover styles — subtle left-border color transition on hover matching status color; cursor: pointer on clickable rows

## Phase 7: Validation

- [X] Task 7.1: Run `go test ./internal/webui/...` to verify template tests pass
- [X] Task 7.2: Verify all 7 acceptance criteria are addressed in the implementation
