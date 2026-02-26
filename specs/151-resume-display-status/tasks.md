# Tasks

## Phase 1: Bug 1 — Synthetic Completion Events for Resume

- [X] Task 1.1: Add synthetic completion event emission in `ResumeFromStep()` after `loadResumeState()` and before execution begins. For each step ID in `resumeState.CompletedSteps`, emit `event.Event{State: "completed", StepID: stepID}` with the step's persona from the full pipeline. Location: `internal/pipeline/resume.go:96-125`.
- [X] Task 1.2: Add lookup helper to resolve step persona from full pipeline for synthetic events. Iterate `p.Steps` to find persona by step ID. Location: `internal/pipeline/resume.go`.
- [X] Task 1.3: Write unit tests for synthetic event emission in `internal/pipeline/resume_display_test.go`. Test cases: (a) resume from step 3 of 5 emits 2 synthetic completions, (b) resume from step 1 emits 0 synthetic completions, (c) synthetic events carry correct step ID and persona.

## Phase 2: Bug 2 — Activity Event Guard in Display

- [X] Task 2.1: In `BubbleTeaProgressDisplay.updateFromEvent()`, add guard to drop `stream_activity` events for steps that are `StateCompleted` or `StateNotStarted`. Location: `internal/display/bubbletea_progress.go:256-260`. [P]
- [X] Task 2.2: In `BasicProgressDisplay.EmitProgress()`, add same guard to skip `stream_activity` events for steps not in running state. Location: `internal/display/progress.go:641-654`. [P]
- [X] Task 2.3: In `BubbleTeaProgressDisplay.updateFromEvent()`, on step completion, also clear `stepToolActivity` for the completing step AND remove any stale global `lastToolName`/`lastToolTarget` if they belonged to the completing step. Location: `internal/display/bubbletea_progress.go:229-237`. [P]
- [X] Task 2.4: Write unit tests for activity event guard in `internal/display/bubbletea_progress_resume_test.go`. Test cases: (a) stream_activity for completed step is dropped, (b) stream_activity for not-started step is dropped, (c) stream_activity for running step is accepted.

## Phase 3: Testing

- [X] Task 3.1: Write integration test extending `cmd/wave/commands/run_test.go` to verify end-to-end resume display behavior with mock adapter. Verify prior steps get synthetic completion events and display state is correct.
- [X] Task 3.2: Run `go test ./...` to verify all existing tests pass with the changes.
- [X] Task 3.3: Run `go test -race ./...` to verify no race conditions in event emission and display update paths.

## Phase 4: Polish

- [X] Task 4.1: Verify changes work with all output formats (auto, text, json, quiet) by tracing event flow for each format.
- [X] Task 4.2: Final validation — run `go vet ./...` and ensure no linting issues.
