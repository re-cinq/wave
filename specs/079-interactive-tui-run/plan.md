# Implementation Plan: Interactive TUI for `wave run`

## Objective

Add an interactive terminal UI to `wave run` that guides users through pipeline selection, input entry, flag configuration, and confirmation when no pipeline argument is provided and stdin is a TTY.

## Approach

Create a new `internal/tui/` package using `charmbracelet/huh` (forms library built on the already-present bubbletea dependency) to implement a 4-step interactive flow. Integrate it into `cmd/wave/commands/run.go` at the entry point of `NewRunCmd()` before the existing pipeline validation check.

### Key Design Decision: `huh` over raw bubbletea

The `huh` library provides high-level form components (Select, Input, MultiSelect, Confirm) that map 1:1 to the four interaction steps. This avoids ~500+ lines of boilerplate Elm-architecture code that raw bubbletea would require, while still allowing drop-down to bubbletea if custom rendering is needed later.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/tui/run_selector.go` | **create** | Core TUI form: pipeline select, input prompt, flag toggles, confirmation |
| `internal/tui/run_selector_test.go` | **create** | Unit tests for pipeline listing, filtering, option building |
| `internal/tui/pipelines.go` | **create** | Pipeline discovery: list available pipelines with metadata |
| `internal/tui/pipelines_test.go` | **create** | Tests for pipeline discovery and sorting |
| `cmd/wave/commands/run.go` | **modify** | Add TTY check + TUI launch at command entry; remove hard error when no pipeline |
| `cmd/wave/commands/run_test.go` | **modify** | Add tests for new interactive path and fallback behavior |
| `go.mod` | **modify** | Add `github.com/charmbracelet/huh` dependency |
| `go.sum` | **modify** | Updated automatically by `go mod tidy` |

## Architecture Decisions

### 1. Package isolation (`internal/tui/`)
The TUI code lives in its own package to:
- Keep `cmd/wave/commands/` focused on cobra command wiring
- Allow the TUI to be tested independently without cobra
- Isolate the `huh` dependency to a single package

### 2. Integration point in `run.go`
The TUI check happens inside the cobra `RunE` function, after positional arg parsing but before the existing pipeline validation:

```go
// After positional arg parsing, before validation:
if opts.Pipeline == "" {
    termInfo := display.NewTerminalInfo()
    if termInfo.IsTTY() && termInfo.SupportsANSI() {
        selected, err := tui.RunPipelineSelector(filter)
        if err != nil {
            // User cancelled (Esc/Ctrl+C) → clean exit
            if errors.Is(err, huh.ErrUserAborted) {
                return nil
            }
            return err
        }
        // Apply selections to opts
        opts.Pipeline = selected.Pipeline
        opts.Input = selected.Input
        // ... flags
    }
}
```

This preserves the existing non-interactive path completely. CI/scripts that provide args work exactly as before.

### 3. Reuse existing `collectPipelines()`
The `list.go` file already has `collectPipelines()` which reads `.wave/pipelines/*.yaml` and extracts name, description, and steps. The TUI pipeline discovery will use the same scanning pattern, extracting it into a shared utility or calling it directly via a thin wrapper in `internal/tui/pipelines.go`.

### 4. TTY detection
Reuse the existing `display.TerminalInfo` for TTY detection, which already handles `WAVE_FORCE_TTY` env var override and `golang.org/x/term.IsTerminal()`. This provides consistent behavior with the existing auto-mode progress display.

### 5. Partial name pre-filtering
When `wave run feat` is provided (1 arg but no matching pipeline), the TUI opens with the filter pre-set to "feat". If exactly one pipeline matches, auto-select it and skip to the input prompt.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| `huh` API instability | Medium — could break on updates | Pin version in go.mod; huh v0.6+ is stable |
| TUI breaks non-TTY behavior | High — CI/pipeline usage fails | TTY check gates the entire TUI path; existing error path preserved |
| Binary size increase | Low — huh is small Go code | Already have bubbletea; huh adds minimal overhead |
| `huh` forms don't support back-navigation | Medium — UX regression | huh groups support Esc-to-go-back natively via form groups |
| Race with SIGWINCH handler | Low — terminal resize during form | huh handles resize internally via bubbletea |

## Testing Strategy

### Unit Tests (`internal/tui/`)
- Pipeline discovery: scanning directory, filtering, sorting
- Option building: converting PipelineInfo to huh select options
- Partial name matching and pre-filtering logic
- Flag selection value extraction
- Command string composition for confirmation display

### Integration Tests (`cmd/wave/commands/`)
- `wave run` without args in non-TTY returns error (existing behavior preserved)
- `wave run <pipeline> <input>` skips TUI entirely
- Flag parsing from TUI selections correctly populates RunOptions

### Manual Testing
- TTY interaction flow through all 4 steps
- Esc back-navigation at each step
- Ctrl+C clean exit
- Partial name filtering (e.g., `wave run feat`)
- Terminal resize during form display
