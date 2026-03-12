# cleanup: Remove or wire non-functional DisplayConfig fields (consolidate with #61)

**Issue**: [#66](https://github.com/re-cinq/wave/issues/66)
**Author**: nextlevelshit
**Labels**: good first issue, cleanup, priority: low
**State**: OPEN

## Summary

Multiple fields in `DisplayConfig` (`internal/display/types.go`) are defined and validated but never consulted during rendering. Users/callers who set these fields get no effect. This issue should be addressed alongside #61 (dead ETA/metrics code in the display package) as a single cleanup effort.

**Note**: Issue #61 is already closed, so the scope narrows to config field removal only.

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

## Recommendation: Remove

Given that these fields have never been wired in since their introduction, and no downstream consumers depend on them, the cleanest path is **removal**. If any of these features are needed later, they can be re-added with proper rendering integration from the start.

## Acceptance Criteria

- [ ] Remove unused `DisplayConfig` fields listed above from `internal/display/types.go`
- [ ] Remove associated constants (`AnimationType` variants) and helper methods (`ShouldUseCompactMode()`, `getAnimationFrames()`)
- [ ] Remove dead functions from #61 in the same PR (ETA calculation, performance metrics panel, performance comparison)
- [ ] Ensure all display tests pass after removal
- [ ] Verify dashboard rendering is unaffected (these fields were never consulted)

Discovered during #56 audit. See also #61.
