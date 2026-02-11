# Audit and resolve non-functional features: p/pause keybinding and related UX issues

**Issue**: [re-cinq/wave#56](https://github.com/re-cinq/wave/issues/56)
**Labels**: bug, ux, cleanup
**Author**: nextlevelshit

## Summary

Several features in Wave's TUI are currently non-functional or behave unexpectedly during pipeline execution. These need to be audited, and each one either fixed, hidden from the UI, or removed entirely.

## Non-Functional Features

### 1. `p` / Pause keybinding during pipeline execution

- **Current behavior**: Pressing `p` during a running pipeline does not pause execution as the keybinding suggests. The BubbleTea model toggles a `paused` boolean that only stops the UI tick timer - it does not pause the underlying pipeline execution (subprocess adapter continues running).
- **Desired outcome**: Either implement working pause/resume functionality, or remove/hide the keybinding so users are not misled.

### 2. Broader audit of non-functional keybindings and features

- **Current behavior**: There may be additional keybindings or UI elements that are wired up but non-functional.
- **Desired outcome**: Scan the TUI implementation for any other features that are advertised but not working, and file separate issues for each.

## Acceptance Criteria

- [ ] The `p`/pause keybinding either works correctly or is no longer visible/accessible in the TUI
- [ ] A scan of the TUI codebase has been performed to identify any other non-functional features
- [ ] Separate issues have been filed for any additional non-functional features discovered
- [ ] No keybindings or UI elements suggest functionality that does not exist

## Technical Analysis

### Current Pause Implementation

The pause feature exists in two display paths:

1. **BubbleTea model** (`internal/display/bubbletea_model.go`):
   - `ProgressModel.paused` field (line 14)
   - `p` keypress toggles `m.paused` (line 49)
   - When paused, tick timer stops (line 56) and status shows "PAUSED" (line 97-100)
   - **Problem**: Only pauses UI updates, not the pipeline execution itself

2. **Dashboard** (`internal/display/dashboard.go`):
   - Shows "Press: p=pause q=quit" text (line 98)
   - No actual keyboard handling (Dashboard is not a BubbleTea model)

### Other Keybindings Identified

- `q` / `ctrl+c` (bubbletea_model.go:45-47): Quits the BubbleTea program. This sets `m.quit = true` and calls `tea.Quit`, but does NOT cancel the pipeline execution context. The pipeline continues running in the background after the TUI exits.

### No Signal Propagation

Neither `p` (pause) nor `q` (quit) propagate signals back to the pipeline executor. The `BubbleTeaProgressDisplay` has no mechanism to signal the executor to pause or cancel. The executor uses a `context.Context` for cancellation, but the display does not have access to the cancel function.
