# feat(cli): interactive TUI for `wave run` pipeline selection

**Issue**: [#79](https://github.com/re-cinq/wave/issues/79)
**Labels**: `ux`, `ready-for-impl`
**Author**: nextlevelshit

## Summary

When `wave run` is invoked without arguments (or with partial input), launch an interactive terminal UI that guides the user through pipeline selection, input, and flag configuration.

## Motivation

Currently `wave run <pipeline>` requires knowing the exact pipeline name. With 23+ built-in pipelines, discoverability is poor. An interactive selector would make the CLI more approachable and faster to use.

## Proposed Behavior

### 1. Pipeline Selection
- `wave run` (no args) opens an interactive fuzzy-filterable list of available pipelines
- Arrow keys to navigate, type to filter/autocomplete
- Each entry shows pipeline name + description (from metadata)
- Enter to select

### 2. Input Prompt
- After selecting a pipeline, prompt for optional input text
- Show the pipeline's `input.example` as placeholder/hint
- Enter to confirm (empty = no input)

### 3. Flag Selection
- Present common flags as toggleable checkboxes:
  - `--output text|json` (output format)
  - `--verbose` (real-time tool activity)
  - `--dry-run` (show what would execute)
  - `--mock` (use mock adapter)
  - `--debug` (debug logging)
- Arrow keys + space to toggle, Enter to confirm and run

### 4. Confirmation
- Show the composed command before execution: `wave run <pipeline> "<input>" --output json --verbose`
- Enter to execute, Esc to go back

## Technical Considerations

- Use [charmbracelet/huh](https://github.com/charmbracelet/huh) (Go TUI forms library, built on bubbletea)
- Fall back to non-interactive mode when stdin is not a TTY (piped input, CI)
- `wave run <partial>` could pre-filter the list to matching pipelines
- Pipeline list sourced from embedded defaults + `.wave/pipelines/` directory
- Keep the TUI in a new `internal/tui/` package to isolate the dependency

## Edge Cases

- **`wave run feat`** (partial name) — pre-filter list to matching pipelines, skip straight to selector
- **`wave run feature "input"`** (full args) — skip TUI entirely, run directly
- **Piped stdin / non-TTY** — skip TUI, require args as today
- **Single match on partial** — auto-select, skip to input prompt
- **Esc at any step** — go back to previous step (or exit if at first step)

## Library Decision

Use `charmbracelet/huh` (forms library on top of bubbletea). It provides:
- **Select** — filterable list with arrow keys (pipeline picker)
- **Input** — text input with placeholder (pipeline input)
- **Confirm** — yes/no (run confirmation)
- **MultiSelect** — toggleable checkboxes (flag selection)

Already Go, no CGO, fits single-binary constraint. bubbletea is already a dependency.

## Out of Scope

- Full dashboard or persistent TUI — this is a one-shot selector
- Editing pipeline YAML from the TUI

## Acceptance Criteria

1. `wave run` (no args, TTY) launches interactive pipeline selector
2. User can fuzzy-filter, select pipeline, enter input, toggle flags, and confirm
3. Non-TTY invocations fall back to existing error behavior (require pipeline arg)
4. `wave run <partial>` pre-filters the pipeline list
5. `wave run <pipeline> <input>` skips TUI entirely (existing behavior preserved)
6. Composed command is shown before execution for confirmation
7. Esc navigates back through steps or exits
8. Unit tests cover TUI data collection and pipeline listing logic
9. Integration with existing `collectPipelines()` and `loadPipeline()` functions
