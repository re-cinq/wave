# Implementation Plan: #689 — /issues page timeout fix

## Objective

Fix the `/issues` page timeout by increasing the context deadline for forge list operations from 15s to 30s.

## Approach

Add a dedicated `ForgeAPIList` timeout constant (30 seconds) for list operations and use it in the webui issues handler. This separates single-item fetch timeouts (15s, adequate) from list operation timeouts (30s, needed for retries + backoff on large repos).

The pagination logic (`issuesPerPage = 50`, fetch 51 to detect "has more") is already correct and does not need changes.

## File Mapping

| File | Action | Change |
|------|--------|--------|
| `internal/timeouts/timeouts.go` | modify | Add `ForgeAPIList = 30 * time.Second` constant |
| `internal/webui/handlers_issues.go` | modify | Use `timeouts.ForgeAPIList` in `getIssueListData` context |
| `internal/webui/handlers_issues_test.go` | modify | Add test verifying list operations use longer timeout |

## Architecture Decisions

1. **Separate list timeout vs bumping global ForgeAPI**: A dedicated `ForgeAPIList` constant keeps single-item fetches (GetIssue, GetPR) at the tighter 15s while giving list operations the headroom they need. This is more precise than a blanket increase.

2. **No HTTP client timeout change**: The `http.Client.Timeout` of 15s applies per individual request and is fine — the problem is the shared context across retries. Changing the HTTP client timeout would be a broader change with wider impact.

3. **No pagination changes**: The existing pagination at 50 items/page is already correct and matches the acceptance criteria. GitHub's API returns paginated results efficiently — the issue is purely timeout-related.

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| 30s still insufficient for extreme rate-limit backoff | Low | Rate limiter already has its own wait logic; 30s covers 2 full retries with backoff |
| Users notice slower error feedback on actual failures | Low | 30s is still reasonable for a page load; real API errors return immediately |

## Testing Strategy

- **Unit test**: Verify `getIssueListData` creates a context with the correct (longer) timeout by testing with a mock forge client that measures the deadline
- **Existing tests**: Ensure all existing `handlers_issues_test.go` tests still pass (they use nil forge client, unaffected)
- **Manual**: Verify `/issues` loads on the wave repo (600+ issues) without timeout
