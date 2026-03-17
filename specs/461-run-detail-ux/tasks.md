# Tasks

## Phase 1: Data Layer & Duration Formatting

- [X] Task 1.1: Add `FormattedStartedAt` string field to `StepDetail` in `types.go` for template rendering
- [X] Task 1.2: Populate `FormattedStartedAt` in `buildStepDetails()` in `handlers_runs.go` using the existing `formatTime` template function format
- [X] Task 1.3: Add JS `formatDuration(ms)` function in `sse.js` that mirrors Go's `formatDurationValue` output ("Xs", "XmYs", "XhYm")
- [X] Task 1.4: Add JS `formatStartTime(isoString)` function for displaying start times in SSE-updated cards

## Phase 2: Step Card Collapsible UI

- [X] Task 2.1: Refactor `step_card.html` to use collapsible structure — header always visible, body togglable [P]
- [X] Task 2.2: Add collapse toggle button (chevron icon) to step header [P]
- [X] Task 2.3: Add CSS for collapsible step cards — `.step-card-collapsed .step-body { display: none }` transitions [P]
- [X] Task 2.4: Add JS `toggleStepCard(stepID)` function and auto-collapse logic (expand running/failed, collapse completed/pending) [P]
- [X] Task 2.5: Preserve collapse state across SSE DOM updates — maintain a `Set` of expanded step IDs

## Phase 3: Enhanced Step Display

- [X] Task 3.1: Add start time display to step header in `step_card.html` [P]
- [X] Task 3.2: Add animated running indicator (CSS spinner) next to running step badge [P]
- [X] Task 3.3: Enhance failed step error display — add expand/collapse for long error messages [P]
- [X] Task 3.4: Update `createStepCard()` in `sse.js` to generate matching HTML with collapsible structure, start time, and animated indicator [P]
- [X] Task 3.5: Add duration display to DAG SVG nodes in `dag_svg.html` (small text below persona)

## Phase 4: Run Header Enhancement

- [X] Task 4.1: Add start time and trigger info (from tags) to run header in `run_detail.html`
- [X] Task 4.2: Ensure human-friendly duration format ("2m 34s") is used consistently in run header

## Phase 5: Log Output Styling

- [X] Task 5.1: Add CSS classes for log highlighting — `.log-error`, `.log-warning`, `.log-info` with appropriate colors [P]
- [X] Task 5.2: Add line number rendering to artifact content display in the `toggleArtifact` JS function [P]
- [X] Task 5.3: Add keyword pattern highlighting in artifact viewer JS — wrap error/warning patterns in styled spans [P]

## Phase 6: CSS Polish & Responsive

- [X] Task 6.1: Add CSS for the running spinner animation (@keyframes spin) [P]
- [X] Task 6.2: Update responsive breakpoints — ensure collapsible steps work on mobile [P]
- [X] Task 6.3: Add `prefers-reduced-motion` support for new animations

## Phase 7: Testing & Validation

- [X] Task 7.1: Add unit tests for `formatDurationValue` edge cases (sub-second, hours, zero, large values)
- [X] Task 7.2: Add template rendering test for run detail page with enhanced step cards
- [X] Task 7.3: Manual validation — verify all 8 acceptance criteria against a live pipeline run
- [X] Task 7.4: Run `go test ./...` to ensure no regressions
