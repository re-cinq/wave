# Display Consistency Checklist

**Feature**: 098-webui-token-display
**Domain**: Cross-interface token display parity (TUI, Web UI, Database)
**Date**: 2026-02-13

This checklist validates that requirements adequately specify consistent token display
across all three rendering surfaces: TUI, Web UI (server-rendered), and Web UI (SSE/JS-rendered).

---

## Formatting Parity

- [ ] CHK-DC001 - Are formatting rules specified identically for Go `FormatTokenCount`, Go template `formatTokens`, and JavaScript `formatTokens`? [Completeness]
- [ ] CHK-DC002 - Is the rounding behavior (`.toFixed(1)` in JS vs `"%.1f"` in Go) specified to produce identical results for all threshold boundary values? [Clarity]
- [ ] CHK-DC003 - Is the spec explicit about whether trailing zeros are acceptable (e.g., "1.0k" for exactly 1000 tokens vs. "1k")? [Clarity]
- [ ] CHK-DC004 - Are formatting requirements specified for the maximum possible token value (int max / int64 max)? [Completeness]

## State-Dependent Display Rules

- [ ] CHK-DC005 - Are the four step states (pending, running, completed, failed) exhaustively mapped to token display behavior in the spec, not just the plan? [Completeness]
- [ ] CHK-DC006 - Is the token display rule for "cancelled" or "skipped" step states addressed, or are these states not possible? [Completeness]
- [ ] CHK-DC007 - Is the distinction between "omit token field" (pending) and "show 0" (completed with zero tokens) specified clearly enough to prevent implementer confusion? [Clarity]
- [ ] CHK-DC008 - Does the spec define whether the "running" state token display should show a loading indicator or just the raw cumulative count? [Completeness]

## Cross-Interface Consistency

- [ ] CHK-DC009 - Is the authoritative data source for total tokens (`pipeline_run.total_tokens`) explicitly named in the spec (not just the plan and clarifications)? [Completeness]
- [ ] CHK-DC010 - Is a requirement specified that prevents client-side summation from diverging from the DB-persisted total? [Clarity]
- [ ] CHK-DC011 - Are requirements defined for what the web UI displays if `buildStepDetails()` reconstructs per-step tokens that don't sum to `RunSummary.TotalTokens`? [Completeness]
- [ ] CHK-DC012 - Is the "tokens" suffix usage rule (bare in compact contexts, labelled in header) specified in the spec's FRs, or only in the clarification CLR-001? [Completeness]

## Token Display Context

- [ ] CHK-DC013 - Are formatting requirements specified for each distinct display location (run list column, step card, run detail header, TUI step line, TUI summary)? [Coverage]
- [ ] CHK-DC014 - Is the visual placement of token data within the step card specified (inline with duration? separate line? after status?)? [Completeness]
- [ ] CHK-DC015 - Are requirements specified for token display ordering relative to other metadata (e.g., always "Duration | Tokens" not "Tokens | Duration")? [Completeness]
