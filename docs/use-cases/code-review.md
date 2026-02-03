# Code Review

Automate pull request reviews with security checks, quality analysis, and actionable feedback. Wave's code review pipeline analyzes changes, identifies issues, and produces a structured review summary.

## Quick Start

```bash
wave run --pipeline code-review --input "review the authentication module"
```

Expected output:

```
[10:00:01] started   diff-analysis     (navigator)              Starting step
[10:00:25] completed diff-analysis     (navigator)   24s   2.5k Analysis complete
[10:00:26] started   security-review   (auditor)                Starting step
[10:00:26] started   quality-review    (auditor)                Starting step
[10:00:45] completed security-review   (auditor)     19s   1.8k Review complete
[10:00:48] completed quality-review    (auditor)     22s   2.1k Review complete
[10:00:49] started   summary           (summarizer)             Starting step
[10:01:05] completed summary           (summarizer)  16s   1.2k Summary complete

Pipeline code-review completed in 64s
Artifacts: output/review-summary.md
```

## Complete Pipeline

This is the full `code-review` pipeline from `.wave/pipelines/code-review.yaml`:

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
        path: output/diff-analysis.json
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
        path: output/security-review.md
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
        path: output/quality-review.md
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
        path: output/review-summary.md
        type: markdown
```

## Example Output

The pipeline produces `output/review-summary.md`:

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
wave run --pipeline code-review --input "focus on error handling in the API layer"
```

### Review a specific PR

```bash
wave run --pipeline code-review --input "review changes in PR #123"
```

### Add contract validation

Add a JSON schema to ensure structured output:

```yaml
- id: diff-analysis
  handover:
    contract:
      type: json_schema
      schema: .wave/contracts/diff-analysis.schema.json
      source: output/diff-analysis.json
      on_failure: retry
      max_retries: 2
```

## Next Steps

- [Security Audit](/use-cases/security-audit) - Deep security analysis beyond code review
- [Test Generation](/use-cases/test-generation) - Generate tests for uncovered code
- [Concepts: Contracts](/guide/contracts) - Add validation to your pipelines
