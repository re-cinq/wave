# Data Model: Guided Workflow Orchestrator TUI

## Entities

### GuidedState (new enum)

Controls the guided flow state machine. Lives as a field on `ContentModel`.

```go
// GuidedState represents the current phase in the guided workflow.
type GuidedState int

const (
    GuidedStateHealth    GuidedState = iota // Startup health check phase
    GuidedStateProposals                    // Pipeline proposals view (maps to ViewSuggest)
    GuidedStateFleet                        // Fleet/pipeline monitoring (maps to ViewPipelines)
    GuidedStateAttached                     // Fullscreen live output for a single pipeline
)
```

### ContentModel extensions (modified)

```go
// New fields on ContentModel
type ContentModel struct {
    // ... existing fields ...

    // Guided workflow mode
    guidedMode    bool         // True when activated from root command (no subcommand)
    guidedState   GuidedState  // Current phase in guided flow
    healthDone    bool         // True when all health checks have completed
}
```

### LaunchDependencies extension (modified)

```go
type LaunchDependencies struct {
    // ... existing fields ...
    GuidedMode bool // When true, TUI starts in guided health-first flow
}
```

### HealthPhaseCompleteMsg (new message)

```go
// HealthPhaseCompleteMsg is emitted when all health checks have completed.
type HealthPhaseCompleteMsg struct {
    AllPassed bool   // True if no critical failures
    Summary   string // "6/6 checks passed" or "5/6 passed, 1 warning"
}
```

### HealthListModel extensions (modified)

```go
type HealthListModel struct {
    // ... existing fields ...
    totalChecks    int  // Total number of checks to run
    completedCount int  // Number of checks that have finished (any status)
}
```

Completion detection added to `Update()`:

```go
case HealthCheckResultMsg:
    // ... existing update logic ...
    m.completedCount++
    if m.completedCount >= m.totalChecks {
        return m, func() tea.Msg {
            return HealthPhaseCompleteMsg{
                AllPassed: m.allPassed(),
                Summary:   m.buildSummary(),
            }
        }
    }
```

### SuggestListModel extensions (modified)

```go
type SuggestListModel struct {
    // ... existing fields ...
    skipped       map[int]bool        // Skipped proposal indices
    inputOverlay  *textinput.Model    // Active input modification overlay (nil when inactive)
    overlayTarget int                 // Index of proposal being modified
    healthSummary string              // Health summary line for header display
}
```

### SuggestDetailModel extensions (modified)

```go
type SuggestDetailModel struct {
    // ... existing fields ...
    // DAG preview rendering added to View() — no new fields needed.
    // Uses existing p.Type, p.Sequence fields on SuggestProposedPipeline.
}
```

### PipelineListModel extensions (modified)

```go
type PipelineListModel struct {
    // ... existing fields ...
    showArchiveDivider bool           // When true, insert divider between running and finished
    sequenceGroups     map[string][]string // groupRunID -> member runIDs for visual grouping
}
```

### New navigable item kind

```go
const (
    // ... existing kinds ...
    itemKindArchiveDivider  // Visual divider between active and archived runs
)
```

## Message Flow

```
Startup (guided mode):
  ContentModel.Init() → starts health checks (HealthListModel.Init())

  HealthCheckResultMsg × 6 → HealthListModel tracks completion count

  All checks done → HealthPhaseCompleteMsg

  ContentModel handles HealthPhaseCompleteMsg:
    → Sets guidedState = GuidedStateProposals
    → Initializes SuggestListModel (triggers FetchSuggestions)
    → Emits ViewChangedMsg{View: ViewSuggest}

Tab key (guided mode):
  GuidedStateProposals → GuidedStateFleet (ViewPipelines)
  GuidedStateFleet → GuidedStateProposals (ViewSuggest)

Enter on proposal:
  SuggestLaunchMsg → ContentModel launches pipeline
  → guidedState = GuidedStateFleet
  → ViewChangedMsg{View: ViewPipelines}

Enter on running pipeline (fleet view):
  → guidedState = GuidedStateAttached
  → Activates LiveOutputModel (existing flow)

Esc from attached:
  → guidedState = GuidedStateFleet
  → Returns to fleet view (existing flow)
```

## View ↔ State Mapping

| GuidedState | ViewType | Primary Model | Notes |
|---|---|---|---|
| Health | ViewHealth | HealthListModel | Auto-created on startup |
| Proposals | ViewSuggest | SuggestListModel + SuggestDetailModel | Enhanced with health summary, DAG preview |
| Fleet | ViewPipelines | PipelineListModel + PipelineDetailModel | Enhanced with archive divider |
| Attached | ViewPipelines | LiveOutputModel (in PipelineDetailModel) | Existing mechanism |
