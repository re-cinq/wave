# cleanup: Remove or wire non-functional DisplayConfig fields (consolidate with #61)

**Issue**: [#66](https://github.com/re-cinq/wave/issues/66)
**Labels**: good first issue, cleanup, priority: low
**Author**: nextlevelshit
**Complexity**: simple

## Summary

Multiple fields in `DisplayConfig` (`internal/display/types.go`) are defined and validated but never consulted during rendering. Users/callers who set these fields get no effect. Related issue #61 (dead functions) is already closed, so this PR focuses on the unused config fields and any remaining dead dashboard methods.

## Unused Fields

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

## Acceptance Criteria

- [x] Remove unused `DisplayConfig` fields listed above from `internal/display/types.go`
- [x] Remove associated helper methods (`ShouldUseCompactMode()`)
- [x] Remove dead dashboard methods never called from production code (`RenderPerformanceMetricsPanel`, `RenderPerformanceComparison`, `renderProjectInfoPanel`, `renderCurrentAction`, `RenderCompact`)
- [x] Remove validation code for removed fields from `Validate()`
- [x] Update `DefaultDisplayConfig()` and `GetOptimalDisplayConfig()` to not set removed fields
- [x] Update all tests referencing removed fields
- [x] Ensure all display tests pass after removal
- [x] Verify dashboard rendering is unaffected (these fields were never consulted)

## Important Scope Note

The `AnimationType` **type** and its **constants** (`AnimationDots`, `AnimationSpinner`, etc.) are actively used by the animation system (`Spinner`, `NewSpinner`, `getAnimationFrames`, `SelectAnimationType`). Only the `DisplayConfig.AnimationType` **field** should be removed — the type definition and constants must be preserved.
