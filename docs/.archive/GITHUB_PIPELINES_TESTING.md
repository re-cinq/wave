# Testing GitHub Pipelines

Comprehensive guide for testing Wave's GitHub integration pipelines safely and effectively.

## Testing Philosophy

### Progressive Testing Strategy

1. **Syntax Validation** - Verify YAML is well-formed
2. **Schema Validation** - Test contract schemas
3. **Read-Only Testing** - Run without modifications
4. **Small Batch Testing** - Test on limited data
5. **Full Execution** - Run on production data
6. **Monitoring** - Track results and metrics

### Safety First

- Always test on non-critical repositories first
- Use read-only modes before write operations
- Review artifacts before enabling write operations
- Start with small batches (5-10 items)
- Keep backups of important issues/PRs

## Test Environments

### Option 1: Personal Test Repository

Create a dedicated test repository:

```bash
# Create test repo
gh repo create wave-pipeline-test --public --description "Testing Wave pipelines"

# Create some test issues
gh issue create --repo yourname/wave-pipeline-test \
  --title "bug" \
  --body "doesnt work"

gh issue create --repo yourname/wave-pipeline-test \
  --title "Add feature X" \
  --body "We should add feature X because reasons"

gh issue create --repo yourname/wave-pipeline-test \
  --title "authentication problem" \
  --body "cant login"
```

### Option 2: Fork Existing Repository

```bash
# Fork a repository
gh repo fork popular-org/popular-repo --clone=false

# Use fork for testing
wave run github-issue-enhancer --input '{"repo": "yourname/popular-repo", "threshold": 70}'
```

### Option 3: Private Sandbox

Use your organization's sandbox/test environment.

## Pipeline-Specific Tests

### 1. Testing github-issue-enhancer

#### Test 1: Read-Only Analysis

```bash
# Set threshold to 100 - nothing will be enhanced
wave run github-issue-enhancer --input '{
  "repo": "yourname/test-repo",
  "threshold": 100
}'

# Check results
cat .wave/workspaces/github-issue-enhancer/scan-issues/artifact.json | jq .
```

**Expected**: Analysis completes, identifies poor quality issues, but doesn't enhance any.

**Verify**:
- `poor_quality_issues` array is populated
- No changes made to GitHub issues
- Quality scores calculated correctly

#### Test 2: Dry Run with Planning

```bash
# Run through planning step
wave run github-issue-enhancer --input '{
  "repo": "yourname/test-repo",
  "threshold": 70
}'

# Stop after plan-enhancements step (modify pipeline to remove later steps temporarily)
# Review the enhancement plan
cat .wave/workspaces/github-issue-enhancer/plan-enhancements/artifact.json | jq .
```

**Expected**: Enhancement plan created with suggested improvements.

**Verify**:
- `issues_to_enhance` contains reasonable suggestions
- `suggested_title` improvements make sense
- `body_template` preserves original content

#### Test 3: Single Issue Enhancement

Modify pipeline temporarily to enhance just one issue:

```bash
# Create test issue
ISSUE=$(gh issue create --repo yourname/test-repo --title "bug" --body "problem" --json number -q .number)

# Run enhancer
wave run github-issue-enhancer --input "{\"repo\": \"yourname/test-repo\", \"threshold\": 70}"

# Verify enhancement
gh issue view $ISSUE --repo yourname/test-repo
```

**Expected**: Issue is enhanced with better title, structured description, labels.

**Verify**:
- Title is improved and capitalized
- Description has structured template
- Original content is preserved
- Labels are added appropriately
- Comment explains the enhancement

#### Test 4: Batch Enhancement

```bash
# Create 5 test issues with varying quality
for i in {1..5}; do
  gh issue create --repo yourname/test-repo \
    --title "issue $i" \
    --body "some text"
done

# Run enhancer
wave run github-issue-enhancer --input '{
  "repo": "yourname/test-repo",
  "threshold": 70
}'

# Check results
cat .wave/workspaces/github-issue-enhancer/verify-enhancements/artifact.json | jq .
```

**Expected**: Multiple issues enhanced successfully.

**Verify**:
- `total_enhanced` matches expected count
- `all_successful: true`
- All enhanced issues have improvements

---

### 2. Testing github-feature-implementation

#### Test 1: Simple Feature Issue

```bash
# Create a simple, well-defined issue
gh issue create --repo yourname/test-repo \
  --title "Add hello world function" \
  --body "Create a function that returns 'Hello, World!'" \
  --label "enhancement"

# Run implementation (note: this WILL create code and a PR)
wave run github-feature-implementation --input '{
  "repo": "yourname/test-repo",
  "issue": 1
}'
```

**Expected**: Feature branch created, code implemented, tests added, PR created.

**Verify**:
- Branch exists: `gh pr view 1 --repo yourname/test-repo --json headRefName`
- Code compiles/runs
- Tests pass
- PR description is comprehensive
- PR links to issue

#### Test 2: Read-Only Analysis

```bash
# Run only the analyze-requirement step
# (Modify pipeline temporarily to stop after first step)

wave run github-feature-implementation --input '{
  "repo": "yourname/test-repo",
  "issue": 1
}'

# Check analysis
cat .wave/workspaces/github-feature-implementation/analyze-requirement/artifact.json | jq .
```

**Expected**: Requirements extracted, no code changes.

**Verify**:
- Requirements are identified correctly
- Affected files are relevant
- Technical approach is reasonable

#### Test 3: Complex Feature

```bash
# Create more complex issue
gh issue create --repo yourname/test-repo \
  --title "Implement user authentication" \
  --body "Add JWT-based authentication with login/logout endpoints" \
  --label "enhancement"

# Run implementation
wave run github-feature-implementation --input '{
  "repo": "yourname/test-repo",
  "issue": 2
}'
```

**Expected**: More complex implementation with multiple files, tests, PR.

**Verify**:
- Multiple files changed
- Comprehensive test coverage
- PR explains complexity
- Code follows project patterns

---

### 3. Testing github-issue-cross-linker

#### Test 1: Small Issue Set

```bash
# Create related issues
gh issue create --repo yourname/test-repo \
  --title "OAuth login fails" \
  --body "Users can't log in with OAuth" \
  --label "bug,authentication"

gh issue create --repo yourname/test-repo \
  --title "JWT token validation error" \
  --body "Token validation returns 401" \
  --label "bug,authentication"

gh issue create --repo yourname/test-repo \
  --title "Add OAuth provider" \
  --body "Support Google OAuth login" \
  --label "enhancement,authentication"

# Run cross-linker
wave run github-issue-cross-linker --input '{
  "repo": "yourname/test-repo",
  "state": "open"
}'
```

**Expected**: Related authentication issues are cross-linked.

**Verify**:
- Issues with shared labels are linked
- Comment rationale makes sense
- Bidirectional links created
- No spurious connections

#### Test 2: Relationship Analysis Only

```bash
# Run through relationship detection only
# (Stop before apply-cross-links step)

wave run github-issue-cross-linker --input '{
  "repo": "yourname/test-repo"
}'

# Review relationships
cat .wave/workspaces/github-issue-cross-linker/analyze-relationships/artifact.json | jq .
```

**Expected**: Relationships detected but not posted.

**Verify**:
- `relationships` array has sensible connections
- `similarity_score` is reasonable
- `relationship_type` is appropriate
- `rationale` explains connection

#### Test 3: Similarity Threshold Tuning

```bash
# Test different thresholds
for threshold in 0.5 0.6 0.7 0.8; do
  echo "Testing threshold: $threshold"
  wave run github-issue-cross-linker --input "{
    \"repo\": \"yourname/test-repo\",
    \"similarity_threshold\": $threshold
  }"

  # Count relationships found
  jq '.total_relationships' \
    .wave/workspaces/github-issue-cross-linker/analyze-relationships/artifact.json
done
```

**Expected**: Lower thresholds find more relationships.

**Verify**:
- Threshold affects relationship count
- Higher threshold = fewer, higher-quality links
- Lower threshold = more links, some may be weak

---

### 4. Testing github-pr-review-automation

#### Test 1: Dry Run Review

```bash
# Create a test PR (or use existing)
gh pr create --repo yourname/test-repo \
  --title "Add feature" \
  --body "Implementation of feature X" \
  --base main \
  --head feature-branch

# Run review WITHOUT posting (default)
wave run github-pr-review-automation --input '{
  "repo": "yourname/test-repo",
  "pr": 1
}'

# Check review
cat .wave/workspaces/github-pr-review-automation/synthesize-review/artifact.json | jq .
```

**Expected**: Review generated but NOT posted to GitHub.

**Verify**:
- `overall_assessment` is appropriate
- `review_comments` identify real issues
- `must_fix` items are critical
- `should_fix` are reasonable suggestions
- `positive_notes` highlight good aspects

#### Test 2: Security Issue Detection

```bash
# Create PR with security issues
# (e.g., hardcoded credentials, SQL injection, etc.)

# Run review
wave run github-pr-review-automation --input '{
  "repo": "yourname/test-repo",
  "pr": 2
}'

# Check security findings
cat .wave/workspaces/github-pr-review-automation/security-review/artifact.json | jq .security_findings
```

**Expected**: Security issues detected with appropriate severity.

**Verify**:
- Critical issues are flagged
- Severity ratings are appropriate
- Suggestions are actionable
- Line numbers are correct (if provided)

#### Test 3: Post Review to GitHub

```bash
# After verifying review looks good, post it
wave run github-pr-review-automation --input '{
  "repo": "yourname/test-repo",
  "pr": 1,
  "post_review": true
}'

# Verify posted review
gh pr view 1 --repo yourname/test-repo --comments
```

**Expected**: Review posted to GitHub with proper formatting.

**Verify**:
- Review appears on PR
- Review type matches overall assessment
- Markdown formatting is correct
- Comments are helpful and constructive

---

## Integration Testing

### End-to-End Workflow Test

```bash
# Complete workflow: Issue → Implementation → Review

# 1. Create issue with poor quality
gh issue create --repo yourname/test-repo \
  --title "bug" \
  --body "doesnt work"

# 2. Enhance issue
wave run github-issue-enhancer --input '{
  "repo": "yourname/test-repo",
  "threshold": 70
}'

# 3. Implement feature (assuming issue #1)
wave run github-feature-implementation --input '{
  "repo": "yourname/test-repo",
  "issue": 1
}'

# 4. Review the created PR (assuming PR #1)
wave run github-pr-review-automation --input '{
  "repo": "yourname/test-repo",
  "pr": 1
}'

# 5. Cross-link related issues
wave run github-issue-cross-linker --input '{
  "repo": "yourname/test-repo"
}'
```

**Expected**: Complete development workflow automated.

**Verify**:
- Issue is enhanced
- Feature is implemented
- PR is created
- Review is generated
- Related issues are linked

---

## Performance Testing

### Stress Test: Large Issue Set

```bash
# Create 50 test issues
for i in {1..50}; do
  gh issue create --repo yourname/test-repo \
    --title "Test issue $i" \
    --body "This is test issue number $i"
done

# Run enhancer
time wave run github-issue-enhancer --input '{
  "repo": "yourname/test-repo",
  "threshold": 70
}'

# Measure performance
echo "Time taken: see above"
echo "Issues processed: $(jq '.total_enhanced' .wave/workspaces/github-issue-enhancer/apply-enhancements/artifact.json)"
```

**Expected**: Pipeline handles large batch without errors.

**Verify**:
- No rate limit errors
- All issues processed
- Reasonable execution time
- No memory issues

---

## Error Handling Tests

### Test 1: Invalid Input

```bash
# Non-existent repository
wave run github-issue-enhancer --input '{
  "repo": "invalid/nonexistent",
  "threshold": 70
}'

# Non-existent issue
wave run github-feature-implementation --input '{
  "repo": "yourname/test-repo",
  "issue": 99999
}'
```

**Expected**: Clear error messages, graceful failure.

### Test 2: Rate Limiting

```bash
# Trigger rate limiting (may need many API calls)
wave run github-issue-enhancer --input '{
  "repo": "yourname/large-repo",
  "threshold": 30
}'
```

**Expected**: Rate limit errors handled gracefully with retry logic.

### Test 3: Network Failures

```bash
# Simulate network issues (disconnect during execution)
wave run github-feature-implementation --input '{
  "repo": "yourname/test-repo",
  "issue": 1
}'
# Disconnect network mid-execution
```

**Expected**: Pipeline fails gracefully with resumable state.

---

## Validation Testing

### Contract Schema Validation

```bash
# Test each contract schema
for schema in .wave/contracts/github-*.schema.json; do
  echo "Testing $schema"

  # Use a JSON schema validator
  # Example with Python:
  python3 -c "
import json
schema = json.load(open('$schema'))
print('Valid JSON:', '$schema')
"
done
```

### Artifact Validation

```bash
# After running a pipeline, validate all artifacts
for artifact in .wave/workspaces/github-*/*/artifact.json; do
  echo "Validating $artifact"

  # Validate JSON structure
  jq . "$artifact" > /dev/null && echo "✓ Valid" || echo "✗ Invalid"
done
```

---

## Regression Testing

### Test Suite

Create a test suite that runs regularly:

```bash
#!/bin/bash
# test-github-pipelines.sh

set -e

REPO="yourname/wave-test"

echo "=== Testing GitHub Pipelines ==="

# Test 1: Issue Enhancer
echo "Test 1: Issue Enhancer..."
wave run github-issue-enhancer --input "{\"repo\": \"$REPO\", \"threshold\": 100}"
echo "✓ Test 1 passed"

# Test 2: Cross Linker (analysis only)
echo "Test 2: Cross Linker..."
wave run github-issue-cross-linker --input "{\"repo\": \"$REPO\"}"
echo "✓ Test 2 passed"

# Test 3: PR Review (dry run)
echo "Test 3: PR Review..."
wave run github-pr-review-automation --input "{\"repo\": \"$REPO\", \"pr\": 1}"
echo "✓ Test 3 passed"

echo "=== All tests passed ==="
```

Run regularly:
```bash
./test-github-pipelines.sh
```

---

## Monitoring and Metrics

### Collect Test Metrics

```bash
# After test runs, collect metrics
cat > test-metrics.sh <<'EOF'
#!/bin/bash

echo "Pipeline Test Metrics"
echo "===================="

# Issue Enhancer metrics
echo "Issue Enhancer:"
jq '{
  total_analyzed: .analyzed_count,
  poor_quality_found: (.poor_quality_issues | length),
  total_enhanced: 0
}' .wave/workspaces/github-issue-enhancer/scan-issues/artifact.json

# Cross Linker metrics
echo "Cross Linker:"
jq '{
  total_relationships: .total_relationships,
  high_priority: .high_priority_count,
  medium_priority: .medium_priority_count,
  low_priority: .low_priority_count
}' .wave/workspaces/github-issue-cross-linker/analyze-relationships/artifact.json

# PR Review metrics
echo "PR Review:"
jq '{
  security_issues: .security_issues,
  quality_issues: .quality_issues,
  overall_assessment: .overall_assessment
}' .wave/workspaces/github-pr-review-automation/synthesize-review/artifact.json
EOF

chmod +x test-metrics.sh
./test-metrics.sh
```

---

## Best Practices

### Before Testing
- [ ] Have test repository ready
- [ ] Authenticate GitHub CLI
- [ ] Review pipeline YAML files
- [ ] Understand expected outputs

### During Testing
- [ ] Start with read-only tests
- [ ] Review artifacts after each run
- [ ] Test incrementally (one step at a time if needed)
- [ ] Monitor rate limits

### After Testing
- [ ] Verify all outputs
- [ ] Check GitHub for actual changes
- [ ] Review audit logs
- [ ] Document any issues found

### Production Deployment
- [ ] All tests passing
- [ ] Dry runs successful
- [ ] Small batch tests successful
- [ ] Error handling verified
- [ ] Monitoring in place

---

## Troubleshooting Tests

### Test Failures

**Pipeline fails at contract validation**:
- Check artifact.json structure
- Compare against schema in .wave/contracts/
- Look for missing required fields

**GitHub API errors**:
- Check authentication: `gh auth status`
- Verify repository access: `gh repo view owner/repo`
- Check rate limits: `gh api rate_limit`

**Unexpected results**:
- Review audit logs in .wave/traces/
- Check persona system prompts
- Verify input parameters

**Performance issues**:
- Reduce batch size
- Check network latency
- Monitor memory usage

---

## Continuous Testing

### GitHub Actions Integration

```yaml
name: Test Wave Pipelines
on:
  schedule:
    - cron: '0 0 * * *'  # Daily
  workflow_dispatch:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup
        run: |
          # Install wave
          # Setup test environment
      - name: Run Tests
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          ./test-github-pipelines.sh
      - name: Report Results
        if: always()
        run: |
          # Collect and report metrics
```

---

## Next Steps

- Test on your own repositories
- Customize thresholds and parameters
- Add custom validation rules
- Create your own test suites
- Monitor production usage

For production deployment, ensure all tests pass consistently before enabling write operations on critical repositories.
