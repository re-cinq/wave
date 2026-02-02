# Research: Wave Ops Commands

**Feature Branch**: `016-wave-ops-commands`
**Research Date**: 2026-02-02
**Status**: Complete

## Overview

This document captures research findings for implementing operational commands in the Wave CLI: `wave status`, `wave logs`, `wave clean` (enhancement), `wave list` (enhancement), `wave cancel`, and `wave artifacts`.

---

## 1. Current State Management

### SQLite State Store

**Location**: `/home/mwc/Coding/recinq/wave/internal/state/store.go`

The existing state store provides the foundation for status and logs commands.

#### StateStore Interface (store.go:51-58)
```go
type StateStore interface {
    SavePipelineState(id string, status string, input string) error
    SaveStepState(pipelineID string, stepID string, state StepState, err string) error
    GetPipelineState(id string) (*PipelineStateRecord, error)
    GetStepStates(pipelineID string) ([]StepStateRecord, error)
    ListRecentPipelines(limit int) ([]PipelineStateRecord, error)
    Close() error
}
```

#### Data Models (store.go:28-48)

**PipelineStateRecord**:
- `PipelineID`, `Name`, `Status`, `Input`
- `CreatedAt`, `UpdatedAt` (time.Time)

**StepStateRecord**:
- `StepID`, `PipelineID`, `State` (StepState enum)
- `RetryCount`, `StartedAt`, `CompletedAt`
- `WorkspacePath`, `ErrorMessage`

#### State Constants (store.go:18-26)
```go
const (
    StatePending   StepState = "pending"
    StateRunning   StepState = "running"
    StateCompleted StepState = "completed"
    StateFailed    StepState = "failed"
    StateRetrying  StepState = "retrying"
)
```

#### Database Schema (schema.sql:1-23)

```sql
CREATE TABLE IF NOT EXISTS pipeline_state (
    pipeline_id TEXT PRIMARY KEY,
    pipeline_name TEXT NOT NULL,
    status TEXT NOT NULL,
    input TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS step_state (
    step_id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL,
    state TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    started_at INTEGER,
    completed_at INTEGER,
    workspace_path TEXT,
    error_message TEXT,
    FOREIGN KEY (pipeline_id) REFERENCES pipeline_state(pipeline_id) ON DELETE CASCADE
);
```

### Gaps for Status Command

1. **No token usage tracking in state store** - Currently only tracked in events (store.go does not persist token counts)
2. **No elapsed time calculation** - Must compute from `started_at` timestamps
3. **No "current step" concept in DB** - Must derive from step states (last running/pending)
4. **No pipeline execution locking** - Multiple runs of same pipeline can conflict

### Recommended Schema Additions

```sql
-- Add to pipeline_state table:
ALTER TABLE pipeline_state ADD COLUMN total_tokens_used INTEGER DEFAULT 0;
ALTER TABLE pipeline_state ADD COLUMN current_step_id TEXT;

-- Add to step_state table:
ALTER TABLE step_state ADD COLUMN tokens_used INTEGER DEFAULT 0;
ALTER TABLE step_state ADD COLUMN output_summary TEXT;
```

---

## 2. Existing CLI Commands

### Command Registration Pattern (main.go:29-41)

```go
func init() {
    rootCmd.PersistentFlags().StringP("manifest", "m", "wave.yaml", "Path to manifest file")
    rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
    rootCmd.PersistentFlags().String("log-format", "text", "Log format (text, json)")

    rootCmd.AddCommand(commands.NewInitCmd())
    rootCmd.AddCommand(commands.NewValidateCmd())
    rootCmd.AddCommand(commands.NewRunCmd())
    // ... more commands
}
```

### Command Structure Pattern (clean.go:12-42)

```go
type CleanOptions struct {
    Pipeline string
    All      bool
    Force    bool
    KeepLast int
    DryRun   bool
}

func NewCleanCmd() *cobra.Command {
    var opts CleanOptions

    cmd := &cobra.Command{
        Use:   "clean",
        Short: "Clean up project artifacts",
        Long:  `...`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runClean(opts)
        },
    }

    cmd.Flags().StringVar(&opts.Pipeline, "pipeline", "", "...")
    // ... more flags

    return cmd
}
```

### Existing Commands Summary

| Command | File | Purpose |
|---------|------|---------|
| `init` | init.go | Initialize Wave project |
| `validate` | validate.go | Validate manifest/pipeline YAML |
| `run` | run.go | Execute pipeline |
| `do` | do.go | Ad-hoc task execution |
| `resume` | resume.go | Resume paused pipeline |
| `clean` | clean.go | Remove workspaces/state |
| `list` | list.go | List pipelines/personas/adapters |

### Resume Command - Good Reference for Status (resume.go:59-113)

The resume command already implements pipeline listing with status display:

```go
func listResumablePipelines(stateDB string) error {
    store, err := state.NewStateStore(stateDB)
    // ...
    pipelines, err := store.ListRecentPipelines(10)
    // ...
    fmt.Printf("  %-36s  %-12s  %-20s  %s\n", "PIPELINE ID", "STATUS", "LAST UPDATED", "STEPS")
    // ...
}
```

Key formatting helpers already exist:
- `formatTimeAgo()` (resume.go:251-278)
- `formatStatus()` (resume.go:281-294)
- `formatStepState()` (resume.go:297-312)
- `truncateString()` (resume.go:315-323)
- `summarizeSteps()` (resume.go:198-248)

**Pattern to follow**: These helpers should be extracted to a shared `formatting` package for reuse.

---

## 3. Workspace Structure

### Default Workspace Root (run.go:105-109, executor.go:149-154)

```go
wsRoot := m.Runtime.WorkspaceRoot
if wsRoot == "" {
    wsRoot = ".wave/workspaces"
}
```

### Directory Structure

```
.wave/
├── state.db              # SQLite database
├── state.db-shm          # WAL shared memory
├── state.db-wal          # Write-ahead log
├── traces/               # Audit logs
│   └── trace-YYYYMMDD-HHMMSS.log
├── workspaces/           # Per-pipeline workspaces
│   └── <pipeline-name>/
│       └── <step-id>/
│           ├── artifacts/
│           │   └── <injected-artifacts>
│           ├── <mounted-files>
│           └── checkpoint.md (if compacted)
└── pipelines/            # Pipeline definitions
    └── *.yaml
```

### Workspace Manager Interface (workspace.go:29-34)

```go
type WorkspaceManager interface {
    Create(cfg WorkspaceConfig, templateVars map[string]string) (string, error)
    InjectArtifacts(workspacePath string, refs []ArtifactRef, resolvedPaths map[string]string) error
    CleanAll(root string) error
}
```

### Workspace Listing Helper (workspace.go:243-272)

```go
func ListWorkspacesSortedByTime(wsDir string) ([]WorkspaceInfo, error)
```

Already implements sorting by modification time for cleanup.

### Clean Command Implementation (clean.go:45-129)

Key implementation details:
- Handles `--all`, `--pipeline`, `--keep-last`, `--dry-run` flags
- Makes readonly directories writable before removal
- Uses `workspace.ListWorkspacesSortedByTime()` for sorting
- Maintains state.db and traces when using `--keep-last`

**Gap**: No confirmation prompt (--force is defined but not implemented)

---

## 4. Event Emission System

### Event Structure (emitter.go:11-21)

```go
type Event struct {
    Timestamp  time.Time `json:"timestamp"`
    PipelineID string    `json:"pipeline_id"`
    StepID     string    `json:"step_id,omitempty"`
    State      string    `json:"state"`
    DurationMs int64     `json:"duration_ms"`
    Message    string    `json:"message,omitempty"`
    Persona    string    `json:"persona,omitempty"`
    Artifacts  []string  `json:"artifacts,omitempty"`
    TokensUsed int       `json:"tokens_used,omitempty"`
}
```

### Event Emitter Patterns (emitter.go:23-91)

1. **NDJSON mode**: Machine-readable, line-delimited JSON
2. **Human-readable mode**: Colored terminal output with formatting

```go
// Human-readable output example (emitter.go:52-87)
stateColors := map[string]string{
    "started":   "\033[36m",
    "running":   "\033[33m",
    "completed": "\033[32m",
    "failed":    "\033[31m",
    "retrying":  "\033[35m",
}
```

### Event States Used in Executor (executor.go)

| State | Description | Location |
|-------|-------------|----------|
| `started` | Pipeline/step started | executor.go:141-146, 304-311 |
| `running` | Step executing | executor.go:304-311 |
| `completed` | Step/pipeline finished | executor.go:432-442 |
| `failed` | Step/pipeline failed | executor.go:169-175 |
| `retrying` | Step retry attempt | executor.go:231-238 |
| `validating` | Contract validation | executor.go:404-410 |
| `contract_passed` | Contract OK | executor.go:423-429 |
| `contract_failed` | Contract failed | executor.go:413-419 |
| `compacting` | Token compaction | executor.go:609-616 |
| `compacted` | Compaction done | executor.go:646-651 |
| `warning` | Non-fatal issue | executor.go:383-389 |

### Gap for Logs Command

Events are emitted to stdout but not persisted. For `wave logs` to work retroactively:

**Option A**: Persist events to SQLite
```sql
CREATE TABLE pipeline_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pipeline_id TEXT NOT NULL,
    step_id TEXT,
    state TEXT NOT NULL,
    message TEXT,
    persona TEXT,
    tokens_used INTEGER,
    artifacts TEXT,  -- JSON array
    timestamp INTEGER NOT NULL,
    FOREIGN KEY (pipeline_id) REFERENCES pipeline_state(pipeline_id) ON DELETE CASCADE
);
CREATE INDEX idx_events_pipeline ON pipeline_events(pipeline_id, timestamp);
```

**Option B**: Use trace logs (audit/logger.go)
- Already writes to `.wave/traces/trace-*.log`
- Format: `timestamp [TOOL/FILE] pipeline=... step=... ...`
- Would need parsing for `wave logs` output

**Recommendation**: Option A for structured queries, keep Option B for detailed audit.

---

## 5. Audit Logging

### AuditLogger Interface (logger.go:11-15)

```go
type AuditLogger interface {
    LogToolCall(pipelineID, stepID, tool, args string) error
    LogFileOp(pipelineID, stepID, op, path string) error
    Close() error
}
```

### TraceLogger Implementation (logger.go:17-88)

- Writes to `.wave/traces/trace-YYYYMMDD-HHMMSS.log`
- Automatic credential scrubbing via regex
- Format: `timestamp [TOOL/FILE] pipeline=... step=... tool=... args=...`

### Credential Patterns Scrubbed (logger.go:23-32)

```go
var credentialPatterns = []string{
    `API[_-]?KEY`,
    `TOKEN`,
    `SECRET`,
    `PASSWORD`,
    `CREDENTIAL`,
    `AUTH`,
    `PRIVATE[_-]?KEY`,
    `ACCESS[_-]?KEY`,
}
```

### Gap for Logs Command

Trace logs are useful for debugging but:
1. No structured format for filtering by step/time
2. No log levels (debug/info/error)
3. No correlation between trace entries and events

---

## 6. Pipeline Executor

### Executor Interface (executor.go:22-26)

```go
type PipelineExecutor interface {
    Execute(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error
    Resume(ctx context.Context, pipelineID string, fromStep string) error
    GetStatus(pipelineID string) (*PipelineStatus, error)
}
```

### PipelineStatus Structure (executor.go:28-36)

```go
type PipelineStatus struct {
    ID             string
    State          string
    CurrentStep    string
    CompletedSteps []string
    FailedSteps    []string
    StartedAt      time.Time
    CompletedAt    *time.Time
}
```

### In-Memory Execution Tracking (executor.go:76-85)

```go
type PipelineExecution struct {
    Pipeline       *Pipeline
    Manifest       *manifest.Manifest
    States         map[string]string
    Results        map[string]map[string]interface{}
    ArtifactPaths  map[string]string
    WorkspacePaths map[string]string
    Input          string
    Status         *PipelineStatus
}
```

### GetStatus Implementation (executor.go:724-734)

Already exists but only works for in-memory executions:

```go
func (e *DefaultPipelineExecutor) GetStatus(pipelineID string) (*PipelineStatus, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    execution, exists := e.pipelines[pipelineID]
    if !exists {
        return nil, fmt.Errorf("pipeline %q not found", pipelineID)
    }

    return execution.Status, nil
}
```

**Gap**: Status only available while pipeline is running. Need to reconstruct from database.

---

## 7. External Tool Reference Patterns

### kubectl logs

Reference: [kubectl logs documentation](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_logs/)

Key patterns:
- `--follow` / `-f`: Stream logs in real-time
- `--tail <n>`: Show last n lines
- `--since <duration>`: Show logs since time (e.g., `5m`, `1h`)
- `--timestamps`: Include timestamps in output
- `--container <name>`: Filter by container (analogous to `--step`)
- `--max-log-requests`: Limit concurrent connections

Applicable to `wave logs`:
```
wave logs                          # All logs from last run
wave logs --follow                 # Stream in real-time
wave logs --step investigate       # Filter by step
wave logs --since 10m              # Last 10 minutes
wave logs --errors                 # Only errors
wave logs <run-id>                 # Specific run
```

### docker ps / docker container ls

Reference: [docker ps documentation](https://docs.docker.com/reference/cli/docker/container/ls/)

Key patterns:
- Default shows only running containers
- `-a` / `--all`: Show all (including stopped)
- `--filter status=<status>`: Filter by state
- `--format`: Custom output formatting (Go templates)
- `--no-trunc`: Don't truncate output

Applicable to `wave status`:
```
wave status                        # Running pipelines
wave status --all                  # All recent pipelines
wave status <run-id>               # Specific run details
wave status --format json          # JSON output
wave status --filter status=failed # Filter by status
```

---

## 8. Testing Patterns

### Test Helper Pattern (clean_test.go:17-115)

```go
type cleanTestEnv struct {
    t          *testing.T
    rootDir    string
    origDir    string
    workspaces []string
}

func newCleanTestEnv(t *testing.T) *cleanTestEnv {
    // Setup temp dir, change working directory
}

func (e *cleanTestEnv) cleanup() {
    // Restore original directory
}

func (e *cleanTestEnv) createWorkspace(name string, modTime time.Time) string {
    // Create test workspace with specific modification time
}
```

### State Store Testing (store_test.go:16-52)

```go
func setupTestStore(t *testing.T) (StateStore, func()) {
    store, err := NewStateStore(":memory:")  // In-memory for unit tests
    // ...
}

func setupTestStoreWithFile(t *testing.T) (StateStore, func()) {
    // File-based for concurrent access tests
}
```

### Command Testing Pattern (list_test.go:63-85)

```go
func executeListCmd(args ...string) (stdout, stderr string, err error) {
    cmd := NewListCmd()

    // Capture stdout since commands use fmt.Printf
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    err = cmd.Execute()

    w.Close()
    os.Stdout = oldStdout
    // ...
}
```

### Table-Driven Tests (store_test.go:74-124, list_test.go:713-756)

```go
testCases := []struct {
    name   string
    // ... fields
}{
    {name: "case 1", ...},
    {name: "case 2", ...},
}

for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        // ...
    })
}
```

---

## 9. Implementation Recommendations

### New Commands to Create

| Command | File | Priority |
|---------|------|----------|
| `status` | cmd/wave/commands/status.go | P1 |
| `logs` | cmd/wave/commands/logs.go | P1 |
| `cancel` | cmd/wave/commands/cancel.go | P2 |
| `artifacts` | cmd/wave/commands/artifacts.go | P2 |

### Enhancements to Existing

| Target | Change |
|--------|--------|
| `clean.go` | Add confirmation prompt when not --force |
| `list.go` | Add JSON output for pipelines subcommand |
| `state/store.go` | Add event persistence, token tracking |

### Shared Utilities to Extract

Create `internal/cli/format.go`:
```go
package cli

func FormatTimeAgo(t time.Time) string
func FormatStatus(status string) string
func FormatStepState(state state.StepState) string
func TruncateString(s string, maxLen int) string
func SummarizeSteps(steps []state.StepStateRecord) string
```

### Database Migrations

Create `internal/state/migrations/`:
1. `001_add_events_table.sql`
2. `002_add_token_tracking.sql`

### Signal Handling for Cancel

Leverage existing pattern from run.go:59-67:
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt)
go func() {
    <-sigChan
    cancel()
}()
```

For `wave cancel`, need:
1. PID file or lock file to identify running pipeline
2. Inter-process signaling (SIGTERM/SIGINT)
3. Or: use database flag for cooperative cancellation

---

## 10. Open Questions Resolution

### Should `wave logs` support log levels?

**Recommendation**: Yes, add `--level` flag with values `debug`, `info`, `warn`, `error`.
- Store level with each event
- Default to `info` and above
- `--debug` flag shows all

### Should `wave clean` have scheduled/automatic mode?

**Recommendation**: No automatic mode in CLI. Users should:
- Use cron/scheduled tasks externally
- Or configure in CI/CD pipelines

Add `--older-than <duration>` flag instead:
```
wave clean --older-than 7d
```

### Should `wave cancel` send SIGTERM or use cancellation token?

**Recommendation**: Use cooperative cancellation via database flag.
- Set `status = 'cancelling'` in pipeline_state
- Executor checks flag between steps
- More reliable than process signals
- Works across network boundaries

---

## 11. File References Summary

| File | Lines | Relevance |
|------|-------|-----------|
| internal/state/store.go | 1-277 | Core state management |
| internal/state/schema.sql | 1-23 | Database schema |
| internal/event/emitter.go | 1-92 | Event emission |
| internal/audit/logger.go | 1-89 | Audit logging |
| internal/workspace/workspace.go | 1-283 | Workspace management |
| internal/pipeline/executor.go | 1-735 | Pipeline execution |
| cmd/wave/main.go | 1-49 | CLI entry point |
| cmd/wave/commands/clean.go | 1-130 | Clean command reference |
| cmd/wave/commands/list.go | 1-450 | List command reference |
| cmd/wave/commands/resume.go | 1-324 | Status formatting helpers |
| cmd/wave/commands/run.go | 1-268 | Run command reference |

---

## 12. External References

- [kubectl logs documentation](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_logs/)
- [docker ps documentation](https://docs.docker.com/reference/cli/docker/container/ls/)
- [Cobra CLI framework](https://github.com/spf13/cobra)
- [SQLite WAL mode](https://www.sqlite.org/wal.html)
