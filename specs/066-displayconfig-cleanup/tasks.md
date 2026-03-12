# Tasks

## Phase 1: Remove Dead Dashboard Methods
- [X] Task 1.1: Remove `ShouldUseCompactMode()` from `internal/display/dashboard.go`
- [X] Task 1.2: Remove `RenderPerformanceMetricsPanel()` from `internal/display/dashboard.go`
- [X] Task 1.3: Remove `RenderPerformanceComparison()` from `internal/display/dashboard.go`
- [X] Task 1.4: Remove `renderProjectInfoPanel()` from `internal/display/dashboard.go`
- [X] Task 1.5: Remove `renderCurrentAction()` from `internal/display/dashboard.go`

## Phase 2: Remove Dead DisplayConfig Fields
- [X] Task 2.1: Remove fields `ShowDetails`, `ShowArtifacts`, `CompactMode`, `MaxHistoryLines`, `EnableTimestamps`, `ShowLogo`, `ShowMetrics`, `AnimationType`, `AnimationEnabled` from `DisplayConfig` struct in `internal/display/types.go` [P]
- [X] Task 2.2: Update `DefaultDisplayConfig()` to remove defaults for removed fields in `internal/display/types.go` [P]
- [X] Task 2.3: Update `Validate()` to remove validation for `MaxHistoryLines`, `AnimationType`, and `AnimationEnabled` in `internal/display/types.go` [P]
- [X] Task 2.4: Update `GetOptimalDisplayConfig()` in `internal/display/capability.go` to remove dead fields from return value

## Phase 3: Update Tests
- [X] Task 3.1: Update `TestDefaultDisplayConfig` in `internal/display/types_test.go` — remove assertions for dead fields [P]
- [X] Task 3.2: Remove `TestDisplayConfig_Validate_MaxHistoryLines` from `internal/display/types_test.go` [P]
- [X] Task 3.3: Remove `TestDisplayConfig_Validate_AnimationType` from `internal/display/types_test.go` [P]
- [X] Task 3.4: Remove `TestDisplayConfig_Validate_AnimationDisabled` from `internal/display/types_test.go` [P]
- [X] Task 3.5: Remove `TestAnimationType_Constants` from `internal/display/types_test.go` — NO, keep this; AnimationType type is still active [P]
- [X] Task 3.6: Remove `TestDashboard_ShouldUseCompactMode` from `internal/display/dashboard_test.go` [P]
- [X] Task 3.7: Update `BenchmarkDisplayConfig_Validate` in `internal/display/helpers_test.go` to remove `MaxHistoryLines` and fix `AnimationType` reference [P]
- [X] Task 3.8: Remove `MaxHistoryLines` assertion from `TestGetOptimalDisplayConfig` in `internal/display/capability_test.go` [P]
- [X] Task 3.9: Update `TestDefaultDisplayConfig` in `tests/unit/display/progress_test.go` — remove assertions for dead fields [P]
- [X] Task 3.10: Update `TestDisplayConfigValidation` in `tests/unit/display/progress_test.go` — remove `MaxHistoryLines`, `AnimationType`, `AnimationEnabled` cases [P]
- [X] Task 3.11: Update `TestResponsiveLayout` in `tests/unit/display/dashboard_test.go` — remove `CompactMode` references [P]
- [X] Task 3.12: Update config validation tests that set `MaxHistoryLines` as part of struct construction (e.g. `TestDisplayConfig_Validate_ColorMode`, `TestDisplayConfig_Validate_ColorTheme`) in `internal/display/types_test.go` [P]
- [X] Task 3.13: Remove `TestDashboard_RenderPanels` project info panel subtest from `internal/display/dashboard_test.go` [P]

## Phase 4: Validation
- [X] Task 4.1: Run `go build ./...` to verify compilation
- [X] Task 4.2: Run `go test ./internal/display/...` to verify display tests pass
- [X] Task 4.3: Run `go test ./tests/unit/display/...` to verify external display tests pass
- [X] Task 4.4: Run `go test ./tests/integration/...` to verify integration tests pass
- [X] Task 4.5: Run `go vet ./internal/display/...` to verify no static analysis issues
