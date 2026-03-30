# fix(webui): PR list shows empty CHANGES and LABELS columns

**Issue**: [re-cinq/wave#692](https://github.com/re-cinq/wave/issues/692)
**Parent**: #687 (item 6)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: (none)

## Problem

The Pull Requests page shows `--` for the CHANGES column and empty LABELS for all PRs. The GitHub API data for additions/deletions/labels isn't being fetched or mapped into the template.

## Root Cause Analysis

The GitHub REST API's `GET /repos/{owner}/{repo}/pulls` (list endpoint) does **not** return `additions`, `deletions`, or `changed_files` fields. These fields are only available from the individual PR endpoint (`GET /repos/{owner}/{repo}/pulls/{number}`).

The existing code correctly maps these fields at every layer (GitHub client -> forge adapter -> webui handler -> template), but the list endpoint returns them as zero. Since the template condition `{{if or .Additions .Deletions .ChangedFiles}}` is false when all are zero, it renders `--`.

For **labels**: The list endpoint DOES return labels. The code correctly parses them. Labels appear empty only when PRs genuinely have no labels assigned. However, there are no tests verifying label rendering with a mock forge client.

## Files to Investigate

- `internal/webui/handlers_prs.go` — `getPRListData()` needs enrichment after fetching list
- `internal/forge/github.go` — `ListPullRequests()` returns incomplete data from list endpoint
- `internal/webui/templates/prs.html` — template logic is correct, data is the problem

## Acceptance Criteria

- [ ] PR list shows additions+deletions count in CHANGES column
- [ ] PR labels are displayed as badges
- [ ] Enrichment is concurrent with bounded parallelism to avoid API rate limits
- [ ] Graceful degradation: if individual PR fetch fails, show `--` for that row
- [ ] Tests verify enrichment logic and label rendering
