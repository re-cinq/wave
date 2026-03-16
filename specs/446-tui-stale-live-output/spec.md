# fix(tui): live output unavailable for detached/stale pipeline runs

**Issue**: [#446](https://github.com/re-cinq/wave/issues/446)
**Labels**: bug
**Author**: nextlevelshit

## Description

TUI live output only works for pipelines launched in the current TUI session. Detached runs or runs from previous sessions show "live output unavailable".

### Root Cause

Live events come from in-memory `LiveOutputModel`, not from persisted event stream. There's no reconnect/tail mechanism for existing runs.

### Partial Fix (4f55d7c)

Commit 4f55d7c added:
- `tailingPersisted` flag and UI indicators (header/footer) on LiveOutputModel
- Changed `renderRunningInfo` text from "live output unavailable" to "Press [Enter] to view live event dashboard"
- `DetachedEventPollTickMsg` polling infrastructure in content.go
- `storedRecords` / `shouldFormatRecord` / `formatStoredEvent` for SQLite-backed event display
- Dashboard population via `updateDashStepFromRecord`

### Remaining Gaps

The reopening comment indicates the stale run live output gap persists. Analysis of the codebase reveals these specific gaps:

1. **Step tracking not updated from stored records**: `LiveOutputModel.stepNumber`, `totalSteps`, and `currentStep` are only updated from in-memory `PipelineEventMsg` (line 706-728 of live_output.go). For detached runs where events come from SQLite via `updateDashStepFromRecord`, these fields stay at zero. The header shows "▶ Tailing persisted events" without step context.

2. **No stale process detection during polling**: `DetachedEventPollTickMsg` handler checks `run.Status` from SQLite to detect completion but does NOT check `IsProcessAlive(run.PID)`. If a detached process crashes without updating its DB status, the TUI polls forever. `IsProcessAlive` exists in `stale_detector.go` and is used in `FetchRunningPipelines()` and `Cancel()`, but not in the poll loop.

3. **Already-completed runs shown as "running"**: When a user selects a detached run that already completed between list refresh intervals, the first poll tick (2s delay) eventually detects completion. But for the initial 2 seconds, the UI incorrectly shows it as running. The initial event load should check run status immediately.

4. **No AfterID-based incremental polling**: The polling uses `Offset` (count-based) which can miss events if concurrent inserts happen. `EventQueryOptions.AfterID` field exists but is unused in the poll handler. Using `AfterID` (the max `LogRecord.ID` seen) would be more reliable.

## Acceptance Criteria

- [ ] Detached/stale pipeline runs show live event dashboard with step progress in header
- [ ] Step number, total steps, and current step are populated from stored records
- [ ] Polling detects dead processes via `IsProcessAlive` and transitions to completed/failed state
- [ ] Already-completed runs detected immediately on selection (no 2s delay)
- [ ] Incremental polling uses `AfterID` instead of count-based offset for reliability
- [ ] Existing tests pass, new tests cover the gap fixes
