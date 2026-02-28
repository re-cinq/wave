# Tasks: Web UI Token Display

**Feature**: 098-webui-token-display
**Generated**: 2026-02-13
**Source**: spec.md, plan.md, data-model.md, research.md

## User Story Summary

| Story | Priority | Description |
|-------|----------|-------------|
| US-1 | P1 | Accurate Per-Step Token Counts in Web UI Run Detail |
| US-2 | P1 | Total Token Count in Web UI Run List |
| US-3 | P2 | Real-Time Token Display During Execution in Web UI |
| US-4 | P2 | Token Display in Pipeline Summary Header |
| US-5 | P1 | Consistent Token Counting Between TUI and Web UI |

---

## Phase 1: Foundational — Token Formatting (blocks all display tasks)

These tasks extend the core formatting function and register the template helper.
All subsequent phases depend on these.

- [X] T001 [P1] [US-5] Extend `FormatTokenCount` to handle M (million) and B (billion) thresholds — `internal/display/formatter.go:500-505`
  - Add `if tokens < 1_000_000` branch returning `"%.1fk"`
  - Add `if tokens < 1_000_000_000` branch returning `"%.1fM"`
  - Add fallback returning `"%.1fB"`
  - Keep existing `< 1000` and `< 1_000_000` (k) behavior unchanged

- [X] T002 [P1] [US-5] Add unit tests for `FormatTokenCount` M/B thresholds — `internal/display/formatter_test.go`
  - Add `TestFormatTokenCount` function with table-driven tests
  - Test vectors: 0→"0", 842→"842", 999→"999", 1000→"1.0k", 1500→"1.5k", 999999→"1000.0k", 1000000→"1.0M", 1500000→"1.5M", 999999999→"1000.0M", 1000000000→"1.0B", 2300000000→"2.3B"
  - Verify backward compatibility: existing k-range values unchanged

- [X] T003 [P1] [US-1,US-2] Register `formatTokens` template function in web UI template FuncMap — `internal/webui/embed.go:35-39`
  - Add `"formatTokens": formatTokensFunc` to the `template.FuncMap`
  - Implement `formatTokensFunc` accepting `interface{}` (int, int64) and delegating to `display.FormatTokenCount`
  - Add import for `internal/display` package if not already present

---

## Phase 2: US-1 — Accurate Per-Step Token Counts in Web UI Run Detail (P1)

Depends on: T001, T003

- [X] T004 [P1] [US-1] Fix step card template to show formatted tokens for completed/failed/running states — `internal/webui/templates/partials/step_card.html:17`
  - Replace `{{if gt .TokensUsed 0}}<span>Tokens: {{.TokensUsed}}</span>{{end}}`
  - With `{{if or (eq .State "completed") (eq .State "failed") (eq .State "running")}}<span>Tokens: {{formatTokens .TokensUsed}}</span>{{end}}`
  - This fixes FR-008 (zero tokens shown for completed/failed) and FR-001 (formatted display)

---

## Phase 3: US-2 — Total Token Count in Web UI Run List (P1)

Depends on: T001, T003

- [X] T005 [P1] [US-2] Format total tokens in run list row template — `internal/webui/templates/partials/run_row.html:8`
  - Replace `<td>{{.TotalTokens}}</td>` with `<td>{{formatTokens .TotalTokens}}</td>`
  - Ensures abbreviated display (e.g., "1.5k" instead of "1500") for FR-002

---

## Phase 4: US-5 — Consistent Token Counting Between TUI and Web UI (P1)

Depends on: T001

- [X] T006 [P1] [US-5] Make TUI completion summary always show total tokens — `cmd/wave/commands/run.go:324-331`
  - Remove the `if totalTokens > 0` conditional branch
  - Always print the format string with tokens: `"✓ Pipeline '%s' completed successfully (%.1fs, %s tokens)"`
  - When tokens are 0, this displays "0 tokens" (FR-010)

---

## Phase 5: US-4 — Token Display in Pipeline Summary Header (P2)

Depends on: T003

- [X] T007 [P2] [US-4] Add total tokens to run detail header — `internal/webui/templates/run_detail.html:7`
  - Append `| Tokens: {{formatTokens .Run.TotalTokens}}` to the `<p class="run-meta">` element
  - Sources from `RunSummary.TotalTokens` (authoritative, per CLR-003)
  - Displays alongside existing Run ID, Started, and Duration fields

---

## Phase 6: US-3 — Real-Time Token Display During Execution (P2)

Depends on: T004

- [X] T008 [P2] [US-3] Add JavaScript `formatTokens` function to SSE client — `internal/webui/static/sse.js` (top of file)
  - Implement `formatTokens(n)` with k/M/B thresholds matching Go `FormatTokenCount` logic
  - Handle `undefined`/`null` inputs returning `'0'`

- [X] T009 [P2] [US-3] Update `createStepCard()` to use `formatTokens()` and state-based display — `internal/webui/static/sse.js:142`
  - Replace `if (step.tokens_used > 0) metaParts.push(...)` with state-based check
  - Show tokens for `completed`, `failed`, and `running` states; omit for `pending`
  - Use `formatTokens(step.tokens_used)` for abbreviated display
  - Add CSS class `token-count` to the token span for SSE targeting

- [X] T010 [P2] [US-3] Add `updateStepCardTokens()` function for SSE-driven token updates — `internal/webui/static/sse.js`
  - Create function that finds a step card by step ID and updates its token span
  - If token span doesn't exist (step transitioned from pending to running), create it
  - Use `formatTokens()` for consistent formatting

- [X] T011 [P2] [US-3] Wire token updates into `handleSSEEvent()` — `internal/webui/static/sse.js:154`
  - After existing step card state updates, check for `data.step_id && data.tokens_used !== undefined`
  - Call `updateStepCardTokens(data.step_id, data.tokens_used)` for both `step_progress` and `completed` events
  - Ensures real-time token updates without page refresh (FR-005)

---

## Phase 7: Polish & Verification

- [X] T012 [P1] Run `go test ./internal/display/...` to verify formatter changes — N/A
  - Ensure T001 (FormatTokenCount extension) passes all existing and new tests
  - Verify backward compatibility: existing `FormatTokens` method tests still pass

- [X] T013 [P1] Run `go test ./internal/webui/...` to verify template function registration — N/A
  - Ensure T003 (formatTokens template function) doesn't break existing template rendering
  - Verify templates parse successfully with the new function in FuncMap

- [X] T014 [P1] Run `go test ./...` full test suite — N/A
  - Full project test suite must pass with all changes applied
  - Run with `-race` flag for race condition detection

- [X] T015 [P1] Verify contract compliance against `contracts/token-display.json` — `specs/098-webui-token-display/contracts/token-display.json`
  - Confirm FormatTokenCount test vectors match contract examples
  - Confirm step card, run row, and run detail templates use `formatTokens`
  - Confirm SSE events carry `tokens_used` for `completed` and `step_progress` events
  - Confirm TUI completion summary always shows tokens

---

## Dependency Graph

```
T001 (FormatTokenCount) ─┬─→ T002 (tests)
                         ├─→ T003 (template func) ─┬─→ T004 (step card) ──→ T008-T011 (SSE)
                         │                         ├─→ T005 (run row)
                         │                         └─→ T007 (run detail header)
                         └─→ T006 (TUI summary)

T012 ← T001, T002
T013 ← T003
T014 ← T001-T011
T015 ← T014
```

## Parallelization Notes

Tasks marked [P] can run in parallel with other tasks at the same phase level:
- T004, T005, T006, T007 are all parallelizable after T001+T003 complete
- T008, T009, T010, T011 are sequential (each builds on the previous)
- T012, T013 are parallelizable; T014 must follow both
