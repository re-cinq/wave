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
- `AnimationType` type and constants - used internally by `Spinner`/`getAnimationFrames` but the `DisplayConfig.AnimationType` field is validated without effect since `bubbletea_model.go` hardcodes its own spinner frames
- `getAnimationFrames()` in `animation.go` - still used by `NewSpinner()`, so must NOT be removed
- `metrics.go` - `PerformanceMetrics` struct - appears to be standalone performance tracking, not called from dashboard rendering; needs further investigation on whether it's used elsewhere

## Recommendation: Remove

Given that these fields have never been wired in since their introduction, and no downstream consumers depend on them, the cleanest path is removal. If any features are needed later, they can be re-added with proper rendering integration from the start.

## Acceptance Criteria

- [ ] Remove unused `DisplayConfig` fields listed above from `internal/display/types.go`
- [ ] Remove associated helper methods (`ShouldUseCompactMode()`)
- [ ] Remove dead dashboard functions: `RenderPerformanceMetricsPanel()`, `RenderPerformanceComparison()`
- [ ] Remove `EstimatedTimeMs` and `AverageStepTimeMs` from `PipelineContext` (always zero)
- [ ] Update `DefaultDisplayConfig()` and `GetOptimalDisplayConfig()` to remove references to deleted fields
- [ ] Update `Validate()` method to remove validation for deleted fields
- [ ] Update all test files to remove assertions on deleted fields
- [ ] Ensure all display tests pass after removal
- [ ] Verify dashboard rendering is unaffected (these fields were never consulted)

## Scope Boundaries

### In Scope
- `internal/display/types.go` - field removal + default/validate cleanup
- `internal/display/dashboard.go` - dead function removal (`RenderPerformanceMetricsPanel`, `RenderPerformanceComparison`, `ShouldUseCompactMode`, dead ETA conditional in `RenderCompact`)
- `internal/display/capability.go` - `GetOptimalDisplayConfig()` cleanup
- `internal/display/*_test.go` - test updates
- `tests/unit/display/*_test.go` - external test updates
- `tests/integration/progress_test.go` - remove `AverageStepTimeMs`/`EstimatedTimeMs` usage

### Out of Scope
- `AnimationType` type and constants - KEEP because `Spinner`, `NewSpinner()`, `getAnimationFrames()`, `SelectAnimationType()`, and `MultiSpinner.Add()` all use them actively
- `getAnimationFrames()` function - KEEP because `NewSpinner()` calls it
- `internal/event/emitter.go` `EstimatedTimeMs` field - OUT OF SCOPE (separate event serialization concern)
- `internal/state/types.go` `EstimatedTimeMs` - OUT OF SCOPE (state persistence concern)
- `metrics.go` / `PerformanceMetrics` - KEEP for now pending separate investigation
