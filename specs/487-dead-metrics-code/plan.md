# Implementation Plan

## Objective

Delete the unused `PerformanceMetrics` and `PerformanceStats` types from `internal/display/metrics.go` and remove all associated test code. This is dead code that was identified in #68 but never actually removed.

## Approach

Straightforward deletion. The `display.PerformanceMetrics` type is completely isolated — no production code references it. The `state.PerformanceMetricRecord` type is separate and unaffected.

## File Mapping

| File | Action | Reason |
|------|--------|--------|
| `internal/display/metrics.go` | **delete** | Entire file is dead code |
| `tests/unit/display/metrics_test.go` | **delete** | Tests only the dead code |
| `tests/integration/progress_test.go` | **modify** | Remove `TestPerformanceMonitoring` and `TestPerformanceOverheadTarget` functions that reference `display.NewPerformanceMetrics` |

## Architecture Decisions

- **Delete rather than wire in**: The issue offers two options (wire it in or delete). Deletion is the correct choice because:
  - The `state.PerformanceMetricRecord` already serves the performance tracking role in the actual system
  - `display.PerformanceMetrics` duplicates concepts already handled by the state store
  - No existing code needs these types — they were scaffolded but never integrated

## Risks

- **Low**: The code is provably unused. The only risk is missing a reference, which `go build` will catch immediately.
- **Mitigation**: Run `go build ./...` and `go test ./...` after deletion to verify clean compilation.

## Testing Strategy

- No new tests needed — this is pure deletion of dead code
- Validate with `go build ./...` (compilation check)
- Validate with `go test ./...` (no test regressions)
- Grep for remaining `display.PerformanceMetrics` references to confirm clean removal
