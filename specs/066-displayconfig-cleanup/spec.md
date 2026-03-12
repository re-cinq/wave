# cleanup: Remove or wire non-functional DisplayConfig fields (consolidate with #61)

**Issue**: [#66](https://github.com/re-cinq/wave/issues/66)
**Labels**: good first issue, cleanup, priority: low
**Author**: nextlevelshit
**Consolidates**: [#61](https://github.com/re-cinq/wave/issues/61) (dead ETA/metrics code)

## Summary

Multiple fields in `DisplayConfig` (`internal/display/types.go`) are defined and validated but never consulted during rendering. Users/callers who set these fields get no effect. This should be addressed alongside #61 (dead functions in the display package: ETA calculation, performance metrics panel, performance comparison) as a single cleanup effort.

## Unused DisplayConfig Fields (Issue #66)

| Field | Location | Notes |
|-------|----------|-------|
| `ShowDetails` | `types.go:93` | Never checked in display rendering |
| `ShowArtifacts` | `types.go:94` | Default true but not wired into dashboard rendering |
| `CompactMode` | `types.go:95` | `ShouldUseCompactMode()` exists but is never called |
| `MaxHistoryLines` | `types.go:99` | Defined but never referenced in display code |
| `EnableTimestamps` | `types.go:100` | Defined but never enables timestamp rendering |
| `AnimationType` | `types.go:92` | Validated but animation selection is hardcoded in `bubbletea_model.go` |
| `ShowLogo` | `types.go:103` | Logo is always shown; config is ignored |
| `ShowMetrics` | `types.go:104` | Dashboard method exists but is never called |

## Dead Functions (Issue #61)

### 1. ETA Calculation
- `EstimatedTimeMs` field in `PipelineContext` - always set to `0` in all context creation paths
- `AverageStepTimeMs` field in `PipelineContext` - calculated but never displayed
- The conditional ETA display in `dashboard.go` `RenderCompact` never triggers because ETA is always 0

### 2. Performance Metrics Panel
- `RenderPerformanceMetricsPanel()` in `dashboard.go` - never called anywhere in the codebase

### 3. Performance Comparison
- `RenderPerformanceComparison()` in `dashboard.go` - never called anywhere in the codebase

### 4. Related Dead Code
- `ShouldUseCompactMode()` in `dashboard.go` - exists but never called from production code

## Scope Boundaries

### In Scope
- `internal/display/types.go` - field removal from `DisplayConfig` and `PipelineContext`, plus default/validate cleanup
- `internal/display/dashboard.go` - dead function removal (`RenderPerformanceMetricsPanel`, `RenderPerformanceComparison`, `ShouldUseCompactMode`, dead ETA conditional in `RenderCompact`)
- `internal/display/capability.go` - `GetOptimalDisplayConfig()` cleanup
- `internal/display/bubbletea_progress.go` - remove dead field assignments
- `internal/display/progress.go` - remove dead field assignments
- `internal/display/*_test.go` - internal test updates
- `tests/unit/display/*_test.go` - external test updates
- `tests/integration/progress_test.go` - remove dead field references

### Out of Scope
- `AnimationType` type and constants - KEEP because `Spinner`, `NewSpinner()`, `getAnimationFrames()`, `SelectAnimationType()`, and `MultiSpinner.Add()` all use them actively
- `getAnimationFrames()` function - KEEP because `NewSpinner()` calls it
- `internal/event/emitter.go` `EstimatedTimeMs` field - separate event serialization concern
- `internal/state/types.go` `EstimatedTimeMs` - state persistence concern
- `metrics.go` / `PerformanceMetrics` - standalone performance tracking utility, not part of dead rendering code

## Acceptance Criteria

- [ ] Remove unused `DisplayConfig` fields listed above from `internal/display/types.go`
- [ ] Remove `EstimatedTimeMs` and `AverageStepTimeMs` from `PipelineContext`
- [ ] Remove associated helper methods (`ShouldUseCompactMode()`)
- [ ] Remove dead dashboard functions: `RenderPerformanceMetricsPanel()`, `RenderPerformanceComparison()`
- [ ] Remove dead ETA conditional from `RenderCompact()`
- [ ] Update `DefaultDisplayConfig()` and `GetOptimalDisplayConfig()` to remove references to deleted fields
- [ ] Update `Validate()` method to remove validation for deleted fields
- [ ] Update all test files to remove assertions on deleted fields
- [ ] Ensure all display tests pass after removal
- [ ] Verify dashboard rendering is unaffected
