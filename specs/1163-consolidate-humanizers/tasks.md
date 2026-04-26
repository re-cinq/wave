# Work Items — #1163

## Phase 1: Setup
- [ ] 1.1: Confirm branch `1163-consolidate-humanizers` checked out from main
- [ ] 1.2: Re-verify the four humanizer locations match `spec.md` table (no drift since assessment)

## Phase 2: Core Implementation
- [ ] 2.1: Refactor `internal/humanize/humanize.go` — extract private `formatTotalSeconds(int)` core, add `DurationMs(int64)`, `DurationSeconds(float64)` entry points
- [ ] 2.2: Extend `internal/humanize/humanize_test.go` with table cases for the new entry points (sub-second, ms, hour boundary, negative input)

## Phase 3: Caller Migration (parallelizable per package)
- [ ] 3.1 [P]: `internal/display/formatter.go` — delete `FormatDuration`
- [ ] 3.2 [P]: `internal/webui/embed.go` — replace template helper registration with `humanize.DurationSeconds`; delete `formatDuration`, `formatDurationShort`, `formatMinSec`
- [ ] 3.3 [P]: `internal/webui/handlers_runs.go` — replace `formatDurationValue` call sites with `humanize.Duration`; delete the function
- [ ] 3.4 [P]: `internal/webui/handlers_compare.go` — replace `formatDurationValue` calls
- [ ] 3.5 [P]: `internal/webui/handlers_retro_page.go` — replace with `humanize.DurationMs`
- [ ] 3.6 [P]: `internal/tui/live_output.go` — swap `display.FormatDuration` for `humanize.Duration`
- [ ] 3.7 [P]: `internal/tui/pipeline_list.go` — swap shim, delete local `formatDuration`
- [ ] 3.8 [P]: `internal/tui/content.go` — swap `display.FormatDuration` for `humanize.DurationMs`
- [ ] 3.9: `internal/webui/handlers_test.go` — update template func registration

## Phase 4: Test Cleanup
- [ ] 4.1: Delete `TestFormatDurationShort`, `TestFormatDuration`, `TestFormatDuration_HTMLEscaping` from `internal/webui/embed_test.go`
- [ ] 4.2: Delete `TestFormatDurationValue` from `internal/webui/handlers_runs_test.go`
- [ ] 4.3: Add a small webui template-render assertion confirming durations render unescaped via the new helper

## Phase 5: Verification
- [ ] 5.1: `go build ./...`
- [ ] 5.2: `go test -race ./...`
- [ ] 5.3: `go vet ./...` + project linter
- [ ] 5.4: Manual: launch `wave webui`, visit `/runs`, `/retro`, `/compare`; confirm durations render
- [ ] 5.5: Manual: launch `wave tui`, confirm pipeline list and live-output durations render

## Phase 6: Polish
- [ ] 6.1: PR description lists the four removed locations and any visual-format changes
- [ ] 6.2: Note follow-up audit for `cmd/`, `internal/audit/`, `internal/tui/pipeline_list.go` cousins (out of scope)
