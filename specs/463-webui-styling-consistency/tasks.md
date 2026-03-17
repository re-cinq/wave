# Tasks

## Phase 1: CSS Foundation — Add Missing Styles

- [X] Task 1.1: Add health page styles to `style.css` — `health-card`, `health-header`, `health-message`, `health-details`, `details-table`, `badge-ok`, `badge-warn`, `badge-error` [P]
- [X] Task 1.2: Add PR badge styles to `style.css` — `badge-merged`, `badge-open`, `badge-closed`, `badge-draft` [P]
- [X] Task 1.3: Add `.additions`/`.deletions` diff coloring and `btn-sm` button variant to `style.css` [P]
- [X] Task 1.4: Add modal/dialog styles to `style.css` — `.modal`, `.form-control`, `.dialog-actions` for the issues pipeline launcher [P]
- [X] Task 1.5: Add `.nav-link-active` style for active navigation highlighting [P]
- [X] Task 1.6: Add `.card-actions` utility class to replace inline `style="margin-top: 0.5rem"` across templates [P]

## Phase 2: Template Unification

- [X] Task 2.1: Update `issues.html` — replace `data-table` with `table` class, remove `table-container` wrapper
- [X] Task 2.2: Update `prs.html` — replace `data-table` with `table` class, remove `table-container` wrapper
- [X] Task 2.3: Update `pipelines.html`, `contracts.html`, `compose.html` — replace inline styles with `.card-actions` class
- [X] Task 2.4: Update `layout.html` — add conditional `nav-link-active` class based on current page

## Phase 3: Active Nav Link Plumbing

- [X] Task 3.1: Add `ActivePage` field to template data structs or handler context
- [X] Task 3.2: Set `ActivePage` in each page handler (runs, pipelines, personas, contracts, skills, compose, issues, prs, health)

## Phase 4: Responsive Polish

- [X] Task 4.1: Add responsive rules for health cards grid at 1024px/768px breakpoints [P]
- [X] Task 4.2: Add responsive rules for dialog/modal at mobile breakpoints [P]
- [X] Task 4.3: Ensure `run-detail-layout` sidebar collapses to stacked layout below 1024px [P]

## Phase 5: Testing & Validation

- [X] Task 5.1: Run `go test ./internal/webui/...` to verify template parsing
- [X] Task 5.2: Run `go test ./...` for full regression check
- [X] Task 5.3: Verify all CSS classes referenced in templates are defined in `style.css`
