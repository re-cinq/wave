# Data Model: Pipeline Step Visibility

**Feature Branch**: `100-pipeline-step-visibility`
**Date**: 2026-02-13

## Entity Changes

### Modified Entity: `PipelineContext` (display/types.go)

The `PipelineContext` struct is the data transfer object between the pipeline execution layer and the rendering layer. It needs one new field to support persona display for all steps.

#### Current Fields (Relevant Subset)

```go
type PipelineContext struct {
    // Step tracking
    StepStatuses   map[string]ProgressState  // stepID → state
    StepOrder      []string                   // ordered step IDs
    StepDurations  map[string]int64           // stepID → duration in ms

    // Current execution state (single running step)
    CurrentStepID   string
    CurrentPersona  string
    CurrentStepName string
}
```

#### New Field

```go
type PipelineContext struct {
    // ... existing fields ...

    // Step persona mapping (NEW)
    StepPersonas map[string]string  // stepID → persona name
}
```

**Justification**: FR-002 requires all steps (not just the running step) to display their persona name. Currently only `CurrentPersona` is available for the active step. The `StepPersonas` map provides persona names for completed, pending, failed, skipped, and cancelled steps.

**Population Points**:
1. `BubbleTeaProgressDisplay.toPipelineContext()` — builds map from `btpd.steps` (each `StepStatus` has a `Persona` field)
2. `ProgressDisplay.toPipelineContext()` — builds map from `pd.steps`
3. `CreatePipelineContext()` — accepts step personas at construction time

### Existing Entity: `StepStatus` (display/progress.go)

No changes needed. Already contains all required fields:

```go
type StepStatus struct {
    StepID        string
    Name          string
    State         ProgressState  // not_started, running, completed, failed, skipped, cancelled
    Persona       string          // ← already stored per step
    StartTime     time.Time
    EndTime       *time.Time
    ElapsedMs     int64
}
```

### Existing Entity: `ProgressState` (display/types.go)

No changes needed. Already defines all required states:

```go
const (
    StateNotStarted ProgressState = "not_started"
    StateRunning    ProgressState = "running"
    StateCompleted  ProgressState = "completed"
    StateFailed     ProgressState = "failed"
    StateSkipped    ProgressState = "skipped"
    StateCancelled  ProgressState = "cancelled"
)
```

### Rendering Entity: Step Display Line

A conceptual rendering entity (not a struct) representing one line in the step list:

```
<indicator> <step-name> (<persona-name>) [(<timing>)]
```

| Component | Source | Condition |
|-----------|--------|-----------|
| `indicator` | Computed from `StepStatuses[stepID]` | Always present |
| `step-name` | `StepOrder[i]` (stepID used as name) | Always present |
| `persona-name` | `StepPersonas[stepID]` | Always present (from AddStep registration) |
| `timing` | Live `time.Since(CurrentStepStart)` for running; `StepDurations[stepID]` for completed | Running or completed only |

## Data Flow Diagram

```
┌─────────────────────────────┐
│   Pipeline YAML             │
│   steps[].id, persona       │
└─────────┬───────────────────┘
          │
          ▼
┌─────────────────────────────┐
│   output.go:93              │
│   btpd.AddStep(id, id,      │
│     step.Persona)           │
└─────────┬───────────────────┘
          │
          ▼
┌─────────────────────────────┐
│   BubbleTeaProgressDisplay   │
│   steps map[string]*StepStatus│
│   stepOrder []string          │
│   stepDurations map[...]      │
└─────────┬───────────────────┘
          │ toPipelineContext()
          ▼
┌─────────────────────────────┐
│   PipelineContext            │
│   StepStatuses  map[→state]  │
│   StepOrder     []string     │
│   StepDurations map[→int64]  │
│   StepPersonas  map[→string] │ ← NEW
│   CurrentStepID              │
│   CurrentPersona             │
│   CurrentStepStart           │
└─────────┬───────────────────┘
          │ SendUpdate()
          ▼
┌─────────────────────────────┐
│   ProgressModel.View()       │
│   renderCurrentStep()        │
│                              │
│   For each stepID in         │
│   StepOrder:                 │
│     state ← StepStatuses     │
│     persona ← StepPersonas   │
│     duration ← StepDurations │
│     → render indicator       │
│     → render name (persona)  │
│     → render timing if any   │
└─────────────────────────────┘
```

## State Transition Diagram

```
                    ┌──────────┐
                    │ not_started│
                    │    ○      │
                    └─────┬────┘
                          │
              ┌───────────┼───────────┐
              ▼           ▼           ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │ running  │ │ skipped  │ │cancelled │
        │  ⠋ (spin)│ │    —     │ │    ⊛     │
        └────┬─────┘ └──────────┘ └──────────┘
             │
       ┌─────┼──────┐
       ▼     ▼      ▼
  ┌────────┐ ┌────────┐ ┌──────────┐
  │completed│ │ failed │ │cancelled │
  │   ✓    │ │   ✗    │ │    ⊛     │
  └────────┘ └────────┘ └──────────┘
```

Terminal states: completed, failed, skipped, cancelled.
Only `running` has a live timer. Only `completed` and `failed` show final duration.

## Impact Analysis

### Files Modified

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/display/types.go` | **Add field** | Add `StepPersonas map[string]string` to `PipelineContext` |
| `internal/display/bubbletea_model.go` | **Rewrite method** | Rewrite `renderCurrentStep()` to render all steps |
| `internal/display/bubbletea_progress.go` | **Add mapping** | Populate `StepPersonas` in `toPipelineContext()` |
| `internal/display/progress.go` | **Add mapping** | Populate `StepPersonas` in `toPipelineContext()` and `CreatePipelineContext()` |
| `internal/display/dashboard.go` | **Rewrite method** | Rewrite `renderStepStatusPanel()` to render all steps in order |

### Files Unchanged

| File | Reason |
|------|--------|
| `internal/pipeline/*` | No pipeline execution changes |
| `internal/event/*` | No event format changes |
| `cmd/wave/commands/output.go` | AddStep already passes persona correctly |
| `cmd/wave/commands/run.go` | No changes needed |
| `internal/display/animation.go` | Spinner unchanged |
| `internal/display/terminal.go` | Terminal detection unchanged |
| `internal/display/formatter.go` | Formatting utilities unchanged |
| `internal/display/metrics.go` | Performance metrics unchanged |

### Test Files to Add/Modify

| File | Change |
|------|--------|
| `internal/display/bubbletea_model_test.go` | Add tests for all-step rendering, persona display, all 6 states |
| `internal/display/types_test.go` | Add test for `StepPersonas` field |
| `internal/display/progress_test.go` | Verify `StepPersonas` populated in `toPipelineContext()` |
| `internal/display/dashboard_test.go` | Add test for all-step rendering in Dashboard |
