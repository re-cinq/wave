# Tasks

## Phase 1: Backend — Types and Helpers
- [X] Task 1.1: Add `FilterState`, `Page`, `HasMore` fields to `IssueListResponse` and `PRListResponse` in `types.go`
- [X] Task 1.2: Add `parsePageNumber(r)` helper to `pagination.go`

## Phase 2: Backend — Handler Logic
- [X] Task 2.1: Refactor `getIssueListData()` to accept `state` and `page` params, pass to `ListIssues()` [P]
- [X] Task 2.2: Refactor `getPRListData()` to accept `state` and `page` params, pass to `ListPullRequests()` [P]
- [X] Task 2.3: Update `handleAPIIssues` and `handleIssuesPage` to read `?state=` and `?page=` from request [P]
- [X] Task 2.4: Update `handleAPIPRs` and `handlePRsPage` to read `?state=` and `?page=` from request [P]

## Phase 3: Frontend — Templates
- [X] Task 3.1: Add state filter dropdown and pagination to `issues.html`, add State badge column, update empty-state text [P]
- [X] Task 3.2: Add state filter dropdown and pagination to `prs.html`, update empty-state text [P]

## Phase 4: Testing
- [X] Task 4.1: Add state/page query param tests in `handlers_issues_test.go`
- [X] Task 4.2: Add state/page query param tests in `handlers_prs_test.go`
- [X] Task 4.3: Add `parsePageNumber` tests in `pagination_test.go`
- [X] Task 4.4: Run `go test ./internal/webui/...` and fix any failures
