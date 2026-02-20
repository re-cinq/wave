# Tasks

## Phase 1: Core Type Cleanup
- [X] Task 1.1: Remove dead fields from `DisplayConfig` struct in `types.go` (ShowDetails, ShowArtifacts, CompactMode, MaxHistoryLines, EnableTimestamps, ShowLogo, ShowMetrics, AnimationType field)
- [X] Task 1.2: Remove `EstimatedTimeMs` and `AverageStepTimeMs` fields from `PipelineContext` struct in `types.go`
- [X] Task 1.3: Update `DefaultDisplayConfig()` in `types.go` to remove defaults for deleted fields
- [X] Task 1.4: Simplify `Validate()` in `types.go` to remove validation for `MaxHistoryLines`, `AnimationType`, and `AnimationEnabled` -> `AnimationType` override

## Phase 2: Dead Function Removal
- [X] Task 2.1: Remove `RenderPerformanceMetricsPanel()` from `dashboard.go` [P]
- [X] Task 2.2: Remove `RenderPerformanceComparison()` from `dashboard.go` [P]
- [X] Task 2.3: Remove `ShouldUseCompactMode()` from `dashboard.go` [P]
- [X] Task 2.4: Remove dead ETA conditional from `RenderCompact()` in `dashboard.go`

## Phase 3: Reference Cleanup
- [X] Task 3.1: Update `GetOptimalDisplayConfig()` in `capability.go` to remove deleted field assignments [P]
- [X] Task 3.2: Remove `EstimatedTimeMs: 0` from context creation in `bubbletea_progress.go` [P]
- [X] Task 3.3: Remove `EstimatedTimeMs: 0` from context creation in `progress.go` [P]

## Phase 4: Internal Test Updates
- [X] Task 4.1: Update `types_test.go` - remove assertions on deleted fields from `TestDefaultDisplayConfig`, `TestDisplayConfig_Validate_MaxHistoryLines`, `TestDisplayConfig_Validate_AnimationType`, `TestDisplayConfig_Validate_AnimationDisabled`, `TestPipelineContext_Structure`
- [X] Task 4.2: Update `dashboard_test.go` - remove `TestDashboard_ShouldUseCompactMode`, remove `EstimatedTimeMs`/`AverageStepTimeMs` from test contexts [P]
- [X] Task 4.3: Update `capability_test.go` - remove `MaxHistoryLines` assertion [P]
- [X] Task 4.4: Update `helpers_test.go` - remove `MaxHistoryLines`/`AnimationType` from validate benchmark [P]
- [X] Task 4.5: Update `progress_test.go` (internal) - no changes needed (no direct field references)

## Phase 5: External Test Updates
- [X] Task 5.1: Update `tests/unit/display/progress_test.go` - remove assertions on deleted DisplayConfig fields and PipelineContext fields
- [X] Task 5.2: Update `tests/unit/display/dashboard_test.go` - remove `CompactMode` reference, adjust `AnimationType` config field usage
- [X] Task 5.3: Update `tests/integration/progress_test.go` - remove `AverageStepTimeMs`/`EstimatedTimeMs` usage

## Phase 6: Validation
- [X] Task 6.1: Run `go build ./...` to verify compilation
- [X] Task 6.2: Run `go test ./internal/display/...` to verify display package tests
- [X] Task 6.3: Run `go test ./...` to verify full test suite
- [X] Task 6.4: Run `go test -race ./internal/display/...` to verify race safety
