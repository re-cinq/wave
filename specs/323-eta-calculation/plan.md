# Implementation Plan: ETA Calculation for Pipeline Steps

## Objective

Wire historical step duration data from the SQLite state DB into the existing ETA display infrastructure so that running pipelines show estimated time remaining.

## Approach

Create a standalone `ETACalculator` in `internal/pipeline/` that:
1. Pre-loads historical step durations from the state store at pipeline start
2. Tracks actual durations of completed steps during the current run
3. Computes remaining time as the sum of estimated durations for incomplete steps
4. Is queried by the progress ticker to populate `EstimatedTimeMs` on heartbeat events

Both display implementations (`ProgressDisplay` and `BubbleTeaProgressDisplay`) already build `PipelineContext` structs with `EstimatedTimeMs` and `AverageStepTimeMs` fields. These will be populated from actual step duration tracking rather than hardcoded to 0.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/eta.go` | **Create** | `ETACalculator` — loads historical data, tracks current run, computes ETA |
| `internal/pipeline/eta_test.go` | **Create** | Unit tests for ETACalculator |
| `internal/pipeline/executor.go` | **Modify** | Initialize ETACalculator at pipeline start; feed step completions; use in progress ticker |
| `internal/display/bubbletea_progress.go` | **Modify** | Populate `EstimatedTimeMs` and `AverageStepTimeMs` in `buildPipelineContext()` |
| `internal/display/progress.go` | **Modify** | Populate `EstimatedTimeMs` and `AverageStepTimeMs` in `toPipelineContext()` |

## Architecture Decisions

### 1. EWMA vs Simple Average

Use simple average of the last 10 successful runs per step. EWMA adds complexity with minimal benefit at this scale (most pipelines have <20 historical runs). The `GetStepPerformanceStats` method already computes AVG(duration_ms) — we reuse this.

### 2. ETACalculator Placement

Place in `internal/pipeline/` rather than `internal/display/` because:
- It needs access to the state store (dependency already present in executor)
- It's a runtime computation, not a rendering concern
- The executor owns the step lifecycle and heartbeat emission

### 3. Historical Data Query Strategy

Query once at pipeline start (not per-tick) to avoid DB pressure. Cache results in memory. For the current run, track actual durations inline and blend with historical averages using a simple heuristic: if a step has completed in the current run, use its actual duration for the "average" (most recent data is most relevant).

### 4. First-Run Behavior

When no historical data exists for a step, return 0 for that step's estimate. The ETA will be "partial" — showing estimates only for steps with history. If no steps have history, no ETA is displayed (graceful degradation).

### 5. Progress Ticker Integration

The existing `startProgressTicker` emits `StateStepProgress` events every 1s. We augment these events with `EstimatedTimeMs` computed by calling `ETACalculator.RemainingMs()`. Additionally, emit a dedicated `StateETAUpdated` event when the ETA changes materially (>10% change or step completion).

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Historical data query slow on large DBs | Low | Query once at start, cached in memory; existing `GetStepPerformanceStats` has indexes |
| ETA accuracy poor for variable-duration steps | Medium | Accept this — simple average is "good enough" for v1. Can iterate to EWMA later |
| Race conditions between ticker goroutine and step completion | Medium | ETACalculator uses sync.Mutex; all mutations are serialized |
| Breaking existing tests | Low | Only adding new behavior to previously-zero fields; no existing assertions on ETA values should break |

## Testing Strategy

1. **Unit tests** (`internal/pipeline/eta_test.go`):
   - ETACalculator with no historical data returns 0
   - ETACalculator with historical data returns correct estimate
   - Step completion updates ETA correctly
   - Thread safety under concurrent access
   - Edge cases: single step, all steps completed, negative durations

2. **Existing test validation**:
   - Run `go test ./...` to confirm no regressions
   - Integration test at `tests/integration/progress_test.go` already exercises ETA field

3. **Manual verification**:
   - Run a pipeline twice; second run should show ETA in dashboard/TUI
