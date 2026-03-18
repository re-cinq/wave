# Tasks

## Phase 1: Run List View Polish
- [X] Task 1.1: Add progress column to run list table — show step completion ratio (e.g., "3/5 steps") [P]
- [X] Task 1.2: Improve run row status indicators — add animated spinner icon for running, checkmark for completed, X for failed [P]
- [X] Task 1.3: Add input preview column or tooltip to run rows — show truncated pipeline input [P]
- [X] Task 1.4: Enhance empty state with icon, descriptive text, and CTA button
- [X] Task 1.5: Improve filter bar — add "active filter" chips with individual clear buttons

## Phase 2: Run Detail View Polish
- [X] Task 2.1: Add run summary section — total duration, steps completed/total, total tokens, pipeline description
- [X] Task 2.2: Convert step cards to GitHub Actions-style vertical timeline — connected vertical line with status dots
- [X] Task 2.3: Improve step duration display — show elapsed time updating live for running steps
- [X] Task 2.4: Add step progress percentage based on token usage or elapsed time heuristic
- [X] Task 2.5: Improve error display — structured error sections with recovery hints prominently visible
- [X] Task 2.6: Improve DAG sidebar — highlight current running step with animation

## Phase 3: Log Viewer UX
- [X] Task 3.1: Add line-wrap toggle button to log viewer toolbar [P]
- [X] Task 3.2: Improve auto-scroll behavior — only auto-scroll when user is at bottom, show "new lines" indicator [P]
- [X] Task 3.3: Default collapsed sections to expand on click with smooth animation
- [X] Task 3.4: Add log level filter (all/errors/warnings) to search bar

## Phase 4: Global Consistency
- [X] Task 4.1: Add consistent loading spinners to all page-level data fetches [P]
- [X] Task 4.2: Add consistent empty states with icons to all list/grid views [P]
- [X] Task 4.3: Style 404 page (`notfound.html`) with proper layout and navigation [P]
- [X] Task 4.4: Fix responsive layout issues — nav overflow on mobile, table horizontal scroll
- [X] Task 4.5: Ensure consistent card hover effects and spacing across pipelines, personas, contracts, skills, compose pages [P]

## Phase 5: Test Coverage
- [X] Task 5.1: Add handler tests for `handlers_compose.go` [P]
- [X] Task 5.2: Add handler tests for `handlers_contracts.go` [P]
- [X] Task 5.3: Add handler tests for `handlers_pipelines.go` [P]
- [X] Task 5.4: Add handler tests for `handlers_personas.go` [P]
- [X] Task 5.5: Add handler tests for `handlers_skills.go` [P]
- [X] Task 5.6: Add SSE reconnection edge case tests to `handlers_sse_test.go`
- [X] Task 5.7: Add pagination edge case tests to `pagination_test.go`

## Phase 6: Final Validation
- [X] Task 6.1: Run `go test -race ./...` and fix any failures
- [X] Task 6.2: Visual audit of all 15 pages in dark and light mode
- [X] Task 6.3: Verify responsive layout on mobile viewport (768px and below)
