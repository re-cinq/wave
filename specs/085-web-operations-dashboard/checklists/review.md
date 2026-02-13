# Quality Review Checklist: Web-Based Pipeline Operations Dashboard

**Feature**: 085-web-operations-dashboard
**Date**: 2026-02-13
**Artifacts Reviewed**: spec.md, plan.md, tasks.md, research.md, data-model.md, contracts/

---

## Completeness

- [ ] CHK001 - Are all seven user stories (US1-US7) fully traceable to functional requirements, with no orphaned stories or requirements? [Completeness]
- [ ] CHK002 - Does every acceptance scenario specify a concrete, verifiable outcome (not just "system updates correctly")? [Completeness]
- [ ] CHK003 - Are error/failure responses defined for all API endpoints (e.g., 404 for missing run, 400 for invalid cursor, 409 for already-cancelled run)? [Completeness]
- [ ] CHK004 - Is the behavior defined for when `wave serve` is started while another instance is already bound to the same port? [Completeness]
- [ ] CHK005 - Are rate limiting or connection limits specified for the SSE endpoint to prevent resource exhaustion from many browser tabs? [Completeness]
- [ ] CHK006 - Is the behavior defined for artifact browsing when a step produced no artifacts? [Completeness]
- [ ] CHK007 - Are the specific HTTP status codes documented for each API endpoint in success and error cases? [Completeness]
- [ ] CHK008 - Is the dashboard's behavior defined when the manifest changes while the server is running (e.g., new persona added)? [Completeness]
- [ ] CHK009 - Are all edge cases from the spec reflected in corresponding acceptance scenarios or explicit requirements? [Completeness]
- [ ] CHK010 - Is the expected URL structure for pagination documented (query parameters, cursor format in response)? [Completeness]

## Clarity

- [ ] CHK011 - Is "real-time" consistently defined with a measurable latency bound across all references (US2 scenarios vs. SC-003 vs. NFR-004)? [Clarity]
- [ ] CHK012 - Is the distinction between "stop" (US4) and "cancel" (FR-008, cancellation table) terminology consistent throughout the spec? [Clarity]
- [ ] CHK013 - Is the meaning of "input" for pipeline start (US4, FR-007) clearly defined — is it free-form text, structured YAML, or tied to InputConfig schema? [Clarity]
- [ ] CHK014 - Is the scope of "responsive" (NFR-003) defined with specific breakpoints or minimum screen sizes? [Clarity]
- [ ] CHK015 - Does FR-013 unambiguously specify which assets are "frontend assets" vs. which are templates — e.g., does the 50 KB limit apply to CSS as well as JS? [Clarity]
- [ ] CHK016 - Is it clear whether the SSE stream carries events for a single run (`/api/runs/{id}/events`) or all runs (global stream)? Is a global stream needed for the run list page? [Clarity]
- [ ] CHK017 - Are the authentication bypass rules for localhost clearly specified — does `127.0.0.1` include `::1` (IPv6 loopback)? [Clarity]

## Consistency

- [ ] CHK018 - Are the latency targets consistent across artifacts: SC-003 says "within 2 seconds", US2 says "within 1 second" for step updates — which is authoritative? [Consistency]
- [ ] CHK019 - Does the task breakdown (tasks.md) cover every functional requirement (FR-001 through FR-018) with at least one task? [Consistency]
- [ ] CHK020 - Are the API routes in research.md R-007 consistent with the handler files and route definitions in plan.md? [Consistency]
- [ ] CHK021 - Are the SSE event types listed in data-model.md consistent with the event states defined in the existing `event.Event` struct? [Consistency]
- [ ] CHK022 - Is the `RunSummary` type in data-model.md consistent with the contract schema in `api-runs-list.json`? [Consistency]
- [ ] CHK023 - Does the task dependency graph in tasks.md match the actual file-level dependencies described in plan.md? [Consistency]
- [ ] CHK024 - Is the default page size (25 per FR-017/C-005) reflected consistently in the API contracts, data model, and handler descriptions? [Consistency]

## Coverage

- [ ] CHK025 - Are security requirements (SR-001 through SR-005) each covered by at least one task and one test in the task breakdown? [Coverage]
- [ ] CHK026 - Are non-functional requirements (NFR-001 through NFR-005) each associated with a measurable verification step in the tasks? [Coverage]
- [ ] CHK027 - Is there a task or acceptance criterion that verifies the `webui` build tag exclusion doesn't break `go test ./...` (no import of webui package from non-tagged code)? [Coverage]
- [ ] CHK028 - Are concurrent access scenarios covered — multiple browser clients reading while a pipeline writes to the state DB? [Coverage]
- [ ] CHK029 - Is graceful shutdown (FR-015) covered with specific scenarios: in-flight SSE streams, in-flight API responses, active pipeline execution triggered from dashboard? [Coverage]
- [ ] CHK030 - Is the credential redaction requirement (FR-016/SR-005) tested against the specific patterns listed in R-008 (AWS keys, OpenAI keys, GitHub PATs, etc.)? [Coverage]
- [ ] CHK031 - Is path traversal prevention (SR-003) covered with test scenarios for `../`, symlinks, absolute paths, and URL-encoded traversal attempts? [Coverage]
- [ ] CHK032 - Are the success criteria (SC-001 through SC-010) each verifiable through the defined acceptance scenarios and tasks? [Coverage]
- [ ] CHK033 - Is there coverage for the "no manifest" edge case — server starting with only a state DB and degraded persona/pipeline display? [Coverage]

---

**Total Items**: 33
**Dimensions**: Completeness (10), Clarity (7), Consistency (7), Coverage (9)
