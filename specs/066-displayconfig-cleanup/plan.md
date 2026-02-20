# Implementation Plan: DisplayConfig Cleanup (#66 + #61)

## 1. Objective

Remove unused `DisplayConfig` fields and dead display functions that were never wired into rendering. This consolidates issue #66 (dead config fields) and #61 (dead ETA/metrics functions) into a single cleanup PR.

## 2. Approach

This is a pure deletion/cleanup task. The strategy is:
1. Remove dead fields from structs
2. Remove dead functions/methods
3. Remove dead validation logic
4. Update defaults to exclude removed fields
5. Fix all test files that reference removed code
6. Verify the build and tests pass

Because none of these fields or functions are consulted during rendering, removal has zero behavioral impact. The `AnimationType` type and its constants are KEPT because they are actively used by `Spinner`, `NewSpinner()`, `getAnimationFrames()`, `SelectAnimationType()`, and `MultiSpinner.Add()`.

## 3. File Mapping

| File | Action | Changes |
|------|--------|---------|
| `internal/display/types.go` | modify | Remove `ShowDetails`, `ShowArtifacts`, `CompactMode`, `MaxHistoryLines`, `EnableTimestamps`, `ShowLogo`, `ShowMetrics` fields from `DisplayConfig`. Remove `AnimationType` field from `DisplayConfig` (keep the type itself). Remove `EstimatedTimeMs`, `AverageStepTimeMs` from `PipelineContext`. Update `DefaultDisplayConfig()`. Update `Validate()` to remove `MaxHistoryLines` and `AnimationType` validation. |
| `internal/display/dashboard.go` | modify | Remove `RenderPerformanceMetricsPanel()`, `RenderPerformanceComparison()`, `ShouldUseCompactMode()`. Remove dead ETA conditional in `RenderCompact()`. |
| `internal/display/capability.go` | modify | Update `GetOptimalDisplayConfig()` to remove references to deleted fields. Keep `SelectAnimationType()` as it's used independently. |
| `internal/display/bubbletea_progress.go` | modify | Remove `EstimatedTimeMs: 0` assignments from context creation. |
| `internal/display/progress.go` | modify | Remove `EstimatedTimeMs: 0` assignments from `toPipelineContext()` and `CreatePipelineContext()`. |
| `internal/display/types_test.go` | modify | Remove tests for deleted fields in `TestDefaultDisplayConfig`, `TestDisplayConfig_Validate_MaxHistoryLines`, `TestDisplayConfig_Validate_AnimationType`, `TestDisplayConfig_Validate_AnimationDisabled`, `TestPipelineContext_Structure`. |
| `internal/display/dashboard_test.go` | modify | Remove `TestDashboard_ShouldUseCompactMode`. Remove `EstimatedTimeMs`/`AverageStepTimeMs` from test contexts. |
| `internal/display/capability_test.go` | modify | Remove `MaxHistoryLines` assertion in `TestGetOptimalDisplayConfig`. |
| `internal/display/helpers_test.go` | modify | Remove `MaxHistoryLines` and `AnimationType` from `BenchmarkDisplayConfig_Validate`. |
| `internal/display/bubbletea_model_test.go` | modify | No changes needed (doesn't reference deleted fields). |
| `internal/display/progress_test.go` | modify | Remove `EstimatedTimeMs`/`AverageStepTimeMs` from test contexts. |
| `tests/unit/display/progress_test.go` | modify | Remove assertions on `ShowDetails`, `ShowArtifacts`, `CompactMode`, `MaxHistoryLines`, `EnableTimestamps`, `ShowLogo`, `ShowMetrics`, `AnimationType`. Remove `EstimatedTimeMs`/`AverageStepTimeMs` references. |
| `tests/unit/display/dashboard_test.go` | modify | Remove `CompactMode` reference. Update `AnimationType` references (keep type, remove config field usage). |
| `tests/integration/progress_test.go` | modify | Remove `AverageStepTimeMs`/`EstimatedTimeMs` usage. |

## 4. Architecture Decisions

1. **Keep `AnimationType` type + constants**: They are actively used by `Spinner`, `MultiSpinner`, `SelectAnimationType()`, etc. Only the `DisplayConfig.AnimationType` field is dead (never consulted during rendering).

2. **Keep `getAnimationFrames()`**: Called by `NewSpinner()` which is used throughout.

3. **Keep `metrics.go`**: The `PerformanceMetrics` struct is a standalone performance tracking utility that may be used for profiling. It's not part of the "dead display rendering" scope.

4. **Keep `EstimatedTimeMs` in `event.Event` and `state.ProgressSnapshotRecord`**: These are in different packages and serve different purposes (event serialization and state persistence). Removing them would change the event API contract and DB schema, which is out of scope for a display cleanup.

5. **Remove `DisplayConfig.AnimationType` field but keep validation framework**: The `Validate()` method becomes simpler but still validates `RefreshRate`, `ColorMode`, and `ColorTheme`. The `AnimationEnabled` field stays because it controls spinner behavior via `animation.go`.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Test files in `tests/` may reference removed fields | Comprehensive grep already done; all references identified |
| External packages may reference `DisplayConfig` fields | Grep shows no usage outside `internal/display/` and test dirs |
| Removing ETA conditional changes `RenderCompact` behavior | No change - ETA was always 0, so the conditional never triggered |
| `AnimationType` removal from config breaks `SelectAnimationType` | NOT removing the type - only the config field |

## 6. Testing Strategy

- **Primary**: Run `go test ./internal/display/...` after each phase to catch breakage early
- **Full suite**: Run `go test ./...` after all changes to verify no cross-package impact
- **Race detector**: Run `go test -race ./internal/display/...` to verify concurrent safety unchanged
- **Verification**: Build with `go build ./...` to catch any compilation errors
