# Research: Expandable Running Pipelines Section

**Feature**: `772-webui-running-pipelines`  
**Phase**: 0 — Outline & Research  
**Date**: 2026-04-11

## Summary

This feature adds an expandable "Running Pipelines" section to the runs overview page (`/runs`).
The section sits between the filter/search toolbar and the main run list. Research covers four
areas: existing collapse/expand patterns, data-access patterns for running runs, filter
integration, and accessibility requirements.

---

## Finding 1: Collapse/Expand Pattern

**Decision**: Reuse the `step_card.html` inline-JS toggle pattern.

**Rationale**: `step_card.html` already establishes the project's expand/collapse idiom with
`role="button"`, `tabindex="0"`, `aria-expanded`, `aria-controls`, `onclick`, and `onkeydown`
handlers — all inline vanilla JS, no dependencies. The sidebar uses `localStorage` for
persistence, but FR-002 explicitly requires collapse state NOT to be persisted across page
loads. A simple `data-expanded` attribute on the section container is sufficient.

**Implementation pattern** (from `step_card.html`):
```html
<div id="rp-section-header"
     role="button" tabindex="0"
     aria-expanded="true"
     aria-controls="rp-section-body"
     onclick="toggleRunningSection()"
     onkeydown="if(event.key==='Enter'||event.key===' '){toggleRunningSection();}">
```

**Alternatives rejected**:
- `<details>/<summary>` HTML element: accessible by default, but cross-browser `open` attribute
  styling is inconsistent with the existing design language, and overriding the default triangle
  requires non-trivial CSS resets. Not used anywhere else in the codebase.
- CSS-only checkbox hack: no JavaScript, but harder to manage `aria-expanded` synchronisation
  and inconsistent with the rest of the UI.

---

## Finding 2: Running Runs Data Access

**Decision**: Fetch running runs in `handleRunsPage` using `state.ListRunsOptions{Status: "running"}`.

**Rationale**: `handleRunsPage` already accepts a `status` query parameter and converts it to
`ListRunsOptions.Status` for the DB query. Adding a second parallel query for `status=running`
(ignoring the user-supplied status filter for the section data, then applying it in Go) is the
minimal change: one extra `s.store.ListRuns()` call at handler start, before the main list
query. No new API endpoints, no client-side fetch, no SSE (CL-001 resolved as page-reload-only).

**Template data extension**: The handler's anonymous `data` struct gains two fields:
- `RunningRuns []RunSummary` — running-only subset (post-enrichment, pre-nesting)
- `RunningCount int` — `len(RunningRuns)` for the header badge

**Alternatives rejected**:
- Separate `/api/runs?status=running` fetch in JavaScript: requires client-side fetch on page
  load, adds latency, inconsistent with the SSR pattern used for all other list data.
- Reusing the existing `Runs` field filtered in the template: Go templates cannot filter slices;
  would require a custom template function, which is more invasive than a handler change.

---

## Finding 3: Pipeline-Name Filter Integration (FR-008)

**Decision**: Apply `FilterPipeline` to the running runs query in the handler.

**Rationale**: The existing filter bar sends `?pipeline=<name>` as a query parameter. The
`handleRunsPage` handler already reads `pipelineFilter := r.URL.Query().Get("pipeline")` and
passes it to `ListRunsOptions.PipelineName`. The running-runs query should use the same
`pipelineFilter` value so the section respects the active filter. Status filter tabs are
irrelevant for the running section (it always shows running runs), but pipeline-name filter
applies. The `FilterStatus` tabs change the main list view; they should NOT suppress the running
section (the section always shows running runs regardless of which status tab is active).

**Alternatives rejected**:
- Client-side filtering in JS after page load: inconsistent with server-side filtering approach;
  the existing `filterRuns()` JS function is for the in-page text-search input only, not
  server-side pipeline filter.

---

## Finding 4: Accessibility (FR-010)

**Decision**: Follow the `step_card.html` pattern — `role="button"`, `tabindex="0"`,
`aria-expanded`, `aria-controls`, keyboard handler for Enter/Space.

**Rationale**: This is already the established accessible pattern in the codebase. The section
body `<div>` gets `id="rp-section-body"` so `aria-controls` on the header can reference it.
`aria-expanded` is toggled by the inline JS function.

**Count badge** (FR-009): `aria-label="N running pipelines"` on the badge span is sufficient.

**Empty state CTA** (FR-005): Standard `<a href="/pipelines">` link — native link semantics,
keyboard-accessible by default.

---

## Finding 5: CSS / Styling

**Decision**: Reuse existing `.wr-*` CSS classes with a new `.rp-section` wrapper; no new
CSS classes for the run cards themselves.

**Rationale**: The existing `.wr-run`, `.wr-accent`, `.wr-body`, `.wr-row1`, `.wr-row2`,
`.wr-name`, `.wr-status`, `.wr-dur`, `.wr-date` classes (and their `.st-running` status
variant) already provide correct run card styling. The section needs only a container with a
header and a collapsible body. New CSS needed: `.rp-section` container, `.rp-header` with
toggle chevron, `.rp-body[hidden]` for collapsed state.

**Collapsed state**: Use `hidden` attribute on body `<div>` — CSS `[hidden] { display: none }`
is standard and requires no extra CSS. Toggled via JS: `el.hidden = !el.hidden`.

---

## Finding 6: Empty-State CTA Destination

**Decision**: Link to `/pipelines` page.

**Rationale**: The empty-state CTA (FR-005, US-3) should direct users where they can start a
pipeline run. Wave's webui has a `/pipelines` route (routes.go) that lists available pipelines.
Clicking a pipeline on that page presents the start-run form. This is the natural starting point
for initiating a new run, and it exists in the current codebase.

**Alternatives rejected**:
- Direct "Run pipeline" button that POSTs to `/api/runs`: requires knowing the pipeline name and
  input upfront; inappropriate in an empty state.

---

## File Impact Summary

| File | Change Type | Reason |
|------|-------------|--------|
| `internal/webui/handlers_runs.go` | Modify | Add second `ListRuns` query for running runs; extend template data struct |
| `internal/webui/templates/runs.html` | Modify | Insert `rp-section` block between toolbar and `wr-list`; add toggle JS |
| `internal/webui/static/style.css` | Modify | Add `.rp-section`, `.rp-header`, `.rp-body`, `.rp-empty` styles |
| `internal/webui/handlers_runs_test.go` | Modify | Add test cases for running section data (RunningRuns field) |
