# Implementation Plan

## Objective

Fix the PR list page so that the CHANGES column shows additions/deletions/changed-files counts and LABELS display as badges. The root cause is that GitHub's list-PRs REST endpoint omits `additions`, `deletions`, and `changed_files` — these fields are only on the individual PR endpoint.

## Approach

After fetching the PR list via `ListPullRequests`, concurrently call `GetPullRequest` for each PR to enrich it with additions/deletions/changed_files data. Use a bounded worker pool (e.g., 5 concurrent goroutines) to stay within API rate limits. Merge the enriched fields back into the PR structs before building `PRSummary` objects.

Labels already work from the list endpoint — the fix is verified by adding test coverage.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/webui/handlers_prs.go` | Modify | Add `enrichPRStats()` function with concurrent individual PR fetches |
| `internal/webui/handlers_prs_test.go` | Modify | Add tests with mock forge client verifying enrichment and label rendering |

## Architecture Decisions

1. **Enrichment in handler, not forge layer**: The forge `ListPullRequests` stays generic — it returns what the API gives. The webui handler owns the enrichment because it's a presentation concern (the list page needs this data, other consumers of `ListPullRequests` might not).

2. **Bounded concurrency with goroutine pool**: Use a fixed worker pool (5 workers) to avoid hitting GitHub's rate limit. A `sync.WaitGroup` + channel pattern keeps it simple.

3. **Graceful degradation**: If `GetPullRequest` fails for a specific PR, log the error and leave that PR's stats at zero (template renders `--`). Don't fail the entire page.

4. **No caching**: PR stats change with each push. Caching adds complexity without clear benefit for a page that's typically loaded once per interaction.

## Risks

| Risk | Mitigation |
|------|-----------|
| N+1 API calls (up to 50 per page) could be slow | Bounded concurrency (5 workers) parallelizes calls; typical latency ~1-3s for 50 PRs |
| GitHub rate limit (5000/hr for authenticated, 60/hr unauthenticated) | 50 calls per page load is fine for authenticated use; unauthenticated users hit limits fast regardless |
| Individual PR fetch failures | Graceful degradation — log and skip, don't crash the page |
| Context cancellation during enrichment | Pass handler context through; workers respect cancellation |

## Testing Strategy

1. **Mock forge client**: Create a `mockForgeClient` implementing `forge.Client` that returns configurable PR data for both `ListPullRequests` and `GetPullRequest`
2. **Enrichment test**: Verify that `getPRListData` returns enriched stats when `GetPullRequest` provides them
3. **Partial failure test**: Verify graceful degradation when `GetPullRequest` fails for some PRs
4. **Label test**: Verify labels from list endpoint flow through to response
5. **No forge client test**: Existing tests already cover this path
