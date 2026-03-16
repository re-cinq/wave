# Implementation Plan: WebUI Feature Parity

## Objective

Add missing views to the WebUI so it achieves feature parity with the TUI: GitHub Issues browser, Pull Requests view, Health checks display, and an issue-to-pipeline launcher control.

## Approach

Mirror the TUI's data provider pattern in the webui package. The TUI uses `IssueDataProvider`, `PRDataProvider`, and `HealthDataProvider` — the webui handlers will call the same `internal/github` client and reuse the TUI's health check logic (via its `HealthDataProvider` interface) to serve data through new HTML pages and JSON API endpoints.

The webui `Server` struct will be extended with a `*github.Client` and a repo slug resolved from git remotes (same approach as TUI). Health checks will instantiate `tui.DefaultHealthDataProvider` and run checks on demand.

## File Mapping

### New Files
| Path | Purpose |
|------|---------|
| `internal/webui/handlers_issues.go` | Issues page + API handlers |
| `internal/webui/handlers_prs.go` | PRs page + API handlers |
| `internal/webui/handlers_health.go` | Health page + API handlers |
| `internal/webui/templates/issues.html` | Issues browser template |
| `internal/webui/templates/prs.html` | PRs view template |
| `internal/webui/templates/health.html` | Health checks template |

### Modified Files
| Path | Change |
|------|--------|
| `internal/webui/server.go` | Add `githubClient`, `repoSlug` fields; init in `NewServer` |
| `internal/webui/routes.go` | Register new routes for issues, PRs, health |
| `internal/webui/types.go` | Add response types for issues, PRs, health |
| `internal/webui/embed.go` | Ensure new templates are embedded |
| `internal/webui/templates/layout.html` | Add nav links for Issues, PRs, Health |

## Architecture Decisions

1. **GitHub client in webui server**: The server will create a `github.Client` using the same token resolution as TUI (`GH_TOKEN` / `GITHUB_TOKEN` / `gh auth token`). This avoids adding a new auth mechanism.

2. **Reuse TUI health provider**: Import and call `tui.DefaultHealthDataProvider` directly rather than duplicating the health check logic. The health endpoint runs checks synchronously and returns JSON results.

3. **No guided flow in v1**: The guided flow (Health → Proposals → Fleet) is a TUI-specific interaction pattern that doesn't translate well to a web dashboard. The web equivalent is simply having all three views accessible via navigation. This can be added later as a wizard-style UI if desired.

4. **Issue-to-pipeline launcher**: The issue detail API response will include available pipelines. The frontend will provide a "Launch Pipeline" button that calls the existing `POST /api/pipelines/{name}/start` endpoint with the issue URL as input.

5. **Server-side token**: The GitHub token is resolved server-side at startup. No OAuth flow needed — this matches TUI behavior and keeps the implementation simple.

## Risks

| Risk | Mitigation |
|------|------------|
| No GitHub token available | Health endpoint works without it; issues/PRs return empty list with informative message |
| TUI health provider import creates circular dependency | `tui` package doesn't import `webui`, so this is safe |
| Rate limiting on GitHub API | Use existing `github.Client` rate limiter; cache responses with short TTL if needed |
| Repo slug detection fails | Fall back gracefully — issues/PRs views show "No repository configured" |

## Testing Strategy

1. **Unit tests**: `handlers_issues_test.go`, `handlers_prs_test.go`, `handlers_health_test.go` — test handler responses with mock data
2. **Integration**: Existing `handlers_test.go` pattern — verify routes are registered and return correct status codes
3. **Template rendering**: Test that templates render without errors with both populated and empty data
