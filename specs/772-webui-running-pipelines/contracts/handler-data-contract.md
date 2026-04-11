# Contract: handleRunsPage Template Data

**Type**: Structural (Go template data contract)  
**Feature**: `772-webui-running-pipelines`

## Description

The `handleRunsPage` handler in `internal/webui/handlers_runs.go` must populate
the following template data struct fields for the running-pipelines section to
render correctly.

## Required Fields (additions to existing struct)

```go
RunningRuns  []RunSummary  // must be non-nil (empty slice when no running runs)
RunningCount int           // must equal len(RunningRuns)
```

## Invariants

1. `RunningRuns` contains ONLY runs with `Status == "running"`.
2. `RunningCount == len(RunningRuns)` always (no discrepancy).
3. `RunningRuns` respects `FilterPipeline` — if `FilterPipeline != ""`, only
   runs matching that pipeline name are included.
4. `RunningRuns` does NOT respect `FilterStatus` — the section always shows
   running runs regardless of the active status tab.
5. Each entry in `RunningRuns` has been processed by `enrichRunSummaries` so
   `Progress`, `StepsCompleted`, `StepsTotal`, `Models`, `TotalTokens` are
   populated.
6. `RunningRuns` does NOT contain child runs (top-level runs only — `ParentRunID == ""`).

## Verification

- Unit test: `TestHandleRunsPage_RunningSection` in `handlers_runs_test.go`
- Checks: `RunningRuns` populated when running runs exist, empty slice (not nil)
  when no running runs, `RunningCount` matches len.
