# GitHub Integration Guide

Complete guide for using Wave's GitHub integration to automate issue management and PR workflows.

## Table of Contents

- [Quick Start](#quick-start)
- [Setup](#setup)
- [Workflows](#workflows)
- [Personas](#personas)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Quick Start

### 1. Set up authentication

```bash
export GITHUB_TOKEN="ghp_your_personal_access_token"
```

### 2. Run issue enhancement workflow

```bash
wave run github-issue-enhancement --input "owner/repo"
```

This will:
1. Scan all open issues in the repository
2. Analyze each issue for quality
3. Identify poorly described issues (score < 70)
4. Generate enhancement recommendations
5. Apply enhancements to issues
6. Verify changes were applied

### 3. Create a pull request

```bash
# Make your code changes and commit them
git checkout -b feature/my-feature
# ... make changes ...
git add .
git commit -m "Add new feature"

# Run PR creation workflow
wave run github-pr-creation --input "feature/my-feature"
```

## Setup

### Prerequisites

1. **GitHub Account** with access to target repositories
2. **GitHub Personal Access Token** with permissions:
   - `repo` (full repository access)
   - `write:discussion` (for issue comments)
3. **GitHub CLI** (`gh`) installed and authenticated (optional but recommended)

### Create GitHub Token

1. Go to GitHub Settings > Developer settings > Personal access tokens
2. Click "Generate new token (classic)"
3. Select scopes:
   - ✅ `repo` - Full control of private repositories
   - ✅ `write:discussion` - Write access to discussions
4. Generate token and save it securely

### Configure Wave

Add GitHub token to your environment:

```bash
# In ~/.bashrc or ~/.zshrc
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxx"

# Or for a single session
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxx"
```

Verify setup:

```bash
# Test GitHub CLI authentication
gh auth status

# Test Wave GitHub integration
wave run github-issue-enhancement --help
```

## Workflows

Wave provides two production-ready GitHub workflows:

### Issue Enhancement Workflow

**Pipeline**: `github-issue-enhancement.yaml`

**Purpose**: Automatically improve poorly described GitHub issues

**Steps**:

1. **Analyze** (github-analyst)
   - Lists all open issues
   - Analyzes quality (0-100 score)
   - Identifies issues scoring below threshold
   - Outputs structured analysis

2. **Generate Enhancements** (github-analyst)
   - Reviews poor-quality issues
   - Creates specific enhancement plans
   - Suggests improved titles
   - Generates description templates
   - Recommends labels

3. **Enhance Issues** (github-enhancer)
   - Executes enhancement plans
   - Updates issue titles
   - Enhances descriptions
   - Adds labels
   - Posts explanatory comments

4. **Verify** (github-analyst)
   - Confirms enhancements were applied
   - Measures quality improvement
   - Reports on success/failures

**Usage**:

```bash
# Enhance issues in a repository
wave run github-issue-enhancement --input "owner/repo"

# With custom quality threshold
wave run github-issue-enhancement --input "owner/repo" \
  --var "threshold=80"

# Dry run (analyze only, don't modify)
wave run github-issue-enhancement --input "owner/repo" \
  --stop-after "generate-enhancements"
```

**Output**:

```json
{
  "total_enhanced": 5,
  "successful_enhancements": 4,
  "failed_enhancements": 1,
  "quality_improvement": {
    "average_score_before": 45,
    "average_score_after": 82,
    "improvement_percentage": 82.2
  }
}
```

### PR Creation Workflow

**Pipeline**: `github-pr-creation.yaml`

**Purpose**: Create well-formatted pull requests with structured descriptions

**Steps**:

1. **Analyze Changes** (navigator)
   - Reviews git diff
   - Analyzes commit messages
   - Identifies changed files
   - Extracts related issues
   - Summarizes changes

2. **Draft PR** (github-pr-creator)
   - Generates clear title
   - Creates structured body
   - Adds sections (Summary, Changes, Test Plan)
   - Links related issues
   - Suggests labels and reviewers

3. **Create PR** (github-pr-creator)
   - Submits PR via GitHub API
   - Applies labels
   - Assigns reviewers
   - Returns PR URL

4. **Verify** (navigator)
   - Confirms PR was created
   - Validates metadata
   - Checks formatting

**Usage**:

```bash
# Create PR from current branch
wave run github-pr-creation

# Specify branch explicitly
wave run github-pr-creation --input "feature/my-feature"

# Create draft PR
wave run github-pr-creation --input "feature/my-feature" \
  --var "draft=true"

# Add specific reviewers
wave run github-pr-creation --input "feature/my-feature" \
  --var "reviewers=user1,user2"
```

**Output**:

```json
{
  "pr_number": 123,
  "pr_url": "https://github.com/owner/repo/pull/123",
  "title": "Add GitHub integration for Wave",
  "state": "open",
  "draft": false,
  "success": true
}
```

## Personas

### github-analyst

**Role**: Read-only analysis of GitHub issues

**Capabilities**:
- List and retrieve issues
- Analyze issue quality
- Score completeness (0-100)
- Identify problems
- Generate recommendations
- Suggest enhancements

**Permissions**:
- ✅ Read files
- ✅ GitHub CLI (read operations)
- ❌ Write files (except artifact.json)
- ❌ Edit files

**Temperature**: 0.2 (precise, analytical)

**Use when**: You need objective analysis without modifications

### github-enhancer

**Role**: Apply improvements to GitHub issues

**Capabilities**:
- Update issue titles
- Enhance descriptions
- Add labels
- Post comments
- Preserve original content

**Permissions**:
- ✅ Read files
- ✅ Write artifact.json
- ✅ GitHub CLI (write operations)
- ❌ Destructive commands (rm, force push)

**Temperature**: 0.3 (creative but controlled)

**Use when**: You need to apply enhancements to issues

### github-pr-creator

**Role**: Create and manage pull requests

**Capabilities**:
- Analyze git changes
- Generate PR titles and bodies
- Create pull requests
- Add labels and reviewers
- Link issues

**Permissions**:
- ✅ Read files
- ✅ Write artifact.json
- ✅ Git commands
- ✅ GitHub CLI
- ❌ Force push

**Temperature**: 0.3 (creative but structured)

**Use when**: You need to create PRs with proper formatting

## Examples

### Example 1: Enhance All Poor Issues

```bash
# Find and enhance all issues with quality score < 70
wave run github-issue-enhancement --input "myorg/myrepo"
```

**What happens**:
1. Scans all open issues
2. Scores each based on title, body, labels
3. Identifies 5 issues with scores: 45, 52, 38, 61, 55
4. Generates enhancement plans for each
5. Updates titles to be more descriptive
6. Adds structured templates to descriptions
7. Adds appropriate labels (bug, enhancement, etc.)
8. Posts friendly comment explaining improvements

**Before**:
- Title: "bug"
- Body: "it doesnt work"
- Labels: none

**After**:
- Title: "Bug: Application Feature Not Working As Expected"
- Body: Enhanced with template including Description, Steps to Reproduce, etc.
- Labels: bug, needs-info

### Example 2: Create Feature PR

```bash
# You're on feature/github-integration branch
wave run github-pr-creation
```

**Generated PR**:

**Title**: Add GitHub Integration for Wave Pipeline Orchestrator

**Body**:
```markdown
## Summary
Implements comprehensive GitHub API integration enabling automated issue
enhancement and PR creation workflows through AI-powered personas.

## Changes
- Add GitHub API client with rate limiting and retry logic
- Implement issue analyzer with quality scoring
- Create github-analyst, github-enhancer, github-pr-creator personas
- Add issue enhancement and PR creation pipelines
- Include JSON schema contracts for validation

## Motivation
Enables Wave to automate GitHub workflows, improving issue quality and
standardizing PR formats across teams. Addresses #123.

## Test Plan
- Run unit tests: `go test ./internal/github/...`
- Test issue analysis with real repository
- Verify PR creation workflow
- Check rate limiting behavior

## Checklist
- [x] Tests added and passing
- [x] Documentation updated
- [x] No breaking changes
- [x] Self-reviewed code
```

### Example 3: Batch Process Multiple Repositories

```bash
# Create a script to process multiple repos
#!/bin/bash

repos=(
  "org/repo1"
  "org/repo2"
  "org/repo3"
)

for repo in "${repos[@]}"; do
  echo "Processing $repo..."
  wave run github-issue-enhancement --input "$repo"
  sleep 5  # Avoid rate limiting
done
```

### Example 4: Custom Quality Threshold

```bash
# Only enhance issues with very poor quality (score < 50)
wave run github-issue-enhancement \
  --input "owner/repo" \
  --var "threshold=50"

# Or be more aggressive (score < 80)
wave run github-issue-enhancement \
  --input "owner/repo" \
  --var "threshold=80"
```

### Example 5: Dry Run Analysis

```bash
# Analyze without making changes
wave run github-issue-enhancement \
  --input "owner/repo" \
  --stop-after "generate-enhancements"

# Review the enhancement plan in the artifacts
cat .wave/workspaces/*/artifact.json | jq .
```

## Best Practices

### Issue Enhancement

1. **Start with dry runs**: Use `--stop-after` to review plans before applying
2. **Be conservative**: Start with low threshold (50-60) to target worst issues
3. **Review comments**: Check the enhancement comments are respectful
4. **Preserve content**: Always maintain original author's content
5. **Batch carefully**: Process repositories in small batches to avoid rate limits

### PR Creation

1. **Descriptive commits**: Write clear commit messages (used in PR body)
2. **Link issues**: Reference issues in commits (#123) for automatic linking
3. **Review before merge**: PRs are created but not merged automatically
4. **Use draft mode**: Create draft PRs for work-in-progress
5. **Assign reviewers**: Specify reviewers via variables for faster review

### Rate Limiting

1. **Monitor usage**: Check rate limit status with `gh api rate_limit`
2. **Add delays**: Sleep between operations when batch processing
3. **Use authenticated requests**: Always set GITHUB_TOKEN (5000/hr vs 60/hr)
4. **Handle errors gracefully**: Wave retries with exponential backoff
5. **Cache when possible**: Avoid unnecessary API calls

### Security

1. **Protect tokens**: Never commit tokens to git
2. **Use environment variables**: Store GITHUB_TOKEN in env
3. **Minimal permissions**: Use fine-grained PATs with minimal scope
4. **Rotate regularly**: Refresh tokens every 90 days
5. **Audit access**: Review token usage in GitHub settings

## Troubleshooting

### Rate Limit Exceeded

**Error**:
```
Error: GitHub API rate limit exceeded (resets at 2024-01-01T12:00:00Z)
```

**Solution**:
1. Wait for rate limit reset (shown in error)
2. Verify GITHUB_TOKEN is set (authenticated = 5000/hr)
3. Add delays between operations
4. Use `--stop-after` to process in stages

### Authentication Failed

**Error**:
```
Error: GitHub API error (status 401): Bad credentials
```

**Solution**:
1. Verify token is correct: `echo $GITHUB_TOKEN`
2. Check token hasn't expired
3. Confirm token has required scopes (repo, write:discussion)
4. Regenerate token if needed

### Issue Not Found

**Error**:
```
Error: GitHub API error (status 404): Not Found
```

**Solution**:
1. Verify repository owner/name are correct
2. Check you have access to the repository
3. Confirm issue number exists
4. Ensure token has repo access

### Validation Failed

**Error**:
```
Error: Contract validation failed: title: can't be blank
```

**Solution**:
1. Check the enhancement plan in artifacts
2. Verify all required fields are populated
3. Review persona output for errors
4. Check JSON schema requirements

### Permission Denied

**Error**:
```
Error: Resource not accessible by integration
```

**Solution**:
1. Check token has write permissions
2. Verify you're a collaborator on the repo
3. Check branch protection rules
4. Confirm token scopes include required permissions

### Pipeline Fails Mid-Execution

**Solution**:
1. Check `.wave/traces/` for detailed logs
2. Review last successful step's artifacts
3. Use `wave resume <run-id>` to continue
4. Fix the issue and retry from failed step

## Advanced Usage

### Custom Personas

Create specialized personas for your team:

```yaml
personas:
  github-strict-reviewer:
    adapter: claude
    description: Strict code review enforcement
    permissions:
      allowed_tools:
        - Read
        - Bash(gh *)
    system_prompt_file: .wave/personas/github-strict-reviewer.md
    temperature: 0.1
```

### Custom Pipelines

Build workflows for your needs:

```yaml
kind: WavePipeline
metadata:
  name: github-stale-issue-cleanup
  description: Close stale issues after analyzing activity

steps:
  - id: find-stale
    persona: github-analyst
    exec:
      type: prompt
      source: |
        Find issues with no activity in 90 days.
        Analyze if they're still relevant.
        Output list of issues to close.
```

### Integration with CI/CD

Run Wave workflows in GitHub Actions:

```yaml
name: Enhance Issues
on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly

jobs:
  enhance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Wave
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          wave run github-issue-enhancement \
            --input "${{ github.repository }}"
```

## Additional Resources

- [GitHub REST API Documentation](https://docs.github.com/en/rest)
- [Wave Pipeline Guide](./pipeline-guide.md)
- [Persona Development](./persona-development.md)
- [Contract Schemas](./contracts.md)
- [GitHub CLI Reference](https://cli.github.com/manual/)

## Support

For issues or questions:
- Check existing GitHub issues
- Review logs in `.wave/traces/`
- Test with simpler cases first
- Verify credentials and permissions
