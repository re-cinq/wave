# Implementation Plan: TUI Redesign (#144)

## Objective

Redesign the Wave pipeline TUI to eliminate visual clutter (duplicate logo, redundant config line), surface per-step model/adapter/temperature metadata, split token counts into input/output, and add collapsible tool call sections and a compact status bar.

## Approach

The implementation follows 5 workstreams that map to the issue's problem sections. Each workstream touches a well-defined set of files and can largely proceed independently, with the data model extension (workstream 2) being a prerequisite for display changes (workstream 4).

### Workstream 1: Logo Deduplication + Screen Clear

**Strategy**: Use `tea.ClearScreen` in `ProgressModel.Init()` to clear the terminal when the bubbletea TUI starts, eliminating the duplicate logo from the interactive menu. This is a 2-line change.

**Files**:
- `internal/display/bubbletea_model.go` — modify `Init()` to return `tea.Batch(tea.ClearScreen, tickCmd())`

### Workstream 2: Model/Adapter/Temperature Data Pipeline

**Strategy**: The event system already carries `Model` and `Adapter` fields on step-start events (`executor.go:585-586`). We need to:
1. Add `Temperature` to `event.Event`
2. Extend `PipelineContext` and `BubbleTeaProgressDisplay` to store per-step model/adapter/temperature
3. Capture these fields from events in `updateFromEvent()`
4. Pass them through to the view layer

**Files**:
- `internal/event/emitter.go` — add `Temperature float64` field to `Event`
- `internal/display/types.go` — add `StepModels`, `StepAdapters`, `StepTemperatures` maps to `PipelineContext`
- `internal/display/bubbletea_progress.go` — add storage maps, capture model/adapter/temperature in `updateFromEvent()`, include in `toPipelineContext()`
- `internal/pipeline/executor.go` — add `Temperature` to the step-start event emission at line ~585

### Workstream 3: Remove Config Line + Header Cleanup

**Strategy**: Remove the `Config: wave.yaml` line from `renderHeader()` in the bubbletea model. Replace it with the model name of the currently running step (if available) for a more useful header.

**Files**:
- `internal/display/bubbletea_model.go` — modify `renderHeader()` to remove `Config:` line
- `internal/display/dashboard.go` — modify `renderHeader()` and `formatElapsedInfo()` to remove manifest path display

### Workstream 4: Model/Adapter Display per Step

**Strategy**: Show model and adapter info alongside the persona name in step lines. Format: `⠋ step-name (persona) [model via adapter @ temp]`. Only show when data is available.

**Files**:
- `internal/display/bubbletea_model.go` — modify `renderCurrentStep()` to include model/adapter/temperature from `PipelineContext`

### Workstream 5: Input/Output Token Split

**Strategy**: The NDJSON `result` event already contains `input_tokens` and `output_tokens`. Extend the token tracking to store both values separately and display them as "Xk in / Yk out" instead of a single total.

**Files**:
- `internal/event/emitter.go` — add `TokensIn int` and `TokensOut int` fields to `Event`
- `internal/display/types.go` — add `StepTokensIn`, `StepTokensOut` maps to `PipelineContext`; add `TotalTokensIn`, `TotalTokensOut` fields
- `internal/display/bubbletea_progress.go` — track input/output tokens separately in `updateFromEvent()` and `toPipelineContext()`
- `internal/display/bubbletea_model.go` — modify `renderCurrentStep()` completed step line and `formatElapsedWithTokens()` to show in/out breakdown
- `internal/pipeline/executor.go` — emit `TokensIn`/`TokensOut` in the step-completed event (parse from adapter result)
- `internal/adapter/adapter.go` — add `TokensIn`, `TokensOut` fields to `AdapterResult`
- `internal/adapter/claude.go` — populate `TokensIn`/`TokensOut` from parsed output

### Workstream 6: Compact Status Bar

**Strategy**: Add a status bar below the progress bar showing the model of the currently running step, token burn rate, and context usage. This replaces the removed `Config:` line with more useful real-time information.

**Files**:
- `internal/display/bubbletea_model.go` — add `renderStatusBar()` method, call from `View()`

### Workstream 7: Collapsible Tool Call Sections

**Strategy**: In verbose mode, add a toggle key (`t`) that collapses/expands tool call details under running steps. Default to expanded. Store toggle state in `ProgressModel`.

**Files**:
- `internal/display/bubbletea_model.go` — add `showToolCalls bool` field, handle `t` keypress in `Update()`, conditionally render tool lines in `renderCurrentStep()`

### Workstream 8 (Optional): Cost Estimation

**Strategy**: Define a simple pricing table for known models (opus, sonnet, haiku) and compute estimated cost from input/output token counts. Display in the header or status bar.

**Files**:
- `internal/display/bubbletea_model.go` — add cost calculation and display (deferred to separate PR if scope grows)

## File Mapping

| File | Action | Workstream |
|------|--------|------------|
| `internal/event/emitter.go` | modify | 2, 5 |
| `internal/display/types.go` | modify | 2, 5 |
| `internal/display/bubbletea_model.go` | modify | 1, 3, 4, 5, 6, 7 |
| `internal/display/bubbletea_progress.go` | modify | 2, 5 |
| `internal/display/dashboard.go` | modify | 3 |
| `internal/pipeline/executor.go` | modify | 2, 5 |
| `internal/adapter/adapter.go` | modify | 5 |
| `internal/adapter/claude.go` | modify | 5 |
| `internal/display/progress.go` | modify | 2, 5 (BasicProgressDisplay) |
| `internal/display/formatter.go` | no change | — |

## Architecture Decisions

1. **No alternate screen mode**: The bubbletea program deliberately avoids `tea.WithAltScreen()` (comment at `bubbletea_progress.go:102`) to prevent terminal corruption. We use `tea.ClearScreen` instead, which clears the scrollback without switching buffers.

2. **Event-driven metadata propagation**: Model/adapter/temperature flow through the existing event system rather than adding a separate channel. The executor already emits these fields; the display layer just needs to capture and render them.

3. **Backward-compatible event extension**: New fields (`Temperature`, `TokensIn`, `TokensOut`) are added as optional fields with `omitempty` JSON tags, so existing consumers are unaffected.

4. **Per-step maps over struct changes**: We add `map[string]string` and `map[string]float64` to `PipelineContext` rather than creating a new struct, maintaining consistency with existing `StepTokens`, `StepPersonas`, etc.

5. **Collapsible sections via model state**: The toggle is stored in the bubbletea `ProgressModel`, not in `PipelineContext`, since it's a view-only concern.

## Risks

| Risk | Mitigation |
|------|------------|
| `tea.ClearScreen` may not work on all terminals | Bubbletea handles terminal detection; non-TTY paths already bypass the TUI entirely |
| Token in/out split unavailable for non-Claude adapters | Display falls back to total-only when `TokensIn`/`TokensOut` are zero |
| Model info missing for steps that haven't started | Only display model info for running/completed steps where data exists |
| Dashboard `renderHeader()` also shows Config | Fix both bubbletea model AND dashboard header for consistency |
| Existing tests may break with new PipelineContext fields | New fields are additive and zero-valued by default; existing tests pass without modification |

## Testing Strategy

1. **Unit tests for new PipelineContext fields**: Verify that `StepModels`, `StepAdapters`, `StepTemperatures`, `StepTokensIn`, `StepTokensOut` are correctly populated from events
2. **Unit tests for token formatting**: Test `FormatTokenCount` with in/out breakdown strings
3. **Unit tests for `Init()` return value**: Verify `tea.ClearScreen` is in the batch command
4. **Integration test for event propagation**: Verify model/adapter fields flow from executor through event emitter to progress display
5. **Visual regression**: Manual verification of TUI layout (no automated screenshot testing infrastructure exists)
6. **Existing test suite**: `go test ./...` must pass — all changes are additive
