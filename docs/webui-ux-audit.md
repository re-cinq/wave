# Web UI UX Audit

Systematic audit of all Wave webui pages. Findings categorized by theme with severity ratings.

## Pages Audited

1. Runs list (`/runs`)
2. Run detail (`/runs/{id}`)
3. Pipelines (`/pipelines`)
4. Personas (`/personas`)
5. Contracts (`/contracts`)
6. Skills (`/skills`)
7. Compose (`/compose`)
8. Issues (`/issues`)
9. PRs (`/prs`)
10. Health (`/health`)
11. Not Found (404)
12. Layout (navbar, theme toggle)

---

## Findings by Theme

### Layout & Consistency

| # | Page | Description | Severity | Status |
|---|------|-------------|----------|--------|
| L1 | All | Grid cards use `<div>` instead of `<article>` — poor semantics | P3 | FIXED (pipelines, personas) |
| L2 | All | Status badges use lowercase ("running") — could be title case for polish | P4 | Open |
| L3 | Pipelines vs Personas | Both use `.personas-grid` class — naming implies persona-specific | P4 | Open |
| L4 | Run list | Branch/trigger info not shown in run rows | P2 | FIXED |
| L5 | Run detail | Branch name not shown in run metadata | P2 | FIXED |

### Empty States & Loading

| # | Page | Description | Severity | Status |
|---|------|-------------|----------|--------|
| E1 | Pipelines | `quickStart()` uses `prompt()` — poor UX, no loading feedback | P1 | FIXED |
| E2 | All forms | No success toast before redirect after form submission | P1 | FIXED |
| E3 | Contract viewer | Loading state is plain text, not a spinner | P3 | Open |
| E4 | Run list | "Load more" doesn't show how many runs loaded vs total | P3 | Open |

### Accessibility

| # | Page | Description | Severity | Status |
|---|------|-------------|----------|--------|
| A1 | Run detail | Step card headers not keyboard navigable | P2 | FIXED |
| A2 | Run detail | `aria-expanded` missing on collapsible step cards | P2 | FIXED |
| A3 | Run detail | Steps timeline missing `aria-live` for SSE updates | P2 | FIXED |
| A4 | Layout | SVG logo has no accessible label | P3 | FIXED |
| A5 | 404 | No "Go Back" button for browser history navigation | P3 | FIXED |
| A6 | Pipelines | Grid cards lack `aria-label` describing content | P3 | FIXED |
| A7 | Run list | Sort headers missing `aria-sort` attribute | P3 | Open |
| A8 | Compose | Flow arrows accessible but container lacks description | P4 | Open |

### Responsive Design

| # | Page | Description | Severity | Status |
|---|------|-------------|----------|--------|
| R1 | All | No breakpoint for phones < 480px — buttons too small to tap | P2 | FIXED |
| R2 | Run detail | Log toolbar wraps poorly on mobile | P2 | FIXED |
| R3 | Run detail | Log line numbers take space on small screens | P3 | FIXED (hidden on 480px) |
| R4 | Run list | Filters not stacked on small phones | P2 | FIXED |

### User Feedback

| # | Page | Description | Severity | Status |
|---|------|-------------|----------|--------|
| F1 | Run detail | Retry/resume don't show success toast | P2 | FIXED |
| F2 | All | Button loading state lacks `cursor: not-allowed` | P2 | FIXED |
| F3 | All | Error toasts show generic messages without HTTP status context | P3 | Open |
| F4 | Run detail | Copy log button feedback is small text change, no toast | P3 | Open |

### Cross-Page Inconsistencies

| # | Pages | Description | Severity | Status |
|---|-------|-------------|----------|--------|
| C1 | Contracts/Skills | Loading indicator is text; elsewhere uses spinner | P3 | Open |
| C2 | All grid pages | Grid class named `.personas-grid` used for pipelines, skills too | P4 | Open |

---

## Summary

| Severity | Total | Fixed | Open |
|----------|-------|-------|------|
| P1 | 2 | 2 | 0 |
| P2 | 9 | 9 | 0 |
| P3 | 9 | 3 | 6 |
| P4 | 3 | 0 | 3 |
| **Total** | **23** | **14** | **9** |

All P1 and P2 issues have been addressed. Remaining P3/P4 items are polish improvements that can be tackled in future iterations.
