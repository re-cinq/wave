# Data Model: Enhanced Pipeline Progress Visualization

**Created**: 2026-02-03
**Feature**: 018-enhanced-progress
**Purpose**: Define data structures and relationships for enhanced progress visualization

## Core Entities

### Pipeline Execution Context

Represents the complete execution state and metadata for a running pipeline.

**Attributes**:
- `PipelineID`: Unique identifier for the pipeline instance
- `Name`: Human-readable pipeline name from manifest
- `Description`: Pipeline description from metadata
- `StartTime`: Execution start timestamp
- `EstimatedDuration`: Predicted total runtime based on historical data
- `TotalSteps`: Total number of steps in the pipeline
- `CurrentStepIndex`: Zero-based index of currently executing step
- `CompletedSteps`: List of successfully completed step IDs
- `FailedSteps`: List of failed step IDs with error details
- `State`: Overall pipeline state (pending, running, completed, failed, cancelled)
- `WorkspacePath`: Root workspace directory for this execution
- `ManifestPath`: Path to the wave.yaml manifest file

**Relationships**:
- Has many `StepStatus` entities
- Has one `ProgressContext`
- Has many `PerformanceMetric` entries

**Data Sources**:
- Pipeline definition from manifest
- Execution state from `internal/pipeline/executor.go`
- Historical data from SQLite state store

### Step Status

Tracks the detailed execution state and timing for individual pipeline steps.

**Attributes**:
- `StepID`: Unique step identifier within pipeline
- `PipelineID`: Reference to parent pipeline execution
- `PersonaName`: Name of the persona executing this step
- `State`: Current step state (pending, running, completed, failed, retrying)
- `StartTime`: Step execution start timestamp
- `EndTime`: Step completion timestamp (null if still running)
- `ElapsedMs`: Milliseconds elapsed since step start
- `TokensUsed`: Number of LLM tokens consumed
- `Artifacts`: List of output artifacts generated
- `ContractStatus`: Contract validation state
- `RetryCount`: Number of retry attempts
- `MaxRetries`: Maximum allowed retries from configuration
- `ErrorMessage`: Error details if step failed
- `Progress`: Optional progress percentage for long-running operations

**Relationships**:
- Belongs to `Pipeline Execution Context`
- Has many `SubTaskProgress` entries for detailed progress tracking

**Data Sources**:
- Events from `internal/event/emitter.go`
- State persistence in SQLite
- Contract validation results

### Progress Context

Contains calculated progress metrics and display state for the entire pipeline.

**Attributes**:
- `PipelineID`: Reference to pipeline execution
- `OverallProgress`: Percentage complete (0-100)
- `EstimatedTimeRemaining`: Predicted time to completion in seconds
- `CurrentStepDescription`: Human-readable description of current activity
- `TokenBurnRate`: Tokens consumed per minute
- `TotalTokensUsed`: Cumulative tokens across all steps
- `FilesModified`: Number of files changed during execution
- `ArtifactsGenerated`: Number of artifacts produced
- `LastUpdateTime`: Timestamp of last progress calculation
- `ProgressHistory`: Array of progress snapshots for trend analysis

**Relationships**:
- Belongs to `Pipeline Execution Context`
- Calculated from `Step Status` entities

**Calculations**:
- `OverallProgress`: (CompletedSteps / TotalSteps) * 100
- `EstimatedTimeRemaining`: Average step duration * remaining steps
- `TokenBurnRate`: TotalTokensUsed / elapsed minutes

### Display State

Manages the terminal display state and animation timing for progress visualization.

**Attributes**:
- `TerminalWidth`: Current terminal width in characters
- `TerminalHeight`: Current terminal height in lines
- `SupportsColor`: Boolean indicating ANSI color support
- `SupportsUnicode`: Boolean indicating Unicode character support
- `RefreshRate`: Display update frequency in milliseconds
- `AnimationFrame`: Current animation frame counter
- `LastRender`: Timestamp of last screen update
- `DashboardLayout`: Current panel layout configuration
- `VisiblePanels`: List of currently displayed panels

**Relationships**:
- Independent entity, manages display for any pipeline

**Data Sources**:
- Terminal capability detection via `golang.org/x/term`
- User preferences from manifest or environment variables

## Extended Event Schema

### Enhanced Event Structure

Extends the existing `Event` struct to support progress visualization without breaking compatibility.

**New Optional Fields**:
```go
type Event struct {
    // Existing fields...
    Timestamp  time.Time `json:"timestamp"`
    PipelineID string    `json:"pipeline_id"`
    StepID     string    `json:"step_id,omitempty"`
    State      string    `json:"state"`
    DurationMs int64     `json:"duration_ms"`
    Message    string    `json:"message,omitempty"`
    Persona    string    `json:"persona,omitempty"`
    Artifacts  []string  `json:"artifacts,omitempty"`
    TokensUsed int       `json:"tokens_used,omitempty"`

    // New progress fields (optional, backward compatible)
    Progress          *int     `json:"progress,omitempty"`           // 0-100 percentage
    SubTaskCurrent    *int     `json:"subtask_current,omitempty"`    // Current subtask number
    SubTaskTotal      *int     `json:"subtask_total,omitempty"`      // Total subtasks
    EstimatedETA      *int64   `json:"estimated_eta,omitempty"`      // Seconds remaining
    FilesProcessed    *int     `json:"files_processed,omitempty"`   // Files modified count
    ArtifactCount     *int     `json:"artifact_count,omitempty"`    // Artifacts generated
    WorkspacePath     string   `json:"workspace_path,omitempty"`    // Current workspace
    ContractProgress  string   `json:"contract_progress,omitempty"` // Contract validation state
}
```

### New Event Types

**Progress Events**:
- `"step_progress"`: Intermediate progress within a long-running step
- `"eta_updated"`: Estimated time remaining recalculated
- `"contract_validating"`: Contract validation in progress
- `"compaction_progress"`: Relay/compaction progress update

### Animation State

**Spinner State**:
- `CurrentFrame`: Index into spinner character array
- `LastUpdate`: Timestamp of last animation frame
- `SpinnerType`: Style of spinner animation

**Counter State**:
- `TargetValue`: Final value for animated counter
- `CurrentValue`: Current displayed value
- `IncrementRate`: Rate of change per second

## Validation Rules

### Data Integrity

**Pipeline Execution Context**:
- `PipelineID` must be unique per execution
- `TotalSteps` must be positive integer
- `CurrentStepIndex` must be within [0, TotalSteps-1]
- `StartTime` must be valid timestamp

**Step Status**:
- `ElapsedMs` must be non-negative
- `RetryCount` must not exceed `MaxRetries`
- `Progress` must be within [0, 100] if present
- `State` must be valid enum value

**Progress Context**:
- `OverallProgress` must be within [0, 100]
- `EstimatedTimeRemaining` must be non-negative
- `TokenBurnRate` calculated from actual usage

### State Transitions

**Valid Step State Transitions**:
```
pending → running → completed
pending → running → failed → retrying → running
pending → running → failed (if max retries exceeded)
```

**Valid Pipeline States**:
```
pending → running → completed
pending → running → failed
pending → running → cancelled
```

## Storage Schema

### SQLite Extensions

**New Tables**:

```sql
-- Progress snapshots for historical analysis
CREATE TABLE progress_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pipeline_id TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    overall_progress INTEGER NOT NULL,
    current_step_index INTEGER NOT NULL,
    estimated_eta INTEGER,
    tokens_used INTEGER NOT NULL,
    files_modified INTEGER DEFAULT 0,
    artifacts_generated INTEGER DEFAULT 0
);

-- Performance metrics for ETA calculation
CREATE TABLE step_performance (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    step_id TEXT NOT NULL,
    persona_name TEXT NOT NULL,
    duration_ms INTEGER NOT NULL,
    tokens_used INTEGER NOT NULL,
    execution_date DATE NOT NULL,
    pipeline_name TEXT NOT NULL
);

-- Display preferences
CREATE TABLE display_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Extended Event Log**:
```sql
-- Add progress columns to existing event_log table
ALTER TABLE event_log ADD COLUMN progress INTEGER;
ALTER TABLE event_log ADD COLUMN subtask_current INTEGER;
ALTER TABLE event_log ADD COLUMN subtask_total INTEGER;
ALTER TABLE event_log ADD COLUMN estimated_eta INTEGER;
ALTER TABLE event_log ADD COLUMN files_processed INTEGER;
ALTER TABLE event_log ADD COLUMN artifact_count INTEGER;
```

### Performance Considerations

**Indexing Strategy**:
- Index on `(pipeline_id, timestamp)` for progress queries
- Index on `(step_id, persona_name)` for performance lookups
- Index on `execution_date` for historical cleanup

**Data Retention**:
- Keep detailed progress data for 30 days
- Aggregate historical performance metrics monthly
- Configurable retention via manifest settings

## API Interfaces

### Progress Query Interface

```go
type ProgressRepository interface {
    GetPipelineProgress(pipelineID string) (*ProgressContext, error)
    GetStepStatus(pipelineID, stepID string) (*StepStatus, error)
    GetHistoricalPerformance(stepID, persona string) ([]PerformanceMetric, error)
    RecordProgressSnapshot(pipelineID string, snapshot ProgressSnapshot) error
    CalculateETA(pipelineID string) (time.Duration, error)
}
```

### Display Management Interface

```go
type DisplayManager interface {
    RenderDashboard(ctx *ProgressContext, steps []StepStatus) error
    UpdateProgress(event Event) error
    HandleTerminalResize(width, height int) error
    StartAnimation() error
    StopAnimation() error
}
```

### Event Enhancement Interface

```go
type ProgressEventEmitter interface {
    EmitProgress(stepID string, progress int, subtasks int, current int) error
    EmitETAUpdate(pipelineID string, eta time.Duration) error
    EmitContractProgress(stepID string, stage string) error
}
```

## Implementation Notes

### Backward Compatibility

**NDJSON Output**: All new fields are optional and marked with `omitempty` tags to maintain compatibility with existing consumers.

**Event Structure**: Existing event fields remain unchanged. New progress events use distinct event types to avoid conflicts.

**API Compatibility**: New interfaces extend rather than replace existing APIs.

### Performance Optimization

**Efficient Rendering**: Use double-buffering to prevent screen flicker during updates.

**Smart Updates**: Only recalculate complex metrics when underlying data changes.

**Background Processing**: Perform ETA calculations and historical analysis in background goroutines.

### Error Handling

**Graceful Degradation**: Fall back to basic progress display if enhanced features fail.

**Data Validation**: Validate all progress data before storage to prevent corruption.

**Recovery**: Detect and recover from interrupted pipeline executions.