# Web UI UX Audit

Systematic UX audit of all Wave Dashboard webui pages.

**Date**: 2026-03-18
**Scope**: All HTML templates in `internal/webui/templates/`
**Related**: #459 (parent: #455)

---

## Methodology

1. Reviewed all 17 templates (11 pages + 1 layout + 5 partials)
2. Reviewed CSS (`style.css`) and JS assets (`app.js`, `sse.js`, `log-viewer.js`, `dag.js`)
3. Each finding rated P1 (critical), P2 (important), or P3 (minor)
4. Findings categorized by theme: layout, states, feedback, accessibility, consistency

---

## Template Inventory

| # | Template | Type | Purpose |
|---|----------|------|---------|
| 1 | `layout.html` | Layout | Master page wrapper with navbar, theme toggle, connection banner |
| 2 | `runs.html` | Page | Pipeline runs list with filters, pagination, start form |
| 3 | `run_detail.html` | Page | Single run: DAG, steps, logs, artifacts, events timeline |
| 4 | `pipelines.html` | Page | Pipeline catalog with quick-start buttons |
| 5 | `personas.html` | Page | AI persona cards with capabilities and tools |
| 6 | `contracts.html` | Page | Output validation schema viewer |
| 7 | `skills.html` | Page | Skill/tool registry with pipeline usage |
| 8 | `compose.html` | Page | Composition pipeline flow visualization |
| 9 | `issues.html` | Page | GitHub issues list with pipeline launch dialog |
| 10 | `prs.html` | Page | Pull requests list with state badges |
| 11 | `health.html` | Page | Project health checks dashboard |
| 12 | `notfound.html` | Page | 404 error page |
| 13 | `step_card.html` | Partial | Collapsible step display (used in run detail) |
| 14 | `run_row.html` | Partial | Table row for runs list |
| 15 | `dag_svg.html` | Partial | SVG pipeline DAG visualization |
| 16 | `resume_dialog.html` | Partial | Step resume selector modal |
| 17 | `artifact_viewer.html` | Partial | Artifact content display |

---

## Findings by Page

### Layout (layout.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| L-01 | P1 | Layout | No mobile hamburger menu — 9+ nav links overflow on small screens |
| L-02 | P2 | Accessibility | No skip-to-content link for keyboard navigation |
| L-03 | P3 | Consistency | Theme toggle uses Unicode emoji (☾/☀) instead of SVG icons — inconsistent rendering across platforms |
| L-04 | P3 | Feedback | Connection banner uses inline `onclick` handlers — mixes JS with markup |

### Runs (runs.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| R-01 | P2 | States | Client-side sort state lost on pagination — clicking "Load more" resets sort |
| R-02 | P2 | Feedback | Auto-refresh via `setTimeout(reload, 10000)` when SSE is already available — unnecessary full page reload |
| R-03 | P2 | Feedback | No loading indicator on filter changes |
| R-04 | P3 | Accessibility | Date filter input lacks visible `<label>` element |
| R-05 | P3 | Layout | Start form uses inline `style="display:none"` instead of CSS class toggle |
| R-06 | P3 | Feedback | No visual indicator (arrow/chevron) showing start form expand/collapse state |

### Run Detail (run_detail.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| RD-01 | P2 | Consistency | Inline styles on Steps header div (`display: flex; margin-bottom: 0.75rem`) — should use CSS classes |
| RD-02 | P2 | Consistency | `escapeHTML` function duplicated (also in contracts page JS) |
| RD-03 | P2 | States | Step expanded state stored in both `data-expanded` attribute AND JavaScript `expandedSteps` Set — can desync |
| RD-04 | P3 | Consistency | Emoji characters (⬇📋) for step card download/copy buttons render inconsistently across OS/browsers |
| RD-05 | P3 | States | Error recovery hints use string matching on error messages — fragile, breaks if error text changes |

### Pipelines (pipelines.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| P-01 | P1 | Feedback | Quick-start uses browser `prompt()` dialog instead of styled modal — breaks immersion, blocks everything |
| P-02 | P2 | Consistency | Uses `.personas-grid`/`.persona-card` CSS classes for pipeline cards — semantic mismatch |
| P-03 | P3 | Accessibility | Card grid is div-based with no keyboard navigation support |

### Personas (personas.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| PE-01 | P3 | States | No actions available on persona cards (no detail view, no "use in pipeline" action) |
| PE-02 | P3 | Feedback | No search/filter capability — becomes unwieldy with many personas |
| PE-03 | P3 | Layout | Denied tools section uses red styling but no visual hierarchy distinguishing allowed vs denied |

### Contracts (contracts.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| C-01 | P2 | Consistency | `escapeHTML` duplicated from run_detail page — should be shared utility |
| C-02 | P2 | Consistency | Uses `.personas-grid` CSS class for contract cards — semantic mismatch |
| C-03 | P3 | Feedback | No close button on expanded schema viewer — must click "View Schema" again to collapse |
| C-04 | P3 | Feedback | No copy-to-clipboard for schema content |

### Skills (skills.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| S-01 | P3 | States | No indication of skill install status (installed vs available) |
| S-02 | P3 | Feedback | No search/filter for skills |
| S-03 | P3 | Feedback | Commands displayed without copy-to-clipboard affordance |

### Compose (compose.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| CO-01 | P3 | Layout | Flow visualization renders all steps linearly with arrows — misrepresents DAG/parallel structures |
| CO-02 | P3 | Accessibility | Color-coded node types have no legend explaining the color meanings |
| CO-03 | P3 | Layout | Horizontal flow doesn't break on small screens — requires horizontal scrolling |

### Issues (issues.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| I-01 | P1 | Consistency | Uses HTML5 `<dialog>` element for pipeline launch — inconsistent with resume dialog's custom `<div>` overlay |
| I-02 | P1 | Layout | No pagination — all issues loaded at once, breaks with 100+ issues |
| I-03 | P2 | Consistency | Uses `const`/arrow functions (ES6+) while other pages use `var`/`function` (ES5) — JS syntax inconsistency |

### Pull Requests (prs.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| PR-01 | P1 | Layout | No pagination for pull requests — same issue as issues page |
| PR-02 | P3 | States | No actions column (can't launch review pipeline or merge from UI) |
| PR-03 | P3 | Layout | Long branch names overflow `<code>` blocks |

### Health (health.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| H-01 | P3 | Feedback | "Re-run Checks" button reloads the page with no running/spinner indicator |
| H-02 | P3 | Accessibility | Status icon glyphs (✓✗⚠) are Unicode — no fallback, read as raw characters by screen readers |

### Not Found (notfound.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| NF-01 | P2 | Consistency | Inline style on empty-state div (`padding: 5rem 1rem`) — should use CSS class |
| NF-02 | P3 | Feedback | Only links to /runs — no helpful navigation suggestions |

---

## Findings by Partial Template

### Step Card (step_card.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| SC-01 | P2 | Consistency | Toggle icon uses ▶/▼ — inconsistent with ▲/▼ used elsewhere |
| SC-02 | P2 | Layout | Step header crams 8 items (toggle, ID, spinner, badge, time, duration, persona, buttons) — too dense on mobile |
| SC-03 | P3 | Feedback | Error banner collapse uses CSS `max-height` hack — no smooth re-expand transition |
| SC-04 | P3 | Feedback | Progress bar text overlaps at 100% due to `min-width: 2rem` |

### Run Row (run_row.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| RR-01 | P2 | Accessibility | Clickable row requires JavaScript to navigate — not keyboard-accessible without JS |

### DAG SVG (dag_svg.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| DAG-01 | P3 | Layout | Tooltip positioned with hardcoded offset — goes off-screen on mobile/small viewports |
| DAG-02 | P3 | Accessibility | No touch event handlers — tooltip only works with mouse hover |

### Resume Dialog (resume_dialog.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| RES-01 | P2 | Accessibility | Custom div-based modal instead of HTML5 `<dialog>` — no focus management or keyboard trap |
| RES-02 | P3 | Accessibility | No Escape key handler to close dialog |

### Artifact Viewer (artifact_viewer.html)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| AV-01 | P2 | Consistency | Artifact display code duplicated between this template and inline JS in run_detail.html |
| AV-02 | P3 | Feedback | No copy-to-clipboard for artifact content |
| AV-03 | P3 | Feedback | Truncation notice shown but no download link |

---

## CSS & JS Asset Findings

### style.css

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| CSS-01 | P2 | Consistency | Duplicate `@keyframes spin` definitions |
| CSS-02 | P2 | Consistency | Duplicate `.step-header` cursor/user-select rules |
| CSS-03 | P2 | Accessibility | Status badge color contrast may be insufficient (colored bg + colored text) |
| CSS-04 | P3 | Consistency | Sort indicator uses 0.7rem font — hard to see |
| CSS-05 | P3 | Layout | Table responsive breakpoint converts to `display: block` — poor accessibility |

### JavaScript (app.js, sse.js, log-viewer.js, dag.js)

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| JS-01 | P2 | Consistency | Mixed `var`/`const`/arrow function syntax across files — ES5 in some, ES6+ in others |
| JS-02 | P2 | States | Global variables pollute window namespace (`expandedSteps`, `sseConnection`, `pollTimer`, etc.) |
| JS-03 | P2 | Consistency | Duration formatting duplicated between Go templates and JavaScript — potential divergence |
| JS-04 | P3 | Feedback | `fetchJSON()` shows generic toast for all errors — loses error specificity |
| JS-05 | P3 | States | Theme toggle overwrites system preference in localStorage — no reset-to-system option |
| JS-06 | P3 | Feedback | `startPipeline()` reloads entire page after start — should navigate to run detail |

---

## Cross-Page Consistency Issues

| ID | Severity | Theme | Description |
|----|----------|-------|-------------|
| XP-01 | P2 | Consistency | `.personas-grid` CSS class reused by 4 different page types (personas, pipelines, contracts, compose) — semantic mismatch |
| XP-02 | P2 | Consistency | Two dialog implementations: HTML5 `<dialog>` (issues page) vs custom `<div>` overlay (resume dialog) |
| XP-03 | P2 | Consistency | `escapeHTML` function duplicated across run_detail and contracts pages |
| XP-04 | P2 | Consistency | Inline styles used inconsistently — some pages use CSS classes, others use `style=""` attributes |
| XP-05 | P2 | Consistency | JS syntax varies: issues page uses modern ES6+ (`const`, arrow functions) while others use ES5 (`var`, `function`) |
| XP-06 | P1 | Layout | No pagination on issues and PRs pages — only runs page has pagination |

---

## Summary by Severity

### P1 — Critical (6 findings)

1. **L-01**: No mobile hamburger menu — nav overflow on small screens
2. **P-01**: Pipeline quick-start uses browser `prompt()` instead of styled modal
3. **I-01**: Inconsistent dialog patterns (`<dialog>` vs `<div>` overlay)
4. **I-02**: No pagination on issues page
5. **PR-01**: No pagination on PRs page
6. **XP-06**: Cross-page pagination inconsistency

### P2 — Important (22 findings)

1. **L-02**: Missing skip-to-content link
2. **R-01**: Sort state lost on pagination
3. **R-02**: Unnecessary page reload when SSE available
4. **R-03**: No loading indicator on filter changes
5. **RD-01**: Inline styles on Steps header
6. **RD-02**: Duplicate `escapeHTML` function
7. **RD-03**: Dual state management for step expand/collapse
8. **P-02**: Semantic CSS class mismatch (`.personas-grid` for pipelines)
9. **C-01**: Duplicate `escapeHTML`
10. **C-02**: Semantic CSS class mismatch (`.personas-grid` for contracts)
11. **I-03**: JS syntax inconsistency (ES5 vs ES6+)
12. **NF-01**: Inline styles on 404 page
13. **SC-01**: Inconsistent toggle icons
14. **SC-02**: Dense step card header on mobile
15. **RR-01**: Clickable row not keyboard-accessible
16. **RES-01**: Custom modal lacks focus management
17. **AV-01**: Duplicate artifact display code
18. **CSS-01**: Duplicate `@keyframes spin`
19. **CSS-02**: Duplicate `.step-header` rules
20. **CSS-03**: Status badge color contrast
21. **JS-01**: Mixed JS syntax across files
22. **JS-02**: Global variable namespace pollution

### P3 — Minor (23 findings)

L-03, L-04, R-04, R-05, R-06, RD-04, RD-05, P-03, PE-01, PE-02, PE-03, C-03, C-04, S-01, S-02, S-03, CO-01, CO-02, CO-03, PR-02, PR-03, H-01, H-02, NF-02, SC-03, SC-04, DAG-01, DAG-02, RES-02, AV-02, AV-03, CSS-04, CSS-05, JS-04, JS-05, JS-06

---

## Recommended Priority Order

1. **Navigation & pagination** (L-01, I-02, PR-01, XP-06) — largest impact on usability
2. **Dialog unification** (I-01, RES-01, P-01) — inconsistent patterns confuse users
3. **Code deduplication** (RD-02/C-01/XP-03, AV-01, CSS-01, CSS-02) — reduces maintenance burden
4. **Accessibility** (L-02, RR-01, CSS-03, H-02) — compliance and inclusivity
5. **State management** (RD-03, R-01, JS-02) — prevents subtle bugs
6. **Consistency polish** (XP-01, JS-01, inline styles) — visual and code coherence
