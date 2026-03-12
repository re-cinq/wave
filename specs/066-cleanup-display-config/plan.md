# Implementation Plan: Remove non-functional DisplayConfig fields

## Objective

Remove 8 unused `DisplayConfig` fields and associated dead code (validation logic, helper methods, dead dashboard methods) from the `internal/display` package. These fields are defined and validated but never consulted during rendering.

## Approach

Pure deletion-based cleanup. Remove fields, their validation logic, their defaults, dead methods that reference them, and update all tests. No new code is needed.

## File Mapping

### Files to modify

| File | Action | Changes |
|------|--------|---------|
| `internal/display/types.go` | modify | Remove 8 fields from `DisplayConfig`; remove their validation from `Validate()`; update `DefaultDisplayConfig()` |
| `internal/display/types_test.go` | modify | Remove tests for removed fields and validation; update struct literals |
| `internal/display/capability.go` | modify | Update `GetOptimalDisplayConfig()` to not set removed fields |
| `internal/display/capability_test.go` | modify | Remove assertions on removed fields |
| `internal/display/dashboard.go` | modify | Remove dead methods: `ShouldUseCompactMode()`, `RenderCompact()`, `RenderPerformanceMetricsPanel()`, `RenderPerformanceComparison()`, `renderProjectInfoPanel()`, `renderCurrentAction()` |
| `internal/display/dashboard_test.go` | modify | Remove tests for dead methods: `TestDashboard_ShouldUseCompactMode`, test for `renderProjectInfoPanel` |
| `internal/display/helpers_test.go` | modify | Update `BenchmarkDisplayConfig_Validate` struct literal |
| `tests/unit/display/progress_test.go` | modify | Remove assertions on removed fields; update validation test struct literals |
| `tests/unit/display/dashboard_test.go` | modify | Remove `CompactMode` usage in `TestResponsiveLayout`; update struct literals |

### Files NOT to modify

| File | Reason |
|------|--------|
| `internal/display/animation.go` | `AnimationType` type, constants, `getAnimationFrames()`, `NewSpinner()` are all actively used |
| `internal/display/animation_test.go` | Tests for active animation code |
| `internal/display/bubbletea_model.go` | No references to removed fields |
| `internal/display/metrics.go` | `PerformanceMetrics`/`PerformanceStats` used by pipeline executor, state, TUI |

## Architecture Decisions

1. **Keep `AnimationType` type and constants** — They are actively used by `Spinner`, `NewSpinner`, `getAnimationFrames`, `SelectAnimationType`, and various animation infrastructure. Only the `DisplayConfig.AnimationType` field is removed.

2. **Keep `getAnimationFrames`** — Called by `NewSpinner()` which is production code. The issue's suggestion to remove it is incorrect; it would break the animation system.

3. **Keep `AnimationEnabled`** — Not listed in the issue's unused fields. After removing the `AnimationType` field, `AnimationEnabled` becomes somewhat orphaned (it only controlled `AnimationType` in `Validate()`), but it's out of scope.

4. **Remove dead dashboard methods** — `ShouldUseCompactMode`, `RenderCompact`, `RenderPerformanceMetricsPanel`, `RenderPerformanceComparison`, `renderProjectInfoPanel`, `renderCurrentAction` are defined but never called from production rendering code.

## Risks

| Risk | Mitigation |
|------|------------|
| Removing `AnimationType` constants breaks animation | Scope limited to the config field only; type and constants preserved |
| External code references removed fields | Grep confirms no references outside display package and its tests |
| Tests fail after removal | Update all test struct literals and assertions to match new config |

## Testing Strategy

1. Update all test files that reference removed fields
2. Run `go test ./internal/display/...` to verify display package tests pass
3. Run `go test ./tests/unit/display/...` to verify external display tests pass
4. Run `go test ./...` to verify no other packages are affected
5. Run `go vet ./...` to check for compilation issues
