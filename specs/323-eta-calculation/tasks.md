# Tasks

## Phase 1: Core ETACalculator

- [X] Task 1.1: Create `internal/pipeline/eta.go` with `ETACalculator` struct that holds historical step durations, current-run durations, step order, and a mutex for thread safety
- [X] Task 1.2: Implement `NewETACalculator(store state.StateStore, pipelineName string, stepIDs []string)` that queries `GetStepPerformanceStats` for each step and caches average durations
- [X] Task 1.3: Implement `RecordStepCompletion(stepID string, durationMs int64)` to track actual durations from the current run
- [X] Task 1.4: Implement `RemainingMs() int64` that sums estimated durations for incomplete steps (using current-run actual if available, else historical average)
- [X] Task 1.5: Implement `AverageStepMs() int64` that returns the average across all steps with data

## Phase 2: Executor Integration

- [X] Task 2.1: Add `etaCalculator *ETACalculator` field to `DefaultPipelineExecutor`
- [X] Task 2.2: Initialize `ETACalculator` in `Execute()` after topological sort, passing store + pipeline name + step IDs
- [X] Task 2.3: Call `etaCalculator.RecordStepCompletion()` after each successful step in `executeStep()`
- [X] Task 2.4: Augment `startProgressTicker` heartbeat events with `EstimatedTimeMs` from `etaCalculator.RemainingMs()`
- [X] Task 2.5: Emit `StateETAUpdated` event on step completion with updated ETA

## Phase 3: Display Integration

- [X] Task 3.1: In `BubbleTeaProgressDisplay.EmitProgress()`, capture `EstimatedTimeMs` from incoming events and store it [P]
- [X] Task 3.2: In `BubbleTeaProgressDisplay.buildPipelineContext()`, populate `EstimatedTimeMs` and `AverageStepTimeMs` from tracked data instead of 0 [P]
- [X] Task 3.3: In `ProgressDisplay.EmitProgress()`, capture `EstimatedTimeMs` from incoming events and store it [P]
- [X] Task 3.4: In `ProgressDisplay.toPipelineContext()`, populate `EstimatedTimeMs` and `AverageStepTimeMs` from tracked data instead of 0 [P]

## Phase 4: Testing

- [X] Task 4.1: Write unit tests for ETACalculator — no history returns 0
- [X] Task 4.2: Write unit tests for ETACalculator — historical data produces correct estimate
- [X] Task 4.3: Write unit tests for ETACalculator — step completion updates remaining correctly
- [X] Task 4.4: Write unit tests for ETACalculator — concurrent access safety
- [X] Task 4.5: Run `go test ./...` to verify no regressions [P]
- [X] Task 4.6: Run `go test -race ./...` to verify no race conditions [P]

## Phase 5: Polish

- [X] Task 5.1: Verify dashboard renders ETA when `EstimatedTimeMs > 0` (already implemented in dashboard.go:412-416)
- [X] Task 5.2: Verify TUI formats `StateETAUpdated` events correctly (already implemented in live_output.go:265-270)
- [X] Task 5.3: Ensure first pipeline run (no history) shows no ETA rather than incorrect values
