# Tasks

## Phase 1: Define Centralized Selection Styles
- [X] Task 1.1: Add `ActiveSelectionStyle()` and `InactiveSelectionStyle()` functions to `internal/tui/theme.go` returning lipgloss styles (white bg/dark fg for active, gray-240 bg/white fg for inactive)

## Phase 2: Update List Components
- [X] Task 2.1: Update `pipeline_list.go` — replace cyan foreground + `›` prefix in `renderPipelineName` with background selection style (active/inactive based on `m.focused`); replace cyan foreground in `renderRunningItem` and `renderFinishedItem`; keep tree connectors (`├`, `└`) and collapse indicators (`▶`/`▼`) for tree nodes [P]
- [X] Task 2.2: Update `issue_list.go` — replace cyan foreground + `›`/`▶`/`▼` prefix in `renderIssueLine` with background selection style; replace cyan foreground in `renderRunningChild` and `renderFinishedChild`; keep tree connectors [P]
- [X] Task 2.3: Update `compose_list.go` — replace cyan foreground `▸` cursor style with background selection style in `View()` [P]
- [X] Task 2.4: Update `persona_list.go` — replace cyan foreground `▶` prefix with background selection style in `View()` [P]
- [X] Task 2.5: Update `skill_list.go` — replace cyan foreground `▶` prefix with background selection style in `View()` [P]
- [X] Task 2.6: Update `health_list.go` — replace cyan foreground `▶` prefix with background selection style in `View()` [P]
- [X] Task 2.7: Update `contract_list.go` — replace cyan foreground `▶` prefix with background selection style in `View()` [P]
- [X] Task 2.8: Update `suggest_list.go` — replace `> ` cursor + `Color("12")` bold with background selection style in `View()` [P]

## Phase 3: Validation
- [X] Task 3.1: Run `go build ./...` to verify compilation
- [X] Task 3.2: Run `go test ./internal/tui/...` to check for regressions
- [X] Task 3.3: Run `go test -race ./internal/tui/...` for concurrency safety
- [X] Task 3.4: Run `golangci-lint run ./internal/tui/...` for static analysis

## Phase 4: Polish
- [X] Task 4.1: Verify all `Foreground(lipgloss.Color("6"))` references in list files are replaced (non-selection uses like detail title styles should remain)
- [X] Task 4.2: Verify no selection-indicator characters (`›`, `▸`) remain in list rendering code (tree `▶`/`▼` for collapse are fine)
