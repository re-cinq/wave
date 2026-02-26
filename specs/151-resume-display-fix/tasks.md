# Tasks

## Phase 1: Core Fix — Synthetic Completion Events

- [X] Task 1.1: Add synthetic completion event emission in `ResumeFromStep()` (`internal/pipeline/resume.go`)
  - After the existing "resuming" event block (lines 96-125), iterate over `resumeState.CompletedSteps`
  - For each completed step, look up the step's persona from the full pipeline `p.Steps`
  - Emit an `event.Event` with `State: "completed"`, `StepID: step.ID`, `Persona: step.Persona`, `Message: "completed in prior run"`, `DurationMs: 0`
  - Keep the existing "resuming" summary event as-is for backward compatibility

- [X] Task 1.2: Add unit test for synthetic event emission (`internal/pipeline/resume_test.go`)
  - Create a test pipeline with 3 steps, set up mock workspace directories for step 1
  - Call `ResumeFromStep()` with `fromStep="step-2"`
  - Capture emitted events via a test emitter
  - Assert that a "completed" event was emitted for step 1 with the correct StepID and Message

## Phase 2: Display Backend Hardening

- [X] Task 2.1: Verify BubbleTeaProgressDisplay handles synthetic events correctly [P]
  - Review `updateFromEvent()` in `internal/display/bubbletea_progress.go` — it already handles "completed" state
  - Add a test in `internal/display/bubbletea_progress_test.go` (or `bubbletea_model_test.go`) confirming that emitting a "completed" event for a registered step transitions it to `StateCompleted`

- [X] Task 2.2: Verify BasicProgressDisplay handles zero-duration completed events [P]
  - Review `EmitProgress()` in `internal/display/progress.go` — it formats `(0.0s, 0 tokens)` for zero values
  - Add a test in `internal/display/progress_test.go` confirming output is reasonable for synthetic events
  - If output reads awkwardly (e.g., "0.0s"), consider special-casing Message field to show "prior run" instead of duration

- [X] Task 2.3: Verify ProgressDisplay (fallback) handles synthetic events [P]
  - Review `EmitProgress()` in `internal/display/progress.go` (the `ProgressDisplay` struct)
  - It already handles "completed" events via step state update — confirm no changes needed

## Phase 3: Shared-Worktree Activity Fix (Secondary)

- [X] Task 3.1: Clear stale stepToolActivity on step completion in BubbleTeaProgressDisplay
  - In `updateFromEvent()` (`internal/display/bubbletea_progress.go`), the "completed" case already calls `delete(btpd.stepToolActivity, evt.StepID)` at line 233
  - Verify this is sufficient: when step A completes and step B (sharing the worktree) starts, step A's activity should already be cleared
  - If the issue is that both steps appear as running simultaneously (they shouldn't with sequential execution), investigate whether duplicate "started" events are emitted

- [X] Task 3.2: Add test for activity cleanup on shared worktrees
  - Create a test where two steps share a workspace ref
  - Emit "started" + "stream_activity" for step A
  - Emit "completed" for step A
  - Emit "started" for step B
  - Verify step A has no remaining `stepToolActivity` entry

## Phase 4: Testing & Validation

- [X] Task 4.1: Run existing test suite to verify no regressions
  - `go test ./internal/pipeline/...`
  - `go test ./internal/display/...`
  - `go test ./cmd/wave/commands/...`

- [X] Task 4.2: Run full test suite with race detector
  - `go test -race ./...`

- [X] Task 4.3: Manual verification documentation
  - Document test commands for manual verification:
    - `wave run <pipeline> --from-step <step> --mock` (auto/TUI mode)
    - `wave run <pipeline> --from-step <step> --mock --output text`
    - `wave run <pipeline> --from-step <step> --mock --output json`
