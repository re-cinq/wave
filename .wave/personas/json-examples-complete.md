# Complete JSON Examples for Wave Personas

This document provides real-world validated JSON examples that demonstrate the templates in action with realistic GitHub repository data.

## GitHub Issue Analysis - Complete Examples

### Example 1: React Repository Analysis
```json
{
  "repository": {
    "owner": "facebook",
    "name": "react"
  },
  "total_issues": 1247,
  "analyzed_count": 50,
  "poor_quality_issues": [
    {
      "number": 28543,
      "title": "bug",
      "body": "it doesn't work",
      "quality_score": 15,
      "problems": [
        "Title is too vague and lowercase",
        "No steps to reproduce provided",
        "No error messages or context",
        "Missing environment details"
      ],
      "recommendations": [
        "Rewrite title as 'Fix rendering error with Suspense and StrictMode'",
        "Add step-by-step reproduction instructions",
        "Include error stack trace or console errors",
        "Add React version, browser, and OS details"
      ],
      "labels": ["Status: Unconfirmed"],
      "url": "https://github.com/facebook/react/issues/28543"
    },
    {
      "number": 28501,
      "title": "Question about hooks",
      "body": "I have a question about using hooks. Can someone help me?",
      "quality_score": 35,
      "problems": [
        "Question posted as bug instead of using discussions",
        "No specific use case or code example",
        "Too vague to provide actionable help"
      ],
      "recommendations": [
        "Move to GitHub Discussions with 'Question' category",
        "Provide specific code example and expected behavior",
        "Include what you've tried so far"
      ],
      "labels": ["Type: Question"],
      "url": "https://github.com/facebook/react/issues/28501"
    }
  ],
  "quality_threshold": 70,
  "timestamp": "2026-02-03T15:30:00Z"
}
```

### Example 2: Kubernetes Repository Analysis
```json
{
  "repository": {
    "owner": "kubernetes",
    "name": "kubernetes"
  },
  "total_issues": 2847,
  "analyzed_count": 100,
  "poor_quality_issues": [
    {
      "number": 123456,
      "title": "kubectl broken",
      "body": "kubectl doesn't work on my machine\n\nPlease fix this ASAP!!!",
      "quality_score": 20,
      "problems": [
        "Title lacks specificity about what's broken",
        "No kubectl version or command attempted",
        "No error output provided",
        "No cluster or environment information",
        "Demanding tone without context"
      ],
      "recommendations": [
        "Specify exact kubectl command that fails",
        "Include kubectl version and Kubernetes cluster version",
        "Provide complete error output",
        "Add cluster configuration details (cloud provider, version, etc.)",
        "Use respectful tone and follow issue template"
      ],
      "labels": ["kind/bug", "needs-triage"],
      "url": "https://github.com/kubernetes/kubernetes/issues/123456"
    }
  ],
  "quality_threshold": 75,
  "timestamp": "2026-02-03T16:15:00Z"
}
```

## GitHub Enhancement Results - Complete Examples

### Example 1: Successful Enhancement
```json
{
  "enhanced_issues": [
    {
      "issue_number": 28543,
      "success": true,
      "changes_made": [
        "Updated title from 'bug' to 'Fix rendering error when using Suspense with StrictMode'",
        "Added structured issue template with reproduction steps section",
        "Applied labels: 'Component: Suspense', 'Type: Bug', 'Status: Needs Reproduction'",
        "Added comment requesting environment details and reproduction steps"
      ],
      "title_updated": true,
      "body_updated": true,
      "labels_added": ["Component: Suspense", "Type: Bug", "Status: Needs Reproduction"],
      "comment_added": true,
      "url": "https://github.com/facebook/react/issues/28543"
    },
    {
      "issue_number": 28501,
      "success": true,
      "changes_made": [
        "Converted issue to discussion in 'Q&A' category",
        "Added helpful template for hook-related questions",
        "Applied 'question' label before conversion"
      ],
      "title_updated": false,
      "body_updated": true,
      "labels_added": ["question"],
      "comment_added": true,
      "url": "https://github.com/facebook/react/discussions/28501"
    }
  ],
  "total_attempted": 2,
  "total_successful": 2,
  "total_failed": 0,
  "timestamp": "2026-02-03T15:45:00Z"
}
```

### Example 2: Mixed Results with Failures
```json
{
  "enhanced_issues": [
    {
      "issue_number": 123456,
      "success": true,
      "changes_made": [
        "Updated title to 'kubectl apply fails with 'connection refused' error'",
        "Added kubectl debug template with version and environment sections",
        "Applied appropriate labels for kubectl issues"
      ],
      "title_updated": true,
      "body_updated": true,
      "labels_added": ["area/kubectl", "kind/bug", "needs-triage"],
      "comment_added": true,
      "url": "https://github.com/kubernetes/kubernetes/issues/123456"
    },
    {
      "issue_number": 123457,
      "success": false,
      "changes_made": [],
      "title_updated": false,
      "body_updated": false,
      "labels_added": [],
      "comment_added": false,
      "error": "GitHub API returned 403: Forbidden - insufficient permissions to edit this issue",
      "url": "https://github.com/kubernetes/kubernetes/issues/123457"
    },
    {
      "issue_number": 123458,
      "success": false,
      "changes_made": [],
      "error": "Issue appears to be locked due to heated discussion, cannot modify",
      "url": "https://github.com/kubernetes/kubernetes/issues/123458"
    }
  ],
  "total_attempted": 3,
  "total_successful": 1,
  "total_failed": 2,
  "timestamp": "2026-02-03T16:30:00Z"
}
```

## GitHub PR Draft - Complete Examples

### Example 1: Feature Enhancement PR
```json
{
  "title": "Add real-time validation for GitHub persona JSON outputs",
  "body": "## Summary\nThis PR implements a comprehensive JSON validation system for Wave personas to eliminate contract validation failures and ensure 100% pipeline reliability.\n\n## Changes\n- Add automated validation scripts for GitHub issue analysis, enhancement results, and PR draft schemas\n- Create standardized JSON templates with realistic examples\n- Implement real-time validation during persona execution\n- Add emergency error recovery and auto-fix mechanisms\n- Create comprehensive test suite for validation scenarios\n\n## Motivation\nAI personas were producing valuable analysis but inconsistent JSON formatting that broke contract validation, causing:\n- Pipeline execution failures requiring manual intervention\n- Development time lost to debugging validation errors\n- Reduced confidence in automated pipeline results\n\nFixes #142 and addresses concerns raised in #98\n\n## Test Plan\n- [x] Manual testing with realistic GitHub repository data (React, Kubernetes)\n- [x] Unit tests for all validation patterns and edge cases\n- [x] Integration tests with existing Wave pipeline execution\n- [x] Performance testing with large JSON outputs (1000+ issues)\n- [x] Error recovery testing with common JSON syntax errors\n- [ ] End-to-end testing with actual GitHub API integration\n\n## Breaking Changes\nNone. This is a backward-compatible enhancement that adds validation without changing existing functionality.\n\n## Checklist\n- [x] Tests added for new validation functionality\n- [x] Documentation updated with usage examples\n- [x] No breaking changes to existing personas\n- [x] Code reviewed for security and performance\n- [x] Error handling covers all identified failure modes",
  "head": "feature/json-validation-system",
  "base": "main",
  "draft": false,
  "labels": ["enhancement", "validation", "json", "github-integration"],
  "reviewers": ["architecture-team", "security-reviewer"],
  "related_issues": [142, 98],
  "breaking_changes": false
}
```

### Example 2: Bug Fix PR
```json
{
  "title": "Fix authentication error handling in GitHub API adapter",
  "body": "## Summary\nThis PR fixes authentication error handling in the GitHub API adapter that was causing pipeline failures when rate limits were exceeded or tokens expired.\n\n## Changes\n- Add exponential backoff retry mechanism for rate limit errors\n- Improve token validation and renewal process\n- Add detailed error logging for authentication failures\n- Implement graceful degradation when API access is limited\n\n## Root Cause\nThe GitHub adapter was not properly handling HTTP 403 responses for rate limiting, causing the entire pipeline to fail instead of implementing retry logic.\n\n## Test Plan\n- [x] Manual testing with expired tokens\n- [x] Rate limit simulation testing\n- [x] Unit tests for retry logic\n- [x] Integration tests with GitHub API sandbox\n\n## Checklist\n- [x] Tests added for error scenarios\n- [x] Error messages improved for debugging\n- [x] No breaking changes to existing API\n- [x] Backward compatible with existing configurations",
  "head": "fix/github-auth-error-handling",
  "base": "main",
  "draft": false,
  "labels": ["bug", "github-api", "authentication", "critical"],
  "reviewers": ["security-team"],
  "related_issues": [203],
  "breaking_changes": false
}
```

## Validation Results

All examples above have been validated with the Wave validation framework:

```bash
# Test all examples
for file in example-*.json; do
    echo "Testing $file..."
    ./.wave/scripts/validate-persona-json.sh "$file"
    echo "---"
done
```

Expected results:
- ✓ All JSON syntax valid
- ✓ All schema requirements met
- ✓ No placeholder content detected
- ✓ Business logic constraints satisfied
- ✓ Character length requirements met
- ✓ Data types correctly unquoted (numbers, booleans)

## Usage Tips

### For GitHub Issue Analyst
1. Use realistic repository names and issue numbers
2. Include specific, actionable problems and recommendations
3. Keep quality scores consistent with identified problems
4. Escape newlines in issue bodies properly

### For GitHub Issue Enhancer
1. Ensure total counts are mathematically correct
2. Include specific change descriptions
3. Handle failures with meaningful error messages
4. Use boolean flags consistently

### For GitHub PR Creator
1. Write comprehensive PR bodies with multiple sections
2. Meet minimum length requirements (title ≥10, body ≥50 chars)
3. Use realistic branch names and related issue numbers
4. Include proper markdown formatting in body

These examples demonstrate the templates working with realistic data and passing all validation requirements.