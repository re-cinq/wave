# Tasks

## Phase 1: Core Fix

- [X] Task 1.1: Add `enrichPRStats` function to `internal/webui/handlers_prs.go` — takes a slice of `*forge.PullRequest`, the forge client, owner, repo, and context; concurrently fetches individual PR details to populate Additions/Deletions/ChangedFiles; uses bounded worker pool (5 goroutines)
- [X] Task 1.2: Wire `enrichPRStats` into `getPRListData` — call after `ListPullRequests` returns, before building `PRSummary` objects

## Phase 2: Testing

- [X] Task 2.1: Create `mockForgeClient` in test file implementing `forge.Client` with configurable `ListPullRequests` and `GetPullRequest` responses [P]
- [X] Task 2.2: Add test `TestGetPRListData_EnrichedStats` — verify enrichment populates Additions/Deletions/ChangedFiles in response [P]
- [X] Task 2.3: Add test `TestGetPRListData_PartialEnrichmentFailure` — verify graceful degradation when some GetPullRequest calls fail [P]
- [X] Task 2.4: Add test `TestGetPRListData_Labels` — verify labels from list endpoint appear in PRSummary [P]

## Phase 3: Validation

- [X] Task 3.1: Run `go test ./internal/webui/...` — all tests pass
- [X] Task 3.2: Run `go vet ./...` — no issues
- [X] Task 3.3: Run `golangci-lint run ./internal/webui/...` — no lint errors
