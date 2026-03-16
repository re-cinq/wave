# Tasks

## Phase 1: Step Tracking from Stored Records

- [X] Task 1.1: Add `updateStepTrackingFromRecord(rec state.LogRecord)` method to `LiveOutputModel` in `live_output.go`. For `StateStarted` events with a StepID, increment `stepNumber`, set `currentStep`, track `stepOrder`. For pipeline-level started events with TotalSteps > 0 (encoded in Message or via a new field mapping), set `totalSteps`.
- [X] Task 1.2: Call `updateStepTrackingFromRecord` alongside `updateDashStepFromRecord` at all three stored-record loading sites in `content.go`: PipelineSelectedMsg handler (~line 1317), Enter key handler (~line 724), and PipelineLaunchedMsg handler (~line 1454).
- [X] Task 1.3: Call `updateStepTrackingFromRecord` in the `DetachedEventPollTickMsg` handler (~line 1558) for newly polled events.

## Phase 2: Immediate Completion Detection

- [X] Task 2.1: After initial event loading in PipelineSelectedMsg handler (~line 1324), check `store.GetRun(runID)` status. If completed/failed/cancelled, set `liveModel.completed = true`, `liveModel.tailingPersisted = false`, append summary line, and skip starting the poll timer. [P]
- [X] Task 2.2: Same immediate completion check in Enter key handler (~line 730). [P]
- [X] Task 2.3: Same immediate completion check in PipelineLaunchedMsg handler (~line 1460) — though typically empty for just-launched pipelines, guards against race conditions. [P]

## Phase 3: AfterID-based Incremental Polling

- [X] Task 3.1: Replace `detachedPollOffset int` field on `ContentModel` with `detachedPollAfterID int64`. Update all assignment sites to track `maxID` from loaded records instead of count.
- [X] Task 3.2: In `DetachedEventPollTickMsg` handler, use `EventQueryOptions{AfterID: m.detachedPollAfterID}` instead of `Offset`. Update `detachedPollAfterID` to the max `ID` from returned records.
- [X] Task 3.3: At all three initial-load sites, compute `maxID` from the loaded events and assign to `detachedPollAfterID`.

## Phase 4: Stale Process Detection in Poll Loop

- [X] Task 4.1: In `DetachedEventPollTickMsg` handler, after fetching events and checking run status, also fetch `RunRecord` and check `if run.PID > 0 && !IsProcessAlive(run.PID)`. If dead, update run status via `store.UpdateRunStatus(runID, "failed", "executor process no longer running", 0)`, mark live output as completed, stop polling.

## Phase 5: Testing

- [X] Task 5.1: Add unit test `TestLiveOutputModel_UpdateStepTrackingFromRecord` verifying stepNumber, currentStep, totalSteps, stepOrder are populated from stored records. [P]
- [X] Task 5.2: Add unit test `TestLiveOutputModel_TailingPersisted_HeaderShowsStepProgress` verifying the header renders step context from stored records. [P]
- [X] Task 5.3: Add test for AfterID-based polling: verify that `detachedPollAfterID` tracks max record ID and new polls use it correctly. [P]
- [X] Task 5.4: Add test for immediate completion detection: verify that selecting an already-completed run doesn't start polling and shows completed state immediately. [P]

## Phase 6: Validation

- [X] Task 6.1: Run `go test ./internal/tui/...` — all tests pass
- [X] Task 6.2: Run `go test -race ./internal/tui/...` — no race conditions
- [X] Task 6.3: Run `go vet ./internal/tui/...` — no warnings
