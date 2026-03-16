# Tasks

## Phase 1: Server Infrastructure
- [X] Task 1.1: Add `githubClient` and `repoSlug` fields to `Server` struct, initialize in `NewServer` using token resolution from env/gh-cli and git remote detection
- [X] Task 1.2: Add response types to `types.go` — `IssueSummary`, `IssueListResponse`, `PRSummary`, `PRListResponse`, `HealthCheckResult`, `HealthListResponse`

## Phase 2: Core Views
- [X] Task 2.1: Create `handlers_issues.go` with `handleIssuesPage` (HTML) and `handleAPIIssues` (JSON) using `github.Client.ListIssues` [P]
- [X] Task 2.2: Create `handlers_prs.go` with `handlePRsPage` (HTML) and `handleAPIPRs` (JSON) using `github.Client.ListPullRequests` [P]
- [X] Task 2.3: Create `handlers_health.go` with `handleHealthPage` (HTML) and `handleAPIHealth` (JSON) using `tui.DefaultHealthDataProvider` [P]

## Phase 3: Templates
- [X] Task 3.1: Create `templates/issues.html` — issue list with number, title, labels, author, date; detail pane with body and "Launch Pipeline" button [P]
- [X] Task 3.2: Create `templates/prs.html` — PR list with number, title, state badges (open/draft/merged), branch info, additions/deletions [P]
- [X] Task 3.3: Create `templates/health.html` — health check list with status icons (ok/warn/error), message, expandable details, re-run button [P]

## Phase 4: Routing and Navigation
- [X] Task 4.1: Register new routes in `routes.go` — `GET /issues`, `GET /prs`, `GET /health`, `GET /api/issues`, `GET /api/prs`, `GET /api/health`
- [X] Task 4.2: Update `templates/layout.html` nav to include Issues, PRs, Health links
- [X] Task 4.3: Ensure new templates are picked up by `parseTemplates` in `embed.go`

## Phase 5: Issue-to-Pipeline Launcher
- [X] Task 5.1: Add `handleAPIStartFromIssue` endpoint that accepts issue URL and pipeline name, calls existing `handleStartPipeline` logic with issue URL as input
- [X] Task 5.2: Add launch UI in issues template — pipeline selector dropdown and start button that POSTs to the new endpoint

## Phase 6: Testing
- [X] Task 6.1: Write unit tests for issues handler with mock GitHub client [P]
- [X] Task 6.2: Write unit tests for PRs handler with mock GitHub client [P]
- [X] Task 6.3: Write unit tests for health handler with mock health provider [P]
- [X] Task 6.4: Test template rendering with empty and populated data [P]

## Phase 7: Polish
- [X] Task 7.1: Handle missing GitHub token gracefully — show informative empty states
- [X] Task 7.2: Run `go test ./...` and `golangci-lint run ./...` — fix all failures
