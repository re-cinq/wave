# audit: partial -- ETA calculation for pipeline steps (#67)

**Issue**: [#323](https://github.com/re-cinq/wave/issues/323)
**Author**: nextlevelshit
**Labels**: audit
**Source**: [#67 -- feat: ETA calculation for pipeline steps](https://github.com/re-cinq/wave/issues/67)

## Category

**Partial** -- Infrastructure exists but ETA is not actively calculated.

## Evidence

- `internal/display/progress.go` has `EstimatedTimeMs` field
- Field is set to 0 -- ETA infrastructure exists but not actively calculated

## Remediation

Implement ETA calculation based on historical step duration data from SQLite state DB.

## Existing Infrastructure

The following ETA-related infrastructure already exists and must be wired together:

| Component | Location | Status |
|-----------|----------|--------|
| `EstimatedTimeMs` field on `event.Event` | `internal/event/emitter.go:29` | Exists, always 0 |
| `StateETAUpdated` event state | `internal/event/emitter.go:98` | Defined, never emitted |
| `EstimatedTimeMs` on `PipelineContext` | `internal/display/types.go:218` | Exists, always 0 |
| `AverageStepTimeMs` on `PipelineContext` | `internal/display/types.go:229` | Exists, never populated |
| `GetStepPerformanceStats()` | `internal/state/store.go:1105` | Queries historical AVG/MIN/MAX duration |
| `FormatETA()` | `internal/display/formatter.go:283` | Formats ms to "ETA: Xs" |
| Dashboard ETA rendering | `internal/display/dashboard.go:412-416` | Renders if `EstimatedTimeMs > 0` |
| TUI `StateETAUpdated` handling | `internal/tui/live_output.go:265-270` | Formats ETA events |
| Progress ticker (heartbeat) | `internal/pipeline/executor.go:1754` | Emits every 1s, no ETA |
| `ProgressDisplay.toPipelineContext()` | `internal/display/progress.go:511` | Hardcoded to 0 |
| `BubbleTeaProgressDisplay.buildPipelineContext()` | `internal/display/bubbletea_progress.go:534` | Hardcoded to 0 |

## Acceptance Criteria

1. **Historical data query**: On pipeline start, query `performance_metric` table for average step durations by pipeline name + step ID
2. **ETA computation**: Use exponentially-weighted moving average (EWMA) of recent runs (last 10) to estimate remaining time per step
3. **Real-time updates**: ETA updates emitted via `StateETAUpdated` events at heartbeat interval (1s)
4. **Graceful degradation**: First run of a pipeline (no historical data) shows no ETA rather than incorrect estimates
5. **Per-pipeline ETA**: Calculate remaining time for the entire pipeline based on sum of remaining step ETAs
6. **Display integration**: Both ProgressDisplay and BubbleTeaProgressDisplay populate `EstimatedTimeMs` in their PipelineContext
7. **All existing tests pass**: No regressions in display, event, state, or pipeline tests
