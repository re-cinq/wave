# Tasks

## Phase 1: Remove TotalPipes from Data Model
- [X] Task 1.1: Remove `TotalPipes` field from `HeaderMetadata` struct in `internal/tui/header_metadata.go`
- [X] Task 1.2: Remove `TotalPipes` field from `RunningCountMsg` struct in `internal/tui/header_messages.go`

## Phase 2: Update Pipeline List Emission
- [X] Task 2.1: Update `handleDataMsg()` in `internal/tui/pipeline_list.go` to emit `RunningCountMsg{Count: len(m.running)}` without `TotalPipes` [P]
- [X] Task 2.2: Update `PipelineLaunchedMsg` handler in `internal/tui/pipeline_list.go` to emit `RunningCountMsg{Count: len(m.running)}` without `TotalPipes` [P]

## Phase 3: Update Header Rendering
- [X] Task 3.1: Remove `m.metadata.TotalPipes = msg.TotalPipes` from `RunningCountMsg` handler in `internal/tui/header.go`
- [X] Task 3.2: Rewrite `renderPipesValue()` in `internal/tui/header.go` to display "N running" format instead of "X/Y"
- [X] Task 3.3: Rename header label from "Pipes:" to "Running:" in `View()` method of `internal/tui/header.go`

## Phase 4: Update Tests
- [X] Task 4.1: Update `TestHeaderModel_Update_RunningCountMsg` test cases to remove `TotalPipes` references in `internal/tui/header_test.go`
- [X] Task 4.2: Add/update render tests for new "N running" format (0 running = "—", N > 0 = "N running") in `internal/tui/header_test.go`
- [X] Task 4.3: Verify header width tests still pass with new label and format in `internal/tui/header_test.go`

## Phase 5: Validation
- [X] Task 5.1: Run `go build ./...` to verify compilation
- [X] Task 5.2: Run `go test ./internal/tui/... -race` to verify all tests pass
- [X] Task 5.3: Run `go vet ./internal/tui/...` for static analysis
