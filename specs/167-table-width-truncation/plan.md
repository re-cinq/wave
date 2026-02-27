# Implementation Plan: Fix Table Width Truncation (#167)

## Objective

Fix `wave list runs` and all other `wave list` subcommands to use dynamic terminal width detection instead of hardcoded column widths and separator lengths. Ensure run IDs are never truncated when terminal width permits.

## Approach

### Strategy: Centralized Table Width Helper + Per-Table Updates

Rather than duplicating terminal width detection in each table function, introduce a small shared utility in `internal/display/` that provides terminal-width-aware table rendering helpers. Then update each table function in `cmd/wave/commands/list.go` and `cmd/wave/commands/status.go` to use dynamic widths.

### Design Principles

1. **ID columns are sacred** — never truncate run IDs unless the terminal is extremely narrow (< 60 cols)
2. **Separator width = min(termWidth, contentWidth)** — separators adapt to actual terminal width
3. **Flexible columns compress first** — description/pipeline/duration columns shrink before ID columns
4. **Reuse existing infrastructure** — `display.getTerminalWidth()` already handles TTY detection, `COLUMNS` env fallback, and defaults to 80

## File Mapping

### Files to Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/display/terminal.go` | modify | Export `GetTerminalWidth()` as a package-level function (currently unexported `getTerminalWidth()`) |
| `internal/display/formatter.go` | modify | Add `SeparatorLine(width int)` convenience method to Formatter |
| `cmd/wave/commands/list.go` | modify | Update all 6 table functions to use dynamic terminal width |
| `cmd/wave/commands/status.go` | modify | Update `outputRuns()` to use dynamic column widths |
| `cmd/wave/commands/list_test.go` | modify | Add tests for dynamic width behavior |

### Files NOT to Change

- JSON output paths (already correct, no table rendering involved)
- `cmd/wave/commands/chat.go` — `listRecentRunsForChat` delegates to `outputRuns` in `status.go` which will be fixed

## Architecture Decisions

### AD-1: Export getTerminalWidth vs. use NewTerminalInfo

**Decision**: Export `getTerminalWidth()` as `GetTerminalWidth()` in `internal/display/terminal.go`.

**Rationale**: `NewTerminalInfo()` creates a full capability detection object which is overkill for just getting the width. The `getTerminalWidth()` function is already well-implemented with TTY detection + COLUMNS env fallback + default 80. Exporting it keeps the call sites clean: `display.GetTerminalWidth()`.

### AD-2: Column width calculation approach

**Decision**: Use a simple proportional allocation with minimum widths and priority ordering.

**Approach for `listRunsTable`**:
- Get terminal width via `display.GetTerminalWidth()`
- Define minimum column widths: ID=8, Pipeline=10, Status=12, Started=20, Duration=8
- Allocate remaining width to ID and Pipeline columns (ID first)
- Cap separator at terminal width

**Approach for other tables** (pipelines, personas, adapters, contracts, skills):
- These use a card-style layout, not columnar tables
- Only the separator line (`strings.Repeat("─", 60)`) needs updating to `strings.Repeat("─", min(termWidth, maxWidth))`

### AD-3: Separator width capping

**Decision**: Cap separator width at terminal width, with a minimum of 40 columns.

**Rationale**: Prevents separator from extending beyond the visible terminal area. The 40-column minimum ensures the separator is always visible even in very narrow terminals.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| ANSI escape codes affect visible width calculation | Column alignment breaks if ANSI codes counted in width | Use raw text length for width calculation, apply color after padding |
| Tests assume specific hardcoded widths | Test failures | Update tests to be width-agnostic (check content presence, not exact formatting) |
| Non-TTY environments (CI, pipes) | Width detection fails | Already handled: `getTerminalWidth()` falls back to COLUMNS env then 80 |

## Testing Strategy

### Unit Tests

1. **Test dynamic separator width**: Verify separator adapts when `COLUMNS` env var is set
2. **Test column width calculation**: Verify ID column gets priority allocation
3. **Test no-truncation at wide terminals**: Verify full IDs displayed at 120+ columns
4. **Test narrow terminal graceful degradation**: Verify output doesn't crash at 60 columns

### Integration Tests

1. **Existing test suite**: Run `go test ./...` — all existing tests must pass
2. **Test with race detector**: `go test -race ./...`

### Manual Verification

1. Resize terminal to 80, 120, 160 columns and run `wave list runs`
2. Verify all `wave list` subcommands show consistent separator widths
