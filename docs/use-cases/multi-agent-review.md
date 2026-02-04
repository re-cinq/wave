---
title: Multi-Agent Review
description: Parallel specialized reviews combining security, performance, and architecture analysis
---

# Multi-Agent Review

<div class="use-case-meta">
  <span class="complexity-badge advanced">Advanced</span>
  <span class="category-badge">Code Quality</span>
</div>

Perform comprehensive code review using multiple specialized personas working in parallel. This advanced pipeline combines security, performance, architecture, and maintainability analysis into a unified review verdict.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Experience with [code-review](/use-cases/code-review) pipeline
- Understanding of parallel step execution
- Familiarity with artifact injection patterns

## Quick Start

```bash
wave run multi-review "comprehensive review of the new payment processing module"
```

Expected output:

```
[10:00:01] started   scope-analysis      (navigator)              Starting step
[10:00:28] completed scope-analysis      (navigator)   27s   2.5k Analysis complete
[10:00:29] started   security-review     (auditor)                Starting step
[10:00:29] started   performance-review  (auditor)                Starting step
[10:00:29] started   architecture-review (philosopher)            Starting step
[10:00:29] started   maintainability     (auditor)                Starting step
[10:00:55] completed security-review     (auditor)     26s   3.2k Review complete
[10:00:58] completed performance-review  (auditor)     29s   2.8k Review complete
[10:01:02] completed architecture-review (philosopher)  33s   3.5k Review complete
[10:01:05] completed maintainability     (auditor)     36s   2.9k Review complete
[10:01:06] started   synthesis           (summarizer)             Starting step
[10:01:32] completed synthesis           (summarizer)  26s   2.1k Synthesis complete

Pipeline multi-review completed in 91s
Artifacts: output/multi-review-verdict.md
```

## Complete Pipeline

Save the following YAML to `.wave/pipelines/multi-review.yaml`:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: multi-review
  description: "Comprehensive multi-agent code review"

input:
  source: cli

steps:
  - id: scope-analysis
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
        Analyze the scope for comprehensive review: {{ input }}

        Map:
        1. All files and modules affected
        2. External dependencies involved
        3. Data flow through the system
        4. Integration points
        5. Critical paths requiring deep review

        Output as JSON:
        {
          "files": [],
          "modules": [],
          "dependencies": [],
          "data_flows": [],
          "integration_points": [],
          "critical_paths": []
        }
    output_artifacts:
      - name: scope
        path: output/scope-analysis.json
        type: json

  - id: security-review
    persona: auditor
    dependencies: [scope-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: scope-analysis
          artifact: scope
          as: scope
    exec:
      type: prompt
      source: |
        Deep security analysis of the changes.

        Focus areas:
        1. Authentication and authorization
        2. Input validation and sanitization
        3. Cryptographic operations
        4. Secret management
        5. SQL/NoSQL injection vectors
        6. Cross-site scripting (XSS)
        7. Cross-site request forgery (CSRF)
        8. Insecure deserialization
        9. Dependency vulnerabilities

        For each finding:
        - Severity: CRITICAL / HIGH / MEDIUM / LOW
        - Location: file:line
        - Description
        - Proof of concept
        - Remediation
    output_artifacts:
      - name: security
        path: output/security-findings.md
        type: markdown

  - id: performance-review
    persona: auditor
    dependencies: [scope-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: scope-analysis
          artifact: scope
          as: scope
    exec:
      type: prompt
      source: |
        Performance analysis of the changes.

        Analyze:
        1. Algorithmic complexity (Big O)
        2. Database query efficiency
        3. Memory allocation patterns
        4. Concurrency and locking
        5. I/O operations and buffering
        6. Caching opportunities
        7. Resource cleanup

        For each finding:
        - Impact: HIGH / MEDIUM / LOW
        - Location: file:line
        - Current behavior
        - Recommended optimization
        - Expected improvement
    output_artifacts:
      - name: performance
        path: output/performance-findings.md
        type: markdown

  - id: architecture-review
    persona: philosopher
    dependencies: [scope-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: scope-analysis
          artifact: scope
          as: scope
    exec:
      type: prompt
      source: |
        Architectural analysis of the changes.

        Evaluate:
        1. Separation of concerns
        2. Dependency direction (clean architecture)
        3. Interface design and contracts
        4. Error handling patterns
        5. Extensibility and flexibility
        6. Testability
        7. Consistency with existing patterns

        For each observation:
        - Type: STRENGTH / CONCERN / SUGGESTION
        - Description
        - Impact on maintainability
        - Recommendation if applicable
    output_artifacts:
      - name: architecture
        path: output/architecture-findings.md
        type: markdown

  - id: maintainability
    persona: auditor
    dependencies: [scope-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: scope-analysis
          artifact: scope
          as: scope
    exec:
      type: prompt
      source: |
        Maintainability analysis of the changes.

        Check:
        1. Code readability and clarity
        2. Documentation completeness
        3. Naming conventions
        4. Function/method length
        5. Cyclomatic complexity
        6. Test coverage and quality
        7. Error messages and logging

        For each finding:
        - Category: READABILITY / DOCUMENTATION / COMPLEXITY / TESTING
        - Location: file:line
        - Issue description
        - Suggested improvement
    output_artifacts:
      - name: maintainability
        path: output/maintainability-findings.md
        type: markdown

  - id: synthesis
    persona: summarizer
    dependencies: [security-review, performance-review, architecture-review, maintainability]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: security-review
          artifact: security
          as: security
        - step: performance-review
          artifact: performance
          as: performance
        - step: architecture-review
          artifact: architecture
          as: architecture
        - step: maintainability
          artifact: maintainability
          as: maintainability
    exec:
      type: prompt
      source: |
        Synthesize all review findings into a comprehensive verdict.

        Structure:
        1. Executive Summary (2-3 sentences)
        2. Overall Verdict: APPROVE / REQUEST_CHANGES / NEEDS_DISCUSSION
        3. Risk Assessment: CRITICAL / HIGH / MEDIUM / LOW
        4. Blocking Issues (must fix before merge)
        5. Important Issues (should fix soon)
        6. Minor Issues (nice to have)
        7. Strengths and Positive Observations
        8. Recommended Next Steps

        Prioritize issues by impact and effort to fix.
    output_artifacts:
      - name: verdict
        path: output/multi-review-verdict.md
        type: markdown
```

</div>

## Expected Outputs

The pipeline produces six artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `scope` | `output/scope-analysis.json` | Review scope and critical paths |
| `security` | `output/security-findings.md` | Security analysis findings |
| `performance` | `output/performance-findings.md` | Performance analysis findings |
| `architecture` | `output/architecture-findings.md` | Architecture analysis |
| `maintainability` | `output/maintainability-findings.md` | Maintainability findings |
| `verdict` | `output/multi-review-verdict.md` | Synthesized review verdict |

### Example Output

The pipeline produces `output/multi-review-verdict.md`:

```markdown
# Multi-Agent Code Review: Payment Processing Module

## Executive Summary

The payment processing module introduces solid foundational architecture with
good separation of concerns. However, critical security issues around token
handling and moderate performance concerns with database queries require
attention before merging.

## Overall Verdict: REQUEST_CHANGES

## Risk Assessment: HIGH

Due to 2 critical security findings in payment token handling.

## Blocking Issues (Must Fix)

### Security

1. **Payment token stored in plain text** (CRITICAL)
   - File: `payment/token.go:45`
   - Tokens must be encrypted at rest
   - Fix: Use AES-256-GCM encryption

2. **Missing rate limiting on payment endpoint** (CRITICAL)
   - File: `api/payment_handler.go:78`
   - Vulnerable to abuse and fraud
   - Fix: Add rate limiting middleware

### Performance

3. **N+1 query in order retrieval** (HIGH)
   - File: `payment/orders.go:112`
   - 50+ queries for 50 orders
   - Fix: Use JOIN or batch loading

## Important Issues (Should Fix)

4. **Missing retry logic for payment provider** (MEDIUM)
   - File: `payment/provider.go:89`
   - Transient failures cause immediate errors

5. **Insufficient error context** (MEDIUM)
   - File: `payment/errors.go`
   - Error messages lack debugging info

## Minor Issues (Nice to Have)

6. Function `processPayment` exceeds 50 lines
7. Missing godoc for exported types
8. Test coverage at 65% (target: 80%)

## Strengths

- Clean interface design for payment providers
- Good use of context for cancellation
- Comprehensive input validation
- Well-structured error types

## Recommended Next Steps

1. Fix blocking security issues (Day 1)
2. Address performance N+1 query (Day 2)
3. Add retry logic and error context (Week 1)
4. Improve test coverage (Week 2)
```

## Customization

### Focus on specific aspects

```bash
wave run multi-review "focus on security and performance for the auth module"
```

### Add compliance review

Add a compliance-focused step:

<div v-pre>

```yaml
- id: compliance-review
  persona: auditor
  dependencies: [scope-analysis]
  exec:
    source: |
      Check compliance with:
      - PCI DSS (for payment handling)
      - GDPR (for personal data)
      - SOC 2 controls
```

</div>

### Weighted scoring

Add scoring to the synthesis step:

<div v-pre>

```yaml
- id: synthesis
  exec:
    source: |
      Score each dimension (0-10):
      - Security: weight 3x
      - Performance: weight 2x
      - Architecture: weight 2x
      - Maintainability: weight 1x

      Calculate weighted total and grade.
```

</div>

## Pipeline Visualization

```
                    +-----------------+
                    | scope-analysis  |
                    |   (navigator)   |
                    +--------+--------+
                             |
        +--------------------+--------------------+
        |                    |                    |
        v                    v                    v
+---------------+   +----------------+   +------------------+
| security      |   | performance    |   | architecture     |
|   (auditor)   |   |   (auditor)    |   |   (philosopher)  |
+-------+-------+   +-------+--------+   +--------+---------+
        |                   |                     |
        |                   v                     |
        |           +----------------+            |
        |           | maintainability|            |
        |           |   (auditor)    |            |
        |           +-------+--------+            |
        |                   |                     |
        +-------------------+---------------------+
                            |
                            v
                    +---------------+
                    |   synthesis   |
                    | (summarizer)  |
                    +---------------+
```

## Related Use Cases

- [Code Review](/use-cases/code-review) - Simpler single-pass review
- [Security Audit](/use-cases/security-audit) - Deep security-only analysis
- [Refactoring](/use-cases/refactoring) - Follow up on architecture findings

## Next Steps

- [Concepts: Pipelines](/concepts/pipelines) - Understanding parallel execution
- [Concepts: Artifacts](/concepts/artifacts) - How artifacts flow between steps

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
