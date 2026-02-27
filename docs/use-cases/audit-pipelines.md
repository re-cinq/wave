---
title: Audit Pipelines
description: Reusable audit pipelines for code quality, security, dependency health, and common flaws
---

# Audit Pipelines

<div class="use-case-meta">
  <span class="complexity-badge beginner">Beginner</span>
  <span class="category-badge">Code Quality</span>
  <span class="category-badge">Security</span>
</div>

A set of reusable audit pipelines that scan any project for quality issues, security vulnerabilities, outdated dependencies, and common code flaws. All pipelines produce structured JSON output using a unified schema.

## Prerequisites

- Wave installed and initialized (`wave init`)
- A project to audit (any language)

## Quick Start

```bash
# Run a code quality audit
wave run audit-quality "audit the entire project"

# Run a security audit
wave run audit-security "audit authentication and API modules"

# Run a dependency health audit
wave run audit-deps "audit project dependencies"

# Run a common flaws audit
wave run audit-flaws "scan for error handling gaps and TODOs"
```

## Available Pipelines

| Pipeline | Command | What It Checks |
|----------|---------|---------------|
| **Code Quality** | `wave run audit-quality` | Linting, formatting, complexity, dead code |
| **Security** | `wave run audit-security` | OWASP Top 10, secrets, dependency vulnerabilities, SAST |
| **Dependency Health** | `wave run audit-deps` | Outdated packages, deprecated dependencies, license compliance |
| **Common Flaws** | `wave run audit-flaws` | Error handling gaps, missing tests, TODO/FIXME tracking, API contract drift |

## Pipeline Structure

Each audit pipeline follows a consistent 3-step pattern:

```
scan (navigator) → verify (auditor) → report (summarizer)
```

1. **Scan**: The navigator persona reads the codebase (readonly) and produces structured findings
2. **Verify**: The auditor persona reviews HIGH/CRITICAL findings against actual code, eliminating false positives
3. **Report**: The summarizer persona synthesizes verified findings into a final actionable report

All three steps produce JSON output validated against the shared `audit-findings.schema.json` contract.

## Output Format

All audit pipelines produce JSON output conforming to the unified audit findings schema. The output file is written to `.wave/output/audit-<category>.json`.

### Schema Structure

```json
{
  "target": "internal/pipeline",
  "audit_type": "quality",
  "findings": [
    {
      "id": "AQ-001",
      "title": "High cyclomatic complexity",
      "severity": "HIGH",
      "category": "complexity",
      "location": "internal/pipeline/executor.go:42",
      "description": "Function has complexity of 25, exceeding threshold of 15",
      "evidence": "...",
      "recommendation": "Extract helper functions to reduce complexity",
      "details": {}
    }
  ],
  "summary": {
    "total_findings": 1,
    "by_severity": { "CRITICAL": 0, "HIGH": 1, "MEDIUM": 0, "LOW": 0 },
    "by_category": { "complexity": 1 },
    "risk_assessment": "Moderate risk: one high-complexity function identified"
  },
  "timestamp": "2026-02-27T18:00:00Z"
}
```

### Finding ID Prefixes

Each audit type uses a distinct prefix for finding IDs:

| Audit Type | Prefix | Example |
|-----------|--------|---------|
| Quality | `AQ-` | `AQ-001` |
| Security | `AS-` | `AS-001` |
| Dependencies | `AD-` | `AD-001` |
| Flaws | `AF-` | `AF-001` |

### Severity Levels

| Level | Meaning |
|-------|---------|
| `CRITICAL` | Must be addressed immediately; blocks release |
| `HIGH` | Should be addressed before next release |
| `MEDIUM` | Should be addressed; schedule for near-term |
| `LOW` | Informational; address when convenient |

## Audit Categories in Detail

### Code Quality (`audit-quality`)

Checks for:
- **Linting**: Style violations, anti-patterns, unused variables/imports
- **Formatting**: Inconsistent formatting, indentation, line lengths
- **Complexity**: High cyclomatic complexity, deep nesting, large functions
- **Dead Code**: Unused exports, unreachable code, commented-out blocks

```bash
wave run audit-quality "audit code quality of the internal/ packages"
```

### Security (`audit-security`)

Checks for:
- **OWASP Top 10**: Injection, broken auth, XSS, misconfigurations
- **Secrets**: Hardcoded API keys, tokens, passwords, private keys
- **Dependency Vulnerabilities**: Known CVEs, outdated security patches
- **SAST**: Unchecked errors, race conditions, path traversal, unsafe reflection

```bash
wave run audit-security "audit the authentication module for vulnerabilities"
```

### Dependency Health (`audit-deps`)

Checks for:
- **Outdated Packages**: Dependencies behind latest stable versions
- **Deprecated Dependencies**: Archived, abandoned, or unmaintained packages
- **License Compliance**: Copyleft conflicts, missing licenses, restrictive terms

```bash
wave run audit-deps "audit dependency health of the project"
```

### Common Flaws (`audit-flaws`)

Checks for:
- **Error Handling Gaps**: Ignored errors, empty catch blocks, missing nil checks
- **Missing Tests**: Public functions without tests, untested error paths
- **TODO/FIXME Tracking**: Untracked deferred work, orphaned TODOs
- **API Contract Drift**: Schema mismatches, stale contracts, missing validation

```bash
wave run audit-flaws "scan for common flaws in the internal/ packages"
```

## Audit vs Focused Pipelines

Wave includes both audit pipelines and focused pipelines. Use whichever fits your needs:

| Need | Use |
|------|-----|
| Broad security overview with unified output | `audit-security` |
| Deep-dive security scan with detailed remediation | `security-scan` |
| Broad quality check including formatting and complexity | `audit-quality` |
| Focused dead code detection with automatic removal | `dead-code` |
| Documentation consistency checks | `doc-audit` |

The audit pipelines are broader in scope and use a unified JSON output schema, making them ideal for CI/CD integration and automated reporting. Focused pipelines go deeper into specific areas and may produce different output formats.

## Composability

Audit pipelines can be composed into larger workflows. Run multiple audits and aggregate results:

```bash
# Run all audits
wave run audit-quality "full project audit"
wave run audit-security "full project audit"
wave run audit-deps "full project audit"
wave run audit-flaws "full project audit"
```

Since all pipelines use the same output schema, results from `.wave/output/audit-*.json` can be aggregated by external tools for unified reporting.

## Related Use Cases

- [Code Review](/use-cases/gh-pr-review) — Automated PR reviews
- [Documentation Consistency](/use-cases/doc-audit) — Check docs against code
- [Test Generation](/use-cases/test-generation) — Generate tests for uncovered code

## Next Steps

- [Concepts: Contracts](/concepts/contracts) — How contract validation works
- [Concepts: Personas](/concepts/personas) — Understand persona capabilities

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
