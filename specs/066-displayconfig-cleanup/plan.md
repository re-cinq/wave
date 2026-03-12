# Implementation Plan: DisplayConfig Cleanup (#66 + #61)

## 1. Objective

Remove unused `DisplayConfig` fields and dead display functions that were never wired into rendering. Consolidates issue #66 (dead config fields) and #61 (dead ETA/metrics functions) into a single cleanup PR.

## 2. Approach

Pure deletion/cleanup task with zero behavioral impact:
1. Remove dead fields from `DisplayConfig` and `PipelineContext` structs
2. Remove dead functions/methods from `dashboard.go`
3. Remove dead validation logic from `Validate()`
4. Update `DefaultDisplayConfig()` and `GetOptimalDisplayConfig()` to exclude removed fields
5. Fix all test files that reference removed code
6. Verify the build and tests pass

Because none of these fields or functions are consulted during rendering, removal has zero behavioral impact. The `AnimationType` type and its constants are KEPT because they are actively used by `Spinner`, `NewSpinner()`, `getAnimationFrames()`, `SelectAnimationType()`, and `MultiSpinner.Add()`.

## 3. File Mapping

| File | Action | Changes |
|------|--------|---------|
| `internal/display/types.go` | modify | Remove `ShowDetails`, `ShowArtifacts`, `CompactMode`, `MaxHistoryLines`, `EnableTimestamps`, `ShowLogo`, `ShowMetrics` fields from `DisplayConfig`. Remove `AnimationType` field from `DisplayConfig` (keep the type itself). Remove `EstimatedTimeMs`, `AverageStepTimeMs` from `PipelineContext`. Update `DefaultDisplayConfig()`. Simplify `Validate()`. |
| `internal/display/dashboard.go` | modify | Remove `RenderPerformanceMetricsPanel()`, `RenderPerformanceComparison()`, `ShouldUseCompactMode()`. Remove dead ETA conditional from `RenderCompact()`. |
| `internal/display/capability.go` | modify | Remove deleted field assignments from `GetOptimalDisplayConfig()`. Keep `SelectAnimationType()`. |
| `internal/display/bubbletea_progress.go` | modify | Remove `EstimatedTimeMs: 0` from context creation. |
| `internal/display/progress.go` | modify | Remove `EstimatedTimeMs: 0` from context creation. |
| `internal/display/types_test.go` | modify | Remove tests for deleted fields. |
| `internal/display/dashboard_test.go` | modify | Remove `TestDashboard_ShouldUseCompactMode`. Remove dead field references from test contexts. |
| `internal/display/capability_test.go` | modify | Remove `MaxHistoryLines` assertion. |
| `internal/display/helpers_test.go` | modify | Remove `MaxHistoryLines`/`AnimationType` from validate benchmark. |
| `tests/unit/display/progress_test.go` | modify | Remove assertions on deleted DisplayConfig and PipelineContext fields. |
| `tests/unit/display/dashboard_test.go` | modify | Remove `CompactMode` reference. Update `AnimationType` config field usage. |
| `tests/integration/progress_test.go` | modify | Remove `AverageStepTimeMs`/`EstimatedTimeMs` usage. |

## 4. Architecture Decisions

1. **Keep `AnimationType` type + constants**: Actively used by `Spinner`, `MultiSpinner`, `SelectAnimationType()`, etc. Only the `DisplayConfig.AnimationType` field is dead.
2. **Keep `getAnimationFrames()`**: Called by `NewSpinner()` which is used throughout.
3. **Keep `metrics.go`**: Standalone performance tracking utility, not part of dead display rendering scope.
4. **Keep `EstimatedTimeMs` in event/state packages**: Different packages with different purposes (event serialization, state persistence). Out of scope.
5. **Keep `AnimationEnabled`**: Controls spinner behavior via `animation.go` - still functional.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Test files reference removed fields | Comprehensive grep identifies all references |
| External packages reference `DisplayConfig` fields | No usage found outside `internal/display/` and test dirs |
| Removing ETA conditional changes `RenderCompact` behavior | ETA was always 0, conditional never triggered |
| `AnimationType` removal from config breaks `SelectAnimationType` | NOT removing the type - only the config field |

## 6. Testing Strategy

- Run `go test ./internal/display/...` after each phase
- Run `go test ./...` after all changes
- Run `go test -race ./internal/display/...` for concurrent safety
- Run `go build ./...` for compilation verification
