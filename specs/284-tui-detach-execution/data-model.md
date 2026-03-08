# Data Model: Detach Pipeline Execution from TUI Process Lifecycle

**Feature Branch**: `284-tui-detach-execution`
**Date**: 2026-03-08

## Entities

### 1. Detached Run (Extension of `RunRecord`)

A pipeline execution running as an independent OS process, tracked by run ID in SQLite with PID metadata for liveness detection.

**Existing struct** (`internal/state/types.go`):
```go
type RunRecord struct {
    RunID        string
    PipelineName string
    Status       string     // "pending", "running", "completed", "failed", "cancelled"
    Input        string
    CurrentStep  string
    TotalTokens  int
    StartedAt    time.Time
    CompletedAt  *time.Time
    CancelledAt  *time.Time
    ErrorMessage string
    Tags         []string
    BranchName   string
}
```

**New field**:
```go
type RunRecord struct {
    // ... existing fields ...
    PID int // OS process ID of the detached subprocess (0 = in-process or unknown)
}
```

**Schema change** (`internal/state/schema.sql`):
```sql
-- Add to pipeline_run table
ALTER TABLE pipeline_run ADD COLUMN pid INTEGER DEFAULT 0;
```

### 2. Cancellation Record (Existing)

Already exists as `cancellation` table with all needed semantics. No changes required.

```sql
CREATE TABLE IF NOT EXISTS cancellation (
    run_id TEXT PRIMARY KEY,
    requested_at INTEGER NOT NULL,
    force BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);
```

### 3. Event Log (Existing)

Already exists as `event_log` table. No changes required. The detached subprocess uses the same `dbLoggingEmitter` pattern from `cmd/wave/commands/run.go`.

```sql
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
```

## State Store Interface Changes

### New Methods

```go
type StateStore interface {
    // ... existing methods ...

    // UpdateRunPID sets the PID for a pipeline run after subprocess spawn.
    UpdateRunPID(runID string, pid int) error

    // GetRunPID returns the PID for a pipeline run (0 if not set or in-process).
    GetRunPID(runID string) (int, error)
}
```

### Modified Record Scanning

The `queryRuns` method (used by `GetRun`, `GetRunningRuns`, `ListRuns`) must scan the new `pid` column into `RunRecord.PID`.

## TUI Model Changes

### RunningPipeline (Extended)

```go
type RunningPipeline struct {
    RunID      string
    Name       string
    BranchName string
    StartedAt  time.Time
    PID        int       // NEW: process ID for liveness checking
    Detached   bool      // NEW: true if this was launched as a subprocess (vs in-process)
}
```

### PipelineLauncher (Refactored)

```go
type PipelineLauncher struct {
    deps      LaunchDependencies
    program   *tea.Program
    mu        sync.Mutex
    // Removed: cancelFns map — no longer cancelling in-process goroutines
    // Removed: buffers map — event buffers are populated from SQLite, not in-memory
}
```

Key behavior changes:
- `Launch()` spawns `exec.Command("wave", "run", ...)` with `Setsid: true` instead of executing in-process
- `Cancel(runID)` calls `store.RequestCancellation(runID, false)` instead of context cancellation
- `CancelAll()` is a no-op for detached pipelines (they survive TUI exit)
- Event buffer is populated from `store.GetEvents()` on demand

### New: StaleRunDetector

```go
// StaleRunDetector checks for dead subprocess PIDs and transitions stale runs to "failed".
type StaleRunDetector struct {
    store state.StateStore
}

func (d *StaleRunDetector) DetectStaleRuns() ([]string, error)
func IsProcessAlive(pid int) bool
```

## Data Flow

### Launch Flow
```
TUI → PipelineLauncher.Launch()
    → store.CreateRun() → run_id
    → exec.Command("wave", "run", pipeline, "--run", run_id, "--input", input)
    → cmd.SysProcAttr = {Setsid: true}
    → cmd.Start()
    → store.UpdateRunPID(run_id, cmd.Process.Pid)
    → return PipelineLaunchedMsg{RunID, PipelineName}
```

### Monitoring Flow
```
TUI (refresh tick)
    → store.GetRunningRuns()
    → For each run with PID > 0: IsProcessAlive(run.PID)
    → If dead: store.UpdateRunStatus(run_id, "failed", "stale: process not found")
    → Display updated fleet view
```

### Cancellation Flow
```
TUI (user presses 'c')
    → store.RequestCancellation(run_id, false)
    → Start 30s timer

Subprocess (every 5s poll)
    → store.CheckCancellation(run_id)
    → If found: initiate graceful shutdown
    → store.UpdateRunStatus(run_id, "cancelled")

TUI (after 30s if still alive)
    → IsProcessAlive(pid)
    → If alive: syscall.Kill(-pid, SIGKILL)
    → store.UpdateRunStatus(run_id, "failed", "cancellation timeout — force killed")
```

### Event Reconnection Flow
```
TUI reopens → store.GetRunningRuns()
    → User selects running pipeline
    → store.GetEvents(run_id, {After: 0})  // Load all historical events
    → Populate EventBuffer with formatted lines
    → Start polling: store.GetEvents(run_id, {After: lastTimestamp})
    → New events flow into LiveOutputModel
```

## Constraints

- `PID` is 0 for legacy in-process runs (backwards compatible)
- `Setsid: true` is Linux/macOS only — Windows would need `CREATE_NEW_PROCESS_GROUP`
- SQLite busy timeout (5s) handles contention between subprocess writer and TUI reader
- The run_id used by the subprocess must be the same one created by the TUI's `store.CreateRun()` — passed via `--run` flag
