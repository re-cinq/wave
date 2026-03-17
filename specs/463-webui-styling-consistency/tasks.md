# Tasks: WebUI Responsive Layout and Styling Consistency (#463)

## Phase 1: CSS Class Definitions

- [X] 1.1 Add health page styles (health-card, health-header, health-message, health-details, details-table, badge-ok, badge-warn, badge-error)
- [X] 1.2 Add PR badge styles (badge-merged, badge-open, badge-closed, badge-draft)
- [X] 1.3 Add diff coloring (.additions, .deletions) and btn-sm button variant
- [X] 1.4 Add modal/dialog styles (.modal, .form-control, .dialog-actions)
- [X] 1.5 Add active nav link style (.nav-link-active)
- [X] 1.6 Add card-actions utility class

## Phase 2: Template Unification

- [X] 2.1 Unify issues table styling (data-table -> table, remove table-container wrapper)
- [X] 2.2 Unify PRs table styling (data-table -> table, remove table-container wrapper)
- [X] 2.3 Replace inline styles with .card-actions in pipelines.html, contracts.html, compose.html
- [X] 2.4 Add active nav link to layout.html (conditional nav-link-active class)

## Phase 3: Go Handler Changes

- [X] 3.1 Add ActivePage string field to template data in all page handlers
- [X] 3.2 Set ActivePage value in each handler: runs, pipelines, personas, contracts, skills, compose, issues, prs, health

## Phase 4: Responsive Breakpoints

- [X] 4.1 Responsive health cards grid at 1024px and 768px breakpoints
- [X] 4.2 Responsive dialog/modal at mobile breakpoints
- [X] 4.3 Responsive run detail layout — sidebar collapses to stacked below 1024px

## Phase 5: Validation

- [X] 5.1 Run webui tests (go test ./internal/webui/...)
- [X] 5.2 Run full test suite (go test -race ./...)
- [X] 5.3 Verify CSS class coverage (all classes used in HTML have definitions in style.css)
