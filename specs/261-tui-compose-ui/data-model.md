# Data Model: Pipeline Composition UI (#261)

**Branch**: `261-tui-compose-ui` | **Date**: 2026-03-07

## Domain Entities

### Sequence

An ordered list of pipeline references for sequential execution.

```go
// Package: internal/tui (or internal/compose if extracted)

// Sequence represents an ordered list of pipelines to execute in series.
type Sequence struct {
    Entries []SequenceEntry
}

// SequenceEntry is a single pipeline in a sequence.
type SequenceEntry struct {
    PipelineName string
    Pipeline     *pipeline.Pipeline // Loaded pipeline definition (for artifact inspection)
}
```

**Behavior**:
- `Add(name string, p *pipeline.Pipeline)` — append entry
- `Remove(index int)` — remove entry at index, re-validate adjacent flows
- `MoveUp(index int)` / `MoveDown(index int)` — reorder
- `Len() int` — number of entries
- `IsEmpty() bool` — true when no entries
- `IsSingle() bool` — true when exactly one entry (delegates to normal launch)

### ArtifactFlow

A directional mapping between adjacent pipelines' artifacts at a boundary.

```go
// ArtifactFlow describes the artifact compatibility at one boundary
// between pipeline N and pipeline N+1 in a sequence.
type ArtifactFlow struct {
    SourcePipeline string        // Pipeline N name
    TargetPipeline string        // Pipeline N+1 name
    Outputs        []ArtifactDef // Last step output_artifacts of pipeline N
    Inputs         []ArtifactRef // First step inject_artifacts of pipeline N+1
    Matches        []FlowMatch   // Resolved matches
}

// FlowMatch describes the match status of a single artifact flow.
type FlowMatch struct {
    OutputName string      // From source pipeline's last step
    InputName  string      // The inject_artifact name (ArtifactRef.Artifact)
    InputAs    string      // The "as" alias for injection
    Status     MatchStatus // compatible, missing, unmatched
    Optional   bool        // True if the input is optional
}

// MatchStatus indicates the result of matching an artifact.
type MatchStatus int

const (
    MatchCompatible MatchStatus = iota // Output name matches inject artifact name
    MatchMissing                       // Inject artifact expected but no matching output
    MatchUnmatched                     // Output produced but not consumed by next pipeline
)
```

**Source data** (from `internal/pipeline/types.go`):
- `ArtifactDef.Name` — output artifact name
- `ArtifactRef.Artifact` — inject artifact name (what to match against)
- `ArtifactRef.As` — alias name for injection
- `ArtifactRef.Optional` — whether missing input is a warning or error

### CompatibilityResult

Aggregated validation result across all boundaries in a sequence.

```go
// CompatibilityResult is the aggregated result of validating all
// artifact flows across a sequence.
type CompatibilityResult struct {
    Flows       []ArtifactFlow
    Status      CompatibilityStatus
    Diagnostics []string // Human-readable messages for each issue
}

// CompatibilityStatus indicates the overall sequence readiness.
type CompatibilityStatus int

const (
    CompatibilityValid   CompatibilityStatus = iota // All flows compatible
    CompatibilityWarning                             // Optional mismatches only
    CompatibilityError                               // Required inputs missing
)

// IsReady returns true if the sequence can be started without issues.
func (r CompatibilityResult) IsReady() bool {
    return r.Status == CompatibilityValid || r.Status == CompatibilityWarning
}
```

## TUI Models

### ComposeListModel (Left Pane)

```go
// ComposeListModel is the Bubble Tea model for the sequence builder list.
type ComposeListModel struct {
    width       int
    height      int
    focused     bool
    sequence    Sequence
    cursor      int
    picking     bool            // True when pipeline picker is active
    picker      *huh.Form       // Pipeline picker form (nil when not picking)
    pickerValue string          // Bound value for picker
    available   []PipelineInfo  // Available pipelines for the picker
    validation  CompatibilityResult
}
```

**Messages emitted**:
- `ComposeSequenceChangedMsg` — when sequence is modified (add/remove/reorder)
- `ComposeStartMsg` — when Enter is pressed with valid/acknowledged sequence
- `ComposeCancelMsg` — when Esc is pressed to exit compose mode
- `ComposeFocusDetailMsg` — when Enter is pressed on a specific boundary for detail view

### ComposeDetailModel (Right Pane)

```go
// ComposeDetailModel renders the artifact flow visualization in the right pane.
type ComposeDetailModel struct {
    width      int
    height     int
    focused    bool
    viewport   viewport.Model
    validation CompatibilityResult
    focusedIdx int // Which boundary is focused (-1 for overview)
}
```

### Compose Mode Integration

```go
// Added to ContentModel:
type ContentModel struct {
    // ... existing fields ...
    composing    bool              // True when compose mode is active
    composeList  *ComposeListModel
    composeDetail *ComposeDetailModel
}

// Added to StatusBarModel state:
type StatusBarModel struct {
    // ... existing fields ...
    composeActive bool // True when compose mode is active
}
```

### New Messages

```go
// ComposeActiveMsg signals the status bar to switch to compose mode hints.
type ComposeActiveMsg struct {
    Active bool
}

// ComposeSequenceChangedMsg signals that the sequence was modified.
type ComposeSequenceChangedMsg struct {
    Sequence Sequence
    Validation CompatibilityResult
}

// ComposeStartMsg signals that the user wants to start the sequence.
type ComposeStartMsg struct {
    Sequence Sequence
}

// ComposeCancelMsg signals that compose mode should close.
type ComposeCancelMsg struct{}
```

## Artifact Resolution Algorithm

```
func ValidateSequence(seq Sequence) CompatibilityResult:
    result = CompatibilityResult{Status: CompatibilityValid}
    
    for i = 0; i < len(seq.Entries) - 1; i++:
        source = seq.Entries[i].Pipeline
        target = seq.Entries[i+1].Pipeline
        
        // Get last step outputs
        outputs = source.Steps[len(source.Steps)-1].OutputArtifacts
        
        // Get first step inputs
        inputs = target.Steps[0].Memory.InjectArtifacts
        
        flow = ArtifactFlow{
            SourcePipeline: source.PipelineName(),
            TargetPipeline: target.PipelineName(),
            Outputs: outputs,
            Inputs: inputs,
        }
        
        // Match each input to an output
        for each input in inputs:
            found = false
            for each output in outputs:
                if output.Name == input.Artifact:
                    flow.Matches = append(flow.Matches, FlowMatch{
                        OutputName: output.Name,
                        InputName: input.Artifact,
                        InputAs: input.As,
                        Status: MatchCompatible,
                        Optional: input.Optional,
                    })
                    found = true
                    break
            if !found:
                status = MatchMissing
                if input.Optional:
                    result.Status = max(result.Status, CompatibilityWarning)
                else:
                    result.Status = CompatibilityError
                    result.Diagnostics = append(result.Diagnostics, 
                        fmt.Sprintf("%s → %s: missing required input '%s'",
                            source.PipelineName(), target.PipelineName(), input.Artifact))
                flow.Matches = append(flow.Matches, FlowMatch{...})
        
        // Mark unmatched outputs
        for each output in outputs:
            if not consumed by any input:
                flow.Matches = append(flow.Matches, FlowMatch{
                    OutputName: output.Name,
                    Status: MatchUnmatched,
                })
        
        result.Flows = append(result.Flows, flow)
    
    return result
```

## File Organization

New files:
- `internal/tui/compose.go` — `Sequence`, `ArtifactFlow`, `CompatibilityResult`, `ValidateSequence()`
- `internal/tui/compose_list.go` — `ComposeListModel` (left pane)
- `internal/tui/compose_detail.go` — `ComposeDetailModel` (right pane, artifact flow visualization)
- `internal/tui/compose_messages.go` — compose-mode-specific message types
- `internal/tui/compose_test.go` — unit tests for sequence validation and artifact matching
- `internal/tui/compose_list_test.go` — unit tests for compose list model
- `internal/tui/compose_detail_test.go` — unit tests for compose detail model
- `cmd/wave/commands/compose.go` — `wave compose` CLI command
- `cmd/wave/commands/compose_test.go` — CLI command tests

Modified files:
- `internal/tui/content.go` — compose mode integration (gate Tab, route `s` key, swap panes)
- `internal/tui/pipeline_messages.go` — add `stateComposing` to `DetailPaneState`
- `internal/tui/statusbar.go` — compose mode hint line
- `internal/tui/app.go` — forward `ComposeActiveMsg` to status bar
- `cmd/wave/commands/helpers.go` — register compose command (if command registration exists)
