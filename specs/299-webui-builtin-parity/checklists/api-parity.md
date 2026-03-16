# API & Feature Parity Checklist

**Feature**: #299 — Embed Web UI as Default Built-in with CLI/TUI Feature Parity
**Generated**: 2026-03-16

This checklist validates that CLI/TUI parity requirements are fully specified.

---

## CLI Command Parity

- [ ] CHK101 - Is the mapping from every `wave` CLI subcommand to its webui equivalent explicitly documented? (SC-003 claims parity but no mapping table exists) [Completeness]
- [ ] CHK102 - Are `wave run` flags (`--from-step`, `--force`, `--debug`, `--verbose`) all represented in the webui start/resume forms? [Completeness]
- [ ] CHK103 - Does the spec address `wave config show` parity (US5-AS3 mentions it) with specific fields and format? [Clarity]
- [ ] CHK104 - Is `wave status` parity defined — does the webui runs list provide equivalent information (active runs, step progress)? [Completeness]

## API Endpoint Specification

- [ ] CHK105 - Are all API endpoints explicitly listed with HTTP method, path, request schema, and response schema? (Only resume endpoint in C2 has a defined schema) [Completeness]
- [ ] CHK106 - Is error response format standardized across all API endpoints (status codes, error body structure)? [Consistency]
- [ ] CHK107 - Are API versioning or backwards-compatibility requirements stated? [Completeness]
- [ ] CHK108 - Does the cancel endpoint (US2-AS2) specify whether cancellation is synchronous (waits for step to stop) or asynchronous (returns immediately)? [Clarity]

## Real-Time Event Specification

- [ ] CHK109 - Are all SSE event types enumerated with their payload schemas? [Completeness]
- [ ] CHK110 - Is the SSE endpoint path specified (is it `/api/runs/{id}/events`, `/api/sse`, or something else)? [Clarity]
- [ ] CHK111 - Does the reconnection backfill (C3) specify a maximum backfill window or event count limit to prevent unbounded queries? [Completeness]
- [ ] CHK112 - Is the relationship between SSE events and the existing `internal/event/` types explicitly defined? [Consistency]

## Security Boundary Specification

- [ ] CHK113 - Is "non-localhost binding" (FR-010) precisely defined — does `0.0.0.0`, `127.0.0.1`, `::1`, or hostname resolution trigger auth? [Clarity]
- [ ] CHK114 - Are CORS requirements specified for the API endpoints? [Completeness]
- [ ] CHK115 - Does the CSP header (FR-011) account for SSE connections and any inline scripts needed for vanilla JS? [Consistency]
