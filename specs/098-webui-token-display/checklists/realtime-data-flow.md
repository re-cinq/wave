# Real-Time Data Flow Checklist

**Feature**: 098-webui-token-display
**Domain**: SSE event pipeline and token data integrity
**Date**: 2026-02-13

This checklist validates that requirements for the real-time SSE token update flow
are sufficiently specified, including event payloads, client-side handling, and fallback behavior.

---

## Event Payload Specification

- [ ] CHK-RT001 - Are the SSE event types that carry token data (`completed`, `step_progress`) explicitly listed in the spec's FRs, or only in the plan? [Completeness]
- [ ] CHK-RT002 - Is it specified whether `step_progress` events always include `tokens_used`, or only when the adapter provides streaming token data? [Clarity]
- [ ] CHK-RT003 - Is the behavior defined when an SSE event carries `tokens_used: 0` vs. when `tokens_used` is absent from the payload? [Clarity]
- [ ] CHK-RT004 - Are requirements specified for which SSE event types should trigger a total tokens update in the run header (not just per-step cards)? [Completeness]

## Client-Side Update Semantics

- [ ] CHK-RT005 - Is the requirement clear that `completed` event token values must overwrite (not add to) intermediate `step_progress` values? [Clarity]
- [ ] CHK-RT006 - Are requirements defined for handling out-of-order SSE events (e.g., `completed` arriving before the last `step_progress`)? [Completeness]
- [ ] CHK-RT007 - Is the DOM update strategy specified — should the client create a new token element if one doesn't exist (step transitioning from pending to running)? [Completeness]
- [ ] CHK-RT008 - Are requirements defined for debouncing rapid `step_progress` token updates to avoid excessive DOM mutations? [Completeness]

## Fallback and Recovery

- [ ] CHK-RT009 - Is the SSE reconnection behavior specified in the spec (US-3 scenario 3 mentions it), and does it address token restoration from persisted state? [Completeness]
- [ ] CHK-RT010 - Is the polling fallback interval (noted as 3s in research) specified in the requirements, or only observed from existing code? [Completeness]
- [ ] CHK-RT011 - Are requirements defined for reconciling token values between SSE-delivered data and polling-delivered data if both arrive? [Completeness]
- [ ] CHK-RT012 - Is it specified what happens if the SSE stream delivers a token count that differs from the subsequent API poll for the same step? [Clarity]

## Data Integrity

- [ ] CHK-RT013 - Is it specified that the `completed` event's `tokens_used` must exactly match the value persisted to `event_log.tokens_used` in the database? [Consistency]
- [ ] CHK-RT014 - Are requirements defined for token count monotonicity within a single step (should cumulative counts only increase)? [Completeness]
- [ ] CHK-RT015 - Is it specified whether retried steps emit fresh token streams (resetting to 0) or continue from the failed attempt's cumulative count? [Completeness]
