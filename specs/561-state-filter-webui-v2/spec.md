# feat(webui): add closed/merged state filter to issues and PR views

**Issue**: [#561](https://github.com/re-cinq/wave/issues/561)
**Parent**: Extracted from #550 — Feature 5
**Labels**: enhancement, ux, frontend
**Author**: nextlevelshit

## Problem

Both `handleAPIIssues` and `handleAPIPRs` hardcode `State: "open"` — closed issues and merged PRs are invisible in the web UI.

## Changes Required

### Backend (minimal)
- `handlers_issues.go`: read `?state=` query param (default `"open"`), pass to `ListIssues()`
- `handlers_prs.go`: same for `ListPullRequests()`
- Both `github.Client` methods already accept `State` — no client changes needed

### Frontend
- Add state filter toggle (Open / Closed / All) to `issues.html` and `prs.html` templates
- Update empty-state text from hardcoded "No open issues" to reflect current filter
- Add pagination controls (backend already supports `PerPage: 50`)
- Ensure state badges render correctly for closed/merged items (PR template already has them)

## Acceptance Criteria

- [ ] Issues page shows closed issues when filter toggled
- [ ] PRs page shows merged PRs when filter toggled
- [ ] State badges (open, closed, merged, draft) display correctly
- [ ] Default view remains "open" for backward compatibility
- [ ] Pagination works for large lists
