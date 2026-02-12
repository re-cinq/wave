# Tasks

## Phase 1: Dependency Setup
- [X] Task 1.1: Add `charmbracelet/huh` dependency via `go get github.com/charmbracelet/huh@latest` and run `go mod tidy`

## Phase 2: Pipeline Discovery
- [X] Task 2.1: Create `internal/tui/pipelines.go` with `DiscoverPipelines()` function that scans `.wave/pipelines/*.yaml` and returns pipeline name, description, step count, and input example [P]
- [X] Task 2.2: Create `internal/tui/pipelines_test.go` with table-driven tests for pipeline discovery, including edge cases (empty dir, malformed YAML, no description) [P]

## Phase 3: Core TUI Implementation
- [X] Task 3.1: Create `internal/tui/run_selector.go` with `RunPipelineSelector(preFilter string) (*Selection, error)` implementing the 4-step form (select pipeline, input text, flag toggles, confirmation)
- [X] Task 3.2: Implement pipeline select step using `huh.NewSelect` with fuzzy filtering, pipeline name + description display
- [X] Task 3.3: Implement input prompt step using `huh.NewInput` with pipeline's `input.example` as placeholder
- [X] Task 3.4: Implement flag selection step using `huh.NewMultiSelect` for toggleable flags (verbose, output json, dry-run, mock, debug)
- [X] Task 3.5: Implement confirmation step using `huh.NewConfirm` showing the composed command string
- [X] Task 3.6: Create `internal/tui/run_selector_test.go` with unit tests for option building, command string composition, pre-filter logic, and Selection struct population

## Phase 4: CLI Integration
- [X] Task 4.1: Modify `cmd/wave/commands/run.go` — replace the hard error on missing pipeline with TTY check + TUI launch, mapping `Selection` fields to `RunOptions`
- [X] Task 4.2: Handle partial name matching: when 1 arg is given but `loadPipeline()` fails, attempt TUI with pre-filter before returning error
- [X] Task 4.3: Handle `huh.ErrUserAborted` (Esc/Ctrl+C) with clean exit (return nil, no error)

## Phase 5: Testing
- [X] Task 5.1: Add/update tests in `cmd/wave/commands/run_tui_test.go` verifying non-TTY behavior still returns error when no pipeline provided
- [X] Task 5.2: Add tests verifying full args (`wave run <pipeline> <input>`) skip TUI entirely
- [X] Task 5.3: Run full test suite (`go test ./...`) and fix any regressions

## Phase 6: Polish
- [X] Task 6.1: Verify `go vet ./...` and `gofmt` pass cleanly
- [X] Task 6.2: Run `go test -race ./...` to verify no race conditions
