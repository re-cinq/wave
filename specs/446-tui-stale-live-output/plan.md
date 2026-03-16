# Implementation Plan: fix(tui) stale live output gaps

## Objective

Close the remaining gaps in the TUI's live output for detached/stale pipeline runs. The partial fix (4f55d7c) added polling infrastructure and UI indicators, but step tracking, stale process detection, immediate completion detection, and incremental polling need to be wired up.

## Approach

Fix four distinct gaps in the existing detached-run polling path. All changes are in `internal/tui/` — no new packages or architectural changes needed.

## File Mapping

| File | Action | Change |
|------|--------|--------|
| `internal/tui/live_output.go` | modify | Add `updateStepTrackingFromRecord()` method to update `stepNumber`, `totalSteps`, `currentStep` from stored records |
| `internal/tui/content.go` | modify | (1) Call `updateStepTrackingFromRecord` when loading stored records; (2) Switch polling from `Offset` to `AfterID`; (3) Add `IsProcessAlive` check in poll handler; (4) Check run status on initial load |
| `internal/tui/live_output_test.go` | modify | Add tests for step tracking from stored records |
| `internal/tui/content_test.go` | modify | Add tests for stale process detection and immediate completion detection |

## Architecture Decisions

1. **`updateStepTrackingFromRecord` on LiveOutputModel**: Rather than duplicating the step-tracking logic from `PipelineEventMsg` handler, create a dedicated method that mirrors it for `LogRecord` inputs. Called from content.go at all three code paths that load stored records (PipelineSelectedMsg, Enter key, PipelineLaunchedMsg).

2. **AfterID over Offset**: Replace `m.detachedPollOffset` (int count) with `m.detachedPollAfterID` (int64 record ID). Uses `EventQueryOptions.AfterID` which is already supported by the store. More reliable when events are inserted concurrently.

3. **IsProcessAlive in poll loop**: Add a PID check after event fetch. If PID > 0 and process is dead AND run status is still "running", mark run as failed/stale and stop polling. Reuses existing `IsProcessAlive()` from `stale_detector.go`.

4. **Immediate completion check**: After loading initial events from SQLite, immediately check `store.GetRun()` status. If already completed/failed/cancelled, set `completed = true` on the live output model and skip starting the poll timer.

## Risks

| Risk | Mitigation |
|------|------------|
| AfterID migration breaks existing offset-based callers | Only the detached poll handler uses offset; AfterID is additive |
| IsProcessAlive false positive on PID reuse | Unlikely within poll interval; staleRunCutoff already filters hour-old runs |
| Race between poll and PipelineEventMsg for in-process runs | In-process runs don't use detached polling (tailingPersisted is false) |

## Testing Strategy

- Unit tests for `updateStepTrackingFromRecord` with various event sequences
- Unit tests for `rebuildBuffer` with stored records and step tracking
- Integration-style test for immediate completion detection on initial load
- Test that AfterID-based polling correctly fetches only new events
- Test that dead-process detection transitions the live output to completed state
