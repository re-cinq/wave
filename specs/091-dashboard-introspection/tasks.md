# Tasks: Dashboard Inspection, Rendering, Statistics & Run Introspection

**Branch**: `091-dashboard-introspection` | **Generated**: 2026-02-14
**Source**: spec.md, plan.md, data-model.md, research.md, contracts/

---

## Phase 1: Setup & Shared Infrastructure

These tasks establish the foundational types, interfaces, and shared components that all subsequent phases depend on.

- [X] T001 [P1] Add new API response types to `internal/webui/types.go` — PipelineDetailResponse, PipelineInputDetail, InputSchemaDetail, PipelineStepDetail, WorkspaceDetail, MountDetail, ContractDetail, ArtifactDefDetail, MemoryDetail, InjectedArtifact, PersonaDetailResponse, HooksDetail, HookRuleDetail, SandboxDetail per data-model.md
- [X] T002 [P1] Add statistics response types to `internal/webui/types.go` — RunStatistics, RunTrendPoint, PipelineStatistics, StatisticsResponse per data-model.md and contracts/api-statistics.json
- [X] T003 [P1] Add enhanced run detail types to `internal/webui/types.go` — EnhancedStepDetail (embedding StepDetail), ContractResultDetail, RecoveryHintDetail, StepPerfDetail per data-model.md and contracts/api-enhanced-run-detail.json
- [X] T004 [P1] Add workspace browsing types to `internal/webui/types.go` — WorkspaceTreeResponse, WorkspaceEntry, WorkspaceFileResponse per data-model.md and contracts/api-workspace.json
- [X] T005 [P1] Add new state record types to `internal/state/types.go` — RunStatisticsRecord, RunTrendRecord, PipelineStatisticsRecord per data-model.md

---

## Phase 2: Foundational — State Store & Shared Backend

These tasks add the backend query methods and shared utilities that handlers will depend on. Must complete before handler phases.

- [X] T006 [P1] Add `GetRunStatistics(since time.Time) (*RunStatisticsRecord, error)` to StateStore interface in `internal/state/store.go` — SQL aggregation query using `GROUP BY` on pipeline_run.status with time range filter on started_at. Returns total, succeeded, failed, cancelled, pending, running counts.
- [X] T007 [P1] Add `GetRunTrends(since time.Time) ([]RunTrendRecord, error)` to StateStore interface in `internal/state/store.go` — SQL query grouping pipeline_run by `strftime('%Y-%m-%d', started_at, 'unixepoch')` with status counts per day.
- [X] T008 [P1] Add `GetPipelineStatistics(since time.Time) ([]PipelineStatisticsRecord, error)` to StateStore interface in `internal/state/store.go` — SQL aggregation query grouping pipeline_run by pipeline_name with run count, success rate, avg duration, avg tokens.
- [X] T009 [P1] Add `GetPipelineStepStats(pipelineName string, since time.Time) ([]StepPerformanceStats, error)` to StateStore interface in `internal/state/store.go` — SQL aggregation on performance_metric table grouped by step_id for a given pipeline name. Reuse existing StepPerformanceStats type.
- [X] T010 [P1] Add `GetLastRunForPipeline(pipelineName string) (*RunRecord, error)` to StateStore interface in `internal/state/store.go` — Query pipeline_run for most recent run by pipeline_name ordered by started_at DESC LIMIT 1.
- [X] T011 [P1] Implement all 5 new StateStore methods (T006-T010) in `internal/state/store.go` on the `stateStore` struct.
- [X] T012 [P1] Implement read-only stubs for new StateStore methods in `internal/state/readonly.go` to satisfy the StateStore interface for the read-only wrapper.
- [X] T013 [P1] Write unit tests for new StateStore methods in `internal/state/store_test.go` — test GetRunStatistics, GetRunTrends, GetPipelineStatistics, GetPipelineStepStats, GetLastRunForPipeline with seeded test data including edge cases (zero runs, mixed statuses, time range filtering).

---

## Phase 3: US1 — Pipeline, Persona & Contract Inspection (P1)

Delivers FR-001 through FR-004: pipeline detail view, persona detail view, contract display, cross-entity navigation.

- [X] T014 [P1] [US1] Create pipeline detail HTML handler `handlePipelineDetailPage` in `internal/webui/handlers_pipelines.go` — loads pipeline YAML via `loadPipelineYAML`, assembles PipelineDetailResponse with all step configs, persona cross-refs from `s.manifest.Personas`, contract definitions (inline schema or resolved schema_path), input config, and DAG layout. Template: `templates/pipeline_detail.html`.
- [X] T015 [P1] [US1] Create pipeline detail API handler `handleAPIPipelineDetail` in `internal/webui/handlers_pipelines.go` — JSON endpoint returning PipelineDetailResponse matching contracts/api-pipeline-detail.json. Include last_run via GetLastRunForPipeline.
- [X] T016 [P1] [US1] Create `templates/pipeline_detail.html` — pipeline inspection view showing: metadata (name, description) header, step list with persona, dependencies, workspace config, contract definitions, input schema with examples, DAG visualization reusing existing `partials/dag_svg.html`. Persona names link to `/personas/{name}`.
- [X] T017 [P1] [US1] Create persona detail HTML handler `handlePersonaDetailPage` in `internal/webui/handlers_personas.go` — reads persona from `s.manifest.Personas[name]`, reads system prompt file content (with graceful missing-file handling per edge case), computes `used_in_pipelines` by scanning all pipeline YAMLs. Template: `templates/persona_detail.html`.
- [X] T018 [P1] [US1] Create persona detail API handler `handleAPIPersonaDetail` in `internal/webui/handlers_personas.go` — JSON endpoint returning PersonaDetailResponse matching contracts/api-persona-detail.json. System prompt content HTML-escaped per FR-031.
- [X] T019 [P1] [US1] Create `templates/persona_detail.html` — persona inspection view showing: name, description, adapter, model, temperature, system prompt content (in a scrollable pre block), allowed/denied tools lists, hooks configuration, sandbox config, and "used in pipelines" list with links to `/pipelines/{name}`.
- [X] T020 [P] [US1] Register new routes in `internal/webui/routes.go` — add `GET /pipelines/{name}`, `GET /api/pipelines/{name}`, `GET /personas/{name}`, `GET /api/personas/{name}`.
- [X] T021 [P] [US1] Register new page templates in `internal/webui/embed.go` — add `templates/pipeline_detail.html` and `templates/persona_detail.html` to `pageTemplates` slice.
- [X] T022 [P1] [US1] Update `templates/pipelines.html` — each pipeline name becomes a link to `/pipelines/{name}` detail view. Show description alongside name.
- [X] T023 [P1] [US1] Update `templates/personas.html` — each persona name becomes a link to `/personas/{name}` detail view.
- [X] T024 [P1] [US1] Update `templates/layout.html` — add nav links for Statistics page (`/statistics`). Ensure existing nav links for Pipelines, Personas remain.
- [X] T025 [P1] [US1] Write unit tests for pipeline detail and persona detail handlers in `internal/webui/handlers_test.go` — test API JSON response shape matches contract schemas, test missing pipeline 404, test persona with missing prompt file graceful degradation.

---

## Phase 4: US2 — Run Statistics Dashboard (P1)

Delivers FR-010 through FR-014: aggregate stats, trends, per-pipeline breakdown, per-step stats, time range filtering.

- [X] T026 [P1] [US2] Create `internal/webui/handlers_statistics.go` — implement `handleStatisticsPage` (HTML) and `handleAPIStatistics` (JSON). Parse `?range=` query param (24h, 7d, 30d, all — default 7d). Compute `since` time.Time from range. Call GetRunStatistics, GetRunTrends, GetPipelineStatistics. Assemble StatisticsResponse. Calculate success_rate as percentage.
- [X] T027 [P1] [US2] Create `templates/statistics.html` — statistics dashboard with: aggregate count cards (total, succeeded, failed, cancelled, success rate %), time range selector dropdown, per-pipeline breakdown table with inline CSS bar charts, daily trend table with bar sparklines. CSS-only visualizations per R-009.
- [X] T028 [P1] [US2] Create `internal/webui/static/stats.js` — time range filter interaction: on `<select>` change, reload page with `?range=` query param. No fetch needed — full page reload for simplicity.
- [X] T029 [P1] [US2] Create `templates/partials/stats_chart.html` — reusable partial for CSS-based horizontal bar charts. Accept data via template parameters (value, max, label, color class).
- [X] T030 [P] [US2] Register statistics routes in `internal/webui/routes.go` — add `GET /statistics`, `GET /api/statistics`.
- [X] T031 [P] [US2] Register statistics template in `internal/webui/embed.go` — add `templates/statistics.html` to `pageTemplates` slice.
- [X] T032 [P1] [US2] Add statistics page empty state handling — when GetRunStatistics returns zero total, display "No pipeline runs recorded yet" message instead of empty charts per edge case.
- [X] T033 [P1] [US2] Write unit tests for statistics handlers in `internal/webui/handlers_test.go` — test API response shape matches contracts/api-statistics.json, test time range parsing, test empty state, test success rate calculation.

---

## Phase 5: US3 — Run Introspection (P1)

Delivers FR-015 through FR-019: event timeline, step drill-down, contract validation results, failure details, recovery hints, artifact display.

- [X] T034 [P1] [US3] Enhance `handleAPIRunDetail` in `internal/webui/handlers_runs.go` — extend the existing handler to populate EnhancedStepDetail fields: contract_result (from pipeline YAML contract config + event error messages), recovery_hints (via recovery.ClassifyError on failed step error messages), performance (from GetPerformanceMetrics), workspace_path (from step_state table), workspace_exists (os.Stat check on workspace path).
- [X] T035 [P1] [US3] Create recovery hint generation helper in `internal/webui/handlers_runs.go` — function `buildRecoveryHints(runID, stepID, pipelineName, errMsg string) []RecoveryHintDetail` that uses the recovery package's error classification to generate hints at display time per R-004.
- [X] T036 [P1] [US3] Enhance `handleRunDetailPage` in `internal/webui/handlers_runs.go` — pass enhanced step details (with contract results, recovery hints, performance data) and full event list to the template.
- [X] T037 [P1] [US3] Update `templates/run_detail.html` — add event timeline section (chronological events with timestamps, state badges, persona, message, token deltas), step drill-down panels (click step to expand contract results, artifacts, performance, recovery hints), failure prominence (error banner for failed steps with recovery hints).
- [X] T038 [P1] [US3] Create `templates/partials/step_inspector.html` — reusable partial for step drill-down panel showing: contract validation result (pass/fail badge, schema, error), artifacts list (name, type, size, preview link), performance metrics (duration, tokens, files modified), recovery hints (for failed steps), workspace browsing link.
- [X] T039 [P1] [US3] Create `internal/webui/static/introspect.js` — step drill-down toggle (click step row to expand/collapse inspector panel), event timeline scroll-to-step (click step in timeline highlights step inspector).
- [X] T040 [P1] [US3] Write unit tests for enhanced run detail handler in `internal/webui/handlers_test.go` — test that enhanced fields are populated for completed/failed runs, test recovery hint generation from error messages, test workspace_exists detection.

---

## Phase 6: US4 — Markdown Rendering (P2)

Delivers FR-005, FR-006, FR-009: markdown rendering with raw/rendered toggle.

- [X] T041 [P] [US4] Create `internal/webui/static/markdown.js` — minimal client-side markdown parser (~200 lines, ~5 KB). Support: headings (h1-h4), unordered/ordered lists, fenced code blocks, inline code, bold/italic emphasis, links, tables. All output text-escaped (no raw HTML passthrough) per FR-031. Export `renderMarkdown(text)` function.
- [X] T042 [P] [US4] Create `templates/partials/markdown_viewer.html` — partial template with raw/rendered toggle. Structure: `<div class="md-viewer">` with `<div class="md-rendered">` and `<div class="md-raw"><pre>` plus toggle buttons. JS init calls `renderMarkdown()` on page load for rendered view.
- [X] T043 [P2] [US4] Integrate markdown viewer into `templates/persona_detail.html` — replace the plain `<pre>` system prompt display with the markdown_viewer partial, enabling raw/rendered toggle for system prompt content.
- [X] T044 [P2] [US4] Integrate markdown viewer into step inspector for `.md` artifact previews — in `templates/partials/step_inspector.html`, detect `.md` artifact type and render preview using markdown_viewer partial.
- [X] T045 [P2] [US4] Add CSS styles for markdown rendering to `internal/webui/static/style.css` — styles for `.md-viewer`, `.md-rendered`, `.md-raw`, toggle buttons, rendered markdown elements (headings, lists, code blocks, tables, links).
- [X] T046 [P2] [US4] Write tests for markdown.js — create `internal/webui/static/markdown_test.html` (manual test page) or verify through handler test that markdown partial renders without JS errors.

---

## Phase 7: US5 — YAML & Schema Rendering (P2)

Delivers FR-007, FR-008: syntax highlighting for YAML, JSON, and source code with raw/formatted toggle.

- [X] T047 [P] [US5] Create `internal/webui/static/highlight.js` — regex-based syntax highlighter (~150 lines, ~4 KB). Language tokenizers for: YAML, JSON, Go, SQL, Shell, JavaScript, CSS, HTML, Markdown. Assign CSS classes (`tok-key`, `tok-str`, `tok-num`, `tok-comment`, `tok-bool`, `tok-kw`). All content HTML-escaped before tokenization per FR-031. Export `highlight(code, language)` function.
- [X] T048 [P] [US5] Create `templates/partials/code_viewer.html` — partial template with raw/formatted toggle. Structure: `<div class="code-viewer">` with `<div class="code-highlighted"><pre><code>` and `<div class="code-raw"><pre>` plus toggle buttons. JS init calls `highlight()` on page load.
- [X] T049 [P2] [US5] Integrate syntax highlighting into `templates/pipeline_detail.html` — render pipeline YAML configuration and JSON schema contract definitions using code_viewer partial with appropriate language parameter.
- [X] T050 [P2] [US5] Integrate syntax highlighting into `templates/persona_detail.html` — render hooks config and sandbox config sections using code_viewer partial with YAML highlighting.
- [X] T051 [P2] [US5] Add CSS styles for syntax highlighting to `internal/webui/static/style.css` — styles for `.code-viewer`, `.code-highlighted`, `.code-raw`, toggle buttons, and token classes (`tok-key`, `tok-str`, `tok-num`, `tok-comment`, `tok-bool`, `tok-kw`). Support both light and dark themes.
- [X] T052 [P2] [US5] Integrate syntax highlighting into step inspector — in `templates/partials/step_inspector.html`, use code_viewer partial for contract schema display and artifact previews with detected language.

---

## Phase 8: US6 — Meta Information Display (P2)

Delivers FR-020 through FR-023: pipeline/persona metadata, last run status, input examples.

- [X] T053 [P2] [US6] Enhance pipeline list page `templates/pipelines.html` — display description alongside name, show step count badge, show last run status indicator (requires calling GetLastRunForPipeline per pipeline). Update `handlePipelinesPage` to include last run data.
- [X] T054 [P2] [US6] Enhance persona list page `templates/personas.html` — display description, adapter, model alongside each persona name per FR-021.
- [X] T055 [P2] [US6] Ensure pipeline detail header shows metadata prominently — verify `templates/pipeline_detail.html` displays name, description at top per FR-020, last run status/time per FR-022, and input examples per FR-023.
- [X] T056 [P2] [US6] Update `handlePipelinesPage` in `internal/webui/handlers_pipelines.go` — extend PipelineSummary or create enriched template data struct to include last run status from GetLastRunForPipeline for each pipeline.

---

## Phase 9: US7 — Workspace & Source Browsing (P3)

Delivers FR-024 through FR-028: file tree browser, syntax-highlighted file viewer, lazy loading, read-only, missing workspace handling.

- [X] T057 [P3] [US7] Create `internal/webui/handlers_workspace.go` — implement `handleWorkspaceTree` (GET /api/runs/{id}/workspace/{step}/tree?path=) and `handleWorkspaceFile` (GET /api/runs/{id}/workspace/{step}/file?path=). Resolve workspace path from step_state table. Validate path traversal via `filepath.Rel` check against workspace root. Return WorkspaceTreeResponse/WorkspaceFileResponse per contracts/api-workspace.json. Max 500 entries per directory, 1MB file size limit, no symlink following, content HTML-escaped.
- [X] T058 [P3] [US7] Create `internal/webui/static/workspace.js` — file tree browser with lazy-loading. On directory click, fetch `/api/runs/{id}/workspace/{step}/tree?path=` and expand subtree. On file click, fetch `/api/runs/{id}/workspace/{step}/file?path=` and display in content pane with syntax highlighting via `highlight()`. Tree uses `<ul>/<li>` with expand/collapse icons.
- [X] T059 [P3] [US7] Create `templates/partials/workspace_tree.html` — workspace browsing partial with two-pane layout: file tree on left, file content viewer on right. Shows "Workspace unavailable" message when workspace_exists is false per FR-028 edge case.
- [X] T060 [P3] [US7] Integrate workspace browser into `templates/run_detail.html` — add "Workspace" tab/section per step that includes the workspace_tree partial. Only show when workspace_path is set.
- [X] T061 [P] [US7] Register workspace API routes in `internal/webui/routes.go` — add `GET /api/runs/{id}/workspace/{step}/tree`, `GET /api/runs/{id}/workspace/{step}/file`.
- [X] T062 [P3] [US7] Add path traversal security tests for workspace handler in `internal/webui/handlers_test.go` — test that `../` paths are rejected, symlinks are not followed, files >1MB are truncated with indicator, missing workspaces return proper error JSON.
- [X] T063 [P3] [US7] Handle large files in workspace viewer — when file size >1MB, return truncated content with `truncated: true` flag and display truncation notice in UI per edge case.

---

## Phase 10: Polish & Cross-Cutting Concerns

- [X] T064 [P] Add responsive CSS for all new views to `internal/webui/static/style.css` — ensure pipeline detail, persona detail, statistics, run introspection, and workspace browser are usable on desktop and tablet per NFR-004. Test at 1024px and 768px breakpoints.
- [X] T065 [P] Verify JS bundle size stays under 50KB gzipped — measure all static assets (existing app.js, dag.js, sse.js + new markdown.js, highlight.js, stats.js, workspace.js, introspect.js) per NFR-001 and R-007 budget analysis.
- [X] T066 XSS audit — review all new templates and JS for unsanitized user content per FR-031. Verify: all Go template output uses `{{.Field}}` (auto-escaped), markdown parser escapes input, syntax highlighter escapes input, workspace file content is escaped, artifact previews are escaped.
- [X] T067 Verify `webui` build tag gating — confirm all new `.go` files have `//go:build webui` tag. Build without tag and verify no binary size increase per FR-029/SC-008.
- [X] T068 Verify bearer token auth on new API endpoints — confirm all new `/api/*` routes go through existing auth middleware per FR-032.
- [X] T069 Run `go test -race ./internal/webui/...` and `go test -race ./internal/state/...` — fix any race conditions in new code.
- [X] T070 Run `go test ./...` — full test suite must pass with all new code.
- [X] T071 Handle edge case: pipeline definition changed since run — in run detail, display notice when current pipeline config may differ from execution-time config per edge case and C-003.
- [X] T072 Handle edge case: statistics for removed pipelines — ensure GetPipelineStatistics returns historical data for pipelines no longer in manifest, display with "pipeline removed" indicator per edge case.
- [X] T073 Handle edge case: unrecognized file types in syntax highlighting — verify `highlight()` falls back to plain text display without errors per edge case.

---

## Dependency Graph

```
Phase 1 (T001-T005) → Phase 2 (T006-T013) → Phase 3 (T014-T025)
                                             → Phase 4 (T026-T033)
                                             → Phase 5 (T034-T040)
Phase 1 (T001-T005) ────────────────────────→ Phase 6 (T041-T046) [P]
                                             → Phase 7 (T047-T052) [P]
Phase 3 + Phase 4 ──────────────────────────→ Phase 8 (T053-T056)
Phase 5 + Phase 7 ──────────────────────────→ Phase 9 (T057-T063)
All phases ─────────────────────────────────→ Phase 10 (T064-T073)
```

Phases 3, 4, 5 can proceed in parallel after Phase 2 completes.
Phases 6, 7 can proceed in parallel after Phase 1 completes (only need types, not state methods).
Phase 8 depends on Phase 3 (inspection views) and Phase 4 (statistics).
Phase 9 depends on Phase 5 (run introspection) and Phase 7 (syntax highlighting).
Phase 10 is final integration testing after all feature phases.

---

## Task Summary

| Phase | User Story | Priority | Tasks | Parallelizable |
|-------|-----------|----------|-------|----------------|
| 1 | Setup | — | T001-T005 (5) | 5 |
| 2 | Foundational | — | T006-T013 (8) | 6 |
| 3 | US1: Inspection | P1 | T014-T025 (12) | 4 |
| 4 | US2: Statistics | P1 | T026-T033 (8) | 3 |
| 5 | US3: Introspection | P1 | T034-T040 (7) | 1 |
| 6 | US4: Markdown | P2 | T041-T046 (6) | 3 |
| 7 | US5: Syntax HL | P2 | T047-T052 (6) | 3 |
| 8 | US6: Meta Info | P2 | T053-T056 (4) | 3 |
| 9 | US7: Workspace | P3 | T057-T063 (7) | 2 |
| 10 | Polish | — | T064-T073 (10) | 5 |
| **Total** | | | **73** | **35** |
