# Tasks

## Phase 1: Add timeout constant
- [X] Task 1.1: Add `ForgeAPIList` constant to `internal/timeouts/timeouts.go` (30 seconds)

## Phase 2: Wire up new timeout
- [X] Task 2.1: Update `getIssueListData` in `internal/webui/handlers_issues.go` to use `timeouts.ForgeAPIList` instead of `timeouts.ForgeAPI`

## Phase 3: Testing
- [X] Task 3.1: Add test in `internal/webui/handlers_issues_test.go` verifying list operations use the longer timeout
- [X] Task 3.2: Run `go test ./internal/webui/... ./internal/timeouts/...` to confirm no regressions

## Phase 4: Validation
- [X] Task 4.1: Run full `go test ./...` to confirm no regressions project-wide
