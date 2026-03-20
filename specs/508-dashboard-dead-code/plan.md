# Implementation Plan: Dashboard Dead Code Removal

## Objective

Remove dead code from `internal/display/dashboard.go` and `internal/display/types.go` that was re-introduced after PR #353's cleanup, and update corresponding tests.

## Approach

Systematic codebase-wide grep analysis identified the following dead code:

### Dead Code in `dashboard.go`

1. **`clearPreviousRender()`** (lines 71-80) — Method on `*Dashboard` that clears previous render output. Never called from any production code. The `Render()` method explicitly comments "No clearing at all" and the `Clear()` method is a no-op. This method is vestigial from a previous rendering approach.

2. **`formatDashboardDuration()`** (lines 350-366) — Unexported function that formats milliseconds to human-readable duration. Never called from any production code. The dashboard uses `fmt.Sprintf("%.1fs", ...)` directly in `formatElapsedInfo()` instead. Only exercised by `TestFormatDashboardDuration` in dashboard_test.go.

3. **`NewDashboardWithConfig()`** (lines 31-38) — Exported constructor that accepts `colorMode` and `asciiOnly` parameters. Never called from any file outside `dashboard.go` — not even from tests. The only production caller is `progress.go:274` which uses `NewDashboard()` (the zero-config constructor).

### Dead Code in `types.go`

4. **`ProgressRenderer` interface** (lines 99-109) — Interface defining `Render`, `Clear`, `Close` methods accepting `*PipelineProgress`. Never implemented or used in production code. Only referenced in `types_test.go` for interface compliance testing with a mock. The actual rendering uses `Dashboard.Render(*PipelineContext)` which has a completely different signature.

### Alive Code (Confirmed NOT Dead)

- `Dashboard` struct, `NewDashboard()`, `Render()`, `Clear()` — used by `progress.go`
- `DisplayConfig`, `DefaultDisplayConfig()`, `Validate()` — used by capability.go, progress tests, integration tests
- `AnimationType` constants — used by animation.go, capability.go, progress.go
- `ColorPalette`, all color scheme vars, `GetColorSchemeByName()` — used by capability.go, tests
- `TerminalCapabilities` — used by terminal.go, capability.go
- `StepProgress`, `PipelineProgress` — used across pipeline, event, state, TUI packages
- `HandoverInfo`, `PipelineContext` — used by progress.go, bubbletea_progress.go, TUI

## File Mapping

| File | Action | Details |
|------|--------|---------|
| `internal/display/dashboard.go` | modify | Remove `clearPreviousRender`, `formatDashboardDuration`, `NewDashboardWithConfig` |
| `internal/display/dashboard_test.go` | modify | Remove `TestFormatDashboardDuration` test |
| `internal/display/types.go` | modify | Remove `ProgressRenderer` interface |
| `internal/display/types_test.go` | modify | Remove `TestProgressRenderer_Interface` test and `mockProgressRenderer` |

## Architecture Decisions

- **No removal of `dashboard.go` itself**: The file contains actively-used code (`Dashboard` struct, `Render`, rendering methods). Only specific dead methods are removed.
- **No removal of `DisplayConfig`**: All 6 occurrences in types.go are actively used (struct definition, DefaultDisplayConfig, Validate, field definitions).
- **Test removal is limited**: Only tests that exclusively test dead code are removed. Tests that cover live functionality are preserved.

## Risks

| Risk | Mitigation |
|------|-----------|
| Removing code that's used via reflection or code generation | Grep verified no dynamic references; Go doesn't support method-name-based reflection calls |
| Breaking external consumers | This is an internal package — no external API contract |
| Missing a caller in a build tag-gated file | Searched all `*.go` files regardless of build tags |

## Testing Strategy

- Run `go build ./...` to verify compilation after removals
- Run `go test ./internal/display/...` to verify display package tests pass
- Run `go test ./...` to verify no downstream breakage
- Run `go vet ./...` to verify no new warnings
