# Implementation Plan: Display ETA in UI

## Objective

Wire the already-captured `EstimatedTimeMs` from `PipelineContext` into the BubbleTea model's header rendering, using the existing `FormatDuration` package-level function to format the remaining time.

## Approach

The fix is surgical: modify `renderHeader()` in `bubbletea_model.go` to append ETA information to the `projectLines` array when `EstimatedTimeMs > 0`. The `FormatETA` method exists on `Formatter` (a struct with ANSI capabilities), but the bubbletea model uses lipgloss for styling. We'll use the package-level `FormatDuration()` function directly to format the milliseconds, keeping the approach consistent with how elapsed time is already rendered.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/display/bubbletea_model.go` | modify | Add ETA line to `renderHeader()` projectLines |
| `internal/display/bubbletea_model_test.go` | modify | Add test for ETA rendering in header |

## Architecture Decisions

1. **Use package-level `FormatDuration()` instead of `Formatter.FormatETA()`**: The bubbletea model uses lipgloss for styling, not the `Formatter` struct. Using `FormatDuration()` directly avoids instantiating an unnecessary `Formatter`. The format will be `ETA: <duration>` to match the `FormatETA` convention.

2. **Show ETA on a separate line in the header**: The header already has 3 lines (Pipeline, Elapsed, Progress). Adding ETA as a 4th line keeps each piece of information scannable. When ETA is 0 (no estimate yet), we omit the line entirely to avoid clutter — this is better than showing "calculating..." which would change the header height dynamically.

3. **Only modify BubbleTea model**: The issue specifically calls out `bubbletea_model.go`. The `ProgressDisplay` (non-TUI) and `Dashboard` are secondary displays. The BubbleTea TUI is the primary interactive display and the one referenced in the issue.

## Risks

- **Header height change**: Adding a 4th line to the header increases its height by 1 row. Since the logo is 3 lines tall and the project info column is rendered beside it (not below), the 4th line will just extend the project column. This is safe.
- **ETA flicker**: ETA updates come via events, so the value may jump. This is inherent to the ETA calculation, not a display concern.

## Testing Strategy

- Add a unit test in `bubbletea_model_test.go` that creates a `ProgressModel` with a `PipelineContext` containing a non-zero `EstimatedTimeMs`, renders the view, and asserts the output contains the formatted ETA string.
- Verify existing tests still pass with `go test ./internal/display/...`.
