# Feature Specification: Web UI Token Display

**Feature Branch**: `098-webui-token-display`
**Created**: 2026-02-13
**Status**: Draft
**Input**: User description: "https://github.com/re-cinq/wave/issues/98 same goes for the webui"
**Issue**: [#98](https://github.com/re-cinq/wave/issues/98) - fix: incorrect token count in pipeline summary and missing token display in TUI output

## Context

Issue #98 identifies two problems with token display in Wave's TUI:

1. **Incorrect token counts** - The token values shown in the pipeline completion summary do not match expected actual LLM API usage (input + output tokens).
2. **Missing token display in TUI** - The interactive TUI progress display only shows elapsed time per step, not token usage. The pipeline summary header also only shows elapsed time, not total tokens.

The user request extends this to the **Web UI**: "same goes for the webui" - meaning the web dashboard should also display accurate, visible token counts in the same places the TUI does (and currently doesn't).

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Accurate Per-Step Token Counts in Web UI Run Detail (Priority: P1)

A user viewing a completed pipeline run in the web dashboard needs to see accurate token counts for each step. The token counts must reflect the actual LLM API usage (input + output tokens) as reported by the adapter, not estimates or byte-based approximations.

**Why this priority**: Token accuracy is the core issue. If displayed tokens are wrong, showing them more prominently only amplifies misinformation. This must be fixed first.

**Independent Test**: Can be fully tested by running a pipeline, comparing the token counts shown in the web UI step cards against the raw NDJSON output from the adapter subprocess. The counts must match.

**Acceptance Scenarios**:

1. **Given** a completed pipeline run with 3 steps, **When** the user views the run detail page in the web UI, **Then** each step card shows a token count that matches the cumulative input + output tokens reported by the adapter for that step.
2. **Given** a pipeline step that used cached prompt tokens, **When** the user views the step in the web UI, **Then** the token count includes cache_read_input_tokens and cache_creation_input_tokens in the total, matching the adapter's reported usage.
3. **Given** a pipeline step where the adapter returned no structured usage data (e.g., non-Claude adapter), **When** the user views the step in the web UI, **Then** the token count shows the byte-estimate fallback value (or zero), and does not display a misleading exact number.

---

### User Story 2 - Total Token Count in Web UI Run List (Priority: P1)

A user browsing the pipeline runs list in the web dashboard needs to see accurate total token counts per run. This total must be the sum of all step-level token counts for that run.

**Why this priority**: The run list is the primary dashboard view. Accurate totals here let users quickly assess LLM usage across runs without drilling into each one.

**Independent Test**: Can be fully tested by running a pipeline, then checking the runs list page - the TotalTokens column must equal the sum of all per-step tokens shown in the run detail.

**Acceptance Scenarios**:

1. **Given** a completed pipeline run, **When** the user views the runs list page, **Then** the TotalTokens column shows the sum of all step-level token counts for that run.
2. **Given** a pipeline run that is still in progress, **When** the user views the runs list page, **Then** the TotalTokens column shows the cumulative tokens consumed so far (sum of completed steps).
3. **Given** a pipeline run where all steps reported zero tokens, **When** the user views the runs list, **Then** the TotalTokens column shows 0 (not blank or missing).

---

### User Story 3 - Real-Time Token Display During Execution in Web UI (Priority: P2)

A user watching a pipeline execute in real-time via the web UI's SSE-powered run detail page should see token counts update as steps complete, without needing to refresh.

**Why this priority**: Real-time feedback is a core UX feature of the web dashboard. Tokens are a key execution metric alongside elapsed time and should be surfaced with the same immediacy.

**Independent Test**: Can be tested by starting a pipeline via the web UI, watching the run detail page, and confirming token counts appear on step cards as each step completes - without manual page refresh.

**Acceptance Scenarios**:

1. **Given** a pipeline is executing and the user has the run detail page open, **When** a step completes and emits a completion event with token data, **Then** the step card updates to show the token count via SSE without page refresh.
2. **Given** a pipeline is executing and the user has the run detail page open, **When** intermediate `step_progress` events are emitted with token data (cumulative tokens from the adapter's streaming output), **Then** the step card shows the latest cumulative token count for the in-progress step. Note: Token counts during execution are cumulative per-turn values from the adapter's NDJSON stream, not final totals. The authoritative per-step token count is set on the `completed` event.
3. **Given** the SSE connection is interrupted and re-established, **When** the page reconnects, **Then** the token counts are restored from the current state (not lost).

---

### User Story 4 - Token Display in Pipeline Summary Header (Priority: P2)

A user viewing a completed run in the web UI should see total token usage in the run summary/header area alongside elapsed time, mirroring what the TUI shows (or should show) in its completion summary.

**Why this priority**: The summary header is the first thing users see on the run detail page. Including tokens alongside elapsed time gives an at-a-glance view of both time and cost dimensions.

**Independent Test**: Can be tested by viewing a completed run's detail page and confirming the header/summary area shows both total elapsed time and total tokens.

**Acceptance Scenarios**:

1. **Given** a completed pipeline run, **When** the user views the run detail page, **Then** the summary header shows total elapsed time AND total token count.
2. **Given** a failed pipeline run, **When** the user views the run detail page, **Then** the summary header shows tokens consumed up to the failure point.

---

### User Story 5 - Consistent Token Counting Between TUI and Web UI (Priority: P1)

A user who runs a pipeline via the CLI (TUI mode) and then views the same run in the web dashboard should see identical token counts in both interfaces.

**Why this priority**: Consistency between interfaces is essential for trust. If TUI and web UI show different numbers for the same run, users cannot trust either.

**Independent Test**: Can be tested by running a pipeline via CLI, noting the TUI token output, then opening the web UI and comparing - numbers must match.

**Acceptance Scenarios**:

1. **Given** a pipeline run completed via CLI, **When** the user views the same run in the web UI, **Then** the per-step and total token counts are identical to what the TUI displayed.
2. **Given** a pipeline run started via the web UI, **When** the user views the run in both the web UI detail page and queries the state database directly, **Then** the token counts match.

---

### Edge Cases

- What happens when a step completes but the adapter reports zero tokens (e.g., dry run, mock adapter)?
  - Token count should display as 0, not omitted or hidden.
- What happens when multiple adapter types are used in the same pipeline (e.g., Claude + a custom adapter)?
  - Each step shows tokens from its own adapter. The total is the sum. No cross-adapter normalization is applied.
- What happens when a step is retried and succeeds on the second attempt?
  - Token count should reflect the successful attempt's tokens. Retry token usage from failed attempts should not be double-counted in the total unless explicitly tracked separately.
- What happens when the web UI is opened after a pipeline completed (cold load vs. SSE)?
  - Token data must be available from the persisted state (database), not only from live SSE events.
- What happens when token values exceed display formatting thresholds (e.g., 1M+ tokens)?
  - Display should use human-readable formatting (e.g., "1.2M") consistent with the `FormatTokenCount` function. The function must be extended to handle M (million) and B (billion) thresholds beyond the current "k" (thousand) support.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The web UI run detail page MUST display per-step token counts on each step card.
- **FR-002**: The web UI runs list page MUST display total token count per run.
- **FR-003**: Token counts displayed in the web UI MUST match the actual adapter-reported usage (input_tokens + output_tokens + cache tokens), not estimates.
- **FR-004**: The web UI run detail page MUST display total token count in the run summary/header area (the `<p class="run-meta">` element) alongside elapsed time. The total token count is sourced from `RunSummary.TotalTokens` (which maps to `pipeline_run.total_tokens` in the database), the same authoritative source used by the TUI completion summary. This value is updated by the executor via `UpdateRunStatus` as steps complete.
- **FR-005**: Token counts in the web UI MUST update in real-time via SSE as steps complete, without requiring page refresh.
- **FR-006**: Token counts persisted in the database MUST be accurate and consistent with what both TUI and web UI display.
- **FR-007**: The web UI MUST display token counts using human-readable abbreviated formatting consistent with TUI formatting. The existing `FormatTokenCount` function handles values up to "k" (e.g., "1.5k"); it MUST be extended to also handle millions ("M") and billions ("B") thresholds. Token values below 1,000 are displayed as raw integers (e.g., "842"). The web UI displays the abbreviated number only (no "tokens" suffix) in tabular/compact contexts (run list, step card metadata), matching the TUI's step-completion format.
- **FR-008**: The web UI MUST display zero tokens as "0" rather than hiding or omitting the token field for completed and failed steps. For steps in "pending" or "not started" state (where no adapter has run), the token field MAY be omitted or shown as "-" since no token data exists yet. For "running" steps, the token field SHOULD show the current cumulative count (which may be 0 at the start).
- **FR-009**: The token counting logic in adapters that support structured usage data (e.g., Claude adapter) MUST correctly sum input_tokens, output_tokens, cache_read_input_tokens, and cache_creation_input_tokens for accurate totals. For adapters that do not provide structured token usage (e.g., the generic `ProcessGroupRunner`), the existing byte-estimate fallback (`len(output)/4`) is acceptable. No visual distinction is required between exact and estimated counts — the priority is displaying _something_ rather than nothing.
- **FR-010**: The TUI pipeline completion summary MUST display total tokens alongside elapsed time (fixing the TUI gap identified in issue #98).

### Key Entities

- **TokenUsage**: Represents the token consumption for a single pipeline step. Key attributes: input_tokens, output_tokens, cache_tokens, total (computed sum). Source: adapter subprocess NDJSON output.
- **RunSummary.TotalTokens**: Aggregate token count across all steps in a pipeline run. Persisted in the `pipeline_run.total_tokens` database column.
- **StepDetail.TokensUsed**: Per-step token count displayed in the web UI step card and TUI step completion line.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Token counts shown in the web UI for any completed step deviate by 0% from the adapter's reported token usage (exact match required).
- **SC-002**: Total tokens shown in the web UI run list equal the sum of all per-step tokens for that run (verifiable by database query).
- **SC-003**: A user viewing a pipeline execution in the web UI sees token counts appear on step cards within 1 second of step completion (via SSE), without page refresh.
- **SC-004**: Token counts for the same pipeline run are identical when viewed in the TUI, web UI, and queried directly from the database.
- **SC-005**: All existing token-related tests pass, and new tests cover the token display paths in the web UI handlers.
- **SC-006**: The pipeline completion summary (both TUI and web UI) displays total tokens alongside elapsed time for 100% of completed runs.

## Clarifications _(resolved during specification refinement)_

### CLR-001: Token formatting thresholds and display style

**Question**: The spec references "1.5k" and "2.3M" formatting, but the existing `FormatTokenCount` function only handles up to "k" (thousands). Additionally, `FormatTokenCount` returns bare numbers ("1.5k") while `FormatTokens` returns labeled numbers ("1.5k tokens"). Which format should the web UI use?

**Resolution**: Extend `FormatTokenCount` to handle M (million) and B (billion) thresholds. The web UI uses the abbreviated bare format (no "tokens" suffix) in compact/tabular contexts (run list columns, step card metadata lines), matching the TUI's step-completion format (`FormatTokenCount`). The summary header may use the labelled format with "tokens" suffix for clarity since it has more horizontal space.

**Rationale**: The TUI already uses `FormatTokenCount` (bare) in step completion lines and `FormatTokenCount` with " tokens" appended in progress display. Following the same pattern keeps TUI/Web UI visually consistent. Extending to M/B is trivial and future-proofs for long-running pipelines.

### CLR-002: Zero-token display for steps that haven't executed

**Question**: FR-008 says "display zero tokens as 0 rather than hiding." But should this apply to steps in "pending" state that haven't started yet? Showing "0" for a step that hasn't run could be misleading.

**Resolution**: FR-008 applies to completed and failed steps. For pending/not-started steps, the token field may be omitted or shown as "-". For running steps, show the current cumulative count (which starts at 0).

**Rationale**: The current `step_card.html` template uses `{{if gt .TokensUsed 0}}` which hides zero tokens. The fix should change this to always show tokens for terminal states (completed/failed) but omit for pending states where no adapter has executed. This matches user expectations — "0 tokens" after completion is meaningful data; "0 tokens" before execution is absence of data.

### CLR-003: Token source for run detail summary header

**Question**: FR-004 says the summary header should show total tokens, but doesn't specify the data source. The web UI could sum step-level tokens client-side, or use `RunSummary.TotalTokens` from the database. These could differ if `UpdateRunStatus` and `buildStepDetails` compute tokens differently.

**Resolution**: Use `RunSummary.TotalTokens` (mapped from `pipeline_run.total_tokens` in the database) as the single source of truth. This is the same value the TUI uses via `executor.GetTotalTokens()` and what `UpdateRunStatus` persists. The `run-meta` paragraph in `run_detail.html` should render this value directly.

**Rationale**: Using a single authoritative source eliminates TUI/Web UI divergence risk. The `pipeline_run.total_tokens` column is already updated by the executor on each step completion via `UpdateRunStatus`, so it's always current. Client-side summation would be fragile and could diverge from the persisted total.

### CLR-004: SSE token update granularity — during execution vs. on completion

**Question**: US-3 scenario 2 mentions "intermediate progress events with token data." Should tokens update live during a step (showing cumulative per-turn tokens from the NDJSON stream), or only when a step completes?

**Resolution**: Both. The executor already emits `step_progress` events with `TokensUsed` from the adapter's streaming output during execution, and `completed` events with the final token count. The SSE client should update the step card on both event types. The `completed` event's token count is authoritative; intermediate counts are best-effort cumulative values.

**Rationale**: The infrastructure already exists — `OnStreamEvent` callback feeds cumulative tokens into `step_progress` events. Not surfacing them would waste available data. The key constraint is that the `completed` event's token count (from `parseOutput`) must overwrite any intermediate value to ensure final accuracy.

### CLR-005: Byte-estimate tokens for non-Claude adapters vs. FR-009 accuracy requirement

**Question**: FR-009 requires "correctly sum input_tokens, output_tokens, cache_read_input_tokens, and cache_creation_input_tokens," but the generic `ProcessGroupRunner` uses `len(text)/4` byte estimation. Is this a contradiction?

**Resolution**: FR-009 applies to adapters that provide structured usage data (primarily the Claude adapter). For adapters without structured token reporting, the byte-estimate fallback is acceptable and no visual distinction is needed in the UI. US-1 scenario 3 already acknowledges this: "shows the byte-estimate fallback value (or zero)."

**Rationale**: Wave's adapter architecture is heterogeneous — different adapters have different capabilities. Requiring structured token data from all adapters would block this feature on adapter upgrades. The byte-estimate is a reasonable approximation and better than showing nothing. Users who need exact counts will use Claude-compatible adapters.
