# Tasks

## Phase 1: Remove Non-Functional Pause Feature

- [ ] Task 1.1: Remove `paused` field and `p` keybinding handler from `ProgressModel` in `internal/display/bubbletea_model.go`
- [ ] Task 1.2: Remove pause-related view logic (PAUSED banner, conditional status line) from `ProgressModel.View()` in `internal/display/bubbletea_model.go`
- [ ] Task 1.3: Remove "p=pause" text from Dashboard header in `internal/display/dashboard.go`
- [ ] Task 1.4: Fix tick behavior - when paused state is removed, the tick should always continue (remove the paused early-return in TickMsg handler)

## Phase 2: Wire Quit to Pipeline Cancellation

- [ ] Task 2.1: Add `cancelFunc context.CancelFunc` field to `BubbleTeaProgressDisplay` struct in `internal/display/bubbletea_progress.go`
- [ ] Task 2.2: Add `cancelFunc` parameter to `NewBubbleTeaProgressDisplay` constructor or provide a `SetCancelFunc` setter method
- [ ] Task 2.3: Pass `cancelFunc` to `ProgressModel` so the `q`/`ctrl+c` handler can call it before `tea.Quit`
- [ ] Task 2.4: Update `cmd/wave/commands/output.go` to accept and store the cancel function in `EmitterResult`
- [ ] Task 2.5: Update `cmd/wave/commands/run.go` to pass `execCancel` to the display through the emitter setup

## Phase 3: Testing and Validation

- [ ] Task 3.1: Update `internal/display/dashboard_test.go` to verify "p=pause" text is no longer present [P]
- [ ] Task 3.2: Run `go test ./internal/display/...` to verify all display tests pass [P]
- [ ] Task 3.3: Run `go test ./cmd/wave/...` to verify command tests pass [P]
- [ ] Task 3.4: Run full `go test ./...` to check for regressions

## Phase 4: Audit and Documentation

- [ ] Task 4.1: Audit remaining TUI keybindings (`q`, `ctrl+c`) to confirm they function correctly
- [ ] Task 4.2: Scan for any other UI elements that advertise non-functional features (e.g., ETA display, performance metrics)
- [ ] Task 4.3: File follow-up issues via `gh issue create` for any additional non-functional features discovered
