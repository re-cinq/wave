# Tasks: Embed Web UI as Default Built-in with CLI/TUI Feature Parity

**Feature**: #299 — WebUI Built-in Parity
**Branch**: `299-webui-builtin-parity`
**Generated**: 2026-03-16

---

## Phase 1: Setup & Build Tag Removal (US1 — Default WebUI Access)

- [X] T001 [P1] [US1] Remove `//go:build webui` tag from `internal/webui/server.go`
- [X] T002 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/auth.go`
- [X] T003 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/auth_test.go`
- [X] T004 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/dag.go`
- [X] T005 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/dag_test.go`
- [X] T006 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/embed.go`
- [X] T007 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/handlers_artifacts.go`
- [X] T008 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/handlers_control.go`
- [X] T009 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/handlers_personas.go`
- [X] T010 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/handlers_pipelines.go`
- [X] T011 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/handlers_runs.go`
- [X] T012 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/handlers_sse.go`
- [X] T013 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/handlers_test.go`
- [X] T014 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/middleware.go`
- [X] T015 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/pagination.go`
- [X] T016 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/pagination_test.go`
- [X] T017 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/redact.go`
- [X] T018 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/redact_test.go`
- [X] T019 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/routes.go`
- [X] T020 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/sse.go`
- [X] T021 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/sse_broker.go`
- [X] T022 [P1] [US1] [P] Remove `//go:build webui` tag from `internal/webui/sse_test.go`
- [X] T023 [P1] [US1] Remove `//go:build webui` tag from `cmd/wave/commands/serve.go`
- [X] T024 [P1] [US1] Delete `cmd/wave/commands/serve_stub.go` (build-tag fallback no longer needed)
- [X] T025 [P1] [US1] Verify `go build ./cmd/wave` compiles without `-tags webui` and document binary size delta in `specs/299-webui-builtin-parity/binary-size.md`

## Phase 2: Foundational — Retry Handler Fix (US2 — Pipeline Execution Control)

_Depends on: Phase 1 (build tags removed so webui compiles unconditionally)_

- [X] T026 [P1] [US2] Extract shared execution logic from `handleStartPipeline` into `launchPipelineExecution(runID, pipelineName, input string) error` method on `Server` in `internal/webui/handlers_control.go`
- [X] T027 [P1] [US2] Refactor `handleStartPipeline` to call `launchPipelineExecution` instead of inline goroutine in `internal/webui/handlers_control.go`
- [X] T028 [P1] [US2] Update `handleRetryRun` to call `launchPipelineExecution` after creating DB record in `internal/webui/handlers_control.go`
- [X] T029 [P1] [US2] Add test: retry handler creates new run AND launches execution in `internal/webui/handlers_test.go`

## Phase 3: Resume-from-Step API (US4 — Resume from Failed Step)

_Depends on: Phase 2 (shared execution helper exists)_

- [X] T030 [P2] [US4] Add `ResumeRunRequest` and `ResumeRunResponse` types to `internal/webui/types.go`
- [X] T031 [P2] [US4] Add `handleResumeRun` handler in `internal/webui/handlers_control.go` — validate run state, load pipeline, create executor with `ResumeManager`, call `ResumeFromStep()`
- [X] T032 [P2] [US4] Register route `POST /api/runs/{id}/resume` in `internal/webui/routes.go`
- [X] T033 [P2] [US4] Create resume dialog partial template `internal/webui/templates/partials/resume_dialog.html` with step picker showing previous status per step
- [X] T034 [P2] [US4] Add resume dropdown UI to `internal/webui/templates/run_detail.html` for failed/cancelled runs
- [X] T035 [P2] [US4] Add tests for resume endpoint (valid step, invalid step, wrong run state) in `internal/webui/handlers_test.go`

## Phase 4: SSE Reconnection Backfill (US3 — Log Streaming)

_Depends on: Phase 1 (build tags removed). Can run in parallel with Phase 3._

- [X] T036 [P1] [US3] [P] Add `AfterID int64` field to `EventQueryOptions` in `internal/state/store.go` and update `GetEvents` query to support `WHERE id > ?` filtering
- [X] T037 [P1] [US3] Update `SSEBroker.EmitProgress` to include `ID` field (from DB row ID) in SSE event output in `internal/webui/sse_broker.go`
- [X] T038 [P1] [US3] Update `handleSSE` in `internal/webui/handlers_sse.go` to parse `Last-Event-ID` header, backfill missed events from DB, and include `id:` field in SSE output format
- [X] T039 [P1] [US3] Update `internal/webui/static/sse.js` with reconnection logic using `Last-Event-ID` and exponential backoff
- [X] T040 [P1] [US3] Add polling fallback logic to `internal/webui/static/app.js` — detect SSE failure and fall back to periodic `GET /api/runs/{id}` polling
- [X] T041 [P1] [US3] Add test for SSE backfill: verify events after `Last-Event-ID` are sent on reconnect in `internal/webui/handlers_test.go`

## Phase 5: Persona & Pipeline Configuration Viewing (US5 — Configuration Viewing)

_Depends on: Phase 1. Can run in parallel with Phases 3-4._

- [X] T042 [P2] [US5] [P] Enhance `internal/webui/templates/personas.html` to display adapter, model, description, and permission summary for each persona
- [X] T043 [P2] [US5] [P] Enhance `internal/webui/templates/pipelines.html` to display step dependencies, step count, and mini DAG preview per pipeline
- [X] T044 [P2] [US5] Verify `handleAPIPersonas` and `handleAPIPipelines` return sufficient data for enhanced templates in `internal/webui/handlers_personas.go` and `internal/webui/handlers_pipelines.go`

## Phase 6: DAG Visualization & Introspection (US6 — DAG Visualization)

_Depends on: Phase 1. Can run in parallel with Phases 3-5._

- [X] T045 [P2] [US6] Add hover tooltip containers and ARIA labels to `internal/webui/templates/partials/dag_svg.html`
- [X] T046 [P2] [US6] Add hover tooltips showing step status, duration, and token usage to `internal/webui/static/dag.js`
- [X] T047 [P2] [US6] Add click-to-inspect linking DAG nodes to step detail section in `internal/webui/static/dag.js`
- [X] T048 [P2] [US6] Add CSS `overflow: auto` and scroll handling for large DAGs (50+ steps) in `internal/webui/static/style.css`
- [X] T049 [P2] [US6] Add contract validation results display to step detail view in `internal/webui/templates/partials/step_card.html`

## Phase 7: Responsive Layout & Accessibility (US7 — Responsive & A11y)

_Depends on: Phase 1. Can run in parallel with Phases 3-6._

- [X] T050 [P3] [US7] [P] Add viewport meta tag and ARIA landmark roles to `internal/webui/templates/layout.html`
- [X] T051 [P3] [US7] [P] Add CSS media queries for 768px, 1024px, and 1920px+ breakpoints to `internal/webui/static/style.css`
- [X] T052 [P3] [US7] Add visible focus indicators (`:focus-visible`) for all interactive elements in `internal/webui/static/style.css`
- [X] T053 [P3] [US7] Add keyboard event handlers (Tab, Enter, Escape) for interactive elements in `internal/webui/static/app.js`
- [X] T054 [P3] [US7] Add ARIA labels to all interactive elements across templates: `internal/webui/templates/runs.html`, `run_detail.html`, `personas.html`, `pipelines.html`

## Phase 8: Security & Error Handling (Cross-cutting)

_Depends on: Phases 2-4 (new handlers must exist for security verification)._

- [X] T055 [P1] Verify credential redaction in `internal/webui/redact.go` covers AWS keys, API tokens, GitHub PATs, and bearer tokens — add missing patterns if any
- [X] T056 [P1] Verify auth enforcement for non-localhost bindings works correctly in `internal/webui/auth.go` — add integration test
- [X] T057 [P1] Verify security headers (CSP, X-Frame-Options, X-Content-Type-Options) are set on all responses in `internal/webui/middleware.go`
- [X] T058 [P1] [US3] Enhance artifact viewer with truncation notice for >100KB artifacts and credential redaction indicator in `internal/webui/templates/partials/artifact_viewer.html`
- [X] T059 [P1] [US2] Add structured error messages with recovery hints for pipeline step failures in `internal/webui/templates/run_detail.html`

## Phase 9: Testing & Validation (Final)

_Depends on: All previous phases._

- [X] T060 Ensure `go test ./internal/webui/...` passes with zero skipped tests (SC-006)
- [X] T061 Run `go test -race ./...` and fix any data race issues
- [X] T062 Measure binary size with and without webui assets, document delta in PR description (FR-017, SC-002)
- [X] T063 Manual verification: `go build ./cmd/wave && ./wave serve` starts dashboard without build tags (SC-001)

---

## Summary

| Phase | Story | Tasks | Parallel |
|-------|-------|-------|----------|
| 1 — Build Tag Removal | US1 | 25 | 21 |
| 2 — Retry Handler Fix | US2 | 4 | 0 |
| 3 — Resume API | US4 | 6 | 0 |
| 4 — SSE Backfill | US3 | 6 | 1 |
| 5 — Config Viewing | US5 | 3 | 2 |
| 6 — DAG Visualization | US6 | 5 | 0 |
| 7 — Responsive & A11y | US7 | 5 | 2 |
| 8 — Security | Cross | 5 | 0 |
| 9 — Validation | Cross | 4 | 0 |
| **Total** | | **63** | **26** |
