# Data Model: TUI Finished Pipeline Interactions

**Date**: 2026-03-06
**Feature**: #258 — TUI Finished Pipeline Interactions

## New Types

### ChatSessionEndedMsg

Message sent when the Claude Code subprocess exits via `tea.Exec()` callback. Triggers data refresh and state restoration.

```go
// ChatSessionEndedMsg signals that an interactive chat session has ended.
// Triggers data refresh (re-fetch finished detail, git state) to reflect changes
// the user may have made during the session.
type ChatSessionEndedMsg struct {
    Err error // Non-nil if the subprocess failed to start or exited with error
}
```

**Flow**: `tea.Exec()` callback → `ChatSessionEndedMsg` → `PipelineDetailModel.Update()` → re-fetch `FinishedDetail` + emit `GitRefreshTickMsg`.

### BranchCheckoutMsg

Message carrying the result of a `git checkout <branch>` attempt.

```go
// BranchCheckoutMsg signals the result of a branch checkout attempt.
type BranchCheckoutMsg struct {
    BranchName string
    Success    bool
    Err        error // Non-nil on failure (uncommitted changes, branch missing, etc.)
}
```

**Flow**: `tea.Cmd` closure running `git checkout` → `BranchCheckoutMsg` → `PipelineDetailModel.Update()` → header refresh on success, error display on failure.

### DiffViewEndedMsg

Message sent when the diff pager subprocess exits.

```go
// DiffViewEndedMsg signals that the diff pager has exited.
type DiffViewEndedMsg struct {
    Err error // Non-nil if the diff command failed
}
```

**Flow**: `tea.Exec()` callback → `DiffViewEndedMsg` → `PipelineDetailModel.Update()` → TUI resume.

### FinishedDetailActiveMsg

Status bar hint switching signal, following the `FormActiveMsg` and `LiveOutputActiveMsg` pattern.

```go
// FinishedDetailActiveMsg signals the status bar to switch to finished detail hints.
type FinishedDetailActiveMsg struct {
    Active bool
}
```

**Flow**: `PipelineDetailModel`/`ContentModel` → `AppModel.Update()` → `StatusBarModel.Update()`.

## Modified Types

### FinishedDetail (extended)

Add workspace path and branch existence fields to the existing struct.

```go
type FinishedDetail struct {
    // ... existing fields ...
    RunID        string
    Name         string
    Status       string
    Duration     time.Duration
    BranchName   string
    StartedAt    time.Time
    CompletedAt  time.Time
    ErrorMessage string
    FailedStep   string
    Steps        []StepResult
    Artifacts    []ArtifactInfo

    // NEW fields
    WorkspacePath string // Filesystem path to pipeline workspace, empty if deleted
    BranchDeleted bool   // True if the branch no longer exists (checked via git rev-parse)
}
```

**`WorkspacePath` derivation**: Computed in `FetchFinishedDetail()`:
1. Construct path: `.wave/workspaces/<RunID>/__wt_<sanitized_branch>/`
2. If `BranchName` is empty, try `filepath.Glob(".wave/workspaces/<RunID>/__wt_*")` for any worktree
3. Validate with `os.Stat()` — if not found, leave empty

**`BranchDeleted` derivation**: Computed in `FetchFinishedDetail()`:
1. If `BranchName` is empty → not applicable (no branch to delete)
2. Run `exec.Command("git", "rev-parse", "--verify", branch).Run()`
3. Non-zero exit → `BranchDeleted = true`

### PipelineDetailModel (extended)

Add action error state for transient error display.

```go
type PipelineDetailModel struct {
    // ... existing fields ...

    // Action error state (NEW)
    actionError string // Transient error message for failed actions, cleared on next keypress
}
```

### StatusBarModel (extended)

Add finished detail active state for hint switching.

```go
type StatusBarModel struct {
    width                int
    contextLabel         string
    focusPane            FocusPane
    formActive           bool
    liveOutputActive     bool
    finishedDetailActive bool  // NEW: finished detail mode active
}
```

### ContentModel

Add handling for new message types:
- `ChatSessionEndedMsg` → forward to detail model
- `BranchCheckoutMsg` → forward to detail model, emit `GitRefreshTickMsg` on success
- `DiffViewEndedMsg` → forward to detail model
- `FinishedDetailActiveMsg` → emit for status bar consumption
- Enter on `stateFinishedDetail` with right pane focused → launch chat session via `tea.Exec()`

### AppModel

Forward `FinishedDetailActiveMsg` to status bar (alongside existing `FocusChangedMsg`, `FormActiveMsg`, `LiveOutputActiveMsg`).

## State Transitions

```
Left pane focused, finished item selected
    ↓ Enter
ContentModel: focus right, emit FocusChangedMsg{Right}
    → PipelineSelectedMsg triggers FetchFinishedDetail()
    → Detail model receives DetailDataMsg, sets stateFinishedDetail
    → Emit FinishedDetailActiveMsg{Active: true}

Right pane focused, stateFinishedDetail
    ↓ Enter (chat)
    → Validate WorkspacePath exists
    → If empty: set actionError = "Workspace directory no longer exists"
    → If exists: tea.Exec(exec.Command("claude"), ChatSessionEndedMsg callback)
    → TUI suspends, Claude Code gets terminal control

    ↓ ChatSessionEndedMsg
    → TUI resumes
    → Re-fetch finished detail (data refresh)
    → Emit GitRefreshTickMsg (header refresh)

    ↓ 'b' (checkout)
    → Validate BranchDeleted == false && BranchName != ""
    → If invalid: no-op (key handler gated)
    → Return tea.Cmd: git rev-parse --verify <branch>, then git checkout <branch>
    → BranchCheckoutMsg → success: emit GitRefreshTickMsg, clear actionError
    → BranchCheckoutMsg → failure: set actionError = error text

    ↓ 'd' (diff)
    → Validate BranchDeleted == false && BranchName != ""
    → If invalid: no-op (key handler gated)
    → tea.Exec(exec.Command("git", "diff", "main..."+branch), DiffViewEndedMsg callback)
    → TUI suspends, pager gets terminal control

    ↓ DiffViewEndedMsg
    → TUI resumes (no data refresh needed — diff is read-only)

    ↓ Esc
    → Focus left, emit FocusChangedMsg{Left}
    → Emit FinishedDetailActiveMsg{Active: false}
    → Clear actionError

    ↓ Any key when actionError != ""
    → Clear actionError (transient error dismissed)
```

## Key Handler Gating

| Key | Active When | Gated By |
|-----|------------|----------|
| Enter (chat) | `paneState == stateFinishedDetail && focused` | WorkspacePath existence (checked at action time) |
| `b` (checkout) | `paneState == stateFinishedDetail && focused` | `!BranchDeleted && BranchName != ""` |
| `d` (diff) | `paneState == stateFinishedDetail && focused` | `!BranchDeleted && BranchName != ""` |
| Esc | `focused` | None (always active when focused) |

## Status Bar Hint Priority Chain

```
formActive && focusPaneRight
    → "Tab: next  Shift+Tab: prev  Enter: launch  Esc: cancel"
liveOutputActive && focusPaneRight
    → "v: verbose  d: debug  o: output-only  ↑↓: scroll  Esc: back"
finishedDetailActive && focusPaneRight          ← NEW
    → "[Enter] Chat  [b] Branch  [d] Diff  [Esc] Back"
focusPaneRight (generic)
    → "↑↓: scroll  Esc: back  q: quit  ctrl+c: exit"
focusPaneLeft
    → "↑↓: navigate  Enter: view  /: filter  q: quit  ctrl+c: exit"
```

## Rendering Changes

### renderFinishedDetail() — Action Hints Section

Current (from #255):
```
[Enter] Open chat  [b] Checkout branch  [d] View diff  [Esc] Back
```

Updated (this feature):
- `[b]` fainted when `branchDeleted || branchName == ""`
- `[d]` fainted when `branchDeleted || branchName == ""` ← NEW (C15 fix)
- `[Enter]` fainted when `workspacePath == ""` ← NEW
- When `actionError != ""`: render red error text instead of hints
