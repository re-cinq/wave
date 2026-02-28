# Implementation Plan: Token Display Fix (#98)

## Objective

Fix two related problems: (1) token counts may be inaccurate due to inconsistent handling of `cache_read_input_tokens` between streaming and final output parsing, and (2) the bubbletea TUI does not display per-step or total token usage despite the data being available in the pipeline executor.

## Approach

The fix touches four layers in a bottom-up order:

1. **Verify and align token counting** in the adapter layer
2. **Thread token data** through the display PipelineContext
3. **Render per-step tokens** in the bubbletea TUI completed step lines
4. **Render total tokens** in the TUI header and pipeline summary banner

## File Mapping

### Modified Files

| File | Change |
|------|--------|
| `internal/adapter/claude.go` | Align `parseStreamLine()` result event token calculation to match `parseOutput()` — exclude `CacheReadInputTokens` from the result event's `TokensIn` |
| `internal/display/types.go` | Add `StepTokens map[string]int` and `TotalTokens int` fields to `PipelineContext` |
| `internal/display/bubbletea_progress.go` | Capture `evt.TokensUsed` on step completion in `updateFromEvent()`; propagate `StepTokens` and `TotalTokens` in `toPipelineContext()` |
| `internal/display/bubbletea_model.go` | Render per-step tokens in `renderCurrentStep()` for completed steps; render total tokens in `renderHeader()` alongside elapsed time |
| `internal/display/dashboard.go` | Add total tokens to `renderHeader()` project info; add per-step tokens to `renderStepStatusPanel()` for completed steps |
| `internal/display/progress.go` | Propagate `StepTokens` and `TotalTokens` in `ProgressDisplay.toPipelineContext()` |
| `internal/adapter/claude_test.go` | Add/update tests for token counting accuracy |
| `internal/display/bubbletea_model_test.go` | Add tests for token display rendering |
| `internal/display/dashboard_test.go` | Add tests for token display in dashboard |

### No New Files Required

The infrastructure for token counting already exists. This is a wiring and rendering fix.

## Architecture Decisions

1. **Exclude `CacheReadInputTokens` consistently**: The `parseOutput()` method already documents why cached context re-reads should be excluded from cumulative totals. The streaming `parseStreamLine()` function for `result` events should follow the same logic.

2. **Per-step tokens via `PipelineContext.StepTokens` map**: Rather than adding tokens to `StepDurations` or creating a parallel tracking struct, add a simple `map[string]int` alongside the existing `StepDurations map[string]int64`. This keeps the pattern consistent.

3. **Total tokens via `PipelineContext.TotalTokens`**: A single `int` field on `PipelineContext` provides the aggregate. Calculated by summing `StepTokens` values during context construction.

4. **Display format**: Use the existing `FormatTokenCount()` helper (already in `formatter.go`) which formats as `"X"` for < 1000 and `"X.Xk"` for >= 1000.

5. **TUI layout for completed steps**: Append tokens after duration: `"✓ stepID (persona) (Xs, Yk tokens)"` — matching the format already used by `BasicProgressDisplay`.

6. **TUI header layout**: Add total tokens to the third line of project info: `"Elapsed: Xm Xs • Yk tokens"`.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Token count still inaccurate after fix | Medium | Add unit tests with known NDJSON payloads to verify exact token extraction |
| Streaming tokens diverge from final count | Low | Streaming events are informational; only the final `result` event count is authoritative and used for display |
| PipelineContext struct changes break tests | Low | The new fields are additive (optional maps); zero-value means "no data" |
| TUI layout too wide with tokens added | Low | `FormatTokenCount()` produces compact output (e.g., "149.1k"); test at 80-col width |

## Testing Strategy

1. **Unit tests for adapter token parsing** (`claude_test.go`):
   - Test `parseOutput()` with sample NDJSON containing result events with various cache token fields
   - Test `parseStreamLine()` for result events to verify `CacheReadInputTokens` exclusion
   - Test fallback chain: result tokens → assistant tokens → byte estimate

2. **Unit tests for display rendering** (`bubbletea_model_test.go`, `dashboard_test.go`):
   - Test that completed step line includes formatted token count when `StepTokens` is populated
   - Test that header includes total tokens when `TotalTokens > 0`
   - Test that zero tokens produces no token display (graceful degradation)

3. **Integration validation**:
   - Run a multi-step pipeline and verify TUI shows per-step and total tokens
   - Compare verbose text output tokens against TUI tokens for consistency
