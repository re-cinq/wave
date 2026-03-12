# Implementation Plan: Green Color Scheme

## Objective

Replace the cyan/teal primary accent color with green across the TUI, display package, and documentation CSS to fulfill the original requirement from issue #301.

## Approach

The color system has two layers:

1. **TUI layer** (`internal/tui/`): Uses lipgloss `Color("6")` (ANSI cyan). The `theme.go` file defines a `cyan` variable used by the huh theme, but ~20 other TUI files hardcode `Color("6")` directly. The fix is to (a) change the color value in `theme.go`, (b) export it as a package-level constant, and (c) update all hardcoded references to use the constant.

2. **Display layer** (`internal/display/`): Uses ANSI escape codes (`\033[36m` = cyan) for the `Primary` color in `ColorScheme` palettes, plus lipgloss `Color("14")` (bright cyan) in the Bubble Tea model. These need to switch to green equivalents.

3. **Docs CSS** (`docs/.vitepress/theme/styles/custom.css`): The `--wave-secondary` variable is `#06b6d4` (cyan). This should change to a green value consistent with the existing `--wave-trust-green` (`#10b981`).

### Color Mapping

| Context | Old (Cyan) | New (Green) |
|---------|-----------|-------------|
| TUI lipgloss | `Color("6")` (ANSI cyan) | `Color("2")` (ANSI green) |
| TUI lipgloss bright | `Color("14")` (bright cyan) | `Color("10")` (bright green) |
| Display ANSI | `\033[36m` | `\033[32m` |
| Display ANSI bold | `\033[1;36m` | `\033[1;32m` |
| Docs CSS secondary | `#06b6d4` | `#10b981` |
| Docs CSS dark secondary | `#22d3ee` | `#34d399` |

## File Mapping

### Modified files

| File | Change |
|------|--------|
| `internal/tui/theme.go` | Rename `cyan` to `green`, change `Color("6")` to `Color("2")`, update comments |
| `internal/tui/header_logo.go` | Change `Color("6")` to `Color("2")` (2 locations) |
| `internal/tui/header.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/compose_list.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/compose_detail.go` | Change `Color("6")` to `Color("2")` (3 locations) |
| `internal/tui/persona_list.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/persona_detail.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/run_selector.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/health_list.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/health_detail.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/contract_list.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/contract_detail.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/live_output.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/issue_list.go` | Change `Color("6")` to `Color("2")` (3 locations) |
| `internal/tui/issue_detail.go` | Change `Color("6")` to `Color("2")` (2 locations) |
| `internal/tui/skill_list.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/skill_detail.go` | Change `Color("6")` to `Color("2")` |
| `internal/tui/pipeline_list.go` | Change `Color("6")` to `Color("2")` (3 locations) |
| `internal/tui/pipeline_detail.go` | Change `Color("6")` to `Color("2")` (3 locations) |
| `internal/display/types.go` | Change Primary from `\033[36m` to `\033[32m` in color schemes |
| `internal/display/bubbletea_model.go` | Change `Color("14")` to `Color("10")` |
| `internal/display/dashboard.go` | Change `Color("14")` to `Color("10")`, update comment |
| `docs/.vitepress/theme/styles/custom.css` | Change `--wave-secondary` to green |

## Architecture Decisions

1. **Use ANSI Color "2" (standard green)** rather than a hex green — keeps consistency with the ANSI color palette approach already used throughout the codebase. The exact shade depends on the user's terminal theme, which is the existing convention.

2. **Direct replacement** of `Color("6")` with `Color("2")` in all TUI files rather than centralizing into a constant — the assessment skips refactoring steps and this is a simple audit fix. Each file's hardcoded color is replaced in-place.

3. **Update display package color schemes** to use green ANSI codes — these define the CLI progress display colors which should be consistent with the TUI.

4. **Update docs CSS `--wave-secondary`** from cyan (`#06b6d4`) to emerald green (`#10b981`) — this matches the existing `--wave-trust-green` value and is a well-known Tailwind emerald-500 that works in both light and dark modes.

## Risks

| Risk | Mitigation |
|------|-----------|
| Missed hardcoded cyan references | Comprehensive grep for `Color("6")`, `Color("14")`, `\033[36m` across codebase |
| Green color poor readability on some terminals | ANSI Color 2 is universally supported and green is high-contrast on both dark and light backgrounds |
| Breaking test assertions that check for cyan ANSI codes | Run `go test ./...` and fix any tests comparing against old color values |
| Dark mode docs CSS readability | Use `#34d399` (emerald-300) for dark mode variant, good contrast on dark backgrounds |

## Testing Strategy

1. Run `go test ./...` after all changes to verify no test regressions
2. Run `go test -race ./...` for race condition checking
3. Grep for any remaining `Color("6")`, `Color("14")`, or `\033[36m` references to confirm complete replacement
4. No new tests needed — this is a cosmetic color swap with no behavioral changes
