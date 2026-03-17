# Implementation Plan: Polish Run List View

## Objective

Polish the webui run list view (`/runs`) with improved status indicators (icon + color + animation), intuitive multi-filter controls (status, pipeline, date range) with active-filter indication, client-side sorting with visual direction indicators, enriched run rows, better hover/click affordances, refined pagination, and a filter-aware empty state.

## Approach

All changes are frontend-only (templates, CSS, JS). The backend already supports status, pipeline, and since filters via `ListRunsOptions`. The HTML handler (`handleRunsPage`) already passes status but not pipeline filter — we add pipeline filter passthrough. Sorting is client-side since the issue says no new API endpoints.

### Key Design Decisions

1. **Status icons**: Use Unicode symbols (✓ ● ✕ ○ ◌) rather than an icon library — keeps the single-binary constraint (no external dependencies). The running indicator uses the existing CSS `pulse` animation.
2. **Sorting**: Client-side JavaScript sort on the table rows. The current cursor pagination fetches a page at a time, so sorting applies within the loaded page. Sort state preserved in URL params.
3. **Pipeline filter**: Use the existing `Pipelines` data already passed to the template (from `getPipelineStartInfos()`). Filter via URL query param `pipeline=<name>`, already supported by the API handler.
4. **Date range filter**: Use native HTML `<input type="date">` for the "since" filter — no JS date picker library needed. Maps to the existing `since` query param.
5. **Active filter indication**: Show a "clear filters" button and highlight active filter controls with a distinct border/background.
6. **Relative timestamps**: Add a JS function to convert absolute times to relative ("2m ago", "1h ago"). Keep absolute time in a `title` attribute for hover.
7. **Row click**: Make entire row clickable via JS click handler (navigates to `/runs/<id>`), with visual cursor change.

## File Mapping

| File | Action | Changes |
|------|--------|---------|
| `internal/webui/templates/runs.html` | modify | Add pipeline dropdown, date range input, sort headers, active-filter bar, filter-aware empty state |
| `internal/webui/templates/partials/run_row.html` | modify | Add status icon, relative timestamp, row click affordance, show tags/trigger info |
| `internal/webui/static/style.css` | modify | Status icon styles, sort header indicators, active filter styles, row hover/click cursor, pagination polish, empty state icon |
| `internal/webui/static/app.js` | modify | Client-side sort logic, relative time formatting, row click handler, filter coordination, active-filter clear |
| `internal/webui/handlers_runs.go` | modify | Pass pipeline filter and pipeline list to template data, pass current filter state for active-filter indication |
| `internal/webui/embed.go` | modify | Add `relativeTime` template function if needed (or handle in JS) |

## Architecture Decisions

1. **No new Go types** — the existing `RunSummary` and `PipelineStartInfo` types have all needed fields
2. **No new API endpoints** — per scope notes, all enhancement is UI-side
3. **Progressive enhancement** — sorting/relative-time are JS-enhanced; the page works without JS (just loses sort and relative time)
4. **Filter state in URL** — all filter/sort params are URL query params so pages are bookmarkable and share-friendly

## Risks

| Risk | Mitigation |
|------|-----------|
| Client-side sort only applies to current page, not full dataset | Document this clearly in UI; cursor pagination already implies partial views |
| Unicode status icons may render differently across platforms | Use widely-supported symbols; test on major browsers |
| Date input styling varies by browser | Use CSS to normalize appearance; native inputs are accessible |
| Adding pipeline filter query param to HTML handler | Backend already supports it — just need to pass it through |

## Testing Strategy

1. **Existing template tests** (`handlers_test.go`) — verify they still pass after template changes
2. **Manual browser testing** — verify each acceptance criterion visually
3. **Accessibility** — verify aria-labels on new controls, keyboard navigation for sort headers
4. **Responsive** — verify mobile layout with new filter controls
5. **No new Go unit tests needed** — changes are primarily template/CSS/JS; the handler change is trivial filter passthrough
