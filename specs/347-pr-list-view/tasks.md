# Tasks

## Phase 1: Foundation — Data Types and Provider

- [X] Task 1.1: Add `ViewPullRequests` to `ViewType` enum in `views.go` (after `ViewIssues`), update `String()` to return `"Pull Requests"`
- [X] Task 1.2: Create `pr_provider.go` with `PRData` struct (Number, Title, State, Author, Labels, Draft, Merged, Additions, Deletions, ChangedFiles, Comments, Commits, CreatedAt, UpdatedAt, Body, HTMLURL, HeadBranch, BaseBranch), `PRDataProvider` interface with `FetchPRs() ([]PRData, error)`, and `DefaultPRDataProvider` using `github.Client.ListPullRequests()`
- [X] Task 1.3: Create `pr_messages.go` with `PRDataMsg` (PRs []PRData, Err error) and `PRSelectedMsg` (Number int, Title string, Index int)

## Phase 2: Core UI Components

- [X] Task 2.1: Create `pr_list.go` — `PRListModel` with flat navigation (no tree children), filtering by title/number/author/label, cursor navigation, scroll offset, rendering with status badge (Draft/Open/Merged/Closed) and PR number [P]
- [X] Task 2.2: Create `pr_detail.go` — `PRDetailModel` with viewport, metadata display (number, title, state, author, labels, branches, additions/deletions/changed files, created date), and body rendering [P]

## Phase 3: Wiring — ContentModel and App Integration

- [X] Task 3.1: Add `PRProvider` field to `ContentProviders` struct in `content.go`
- [X] Task 3.2: Add `prList *PRListModel`, `prDetail *PRDetailModel`, `prProvider PRDataProvider` fields to `ContentModel`
- [X] Task 3.3: Wire `PRDataProvider` in `NewContentModel()` from `ContentProviders`
- [X] Task 3.4: Update `cycleView()` — change modulo from 7 to 8, add `ViewPullRequests` case for lazy init of `prList`/`prDetail`
- [X] Task 3.5: Update `SetSize()` — propagate dimensions to `prList`/`prDetail` when non-nil
- [X] Task 3.6: Update `IsFiltering()` — add `ViewPullRequests` case checking `prList.filtering`
- [X] Task 3.7: Update `Update()` message routing — add `PRDataMsg` and `PRSelectedMsg` handlers, route key messages via `routeToActiveList`/`routeToActiveDetail`
- [X] Task 3.8: Update `handleAlternativeViewEnter()` and `handleAlternativeViewEscape()` — add `ViewPullRequests` cases
- [X] Task 3.9: Update `routeToActiveList()` and `routeToActiveDetail()` — add `ViewPullRequests` cases
- [X] Task 3.10: Update `View()` — add `ViewPullRequests` rendering case with left/right placeholders
- [X] Task 3.11: Wire `PRDataProvider` in `RunTUI()` in `app.go` — create using existing `ghClient` and `repoSlug`, assign to `cp.PRProvider`

## Phase 4: Testing

- [X] Task 4.1: Create `pr_list_test.go` — tests for Init, DataLoading, Navigation, Filtering (title, number, author, label), EmptyState, EmptyFilter, ViewRendering, UnfocusedIgnoresKeys, NilProvider [P]
- [X] Task 4.2: Create `pr_detail_test.go` — tests for EmptyState, SetPR, SetSize, SelectionUpdatesContent, metadata rendering [P]
- [X] Task 4.3: Update `views_test.go` — add `ViewPullRequests` to `TestViewType_String` table, update `TestContentModel_TabCyclesThroughAllViews` expected views list and modulo

## Phase 5: Validation

- [X] Task 5.1: Run `go test ./internal/tui/...` — all tests pass
- [X] Task 5.2: Run `go test -race ./...` — no race conditions
- [X] Task 5.3: Run `go vet ./...` — no issues
