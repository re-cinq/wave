# Tasks

## Phase 1: Remove Pause Keybinding from BubbleTea Model

- [X] Task 1.1: Remove `paused` field from `ProgressModel` struct in `bubbletea_model.go`
- [X] Task 1.2: Remove `case "p":` handler from `Update()` method in `bubbletea_model.go`
- [X] Task 1.3: Remove pause-conditional branch in `TickMsg` handler (the `if m.paused` block) in `bubbletea_model.go`
- [X] Task 1.4: Simplify `View()` to remove the `if m.paused` conditional and show only `"Press: q=quit"` status line in `bubbletea_model.go`
- [X] Task 1.5: Update `NewProgressModel()` to remove `paused: false` initialization in `bubbletea_model.go`

## Phase 2: Remove Pause Reference from Dashboard

- [X] Task 2.1: Update help text in `renderHeader()` from `" Press: p=pause q=quit"` to `" Press: q=quit"` in `dashboard.go`

## Phase 3: Audit TUI for Other Non-Functional Features [P]

- [X] Task 3.1: Audit all keybinding handlers in `bubbletea_model.go` for unimplemented functionality
- [X] Task 3.2: Audit `dashboard.go` for UI elements that reference unimplemented features
- [X] Task 3.3: Audit `bubbletea_progress.go` for dead code or stub functionality
- [X] Task 3.4: Check `types.go` `DisplayConfig` fields for features that are defined but never used
- [X] Task 3.5: File separate GitHub issues for each non-functional feature discovered

## Phase 4: Testing

- [X] Task 4.1: Write unit tests for `ProgressModel.Update()` verifying `p` key is no longer handled
- [X] Task 4.2: Write unit test for `ProgressModel.View()` verifying status line shows only `q=quit`
- [X] Task 4.3: Run `go test ./internal/display/...` and fix any regressions [P]
- [X] Task 4.4: Run `go test ./...` full suite to catch any cross-package regressions

## Phase 5: Final Validation

- [X] Task 5.1: Verify no remaining references to `p=pause` in the TUI codebase
- [X] Task 5.2: Verify the build compiles cleanly (`go build ./...`)
