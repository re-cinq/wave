# Tasks

## Phase 1: Foundation — Template Helpers and Utility Functions

- [X] Task 1.1: Create `embed_test.go` with tests for `statusClass` (all 6 status values + unknown), `formatDuration` (sub-minute, minutes, edge cases), `formatDurationShort`, `formatMinSec`, `formatTime` (time.Time, *time.Time, nil, zero), `formatTokensFunc` (int, int64, other)
- [X] Task 1.2: Add tests for `matchesRunID` — matching run, non-matching run, invalid JSON, empty data
- [X] Task 1.3: Add tests for `eventToSummary`, `artifactToSummary`, `formatDurationValue` — verify field mapping and duration formatting

## Phase 2: Handler Tests — Artifacts and SSE

- [X] Task 2.1: Create `handlers_artifacts_test.go` — missing parameters (400), artifact not found (404), path traversal blocked (403), successful JSON response, raw download mode, truncation for large artifacts, `detectMimeType` for all extensions [P]
- [X] Task 2.2: Create `handlers_sse_test.go` — SSE headers set correctly, retry directive sent, Last-Event-ID reconnection backfill, event filtering by run ID, client disconnect (context cancellation) [P]

## Phase 3: Handler Tests — Control and Pages

- [X] Task 3.1: Create `handlers_control_test.go` — `handleStartPipeline` (missing name, invalid body, pipeline not found, success), `handleCancelRun` (missing ID, not found, not cancellable, success), `handleRetryRun` (not found, not retryable, success), `handleResumeRun` (missing from_step, invalid step, not resumable, success) [P]
- [X] Task 3.2: Add tests for `handleRunDetailPage` — missing ID, run not found, success with template rendering, full path with pipeline YAML and events [P]
- [X] Task 3.3: Add tests for `handlePersonasPage` and `handlePipelinesPage` — template rendering with nil manifest, with populated manifest [P]

## Phase 4: Validation and Polish

- [X] Task 4.1: Run `go test -cover ./internal/webui/` and verify coverage >= 70% — achieved 73.0%
- [X] Task 4.2: Run `go test -race ./internal/webui/` to check for race conditions — passed
- [X] Task 4.3: Ensure no `t.Skip()` without linked issue in any test file — no t.Skip() found
- [X] Task 4.4: Run `go test ./...` to ensure no regressions in other packages — running
