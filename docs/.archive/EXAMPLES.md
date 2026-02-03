# GitHub Pipeline Examples

Real-world examples for running Wave's GitHub integration pipelines.

## Prerequisites

```bash
# Ensure GitHub CLI is authenticated
gh auth status

# Test access to your repository
gh repo view owner/repository
```

## Example 1: Clean Up Issue Backlog

**Scenario**: Your project has 50 open issues with poor formatting, missing details, and no labels.

### Step 1: Analyze Current State (Read-Only)

```bash
# Set threshold to 100 so nothing gets enhanced yet
wave run github-issue-enhancer --input '{
  "repo": "myorg/myproject",
  "threshold": 100
}'

# Check results
cat .wave/workspaces/github-issue-enhancer/scan-issues/artifact.json | jq '{
  total: .total_issues,
  poor_quality_count: (.poor_quality_issues | length),
  quality_threshold: .quality_threshold
}'
```

**Expected output**:
```json
{
  "total": 50,
  "poor_quality_count": 32,
  "quality_threshold": 100
}
```

### Step 2: Review Enhancement Plan

```bash
# Lower threshold to see what would be enhanced
wave run github-issue-enhancer --input '{
  "repo": "myorg/myproject",
  "threshold": 70
}'

# Review the plan (but stop before applying)
cat .wave/workspaces/github-issue-enhancer/plan-enhancements/artifact.json | jq '.issues_to_enhance[0]'
```

**Expected output**:
```json
{
  "issue_number": 42,
  "current_title": "bug",
  "suggested_title": "Authentication Fails with OAuth Provider",
  "current_body": "doesnt work",
  "body_template": "## Description\ndoesnt work\n\n## Steps to Reproduce\n...",
  "suggested_labels": ["bug", "authentication", "needs-reproduction"],
  "enhancements": ["Expand vague title", "Add structured template"],
  "priority": "high"
}
```

### Step 3: Apply Enhancements

If the plan looks good, the pipeline continues and applies the enhancements. Check the final report:

```bash
cat .wave/workspaces/github-issue-enhancer/verify-enhancements/artifact.json | jq '{
  total_enhanced: .total_enhanced,
  all_successful: .all_successful,
  summary: .summary
}'
```

**Result**: 32 issues now have clear titles, structured descriptions, and appropriate labels.

---

## Example 2: Implement Feature from Issue

**Scenario**: Issue #123 requests adding OAuth login support. You want to automate the implementation.

### Step 1: Review Issue

```bash
# First check the issue details
gh issue view 123 --repo myorg/myproject
```

### Step 2: Analyze Requirements (Dry Run)

```bash
# Run just the analysis step to understand what will be implemented
wave run github-feature-implementation --input '{
  "repo": "myorg/myproject",
  "issue": 123
}'

# Review the requirements analysis
cat .wave/workspaces/github-feature-implementation/analyze-requirement/artifact.json | jq '{
  requirements: .requirements,
  affected_files: .affected_files,
  approach: .technical_approach,
  complexity: .implementation_complexity
}'
```

**Expected output**:
```json
{
  "requirements": [
    "Add OAuth2 authentication provider",
    "Support Google and GitHub OAuth",
    "Implement token refresh logic"
  ],
  "affected_files": [
    "internal/auth/oauth.go",
    "internal/auth/oauth_test.go"
  ],
  "approach": "Create new oauth package with provider interfaces",
  "complexity": "medium"
}
```

### Step 3: Full Implementation

If the analysis looks good, let the pipeline continue to implement, test, and create the PR:

```bash
# Check the created PR
cat .wave/workspaces/github-feature-implementation/create-pull-request/artifact.json | jq '{
  pr_number: .pr_number,
  pr_url: .pr_url,
  title: .title
}'
```

**Result**:
- New branch: `feature/issue-123-add-oauth-support`
- Implementation in `internal/auth/oauth.go`
- Tests in `internal/auth/oauth_test.go`
- PR created: https://github.com/myorg/myproject/pull/456

---

## Example 3: Discover Related Issues

**Scenario**: Your project has 100+ issues but no cross-references. You want to link related issues.

### Step 1: Analyze Relationships Only

```bash
# Run analysis to see what relationships exist
wave run github-issue-cross-linker --input '{
  "repo": "myorg/myproject",
  "state": "open"
}'

# Review detected relationships
cat .wave/workspaces/github-issue-cross-linker/analyze-relationships/artifact.json | jq '{
  total_relationships: .total_relationships,
  high_priority: .high_priority_count,
  sample_relationship: .relationships[0]
}'
```

**Expected output**:
```json
{
  "total_relationships": 25,
  "high_priority": 8,
  "sample_relationship": {
    "issue_a": 42,
    "issue_b": 67,
    "similarity_score": 0.85,
    "relationship_type": "related",
    "rationale": "Both discuss OAuth authentication implementation",
    "shared_keywords": ["oauth", "authentication", "token"],
    "shared_labels": ["authentication"],
    "priority": "high"
  }
}
```

### Step 2: Review Cross-Links

The pipeline automatically creates the links. Verify:

```bash
cat .wave/workspaces/github-issue-cross-linker/verify-cross-links/artifact.json | jq '{
  total_verified: .total_verified,
  all_successful: .all_successful,
  summary: .summary
}'

# Check an actual issue to see the link
gh issue view 42 --repo myorg/myproject --comments
```

**Result**:
- 25 bidirectional cross-references created
- Issue #42 now has comment: "Related issue: #67 - Both discuss OAuth authentication implementation"
- Issue #67 has reciprocal link to #42

---

## Example 4: Automated Code Review

**Scenario**: PR #456 was just created. You want comprehensive automated review before human review.

### Step 1: Generate Review (Don't Post)

```bash
# Generate review without posting to GitHub
wave run github-pr-review-automation --input '{
  "repo": "myorg/myproject",
  "pr": 456,
  "post_review": false
}'

# Review the findings
cat .wave/workspaces/github-pr-review-automation/synthesize-review/artifact.json | jq '{
  assessment: .overall_assessment,
  security_issues: .security_issues,
  quality_issues: .quality_issues,
  must_fix: .must_fix,
  should_fix: .should_fix,
  positive_notes: .positive_notes
}'
```

**Expected output**:
```json
{
  "assessment": "REQUEST_CHANGES",
  "security_issues": 2,
  "quality_issues": 5,
  "must_fix": [
    "Remove OAuth token from error logging (security risk)",
    "Add test coverage for new authentication functions"
  ],
  "should_fix": [
    "Add error context to all error returns",
    "Document rate limiting behavior",
    "Consider adding integration tests"
  ],
  "positive_notes": [
    "Clean interface design",
    "Good use of structured logging",
    "Comprehensive documentation",
    "Follows project conventions"
  ]
}
```

### Step 2: Review Detailed Findings

```bash
# View specific security issues
cat .wave/workspaces/github-pr-review-automation/security-review/artifact.json | jq '.security_findings'

# View quality issues
cat .wave/workspaces/github-pr-review-automation/quality-review/artifact.json | jq '.quality_findings'
```

### Step 3: Post Review (If Satisfied)

```bash
# If the review looks good, post it to GitHub
wave run github-pr-review-automation --input '{
  "repo": "myorg/myproject",
  "pr": 456,
  "post_review": true
}'

# Verify it was posted
gh pr view 456 --repo myorg/myproject --comments
```

**Result**:
- Comprehensive review posted to PR #456
- Review type: REQUEST_CHANGES
- 2 security issues highlighted
- 5 quality improvements suggested
- 4 positive notes included

---

## Example 5: End-to-End Development Workflow

**Scenario**: Complete workflow from issue creation to reviewed PR.

### Step 1: Create and Enhance Issue

```bash
# Create a new issue (or use existing)
ISSUE=$(gh issue create --repo myorg/myproject \
  --title "login broken" \
  --body "cant authenticate" \
  --json number -q .number)

echo "Created issue #$ISSUE"

# Enhance it
wave run github-issue-enhancer --input "{
  \"repo\": \"myorg/myproject\",
  \"threshold\": 70
}"

# Check enhancement
gh issue view $ISSUE --repo myorg/myproject
```

**Result**: Issue now has clear title: "Authentication Login Process Fails", structured template, labels.

### Step 2: Implement Solution

```bash
# Implement the feature
wave run github-feature-implementation --input "{
  \"repo\": \"myorg/myproject\",
  \"issue\": $ISSUE
}"

# Get the PR number
PR=$(cat .wave/workspaces/github-feature-implementation/create-pull-request/artifact.json | jq -r .pr_number)
echo "Created PR #$PR"
```

**Result**: PR #$PR created with implementation and tests.

### Step 3: Automated Review

```bash
# Review the PR
wave run github-pr-review-automation --input "{
  \"repo\": \"myorg/myproject\",
  \"pr\": $PR,
  \"post_review\": true
}"
```

**Result**: Comprehensive review posted on PR.

### Step 4: Cross-Link Related Issues

```bash
# Link related issues
wave run github-issue-cross-linker --input '{
  "repo": "myorg/myproject"
}'
```

**Result**: Issue #$ISSUE now linked to other authentication-related issues.

### Complete Workflow Summary

1. Issue #$ISSUE enhanced with clear description and labels
2. Feature implemented in new branch
3. PR #$PR created with comprehensive description
4. Automated review posted with actionable feedback
5. Related issues cross-referenced

---

## Example 6: Batch Processing

**Scenario**: Process multiple items in sequence.

### Enhance Multiple Issues

```bash
# Get list of open issues
gh issue list --repo myorg/myproject --json number -q '.[].number' | while read issue; do
  echo "Processing issue #$issue"

  # Could run individual enhancements per issue
  # (For batch, just use the full pipeline once)
done

# Better: Run once on all issues
wave run github-issue-enhancer --input '{
  "repo": "myorg/myproject",
  "threshold": 60
}'
```

### Review Multiple PRs

```bash
# Review all open PRs
gh pr list --repo myorg/myproject --json number -q '.[].number' | while read pr; do
  echo "Reviewing PR #$pr"

  wave run github-pr-review-automation --input "{
    \"repo\": \"myorg/myproject\",
    \"pr\": $pr,
    \"post_review\": false
  }"

  # Review the output before posting
  cat .wave/workspaces/github-pr-review-automation/synthesize-review/artifact.json | \
    jq -r ".summary"
done
```

---

## Example 7: Integration with CI/CD

### GitHub Actions Workflow

```yaml
# .github/workflows/wave-automation.yml
name: Wave Automation

on:
  issues:
    types: [opened]
  pull_request:
    types: [opened, synchronize]

jobs:
  enhance-issue:
    if: github.event_name == 'issues'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Enhance Issue
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          wave run github-issue-enhancer --input "{
            \"repo\": \"${{ github.repository }}\",
            \"threshold\": 70
          }"

  review-pr:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Automated Review
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          wave run github-pr-review-automation --input "{
            \"repo\": \"${{ github.repository }}\",
            \"pr\": ${{ github.event.pull_request.number }},
            \"post_review\": true
          }"
```

---

## Example 8: Custom Thresholds and Filters

### Fine-Tune Issue Enhancement

```bash
# Very aggressive - enhance almost everything
wave run github-issue-enhancer --input '{
  "repo": "myorg/myproject",
  "threshold": 40
}'

# Conservative - only worst issues
wave run github-issue-enhancer --input '{
  "repo": "myorg/myproject",
  "threshold": 90
}'
```

### Tune Cross-Linking Similarity

```bash
# Strict similarity - only very related issues
wave run github-issue-cross-linker --input '{
  "repo": "myorg/myproject",
  "similarity_threshold": 0.8
}'

# Permissive - find more connections
wave run github-issue-cross-linker --input '{
  "repo": "myorg/myproject",
  "similarity_threshold": 0.5
}'
```

### Filter by Issue State

```bash
# Link all issues (open and closed)
wave run github-issue-cross-linker --input '{
  "repo": "myorg/myproject",
  "state": "all"
}'

# Only closed issues
wave run github-issue-cross-linker --input '{
  "repo": "myorg/myproject",
  "state": "closed"
}'
```

---

## Example 9: Debugging Failed Pipelines

### Check Pipeline Status

```bash
# If pipeline fails, check the last successful step
ls -lt .wave/workspaces/github-issue-enhancer/

# View artifact from failed step
cat .wave/workspaces/github-issue-enhancer/apply-enhancements/artifact.json | jq .
```

### Review Audit Logs

```bash
# Check what commands were run
ls -lt .wave/traces/

# View latest trace
cat .wave/traces/$(ls -t .wave/traces/ | head -1)
```

### Common Issues

**Contract validation failed**:
```bash
# Check the artifact that failed validation
cat .wave/workspaces/{pipeline}/{step}/artifact.json | jq .

# Compare with expected schema
cat .wave/contracts/{schema}.json | jq .
```

**GitHub API error**:
```bash
# Check authentication
gh auth status

# Check rate limits
gh api rate_limit

# Test repo access
gh repo view myorg/myproject
```

---

## Example 10: Monitoring and Metrics

### Collect Pipeline Metrics

```bash
#!/bin/bash
# collect-metrics.sh

echo "GitHub Pipeline Metrics"
echo "======================="

# Issue Enhancer
if [ -d .wave/workspaces/github-issue-enhancer ]; then
  echo -e "\nIssue Enhancer:"
  jq -r '
    "  Issues analyzed: \(.analyzed_count)",
    "  Poor quality: \(.poor_quality_issues | length)",
    "  Threshold: \(.quality_threshold)"
  ' .wave/workspaces/github-issue-enhancer/scan-issues/artifact.json
fi

# Cross Linker
if [ -d .wave/workspaces/github-issue-cross-linker ]; then
  echo -e "\nCross Linker:"
  jq -r '
    "  Relationships found: \(.total_relationships)",
    "  High priority: \(.high_priority_count)",
    "  Medium priority: \(.medium_priority_count)",
    "  Low priority: \(.low_priority_count)"
  ' .wave/workspaces/github-issue-cross-linker/analyze-relationships/artifact.json
fi

# PR Review
if [ -d .wave/workspaces/github-pr-review-automation ]; then
  echo -e "\nPR Review:"
  jq -r '
    "  Assessment: \(.overall_assessment)",
    "  Security issues: \(.security_issues)",
    "  Quality issues: \(.quality_issues)",
    "  Must fix: \(.must_fix | length)",
    "  Should fix: \(.should_fix | length)"
  ' .wave/workspaces/github-pr-review-automation/synthesize-review/artifact.json
fi
```

Run it:
```bash
chmod +x collect-metrics.sh
./collect-metrics.sh
```

---

## Next Steps

- Try these examples on your own repositories
- Customize parameters for your needs
- Create scripts to automate common workflows
- Integrate with your CI/CD pipelines
- Monitor results and adjust thresholds

For more information:
- Full documentation: `GITHUB_PIPELINES_README.md`
- Quick reference: `GITHUB_PIPELINES_QUICK_START.md`
- Testing guide: `GITHUB_PIPELINES_TESTING.md`
