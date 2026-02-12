# Audit and resolve non-functional features: p/pause keybinding and related UX issues

**Issue**: [#56](https://github.com/re-cinq/wave/issues/56)
**Labels**: bug, ux, cleanup
**Author**: nextlevelshit

## Summary

Several features in Wave's TUI are currently non-functional or behave unexpectedly during pipeline execution. These need to be audited, and each one either fixed, hidden from the UI, or removed entirely.

## Non-Functional Features

### 1. `p` / Pause keybinding during pipeline execution
- **Current behavior**: Pressing `p` during a running pipeline toggles a `paused` bool in `ProgressModel` which stops the UI tick (animation updates) but does NOT pause actual pipeline execution. The pipeline continues running in the background while the UI appears frozen.
- **Desired outcome**: Either implement working pause/resume functionality, or remove/hide the keybinding so users are not misled.

### 2. Broader audit of non-functional keybindings and features
- **Current behavior**: There may be additional keybindings or UI elements that are wired up but non-functional.
- **Desired outcome**: Scan the TUI implementation for any other features that are advertised but not working, and file separate issues for each.

## Acceptance Criteria

- [ ] The `p`/pause keybinding either works correctly or is no longer visible/accessible in the TUI
- [ ] A scan of the TUI codebase has been performed to identify any other non-functional features
- [ ] Separate issues have been filed for any additional non-functional features discovered
- [ ] No keybindings or UI elements suggest functionality that does not exist

## Technical Context

### Affected Files
- `internal/display/bubbletea_model.go` - Contains the `paused` field and `p` keybinding handler
- `internal/display/dashboard.go` - Contains the `"Press: p=pause q=quit"` help text
- `internal/display/bubbletea_progress.go` - Owns `BubbleTeaProgressDisplay` which creates the model

### Current Pause Implementation (Non-functional)
The `ProgressModel` struct has a `paused bool` field. When `p` is pressed:
1. `m.paused` is toggled (`bubbletea_model.go:49`)
2. The status line changes to "PAUSED - Press 'p' to resume, 'q' to quit" (`bubbletea_model.go:97-100`)
3. The tick command is suppressed, stopping UI animation updates (`bubbletea_model.go:55-58`)
4. **However**: The pipeline executor has no awareness of this pause state - execution continues uninterrupted

The `dashboard.go` fallback renderer also shows "Press: p=pause q=quit" (`dashboard.go:98`) but has no pause handling at all.

### Pipeline Pause Infrastructure
The pipeline executor (`internal/pipeline/executor.go`) has a `Resume` method and the state store supports a `"paused"` status, but there is no mechanism to signal a running pipeline to pause. The `wave resume` command (`cmd/wave/commands/resume.go`) handles the `"paused"` state string but there is no corresponding `wave pause` command or runtime pause signal.
