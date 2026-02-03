# GitHub Pipelines Quick Start

One-page reference for Wave's GitHub integration pipelines.

## Setup

```bash
# Install GitHub CLI if not already installed
# See: https://cli.github.com/

# Authenticate
gh auth login

# Verify access to your repository
gh repo view owner/repository
```

## Pipeline Commands

### 1. Enhance Poor Quality Issues

Analyze and improve issue quality automatically.

```bash
# Analyze issues (read-only, high threshold)
wave run github-issue-enhancer --input '{
  "repo": "owner/repo",
  "threshold": 100
}'

# Actually enhance issues scoring below 70
wave run github-issue-enhancer --input '{
  "repo": "owner/repo",
  "threshold": 70
}'
```

**Use when**: Issues lack structure, clear titles, or proper labels.

---

### 2. Implement Feature from Issue

Turn an issue into a working PR with tests.

```bash
# Implement feature from issue #42
wave run github-feature-implementation --input '{
  "repo": "owner/repo",
  "issue": 42
}'

# Use custom base branch
wave run github-feature-implementation --input '{
  "repo": "owner/repo",
  "issue": 42,
  "base_branch": "develop"
}'
```

**Use when**: You have a well-defined issue and want automated implementation.

---

### 3. Cross-Link Related Issues

Discover and link related issues automatically.

```bash
# Link related open issues
wave run github-issue-cross-linker --input '{
  "repo": "owner/repo"
}'

# Include closed issues, higher similarity threshold
wave run github-issue-cross-linker --input '{
  "repo": "owner/repo",
  "state": "all",
  "similarity_threshold": 0.75
}'
```

**Use when**: Issues lack cross-references, duplicates exist, or relationships aren't clear.

---

### 4. Automated PR Review

Comprehensive security and quality review of pull requests.

```bash
# Prepare review (doesn't post to GitHub)
wave run github-pr-review-automation --input '{
  "repo": "owner/repo",
  "pr": 42
}'

# Prepare AND post review to GitHub
wave run github-pr-review-automation --input '{
  "repo": "owner/repo",
  "pr": 42,
  "post_review": true
}'
```

**Use when**: You want automated code review before human review.

---

## Pipeline Outputs

All pipelines create verification reports in their final step:

```bash
# Check the last step's artifact for summary
cat .wave/workspaces/{pipeline-name}/verify-*/artifact.json | jq .
```

Common output locations:
- `.wave/workspaces/{pipeline}/` - Step workspaces with artifacts
- `.wave/traces/` - Audit logs with all GitHub CLI commands

## Quick Troubleshooting

| Error | Solution |
|-------|----------|
| `gh: command not found` | Install GitHub CLI: https://cli.github.com/ |
| `HTTP 401` | Run `gh auth login` |
| `Rate limit exceeded` | Wait 1 hour or use smaller batches |
| `Issue/PR not found` | Check repo format: "owner/repo" |
| `Contract validation failed` | Check artifact.json in workspace for details |

## Safety Tips

1. **Test read-only first**: Use high thresholds or dry-run modes
2. **Start small**: Test on 5-10 items before full runs
3. **Review artifacts**: Check outputs before re-running with write operations
4. **Use test repositories**: Practice on non-critical repos first

## Common Workflows

### Workflow 1: Issue Cleanup
```bash
# Step 1: Enhance poor issues
wave run github-issue-enhancer --input '{"repo": "myorg/myproject", "threshold": 60}'

# Step 2: Cross-link related issues
wave run github-issue-cross-linker --input '{"repo": "myorg/myproject"}'
```

### Workflow 2: Feature Development
```bash
# Step 1: Implement feature
wave run github-feature-implementation --input '{"repo": "myorg/myproject", "issue": 42}'

# Step 2: Review the created PR (find PR number from step 1 output)
wave run github-pr-review-automation --input '{"repo": "myorg/myproject", "pr": 123}'
```

### Workflow 3: Code Quality
```bash
# Review all open PRs
for pr in $(gh pr list --repo myorg/myproject --json number -q '.[].number'); do
  wave run github-pr-review-automation --input "{\"repo\": \"myorg/myproject\", \"pr\": $pr}"
done
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Wave PR Review
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup Wave
        run: |
          # Install wave binary
          curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-linux-amd64 -o wave
          chmod +x wave
      - name: Run automated review
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          ./wave run github-pr-review-automation --input "{
            \"repo\": \"${{ github.repository }}\",
            \"pr\": ${{ github.event.pull_request.number }},
            \"post_review\": true
          }"
```

## Advanced Usage

### Custom Quality Thresholds
```bash
# Very aggressive - enhance almost everything
wave run github-issue-enhancer --input '{"repo": "owner/repo", "threshold": 40}'

# Conservative - only worst issues
wave run github-issue-enhancer --input '{"repo": "owner/repo", "threshold": 90}'
```

### Similarity Tuning
```bash
# Strict - only very similar issues linked
wave run github-issue-cross-linker --input '{"repo": "owner/repo", "similarity_threshold": 0.8}'

# Permissive - more connections
wave run github-issue-cross-linker --input '{"repo": "owner/repo", "similarity_threshold": 0.5}'
```

### State Filtering
```bash
# All issues (open and closed)
wave run github-issue-cross-linker --input '{"repo": "owner/repo", "state": "all"}'

# Only closed issues
wave run github-issue-cross-linker --input '{"repo": "owner/repo", "state": "closed"}'
```

## Monitoring and Metrics

Check pipeline results:
```bash
# View verification report
cat .wave/workspaces/github-issue-enhancer/verify-enhancements/artifact.json | jq .

# Count enhancements made
jq '.total_enhanced' .wave/workspaces/github-issue-enhancer/apply-enhancements/artifact.json

# View review assessment
jq '.overall_assessment' .wave/workspaces/github-pr-review-automation/synthesize-review/artifact.json
```

## Next Steps

- Read full documentation: `GITHUB_PIPELINES_README.md`
- Review individual pipeline YAML files for step details
- Check contract schemas in `.wave/contracts/github-*.schema.json`
- Customize personas in `.wave/personas/github-*.md`

---

**Quick Support**:
- Pipeline not working? Check `.wave/traces/` for audit logs
- Unexpected results? Review `.wave/workspaces/{pipeline}/*/artifact.json`
- Need help? See main README and Wave documentation
