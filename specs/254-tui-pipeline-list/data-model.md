# Data Model: TUI Pipeline List Left Pane

**Feature**: #254 — TUI Pipeline List Left Pane  
**Date**: 2026-03-05

## Entities

### PipelineDataProvider (Interface)

Follows the `MetadataProvider` pattern from `header_provider.go`.

```go
// PipelineDataProvider fetches pipeline data for the list component.
// Decoupled from state store for testability.
type PipelineDataProvider interface {
    FetchRunningPipelines() ([]RunningPipeline, error)
    FetchFinishedPipelines(limit int) ([]FinishedPipeline, error)
    FetchAvailablePipelines() ([]PipelineInfo, error)
}
```

### RunningPipeline (Value Object)

TUI-specific view of a running pipeline. Derived from `state.RunRecord`.

```go
type RunningPipeline struct {
    RunID      string
    Name       string
    BranchName string
    StartedAt  time.Time
}
```

### FinishedPipeline (Value Object)

TUI-specific view of a finished pipeline. Derived from `state.RunRecord`.

```go
type FinishedPipeline struct {
    RunID       string
    Name        string
    BranchName  string
    Status      string    // "completed", "failed", "cancelled"
    StartedAt   time.Time
    CompletedAt time.Time
    Duration    time.Duration
}
```

### PipelineInfo (Existing — `pipelines.go`)

Already defined in `internal/tui/pipelines.go`. Reused for the Available section.

```go
type PipelineInfo struct {
    Name         string
    Description  string
    StepCount    int
    InputExample string
    Release      bool
    Category     string
}
```

### PipelineListModel (Bubble Tea Model)

The left pane model implementing `Init()`, `Update()`, `View()`.

```go
type PipelineListModel struct {
    width    int
    height   int
    provider PipelineDataProvider

    // Section data
    running   []RunningPipeline
    finished  []FinishedPipeline
    available []PipelineInfo

    // Navigation state
    cursor     int              // index into navigable items
    navigable  []navigableItem  // flattened list of headers + items

    // Filter state
    filtering   bool
    filterInput textinput.Model
    filterQuery string

    // Section collapse state
    collapsed [3]bool // [Running, Finished, Available]

    // Focus state
    focused bool
}
```

### navigableItem (Internal Type)

Discriminated union for the flat navigation list.

```go
type itemKind int

const (
    itemKindSectionHeader itemKind = iota
    itemKindRunning
    itemKindFinished
    itemKindAvailable
)

type navigableItem struct {
    kind         itemKind
    sectionIndex int    // 0=Running, 1=Finished, 2=Available
    dataIndex    int    // index into section's data slice (-1 for headers)
    label        string // display text
}
```

### Messages (Bubble Tea)

```go
// PipelineDataMsg carries refreshed pipeline data from the provider.
type PipelineDataMsg struct {
    Running   []RunningPipeline
    Finished  []FinishedPipeline
    Available []PipelineInfo
    Err       error
}

// PipelineRefreshTickMsg triggers periodic data refresh.
type PipelineRefreshTickMsg struct{}
```

`PipelineSelectedMsg` — reused from `header_messages.go` (no changes needed).

## Data Flow

```
                    ┌──────────────┐
                    │  StateStore  │ (SQLite, read-only)
                    └──────┬───────┘
                           │
                    ┌──────▼───────────────┐
                    │ PipelineDataProvider  │
                    │  (DefaultPipeline     │
                    │   DataProvider)       │
                    └──────┬───────────────┘
                           │ FetchRunning/Finished/Available
                    ┌──────▼───────────────┐
                    │  PipelineListModel   │
                    │  - sections data     │
                    │  - navigation state  │
                    │  - filter state      │
                    └──────┬───────────────┘
                           │ PipelineSelectedMsg
                    ┌──────▼───────────────┐
                    │   ContentModel       │
                    │  (left + right pane) │
                    └──────┬───────────────┘
                           │ tea.Cmd(PipelineSelectedMsg)
                    ┌──────▼───────────────┐
                    │     AppModel         │
                    │  (routes to header)  │
                    └──────────────────────┘
```

## Polling Sequence

1. `Init()` → batch: `fetchPipelineData` (initial load) + `refreshTick()` (5s timer)
2. Timer fires `PipelineRefreshTickMsg`
3. `Update()` returns `tea.Batch(fetchPipelineData, refreshTick())`
4. `fetchPipelineData` calls all three provider methods, returns `PipelineDataMsg`
5. `Update(PipelineDataMsg)` updates section data, rebuilds navigable items
6. If cursor was on a pipeline item, re-emit `PipelineSelectedMsg` if the item still exists
