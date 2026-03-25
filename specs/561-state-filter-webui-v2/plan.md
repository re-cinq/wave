# Implementation Plan — Issue #561

## Objective

Add a state filter (open/closed/all) to the web UI's issues and PRs pages, and add page-based pagination, so users can view closed issues and merged PRs.

## Approach

Follow the existing filter pattern established in `handlers_runs.go` / `runs.html`:
1. Read `?state=` and `?page=` query params in handlers (default: `"open"`, page 1)
2. Pass them through to the GitHub client methods (which already accept `State` and `Page`)
3. Add filter `<select>` dropdowns and pagination links to templates, matching the runs page UX
4. Add a `State` field to response types so the frontend knows the active filter
5. Add `State` column/badge to issues template (PRs already have it)

The GitHub client (`ListIssuesOptions`, `ListPullRequestsOptions`) already supports `State: "open"|"closed"|"all"` and `Page`/`PerPage` — no client changes needed.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/webui/handlers_issues.go` | modify | Read `?state=`/`?page=` from request, pass to `ListIssues()`, add state/page to response |
| `internal/webui/handlers_prs.go` | modify | Read `?state=`/`?page=` from request, pass to `ListPullRequests()`, add state/page to response |
| `internal/webui/types.go` | modify | Add `State`, `Page`, `HasMore` fields to `IssueListResponse` and `PRListResponse` |
| `internal/webui/templates/issues.html` | modify | Add state filter dropdown, state badge column, dynamic empty-state text, pagination |
| `internal/webui/templates/prs.html` | modify | Add state filter dropdown, dynamic empty-state text, pagination |
| `internal/webui/handlers_issues_test.go` | modify | Add tests for state/page query param parsing |
| `internal/webui/handlers_prs_test.go` | modify | Add tests for state/page query param parsing |

## Architecture Decisions

1. **Page-based pagination (not cursor-based)**: The GitHub API uses page numbers. Cursor-based pagination (used for runs, which are in SQLite) doesn't apply here. Simple `?page=N` with prev/next links.

2. **State validation**: Accept only `"open"`, `"closed"`, `"all"`. Invalid values silently default to `"open"` (defensive, no error page for a bad query param).

3. **PerPage fixed at 50**: Keep the existing 50-item limit. Adding a user-controllable page size is out of scope.

4. **Filter via full page reload (URL params)**: Match the runs page pattern — filter changes update the URL and reload the page. This keeps state shareable via URL and avoids client-side JS complexity.

5. **Reuse `parsePageSize` pattern**: For page number parsing, add a small `parsePageNumber(r)` helper alongside the existing `parsePageSize` in `pagination.go`.

## Risks

| Risk | Mitigation |
|------|-----------|
| State filter breaks "Launch Pipeline" button on issues page | Buttons use `data-issue-url` from row data — unaffected by filter state |
| GitHub API rate limits with frequent page navigation | Already mitigated by 15s context timeout; 50-item pages are reasonable |
| Template changes break existing rendering | All changes are additive; existing badge CSS already handles open/closed/merged/draft states |

## Testing Strategy

1. **Unit tests** (`handlers_issues_test.go`): Test `?state=open`, `?state=closed`, `?state=all`, `?state=invalid` (defaults to open), `?page=2`, `?page=0` (defaults to 1)
2. **Unit tests** (`handlers_prs_test.go`): Same state/page param tests
3. **Unit test** (`pagination_test.go`): Test `parsePageNumber()` helper
4. **Existing tests**: Verify all existing tests still pass (they use no query params → default "open" behavior preserved)
