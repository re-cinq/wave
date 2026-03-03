# Tasks

## Phase 1: Data Model Extension
- [x] Task 1.1: Add `Temperature float64` field to `event.Event` in `internal/event/emitter.go`
- [x] Task 1.2: Add `TokensIn int` and `TokensOut int` fields to `event.Event` in `internal/event/emitter.go`
- [x] Task 1.3: Add `TokensIn int` and `TokensOut int` fields to `AdapterResult` in `internal/adapter/adapter.go`
- [x] Task 1.4: Populate `TokensIn`/`TokensOut` in `ClaudeAdapter.parseOutput()` in `internal/adapter/claude.go`
- [x] Task 1.5: Add per-step metadata maps to `PipelineContext` in `internal/display/types.go`: `StepModels map[string]string`, `StepAdapters map[string]string`, `StepTemperatures map[string]float64`, `StepTokensIn map[string]int`, `StepTokensOut map[string]int`, `TotalTokensIn int`, `TotalTokensOut int`

## Phase 2: Event Pipeline Wiring
- [x] Task 2.1: Emit `Temperature` in step-start event in `internal/pipeline/executor.go` (around line 585) [P]
- [x] Task 2.2: Emit `TokensIn`/`TokensOut` in step-completed event in `internal/pipeline/executor.go` [P]
- [x] Task 2.3: Add storage maps (`stepModels`, `stepAdapters`, `stepTemperatures`, `stepTokensIn`, `stepTokensOut`) to `BubbleTeaProgressDisplay` in `internal/display/bubbletea_progress.go`
- [x] Task 2.4: Capture model/adapter/temperature from "started"/"running" events in `updateFromEvent()` in `internal/display/bubbletea_progress.go`
- [x] Task 2.5: Capture `TokensIn`/`TokensOut` from "completed" events in `updateFromEvent()` in `internal/display/bubbletea_progress.go`
- [x] Task 2.6: Populate new `PipelineContext` fields in `toPipelineContext()` in `internal/display/bubbletea_progress.go`
- [x] Task 2.7: Update `BasicProgressDisplay` in `internal/display/progress.go` to track and display model/adapter per step [P]

## Phase 3: Core Display Changes
- [x] Task 3.1: Fix duplicate logo — modify `ProgressModel.Init()` to return `tea.Batch(tea.ClearScreen, tickCmd())` in `internal/display/bubbletea_model.go`
- [x] Task 3.2: Remove `Config: wave.yaml` line from `renderHeader()` in `internal/display/bubbletea_model.go`
- [x] Task 3.3: Remove manifest path from `formatElapsedInfo()` in `internal/display/dashboard.go`
- [x] Task 3.4: Show model/adapter/temperature per step in `renderCurrentStep()` — running steps show `[model via adapter]`, completed steps show `[model]` after duration in `internal/display/bubbletea_model.go` [P]
- [x] Task 3.5: Split token display into input/output in completed step lines: change from `Xk tokens` to `Xk in / Yk out` in `renderCurrentStep()` in `internal/display/bubbletea_model.go` [P]
- [x] Task 3.6: Update `formatElapsedWithTokens()` in header to show in/out split when available in `internal/display/bubbletea_model.go` [P]

## Phase 4: Status Bar + Collapsible Sections
- [x] Task 4.1: Add `renderStatusBar()` method to `ProgressModel` in `internal/display/bubbletea_model.go` — show model of running step, token burn rate
- [x] Task 4.2: Call `renderStatusBar()` from `View()` between progress and step list in `internal/display/bubbletea_model.go`
- [x] Task 4.3: Add `showToolCalls bool` field to `ProgressModel`, default `true` in `internal/display/bubbletea_model.go`
- [x] Task 4.4: Handle `t` keypress toggle in `Update()` to flip `showToolCalls` in `internal/display/bubbletea_model.go`
- [x] Task 4.5: Conditionally render tool activity lines in `renderCurrentStep()` based on `showToolCalls` in `internal/display/bubbletea_model.go`
- [x] Task 4.6: Update status line hint to include `t=tools` alongside `q=quit` in `View()` in `internal/display/bubbletea_model.go`

## Phase 5: Testing
- [x] Task 5.1: Add unit tests for new `PipelineContext` fields population in `internal/display/bubbletea_progress.go` [P]
- [x] Task 5.2: Add unit test verifying `Init()` returns `tea.ClearScreen` in batch in `internal/display/bubbletea_model_test.go` [P]
- [x] Task 5.3: Add unit tests for token in/out formatting in `internal/display/formatter_test.go` [P]
- [x] Task 5.4: Add unit tests for `parseOutput` returning `TokensIn`/`TokensOut` in `internal/adapter/claude_test.go` [P]
- [x] Task 5.5: Run full test suite `go test ./...` and fix any regressions

## Phase 6: Polish
- [x] Task 6.1: Verify dashboard `renderHeader()` and `renderStepStatusPanel()` consistency with bubbletea model
- [x] Task 6.2: Update `BasicProgressDisplay.EmitProgress()` completed step line to show in/out token split
- [ ] Task 6.3: (Optional) Add cost estimation helper and display if time permits
