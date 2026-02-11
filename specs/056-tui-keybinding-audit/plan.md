# Implementation Plan: TUI Keybinding Audit (#56)

## Objective

Remove the non-functional `p`/pause keybinding from Wave's TUI and audit all other keybindings to ensure no UI element suggests functionality that does not exist. The `q`/quit keybinding needs to be wired to actually cancel the pipeline execution.

## Approach

**Remove pause, fix quit.** Implementing true pause/resume would require significant architectural changes (pausing subprocess adapters, handling mid-step interruption, preserving state). This is out of scope for a UX cleanup issue. Instead:

1. Remove the `p` keybinding handler and `paused` state from `ProgressModel`
2. Remove all "p=pause" text from both the BubbleTea and Dashboard displays
3. Wire the `q`/`ctrl+c` keybinding to actually cancel the pipeline execution context
4. File a follow-up issue if real pause/resume is desired in the future

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/display/bubbletea_model.go` | modify | Remove `paused` field, remove `p` case in Update, remove paused view logic, simplify status line |
| `internal/display/bubbletea_progress.go` | modify | Accept a `context.CancelFunc` to propagate quit signals; store it for use when `q` is pressed |
| `internal/display/dashboard.go` | modify | Remove "p=pause" from header project info text |
| `internal/display/dashboard_test.go` | modify | Update any tests that reference pause text |
| `cmd/wave/commands/output.go` | modify | Pass the execution cancel function to BubbleTeaProgressDisplay |
| `cmd/wave/commands/run.go` | modify | Thread the cancel function through to the display |

## Architecture Decisions

### Decision 1: Remove pause rather than implement it

**Rationale**: The issue says "either fix or remove." Implementing pause requires:
- Signaling subprocess adapters to pause (not supported by Claude Code CLI)
- Handling partial step state during pause
- Resume semantics for contracts/validation

This is substantial new functionality better suited for a dedicated feature issue.

### Decision 2: Wire quit to context cancellation

**Rationale**: The `q` keybinding currently only exits the TUI, leaving the pipeline subprocess running. This is misleading - when a user presses `q`, they expect the pipeline to stop. We should cancel the execution context, which will propagate to the subprocess adapter.

### Decision 3: Keep status line with quit-only hint

Replace `"Press: p=pause q=quit"` with `"Press: q=quit"` to keep the quit hint visible.

## Risks

1. **Context cancellation race**: Canceling the context while the executor is mid-step could leave workspace in inconsistent state. Mitigated by existing cleanup mechanisms in the workspace manager.
2. **BubbleTea lifecycle**: Need to ensure the program exits cleanly after context cancellation without hanging. The existing `Clear()` method already handles cursor restoration.
3. **Test coverage**: Dashboard test may reference the pause text string and need updates.

## Testing Strategy

1. **Unit tests**: Update existing bubbletea_model tests (if any) to remove pause-related assertions
2. **Dashboard tests**: Update `dashboard_test.go` to verify pause text is removed
3. **Integration**: Manual verification that pressing `q` during pipeline execution actually stops the pipeline
4. **Regression**: Run `go test ./internal/display/...` and `go test ./cmd/wave/...` to catch any breakage
