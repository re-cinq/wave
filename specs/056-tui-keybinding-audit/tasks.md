# Tasks

## Phase 1: Remove Non-Functional Pause Feature

- [X] Task 1.1: Remove `paused` field and `p` keybinding handler from `ProgressModel` in `internal/display/bubbletea_model.go`
- [X] Task 1.2: Remove pause-related view logic (PAUSED banner, conditional status line) from `ProgressModel.View()` in `internal/display/bubbletea_model.go`
- [X] Task 1.3: Remove "p=pause" text from Dashboard header in `internal/display/dashboard.go`
- [X] Task 1.4: Fix tick behavior - when paused state is removed, the tick should always continue (remove the paused early-return in TickMsg handler)

## Phase 2: Wire Quit to Pipeline Cancellation

- [X] Task 2.1: Add `cancelFunc context.CancelFunc` field to `ProgressModel` in `internal/display/bubbletea_model.go`
- [X] Task 2.2: Add `SetCancelFunc` setter method on `BubbleTeaProgressDisplay` in `internal/display/bubbletea_progress.go`
- [X] Task 2.3: Pass `cancelFunc` to `ProgressModel` so the `q`/`ctrl+c` handler can call it before `tea.Quit`
- [X] Task 2.4: Update `cmd/wave/commands/output.go` to store `BubbleTeaProgressDisplay` reference in `EmitterResult`
- [X] Task 2.5: Update `cmd/wave/commands/run.go` to pass `execCancel` to the display via `SetCancelFunc`

## Phase 3: Testing and Validation

- [X] Task 3.1: Update `internal/display/dashboard_test.go` to verify "p=pause" text is no longer present [P] — no test changes needed, no existing tests referenced pause text
- [X] Task 3.2: Run `go test ./internal/display/...` to verify all display tests pass [P]
- [X] Task 3.3: Run `go test ./cmd/wave/...` to verify command tests pass [P]
- [X] Task 3.4: Run full `go test -race ./...` to check for regressions

## Phase 4: Audit and Documentation

- [X] Task 4.1: Audit remaining TUI keybindings (`q`, `ctrl+c`) to confirm they function correctly
- [X] Task 4.2: Scan for any other UI elements that advertise non-functional features (e.g., ETA display, performance metrics)
- [X] Task 4.3: File follow-up issue (#61) for dead code: unused ETA calculation and performance metrics
