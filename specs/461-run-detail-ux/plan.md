# Implementation Plan — #461 Run Detail UX

## Objective

Transform the run detail page from a flat step list into a GitHub Actions-style view with collapsible step sections, per-step duration/start time, animated running indicators, prominent error display, and improved log readability.

## Approach

This is primarily a frontend change (HTML templates, CSS, JS) with a minor Go handler enhancement. The existing data model already carries most needed fields (`StartedAt`, `Duration`, `State`, `Error`, `Artifacts`). The main work is:

1. **Collapsible step cards** — add a toggle button to each step header that collapses/expands the step body. Default: running/failed steps expanded, completed/pending collapsed.
2. **Enhanced step header** — show step name, status badge, duration, and start time inline in the header row.
3. **Animated running indicator** — CSS animation (spinner or pulsing dot) for steps in `running` state, beyond the existing badge pulse.
4. **Prominent failed steps** — add a distinct error banner with expandable details within the step card.
5. **Run header enhancement** — add start time and ensure trigger info (from tags) is shown.
6. **Human-friendly duration** — the Go `formatDurationValue` already outputs `Xm Ys` format; ensure JS `createStepCard` mirrors this for SSE-updated cards.
7. **Log output styling** — improve the artifact viewer's `<pre>` blocks with line numbers and keyword highlighting (errors in red, warnings in yellow) via CSS classes.
8. **Start time display** — pass `StartedAt` through to the step card template and JS.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/webui/templates/partials/step_card.html` | modify | Add collapsible toggle, start time, animated indicator |
| `internal/webui/templates/run_detail.html` | modify | Enhance run header with start time/trigger info |
| `internal/webui/static/style.css` | modify | Collapsible step styles, running animation, error prominence, log highlighting |
| `internal/webui/static/sse.js` | modify | Update `createStepCard` JS to match new template; add collapse toggle logic; add duration formatting JS function |
| `internal/webui/types.go` | modify | Ensure `StepDetail.StartedAt` is serialized as a formatted string for templates |
| `internal/webui/handlers_runs.go` | modify | Add `FormattedStartedAt` field to step details for template rendering |
| `internal/webui/templates/partials/dag_svg.html` | modify | Minor: add duration display inside DAG nodes |

## Architecture Decisions

1. **No new API endpoints** — the existing `/api/runs/{id}` response already includes `steps[].started_at` and `steps[].duration`. We just need to render them in the template.
2. **Collapsible via CSS + JS** — use a `<details>`/`<summary>` pattern for collapsible steps (semantic HTML, no framework needed). Fallback: CSS class toggle.
3. **Reuse existing DAG** — the DAG sidebar already shows status per step. Enhance it with duration text, don't rebuild.
4. **Client-side duration formatting** — add a JS `formatDuration` function for SSE-updated cards rather than round-tripping to the server.
5. **Log highlighting via CSS classes** — inject `<span class="log-error">` / `<span class="log-warning">` around matched patterns in the artifact viewer JS, not server-side.

## Risks

| Risk | Mitigation |
|------|------------|
| `<details>` element may not match design intent | Can fallback to div + CSS class toggle with JS |
| SSE-updated step cards may lose collapse state | Track expanded state in a JS Set by stepID; re-apply after DOM rebuild |
| Artifact viewer log highlighting may be slow for large logs | Apply regex only on visible/loaded content, not full artifact body |
| Existing CSS specificity conflicts | Scope new styles under `.step-card-v2` or use BEM-like naming to avoid conflicts |

## Testing Strategy

1. **Manual testing** — load a run detail page with running, completed, failed, and pending steps; verify visual accuracy
2. **Go unit tests** — verify `formatDurationValue` edge cases (sub-second, hours, zero)
3. **Template rendering tests** — add test case in `handlers_test.go` to ensure run detail page renders without template errors when step cards include new fields
4. **Responsive check** — verify collapsible steps work on mobile viewport
5. **SSE integration** — verify step cards update correctly via SSE without losing collapse state
