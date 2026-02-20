---
title: Code Review
description: Automated PR reviews with security checks, quality analysis, and actionable feedback
---

# Code Review

<div class="use-case-meta">
  <span class="complexity-badge beginner">Beginner</span>
  <span class="category-badge">Code Quality</span>
</div>

Automate pull request reviews with security checks, quality analysis, and actionable feedback. Wave's code review pipeline analyzes changes, identifies issues, and produces a structured review summary.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Git repository with changes to review
- Basic understanding of YAML configuration

## Quick Start

```bash
wave run code-review "review the authentication module"
```

With `-o text`:

```
[10:00:01] → diff-analysis (navigator)
[10:00:01]   diff-analysis: Executing agent
[10:00:25] ✓ diff-analysis completed (24.0s, 2.5k tokens)
[10:00:26] → security-review (auditor)
[10:00:26] → quality-review (auditor)
[10:00:45] ✓ security-review completed (19.0s, 1.8k tokens)
[10:00:48] ✓ quality-review completed (22.0s, 2.1k tokens)
[10:00:49] → summary (summarizer)
[10:01:05] ✓ summary completed (16.0s, 1.2k tokens)

  ✓ Pipeline 'code-review' completed successfully (64s)
```

## Complete Pipeline

This is the full `code-review` pipeline from `.wave/pipelines/code-review.yaml`:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Comprehensive code review for pull requests"

input:
  source: cli

steps:
  - id: diff-analysis
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze the code changes for: {{ input }}

        1. Identify all modified files and their purposes
        2. Map the change scope (which modules/packages affected)
        3. Find related tests that should be updated
        4. Check for breaking API changes

        Output as JSON:
        {
          "files_changed": [{"path": "", "change_type": "added|modified|deleted", "purpose": ""}],
          "modules_affected": [],
          "related_tests": [],
          "breaking_changes": []
        }
    output_artifacts:
      - name: diff
        path: .wave/output/diff-analysis.json
        type: json

  - id: security-review
    persona: auditor
    dependencies: [diff-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: diff-analysis
          artifact: diff
          as: changes
    exec:
      type: prompt
      source: |
        Security review of the changes:

        Check for:
        1. SQL injection, XSS, CSRF vulnerabilities
        2. Hardcoded secrets or credentials
        3. Insecure deserialization
        4. Missing input validation
        5. Authentication/authorization gaps
        6. Sensitive data exposure

        Output findings with severity (CRITICAL/HIGH/MEDIUM/LOW).
    output_artifacts:
      - name: security
        path: .wave/output/security-review.md
        type: markdown

  - id: quality-review
    persona: auditor
    dependencies: [diff-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: diff-analysis
          artifact: diff
          as: changes
    exec:
      type: prompt
      source: |
        Quality review of the changes:

        Check for:
        1. Error handling completeness
        2. Edge cases not covered
        3. Code duplication
        4. Naming consistency
        5. Missing or inadequate tests
        6. Performance implications
        7. Documentation gaps

        Output findings with severity and suggestions.
    output_artifacts:
      - name: quality
        path: .wave/output/quality-review.md
        type: markdown

  - id: summary
    persona: summarizer
    dependencies: [security-review, quality-review]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: security-review
          artifact: security
          as: security_findings
        - step: quality-review
          artifact: quality
          as: quality_findings
    exec:
      type: prompt
      source: |
        Synthesize the review findings into a final verdict:

        1. Overall assessment (APPROVE / REQUEST_CHANGES / NEEDS_DISCUSSION)
        2. Critical issues that must be fixed
        3. Suggested improvements (optional but recommended)
        4. Positive observations

        Format as a PR review comment.
    output_artifacts:
      - name: verdict
        path: .wave/output/review-summary.md
        type: markdown
```

</div>

## Expected Outputs

The pipeline produces three artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `diff` | `.wave/output/diff-analysis.json` | JSON analysis of changed files and scope |
| `security` | `.wave/output/security-review.md` | Security findings with severity levels |
| `verdict` | `.wave/output/review-summary.md` | Final review summary and recommendation |

### Example Output

The pipeline produces `.wave/output/review-summary.md`:

```markdown
## Code Review: Authentication Module

### Overall Assessment: REQUEST_CHANGES

### Critical Issues (Must Fix)

1. **Missing rate limiting on login endpoint** (HIGH)
   - File: `auth/handler.go:45`
   - Risk: Brute force attacks possible
   - Fix: Add rate limiting middleware

2. **Password comparison not constant-time** (MEDIUM)
   - File: `auth/verify.go:23`
   - Risk: Timing attack vulnerability
   - Fix: Use `crypto/subtle.ConstantTimeCompare`

### Suggested Improvements

- Add context timeout to database queries
- Consider extracting JWT logic into separate package
- Add table-driven tests for edge cases

### Positive Observations

- Clean separation between handlers and business logic
- Good use of structured logging
- Comprehensive input validation
```

## Customization

### Focus on specific areas

```bash
wave run code-review "focus on error handling in the API layer"
```

### Review a specific PR

```bash
wave run code-review "review changes in PR #123"
```

### Add contract validation

Add a JSON schema to ensure structured output:

<div v-pre>

```yaml
- id: diff-analysis
  handover:
    contract:
      type: json_schema
      schema_path: .wave/contracts/diff-analysis.schema.json
      source: .wave/output/diff-analysis.json
      on_failure: retry
      max_retries: 2
```

</div>

### Customize severity thresholds

Modify the security review step to focus on specific severity levels:

<div v-pre>

```yaml
- id: security-review
  exec:
    source: |
      Security review focusing on CRITICAL and HIGH severity only.
      Skip informational findings.
      ...
```

</div>

## Related Use Cases

- [Test Generation](/use-cases/test-generation) - Generate tests for uncovered code

## Next Steps

- [Concepts: Contracts](/concepts/contracts) - Add validation to your pipelines
- [Concepts: Personas](/concepts/personas) - Understand persona capabilities

<style>
.use-case-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
}
.complexity-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 12px;
  text-transform: uppercase;
}
.complexity-badge.beginner {
  background: #dcfce7;
  color: #166534;
}
.complexity-badge.intermediate {
  background: #fef3c7;
  color: #92400e;
}
.complexity-badge.advanced {
  background: #fee2e2;
  color: #991b1b;
}
.category-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 500;
  border-radius: 12px;
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}
</style>
