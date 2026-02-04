# GitHub Integration Guide

Guide for using Wave's GitHub integration to automate issue management workflows.

## Quick Start

### 1. Set up authentication

```bash
export GITHUB_TOKEN="ghp_your_personal_access_token"
```

### 2. Run issue enhancement workflow

```bash
wave run github-issue-enhancer --input '{"repo": "owner/repo"}'
```

This will:
1. Scan all open issues in the repository
2. Analyze each issue for quality
3. Identify poorly described issues (score < 70)
4. Generate enhancement recommendations
5. Apply enhancements to issues
6. Verify changes were applied

## Setup

### Prerequisites

1. **GitHub Account** with access to target repositories
2. **GitHub Personal Access Token** with permissions:
   - `repo` (full repository access)
   - `write:discussion` (for issue comments)
3. **GitHub CLI** (`gh`) installed and authenticated

### Create GitHub Token

1. Go to GitHub Settings > Developer settings > Personal access tokens
2. Click "Generate new token (classic)"
3. Select scopes:
   - `repo` - Full control of private repositories
   - `write:discussion` - Write access to discussions
4. Generate token and save it securely

### Configure Wave

Add GitHub token to your environment:

```bash
# In ~/.bashrc or ~/.zshrc
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxx"
```

Verify setup:

```bash
# Test GitHub CLI authentication
gh auth status
```

## Issue Enhancement Workflow

**Pipeline**: `github-issue-enhancer.yaml`

**Purpose**: Automatically improve poorly described GitHub issues

**Steps**:

1. **scan-issues** (github-analyst)
   - Lists all open issues
   - Analyzes quality (0-100 score)
   - Identifies issues scoring below threshold

2. **plan-enhancements** (github-analyst)
   - Reviews poor-quality issues
   - Creates specific enhancement plans
   - Suggests improved titles and body templates
   - Recommends labels

3. **apply-enhancements** (github-enhancer)
   - Executes enhancement plans via gh CLI
   - Updates issue titles and bodies
   - Adds labels

4. **verify-enhancements** (github-analyst)
   - Confirms enhancements were applied
   - Reports on success/failures

**Usage**:

```bash
# Enhance issues in a repository
wave run github-issue-enhancer --input '{"repo": "owner/repo"}'

# With custom quality threshold
wave run github-issue-enhancer --input '{"repo": "owner/repo", "threshold": 80}'
```

## Personas

### github-analyst

**Role**: Read-only analysis of GitHub issues

**Capabilities**:
- List and retrieve issues via gh CLI
- Analyze issue quality
- Score completeness (0-100)
- Identify problems
- Generate recommendations

**Use when**: You need objective analysis without modifications

### github-enhancer

**Role**: Apply improvements to GitHub issues

**Capabilities**:
- Update issue titles via gh CLI
- Enhance descriptions
- Add labels

**Use when**: You need to apply enhancements to issues

## Best Practices

### Issue Enhancement

1. **Start with dry runs**: Use `--stop-after` to review plans before applying
2. **Be conservative**: Start with low threshold (50-60) to target worst issues
3. **Review comments**: Check the enhancement comments are respectful
4. **Preserve content**: Always maintain original author's content
5. **Batch carefully**: Process repositories in small batches to avoid rate limits

### Rate Limiting

1. **Monitor usage**: Check rate limit status with `gh api rate_limit`
2. **Add delays**: Sleep between operations when batch processing
3. **Use authenticated requests**: Always set GITHUB_TOKEN (5000/hr vs 60/hr)

### Security

1. **Protect tokens**: Never commit tokens to git
2. **Use environment variables**: Store GITHUB_TOKEN in env
3. **Minimal permissions**: Use fine-grained PATs with minimal scope
4. **Rotate regularly**: Refresh tokens every 90 days

## Troubleshooting

### Rate Limit Exceeded

**Error**:
```
Error: GitHub API rate limit exceeded
```

**Solution**:
1. Wait for rate limit reset
2. Verify GITHUB_TOKEN is set (authenticated = 5000/hr)
3. Add delays between operations

### Authentication Failed

**Error**:
```
Error: gh: authentication required
```

**Solution**:
1. Run `gh auth login`
2. Verify token is correct: `gh auth status`
3. Check token has required scopes

### Issue Not Found

**Error**:
```
Error: issue not found
```

**Solution**:
1. Verify repository owner/name are correct
2. Check you have access to the repository
3. Confirm issue number exists

## Additional Resources

- [GitHub REST API Documentation](https://docs.github.com/en/rest)
- [GitHub CLI Reference](https://cli.github.com/manual/)
