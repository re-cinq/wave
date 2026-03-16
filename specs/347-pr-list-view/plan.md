# Implementation Plan: PR List View

## Objective

Add a "Pull Requests" view to the Wave TUI that mirrors the Issues view architecture, allowing users to browse, filter, and inspect PRs from the terminal.

## Approach

Mirror the existing Issue list/detail pattern exactly. The TUI uses a consistent architecture for alternative views: a `ViewType` enum entry, lazy-initialized list + detail models in `ContentModel`, a data provider interface, and message types for async data flow. The PR view follows this same pattern with PR-specific data types and rendering.

The GitHub client already has `ListPullRequests` and `GetPullRequest` methods plus a full `PullRequest` struct — no API layer changes needed.

## File Mapping

### New Files (create)
| File | Purpose |
|------|---------|
| `internal/tui/pr_provider.go` | `PRData` struct, `PRDataProvider` interface, `DefaultPRDataProvider` using GitHub client |
| `internal/tui/pr_messages.go` | `PRDataMsg`, `PRSelectedMsg` message types |
| `internal/tui/pr_list.go` | `PRListModel` — left pane list with filtering, navigation, status badges |
| `internal/tui/pr_detail.go` | `PRDetailModel` — right pane detail with viewport, metadata, body |
| `internal/tui/pr_list_test.go` | Unit tests for PR list model (data loading, navigation, filtering, rendering) |
| `internal/tui/pr_detail_test.go` | Unit tests for PR detail model (empty state, set PR, rendering) |

### Modified Files (modify)
| File | Change |
|------|--------|
| `internal/tui/views.go` | Add `ViewPullRequests` to `ViewType` enum and `String()` method |
| `internal/tui/content.go` | Add `prList`/`prDetail`/`prProvider` fields, wire into `cycleView()`, `Update()`, `View()`, `SetSize()`, `IsFiltering()`, `handleAlternativeViewEnter/Escape`, `routeToActiveList/Detail` |
| `internal/tui/app.go` | Wire `PRDataProvider` in `RunTUI()` using existing `ghClient` |
| `internal/tui/views_test.go` | Add `ViewPullRequests` to `TestViewType_String` table, update cycle count |

## Architecture Decisions

1. **Use GitHub API client, not `gh` CLI**: The codebase already has `github.Client.ListPullRequests()` — reuse it for consistency and testability (mock-friendly interface vs subprocess execution).

2. **Separate view from Issues**: Rather than combining Issues and PRs in one view, add a distinct `ViewPullRequests` entry. This matches the existing single-purpose view convention (each `ViewType` has its own list+detail pair) and keeps the PR view independently navigable.

3. **Simpler list model than Issues**: The Issue list has pipeline-linked children (running/finished pipelines). The PR list won't have this — PRs don't launch pipelines. This means a simpler `PRListModel` without `issueNavItem` tree structure, just flat PR entries with filtering.

4. **PR status display**: Map PR state to user-friendly labels:
   - `open` + `Draft == true` → "Draft"
   - `open` + review requested → "Review"
   - `open` (default) → "Open"
   - `closed` + `Merged == true` → "Merged"
   - `closed` → "Closed"

5. **View position**: Insert `ViewPullRequests` after `ViewIssues` in the tab cycle. Update `cycleView()` modulo from 7 to 8.

## Risks

| Risk | Mitigation |
|------|------------|
| View count change breaks cycle modulo | Update `cycleView()` modulo, `Shift+Tab` offset, and all tests that assert on view cycling |
| GitHub API rate limiting with both Issues and PRs fetching | Both use the same `github.Client` with built-in rate limiter; lazy init means PR data only fetches when user first navigates to the view |
| PR detail missing check/review data from list endpoint | The GitHub list endpoint returns basic PR fields. For checks and reviews, a future enhancement could call `GetPullRequest()` on selection. For this issue, display what's available from the list response. |

## Testing Strategy

1. **Unit tests for `PRListModel`**: Data loading, navigation (up/down), filtering (by title, author, label, number), empty state, unfocused state, nil provider — mirror `issue_list_test.go` structure
2. **Unit tests for `PRDetailModel`**: Empty state, set PR, metadata rendering, size updates — mirror `issue_detail_test.go`
3. **View cycling tests**: Update existing `TestContentModel_TabCyclesThroughAllViews` to include `ViewPullRequests`, add `ViewPullRequests` to `TestViewType_String`
4. **Integration**: `go test ./internal/tui/...` and `go test -race ./...` pass
