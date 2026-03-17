# Implementation Plan

## Objective

Fix styling inconsistencies across all 15 webui pages so they use a unified CSS system — consistent classes for tables, badges, buttons, health cards, and dialog modals — and ensure responsive behavior at 1024px, 1440px, and 1920px breakpoints with polished dark/light mode support.

## Approach

The root cause is that pages were built incrementally, each introducing ad-hoc CSS classes (or missing them entirely). The fix is:

1. **Define missing CSS classes** in `style.css` for all undefined selectors used in templates
2. **Unify table styling** — `issues.html` and `prs.html` use `data-table` (undefined) instead of `table` (defined)
3. **Add active nav link styling** — layout has no way to highlight the current page
4. **Add missing badge variants** for PRs (`badge-merged`, `badge-open`, `badge-closed`, `badge-draft`) and health (`badge-ok`, `badge-warn`, `badge-error`)
5. **Add missing button variant** `btn-sm` used in issues page
6. **Add health page styles** — `health-card`, `health-header`, `health-message`, `health-details`, `details-table` are all unstyled
7. **Add dialog/modal styles** for the issues page pipeline launcher
8. **Add PR diff coloring** for `.additions`/`.deletions` spans
9. **Eliminate inline styles** in templates by using CSS classes
10. **Verify responsive breakpoints** cover all page types (tables, grids, detail layouts)

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/webui/static/style.css` | modify | Add missing classes, unify patterns, improve responsive rules |
| `internal/webui/templates/issues.html` | modify | Replace `data-table` with `table`, `table-container` with standard wrapper |
| `internal/webui/templates/prs.html` | modify | Replace `data-table` with `table`, add standard wrapper |
| `internal/webui/templates/health.html` | modify | Minor class alignment if needed after CSS additions |
| `internal/webui/templates/layout.html` | modify | Add active nav link support (pass current page to template) |
| `internal/webui/templates/pipelines.html` | modify | Remove inline `style=` attributes |
| `internal/webui/templates/contracts.html` | modify | Remove inline `style=` attributes |
| `internal/webui/templates/compose.html` | modify | Remove inline `style=` attributes |
| `internal/webui/routes.go` | modify | Pass active page name to layout template context |
| `internal/webui/server.go` | modify | Ensure template data includes page identifier (if not already) |

## Architecture Decisions

1. **No CSS framework** — keep the single `style.css` approach. The file is ~500 lines and manageable.
2. **CSS custom properties for spacing** — not adding a spacing scale (e.g. `--space-1`) since the existing hardcoded rem values (0.5, 0.75, 1, 1.25, 1.5) are consistent enough. Adding a scale would be over-engineering.
3. **Active nav via template data** — pass the current page name to the layout template so it can apply `.nav-link-active` conditionally. This is a minimal Go change.
4. **Semantic class renaming deferred** — `personas-grid`/`persona-card` are used by pipelines, contracts, skills, and compose pages. Renaming to generic `card-grid`/`item-card` would be cleaner but touches many files for cosmetic benefit. Deferring.

## Risks

| Risk | Mitigation |
|------|------------|
| CSS changes affect existing pages negatively | Verify each page after changes; the existing class names are scoped enough to avoid cascade issues |
| Active nav link requires Go handler changes | Minimal change — add a string field to template data |
| Template changes cause rendering errors | Run `go test ./internal/webui/...` to catch template parsing failures |

## Testing Strategy

1. **Unit tests**: Run existing `go test ./internal/webui/...` — template parsing tests will catch syntax errors
2. **Visual verification**: Manual browser check at 1024px, 1440px, 1920px viewports in both dark and light mode
3. **Regression**: `go test ./...` to ensure no side effects on other packages
