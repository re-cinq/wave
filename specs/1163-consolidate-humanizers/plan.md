# Implementation Plan — #1163

## 1. Objective

Collapse the four duration humanizers into a single canonical implementation in `internal/humanize`, delete the duplicates in `internal/display` and `internal/webui`, and migrate every caller.

## 2. Approach

Extend `internal/humanize` with the missing entry points so all callers can switch without per-call conversion code at the call site:

- Keep `humanize.Duration(d time.Duration) string` as the canonical formatter.
- Add `humanize.DurationMs(ms int64) string` — accepts `int64` milliseconds, used by `display.FormatDuration` callers and webui retro/compare handlers.
- Add `humanize.DurationSeconds(s float64) string` — accepts `float64` seconds, used by the webui template helper. Keeps the `<1s` short-form for sub-second values.

The webui template helper presently HTML-escapes its return. Since `text/template` and `html/template` already escape interpolated strings, the explicit `template.HTMLEscapeString` call is redundant when the helper returns plain text. The replacement helper returns plain text and lets the template engine escape; `embed_test.go` assertions update to match.

Output formatting unifies on the existing `humanize.Duration` style:
- zero → `-`
- `< 1s` → `<1s` (only via `DurationSeconds` and `DurationMs`)
- `< 1min` → `5s`
- `< 1h` → `3m15s`
- `>= 1h` → `2h30m`

This is one breaking visual change vs. the old `display.FormatDuration` and webui helpers (no space between unit groups, no `ms` suffix). That trade-off is intentional — uniform output is the point of consolidation. Tests are updated to reflect new style.

## 3. File Mapping

### Create / Modify

- `internal/humanize/humanize.go` — add `DurationMs(int64) string` and `DurationSeconds(float64) string`. Refactor `Duration` body so all three share one implementation.
- `internal/humanize/humanize_test.go` — extend with cases for the two new entry points (sub-second, ms range, edge cases).

### Modify (call-site migration)

- `internal/display/formatter.go` — delete `FormatDuration`. If callers outside the deleted set remain, leave a thin shim `FormatDuration = humanize.DurationMs` only if removing it cascades into 5+ unrelated packages; otherwise update callers directly.
- `internal/webui/embed.go` — delete `formatDuration` and `formatDurationShort` and `formatMinSec`. Replace template func registration with `humanize.DurationSeconds`.
- `internal/webui/handlers_runs.go` — delete `formatDurationValue`. Replace call sites (lines 608, 611, 880, 882) with `humanize.Duration(...)`.
- `internal/webui/handlers_compare.go` — replace `formatDurationValue` calls (lines 308, 340) with `humanize.Duration`.
- `internal/webui/handlers_retro_page.go` — replace `formatDurationValue` call (line 117) with `humanize.DurationMs(avgMs)`.
- `internal/webui/handlers_test.go:28` — update template func registration in test harness.
- `internal/webui/embed_test.go` — drop `formatDurationShort`/`formatDuration` tests OR rewrite as integration check that the template registers `humanize.DurationSeconds`.
- `internal/webui/handlers_runs_test.go` — drop `formatDurationValue` test, covered by humanize tests.
- `internal/tui/live_output.go:290` — replace `display.FormatDuration(d.Milliseconds())` with `humanize.Duration(d)`.
- `internal/tui/pipeline_list.go:838` — replace wrapper, drop `formatDuration` helper.
- `internal/tui/content.go:2203` — replace `display.FormatDuration(ev.DurationMs)` with `humanize.DurationMs(ev.DurationMs)`.

### Delete

- `internal/display/formatter.go` — `FormatDuration` function (~25 lines).
- `internal/webui/embed.go` — `formatDuration`, `formatDurationShort`, `formatMinSec` (~20 lines).
- `internal/webui/handlers_runs.go` — `formatDurationValue` function (~20 lines).
- `internal/webui/handlers_runs_test.go` — `TestFormatDurationValue` block (covered upstream).
- `internal/webui/embed_test.go` — `TestFormatDurationShort`, `TestFormatDuration`, `TestFormatDuration_HTMLEscaping` (covered upstream).

## 4. Architecture Decisions

- **One canonical style.** The issue explicitly asks for "single duration humanizer". The existing `humanize.Duration` style wins because it has the longest test coverage and matches ADR-003 layer conventions for the package. Visual consistency across CLI/TUI/WebUI is a feature, not a regression.
- **No HTML-escape inside humanize.** Escaping is the template engine's job. Removing the redundant escape simplifies the API and prevents double-escape bugs in non-template contexts.
- **No back-compat shim in `display`.** Pre-1.0; per memory rules, no deprecated functions retained.
- **Three entry points, not one generic.** `Duration(time.Duration)`, `DurationMs(int64)`, `DurationSeconds(float64)` — all three thin wrappers around a private `formatTotal(secs int)` core. Avoids forcing every caller to convert at the call site, which is the readability cost we noticed in the existing `internal/tui/pipeline_list.go` shim.

## 5. Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Visual regression in WebUI templates (no space between units, lost `ms` suffix) | Document in PR. Snapshot-test affected handlers if they exist; manually verify in browser. |
| `<1s` HTML-escape removed → template double-escapes | New helper returns plain text; `html/template` escapes once. Verified by template render test. |
| Out-of-scope humanizers (cmd/, audit/, tui/pipeline_list.go) drift | Issue explicitly scopes to display+webui. List remaining ones in PR description; file follow-up if owners want full sweep. |
| Existing callers in `internal/tui` break when `display.FormatDuration` deletes | Migrate the three TUI callers in same PR; CI catches anything missed. |

## 6. Testing Strategy

- **Unit:** Extend `internal/humanize/humanize_test.go` table-tests with new entry-point cases: zero, sub-second (`<1s`), ms boundary, hour boundary, large values.
- **Integration:** Run `go test ./internal/webui/... ./internal/tui/... ./internal/display/...` to catch any missed call site or template render error.
- **Build:** `go build ./...` to catch deletions of still-referenced symbols.
- **Lint:** `go vet ./...` and project linter.
- **Race:** `go test -race ./...` per project convention before PR.
- **Manual:** Boot `wave webui`, navigate to runs/retro/compare pages, confirm durations render. Boot `wave tui`, confirm pipeline list/live output show durations correctly.
