# Data Model: TUI Pipeline Launch Flow

**Date**: 2026-03-06
**Feature**: #256 — TUI Pipeline Launch Flow

## New Types

### LaunchDependencies

Lightweight struct carrying pre-loaded dependencies for pipeline execution.
Passed through `NewAppModel()` → `NewContentModel()` → `PipelineLauncher`.

```go
// LaunchDependencies holds the dependencies needed to launch pipelines from the TUI.
// Passed at TUI construction time; executor infrastructure is created on demand.
type LaunchDependencies struct {
    Manifest     *manifest.Manifest
    Store        state.StateStore
    PipelinesDir string
}
```

**Lifecycle**: Created once at `RunTUI()`, immutable thereafter.

### LaunchConfig

Data assembled from the argument form submission. Maps to the existing `Selection` type.

```go
// LaunchConfig holds the user's pipeline launch configuration from the argument form.
type LaunchConfig struct {
    PipelineName  string
    Input         string
    ModelOverride string
    Flags         []string  // e.g., "--verbose", "--debug", "--dry-run"
    DryRun        bool      // Extracted from Flags for convenience
}
```

**Lifecycle**: Created when form completes, consumed by `PipelineLauncher.Launch()`, then discarded.

### PipelineLauncher

Component managing TUI-launched pipeline lifecycles. Field on `ContentModel`.

```go
// PipelineLauncher manages pipeline execution from the TUI.
// It constructs executors on demand and tracks cancel functions for running pipelines.
type PipelineLauncher struct {
    deps       LaunchDependencies
    cancelFns  map[string]context.CancelFunc  // runID → cancel function
    mu         sync.Mutex                      // protects cancelFns
}
```

**Methods**:
- `NewPipelineLauncher(deps LaunchDependencies) *PipelineLauncher`
- `Launch(config LaunchConfig) tea.Cmd` — returns batched cmd (immediate + executor)
- `Cancel(runID string)` — invokes cancel function for a specific run
- `CancelAll()` — cancels all running pipelines (called on TUI exit)

**Lifecycle**: Created with `ContentModel`, lives for TUI session duration.

### DetailPaneState

Explicit state enum for the right pane's rendering mode.

```go
type DetailPaneState int

const (
    stateEmpty           DetailPaneState = iota  // No selection
    stateLoading                                 // Fetching data
    stateAvailableDetail                         // Available pipeline config
    stateFinishedDetail                          // Finished pipeline results
    stateRunningInfo                             // Running pipeline info
    stateConfiguring                             // Argument form active
    stateLaunching                               // Brief "Starting..." indicator
    stateError                                   // Launch error display
)
```

**Lifecycle**: Transitions driven by user actions and message handling in `PipelineDetailModel`.

## New Messages

### LaunchRequestMsg

Emitted by `PipelineDetailModel` when the huh.Form completes.

```go
// LaunchRequestMsg is emitted when the argument form is submitted.
type LaunchRequestMsg struct {
    Config LaunchConfig
}
```

**Flow**: `PipelineDetailModel` → `ContentModel.Update()` → `launcher.Launch(config)`

### PipelineLaunchedMsg

Emitted immediately when a pipeline executor starts.

```go
// PipelineLaunchedMsg signals that a pipeline launch was initiated.
type PipelineLaunchedMsg struct {
    RunID        string
    PipelineName string
}
```

**Flow**: `PipelineLauncher` → `ContentModel` → `PipelineListModel` (insert into Running section)

### PipelineLaunchResultMsg

Emitted when the executor goroutine completes (success or failure).

```go
// PipelineLaunchResultMsg signals that a launched pipeline has finished execution.
type PipelineLaunchResultMsg struct {
    RunID string
    Err   error   // nil on success
}
```

**Flow**: `PipelineLauncher` goroutine → `ContentModel` → cleanup cancel map

### LaunchErrorMsg

Emitted when the pipeline fails to start (before executor runs).

```go
// LaunchErrorMsg signals a pre-execution failure (adapter resolution, manifest loading, etc.).
type LaunchErrorMsg struct {
    PipelineName string
    Err          error
}
```

**Flow**: `PipelineLauncher.Launch()` → `ContentModel` → `PipelineDetailModel` (show error)

## Modified Types

### PipelineDetailModel

Add fields for form state and launch config:

```go
type PipelineDetailModel struct {
    // ... existing fields ...
    
    // Launch form state
    paneState    DetailPaneState
    launchForm   *huh.Form      // Non-nil when configuring
    launchInput  string          // Bound to form input field
    launchModel  string          // Bound to form model override field
    launchFlags  []string        // Bound to form flag multi-select
    launchError  string          // Error message for stateError
}
```

### ContentModel

Add launcher field and route new messages:

```go
type ContentModel struct {
    // ... existing fields ...
    launcher *PipelineLauncher
}
```

### AppModel

- Modify `q`-to-quit check to gate on `m.content.focus == FocusPaneLeft`
- Call `m.content.CancelAll()` before `tea.Quit` on exit

### StatusBarModel

- Add form-context hints when right pane is in configuring state
- New hint text: `"Tab: next  Shift+Tab: prev  Enter: launch  Esc: cancel"`

### RunTUI()

- Change signature to accept `LaunchDependencies` (or its constituent parts)
- Pass dependencies through to `NewAppModel()`

### NewAppModel()

- Add `LaunchDependencies` parameter
- Pass to `NewContentModel()`

### NewContentModel()

- Add `LaunchDependencies` parameter
- Create `PipelineLauncher` with dependencies

## State Transitions

```
Left pane focused, available item selected
    ↓ Enter
PipelineDetailModel: stateAvailableDetail → stateConfiguring
    (form created, focus → right pane)
    ↓ Form submitted (huh.StateCompleted)
PipelineDetailModel: stateConfiguring → stateLaunching
    (emit LaunchRequestMsg)
    ↓ PipelineLaunchedMsg received
PipelineDetailModel: stateLaunching → stateRunningInfo
PipelineListModel: insert synthetic RunningPipeline at top
    (focus → left pane, cursor → new running entry)
    ↓ PipelineLaunchResultMsg received
PipelineLauncher: remove cancel function
    (normal refresh tick updates Running/Finished sections)

Alternative paths:
    ↓ Esc during form
PipelineDetailModel: stateConfiguring → stateAvailableDetail
    (focus → left pane)
    ↓ LaunchErrorMsg
PipelineDetailModel: stateLaunching → stateError
    (show error, focus → left pane)
    ↓ Esc from error / navigate away
PipelineDetailModel: stateError → stateAvailableDetail
```

## Interaction Flow Diagram

```
User: Enter on available pipeline
  → ContentModel: detect available + Enter → transition focus right
  → PipelineDetailModel: create huh.Form, set stateConfiguring
  
User: Fill form, press Enter on submit
  → huh.Form: StateCompleted
  → PipelineDetailModel: extract values, emit LaunchRequestMsg
  → ContentModel: handle LaunchRequestMsg, call launcher.Launch()
  → PipelineLauncher: create context, resolve adapter, build executor
    → Immediate cmd: return PipelineLaunchedMsg
    → Background cmd: run executor, return PipelineLaunchResultMsg
  → ContentModel: handle PipelineLaunchedMsg
    → PipelineListModel: insert RunningPipeline, move cursor
    → Focus → left pane
    
User: Press 'c' on running pipeline
  → PipelineListModel: forward to ContentModel
  → ContentModel: call launcher.Cancel(runID)
  → context cancelled → executor stops

User: q / Ctrl+C
  → AppModel: call content.CancelAll()
  → All cancel functions invoked
  → tea.Quit
```
