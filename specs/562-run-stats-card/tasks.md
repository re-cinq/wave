# Tasks

## Phase 1: Backend — Extend Types and Mapping

- [X] Task 1.1: Add `Input`, `LinkedURL`, `FormattedStartedAt`, `FormattedCompletedAt` fields to `RunSummary` in `internal/webui/types.go`
- [X] Task 1.2: Add `parseLinkedURL(input string) string` helper in `internal/webui/handlers_runs.go` — regex-based extraction of GitHub issue/PR URLs from input text
- [X] Task 1.3: Update `runToSummary` in `internal/webui/handlers_runs.go` to populate `Input` (full text), `LinkedURL` (from parser), `FormattedStartedAt`, and `FormattedCompletedAt`

## Phase 2: Frontend — Stats Card Grid

- [X] Task 2.1: Replace `run-summary-bar` in `internal/webui/templates/run_detail.html` with a stats card grid containing: Run ID (copyable), Pipeline, Input (expandable), Start Time, Duration, Finish Time, Tokens, Branch, Linked Issue/PR [P]
- [X] Task 2.2: Add CSS for `.stats-card-grid`, `.stats-card`, `.stats-card-label`, `.stats-card-value`, `.stats-card-copy-btn`, responsive breakpoints in `internal/webui/static/style.css` [P]
- [X] Task 2.3: Add copy-to-clipboard JS for Run ID in the `{{define "scripts"}}` block of `run_detail.html`

## Phase 3: Testing

- [X] Task 3.1: Add table-driven unit tests for `parseLinkedURL` — GitHub issue URLs, PR URLs, non-GitHub URLs, empty input, multiple URLs in input
- [X] Task 3.2: Add unit tests for `runToSummary` verifying `Input`, `LinkedURL`, `FormattedStartedAt`, `FormattedCompletedAt` population
- [X] Task 3.3: Verify existing integration tests pass (`TestHandleRunDetailPage_ValidRun`, `TestHandleRunDetailPage_WithPipelineAndEvents`)

## Phase 4: Polish

- [X] Task 4.1: Run `go test ./internal/webui/...` and fix any regressions
- [X] Task 4.2: Run `go vet ./...` and verify clean output
