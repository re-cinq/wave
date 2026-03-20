# audit: partial — ETA never displayed in UI (#67)

**Issue**: https://github.com/re-cinq/wave/issues/501
**Labels**: audit
**Author**: nextlevelshit
**Source**: #67 — ETA calculation exists but is never displayed

## Problem

ETA (Estimated Time of Arrival) is calculated by `ETACalculator` in `internal/pipeline/eta.go` and emitted via pipeline events (`EstimatedTimeMs`), but it is never rendered in any of the three display backends:

1. **BubbleTea TUI** (`bubbletea_model.go`) — captures `EstimatedTimeMs` in `PipelineContext` but never renders it
2. **Dashboard** (`dashboard.go`) — no ETA rendering at all
3. **ProgressDisplay** (`progress.go`) — captures ETA but never renders it

## Evidence

- `internal/pipeline/executor.go:922`: `etaCalculator.RemainingMs()` emitted in events
- `internal/display/bubbletea_progress.go:241-242`: `EstimatedTimeMs` captured from events
- `internal/display/formatter.go:283-289`: `FormatETA` function defined but unused by display
- `internal/display/progress.go:356-358`: ETA captured in `ProgressDisplay` state

## Acceptance Criteria

- [ ] ETA is displayed in the BubbleTea TUI header alongside elapsed time
- [ ] ETA uses the existing `FormatETA` or equivalent formatting
- [ ] ETA displays "calculating..." when no estimate is available yet (0 value)
- [ ] ETA disappears or shows completion when pipeline is done
- [ ] Existing tests pass; new test covers ETA rendering in header
