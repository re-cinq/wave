# Tasks

## Phase 1: Infrastructure — Export Terminal Width

- [X] Task 1.1: Export `getTerminalWidth()` as `GetTerminalWidth()` in `internal/display/terminal.go`
  - Add exported wrapper function that calls the existing unexported `getTerminalWidth()`
  - File: `internal/display/terminal.go` (modify)

- [X] Task 1.2: Add `GetTerminalHeight()` exported wrapper for consistency
  - File: `internal/display/terminal.go` (modify)

## Phase 2: Core Fix — Update `listRunsTable` with Dynamic Width

- [X] Task 2.1: Refactor `listRunsTable` in `cmd/wave/commands/list.go` to use dynamic terminal width
  - Replace hardcoded `strings.Repeat("─", 100)` separator with `display.GetTerminalWidth()`-aware separator
  - Replace hardcoded `%-30s`, `%-22s`, `%-12s`, `%-20s` column format strings with dynamically calculated widths
  - Remove the `if len(runID) > 30 { runID = runID[:27] + "..." }` truncation — let width calculation handle this
  - Remove the `if len(pipeline) > 22 { pipeline = pipeline[:19] + "..." }` truncation
  - Implement column width allocation: fixed widths for Status (12), Started (20), Duration (10); remainder split between RunID (priority) and Pipeline
  - File: `cmd/wave/commands/list.go:1118-1174` (modify)

## Phase 3: Consistency — Update Other Table Separators [P]

- [X] Task 3.1: Update `listPipelinesTable` separator to use terminal width [P]
  - Replace `strings.Repeat("─", 60)` at line 451 with `display.GetTerminalWidth()`-capped separator
  - File: `cmd/wave/commands/list.go:451` (modify)

- [X] Task 3.2: Update `listPersonasTable` separator to use terminal width [P]
  - Replace `strings.Repeat("─", 60)` at line 566 with dynamic separator
  - File: `cmd/wave/commands/list.go:566` (modify)

- [X] Task 3.3: Update `listAdaptersTable` separator to use terminal width [P]
  - Replace `strings.Repeat("─", 60)` at line 642 with dynamic separator
  - File: `cmd/wave/commands/list.go:642` (modify)

- [X] Task 3.4: Update `listContractsTable` separator to use terminal width [P]
  - Replace `strings.Repeat("─", 60)` at line 1294 with dynamic separator
  - File: `cmd/wave/commands/list.go:1294` (modify)

- [X] Task 3.5: Update `listSkillsTable` separator to use terminal width [P]
  - Replace `strings.Repeat("─", 60)` at line 1426 with dynamic separator
  - File: `cmd/wave/commands/list.go:1426` (modify)

## Phase 4: Fix Status Command Tables

- [X] Task 4.1: Update `outputRuns` in `cmd/wave/commands/status.go` to use dynamic column widths
  - Replace hardcoded `%-26s`, `%-15s`, `%-12s`, `%-15s`, `%-10s` format strings with dynamic widths
  - Remove `truncateString(run.RunID, 26)` — prioritize showing full run ID
  - File: `cmd/wave/commands/status.go:213-244` (modify)

## Phase 5: Testing

- [X] Task 5.1: Add unit test for dynamic separator width in `listRunsTable`
  - Set `COLUMNS=120` env var, run list, verify separator adapts
  - File: `cmd/wave/commands/list_test.go` (modify)

- [X] Task 5.2: Add unit test verifying run IDs are not truncated at wide terminal
  - Insert run with long ID, set `COLUMNS=160`, verify full ID in output
  - File: `cmd/wave/commands/list_test.go` (modify)

- [X] Task 5.3: Run full test suite `go test ./...`
  - Verify no regressions across all packages
  - Fix any tests that depend on hardcoded column widths

- [X] Task 5.4: Run race detector `go test -race ./...`

## Phase 6: Polish

- [X] Task 6.1: Verify `wave list` (no filter) shows all sections with consistent separators
- [X] Task 6.2: Verify JSON output is unaffected by changes
