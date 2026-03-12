# Implementation Plan: DisplayConfig Cleanup (#66)

## 1. Objective

Remove 8 unused `DisplayConfig` fields and their associated dead code (constants, helper methods, dead dashboard functions) from the `internal/display/` package. These fields are defined and validated but never consulted during rendering.

## 2. Approach

**Strategy**: Careful surgical removal in dependency order — remove field consumers first (dead methods, dead dashboard functions, dead tests), then remove the fields and constants themselves, then update `DefaultDisplayConfig()`, `Validate()`, and `GetOptimalDisplayConfig()` to stop referencing them.

**Key constraint**: `AnimationType` is NOT fully dead — it is used by `Spinner`, `getAnimationFrames()`, `NewSpinner()`, `MultiSpinner.Add()`, `SelectAnimationType()`, `animation.go`, and the bubbletea model's hardcoded spinner. The issue says the `DisplayConfig.AnimationType` *field* is dead (animation is hardcoded), but the `AnimationType` *type* and its constants are actively used by `animation.go`. Therefore we must **keep** `AnimationType` type/constants and `getAnimationFrames()`, and only remove the `AnimationType` field from `DisplayConfig`.

## 3. File Mapping

### Files to Modify

| File | Action | Changes |
|------|--------|---------|
| `internal/display/types.go` | modify | Remove 8 fields from `DisplayConfig`; remove their defaults from `DefaultDisplayConfig()`; remove `MaxHistoryLines` and `AnimationType` validation from `Validate()` |
| `internal/display/capability.go` | modify | Remove dead fields from `GetOptimalDisplayConfig()` return value |
| `internal/display/dashboard.go` | modify | Remove `ShouldUseCompactMode()`, `RenderPerformanceMetricsPanel()`, `RenderPerformanceComparison()`, `renderProjectInfoPanel()`, `renderCurrentAction()` dead methods |
| `internal/display/types_test.go` | modify | Remove tests for dead fields (`TestDefaultDisplayConfig` assertions, `TestDisplayConfig_Validate_MaxHistoryLines`, `TestDisplayConfig_Validate_AnimationType`, `TestDisplayConfig_Validate_AnimationDisabled`) |
| `internal/display/dashboard_test.go` | modify | Remove `TestDashboard_ShouldUseCompactMode` test |
| `internal/display/helpers_test.go` | modify | Update `BenchmarkDisplayConfig_Validate` to remove dead field references |
| `internal/display/capability_test.go` | modify | Remove `MaxHistoryLines` assertion from `TestGetOptimalDisplayConfig` |
| `tests/unit/display/progress_test.go` | modify | Remove assertions for dead fields in `TestDefaultDisplayConfig`, update `TestDisplayConfigValidation` cases |
| `tests/unit/display/dashboard_test.go` | modify | Update `TestResponsiveLayout` to remove `CompactMode` references |

### Files NOT to Modify

- `internal/display/animation.go` — `AnimationType` type, constants, and `getAnimationFrames()` are actively used
- `internal/display/animation_test.go` — Tests for active animation code
- `internal/display/bubbletea_model.go` — No references to dead fields
- `internal/display/bubbletea_progress.go` — No references to dead fields
- `internal/display/metrics.go` — `PerformanceMetrics` struct is separate from `DisplayConfig`; used by state store. **Keep**.
- `internal/state/store.go` — `PerformanceMetricRecord` is a different concept entirely

## 4. Architecture Decisions

1. **Keep `AnimationType` type and constants** — They are actively used by `Spinner`, `NewSpinner()`, `MultiSpinner.Add()`, `SelectAnimationType()`, and throughout `animation.go`. Only the `DisplayConfig.AnimationType` *field* is dead.

2. **Keep `getAnimationFrames()`** — Called by `NewSpinner()` in `animation.go`. Not dead code.

3. **Keep `metrics.go` entirely** — The `PerformanceMetrics` struct tracks rendering performance overhead and is referenced by the state store's `PerformanceMetricRecord`. It is not a "display config" field.

4. **Remove `ShouldUseCompactMode()`** — Only method on `Dashboard` that references terminal size for compact decisions but is never called.

5. **Remove `RenderPerformanceMetricsPanel()` and `RenderPerformanceComparison()`** — Dashboard methods that exist but are never called from any rendering path.

6. **Remove `renderProjectInfoPanel()` and `renderCurrentAction()`** — Private dashboard methods that are never called from `Render()` or `RenderCompact()`.

7. **Keep `AnimationEnabled` field** — It IS wired: `Validate()` sets `AnimationType` to `AnimationDots` when disabled, and this affects `NewSpinner()` behavior. However, looking more carefully: `AnimationEnabled` is set in `DisplayConfig` but `DisplayConfig` is never consulted by the rendering pipeline — `bubbletea_model.go` hardcodes its own spinner. So `AnimationEnabled` is also dead. Remove it but keep the animation-disabled validation test pattern if `AnimationType` field is removed.

8. **`AnimationEnabled` is also dead** — The `Validate()` method uses it to override `AnimationType`, but since neither field is consulted by the rendering pipeline, both are dead. Remove both.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Removing `AnimationType` field breaks `SelectAnimationType()` | `SelectAnimationType()` takes `AnimationType` as a parameter and returns one — it doesn't read from `DisplayConfig`. Safe. |
| External consumers reference `DisplayConfig` fields | Grep confirms no references outside `internal/display/` and tests. Safe. |
| `GetOptimalDisplayConfig()` callers break | Only called in tests. Return value just loses dead fields. |
| Test failures from removed assertions | Update all test files to remove assertions on dead fields |
| `renderProjectInfoPanel` / `renderCurrentAction` removal breaks compilation | Only called from test, not from `Render()`. Confirm with grep. |

## 6. Testing Strategy

1. **Remove dead test assertions** — tests that check `ShowDetails`, `ShowArtifacts`, `CompactMode`, `MaxHistoryLines`, `EnableTimestamps`, `ShowLogo`, `ShowMetrics`, `AnimationType` (on DisplayConfig), `AnimationEnabled`
2. **Update validation tests** — Remove `TestDisplayConfig_Validate_MaxHistoryLines`, `TestDisplayConfig_Validate_AnimationType`, `TestDisplayConfig_Validate_AnimationDisabled`; update benchmark
3. **Run `go test ./internal/display/...`** — Verify all display package tests pass
4. **Run `go test ./tests/unit/display/...`** — Verify external display tests pass
5. **Run `go test ./tests/integration/...`** — Verify integration tests pass
6. **Run `go test ./...`** — Full test suite to catch any cross-package breakage
