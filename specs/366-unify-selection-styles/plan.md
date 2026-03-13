# Implementation Plan: Unify Selection Highlighting

## Objective

Replace cyan foreground-only selection highlighting with background-based selection styles across all TUI list components, with focus-aware active/inactive distinction. Remove caret/arrow selection indicators.

## Approach

### Strategy: Centralized Style Constants + Per-Component Updates

1. **Define centralized selection styles in `theme.go`** — add exported style variables for active selection (white bg/dark fg), inactive selection (dimmed bg matching border `Color("240")`), and remove the caret-based prefix pattern.

2. **Update each list component** — replace the per-item `Foreground(lipgloss.Color("6"))` selected style with the new background-based style applied to the full-width rendered line. Remove `›`, `▶`, `▸`, `>` selection prefixes, replacing them with consistent spacing.

3. **Pass focus state into rendering** — list models already have a `focused` field set via `SetFocused()`. The render methods need to use `focused` to choose between active and inactive selection styles.

4. **Remove pane-level `Faint(true)` for left pane** — currently `content.go` applies `Faint(true)` to the entire unfocused pane. With per-item inactive styles, this blanket faint can remain for the non-selected items but the selected item should use the explicit inactive selection style. Actually, keep `Faint(true)` as-is — it provides the right dimming for non-selected items. The selected item's explicit inactive style (dim background) will override the faint effect visually.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/theme.go` | modify | Add `ActiveSelectionStyle`, `InactiveSelectionStyle` lipgloss styles |
| `internal/tui/pipeline_list.go` | modify | Replace cyan fg + `›`/`▶`/`▼` prefixes with bg-based selection in `renderPipelineName`, `renderRunningItem`, `renderFinishedItem` |
| `internal/tui/issue_list.go` | modify | Replace cyan fg + `›`/`▶`/`▼` prefixes with bg-based selection in `renderIssueLine`, `renderRunningChild`, `renderFinishedChild` |
| `internal/tui/compose_list.go` | modify | Replace cyan fg `▸` cursor with bg-based selection in `View()` |
| `internal/tui/persona_list.go` | modify | Replace cyan fg `▶` prefix with bg-based selection in `View()` |
| `internal/tui/skill_list.go` | modify | Replace cyan fg `▶` prefix with bg-based selection in `View()` |
| `internal/tui/health_list.go` | modify | Replace cyan fg `▶` prefix with bg-based selection in `View()` |
| `internal/tui/contract_list.go` | modify | Replace cyan fg `▶` prefix with bg-based selection in `View()` |
| `internal/tui/suggest_list.go` | modify | Replace `> ` cursor + cyan fg + `Color("12")` with bg-based selection in `View()` |

## Architecture Decisions

1. **Style location**: New styles defined as package-level functions in `theme.go` rather than inline in each component. This provides a single source of truth.

2. **Background colors**:
   - Active selection: `Background(lipgloss.Color("7"))` (white) + `Foreground(lipgloss.Color("0"))` (black) — high contrast.
   - Inactive selection: `Background(lipgloss.Color("240"))` (same gray as border separator) + `Foreground(lipgloss.Color("7"))` (white) — dimmed but visible.

3. **Prefix replacement**: Replace all selection indicator characters (`›`, `▶`, `▸`, `>`) with plain spaces (`  `) for uniform alignment. Keep non-selection-related indicators (tree connectors `├`, `└`, collapse `▶`/`▼` for tree nodes) intact.

4. **Width handling**: Background-based styles need full-width rendering. Most components already use `.Width(m.width)` — ensure all selected lines use this to get a consistent highlight bar.

5. **WaveTheme (huh forms) unchanged**: The huh form picker theme (`WaveTheme()`) is separate from list selection and uses cyan appropriately for form elements (cursors, borders, buttons). This is not part of the selection highlighting issue.

6. **Keep cyan for non-selection uses**: Cyan remains for the logo, header accent, detail pane section titles, and huh form elements — the issue specifically targets selection/hover highlighting only.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Tree connector indicators (`▶`/`▼` for collapse) confused with selection | Medium | Only remove selection-specific prefixes; tree node collapse indicators stay |
| Background highlight clashes with ANSI color status icons (green ●, red ✗) | Low | Status icons use foreground colors that are visible against both white and gray backgrounds |
| Tests break due to changed rendering output | Low | No existing tests assert on rendered selection output (verified via grep) |
| Wide-character or emoji rendering breaks background alignment | Low | Already handled by `truncateName` and `lipgloss.Width()` usage |

## Testing Strategy

1. **Manual visual testing**: Primary validation — run `wave tui` and verify:
   - Active pane selection has white bg / dark fg
   - Switching panes dims left selection to border-matching gray
   - Switching back restores full intensity
   - No caret indicators remain
   - All 8 list types render consistently

2. **Existing test suite**: Run `go test ./internal/tui/...` to ensure no regressions in:
   - Pipeline list navigation tests
   - Content model focus switching tests
   - Compose list state tests
   - Issue list navigation tests

3. **Build verification**: `go build ./...` to ensure no compilation errors

4. **Race detector**: `go test -race ./internal/tui/...` for concurrency safety
