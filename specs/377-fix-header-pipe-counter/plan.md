# Implementation Plan: Fix Header Pipe Counter (#377)

## Objective

Replace the misleading "Pipes: X/Y" counter in the TUI header bar with a clear "N running" display that accurately represents the number of active pipelines without implying a capacity limit.

## Approach

The fix is minimal and surgical: change the `renderPipesValue()` function to display only the running count (e.g., "2 running") instead of the confusing `running/total` ratio format. Remove the `TotalPipes` field from `HeaderMetadata` and `RunningCountMsg` since it served no purpose other than feeding the misleading denominator. Update `pipeline_list.go` to stop computing and emitting the total. Update all tests to reflect the new format.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/header.go` | modify | Rewrite `renderPipesValue()` to show "N running" instead of "X/Y". Remove `TotalPipes` usage from `Update()` handler for `RunningCountMsg`. Rename label from "Pipes" to "Running" for clarity. |
| `internal/tui/header_metadata.go` | modify | Remove `TotalPipes` field from `HeaderMetadata` struct |
| `internal/tui/header_messages.go` | modify | Remove `TotalPipes` field from `RunningCountMsg` struct |
| `internal/tui/pipeline_list.go` | modify | Remove `totalPipes` computation in `handleDataMsg()` and `PipelineLaunchedMsg` handler; emit `RunningCountMsg{Count: len(m.running)}` only |
| `internal/tui/header_test.go` | modify | Update tests for new render format ("N running" instead of "X/Y"), remove references to `TotalPipes` |

## Architecture Decisions

1. **"N running" format over other alternatives**: The issue suggests "2 running" as the clearest option. This accurately describes the state without implying capacity. When 0 pipelines are running, show "—" (em dash) as a placeholder, consistent with other header elements.

2. **Remove TotalPipes entirely**: Rather than keeping `TotalPipes` for potential future use, remove it. YAGNI — if a pipeline concurrency limit is ever added, the counter can be reintroduced with proper semantics. This keeps the codebase clean.

3. **Rename label to "Running"**: The label "Pipes:" is ambiguous. "Running:" directly describes what the value represents.

## Risks

| Risk | Mitigation |
|------|------------|
| Breaking tests that assert on "Pipes:" label or "X/Y" format | Update all affected test assertions in the same PR |
| Other components depending on `TotalPipes` | Grep confirmed only `header.go`, `header_metadata.go`, `header_messages.go`, and `pipeline_list.go` reference it |
| Semantic regression if future pipeline limit feature is added | Documented in issue that this counter should return only if a real concurrency limit is implemented |

## Testing Strategy

- Update existing `renderPipesValue()` tests to assert new format:
  - `RunningCount == 0 && no pipelines` => "—"
  - `RunningCount == 1` => "1 running" (bold yellow)
  - `RunningCount == 5` => "5 running" (bold yellow)
  - `RunningCount == 0 && pipelines exist` => "0 running" (dim)
- Update `RunningCountMsg` tests to remove `TotalPipes` assertions
- Verify full header render at different widths still passes
- Run `go test ./internal/tui/... -race` to ensure no regressions
