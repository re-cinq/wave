# Wave GitHub Integration

Production-ready GitHub API integration for Wave pipeline orchestrator.

## Overview

This package provides comprehensive GitHub integration for Wave, enabling automated issue enhancement, PR creation, and repository management through AI-powered workflows.

## Components

### Client (`client.go`)

Production-ready GitHub API client with:
- Full REST API v3 support
- Automatic rate limiting and retry logic
- Context-aware request handling
- Comprehensive error handling
- Token-based authentication

**Key Features:**
- Issues: List, get, update, comment
- Pull Requests: Get, create
- Repositories: Get info, create branches
- References: Create and manage git refs
- Rate limiting: Automatic backoff and retry

**Usage:**
```go
client := github.NewClient(github.ClientConfig{
    Token: os.Getenv("GITHUB_TOKEN"),
})

issue, err := client.GetIssue(ctx, "owner", "repo", 123)
```

### Analyzer (`analyzer.go`)

Issue quality analysis and enhancement recommendations:
- Quality scoring (0-100)
- Problem identification
- Enhancement suggestions
- Label recommendations

**Analysis Criteria:**
- Title quality (30 points): length, clarity, capitalization
- Description quality (40 points): completeness, structure, detail
- Metadata quality (30 points): labels, assignees, milestones

**Usage:**
```go
analyzer := github.NewAnalyzer(client)
analysis := analyzer.AnalyzeIssue(ctx, issue)

// analysis.QualityScore: 0-100
// analysis.Problems: []string of identified issues
// analysis.Recommendations: []string of suggestions
```

### Types (`types.go`)

Comprehensive type definitions for:
- Issue, IssueUpdate, IssueComment
- PullRequest, CreatePullRequestRequest
- Repository, Reference, GitRef
- User, Label, Milestone
- Rate limit status
- API errors

### Rate Limiter (`ratelimit.go`)

Thread-safe rate limit management:
- Tracks GitHub API rate limits
- Automatic backoff when limits reached
- Context-aware waiting
- Real-time status updates from response headers

## Adapter Integration

The GitHub adapter (`internal/adapter/github.go`) wraps the client for Wave pipeline integration:

**Supported Operations:**
- `list_issues`: List repository issues
- `analyze_issues`: Find poor quality issues
- `get_issue`: Retrieve single issue
- `update_issue`: Update issue fields
- `create_pr`: Create pull request
- `get_repo`: Get repository info
- `create_branch`: Create new branch

**Usage in Pipelines:**

```yaml
exec:
  type: prompt
  source: |
    Use GitHub CLI to list and analyze issues.
    Save results to artifact.json.
```

## Personas

Three specialized personas for GitHub workflows:

### github-analyst
- Read-only analysis of issues
- Quality scoring and problem identification
- Recommendation generation
- Temperature: 0.2 (precise, analytical)

### github-enhancer
- Issue enhancement execution
- Title and body updates
- Label management
- Temperature: 0.3 (creative but controlled)

### github-pr-creator
- Pull request creation
- PR description generation
- Branch and reviewer management
- Temperature: 0.3 (creative but structured)

## Pipelines

### github-issue-enhancement.yaml

Complete workflow for finding and enhancing poor quality issues:

1. **Analyze**: Scan repository for issues, score quality
2. **Generate Enhancements**: Create specific improvement plans
3. **Enhance Issues**: Execute enhancements via GitHub API
4. **Verify**: Confirm enhancements were applied successfully

**Usage:**
```bash
wave run github-issue-enhancement --input "owner/repo"
```

### github-pr-creation.yaml

Workflow for creating well-formatted pull requests:

1. **Analyze Changes**: Review git diff and commits
2. **Draft PR**: Generate title and structured body
3. **Create PR**: Submit PR via GitHub API
4. **Verify**: Confirm PR was created correctly

**Usage:**
```bash
wave run github-pr-creation --input "feature-branch"
```

## Contracts

JSON schemas for pipeline validation:

- `github-issue-analysis.schema.json`: Issue analysis output
- `github-enhancement-plan.schema.json`: Enhancement recommendations
- `github-enhancement-results.schema.json`: Enhancement execution results
- `github-verification-report.schema.json`: Verification report
- `github-change-analysis.schema.json`: Git change analysis
- `github-pr-draft.schema.json`: PR draft content
- `github-pr-info.schema.json`: Created PR information

## Authentication

Set GitHub token via environment variable:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

Or pass directly to client:

```go
client := github.NewClient(github.ClientConfig{
    Token: "ghp_your_token_here",
})
```

**Token Permissions Required:**
- `repo` (full repository access)
- `write:discussion` (for comments)

## Rate Limiting

GitHub API rate limits:
- Authenticated: 5,000 requests/hour
- Unauthenticated: 60 requests/hour

The client automatically:
- Tracks remaining requests
- Waits for limit reset when exhausted
- Updates limits from response headers
- Retries with exponential backoff

**Check current status:**
```go
status, err := client.GetRateLimit(ctx)
// status.Remaining: requests left
// status.Reset: unix timestamp of reset time
```

## Error Handling

Comprehensive error types:

**APIError**: HTTP errors from GitHub API
- Status code
- Error message
- Validation errors (if applicable)
- Documentation URL

**RateLimitError**: Rate limit exceeded
- Reset time
- Descriptive message

**Usage:**
```go
_, err := client.GetIssue(ctx, owner, repo, number)
if err != nil {
    if apiErr, ok := err.(*github.APIError); ok {
        // Handle API error
        log.Printf("GitHub API error: %d - %s", apiErr.StatusCode, apiErr.Message)
    }
    if rlErr, ok := err.(*github.RateLimitError); ok {
        // Handle rate limit
        log.Printf("Rate limited until: %s", rlErr.ResetTime)
    }
}
```

## Testing

Comprehensive test coverage:

```bash
# Run all GitHub package tests
go test ./internal/github/...

# Run with race detection
go test -race ./internal/github/...

# Run with coverage
go test -cover ./internal/github/...

# Run adapter tests
go test ./internal/adapter/github_test.go
```

**Test Coverage:**
- Client operations (list, get, update, create)
- Rate limiting behavior
- Error handling
- Authentication
- Issue analysis
- Quality scoring
- Enhancement suggestions
- Adapter operation parsing

## Production Considerations

### Security
- Never commit GitHub tokens
- Use environment variables or secret management
- Rotate tokens regularly
- Use fine-grained PATs when possible
- Audit token permissions

### Performance
- Batch operations when possible
- Use pagination for large result sets
- Monitor rate limit usage
- Cache repository metadata
- Use conditional requests (ETags)

### Reliability
- Handle rate limits gracefully
- Retry transient failures
- Validate inputs before API calls
- Log all API errors
- Monitor API status (status.github.com)

### Best Practices
- Use descriptive issue titles and PR descriptions
- Always preserve original content when enhancing
- Be respectful in automated comments
- Link related issues and PRs
- Follow repository conventions
- Test changes before applying

## Examples

### List Issues
```go
issues, err := client.ListIssues(ctx, "owner", "repo", github.ListIssuesOptions{
    State:   "open",
    Labels:  []string{"bug"},
    PerPage: 50,
})
```

### Analyze Issue Quality
```go
analyzer := github.NewAnalyzer(client)
analysis := analyzer.AnalyzeIssue(ctx, issue)

if analysis.QualityScore < 70 {
    analyzer.GenerateEnhancementSuggestions(issue, analysis)
    // Use analysis.SuggestedTitle, SuggestedBody, SuggestedLabels
}
```

### Update Issue
```go
newTitle := "Improved: " + issue.Title
update := github.IssueUpdate{
    Title: &newTitle,
}
updatedIssue, err := client.UpdateIssue(ctx, "owner", "repo", issueNum, update)
```

### Create Pull Request
```go
pr, err := client.CreatePullRequest(ctx, "owner", "repo", github.CreatePullRequestRequest{
    Title: "Add GitHub integration",
    Body:  "Comprehensive GitHub API integration for Wave",
    Head:  "feature/github-integration",
    Base:  "main",
})
```

### Find Poor Quality Issues
```go
analyzer := github.NewAnalyzer(client)
poorIssues, err := analyzer.FindPoorQualityIssues(ctx, "owner", "repo", 70)
// Returns issues with quality score < 70
```

## Troubleshooting

### Rate Limit Exceeded
```
Error: GitHub API rate limit exceeded (resets at 2024-01-01T12:00:00Z)
```
**Solution**: Wait for reset or use authenticated requests (higher limit)

### Authentication Failed
```
Error: GitHub API error (status 401): Bad credentials
```
**Solution**: Check token is valid and has required permissions

### Issue Not Found
```
Error: GitHub API error (status 404): Not Found
```
**Solution**: Verify owner/repo/number are correct and accessible

### Validation Error
```
Error: Validation Failed - title: can't be blank
```
**Solution**: Ensure all required fields are provided

## Future Enhancements

Potential additions:
- [ ] GitHub Actions integration
- [ ] Webhook support for real-time updates
- [ ] Project board management
- [ ] Advanced search with GraphQL API
- [ ] Bulk issue operations
- [ ] PR review automation
- [ ] Issue templates and forms
- [ ] GitHub Apps support
- [ ] Organization-level operations
- [ ] Team management

## Contributing

When contributing to the GitHub integration:

1. Maintain test coverage above 80%
2. Add tests for all new features
3. Update documentation
4. Follow Go conventions
5. Handle errors comprehensively
6. Respect rate limits in tests
7. Use mocks for unit tests
8. Test against real API for integration tests (when safe)

## License

Part of the Wave project - see main LICENSE file.
