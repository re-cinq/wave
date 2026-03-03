# feat(display): redesign pipeline TUI — deduplicate logo, clear screen on start, model/adapter visibility, enterprise polish

**Issue**: [#144](https://github.com/re-cinq/wave/issues/144)
**Author**: nextlevelshit
**Labels**: enhancement
**State**: OPEN

## Summary

The pipeline TUI needs a visual overhaul to improve clarity, reduce clutter, and surface useful runtime metrics. Several items from the original scope have been addressed (token display), but key UX issues remain: duplicate logo rendering, missing screen clear on start, no model/adapter/temperature visibility per step, and the redundant `Config: wave.yaml` line.

## Problems

### 1. Duplicate logo display

When launching a pipeline via the interactive menu (`wave run`), the Wave ASCII logo renders twice — once for the menu and again for the pipeline view. Only one logo should be visible at a time.

The `ProgressModel.Init()` in `internal/display/bubbletea_model.go:36` currently just starts the tick loop without clearing the screen or suppressing the menu logo. A `tea.ClearScreen` command or equivalent is needed.

### 2. No terminal clearing on pipeline start

When a pipeline starts in normal (non-verbose) mode, the terminal should be cleared so the logo and progress output start at the top-left corner for better visibility.

The `ClearScreen()` function exists in `internal/display/formatter.go:42` but is not called during pipeline startup. The bubbletea program initialization in `internal/display/bubbletea_progress.go:102-106` does not use alternate screen mode or clear the screen.

### 3. ~~Missing token usage display~~ DONE

Token display was implemented in commits `b37707a` and `7baa18d`.

**Remaining gap**: Input vs output token breakdown is not shown — only total tokens per step. Estimated cost based on model pricing is also not implemented.

### 4. General TUI improvements — partially done

**Still missing**:
- Compact status bar showing model, context usage percentage, and file changes
- Collapsible tool call sections
- Model/adapter name and temperature displayed per step
- Removing the redundant `Config: wave.yaml` line from the header

### 5. `Config: wave.yaml` still displayed

The header still shows `Config: wave.yaml`, which should be removed as obvious information.

### 6. No model/adapter/temperature visibility per step

Requires extending PipelineContext or StepProgress to carry model/adapter metadata, passing model info from the adapter layer through the event system, and displaying it alongside the persona name.

## Acceptance Criteria

- [ ] Logo appears exactly once when launching via the interactive menu
- [ ] Terminal is cleared when a pipeline starts in normal mode
- [ ] Token usage split into input/output counts (not just total)
- [ ] Remove `Config: wave.yaml` (obvious information)
- [ ] Add adapter name to step display
- [ ] Make visible which model is used for which step and temperature
- [ ] Compact status bar showing model and context usage
- [ ] Collapsible tool call sections in verbose mode
- [ ] (Optional) Estimated cost based on model pricing

## Research Notes

A comprehensive research report was posted by the issue author confirming that all 10 research topics have proven solutions within the Bubble Tea ecosystem. Key findings:
- `tea.ClearScreen` is the idiomatic way to clear the terminal in bubbletea
- Token input/output breakdown is available from Claude Code's NDJSON `result` events (`input_tokens`, `output_tokens`, `cache_creation_input_tokens`)
- Model/adapter info is already emitted in step-start events via `event.Event.Model` and `event.Event.Adapter` fields
- Collapsible sections can be implemented with a toggle key and conditional rendering
