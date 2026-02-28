# Tasks

## Phase 1: Fix Token Counting Accuracy

- [X] Task 1.1: Fix `parseStreamLine()` result event token calculation in `internal/adapter/claude.go` — exclude `CacheReadInputTokens` from `TokensIn` to match `parseOutput()` logic
- [X] Task 1.2: Add unit tests in `internal/adapter/claude_test.go` verifying token extraction from sample NDJSON payloads with various cache token fields
- [X] Task 1.3: Add unit test for the fallback chain (result → assistant → byte estimate)

## Phase 2: Thread Token Data Through Display Context

- [X] Task 2.1: Add `StepTokens map[string]int` and `TotalTokens int` fields to `PipelineContext` in `internal/display/types.go` [P]
- [X] Task 2.2: Capture `evt.TokensUsed` on step completion in `BubbleTeaProgressDisplay.updateFromEvent()` in `internal/display/bubbletea_progress.go` (store in a new `stepTokens map[string]int` field)
- [X] Task 2.3: Propagate `StepTokens` and `TotalTokens` in `BubbleTeaProgressDisplay.toPipelineContext()` — compute total by summing per-step values
- [X] Task 2.4: Propagate `StepTokens` and `TotalTokens` in `ProgressDisplay.toPipelineContext()` in `internal/display/progress.go` for non-bubbletea display path

## Phase 3: Render Tokens in TUI

- [X] Task 3.1: Update `renderCurrentStep()` in `internal/display/bubbletea_model.go` — for completed steps, append formatted token count after duration: `"✓ stepID (persona) (Xs, Yk tokens)"` [P]
- [X] Task 3.2: Update `renderHeader()` in `internal/display/bubbletea_model.go` — add total tokens to the Elapsed line: `"Elapsed: Xm Xs • Yk tokens"` [P]
- [X] Task 3.3: Update `renderStepStatusPanel()` in `internal/display/dashboard.go` — add per-step tokens for completed/failed steps alongside duration [P]
- [X] Task 3.4: Update `renderHeader()` in `internal/display/dashboard.go` — add total tokens to project info line [P]

## Phase 4: Testing

- [X] Task 4.1: Add unit tests in `internal/display/bubbletea_model_test.go` — verify completed step lines include token count when `StepTokens` is populated
- [X] Task 4.2: Add unit tests for zero-token graceful degradation (no token display when tokens are 0)
- [X] Task 4.3: Add unit tests in `internal/display/dashboard_test.go` for token rendering in dashboard panels

## Phase 5: Validation

- [X] Task 5.1: Run `go build ./...` to verify compilation
- [X] Task 5.2: Run `go test ./internal/adapter/... ./internal/display/...` to verify all tests pass
- [ ] Task 5.3: Manual verification — run a pipeline and confirm TUI shows per-step and total tokens
