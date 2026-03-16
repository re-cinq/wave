# Data Model: Continuous Pipeline Execution

**Feature**: #201 — Continuous Pipeline Execution
**Date**: 2026-03-16

## Entities

### ContinuousRun (in-memory only)

The top-level coordination struct managing the iteration loop. Not persisted to SQLite — each child pipeline run is persisted via the existing state store.

```go
// ContinuousRun tracks the overall continuous session state.
type ContinuousRun struct {
    Source        WorkItemSource     // Source producing work items
    PipelineName  string             // Pipeline to execute per iteration
    OnFailure     FailurePolicy      // "halt" or "skip"
    MaxIterations int                // 0 = unlimited
    Delay         time.Duration      // Delay between iterations

    // Accumulated state (updated after each iteration)
    Iterations    []IterationResult  // Ordered results
    ProcessedIDs  map[string]bool    // Dedup tracking (work item ID → processed)
    Succeeded     int                // Cumulative success count
    Failed        int                // Cumulative failure count
    Skipped       int                // Cumulative skip count (dedup or source exhaustion)
}
```

### WorkItem

Represents a single unit of work produced by a source.

```go
// WorkItem is a single input to a pipeline iteration.
type WorkItem struct {
    ID    string // Unique identifier (e.g., "42" for GitHub issue #42, line content for file source)
    Input string // The full input string passed to pipeline.Execute() (e.g., issue URL)
}
```

### WorkItemSource (interface)

Abstraction for work item discovery. Each call to `Next()` returns the next unprocessed item, or `nil` when exhausted.

```go
// WorkItemSource produces work items for the continuous loop.
type WorkItemSource interface {
    // Next returns the next work item, or nil when exhausted.
    // Returns error on transient failures (e.g., rate limits, network errors).
    Next(ctx context.Context) (*WorkItem, error)

    // Name returns a human-readable description of the source for logging.
    Name() string
}
```

### GitHubSource

Queries GitHub issues via `gh issue list`.

```go
// GitHubSource fetches work items from GitHub issues using the gh CLI.
type GitHubSource struct {
    Label     string // Issue label filter
    State     string // "open" (default)
    Sort      string // "created" (default)
    Direction string // "asc" (default)
    Limit     int    // Max items to fetch (default: 100)

    // Internal state
    items     []*WorkItem // Pre-fetched items
    index     int         // Current position
    fetched   bool        // Whether initial fetch has occurred
}
```

### FileSource

Reads work items from a local file, one per line.

```go
// FileSource reads work items from a file, one line per item.
type FileSource struct {
    Path  string      // File path
    items []*WorkItem // Pre-loaded items
    index int         // Current position
}
```

### IterationResult

The outcome of a single pipeline execution within the loop.

```go
// IterationResult records the outcome of one iteration.
type IterationResult struct {
    Iteration int           // 1-based iteration number
    WorkItem  *WorkItem     // Input work item
    RunID     string        // Pipeline run ID (for wave logs)
    Status    IterationStatus // success, failed, skipped
    Duration  time.Duration // Execution time
    Error     error         // Non-nil for failed iterations
}

type IterationStatus string

const (
    IterationSuccess IterationStatus = "success"
    IterationFailed  IterationStatus = "failed"
    IterationSkipped IterationStatus = "skipped"
)
```

### FailurePolicy

Controls loop behavior on iteration failure.

```go
type FailurePolicy string

const (
    FailurePolicyHalt FailurePolicy = "halt" // Default: stop on first failure
    FailurePolicySkip FailurePolicy = "skip" // Log failure, continue to next item
)
```

### SourceConfig

Parsed representation of the `--source` URI.

```go
// SourceConfig holds the parsed source URI configuration.
type SourceConfig struct {
    Provider string            // "github", "file"
    Params   map[string]string // Key-value parameters
    RawURI   string            // Original URI for error messages
}
```

## Event Extensions

New fields on `event.Event` for iteration observability:

```go
// Added to existing Event struct
Iteration      int    `json:"iteration,omitempty"`        // Current iteration number (1-based)
TotalProcessed int    `json:"total_processed,omitempty"`  // Cumulative items processed
WorkItemID     string `json:"work_item_id,omitempty"`     // Current work item identifier
```

New event states emitted by the continuous runner:

| State | When | Key Fields |
|-------|------|------------|
| `loop_start` | Continuous run begins | Source name |
| `loop_iteration_start` | Before each iteration | Iteration, WorkItemID |
| `loop_iteration_complete` | After successful iteration | Iteration, RunID, DurationMs |
| `loop_iteration_failed` | After failed iteration | Iteration, RunID, Error, FailurePolicy |
| `loop_summary` | After all iterations complete | TotalProcessed, Succeeded, Failed |

## CLI Extensions

New fields on `RunOptions` struct:

```go
// Added to existing RunOptions struct
Continuous    bool   // --continuous flag
Source        string // --source URI
MaxIterations int    // --max-iterations N
Delay         string // --delay duration
OnFailure     string // --on-failure halt|skip
```

## Relationships

```
RunOptions --[--continuous]--> ContinuousRun
ContinuousRun --[source]--> WorkItemSource
ContinuousRun --[iterates]--> IterationResult[]
IterationResult --[run_id]--> state.RunRecord (existing, persisted in SQLite)
WorkItemSource <|-- GitHubSource
WorkItemSource <|-- FileSource
```

## Invariants

1. Each `IterationResult.RunID` maps to exactly one `state.RunRecord` in the SQLite state store
2. `ContinuousRun.ProcessedIDs` grows monotonically within a session (items never un-processed)
3. `ContinuousRun.Succeeded + ContinuousRun.Failed + ContinuousRun.Skipped == len(ContinuousRun.Iterations)`
4. When `OnFailure == "halt"`, the loop exits immediately after the first `IterationFailed` result
5. `--continuous` and `--from-step` are mutually exclusive (validated in CLI)
6. `GitHubSource` fetches all matching items once and iterates locally (no per-iteration API calls)
