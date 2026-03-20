# audit: regressed — dead PerformanceMetrics code (#68)

**Issue**: [#487](https://github.com/re-cinq/wave/issues/487)
**Labels**: audit
**Author**: nextlevelshit
**Detected by**: wave-audit pipeline run 2026-03-20

## Background

Issue #68 originally identified dead code (`PerformanceMetrics` in `internal/display/metrics.go`) that was never used by production code. The issue was closed, but the dead code remains — a regression.

## Evidence

- `internal/display/metrics.go` defines `PerformanceMetrics` struct, `PerformanceStats` struct, and `NewPerformanceMetrics()` constructor
- No production code outside `metrics.go` instantiates or uses these types
- Only test files reference `NewPerformanceMetrics`:
  - `tests/unit/display/metrics_test.go` — unit tests for the dead code
  - `tests/integration/progress_test.go` — integration tests for the dead code
- `state.PerformanceMetricRecord` is a **separate type** in `internal/state/` that IS actively used by the state store, TUI, and pipeline ETA calculations

## Acceptance Criteria

- [ ] `internal/display/metrics.go` is deleted
- [ ] `tests/unit/display/metrics_test.go` is deleted
- [ ] Tests referencing `display.NewPerformanceMetrics()` in `tests/integration/progress_test.go` are removed
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] No remaining references to `display.PerformanceMetrics` or `display.PerformanceStats`
