# Implementation Plan: Wave Ops Commands

**Feature**: 016-wave-ops-commands
**Created**: 2026-02-02
**Status**: Draft

## 1. Architecture Overview

The Wave Ops Commands feature adds operational CLI commands for pipeline management, monitoring, and maintenance. These commands integrate with the existing Wave architecture:

```
cmd/wave/
  commands/
    status.go      # NEW - Pipeline status monitoring
    logs.go        # NEW - Pipeline log viewing
    cancel.go      # NEW - Pipeline cancellation
    artifacts.go   # NEW - Artifact inspection/export
    clean.go       # EXISTING - Enhanced with age-based cleanup
    list.go        # EXISTING - Already implemented

internal/
  state/
    store.go       # EXTEND - Add run tracking, token aggregation, cancellation flags
    schema.sql     # EXTEND - Add event_log table, artifacts tracking
  pipeline/
    executor.go    # EXTEND - Add cancellation check between steps
```

### Design Principles

1. **Read-only by default**: Status, logs, artifacts commands never modify state
2. **Non-blocking queries**: Operations should not block running pipelines
3. **Consistent output**: All commands support `--format json` for scripting
4. **Graceful degradation**: Commands work even if state database is unavailable

## 2. Component Design

### 2.1 Status Command (`wave status`)

**Purpose**: Display pipeline execution status and progress.

**File**: `cmd/wave/commands/status.go`

```go
type StatusOptions struct {
    All       bool   // Show all recent pipelines
    RunID     string // Specific run to show
    Watch     bool   // Continuously refresh
    Format    string // table, json
}
```

**Subcommands**:
- `wave status` - Show currently running pipeline (if any)
- `wave status --all` - Show table of recent pipelines
- `wave status <run-id>` - Show detailed status for specific run

**Output (table format)**:
```
Pipeline     Status      Step          Elapsed    Tokens
debug        running     investigate   2m 34s     45k
hello-world  completed   -             1m 12s     12k
```

**Output (JSON format)**:
```json
{
  "pipelines": [
    {
      "id": "debug-20260202-143022",
      "pipeline": "debug",
      "status": "running",
      "current_step": "investigate",
      "elapsed_seconds": 154,
      "total_tokens": 45000,
      "steps": [...]
    }
  ]
}
```

**Integration Points**:
- `internal/state.StateStore.GetPipelineState()` - Existing
- `internal/state.StateStore.GetStepStates()` - Existing
- `internal/state.StateStore.ListRecentPipelines()` - Existing
- NEW: `internal/state.StateStore.GetRunningPipeline()` - Find active run

### 2.2 Logs Command (`wave logs`)

**Purpose**: View execution logs and adapter output.

**File**: `cmd/wave/commands/logs.go`

```go
type LogsOptions struct {
    RunID    string // Specific run (default: most recent)
    Step     string // Filter by step ID
    Errors   bool   // Only show errors
    Follow   bool   // Stream logs in real-time
    Lines    int    // Number of lines (default: 100)
    Format   string // text, json
}
```

**Subcommands**:
- `wave logs` - Show logs from most recent run
- `wave logs <run-id>` - Show logs for specific run
- `wave logs --step investigate` - Filter to specific step
- `wave logs --errors` - Only errors and failures
- `wave logs --follow` - Stream logs from running pipeline

**Log Sources** (priority order):
1. `.wave/traces/trace-*.log` - Audit logs with tool calls
2. Event stream from `internal/event.EventEmitter`
3. Step workspace stdout files (if persisted)

**Output (text format)**:
```
[14:30:22] started    debug input="fix bug" steps=3
[14:30:22] running    investigate (investigator)
[14:32:45] completed  investigate (investigator) 143.2s 45k tokens
[14:32:46] running    plan (planner)
```

**Output (JSON format)**:
```json
{
  "logs": [
    {
      "timestamp": "2026-02-02T14:30:22Z",
      "pipeline_id": "debug",
      "step_id": "investigate",
      "state": "running",
      "message": "Starting investigator persona",
      "tokens_used": 0
    }
  ]
}
```

**Integration Points**:
- `internal/audit.TraceLogger` - Read trace files
- `internal/event.Event` - Log entry structure
- `internal/state.Store` - Query log events from state database

### 2.3 Clean Command (`wave clean`) - Enhancement

**Purpose**: Enhanced cleanup with age-based retention.

**File**: `cmd/wave/commands/clean.go` (existing)

**New Options**:
```go
type CleanOptions struct {
    Pipeline string // EXISTING
    All      bool   // EXISTING
    Force    bool   // EXISTING
    KeepLast int    // EXISTING
    DryRun   bool   // EXISTING
    // NEW options:
    OlderThan string // e.g., "7d", "24h"
    Status    string // Filter by status: completed, failed
}
```

**New Behaviors**:
- `wave clean --older-than 7d` - Remove runs older than 7 days
- `wave clean --status failed` - Clean only failed pipeline runs
- `wave clean --older-than 24h --status completed` - Combine filters

**Integration Points**:
- `internal/workspace.ListWorkspacesSortedByTime()` - Existing
- NEW: `internal/state.StateStore.ListPipelinesByAge()` - Query by age
- NEW: `internal/state.StateStore.DeletePipelineState()` - Remove state records

### 2.4 List Command (`wave list`) - Already Implemented

**File**: `cmd/wave/commands/list.go` (existing)

The list command is already implemented with:
- `wave list pipelines` - List available pipelines
- `wave list personas` - List configured personas
- `wave list adapters` - List adapters with availability
- `--format json` support

No changes needed for this feature.

### 2.5 Cancel Command (`wave cancel`)

**Purpose**: Stop a running pipeline gracefully.

**File**: `cmd/wave/commands/cancel.go`

```go
type CancelOptions struct {
    RunID string // Specific run to cancel (default: current)
    Force bool   // Interrupt immediately vs wait for step
}
```

**Subcommands**:
- `wave cancel` - Cancel currently running pipeline
- `wave cancel <run-id>` - Cancel specific run
- `wave cancel --force` - Interrupt immediately

**Cancellation Flow**:
1. Set cancellation flag in state database
2. Executor checks flag between steps (graceful)
3. With `--force`: Send SIGTERM to adapter process group

**Integration Points**:
- `internal/state.StateStore` - NEW: `SetCancellationFlag()`
- `internal/pipeline.DefaultPipelineExecutor` - Check cancellation between steps
- `internal/adapter.AdapterRunner` - Process group termination (existing)

### 2.6 Artifacts Command (`wave artifacts`)

**Purpose**: List and export pipeline artifacts.

**File**: `cmd/wave/commands/artifacts.go`

```go
type ArtifactsOptions struct {
    RunID  string // Specific run (default: most recent)
    Step   string // Filter by step
    Export string // Export directory path
    Format string // table, json
}
```

**Subcommands**:
- `wave artifacts` - List artifacts from most recent run
- `wave artifacts <run-id>` - List artifacts from specific run
- `wave artifacts --step investigate` - Filter by step
- `wave artifacts --export ./output` - Copy artifacts to directory

**Output (table format)**:
```
Step          Artifact          Type      Size      Path
investigate   analysis.md       markdown  4.2KB     .wave/workspaces/debug/investigate/analysis.md
plan          plan.md           markdown  2.1KB     .wave/workspaces/debug/plan/plan.md
```

**Integration Points**:
- `internal/pipeline.PipelineExecution.ArtifactPaths` - Artifact registry
- `internal/workspace` - Read artifact files
- NEW: `internal/state.StateStore.GetArtifacts()` - Persist artifact metadata

## 3. Data Model Extensions

### 3.1 Schema Changes (`internal/state/schema.sql`)

```sql
-- Track individual pipeline runs (not just pipeline definitions)
CREATE TABLE IF NOT EXISTS pipeline_run (
    run_id TEXT PRIMARY KEY,
    pipeline_name TEXT NOT NULL,
    status TEXT NOT NULL,           -- pending, running, completed, failed, cancelled
    input TEXT,
    current_step TEXT,
    total_tokens INTEGER DEFAULT 0,
    started_at INTEGER NOT NULL,
    completed_at INTEGER,
    cancelled_at INTEGER,
    error_message TEXT,
    FOREIGN KEY (pipeline_name) REFERENCES pipeline_state(pipeline_name)
);

CREATE INDEX IF NOT EXISTS idx_run_pipeline ON pipeline_run(pipeline_name);
CREATE INDEX IF NOT EXISTS idx_run_status ON pipeline_run(status);
CREATE INDEX IF NOT EXISTS idx_run_started ON pipeline_run(started_at);

-- Track artifacts produced by runs
CREATE TABLE IF NOT EXISTS artifact (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    type TEXT,
    size_bytes INTEGER,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifact_run ON artifact(run_id);

-- Store event log entries for logs command
CREATE TABLE IF NOT EXISTS event_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    step_id TEXT,
    state TEXT NOT NULL,
    persona TEXT,
    message TEXT,
    tokens_used INTEGER,
    duration_ms INTEGER,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_event_run ON event_log(run_id);
CREATE INDEX IF NOT EXISTS idx_event_timestamp ON event_log(timestamp);

-- Cancellation flags for graceful shutdown
CREATE TABLE IF NOT EXISTS cancellation (
    run_id TEXT PRIMARY KEY,
    requested_at INTEGER NOT NULL,
    force BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);
```

### 3.2 StateStore Interface Extensions

```go
// internal/state/store.go - Interface additions

type StateStore interface {
    // Existing methods
    SavePipelineState(id string, status string, input string) error
    SaveStepState(pipelineID string, stepID string, state StepState, err string) error
    GetPipelineState(id string) (*PipelineStateRecord, error)
    GetStepStates(pipelineID string) ([]StepStateRecord, error)
    ListRecentPipelines(limit int) ([]PipelineStateRecord, error)
    Close() error

    // NEW: Run tracking
    CreateRun(pipelineName string, input string) (string, error)
    UpdateRunStatus(runID string, status string, currentStep string, tokens int) error
    GetRun(runID string) (*RunRecord, error)
    GetRunningRun() (*RunRecord, error)
    ListRuns(opts ListRunsOptions) ([]RunRecord, error)
    DeleteRun(runID string) error

    // NEW: Event logging
    LogEvent(runID string, event Event) error
    GetEvents(runID string, opts EventQueryOptions) ([]Event, error)

    // NEW: Artifact tracking
    RegisterArtifact(runID string, stepID string, name string, path string, artifactType string) error
    GetArtifacts(runID string, stepID string) ([]ArtifactRecord, error)

    // NEW: Cancellation
    RequestCancellation(runID string, force bool) error
    CheckCancellation(runID string) (*CancellationRecord, error)
    ClearCancellation(runID string) error
}

type RunRecord struct {
    RunID        string
    PipelineName string
    Status       string
    Input        string
    CurrentStep  string
    TotalTokens  int
    StartedAt    time.Time
    CompletedAt  *time.Time
    CancelledAt  *time.Time
    ErrorMessage string
}

type ListRunsOptions struct {
    PipelineName string
    Status       string
    OlderThan    time.Duration
    Limit        int
}

type EventQueryOptions struct {
    StepID     string
    ErrorsOnly bool
    Limit      int
    Offset     int
}

type ArtifactRecord struct {
    RunID     string
    StepID    string
    Name      string
    Path      string
    Type      string
    SizeBytes int64
    CreatedAt time.Time
}

type CancellationRecord struct {
    RunID       string
    RequestedAt time.Time
    Force       bool
}
```

## 4. Integration Points

### 4.1 Executor Integration

The `DefaultPipelineExecutor` needs modifications to:

1. **Create run records**: Call `CreateRun()` at pipeline start
2. **Update run status**: Call `UpdateRunStatus()` after each step
3. **Log events to DB**: Call `LogEvent()` for each emitted event
4. **Register artifacts**: Call `RegisterArtifact()` when writing outputs
5. **Check cancellation**: Call `CheckCancellation()` between steps

```go
// internal/pipeline/executor.go modifications

func (e *DefaultPipelineExecutor) Execute(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error {
    // Create run record
    runID := fmt.Sprintf("%s-%s", p.Metadata.Name, time.Now().Format("20060102-150405"))
    if e.store != nil {
        e.store.CreateRun(p.Metadata.Name, input)
    }

    // ... existing code ...

    for _, step := range sortedSteps {
        // Check cancellation before each step
        if e.store != nil {
            if cancel, _ := e.store.CheckCancellation(runID); cancel != nil {
                e.store.UpdateRunStatus(runID, "cancelled", step.ID, totalTokens)
                return fmt.Errorf("pipeline cancelled by user")
            }
        }

        // Execute step
        if err := e.executeStep(ctx, execution, step); err != nil {
            // ... existing error handling ...
        }

        // Update run status after step
        if e.store != nil {
            e.store.UpdateRunStatus(runID, "running", step.ID, totalTokens)
        }
    }

    // ... rest of function ...
}
```

### 4.2 Event Emitter Integration

Modify the emitter to persist events to the database:

```go
// internal/event/emitter.go - Add DB persistence option

type PersistentEmitter struct {
    *NDJSONEmitter
    store state.StateStore
    runID string
}

func (e *PersistentEmitter) Emit(event Event) {
    // Write to stdout (existing behavior)
    e.NDJSONEmitter.Emit(event)

    // Persist to database
    if e.store != nil {
        e.store.LogEvent(e.runID, event)
    }
}
```

### 4.3 Command Registration

Add new commands to `cmd/wave/main.go`:

```go
func init() {
    // ... existing commands ...

    rootCmd.AddCommand(commands.NewStatusCmd())
    rootCmd.AddCommand(commands.NewLogsCmd())
    rootCmd.AddCommand(commands.NewCancelCmd())
    rootCmd.AddCommand(commands.NewArtifactsCmd())
}
```

## 5. Testing Strategy

### 5.1 Unit Tests

Each command gets comprehensive unit tests:

**`cmd/wave/commands/status_test.go`**:
- Test status display for running pipeline
- Test status display for completed pipeline
- Test `--all` flag with multiple pipelines
- Test JSON output format
- Test when no pipelines exist

**`cmd/wave/commands/logs_test.go`**:
- Test log retrieval from database
- Test `--step` filter
- Test `--errors` filter
- Test JSON output format
- Test log streaming (mock)

**`cmd/wave/commands/cancel_test.go`**:
- Test graceful cancellation flag
- Test `--force` cancellation
- Test cancel when no pipeline running
- Test cancel non-existent run ID

**`cmd/wave/commands/artifacts_test.go`**:
- Test artifact listing
- Test `--step` filter
- Test `--export` directory creation
- Test export file copying
- Test when no artifacts exist

**`internal/state/store_test.go`** (extensions):
- Test run creation and retrieval
- Test event logging and queries
- Test artifact registration
- Test cancellation flag operations
- Test cleanup of old runs

### 5.2 Integration Tests

Test end-to-end command workflows:

```go
// cmd/wave/commands/integration_test.go

func TestStatusWhilePipelineRunning(t *testing.T) {
    // Start a pipeline in background
    // Run `wave status`
    // Verify running state displayed
}

func TestLogsAfterPipelineComplete(t *testing.T) {
    // Run a simple pipeline
    // Run `wave logs`
    // Verify all events captured
}

func TestCancelRunningPipeline(t *testing.T) {
    // Start a long-running pipeline
    // Run `wave cancel`
    // Verify pipeline stops gracefully
}

func TestArtifactExport(t *testing.T) {
    // Run pipeline that produces artifacts
    // Run `wave artifacts --export ./test-output`
    // Verify files copied correctly
}
```

### 5.3 Race Detection

All tests run with `-race` flag to detect concurrent access issues:

```bash
go test -race ./cmd/wave/commands/...
go test -race ./internal/state/...
go test -race ./internal/pipeline/...
```

## 6. Implementation Phases

### Phase 1: Data Model & State Extensions (1-2 days)

**Tasks**:
1. Extend `schema.sql` with new tables
2. Add new `StateStore` interface methods
3. Implement run tracking methods
4. Implement event logging methods
5. Implement artifact tracking methods
6. Add unit tests for new store methods

**Files**:
- `internal/state/schema.sql`
- `internal/state/store.go`
- `internal/state/store_test.go`

### Phase 2: Status Command (1 day)

**Tasks**:
1. Create `status.go` command file
2. Implement basic status display
3. Add `--all` flag for history
4. Add JSON output support
5. Add unit tests

**Files**:
- `cmd/wave/commands/status.go`
- `cmd/wave/commands/status_test.go`

### Phase 3: Logs Command (1-2 days)

**Tasks**:
1. Create `logs.go` command file
2. Implement log retrieval from DB
3. Add trace file reading fallback
4. Add `--step`, `--errors` filters
5. Add `--follow` streaming (basic)
6. Add JSON output support
7. Add unit tests

**Files**:
- `cmd/wave/commands/logs.go`
- `cmd/wave/commands/logs_test.go`
- `internal/state/store.go` (extend with log queries)

### Phase 4: Cancel Command (1 day)

**Tasks**:
1. Create `cancel.go` command file
2. Implement cancellation flag setting
3. Integrate cancellation check in executor
4. Add `--force` flag
5. Add unit tests

**Files**:
- `cmd/wave/commands/cancel.go`
- `cmd/wave/commands/cancel_test.go`
- `internal/pipeline/executor.go` (modify)

### Phase 5: Artifacts Command (1 day)

**Tasks**:
1. Create `artifacts.go` command file
2. Implement artifact listing
3. Implement `--export` directory copy
4. Add JSON output support
5. Add unit tests

**Files**:
- `cmd/wave/commands/artifacts.go`
- `cmd/wave/commands/artifacts_test.go`

### Phase 6: Clean Command Enhancement (0.5 days)

**Tasks**:
1. Add `--older-than` flag
2. Add `--status` filter flag
3. Integrate with new state methods
4. Update tests

**Files**:
- `cmd/wave/commands/clean.go` (modify)
- `cmd/wave/commands/clean_test.go` (extend)

### Phase 7: Executor Integration (1 day)

**Tasks**:
1. Generate run IDs for each execution
2. Call run tracking methods from executor
3. Persist events to database
4. Register artifacts when written
5. Check cancellation between steps
6. Update integration tests

**Files**:
- `internal/pipeline/executor.go` (modify)
- `internal/event/emitter.go` (modify)

### Phase 8: Integration Testing & Polish (1 day)

**Tasks**:
1. Write end-to-end integration tests
2. Run all tests with race detector
3. Test CLI help text and examples
4. Document commands in CLAUDE.md
5. Final code review

**Files**:
- `cmd/wave/commands/integration_test.go`
- `CLAUDE.md` (documentation update)

## 7. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Database locking during concurrent access | Use WAL mode (already configured), busy timeout |
| Large log files slow down queries | Add pagination with `LIMIT`/`OFFSET`, index timestamps |
| Force cancel leaves orphan processes | Use process groups (already implemented in adapter) |
| Artifact export fails mid-copy | Use atomic copy (temp file + rename) |
| State DB corruption | Graceful degradation - commands work without DB |

## 8. Open Questions Resolution

From the spec:

1. **Should `wave logs` support log levels (debug/info/error)?**
   - **Decision**: Yes, via `--errors` flag for filtering. Full level support can be added later.

2. **Should `wave clean` have a scheduled/automatic mode?**
   - **Decision**: Not in this phase. Users can set up cron jobs with `wave clean --older-than 7d`.

3. **Should `wave cancel` send SIGTERM or use a cancellation token?**
   - **Decision**: Both. Default is cancellation token (graceful), `--force` sends SIGTERM to process group.

## 9. Success Criteria

- [ ] All 6 user stories from spec pass acceptance scenarios
- [ ] Performance: `wave status` < 100ms, `wave logs` streaming < 500ms latency
- [ ] All tests pass with `-race` flag
- [ ] Commands work when state DB is unavailable (graceful degradation)
- [ ] JSON output is valid and scriptable for all commands
- [ ] CLI help text includes examples for each command
