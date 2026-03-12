# Tasks

## Phase 1: Remove fields and validation from types.go

- [X] Task 1.1: Remove 8 unused fields from `DisplayConfig` struct (`ShowDetails`, `ShowArtifacts`, `CompactMode`, `MaxHistoryLines`, `EnableTimestamps`, `AnimationType`, `ShowLogo`, `ShowMetrics`)
- [X] Task 1.2: Remove `MaxHistoryLines` validation block from `Validate()`
- [X] Task 1.3: Remove `AnimationType` validation block from `Validate()` (the `validAnimations` map and the check)
- [X] Task 1.4: Remove `AnimationEnabled` → `AnimationDots` override from `Validate()` (references removed `AnimationType` field)
- [X] Task 1.5: Update `DefaultDisplayConfig()` to not set removed fields

## Phase 2: Remove dead dashboard methods [P]

- [X] Task 2.1: Remove `ShouldUseCompactMode()` from `dashboard.go` [P]
- [X] Task 2.2: Remove `RenderCompact()` from `dashboard.go` [P]
- [X] Task 2.3: Remove `RenderPerformanceMetricsPanel()` from `dashboard.go` [P]
- [X] Task 2.4: Remove `RenderPerformanceComparison()` from `dashboard.go` [P]
- [X] Task 2.5: Remove `renderProjectInfoPanel()` from `dashboard.go` [P]
- [X] Task 2.6: Remove `renderCurrentAction()` from `dashboard.go` [P]

## Phase 3: Update capability.go

- [X] Task 3.1: Update `GetOptimalDisplayConfig()` in `capability.go` to not set removed fields (`ShowDetails`, `ShowArtifacts`, `CompactMode`, `MaxHistoryLines`, `EnableTimestamps`, `AnimationType`)

## Phase 4: Update internal display tests [P]

- [X] Task 4.1: Update `types_test.go` — remove assertions for removed fields in `TestDefaultDisplayConfig`, remove `TestDisplayConfig_Validate_MaxHistoryLines`, remove `TestDisplayConfig_Validate_AnimationType`, remove `TestDisplayConfig_Validate_AnimationDisabled`, update struct literals in remaining validation tests [P]
- [X] Task 4.2: Update `dashboard_test.go` — remove `TestDashboard_ShouldUseCompactMode`, remove `renderProjectInfoPanel` test in `TestDashboard_RenderPanels` [P]
- [X] Task 4.3: Update `capability_test.go` — remove `MaxHistoryLines` assertion in `TestGetOptimalDisplayConfig` [P]
- [X] Task 4.4: Update `helpers_test.go` — remove `MaxHistoryLines` and `AnimationType` from `BenchmarkDisplayConfig_Validate` struct literal [P]

## Phase 5: Update external display tests [P]

- [X] Task 5.1: Update `tests/unit/display/progress_test.go` — remove assertions for removed fields in `TestDefaultDisplayConfig`, update validation test struct literals [P]
- [X] Task 5.2: Update `tests/unit/display/dashboard_test.go` — remove `CompactMode` usage in `TestResponsiveLayout` [P]

## Phase 6: Validation

- [X] Task 6.1: Run `go vet ./internal/display/...` to verify compilation
- [X] Task 6.2: Run `go test ./internal/display/...` to verify internal tests pass
- [X] Task 6.3: Run `go test ./tests/unit/display/...` to verify external tests pass
- [X] Task 6.4: Run `go test ./...` to verify full test suite passes
