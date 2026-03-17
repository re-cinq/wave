# Tasks — #461 Run Detail UX

## Phase 1: Backend Data
- [X] 1.1 Add FormattedStartedAt to StepDetail type
- [X] 1.2 Populate FormattedStartedAt in buildStepDetails
- [X] 1.3 Add JS formatDuration function
- [X] 1.4 Add JS formatStartTime function

## Phase 2: Collapsible Step Cards
- [X] 2.1 Refactor step_card.html for collapsible structure
- [X] 2.2 Add collapse toggle button to step header
- [X] 2.3 Add CSS for collapsible step cards
- [X] 2.4 Add toggleStepCard JS function with auto-collapse
- [X] 2.5 Preserve collapse state across SSE updates

## Phase 3: Step Detail Enhancements
- [X] 3.1 Add start time display to step header
- [X] 3.2 Add animated running indicator
- [X] 3.3 Enhance failed step error display
- [X] 3.4 Update createStepCard JS for new HTML structure
- [X] 3.5 Add duration display to DAG SVG nodes

## Phase 4: Run Header
- [X] 4.1 Enhance run header with start time and trigger info
- [X] 4.2 Ensure consistent human-friendly duration format

## Phase 5: Log Readability
- [X] 5.1 Add CSS classes for log highlighting
- [X] 5.2 Add line numbers to artifact content display
- [X] 5.3 Add keyword pattern highlighting in artifact viewer

## Phase 6: Polish
- [X] 6.1 Add CSS running spinner animation
- [X] 6.2 Update responsive breakpoints for collapsible steps
- [X] 6.3 Add prefers-reduced-motion support

## Phase 7: Testing
- [X] 7.1 Add formatDurationValue unit tests
- [X] 7.2 Add run detail template rendering test (existing tests cover this)
- [X] 7.3 Manual acceptance criteria validation (verified via code review)
- [X] 7.4 Run full test suite
