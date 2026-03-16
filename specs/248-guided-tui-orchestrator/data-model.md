# Data Model: Guided TUI Orchestrator

**Feature Branch**: `248-guided-tui-orchestrator`
**Date**: 2026-03-16

## Entities

### GuidedFlowState

Controls the guided workflow state machine. Layered above existing `ViewType` system.

```go
// GuidedFlowPhase represents the current phase of the guided workflow.
type GuidedFlowPhase int

const (
    GuidedPhaseHealth    GuidedFlowPhase = iota // Infrastructure health checks running
    GuidedPhaseProposals                        // Showing pipeline proposals (ViewSuggest)
    GuidedPhaseFleet                            // Monitoring active runs (ViewPipelines)
    GuidedPhaseAttached                         // Attached to live output of a running pipeline
)

// GuidedFlowState manages the guided workflow lifecycle.
// When non-nil on ContentModel, it overrides startup view and Tab behavior.
type GuidedFlowState struct {
    Phase           GuidedFlowPhase
    HealthComplete  bool   // All infrastructure checks finished
    HasErrors       bool   // At least one health check returned error
    UserConfirmed   bool   // User chose to continue despite health errors
    TransitionTimer bool   // Auto-transition timer is running
}
```

**Location**: `internal/tui/guided_flow.go` (new file)

**Relationships**:
- Referenced by `ContentModel` as `guidedFlow *GuidedFlowState`
- Reads from `HealthListModel.checks` to determine completion
- Controls `cycleView()` behavior (Tab toggle vs 8-view cycle)

### HealthCompletionTracker

Tracks async health check completion for auto-transition.

```go
// Embedded in HealthListModel or tracked by ContentModel.
// Fields:
//   totalChecks    int  // len(checks)
//   completedCount int  // checks with status != HealthCheckChecking
```

**Note**: This is not a separate struct — it's additional state on the existing `HealthListModel`. When `completedCount == totalChecks`, emit `HealthAllCompleteMsg`.

### Archive Divider (navigable item extension)

```go
const (
    itemKindPipelineName itemKind = iota
    itemKindRunning
    itemKindFinished
    itemKindAvailable
    itemKindDivider  // NEW: visual separator between active and archived runs
)
```

**Location**: `internal/tui/pipeline_list.go` (extend existing enum)

### DAGPreview (rendering model)

Not a persistent entity — computed at render time from `SuggestProposedPipeline`.

```go
// DAGNode represents a pipeline in the execution DAG visualization.
type DAGNode struct {
    Name      string
    Artifacts []string // Output artifacts flowing to next node
}

// RenderDAG produces a text-based DAG visualization for the detail pane.
// For sequences: [A] ──artifacts──→ [B] ──artifacts──→ [C]
// For parallels: [A] ╶╶ [B] (concurrent)
func RenderDAG(proposal SuggestProposedPipeline, pipelinesDir string) string
```

**Location**: `internal/tui/suggest_dag.go` (new file)

### FleetRun Extensions

Extend existing `RunningPipeline` and `FinishedPipeline` with sequence grouping.

```go
// RunningPipeline (existing in pipeline_messages.go) — add field:
type RunningPipeline struct {
    RunID          string
    Name           string
    Input          string
    BranchName     string
    StartedAt      time.Time
    CurrentStep    string
    SequenceGroup  string    // NEW: compose group run ID, empty for standalone runs
}

// FinishedPipeline (existing in pipeline_messages.go) — add field:
type FinishedPipeline struct {
    RunID          string
    Name           string
    Input          string
    BranchName     string
    Status         string
    Duration       time.Duration
    StartedAt      time.Time
    SequenceGroup  string    // NEW: compose group run ID
}
```

**Location**: `internal/tui/pipeline_messages.go` (modify existing structs)

## Messages (New)

```go
// HealthAllCompleteMsg signals that all infrastructure health checks have resolved.
type HealthAllCompleteMsg struct {
    HasErrors bool // true if any check returned HealthCheckErr
}

// HealthTransitionMsg triggers the auto-transition from health to proposals.
type HealthTransitionMsg struct{}

// HealthContinueMsg signals the user chose to continue despite health errors.
type HealthContinueMsg struct{}

// SuggestModifyMsg requests input modification for a proposal before launch.
type SuggestModifyMsg struct {
    Pipeline SuggestProposedPipeline
}
```

**Location**: `internal/tui/guided_messages.go` (new file)

## View/State Mapping

| GuidedFlowPhase | ViewType      | Tab Target     | Auto-Transition |
|-----------------|---------------|----------------|-----------------|
| HealthPhase     | ViewHealth    | ViewSuggest    | → Proposals on all checks complete |
| Proposals       | ViewSuggest   | ViewPipelines  | → Fleet on pipeline launch |
| Fleet           | ViewPipelines | ViewSuggest    | → Attached on Enter |
| Attached        | ViewPipelines | (blocked)      | → Fleet on Esc |

## File Impact Summary

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/tui/guided_flow.go` | NEW | GuidedFlowState, phase transitions, Tab override |
| `internal/tui/guided_messages.go` | NEW | Health completion, transition, and modify messages |
| `internal/tui/suggest_dag.go` | NEW | DAG preview rendering for sequence/parallel proposals |
| `internal/tui/content.go` | MODIFY | Add guidedFlow field, override Init/cycleView, handle new messages, number-key navigation |
| `internal/tui/app.go` | MODIFY | Accept guided mode flag, pass to ContentModel |
| `internal/tui/views.go` | MODIFY | No changes needed (ViewType enum already complete) |
| `internal/tui/health_list.go` | MODIFY | Track completion count, emit HealthAllCompleteMsg |
| `internal/tui/suggest_list.go` | MODIFY | Add `m` key for modify, `s` key for skip/dismiss |
| `internal/tui/suggest_detail.go` | MODIFY | Integrate DAG preview rendering |
| `internal/tui/pipeline_list.go` | MODIFY | Add itemKindDivider, archive layout, sequence grouping |
| `internal/tui/pipeline_messages.go` | MODIFY | Add SequenceGroup to RunningPipeline/FinishedPipeline |
| `internal/tui/pipeline_provider.go` | MODIFY | Expose SequenceGroup from state store |
| `internal/tui/statusbar.go` | MODIFY | Update hints for guided mode views |
| `cmd/wave/main.go` | MODIFY | Pass guided=true flag when no subcommand |
