# Phase 2.4: Run-on-this-issue dispatch wiring

**Issue:** [re-cinq/wave#1595](https://github.com/re-cinq/wave/issues/1595)
**Labels:** `enhancement`, `ready-for-impl`
**State:** OPEN
**Author:** nextlevelshit
**Epic:** #1565 Phase 2 (work-source dispatch)

## Body

Part of Epic #1565 Phase 2 (work-source dispatch).

### Goal

Wire the "Run on this issue" button (from #2.3 work board detail) so it: looks up matching bindings via `worksource.Service.MatchBindings`, picks the appropriate pipeline, and invokes `runner.LaunchInProcess` (or detached) — closing the loop from work item → real run.

### Acceptance criteria

- [ ] `internal/webui/handlers_work_dispatch.go` (or extend handlers_work.go):
  - `POST /work/{forge}/{owner}/{repo}/{number}/dispatch` — accepts pipeline name, validates against MatchBindings result, launches the run
  - Returns redirect to the run detail page on success
  - Returns 409 if no binding matches; 400 if multiple match and pipeline param missing
- [ ] Goes through the existing service layer (no bypass of `runner.BuildExecutorOptions`)
- [ ] Auto-injects work_item_ref as the run input (so the persona has the issue context)
- [ ] Test coverage: round-trip test that fakes a binding + WorkItemRef and verifies a run is created in state store

### Note on pipeline choice

Originally tagged `impl-finding` in epic plan. Use `impl-issue` instead — `impl-finding` requires parent-pipeline-injected pr-context which doesn't fit fresh issue dispatch (lesson from PR #1588 dispatch).

### Dependencies

- #2.3 (work board) — UI surface
- #1591 WorkSourceService.MatchBindings (MERGED)
- PRE-1 service layer (MERGED) — uses runner.LaunchInProcess

## Acceptance Criteria (extracted)

1. New handler at `POST /work/{forge}/{owner}/{repo}/{number}/dispatch`.
2. Accepts optional `pipeline` parameter (form or query) for disambiguation.
3. 409 Conflict when zero bindings match the work item.
4. 400 Bad Request when multiple bindings match and `pipeline` param is missing.
5. Successful dispatch returns 302 redirect to `/runs/{runID}`.
6. Run input is the JSON-serialized `work_item_ref` (shared schema #1590).
7. Launch goes through `runner.LaunchInProcess` (via `s.launchPipelineExecution`) — no executor option bypass.
8. Round-trip test creates a binding, fakes a `WorkItemRef`, hits the endpoint, asserts a run row exists in the state store with the expected pipeline + input.

## Metadata

- Branch: `1595-dispatch-wiring`
- Complexity: medium
- Skipped speckit steps: specify, clarify, checklist, analyze
