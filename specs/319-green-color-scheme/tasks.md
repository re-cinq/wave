# Tasks

## Phase 1: Core Theme Update
- [X] Task 1.1: Update `internal/tui/theme.go` — rename `cyan` variable to `green`, change `Color("6")` to `Color("2")`, update all comments referencing cyan
- [X] Task 1.2: Update `internal/tui/header_logo.go` — change both `Color("6")` references to `Color("2")`

## Phase 2: TUI File Updates (all parallelizable)
- [X] Task 2.1: Update `internal/tui/header.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.2: Update `internal/tui/compose_list.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.3: Update `internal/tui/compose_detail.go` — change `Color("6")` to `Color("2")` (3 locations) [P]
- [X] Task 2.4: Update `internal/tui/persona_list.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.5: Update `internal/tui/persona_detail.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.6: Update `internal/tui/run_selector.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.7: Update `internal/tui/health_list.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.8: Update `internal/tui/health_detail.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.9: Update `internal/tui/contract_list.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.10: Update `internal/tui/contract_detail.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.11: Update `internal/tui/live_output.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.12: Update `internal/tui/issue_list.go` — change `Color("6")` to `Color("2")` (3 locations) [P]
- [X] Task 2.13: Update `internal/tui/issue_detail.go` — change `Color("6")` to `Color("2")` (2 locations) [P]
- [X] Task 2.14: Update `internal/tui/skill_list.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.15: Update `internal/tui/skill_detail.go` — change `Color("6")` to `Color("2")` [P]
- [X] Task 2.16: Update `internal/tui/pipeline_list.go` — change `Color("6")` to `Color("2")` (3 locations) [P]
- [X] Task 2.17: Update `internal/tui/pipeline_detail.go` — change `Color("6")` to `Color("2")` (3 locations) [P]

## Phase 3: Display Package Updates
- [X] Task 3.1: Update `internal/display/types.go` — change `Primary` from `\033[36m` to `\033[32m` in StandardColorScheme, HighContrastColorScheme, and BoldColorScheme [P]
- [X] Task 3.2: Update `internal/display/bubbletea_model.go` — change `Color("14")` to `Color("10")` for colorPrimary, colorShimmerMid, colorShimmerBase [P]
- [X] Task 3.3: Update `internal/display/dashboard.go` — change `Color("14")` to `Color("10")` and update "cyan" comment [P]

## Phase 4: Documentation CSS
- [X] Task 4.1: Update `docs/.vitepress/theme/styles/custom.css` — change `--wave-secondary` from `#06b6d4` to `#10b981` and dark mode variant from `#22d3ee` to `#34d399`

## Phase 5: Validation
- [X] Task 5.1: Run `go test ./...` and fix any test failures caused by color constant changes
- [X] Task 5.2: Grep for remaining `Color("6")`, `Color("14")`, `\033[36m` references to ensure complete replacement
- [X] Task 5.3: Run `go test -race ./...` for race condition check
