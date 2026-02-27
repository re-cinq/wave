# Feature Specification: Reusable Audit Pipelines

**Feature Branch**: `182-audit-pipelines`
**Created**: 2026-02-27
**Status**: Draft
**Issue**: [#182](https://github.com/re-cinq/wave/issues/182)
**Labels**: enhancement, pipeline

## Summary

Create a set of reusable audit pipelines that can be applied to any project using Wave. These pipelines should cover common audit categories to help teams identify quality issues, security vulnerabilities, outdated dependencies, and common code flaws.

## Motivation

Currently there are no built-in audit pipelines that users can apply out of the box. Having a standard set of audit pipelines would provide immediate value for new Wave users and establish best-practice patterns for the community.

## Proposed Audit Categories

- **Code Quality** — linting, formatting, complexity analysis, dead code detection
- **Security** — dependency vulnerability scanning, secret detection, SAST checks
- **Dependency Health** — outdated packages, deprecated dependencies, license compliance
- **Common Flaws** — error handling gaps, missing tests, TODO/FIXME tracking, API contract drift

## Scope

These pipelines should be project-agnostic and configurable via `wave.yaml` manifest entries. They should work as standalone pipelines or be composable into larger CI/CD workflows.

## User Scenarios & Testing

### User Story 1 - Run a Code Quality Audit (Priority: P1)

A developer wants to quickly assess the code quality of their project by running `wave run audit-quality`. The pipeline scans the codebase for linting issues, complexity hotspots, formatting violations, and dead code, then produces a structured JSON report with findings.

**Why this priority**: Code quality is the most broadly applicable audit category and provides immediate value to every project.

**Independent Test**: Can be fully tested by running `wave run audit-quality` against the Wave project itself and verifying structured JSON output with findings.

**Acceptance Scenarios**:

1. **Given** a Go project with Wave configured, **When** the user runs `wave run audit-quality`, **Then** a JSON report with findings categorized by type (lint, complexity, formatting, dead_code) is produced at `.wave/output/audit-quality.json`.
2. **Given** an empty or clean project, **When** the user runs `wave run audit-quality`, **Then** the report contains zero findings and a clean summary.

---

### User Story 2 - Run a Security Audit (Priority: P1)

A developer wants to audit their project for security vulnerabilities by running `wave run audit-security`. The pipeline performs dependency vulnerability scanning, secret detection, and SAST checks, producing a structured report.

**Why this priority**: Security auditing is critical and the existing `security-scan` pipeline provides a foundation to build on.

**Independent Test**: Can be tested by running `wave run audit-security` against a project and verifying the output matches the unified audit finding schema.

**Acceptance Scenarios**:

1. **Given** a project with dependencies, **When** the user runs `wave run audit-security`, **Then** a JSON report with security findings including severity, category, and recommendations is produced.
2. **Given** findings with severity HIGH or CRITICAL, **When** the deep-dive step runs, **Then** each finding is verified against actual source code and false positives are eliminated.

---

### User Story 3 - Run a Dependency Health Audit (Priority: P2)

A developer wants to check the health of their project dependencies by running `wave run audit-deps`. The pipeline checks for outdated packages, deprecated dependencies, and license compliance issues.

**Why this priority**: Dependency health is important for long-term maintainability but is less urgent than quality and security.

**Independent Test**: Can be tested by running `wave run audit-deps` against a project with a `go.mod` or `package.json`.

**Acceptance Scenarios**:

1. **Given** a Go project with outdated dependencies, **When** the user runs `wave run audit-deps`, **Then** the report lists outdated packages with current and latest versions.
2. **Given** dependencies with problematic licenses, **When** the audit runs, **Then** license compliance issues are flagged.

---

### User Story 4 - Run a Common Flaws Audit (Priority: P2)

A developer wants to scan for common code flaws by running `wave run audit-flaws`. The pipeline checks for error handling gaps, missing tests, TODO/FIXME tracking, and API contract drift.

**Why this priority**: Common flaws detection adds polish but depends on patterns established by the other audit pipelines.

**Independent Test**: Can be tested by running `wave run audit-flaws` against a project with known TODO comments and missing test coverage.

**Acceptance Scenarios**:

1. **Given** a project with TODO/FIXME comments, **When** the user runs `wave run audit-flaws`, **Then** the report includes a list of tracked TODO/FIXME items with file locations.
2. **Given** a project with functions lacking error handling, **When** the audit runs, **Then** error handling gaps are flagged with locations and recommendations.

---

### Edge Cases

- What happens when the project has no source files matching the expected language?
- How does the system handle projects with no dependency manifest (no go.mod, package.json, etc.)?
- What happens when external tools (linters, vulnerability scanners) are not installed?
- How does the system handle very large codebases with thousands of findings?

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide at least one audit pipeline per category: `audit-quality`, `audit-security`, `audit-deps`, `audit-flaws`
- **FR-002**: Each pipeline MUST produce structured JSON output conforming to a contract schema
- **FR-003**: All audit pipelines MUST use a consistent output format with findings, severity, and recommendations
- **FR-004**: Pipelines MUST be runnable via `wave run audit-<category>`
- **FR-005**: Each pipeline MUST have a corresponding contract schema in `.wave/contracts/`
- **FR-006**: Pipelines MUST work as standalone or composable into larger workflows
- **FR-007**: Pipelines MUST be project-agnostic (not hardcoded to Go or any specific language)
- **FR-008**: Each pipeline MUST have documentation with usage examples
- **FR-009**: Pipelines MUST reuse existing personas (navigator, auditor, summarizer) where appropriate

### Key Entities

- **Audit Finding**: A single finding with id, title, severity, category, location, description, evidence, recommendation
- **Audit Report**: Collection of findings with summary statistics (total, by_severity, by_category)
- **Audit Category**: One of quality, security, deps, flaws

## Existing Assets

The following existing pipelines overlap with the requested audit categories:

- `security-scan.yaml` — covers security vulnerability scanning (overlaps with `audit-security`)
- `dead-code.yaml` — covers dead code detection (overlaps with `audit-quality`)
- `doc-audit.yaml` — covers documentation consistency (related pattern, different scope)

**Decision**: Keep existing pipelines as-is. The new `audit-*` pipelines are broader in scope and use a unified output schema. Existing pipelines remain available for users who prefer the focused versions.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All four audit pipelines (`audit-quality`, `audit-security`, `audit-deps`, `audit-flaws`) can be invoked and produce valid JSON output
- **SC-002**: Output from all audit pipelines validates against their respective contract schemas
- **SC-003**: Each pipeline is documented in `docs/use-cases/` with usage examples
- **SC-004**: Integration tests validate each pipeline against the Wave project itself as a sample
