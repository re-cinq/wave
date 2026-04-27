# Implementation Plan — #1164 Dedupe Event-Line Renderer

## Objective

Extract a single canonical event-line renderer in `internal/display` and replace the three current copies (`tui.formatEventLine`, `tui.formatStoredEvent`, `display.BasicProgressDisplay.EmitProgress`) with calls into it. Output must be byte-identical to current behavior.

## Approach

1. **Capture before refactor**: Add golden tests against the current three renderers to lock byte-exact output for every event state.
2. **Introduce canonical formatter**: New file `internal/display/eventline.go` exporting `EventLine` formatter with options for the three call-site framing variants (stepID-prefix vs timestamp-prefix; live vs CLI casing; truncation policy).
3. **Adapter for `state.LogRecord`**: Single function `EventFromLogRecord(state.LogRecord) event.Event` (or equivalent view) so the renderer has one input type.
4. **Migrate call sites**: Replace each of the three implementations with a thin wrapper that delegates to the canonical renderer with the appropriate options. Keep existing function signatures so callers do not change.
5. **Verify**: Re-run golden tests; output must remain byte-identical.

## File Mapping

### Created
- `internal/display/eventline.go` — canonical formatter with `EventLine(view, opts) string`, supporting options for prefix style, color, and stream-activity truncation.
- `internal/display/eventline_test.go` — table-driven tests covering every event state across both option profiles (live-tui, basic-cli).
- `internal/display/eventline_golden_test.go` *(optional)* — byte-exact regression vectors.
- `internal/tui/event_record_adapter.go` — `eventFromLogRecord(state.LogRecord) event.Event` mapper (or place in `internal/state` if more natural).

### Modified
- `internal/tui/live_output.go`
  - `formatEventLine` reduced to delegation: build options for live-TUI profile, call `display.EventLine`.
  - `formatTokenCount`/`formatCompactDuration` already delegate to display package — keep.
- `internal/tui/content.go`
  - `formatStoredEvent` reduced to delegation: convert `LogRecord → event.Event` view, call `display.EventLine` with the live-TUI profile (matches current output).
- `internal/display/progress.go`
  - `BasicProgressDisplay.EmitProgress` keeps state-tracking side effects (stepStates, stepOrder, handoverInfo) but writes its line via `display.EventLine` with the basic-cli profile (timestamp prefix, lowercase verbs, terminal-aware truncation).
- `internal/tui/live_output_test.go` — no changes expected (golden output preserved).
- `internal/display/progress_test.go` — no changes expected; if any test assertion is on internal helpers, retarget to public formatter.

### Deleted
- Inline switch-case body inside `BasicProgressDisplay.EmitProgress` that emits formatted strings (replaced by single `display.EventLine` call).
- Switch body of `formatEventLine` (replaced by delegation).
- Switch body of `formatStoredEvent` (replaced by delegation).

## Architecture Decisions

1. **Canonical home: `internal/display`** — TUI already imports display; display has no TUI dependency. Reverse direction would create a cycle.
2. **One input type**: `event.Event`. `state.LogRecord` is mapped via a thin adapter. Avoids generics or a second interface.
3. **Profiles via options struct**, not subclasses — `EventLineOpts{ Prefix Prefix, Color bool, StreamTruncate Truncator }`. Two predefined profiles: `LiveTUIProfile`, `BasicCLIProfile`. Keeps formatter pure-functional and easy to test.
4. **Callers retain side-effect responsibilities**: `BasicProgressDisplay` continues to manage step state, ordering, and handover metadata. Only the textual line generation moves.
5. **No public renaming**: keep `formatEventLine` / `formatStoredEvent` as package-local wrappers so call sites do not churn.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Subtle output drift (whitespace, ANSI codes) | Lock byte-exact output with golden tests *before* refactor. Diff fail = revert. |
| `state.LogRecord` lacks fields present on `event.Event` (`TokensIn/Out`, `ToolName/Target`) | Adapter populates only available fields; formatter's existing branches degrade gracefully (already do — see `formatStoredEvent`). |
| `BasicProgressDisplay` width-aware truncation differs from live-TUI fixed-60 | Encode via `Truncator` option: `FixedWidth(60)` for live, `TerminalAware(termInfo)` for CLI. |
| Casing difference (`completed` vs `Completed`) | Two pre-defined verb tables in profiles, not a single string. |
| NO_COLOR handling differs (live checks env per-call, CLI does not) | Pass `Color bool` via options; live-TUI profile reads NO_COLOR, CLI profile uses its own setting. |
| Hidden fourth caller emerges later | Add a single integration test that walks an event series through all three call sites and snapshots the union output. |

## Testing Strategy

1. **Pre-refactor golden capture**
   - Add `internal/tui/live_output_golden_test.go`: feed canonical event sequence through `formatEventLine`, snapshot output to `testdata/live_output.golden`.
   - Add `internal/tui/content_golden_test.go`: feed `LogRecord` sequence through `formatStoredEvent`, snapshot to `testdata/stored_events.golden`.
   - Add `internal/display/progress_golden_test.go`: feed event sequence through `BasicProgressDisplay`, snapshot stderr to `testdata/basic_progress.golden`.

2. **Canonical formatter unit tests**
   - Table-driven in `internal/display/eventline_test.go`: every event state × both profiles × `Color/NoColor`.
   - Edge cases: empty stepID, empty message, missing tokens, oversize tool target, contract states.

3. **Post-refactor verification**
   - Re-run golden tests; require byte-identical output.
   - Run `go test -race ./internal/tui/... ./internal/display/...`.
   - Run `go test -race ./...` to catch any unrelated regressions (live_output references in other packages).

4. **Manual verification**
   - `wave run` a small pipeline in non-TTY mode: confirm CLI output unchanged.
   - `wave tui` a finished run: confirm stored-event rendering unchanged.
   - `wave tui` a live run: confirm formatted live output unchanged.

## Out of Scope

- Refactoring `BubbleTeaProgressDisplay` (already a state machine, not a line formatter — only superficially similar to the three identified copies).
- Rewriting `internal/tui/pipeline_detail.go` `renderRunningInfo` / `renderFinishedDetail` (different concern: they render aggregated detail panes, not per-event lines).
- Public API changes — internal packages only.
