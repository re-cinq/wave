# Data Model: TUI Pipeline Detail Right Pane

**Date**: 2026-03-06 | **Feature**: #255

## Entities

### FocusPane (enum)

Tracks which pane in the content area currently has keyboard focus.

```go
type FocusPane int

const (
    FocusPaneLeft  FocusPane = iota // Pipeline list (default)
    FocusPaneRight                  // Detail view
)
```

- **Owner**: `ContentModel` — the parent of both `PipelineListModel` and `PipelineDetailModel`
- **Transitions**: `FocusPaneLeft → FocusPaneRight` (Enter on non-header, non-running item), `FocusPaneRight → FocusPaneLeft` (Esc)
- **Propagation**: `ContentModel` calls `SetFocused(bool)` on both child models when focus changes

### PipelineDetailModel

The Bubble Tea model for the right pane. Renders detail content based on the currently selected pipeline and manages scrolling when focused.

```go
type PipelineDetailModel struct {
    width    int
    height   int
    focused  bool
    viewport viewport.Model

    // Current selection state
    selectedName string
    selectedKind itemKind
    selectedRunID string

    // Data for current selection
    availableDetail *AvailableDetail
    finishedDetail  *FinishedDetail
    branchDeleted   bool
    loading         bool
    errorMsg        string

    // Data provider
    provider DetailDataProvider
}
```

- **Lifecycle**: Created by `ContentModel` constructor. Receives data via messages.
- **Key methods**: `Init()`, `Update(tea.Msg)`, `View()`, `SetSize(w, h)`, `SetFocused(bool)`
- **Selection flow**: Receives `PipelineSelectedMsg` → dispatches async fetch → receives `DetailDataMsg` → re-renders

### AvailableDetail

Data projection for rendering an available pipeline's configuration. Derived from parsing the full pipeline YAML via `DetailDataProvider`.

```go
type AvailableDetail struct {
    Name         string
    Description  string
    Category     string
    StepCount    int
    Steps        []StepSummary
    InputSource  string
    InputExample string
    Artifacts    []string  // Output artifact names across all steps
    Skills       []string  // Required skill names
    Tools        []string  // Required tool names
}

type StepSummary struct {
    ID      string
    Persona string
}
```

- **Source**: Parsed from pipeline YAML file via `DetailDataProvider.FetchAvailableDetail(name)`
- **Scope**: Extends `PipelineInfo` with step-level detail (step IDs + personas) and dependency info

### FinishedDetail

Data projection for rendering a finished pipeline's execution summary. Derived from state store queries.

```go
type FinishedDetail struct {
    RunID        string
    Name         string
    Status       string   // "completed", "failed", "cancelled"
    Duration     time.Duration
    BranchName   string
    StartedAt    time.Time
    CompletedAt  time.Time
    ErrorMessage string   // Non-empty for failed runs
    FailedStep   string   // Step ID that failed (empty if none)
    Steps        []StepResult
    Artifacts    []ArtifactInfo
}

type StepResult struct {
    ID       string
    Status   string        // "completed", "failed", "skipped", "pending"
    Duration time.Duration
    Persona  string
}

type ArtifactInfo struct {
    Name string
    Path string
    Type string
}
```

- **Source**: Composed from multiple state store queries:
  - `store.GetRun(runID)` → run record (status, timestamps, error)
  - `store.GetPerformanceMetrics(runID, "")` → step-level timing and status
  - `store.GetArtifacts(runID, "")` → artifacts produced
- **Note**: `FailedStep` is derived from performance metrics where `Success == false`

### DetailDataProvider (interface)

Interface for fetching detailed pipeline data. Separate from `PipelineDataProvider` to maintain single responsibility (list-level vs detail-level).

```go
type DetailDataProvider interface {
    FetchAvailableDetail(name string) (*AvailableDetail, error)
    FetchFinishedDetail(runID string) (*FinishedDetail, error)
}
```

- **Implementation**: `DefaultDetailDataProvider` wraps `state.StateStore` and pipeline directory path
- **FetchAvailableDetail**: Finds pipeline YAML by name in pipelines dir, parses to `pipeline.Pipeline`, maps to `AvailableDetail`
- **FetchFinishedDetail**: Queries run record, performance metrics, and artifacts from state store, composes into `FinishedDetail`
- **Testing**: Mock implementation for unit tests

### DetailDataMsg

Async message carrying fetched detail data.

```go
type DetailDataMsg struct {
    AvailableDetail *AvailableDetail
    FinishedDetail  *FinishedDetail
    Err             error
}
```

### FocusChangedMsg

Message emitted by `ContentModel` when focus transitions between panes. Consumed by `StatusBarModel` to update key hints.

```go
type FocusChangedMsg struct {
    Pane FocusPane
}
```

### PipelineSelectedMsg (extended)

Existing message extended with `Name` and `Kind` fields.

```go
type PipelineSelectedMsg struct {
    RunID         string
    BranchName    string
    BranchDeleted bool
    Name          string   // NEW: pipeline name for all item types
    Kind          itemKind // NEW: item type (running/finished/available/section header)
}
```

- **Backward compatible**: Existing consumers (header) that only read `RunID`, `BranchName`, `BranchDeleted` are unaffected
- **Emitter change**: `PipelineListModel.emitSelectionMsg()` populates `Name` and `Kind` for all item types

## Data Flow

```
PipelineListModel (cursor move)
  → PipelineSelectedMsg{Name, Kind, RunID, ...}
    → ContentModel.Update()
      → async: provider.FetchAvailableDetail(name) or FetchFinishedDetail(runID)
        → DetailDataMsg{AvailableDetail/FinishedDetail}
          → PipelineDetailModel.Update() → re-render
    → HeaderModel.Update() (existing — branch override)
    → FocusChangedMsg (on Enter/Esc)
      → StatusBarModel.Update() → update key hints
```

## State Transitions

```
                     Enter (on available/finished item)
  FocusPaneLeft  ──────────────────────────────────────→  FocusPaneRight
       ↑                                                       │
       │                    Esc                                │
       └───────────────────────────────────────────────────────┘
```

Focus does NOT change when:
- Enter is pressed on a section header (toggles collapse)
- Enter is pressed on a running pipeline item (informational only)
- Any key is pressed while filtering (filter owns the input)
