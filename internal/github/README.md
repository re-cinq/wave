# Wave GitHub Integration

GitHub API integration for Wave pipeline orchestrator.

## Overview

This package provides GitHub integration for Wave, enabling automated issue enhancement through AI-powered workflows using the GitHub CLI (`gh`).

## Components

### Client (`client.go`)

GitHub API client with:
- REST API v3 support
- Rate limiting and retry logic
- Context-aware request handling
- Token-based authentication

### Analyzer (`analyzer.go`)

Issue quality analysis:
- Quality scoring (0-100)
- Problem identification
- Enhancement suggestions
- Label recommendations

### Types (`types.go`)

Type definitions for:
- Issue, IssueUpdate, IssueComment
- PullRequest, CreatePullRequestRequest
- Repository, Reference
- Rate limit status
- API errors

### Rate Limiter (`ratelimit.go`)

Thread-safe rate limit management:
- Tracks GitHub API rate limits
- Automatic backoff when limits reached
- Context-aware waiting

## Personas

Two specialized personas for GitHub workflows:

### github-analyst
- Read-only analysis of issues via gh CLI
- Quality scoring and problem identification
- Recommendation generation

### github-enhancer
- Issue enhancement execution via gh CLI
- Title and body updates
- Label management

## Pipeline

### github-issue-enhancer.yaml

Workflow for finding and enhancing poor quality issues:

1. **scan-issues**: Scan repository for issues, score quality
2. **plan-enhancements**: Create specific improvement plans
3. **apply-enhancements**: Execute enhancements via gh CLI
4. **verify-enhancements**: Confirm enhancements were applied

**Usage:**
```bash
wave run github-issue-enhancer --input '{"repo": "owner/repo"}'
```

## Contracts

JSON schemas for pipeline validation:
- `github-issue-analysis.schema.json`: Issue analysis output
- `github-enhancement-plan.schema.json`: Enhancement recommendations
- `github-enhancement-results.schema.json`: Enhancement execution results
- `github-verification-report.schema.json`: Verification report

## Authentication

The pipeline uses GitHub CLI authentication:

```bash
gh auth login
gh auth status
```

## Rate Limiting

GitHub API rate limits:
- Authenticated: 5,000 requests/hour
- Unauthenticated: 60 requests/hour

Check status: `gh api rate_limit`

## Testing

```bash
# Run all GitHub package tests
go test ./internal/github/...

# Run with race detection
go test -race ./internal/github/...
```

## License

Part of the Wave project - see main LICENSE file.
