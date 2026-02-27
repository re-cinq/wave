# Tasks

## Phase 1: Foundation — Shared Contract Schema

- [X] Task 1.1: Create unified `audit-findings.schema.json` contract schema at `.wave/contracts/audit-findings.schema.json` with fields: target, audit_type, findings[], summary, timestamp. Finding items must include id, title, severity, category, location, description, evidence, recommendation, details.

## Phase 2: Core Pipelines

- [X] Task 2.1: Create `audit-quality` pipeline at `.wave/pipelines/audit-quality.yaml` with 3 steps (scan → verify → report). Scan step checks linting, formatting, complexity, dead code. Uses navigator persona for scan, auditor for verify, summarizer for report. [P]
- [X] Task 2.2: Create `audit-security` pipeline at `.wave/pipelines/audit-security.yaml` with 3 steps (scan → verify → report). Scan step checks OWASP Top 10, secrets, dependency vulnerabilities, SAST. Uses navigator for scan, auditor for verify, summarizer for report. [P]
- [X] Task 2.3: Create `audit-deps` pipeline at `.wave/pipelines/audit-deps.yaml` with 3 steps (scan → verify → report). Scan step checks outdated packages, deprecated deps, license compliance. Uses navigator for scan, auditor for verify, summarizer for report. [P]
- [X] Task 2.4: Create `audit-flaws` pipeline at `.wave/pipelines/audit-flaws.yaml` with 3 steps (scan → verify → report). Scan step checks error handling gaps, missing tests, TODO/FIXME items, API contract drift. Uses navigator for scan, auditor for verify, summarizer for report. [P]

## Phase 3: Testing

- [X] Task 3.1: Create schema validation tests in `tests/pipeline_audit_test.go` that verify `audit-findings.schema.json` accepts valid sample data and rejects invalid data
- [X] Task 3.2: Create integration test helpers that verify each audit pipeline YAML is valid and parseable by the manifest loader

## Phase 4: Documentation

- [X] Task 4.1: Create `docs/use-cases/audit-pipelines.md` with overview, usage examples for each pipeline, output format reference, and guidance on when to use audit-* vs existing focused pipelines (security-scan, dead-code)
- [X] Task 4.2: Add entries to `docs/use-cases/index.md` linking to the new audit pipelines documentation

## Phase 5: Validation

- [X] Task 5.1: Run `go test ./...` to verify no regressions
- [X] Task 5.2: Validate all new pipeline YAML files load correctly with `go vet ./...`
