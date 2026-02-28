# fix: incorrect token count in pipeline summary and missing token display in TUI output

**Feature Branch**: `098-token-display-fix`
**Issue**: [#98](https://github.com/re-cinq/wave/issues/98)
**Labels**: bug, ux
**Status**: Draft
**Complexity**: Medium

## Problem

The token count displayed in the pipeline completion summary appears incorrect. The reported token values do not match expected usage for the operations performed.

Additionally, token usage information (both per-step and total) is not visible in the TUI (interactive terminal) output — it only appears in the verbose/text log output.

## Expected Behavior

1. **Accurate token counts**: The token count reported per step and in the pipeline summary should accurately reflect actual LLM API token usage (input + output tokens)
2. **Token visibility in TUI**: The interactive TUI display should show:
   - Per-step token usage (next to elapsed time per step)
   - Total pipeline token usage in the summary header or completion banner
3. **Token visibility in text/JSON output**: The `--verbose` log output should include per-step and total token counts (this partially works already but values may be incorrect)

## Actual Behavior

The verbose output shows token counts per step (e.g., `149.1k tokens`), but:
- The reported values may not be correct — needs verification against actual API usage
- The TUI progress display only shows elapsed time per step, not token usage
- The pipeline summary header only shows `Elapsed` time, not total tokens

## Steps to Reproduce

1. Run any multi-step pipeline: `wave run gh-issue-rewrite <issue-url>`
2. Observe the TUI display — no token counts are shown
3. Compare the token counts in `--verbose` output against expected values
4. Check the pipeline completion summary — no total token count is displayed

## Acceptance Criteria

- [ ] Token counts per step are verified against actual adapter/API usage and are accurate
- [ ] TUI display shows per-step token usage alongside elapsed time
- [ ] Pipeline summary (both TUI and text output) shows total token usage across all steps
- [ ] Unit tests cover token counting logic
- [ ] Existing token display in verbose output remains functional

## Components Affected

- `internal/display/` — TUI rendering and progress display
- `internal/adapter/` — Token counting from subprocess output
- `internal/event/` — Progress events carrying token data
- `internal/pipeline/` — Pipeline summary generation

## Codebase Analysis

### Current Token Flow

1. **Adapter layer** (`internal/adapter/claude.go`):
   - `parseOutput()` extracts tokens from NDJSON `result` events using `InputTokens + OutputTokens + CacheCreationInputTokens` (excluding `CacheReadInputTokens`)
   - `parseStreamLine()` extracts streaming token data into `StreamEvent.TokensIn` / `StreamEvent.TokensOut` — but includes `CacheReadInputTokens` in TokensIn, creating an inconsistency with `parseOutput()`
   - Fallback to `assistantTokens` (from last assistant event) if result tokens are 0
   - Final fallback: byte-length estimate `len(data) / 4`

2. **Event layer** (`internal/event/emitter.go`):
   - `Event.TokensUsed` carries per-step token count
   - Emitted on step completion by executor

3. **Pipeline executor** (`internal/pipeline/executor.go`):
   - Sets `Event.TokensUsed = result.TokensUsed` on step completion
   - `GetTotalTokens()` sums `tokens_used` across all step results
   - `BuildOutcome()` receives total tokens for the outcome summary

4. **Display layer**:
   - **BasicProgressDisplay** (`progress.go`): Shows tokens on completed steps in verbose text output: `"✓ step completed (Xs, Yk tokens)"`
   - **StepStatus.Render()** (`progress.go`): Shows tokens for completed/failed steps: `"• Xk tokens"`
   - **BubbleTeaProgressDisplay** (`bubbletea_progress.go`): Does NOT capture `TokensUsed` from events into internal state; `toPipelineContext()` does NOT propagate token data to `PipelineContext`
   - **ProgressModel.View()** (`bubbletea_model.go`): Completed step lines show duration but NOT token counts
   - **Dashboard header** (`bubbletea_model.go:renderHeader()`): Shows `Elapsed` time but NOT total tokens
   - **PipelineContext** (`types.go`): Has no field for per-step token counts or total tokens

### Identified Issues

1. **Token counting inconsistency**: `parseStreamLine()` includes `CacheReadInputTokens` in streaming events, but `parseOutput()` excludes it from the final total. The streaming tokens will appear inflated compared to the final count.

2. **TUI missing token display**: `BubbleTeaProgressDisplay.updateFromEvent()` does not capture `evt.TokensUsed` when a step completes. The `PipelineContext` struct has no per-step token map or total token field, so the bubbletea model has no data to render.

3. **Dashboard header lacks total tokens**: `renderHeader()` only shows elapsed time — no total token aggregation.

4. **Completed step lines lack tokens**: `renderCurrentStep()` in `bubbletea_model.go` renders `"✓ stepID (persona) (duration)"` but never includes token counts.
