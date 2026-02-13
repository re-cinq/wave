# Research: Web UI Token Display

**Feature**: 098-webui-token-display
**Date**: 2026-02-13
**Branch**: `098-webui-token-display`

## Codebase Analysis

### Current Token Flow (End to End)

1. **Adapter layer** (`internal/adapter/claude.go`): Parses NDJSON stream from Claude CLI. Extracts `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens` from `usage` objects. Sums all four into `result.TokensUsed`. The generic `ProcessGroupRunner` falls back to `len(output)/4`.

2. **Executor** (`internal/pipeline/executor.go`): After step completion, stores `result.TokensUsed` in `execution.Results[stepID]["tokens_used"]`. Emits `completed` event with `TokensUsed` field. `GetTotalTokens()` sums all step results.

3. **State persistence** (`internal/state/store.go`): `UpdateRunStatus()` writes `total_tokens` to `pipeline_run` table. Per-step tokens are persisted via `event_log` entries (each event has `tokens_used` column).

4. **Event system** (`internal/event/emitter.go`): `Event` struct has `TokensUsed int` field. Events flow through `NDJSONEmitter` to stdout (NDJSON) and optionally to a `ProgressEmitter` (stderr TUI).

5. **TUI display** (`internal/display/progress.go`): `StepStatus.Render()` shows tokens for completed/failed steps: `"• {FormatTokenCount} tokens"`. Only shows when `TokensUsed > 0`.

6. **Web UI** (`internal/webui/`): Reads from state DB and event_log. `handlers_runs.go:buildStepDetails()` reconstructs per-step tokens from events. `run_row.html` displays `{{.TotalTokens}}` in run list. `step_card.html` displays `{{if gt .TokensUsed 0}}Tokens: {{.TokensUsed}}{{end}}`.

### Identified Gaps

| # | Gap | Location | Impact |
|---|-----|----------|--------|
| G1 | `FormatTokenCount` only handles "k" threshold | `internal/display/formatter.go:500-505` | Large token counts (1M+) display as "1000.0k" instead of "1.0M" |
| G2 | Step card hides zero tokens for completed steps | `step_card.html:17` | Violates FR-008: completed steps with 0 tokens show no token info |
| G3 | Run detail header missing total tokens | `run_detail.html:7` | Only shows duration, not tokens (FR-004) |
| G4 | SSE client doesn't update tokens in real-time | `sse.js:142` | `createStepCard` shows tokens but `handleSSEEvent` doesn't update existing card tokens |
| G5 | Raw integer display in web UI | `step_card.html:17`, `run_row.html:8` | Uses raw `{{.TokensUsed}}` instead of formatted string |
| G6 | TUI completion summary conditionally shows tokens | `cmd/wave/commands/run.go:324-331` | Shows tokens only when `> 0`, which is correct but summary line doesn't always print tokens |
| G7 | Web UI polling rebuilds step cards without token formatting | `sse.js:142` | `step.tokens_used` rendered as raw integer |

## Decisions

### D1: Token Formatting — Extend `FormatTokenCount` with M/B thresholds

**Decision**: Extend the existing `FormatTokenCount` function in `internal/display/formatter.go` to handle million (M) and billion (B) thresholds.

**Rationale**: The function is already the canonical formatter used by both TUI and web UI (via Go templates). Extending it maintains the single-source-of-truth pattern. No new functions needed.

**Alternatives Rejected**:
- *Create separate web-specific formatter*: Would diverge from TUI formatting, violating US-5 (consistency).
- *JavaScript-side formatting*: Would require maintaining formatting logic in two languages.

**Implementation**:
```
tokens < 1000       → "%d"       (e.g., "842")
tokens < 1_000_000  → "%.1fk"   (e.g., "1.5k")
tokens < 1_000_000_000 → "%.1fM" (e.g., "2.3M")
tokens >= 1_000_000_000 → "%.1fB" (e.g., "1.2B")
```

### D2: Template Function for Token Formatting

**Decision**: Register `FormatTokenCount` as a Go template function (`formatTokens`) available to all HTML templates, alongside the existing `formatTime` and `statusClass` functions.

**Rationale**: Templates currently render raw integers (`{{.TokensUsed}}`). A template function allows `{{formatTokens .TokensUsed}}` which produces the same abbreviated output as the TUI. This eliminates the need for JavaScript formatting.

**Alternatives Rejected**:
- *Pre-format in handler*: Would require adding a `FormattedTokens string` field to every type. Clutters the data model with display concerns.
- *JavaScript formatter*: Two sources of truth for formatting logic.

### D3: Step Card Token Display Logic (FR-008)

**Decision**: Change step card template to always show tokens for completed and failed states, show current cumulative for running, and omit for pending.

**Rationale**: Directly implements FR-008. The current `{{if gt .TokensUsed 0}}` guard incorrectly hides zero-token completed steps.

**Implementation**:
```html
{{if or (eq .State "completed") (eq .State "failed")}}
<span>Tokens: {{formatTokens .TokensUsed}}</span>
{{else if eq .State "running"}}
<span>Tokens: {{formatTokens .TokensUsed}}</span>
{{end}}
```

### D4: Run Detail Header — Total Tokens from `RunSummary.TotalTokens`

**Decision**: Add total tokens to the `run-meta` paragraph in `run_detail.html`, sourced from `RunSummary.TotalTokens` (which maps to `pipeline_run.total_tokens` in the database).

**Rationale**: Per CLR-003, `RunSummary.TotalTokens` is the single source of truth. It's already populated by `runToSummary()` from `RunRecord.TotalTokens`. No new DB queries needed.

**Implementation**: Add `| Tokens: {{formatTokens .Run.TotalTokens}}` to the `run-meta` paragraph.

### D5: SSE Real-Time Token Updates

**Decision**: Enhance SSE client-side JavaScript to update step card token counts on both `step_progress` and `completed` events.

**Rationale**: The SSE data already includes `tokens_used` in event payloads. The client just needs to extract and display it. The polling fallback (`updatePageFromAPI`) already rebuilds cards with token data.

**Implementation**:
- In `handleSSEEvent`: extract `data.tokens_used` and update the step card's token span.
- In `createStepCard`: Use `formatTokensJS()` helper for consistent abbreviated formatting.
- Add a small JS `formatTokens(n)` function that mirrors the Go logic (k/M/B thresholds).

### D6: Run List Token Display — Use `formatTokens` Template Function

**Decision**: Replace `{{.TotalTokens}}` in `run_row.html` with `{{formatTokens .TotalTokens}}` for abbreviated display.

**Rationale**: Large token counts (e.g., 1500000) should display as "1.5M" not "1500000". The template function provides consistent formatting.

### D7: TUI Completion Summary — Always Show Tokens (FR-010)

**Decision**: The TUI completion summary in `cmd/wave/commands/run.go` already shows tokens when `> 0`. For FR-010 compliance, always display total tokens in the summary line (showing "0" when no tokens were used).

**Rationale**: FR-010 requires the pipeline completion summary to display total tokens alongside elapsed time. The current conditional display (`if totalTokens > 0`) means zero-token runs don't show token info. This should be unconditional.

## Technology Choices

### No New Dependencies

All changes use existing Go stdlib (`html/template`, `fmt`, `net/http`) and the existing `internal/display`, `internal/webui`, and `internal/event` packages. No new Go modules required.

### Go Template Functions vs JavaScript

For server-rendered pages, Go template functions handle formatting. For SSE-driven dynamic updates, a small JavaScript formatting function (~10 lines) mirrors the Go logic. This is acceptable duplication since the logic is trivial and keeping it synchronized is straightforward.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Token format inconsistency between JS and Go | Low | Medium | Unit test Go function, test JS function in SSE integration test |
| Breaking existing template rendering | Low | High | All template changes are additive (adding token display, not removing existing elements) |
| SSE event payload missing tokens_used | Low | Low | Polling fallback rebuilds cards from API every 3s, ensuring eventual consistency |
| FormatTokenCount changes break existing callers | Low | Medium | Only adding new thresholds above 1M; existing k-threshold behavior unchanged |
