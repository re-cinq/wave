# API & Architecture Quality Checklist: Web-Based Pipeline Operations Dashboard

**Feature**: 085-web-operations-dashboard
**Date**: 2026-02-13
**Focus**: API design completeness, SSE architecture, and build isolation

---

## API Design

- [ ] CHK-A01 - Are all API endpoints documented with request/response schemas, including query parameters for filtering and pagination? [Completeness]
- [ ] CHK-A02 - Is the API versioning strategy defined, or is it explicitly stated that no versioning is needed during prototype phase? [Completeness]
- [ ] CHK-A03 - Are content negotiation rules defined — does the server return JSON vs. HTML based on Accept header or URL pattern (/api/ prefix)? [Clarity]
- [ ] CHK-A04 - Is the response format for execution control endpoints (start/cancel/retry) defined — what does the server return after a successful POST? [Completeness]
- [ ] CHK-A05 - Are query parameter names and formats for run list filtering (status, pipeline name, time range) explicitly defined? [Completeness]

## SSE Architecture

- [ ] CHK-A06 - Is the SSE event format fully specified — event type naming, data payload schema, and `id` field for resumption? [Completeness]
- [ ] CHK-A07 - Is it specified whether SSE supports `Last-Event-ID` for event replay after reconnection, or only live events from reconnection time? [Completeness]
- [ ] CHK-A08 - Is the behavior defined when an SSE client connects for a run that has already completed — does it receive a terminal event and close, or receive nothing? [Completeness]
- [ ] CHK-A09 - Is the SSE broker's behavior defined under memory pressure — what happens when a slow client's channel buffer fills up? [Completeness]
- [ ] CHK-A10 - Is the relationship between the SSE broker and the existing `event.EventEmitter` interface clearly specified — does the broker register globally or per-run? [Clarity]

## Build Tag Isolation

- [ ] CHK-A11 - Is the import graph verified to ensure no non-tagged package imports `internal/webui/` — preventing accidental compilation of webui code? [Coverage]
- [ ] CHK-A12 - Is it defined how `go:embed` directives interact with the build tag — are static/ and templates/ directories required to exist even without the tag? [Completeness]
- [ ] CHK-A13 - Is the CI/release pipeline updated to produce both tagged and untagged binaries, or is the build tag selection documented for operators? [Completeness]

## Database & Concurrency

- [ ] CHK-A14 - Is the connection pool sizing for the read-only store justified — why 10 connections, and what happens if more than 10 concurrent HTTP handlers need DB access? [Clarity]
- [ ] CHK-A15 - Are transaction isolation requirements specified for read-only queries — e.g., should a run detail query see a consistent snapshot across runs, steps, and artifacts? [Completeness]
- [ ] CHK-A16 - Is the behavior defined when the read-write store (for execution control) fails to open because the pipeline executor holds a lock? [Completeness]

---

**Total Items**: 16
**Dimensions**: Completeness (11), Clarity (3), Coverage (2)
