# Implementation Plan: Reusable Audit Pipelines

## Objective

Create four reusable audit pipelines (`audit-quality`, `audit-security`, `audit-deps`, `audit-flaws`) with a unified output schema, contract validation, and documentation. Each pipeline is a multi-step DAG that scans a project, verifies findings, and produces a final report.

## Approach

### Unified Audit Finding Schema

All four pipelines share a common output schema (`audit-findings.schema.json`) that standardizes the finding format across audit types. This ensures consistent tooling, reporting, and composability.

The shared schema defines:
- `target`: What was scanned
- `audit_type`: Category (quality, security, deps, flaws)
- `findings[]`: Array of findings, each with id, title, severity (CRITICAL/HIGH/MEDIUM/LOW), category, location, description, evidence, recommendation
- `summary`: Total counts, by_severity, by_category, risk_assessment
- `timestamp`: ISO 8601

### Pipeline Structure Pattern

Each audit pipeline follows a consistent 3-step pattern:

1. **scan** (persona: navigator, readonly mount) — Performs the initial analysis and produces structured JSON findings
2. **verify** (persona: auditor, readonly mount, depends: scan) — Reviews HIGH/CRITICAL findings against actual code, eliminates false positives
3. **report** (persona: summarizer, depends: verify) — Synthesizes verified findings into a final actionable report

### Relationship to Existing Pipelines

Existing pipelines (`security-scan`, `dead-code`, `doc-audit`) are kept as-is. The new `audit-*` pipelines are broader in scope and use the unified schema. No refactoring of existing pipelines is needed.

## File Mapping

### New Files to Create

| Path | Action | Description |
|------|--------|-------------|
| `.wave/contracts/audit-findings.schema.json` | create | Shared contract schema for all audit pipelines |
| `.wave/pipelines/audit-quality.yaml` | create | Code quality audit pipeline |
| `.wave/pipelines/audit-security.yaml` | create | Security audit pipeline |
| `.wave/pipelines/audit-deps.yaml` | create | Dependency health audit pipeline |
| `.wave/pipelines/audit-flaws.yaml` | create | Common flaws audit pipeline |
| `docs/use-cases/audit-pipelines.md` | create | Documentation with usage examples for all audit pipelines |
| `tests/pipeline_audit_test.go` | create | Integration tests validating audit pipeline schemas |

### Files to Modify

| Path | Action | Description |
|------|--------|-------------|
| None | — | No existing files need modification; existing pipelines remain untouched |

## Architecture Decisions

### AD-001: Shared vs Per-Pipeline Schema

**Decision**: Use a single shared `audit-findings.schema.json` with an `audit_type` discriminator field.

**Rationale**: Consistent tooling across all audit types. Downstream consumers can parse any audit output the same way. The `audit_type` and `category` fields provide enough differentiation.

**Trade-off**: Less type-specific validation (e.g., dependency findings can't enforce `current_version`/`latest_version` fields as required). Mitigated by using the `details` object for type-specific data.

### AD-002: Reuse Existing Personas

**Decision**: Reuse `navigator`, `auditor`, and `summarizer` personas rather than creating audit-specific ones.

**Rationale**: These personas already have the right permission models (navigator=readonly scan, auditor=security review, summarizer=report synthesis). Creating new personas would add maintenance burden without functional benefit.

### AD-003: Keep Existing Pipelines Separate

**Decision**: Do not refactor `security-scan`, `dead-code`, or `doc-audit` into the `audit-*` scheme.

**Rationale**: Existing pipelines have established users and different output schemas. The `audit-*` pipelines serve a different purpose (unified reporting) and can coexist.

### AD-004: Pipeline-Level Finding ID Prefixes

**Decision**: Each audit type uses a distinct ID prefix: `AQ-` (quality), `AS-` (security), `AD-` (deps), `AF-` (flaws).

**Rationale**: Enables cross-pipeline deduplication and clear provenance when findings from multiple audits are aggregated.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| LLM produces inconsistent JSON structure across runs | Medium | Medium | Contract schema validation with retry on failure (max 2 retries) |
| Scan step overwhelmed by large codebases | Low | Medium | Prompt instructs persona to limit to top N findings by severity |
| False positives in scan step | High | Low | Verify step explicitly checks HIGH/CRITICAL findings against code |
| Audit pipelines overlap with existing pipelines causing confusion | Medium | Low | Documentation clearly explains the difference and when to use each |

## Testing Strategy

### Unit Tests
- Schema validation: Test that the `audit-findings.schema.json` correctly validates sample outputs and rejects invalid ones
- Test valid and invalid finding structures against the schema

### Integration Tests
- Run each audit pipeline against the Wave project itself using `wave run audit-<category>`
- Verify output files exist and validate against the contract schema
- Verify the 3-step pipeline structure (scan → verify → report) executes correctly

### Contract Validation Tests
- Test that the schema is parseable by Wave's contract validation engine
- Test retry behavior when output doesn't match schema
