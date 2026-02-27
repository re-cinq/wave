# Fix table width truncation in 'wave list runs' to preserve ID visibility and adapt to terminal width

**Issue**: [#167](https://github.com/re-cinq/wave/issues/167)
**Feature Branch**: `167-table-width-truncation`
**Labels**: bug, display
**Author**: nextlevelshit
**Status**: Open
**Complexity**: Medium

## Problem

`wave list runs` displays a table that truncates the run ID column, which is crucial for identifying and working with specific runs. The table layout does not adapt to terminal width, leading to important data being hidden.

## Current Behavior

- Run ID column is truncated in the output (hardcoded `%-30s` format, truncation at 30 chars)
- Table width is fixed and does not respond to terminal window dimensions (separator hardcoded to `strings.Repeat("─", 100)`)
- Other `wave list` commands (`wave list pipelines`, `wave list personas`, etc.) and `wave chat --list` have similar hardcoded separator widths (`strings.Repeat("─", 60)`)
- The `wave status` command also uses hardcoded column widths with `truncateString(run.RunID, 26)`

## Expected Behavior

- All columns, especially the ID column, should be fully visible without truncation
- Table should dynamically adapt to terminal width (using existing `display.getTerminalWidth()` / `display.NewTerminalInfo()`)
- Consistent behavior across all `wave list` variants and related listing commands
- ID columns are never truncated

## Scope

- [X] Fix `wave list runs` table width and column visibility
- [X] Implement dynamic terminal width detection (reference existing implementation in `internal/display/terminal.go`)
- [X] Review and fix `wave chat --list` and other `wave list` options for consistency
- [X] Ensure ID columns are never truncated

## User Scenarios & Testing

### User Story 1 - Run ID visibility at standard terminal widths (Priority: P1)

A developer runs `wave list runs` at a standard terminal width (120+ columns) and sees the full run ID without any truncation, allowing them to copy-paste the ID for `wave chat <run-id>`.

**Why this priority**: Without visible run IDs, users cannot interact with specific runs—this is the core bug.

**Independent Test**: Run `wave list runs` in a 120-column terminal and verify all run IDs are fully visible.

**Acceptance Scenarios**:

1. **Given** a terminal with 120 columns, **When** running `wave list runs`, **Then** all run ID values are fully displayed without truncation or ellipsis.
2. **Given** a terminal with 160 columns, **When** running `wave list runs`, **Then** table separator and columns expand to use available width.

---

### User Story 2 - Graceful degradation at narrow widths (Priority: P2)

A developer with a narrow terminal (80 columns) runs `wave list runs` and sees all columns, with non-ID columns compressed first before any ID truncation occurs.

**Why this priority**: While 80-column terminals are common, this is a secondary concern since most developers use wider terminals.

**Independent Test**: Run `wave list runs` in an 80-column terminal and verify the run ID column is prioritized over other columns.

**Acceptance Scenarios**:

1. **Given** a terminal with 80 columns, **When** running `wave list runs`, **Then** the run ID is displayed with maximum available width and non-essential columns are compressed.
2. **Given** a terminal with 80 columns, **When** running `wave list runs`, **Then** the separator line matches the actual terminal width rather than being hardcoded.

---

### User Story 3 - Consistent table rendering across all list commands (Priority: P3)

All `wave list` subcommands (`adapters`, `runs`, `pipelines`, `personas`, `contracts`, `skills`) and `wave chat --list` use consistent, terminal-width-aware table rendering.

**Why this priority**: Consistency is important for UX but secondary to fixing the core truncation bug.

**Independent Test**: Run each `wave list <subcommand>` and verify separator widths adapt to terminal width.

**Acceptance Scenarios**:

1. **Given** any terminal width, **When** running `wave list adapters`, **Then** the separator line width matches the terminal width (capped at a reasonable maximum).
2. **Given** any terminal width, **When** running `wave chat --list`, **Then** the table columns adapt to the terminal width.

---

### Edge Cases

- What happens when the terminal width is below 60 columns? Columns should still render without crashing; minimum viable widths should be enforced.
- What happens when `COLUMNS` env var is set but stdout is not a TTY? Should respect `COLUMNS` (already handled by `getTerminalWidth()`).
- What happens with extremely long run IDs (>50 chars)? Should be displayed fully up to a maximum, then truncated only as a last resort.

## Requirements

### Functional Requirements

- **FR-001**: `listRunsTable` MUST use dynamic terminal width from `display.getTerminalWidth()` instead of hardcoded column widths.
- **FR-002**: Run ID column MUST be displayed without truncation at terminal widths >= 100 columns.
- **FR-003**: Table separator lines MUST match the actual terminal width (or content width, whichever is smaller).
- **FR-004**: All `wave list` table functions (`listRunsTable`, `listPipelinesTable`, `listPersonasTable`, `listAdaptersTable`, `listContractsTable`, `listSkillsTable`) MUST use terminal-width-aware separators.
- **FR-005**: `wave status` / `wave chat --list` table rendering MUST use dynamic column widths consistent with `wave list runs`.
- **FR-006**: A shared table width calculation utility SHOULD be introduced to avoid duplicating width logic across commands.

### Key Entities

- **TerminalInfo**: Existing terminal capability detection (width, height, TTY status) in `internal/display/terminal.go`
- **Formatter**: Existing text formatting utility in `internal/display/formatter.go` — already has `TableRow()`, `Truncate()`, `HorizontalRule()`

## Success Criteria

### Measurable Outcomes

- **SC-001**: All run IDs render fully visible at >= 100 column terminal widths.
- **SC-002**: Separator lines match terminal width across all `wave list` subcommands.
- **SC-003**: All existing tests pass (`go test ./...`).
- **SC-004**: No regression in JSON output format (JSON output is unaffected by table width changes).
