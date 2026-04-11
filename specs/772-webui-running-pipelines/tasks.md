# Tasks: Expandable Running Pipelines Section

**Feature**: `772-webui-running-pipelines`  
**Branch**: `772-webui-running-pipelines`  
**Generated**: 2026-04-11  
**Spec**: `specs/772-webui-running-pipelines/spec.md`  
**Plan**: `specs/772-webui-running-pipelines/plan.md`

## Summary

4 phases, 17 tasks. Pure UI change: handler data extension, template block insertion,
CSS additions, and test coverage. No new routes, no new files, no new dependencies.

All source changes are confined to `internal/webui/`:
- `handlers_runs.go` — add `RunningRuns`/`RunningCount` to template data struct (Phase 2)
- `templates/runs.html` — insert `rp-section` block (Phase 3)
- `static/style.css` — add `.rp-*` CSS classes (Phase 3, parallelizable)
- `handlers_runs_test.go` — extend with `TestHandleRunsPage_RunningSection` cases (Phase 4)

---

## Phase 1 — Setup

> Verify the development environment and understand the test harness before modifying code.

- [X] T001 [P1] Run `go test ./internal/webui/... -run TestHandleRunsPage` to confirm existing runs page tests pass as baseline. File: `internal/webui/handlers_runs_test.go`
- [X] T002 [P1] Read `internal/webui/handlers_runs.go:155–233` to understand the full `handleRunsPage` handler structure, especially the `data` struct and existing `ListRuns` call pattern.
- [X] T003 [P1] Read `internal/webui/templates/runs.html` in full to understand the current template structure (toolbar position, `.wr-list` class, `{{define "scripts"}}` block).
- [X] T004 [P1] Read `internal/webui/static/style.css` (`.wr-*` block and `.badge` definitions) to understand the CSS patterns the new `.rp-*` classes must match.

---

## Phase 2 — Handler Extension (US1 + US3 + FR-008 prerequisite)

> Extend `handleRunsPage` to query and expose running runs. This is the blocking prerequisite
> for all template and test work.

**User Story 1**: View Active Pipelines at a Glance (P1)  
**User Story 3**: Empty State — No Running Pipelines (P3)  
**FR-008**: Filter integration (pipeline filter applied to running query)

- [X] T005 [P1][S1] In `internal/webui/handlers_runs.go`, after line 159 (`pipelineFilter := ...`), add a second `ListRuns` query: `opts := state.ListRunsOptions{Status: "running", PipelineName: pipelineFilter, Limit: 0}`. Assign result to `runningRecs`. File: `internal/webui/handlers_runs.go`
- [X] T006 [P1][S1] Convert `runningRecs` to `[]RunSummary` using the same `runToSummary` loop pattern used for the main list (lines 188–192). Apply `enrichRunSummaries`. Filter to top-level only (exclude `ParentRunID != ""`). Assign to `runningRuns []RunSummary`. File: `internal/webui/handlers_runs.go`
- [X] T007 [P1][S1] Extend the anonymous `data` struct (line 211) with two new fields: `RunningRuns []RunSummary` and `RunningCount int`. Populate them: `RunningRuns: runningRuns, RunningCount: len(runningRuns)`. File: `internal/webui/handlers_runs.go`
- [X] T008 [P1][S1] Verify `RunningRuns` is initialized as an empty slice (not nil) when no running runs exist — use `make([]RunSummary, 0)` if `runningRecs` is empty. Ensures template `{{if eq .RunningCount 0}}` renders correctly. File: `internal/webui/handlers_runs.go`

---

## Phase 3 — Template + CSS (US1, US2, US3, US4)

> Add the `rp-section` HTML block to `runs.html` and the supporting CSS classes.
> T009–T012 (template) and T013–T014 (CSS) are independent and can be worked in parallel
> after Phase 2 is complete.

**User Story 1**: View Active Pipelines at a Glance (P1)  
**User Story 2**: Collapse/Expand Running Pipelines Section (P2)  
**User Story 3**: Empty State — No Running Pipelines (P3)  
**User Story 4**: Navigate to Run Detail from Running Section (P2)

### 3a — Template (runs.html)

- [X] T009 [P] [S1,S2,S3,S4] In `internal/webui/templates/runs.html`, insert a `<div class="rp-section">` container block immediately after the closing `</div>` of `.wr-toolbar` and before `<div class="wr-list">`. The block structure:
  ```html
  <div class="rp-section">
    <div class="rp-header" role="button" tabindex="0"
         aria-expanded="true" aria-controls="rp-section-body"
         onclick="toggleRunningSection()"
         onkeydown="if(event.key==='Enter'||event.key===' '){toggleRunningSection();event.preventDefault()}">
      <span class="rp-label">Running</span>
      <span class="rp-badge">{{.RunningCount}}</span>
      <span class="rp-chevron">▾</span>
    </div>
    <div id="rp-section-body" class="rp-body">
      ...empty-state or run cards (T010, T011)...
    </div>
  </div>
  ```
  File: `internal/webui/templates/runs.html`

- [X] T010 [P] [S3] Inside `<div class="rp-body">`, add the `{{if eq .RunningCount 0}}` empty-state block:
  ```html
  {{if eq .RunningCount 0}}
  <div class="rp-empty">
    <p>No pipelines running</p>
    <a href="/pipelines" class="rp-cta">Start a pipeline →</a>
  </div>
  {{else}}
  ...run cards (T011)...
  {{end}}
  ```
  File: `internal/webui/templates/runs.html`

- [X] T011 [P] [S1,S4] In the `{{else}}` branch of T010, add a `{{range .RunningRuns}}` loop that renders each run card. Reuse the existing `.wr-run` card markup pattern from the main list (copy the `<a href="/runs/{{.RunID}}">` card structure). Each card must be a navigable `<a>` link to `/runs/{{.RunID}}`. File: `internal/webui/templates/runs.html`

- [X] T012 [S2] In the `{{define "scripts"}}` block of `runs.html`, add the `toggleRunningSection()` JS function:
  ```js
  function toggleRunningSection() {
    const section = document.querySelector('.rp-section');
    const header = section.querySelector('.rp-header');
    const body = document.getElementById('rp-section-body');
    const expanded = header.getAttribute('aria-expanded') === 'true';
    header.setAttribute('aria-expanded', String(!expanded));
    section.classList.toggle('collapsed', expanded);
  }
  ```
  File: `internal/webui/templates/runs.html`

### 3b — CSS (style.css)

- [X] T013 [P] [S1,S2,S3] Add the following new CSS rule blocks to `internal/webui/static/style.css` in the "runs overview" section (near `.wr-*` rules). Classes to add:
  - `.rp-section` — `margin-bottom` matching `.wr-toolbar`
  - `.rp-header` — `display: flex; align-items: center; gap: …; cursor: pointer;` with hover state
  - `.rp-label` — typography matching existing section headings
  - `.rp-badge` — inherits `.badge` base; uses `var(--color-running)` background
  - `.rp-chevron` — `transition: transform …;` for rotate animation
  - `.rp-body` — `display: block;`
  - `.rp-empty` — `display: flex; flex-direction: column; align-items: center; padding: …; color: var(--color-muted);`
  - `.rp-cta` — styled as secondary button/link (match existing button patterns)
  File: `internal/webui/static/style.css`

- [X] T014 [P] [S2] Add collapse CSS rules for when `.rp-section.collapsed` is active:
  ```css
  .rp-section.collapsed .rp-body { display: none; }
  .rp-section.collapsed .rp-chevron { transform: rotate(-90deg); }
  ```
  File: `internal/webui/static/style.css`

---

## Phase 4 — Tests (All Stories)

> Extend `handlers_runs_test.go` with test cases covering all contract invariants.
> Tests verify handler data correctness (SSR only — no browser test needed).

- [X] T015 [P1][S1,S3] Add `TestHandleRunsPage_RunningSection_Populated` to `internal/webui/handlers_runs_test.go`. Setup: insert a run record with `Status="running"` into the test store. Assert: response body contains `rp-section`, `rp-badge` with count "1", and a link `href="/runs/{runID}"`. File: `internal/webui/handlers_runs_test.go`

- [X] T016 [P1][S3] Add `TestHandleRunsPage_RunningSection_Empty` to `internal/webui/handlers_runs_test.go`. Setup: no running runs in test store. Assert: response body contains `rp-empty` and `href="/pipelines"` CTA. File: `internal/webui/handlers_runs_test.go`

- [X] T017 [P1][S1] Add `TestHandleRunsPage_RunningSection_FilterRespected` to `internal/webui/handlers_runs_test.go`. Setup: two running runs with different pipeline names; request with `?pipeline=<name1>`. Assert: response body contains only the run card for `name1`, not `name2`. Verifies FR-008 (filter applied to running section). File: `internal/webui/handlers_runs_test.go`

---

## Dependency Order

```
T001–T004 (read/verify)
    ↓
T005–T008 (handler extension — sequential, each builds on prior)
    ↓
T009–T014 (template + CSS — T009→T010→T011→T012 sequential; T013–T014 parallel with T009–T012)
    ↓
T015–T017 (tests — parallel with each other; sequential after all source changes)
```

## Parallelizable Tasks

T009, T010, T011 (template work) can proceed in parallel with T013, T014 (CSS) after Phase 2.  
T015, T016, T017 (tests) are independent of each other and can run in parallel.
