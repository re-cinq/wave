# Requirements Quality Review Checklist

**Feature**: #299 — Embed Web UI as Default Built-in with CLI/TUI Feature Parity
**Generated**: 2026-03-16

---

## Completeness

- [ ] CHK001 - Are all CLI commands that have webui equivalents explicitly enumerated? (FR-003 says "listing, starting, cancelling, and retrying" but SC-003 also includes "resume-from-step" — is the FR list exhaustive?) [Completeness]
- [ ] CHK002 - Is the artifact type rendering requirement (FR-006) specific about which formats are supported beyond JSON, Markdown, and plain text? (e.g., YAML, binary, images) [Completeness]
- [ ] CHK003 - Does FR-010 (bearer token auth) specify how the token is provisioned, rotated, or revoked? Or is this deferred to existing implementation? [Completeness]
- [ ] CHK004 - Are error states fully specified for the resume-from-step API (C2)? The resolution mentions "phase validation" and "stale artifact detection" — are error responses for these cases defined? [Completeness]
- [ ] CHK005 - Is the behavior specified when `wave serve` is run but no pipelines are configured in the manifest? (Empty state handling) [Completeness]
- [ ] CHK006 - Does the spec define what happens when the SSE polling fallback (FR-005) activates? Is there a polling interval, backoff strategy, or user notification? [Completeness]
- [ ] CHK007 - Are token usage display requirements (US2-AS4, US6-AS2) defined with specific fields (input tokens, output tokens, cost)? [Completeness]
- [ ] CHK008 - Is the credential redaction scope (FR-009) defined with a pattern list or regex set, or only by example categories (AWS keys, tokens, PATs)? [Completeness]

## Clarity

- [ ] CHK009 - Is the distinction between "retry" (US2-AS3: new run from failed config) and "resume" (US4: continue from specific step) unambiguous in the UI requirements? [Clarity]
- [ ] CHK010 - Does SC-005 ("within 1 second") specify measurement methodology — is this end-to-end browser render or server emit-to-SSE-send latency? [Clarity]
- [ ] CHK011 - Is "sufficient data" for persona/pipeline display (T044) defined with specific fields, or left to implementation judgment? [Clarity]
- [ ] CHK012 - Does "responsive layout" (FR-012) define specific breakpoint behaviors (what collapses, what stacks, what hides) or only the viewport range? [Clarity]
- [ ] CHK013 - Is "structured error messages with recovery hints" (FR-016) defined with specific error categories, hint templates, or format requirements? [Clarity]
- [ ] CHK014 - Does the "mini DAG preview" for pipelines page (T043) have defined dimensions, interactivity level, or is it purely decorative? [Clarity]

## Consistency

- [ ] CHK015 - Does the SSE event ID scheme (C3: database row ID) align with the state store's actual auto-increment behavior across all event types? [Consistency]
- [ ] CHK016 - Is FR-018 ("same event system as TUI") consistent with the SSE Last-Event-ID backfill approach, which requires DB-stored event IDs that the TUI event system may not use? [Consistency]
- [ ] CHK017 - Does the resume API response (C2: "mirrors StartPipelineResponse") account for the fact that resumed runs may have different step counts than fresh starts? [Consistency]
- [ ] CHK018 - Are the edge case for "50+ steps" (DAG scrolling) and SC-010 ("up to 20 parallel branches") consistent in their scale expectations? [Consistency]
- [ ] CHK019 - Is the security header CSP policy (FR-011) compatible with inline vanilla JS event handlers on SVG elements (C4 resolution)? [Consistency]
- [ ] CHK020 - Does the "no JavaScript build step" constraint (plan Technical Context) conflict with any of the frontend enhancement requirements (app.js, sse.js, dag.js modifications)? [Consistency]

## Coverage

- [ ] CHK021 - Are all 7 user stories traced to at least one functional requirement and one success criterion? [Coverage]
- [ ] CHK022 - Does every functional requirement (FR-001 through FR-018) have at least one task in tasks.md? [Coverage]
- [ ] CHK023 - Are all 5 clarification resolutions (C1-C5) reflected as tasks in the implementation plan? [Coverage]
- [ ] CHK024 - Does the test coverage requirement (SC-006) specify which "core routes" must be covered — is this every API endpoint or a defined subset? [Coverage]
- [ ] CHK025 - Are all 6 edge cases (SSE drop, concurrent users, reverse proxy, DB lock, 50+ steps, cross-arch) covered by either acceptance scenarios or tasks? [Coverage]
- [ ] CHK026 - Is there a requirement or task for the reverse proxy path prefix edge case ("served behind a reverse proxy with path prefix")? [Coverage]
- [ ] CHK027 - Is there a requirement or task for the SQLite DB lock edge case ("clear error message, not a hang or blank page")? [Coverage]
- [ ] CHK028 - Are accessibility requirements (US7) covered with specific WCAG level targets (A, AA, AAA) or left undefined? [Coverage]
