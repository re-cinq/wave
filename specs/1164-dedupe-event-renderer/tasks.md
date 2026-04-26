# Work Items — #1164 Dedupe Event-Line Renderer

## Phase 1: Lock Current Behavior

- [ ] 1.1: Add `internal/tui/live_output_golden_test.go` capturing canonical event sequence rendered by `formatEventLine` to `testdata/live_output.golden`.
- [ ] 1.2: Add `internal/tui/content_golden_test.go` capturing canonical `state.LogRecord` sequence rendered by `formatStoredEvent` to `testdata/stored_events.golden`. [P]
- [ ] 1.3: Add `internal/display/progress_golden_test.go` capturing canonical event sequence written by `BasicProgressDisplay.EmitProgress` to `testdata/basic_progress.golden`. [P]
- [ ] 1.4: Run all three golden tests; commit golden fixtures as the baseline.

## Phase 2: Canonical Formatter

- [ ] 2.1: Create `internal/display/eventline.go` with `EventLine(evt event.Event, opts EventLineOpts) string`.
- [ ] 2.2: Define `EventLineOpts` (`Prefix`, `Color`, `StreamTruncate Truncator`) and verb/symbol tables for live-TUI and basic-CLI profiles.
- [ ] 2.3: Define exported profile builders: `LiveTUIProfile()`, `BasicCLIProfile(termInfo *TerminalInfo, timestamp string)`.
- [ ] 2.4: Add `internal/display/eventline_test.go` — table-driven unit tests covering every event state × both profiles × color/no-color. [P]

## Phase 3: Adapter

- [ ] 3.1: Add `eventFromLogRecord(state.LogRecord) event.Event` (place in `internal/tui/event_record_adapter.go` or `internal/state` if cleaner).
- [ ] 3.2: Unit test the adapter for round-trip of all LogRecord fields used by the formatter. [P]

## Phase 4: Migrate Call Sites

- [ ] 4.1: Replace `internal/tui/live_output.go` `formatEventLine` body with delegation to `display.EventLine` using `LiveTUIProfile`.
- [ ] 4.2: Replace `internal/tui/content.go` `formatStoredEvent` body with adapter call + delegation to `display.EventLine` using `LiveTUIProfile`.
- [ ] 4.3: Replace `BasicProgressDisplay.EmitProgress` line-emission switch (`internal/display/progress.go:660-766`) with single call to `display.EventLine` using `BasicCLIProfile`. Keep state-tracking side effects (stepStates, stepOrder, handoverInfo population, renderHandoverMetadata).
- [ ] 4.4: Run `go build ./...` and fix any signature drift.

## Phase 5: Verification

- [ ] 5.1: Run `go test ./internal/tui/... ./internal/display/...` — all golden tests must produce zero diff.
- [ ] 5.2: Run `go test -race ./...` to detect regressions in unrelated packages.
- [ ] 5.3: Run `golangci-lint run ./internal/tui/... ./internal/display/...`.
- [ ] 5.4: Manual validation: `wave run` a pipeline in non-TTY mode, capture output, diff against pre-refactor capture.
- [ ] 5.5: Manual validation: `wave tui` against a finished run; verify stored-event rendering unchanged.

## Phase 6: Cleanup & Polish

- [ ] 6.1: Remove any orphaned helpers in `internal/tui` whose sole consumer was the former `formatEventLine` switch body.
- [ ] 6.2: Update doc comments on `formatEventLine` / `formatStoredEvent` to point to `display.EventLine` as the canonical implementation.
- [ ] 6.3: Verify no new dead code via `go vet` and `staticcheck`.
- [ ] 6.4: Final `go test -race ./...` and `golangci-lint run ./...`.
