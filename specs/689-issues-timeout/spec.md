# fix(webui): /issues page times out — context deadline exceeded

**Issue**: [re-cinq/wave#689](https://github.com/re-cinq/wave/issues/689)
**Parent**: #687 (item 2)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

## Problem

The `/issues` page shows `Failed to fetch issues: failed to decode issues: context deadline exceeded`. The GitHub API call to list issues times out on repos with 600+ issues.

## Expected

The page should load within a reasonable time, paginating or limiting the initial fetch.

## Files to investigate

- `internal/webui/handlers_issues.go` — check timeout and pagination
- The forge client's issue listing method — may need a shorter page size or increased timeout

## Acceptance Criteria

- [ ] `/issues` loads without timeout errors
- [ ] Issues are paginated or limited to a reasonable count (e.g., 50)

## Root Cause Analysis

The webui issues handler already paginates correctly (51 per page via `issuesPerPage + 1`). The real problem is the **context timeout**:

1. `getIssueListData` creates a `context.WithTimeout(ctx, timeouts.ForgeAPI)` — **15 seconds**
2. The GitHub `http.Client` is also configured with a 15-second `Timeout`
3. The `doRequest` method shares the context across up to 3 retry attempts with exponential backoff
4. If a request takes ~12s and encounters a transient error, the retry has only ~2s before the context expires
5. The error surfaces as `"failed to decode issues: context deadline exceeded"` when the context cancels during response body decoding

The 15-second timeout is too tight for list operations that may involve retries and rate-limit waits. Single-item fetches (GetIssue, GetPR) complete well within 15 seconds, but list operations need more headroom.
