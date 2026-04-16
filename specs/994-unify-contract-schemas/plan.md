# Implementation Plan: Unify Contract Schemas

## Objective

Complete the contract schema unification started in #654 by resolving schema divergence between `.wave/contracts/` and `internal/defaults/contracts/`, standardizing severity enums, extracting shared definitions for common artifact types (PR result, findings), wiring input-side schema validation on artifact injections across all chaining pipelines, and removing dead schemas.

## Approach

This is a schema-layer refactoring — no Go runtime logic changes are needed (input_validator.go already works correctly). The work is:

1. **Sync schemas** — make `.wave/contracts/` authoritative, copy to `internal/defaults/contracts/`
2. **Standardize severity** — adopt a single canonical severity enum across all schemas
3. **Extract shared base patterns** — create `shared-pr-result.schema.json` for PR output artifacts
4. **Wire input validation** — add `schema_path:` to `inject_artifacts` entries across pipelines
5. **Clean dead schemas** — remove unreferenced `dead-code-scan.schema.json`
6. **Update docs** — extend contract-chaining guide
7. **Promote schemas** — copy remaining `.wave/`-only schemas to `internal/defaults/contracts/`

## File Mapping

### Schemas to Modify

| File | Action | Change |
|------|--------|--------|
| `internal/defaults/contracts/shared-findings.schema.json` | modify | Sync with `.wave/contracts/` version (add 4 type values, `additionalProperties`, `format`) |
| `.wave/contracts/shared-review-verdict.schema.json` | modify | Add `additionalProperties: false` |
| `internal/defaults/contracts/shared-review-verdict.schema.json` | modify | Same sync |
| `.wave/contracts/security-scan.schema.json` | modify | Normalize severity to lowercase enum |
| `.wave/contracts/doc-fix-scan.schema.json` | modify | Normalize severity to lowercase enum |
| `.wave/contracts/triage-verdict.schema.json` | modify | Add `"info"` and align severity enum |
| `.wave/contracts/feature-exploration.schema.json` | modify | Add `"critical"`, `"info"` to risk severity |
| `.wave/contracts/review-findings.schema.json` | modify | Already aligned — verify |

### Schemas to Create

| File | Action | Purpose |
|------|--------|---------|
| `.wave/contracts/shared-pr-result.schema.json` | create | Shared PR URL/number/branch/summary pattern |
| `internal/defaults/contracts/shared-pr-result.schema.json` | create | Embedded default copy |

### Schemas to Delete

| File | Action | Reason |
|------|--------|--------|
| `.wave/contracts/dead-code-scan.schema.json` | delete | Unreferenced by any pipeline; superseded by `shared-findings.schema.json` |
| `internal/defaults/contracts/dead-code-scan.schema.json` | delete | Same |

### Schemas to Promote (copy .wave/ → internal/defaults/)

| File | Action |
|------|--------|
| `internal/defaults/contracts/aggregated-findings.schema.json` | create (copy from `.wave/contracts/`) |
| `internal/defaults/contracts/rework-gate-verdict.schema.json` | create (copy from `.wave/contracts/`) |
| `internal/defaults/contracts/config-check.schema.json` | create (copy from `.wave/contracts/`) |
| `internal/defaults/contracts/smoke-review.schema.json` | create (copy from `.wave/contracts/`) |
| `internal/defaults/contracts/wave-smoke-test.schema.json` | create (copy from `.wave/contracts/`) |

### Pipeline YAMLs to Modify (add input `schema_path:`)

| Pipeline | Step | Injected Artifact | Schema to Validate Against |
|----------|------|-------------------|---------------------------|
| `audit-architecture.yaml` | report | scan_findings | `shared-findings.schema.json` |
| `audit-tests.yaml` | report | scan_findings | `shared-findings.schema.json` |
| `audit-correctness.yaml` | report | scan_findings | `shared-findings.schema.json` |
| `audit-duplicates.yaml` | report | duplicate_findings | `shared-findings.schema.json` |
| `audit-doc.yaml` | normalize | scan_results | `doc-scan-results.schema.json` |
| `audit-doc.yaml` | publish | findings | `shared-findings.schema.json` |
| `audit-doc-scan.yaml` | report | scan_findings | `shared-findings.schema.json` |
| `audit-consolidate.yaml` | consolidate | (multiple inputs) | `shared-findings.schema.json` |
| `audit-dead-code-issue.yaml` | verify | scan_findings | `shared-findings.schema.json` |
| `ops-pr-fix-review.yaml` | triage | raw_findings | `review-findings.schema.json` |
| `ops-pr-review.yaml` | publish | verdict (if injected) | `shared-review-verdict.schema.json` |

### Documentation

| File | Action | Change |
|------|--------|--------|
| `docs/guides/contract-chaining.md` | modify | Add shared-pr-result, severity conventions, input validation section |
| `internal/contract/README.md` | modify | Update with input validation examples |

### Tests

| File | Action | Change |
|------|--------|--------|
| `internal/contract/input_validator_test.go` | modify | Add tests for shared schema validation |

## Architecture Decisions

1. **No `$ref` across files** — JSON Schema `$ref` with external file URIs requires the validator to resolve relative paths, and the current `jsonschema/v6` setup compiles schemas individually. Adding cross-file `$ref` would require changes to the compiler setup in both `jsonschema.go` and `input_validator.go`. Instead, we standardize by convention (shared severity enum values, naming conventions) and use copy-sync to keep schemas consistent. This is pragmatic for the current scale (~85 schemas).

2. **`.wave/contracts/` is authoritative** — the embedded `internal/defaults/contracts/` is a distribution mechanism. Changes flow `.wave/` → `internal/defaults/`. A CI check or test should verify sync.

3. **Severity enum standardization** — canonical set: `["critical", "high", "medium", "low", "info"]`. The `shared-review-verdict` uses `["critical", "major", "minor", "suggestion"]` which is semantically distinct (review severity vs scan severity) and kept separate intentionally.

4. **Input validation is opt-in** — `schema_path` on `inject_artifacts` is only added where the producing step has a matching output contract schema. Steps that produce markdown or unstructured output are not candidates.

## Risks

| Risk | Mitigation |
|------|------------|
| Changing severity enums breaks existing pipeline outputs | Only normalize casing (CRITICAL→critical); don't remove values. Add missing values to enums. |
| Removing `dead-code-scan.schema.json` breaks something | Grep confirms zero references. Safe to delete. |
| Adding `schema_path` on injections causes pipeline failures | Use `on_failure: warn` or verify schema compatibility before wiring. Run affected pipelines. |
| Schema sync between `.wave/` and `internal/defaults/` drifts again | Add a test that compares the two directories for shared schemas. |

## Testing Strategy

1. **Unit tests**: `go test ./internal/contract/...` — existing + new input validation tests
2. **Schema validation**: Manually validate that each modified schema is valid JSON Schema Draft-07
3. **Full test suite**: `go test ./...` to catch any regressions
4. **Sync test**: New test comparing `.wave/contracts/shared-*.schema.json` with `internal/defaults/contracts/shared-*.schema.json`
5. **Pipeline smoke**: Run an audit pipeline (e.g. `audit-architecture`) to verify input validation wiring works end-to-end
