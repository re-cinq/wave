# Tasks

## Phase 1: Data Model

- [X] Task 1.1: Add `HandoverInfo` struct to `internal/display/types.go` with fields for artifact paths ([]string), contract status (string: "passed"/"failed"/"soft_failure"/""), contract schema name (string), and handover target step name (string)
- [X] Task 1.2: Add `HandoversByStep map[string]*HandoverInfo` field to `PipelineContext` in `internal/display/types.go`
- [X] Task 1.3: Add `verbose bool` field to `PipelineContext` in `internal/display/types.go` so the rendering layer knows whether to show handover metadata

## Phase 2: Event Capture — BubbleTea Path

- [X] Task 2.1: In `BubbleTeaProgressDisplay.updateFromEvent()` (`bubbletea_progress.go`), capture `contract_passed`, `contract_failed`, and `contract_soft_failure` events into a per-step `HandoverInfo` map
- [X] Task 2.2: In `BubbleTeaProgressDisplay.updateFromEvent()`, capture artifact paths from `completed` events (via `evt.Artifacts` field) into the per-step `HandoverInfo`
- [X] Task 2.3: In `BubbleTeaProgressDisplay.toPipelineContext()`, propagate the captured `HandoverInfo` map and verbose flag to `PipelineContext.HandoversByStep` and `PipelineContext.Verbose`
- [X] Task 2.4: Determine handover target step name: for each completed step, find the next step in `StepOrder` and set `HandoverInfo.TargetStep`

## Phase 3: Event Capture — Non-TTY Path

- [X] Task 3.1: Add per-step `HandoverInfo` tracking to `BasicProgressDisplay` struct in `progress.go` [P]
- [X] Task 3.2: In `BasicProgressDisplay.EmitProgress()`, capture `contract_passed`, `contract_failed`, `contract_soft_failure`, and `completed` event data into per-step `HandoverInfo` [P]

## Phase 4: Rendering — BubbleTea TUI

- [X] Task 4.1: In `renderCurrentStep()` in `bubbletea_model.go`, after the deliverables tree-format block for completed steps, add handover metadata rendering: artifact lines, contract line, handover target line — using ├─/└─ format, gated by `PipelineContext.Verbose`
- [X] Task 4.2: Ensure the last handover metadata line uses └─ (not ├─) and that deliverable lines are adjusted if both deliverables and handover metadata are present

## Phase 5: Rendering — Non-TTY (BasicProgressDisplay)

- [X] Task 5.1: In `BasicProgressDisplay.EmitProgress()`, when a `completed` event is received and verbose is true, emit the handover metadata lines (artifact, contract, handover target) in tree format after the step completion line

## Phase 6: Testing

- [X] Task 6.1: Add unit tests in `bubbletea_model_test.go` verifying that `renderCurrentStep()` renders handover metadata when verbose=true and omits it when verbose=false [P]
- [X] Task 6.2: Add unit tests in `progress_test.go` verifying that `BasicProgressDisplay.EmitProgress()` emits handover lines in verbose mode and omits them in non-verbose mode [P]
- [X] Task 6.3: Add unit tests verifying correct tree-format line connectors (├─ for intermediate lines, └─ for last line) when both deliverables and handover metadata are present [P]
- [X] Task 6.4: Run `go test ./internal/display/...` and `go test ./...` to verify no regressions

## Phase 7: Polish

- [X] Task 7.1: Verify non-verbose output is completely unchanged by running existing test suite
- [X] Task 7.2: Review terminal width handling — ensure handover metadata lines are properly truncated on narrow terminals
