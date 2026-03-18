# feat(webui): rebuild web UI with GitHub Actions-quality UX and stability

**Issue**: [#455](https://github.com/re-cinq/wave/issues/455)
**Labels**: enhancement, ux, frontend
**Author**: nextlevelshit

## Problem

The current web UI has functional coverage but needs UX polish and stability improvements to match the professionalism of GitHub Actions. While recent work (issues, PRs, health views, composition, skills, contracts, artifact viewer, SSE streaming, cursor pagination, dark/light mode) has brought the webui to feature parity with the TUI, the UX quality gap remains.

## Current State

The webui (`internal/webui/`) already implements:
- **15 HTML pages**: runs, run detail, pipelines, personas, contracts, skills, compose, issues, PRs, health
- **18 API endpoints**: CRUD + SSE streaming + artifact viewer + pipeline start/cancel/retry/resume
- **Real-time updates**: SSE broker with `Last-Event-ID` reconnection backfill (`sse_broker.go`, `handlers_sse.go`)
- **Auth**: Token-based authentication middleware (`auth.go`)
- **Embedded assets**: Static JS/CSS + Go HTML templates via `embed.go`
- **Dark/light mode**: Toggle with Wave brand colors
- **Cursor pagination**: For runs list with status/pipeline/since filters (`pagination.go`)

Test coverage is currently at **53.3%**.

## Scope

- [ ] Audit current webui UX issues — identify rough edges, inconsistencies, and missing polish
- [ ] Improve run list view UX: better status indicators, filtering, sorting
- [ ] Improve run detail view UX: clearer step progress visualization, duration display, log readability
- [ ] Add proper error states, loading indicators, and empty states across all views
- [ ] Improve responsive layout and ensure consistent styling across all 15 pages
- [ ] Improve log streaming UX (auto-scroll, search, collapsible sections)
- [ ] Increase test coverage beyond 53.3% — target handler and SSE edge cases

## Acceptance Criteria

- [ ] All known UX issues audited and resolved or tracked as sub-issues
- [ ] Pipeline runs display with polished real-time status (running, completed, failed) — on par with GitHub Actions
- [ ] Step-level detail view shows clear progress, duration, and streaming logs
- [ ] UI matches the professionalism and clarity of GitHub Actions (see screenshots)
- [ ] No regressions in existing webui functionality (all 18 API endpoints, all 15 pages)
- [ ] Test coverage improved beyond current 53.3%
