---
title: Security Audit
description: Comprehensive vulnerability scanning, dependency checks, and compliance verification
---

# Security Audit

<div class="use-case-meta">
  <span class="complexity-badge intermediate">Intermediate</span>
  <span class="category-badge">Security</span>
</div>

Perform comprehensive security analysis of your codebase. This pipeline performs deep vulnerability scanning, dependency auditing, compliance checking, and produces an executive security report.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Git repository to audit
- Understanding of [code-review](/use-cases/code-review) pipeline (recommended)
- Familiarity with OWASP Top 10 and common vulnerability types

## Quick Start

For basic security review, use the built-in `code-review` pipeline:

```bash
wave run code-review "security audit of the entire codebase"
```

For comprehensive security audits, create a dedicated pipeline (see below):

```bash
wave run security-audit "full security audit"
```

Expected output:

```
[10:00:01] started   inventory          (navigator)              Starting step
[10:00:35] completed inventory          (navigator)   34s   3.2k Analysis complete
[10:00:36] started   vulnerability-scan (auditor)                Starting step
[10:00:36] started   compliance-check   (auditor)                Starting step
[10:01:15] completed vulnerability-scan (auditor)     39s   4.5k Scan complete
[10:01:18] completed compliance-check   (auditor)     42s   3.8k Check complete
[10:01:19] started   report             (summarizer)             Starting step
[10:01:40] completed report             (summarizer)  21s   2.1k Report complete

Pipeline security-audit completed in 99s
Artifacts: output/security-report.md
```

## Complete Pipeline

Save the following YAML to `.wave/pipelines/security-audit.yaml`:

```yaml
kind: WavePipeline
metadata:
  name: security-audit
  description: "Comprehensive security audit with vulnerability scanning"

input:
  source: cli

steps:
  - id: inventory
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
        Create a security-relevant inventory of: {{ input }}

        Identify:
        1. Authentication and authorization code paths
        2. Data input points (APIs, forms, file uploads)
        3. Sensitive data handling (PII, credentials, tokens)
        4. External service integrations
        5. Cryptographic operations
        6. Dependencies with known vulnerabilities

        Output as JSON:
        {
          "auth_paths": [],
          "input_points": [],
          "sensitive_data": [],
          "external_services": [],
          "crypto_usage": [],
          "dependencies": []
        }
    output_artifacts:
      - name: inventory
        path: output/security-inventory.json
        type: json

  - id: vulnerability-scan
    persona: auditor
    dependencies: [inventory]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: inventory
          artifact: inventory
          as: targets
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Scan for vulnerabilities based on the security inventory.

        Check each category:
        1. **Injection**: SQL, NoSQL, OS command, LDAP
        2. **Authentication**: Weak passwords, missing MFA, session issues
        3. **Authorization**: IDOR, privilege escalation, missing checks
        4. **Data Exposure**: Logging secrets, unencrypted storage
        5. **Configuration**: Debug modes, default credentials, open ports
        6. **Dependencies**: CVEs in third-party packages

        For each finding, include:
        - Severity (CRITICAL/HIGH/MEDIUM/LOW)
        - File location and line number
        - Proof of concept or reproduction steps
        - Remediation recommendation
    output_artifacts:
      - name: vulnerabilities
        path: output/vulnerabilities.md
        type: markdown

  - id: compliance-check
    persona: auditor
    dependencies: [inventory]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: inventory
          artifact: inventory
          as: targets
    exec:
      type: prompt
      source: |
        Verify compliance with security standards.

        Check against:
        1. OWASP Top 10 (2021)
        2. CWE/SANS Top 25
        3. Industry best practices

        For each control:
        - Status: PASS / FAIL / PARTIAL / N/A
        - Evidence or gap description
        - Recommendation if not passing
    output_artifacts:
      - name: compliance
        path: output/compliance-report.md
        type: markdown

  - id: report
    persona: summarizer
    dependencies: [vulnerability-scan, compliance-check]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: vulnerability-scan
          artifact: vulnerabilities
          as: vulns
        - step: compliance-check
          artifact: compliance
          as: compliance
    exec:
      type: prompt
      source: |
        Create an executive security report.

        Include:
        1. Executive summary (1-2 paragraphs)
        2. Risk rating (Critical/High/Medium/Low)
        3. Top 5 findings requiring immediate attention
        4. Compliance summary
        5. Recommended remediation timeline
    output_artifacts:
      - name: report
        path: output/security-report.md
        type: markdown
```

## Expected Outputs

The pipeline produces four artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `inventory` | `output/security-inventory.json` | Security-relevant codebase inventory |
| `vulnerabilities` | `output/vulnerabilities.md` | Detailed vulnerability findings |
| `compliance` | `output/compliance-report.md` | Standards compliance status |
| `report` | `output/security-report.md` | Executive summary report |

### Example Output

The pipeline produces `output/security-report.md`:

```markdown
# Security Audit Report

**Date**: 2026-02-04
**Scope**: Full codebase
**Risk Rating**: HIGH

## Executive Summary

The security audit identified 3 critical and 7 high-severity vulnerabilities
requiring immediate attention. The codebase demonstrates good practices in
input validation but has significant gaps in authentication token handling
and dependency management.

## Top 5 Findings

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| 1 | CRITICAL | JWT secret hardcoded in source | config/auth.go:12 |
| 2 | CRITICAL | SQL injection in search endpoint | api/search.go:45 |
| 3 | CRITICAL | Missing rate limiting on auth endpoints | middleware/auth.go |
| 4 | HIGH | Outdated dependency with known CVE | go.mod (jwt-go v3.2.0) |
| 5 | HIGH | Sensitive data logged in debug mode | logger/debug.go:78 |

## Compliance Summary

| Standard | Status | Coverage |
|----------|--------|----------|
| OWASP Top 10 | PARTIAL | 7/10 controls passing |
| CWE Top 25 | PARTIAL | 18/25 controls passing |

## Remediation Timeline

| Priority | Items | Target |
|----------|-------|--------|
| Immediate (24h) | 3 critical findings | Day 1 |
| Short-term (1 week) | 7 high findings | Week 1 |
| Medium-term (1 month) | 12 medium findings | Month 1 |
```

## Customization

### Focus on specific areas

```bash
wave run security-audit "audit authentication and session management"
```

### Audit specific files

```bash
wave run security-audit "security review of internal/api/ directory"
```

### Add dependency scanning

Extend the pipeline with a dependency-focused step:

```yaml
- id: dependency-audit
  persona: navigator
  exec:
    type: prompt
    source: |
      Scan dependencies for known vulnerabilities.

      Run: go list -m all | grep -v "^module"
      Cross-reference with NVD/CVE databases.
      Check for outdated packages with security patches available.
```

### Add contract validation

Ensure compliance report follows expected format:

```yaml
- id: compliance-check
  handover:
    contract:
      type: json_schema
      schema_path: .wave/contracts/compliance.schema.json
      source: output/compliance-report.json
      on_failure: retry
      max_retries: 2
```

## Related Use Cases

- [Code Review](/use-cases/code-review) - Regular PR reviews with security checks
- [Test Generation](/use-cases/test-generation) - Generate security-focused tests
- [Incident Response](/use-cases/incident-response) - Respond to security incidents

## Next Steps

- [Concepts: Contracts](/concepts/contracts) - Validate audit output format
- [Concepts: Personas](/concepts/personas) - Understanding the auditor persona

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
