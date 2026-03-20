# Tasks

## Phase 1: Core Implementation
- [X] Task 1.1: Modify `renderHeader()` in `internal/display/bubbletea_model.go` to append an ETA line to `projectLines` when `m.ctx.EstimatedTimeMs > 0`, formatted as `fmt.Sprintf("ETA:      %s", FormatDuration(m.ctx.EstimatedTimeMs))`

## Phase 2: Testing
- [X] Task 2.1: Add test in `internal/display/bubbletea_model_test.go` that verifies ETA appears in rendered header when `EstimatedTimeMs > 0`
- [X] Task 2.2: Add test that verifies ETA line is absent when `EstimatedTimeMs == 0`
- [X] Task 2.3: Run `go test ./internal/display/...` to verify all existing tests pass

## Phase 3: Validation
- [X] Task 3.1: Run `go test ./...` for full suite validation
