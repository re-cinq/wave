# Requirements Quality Review Checklist

**Feature**: 098-webui-token-display
**Date**: 2026-02-13
**Artifacts Reviewed**: spec.md, plan.md, tasks.md, research.md, data-model.md, contracts/token-display.json

---

## Completeness

- [ ] CHK001 - Are all 10 functional requirements (FR-001 through FR-010) traceable to at least one user story? [Completeness]
- [ ] CHK002 - Does every user story have at least one acceptance scenario that exercises the primary success path? [Completeness]
- [ ] CHK003 - Are error/failure states specified for all token display locations (step card, run list, run header, TUI summary)? [Completeness]
- [ ] CHK004 - Is the behavior for negative token values defined (e.g., integer overflow, corrupted DB data)? [Completeness]
- [ ] CHK005 - Are requirements defined for what happens when the database column `total_tokens` is NULL vs. 0? [Completeness]
- [ ] CHK006 - Is the behavior specified when a browser loads the run detail page mid-execution (neither cold load nor initial SSE)? [Completeness]
- [ ] CHK007 - Are accessibility requirements defined for token display elements (screen reader labels, ARIA attributes)? [Completeness]
- [ ] CHK008 - Is the formatting behavior specified for exactly 1000, 1000000, and 1000000000 boundary values? [Completeness]
- [ ] CHK009 - Are requirements specified for the token column header label in the run list table? [Completeness]

## Clarity

- [ ] CHK010 - Is "accurate token counts" unambiguously defined — does it mean exact integer match or within a tolerance? [Clarity]
- [ ] CHK011 - Does FR-003 clearly distinguish which adapters provide "actual adapter-reported usage" vs. byte-estimate fallback? [Clarity]
- [ ] CHK012 - Is the term "cache tokens" used consistently — does the spec always enumerate all four token types (input, output, cache_read_input, cache_creation_input)? [Clarity]
- [ ] CHK013 - Is the phrase "without page refresh" in FR-005 sufficiently precise — does it mean no full page reload, or also no AJAX polling? [Clarity]
- [ ] CHK014 - Is "within 1 second" (SC-003) measured from step completion event emission or from adapter subprocess exit? [Clarity]
- [ ] CHK015 - Does CLR-001 clearly resolve whether the summary header uses "1.5k" (bare) or "1.5k tokens" (labelled)? The resolution says "may use" labelled format — is this ambiguous? [Clarity]
- [ ] CHK016 - Is "byte-estimate fallback value (or zero)" in US-1 scenario 3 unambiguous about when each applies? [Clarity]

## Consistency

- [ ] CHK017 - Is the JavaScript `formatTokens` function specified to produce identical output to Go `FormatTokenCount` for all test vectors in the contract? [Consistency]
- [ ] CHK018 - Does the plan's step card template change (Change 3) match the research decision D3 exactly? D3 uses separate `{{else if}}` while the plan uses `{{if or ...}}` — are both functionally equivalent? [Consistency]
- [ ] CHK019 - Does the test vector `999999 → "1000.0k"` in tasks.md T002 match the contract's test vector `999999 → "1000.0k"`? (Confirming no rounding discrepancy) [Consistency]
- [ ] CHK020 - Is the `formatTokensFunc` wrapper (accepting `interface{}`) in plan Change 2 consistent with the data-model.md which shows direct `display.FormatTokenCount` registration (no wrapper)? [Consistency]
- [ ] CHK021 - Are SSE event field names (`tokens_used` vs `TokensUsed`) consistent between the Go event struct, SSE JSON payload, and JavaScript handler references? [Consistency]
- [ ] CHK022 - Is the TUI completion summary format string consistent between plan Change 7 (using `%s tokens`) and the existing TUI progress display format (`FormatTokenCount` + " tokens")? [Consistency]

## Coverage

- [ ] CHK023 - Are test requirements specified for the JavaScript `formatTokens` function, or only for the Go `FormatTokenCount`? [Coverage]
- [ ] CHK024 - Are integration test requirements defined for the SSE token update flow (event emission through DOM update)? [Coverage]
- [ ] CHK025 - Do the success criteria (SC-001 through SC-006) cover all 5 user stories, or are any user stories unverified? [Coverage]
- [ ] CHK026 - Are edge cases defined for concurrent SSE updates to the same step card (e.g., rapid step_progress events)? [Coverage]
- [ ] CHK027 - Is there a requirement or test for the polling fallback path (when SSE fails) delivering correct token data? [Coverage]
- [ ] CHK028 - Are regression test requirements defined to verify existing k-range formatting is unchanged after adding M/B thresholds? [Coverage]
- [ ] CHK029 - Is the contract `token-display.json` validated by any automated process, or is it documentation-only? [Coverage]
