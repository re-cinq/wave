# Implementation Plan: Unify Contract Schemas

## Objective

Complete the 6 unverified acceptance criteria from #654 by: (1) creating shared base schemas with `$defs` for reusable types, (2) refactoring existing schemas to `$ref` these shared definitions, (3) normalizing severity and verdict enums, (4) enabling input-side artifact validation in pipeline YAMLs, and (5) adding integration tests for cross-pipeline artifact chaining.

## Approach

### Strategy: Incremental $ref Extraction with Validator Enhancement

JSON Schema draft-07 supports `$ref` across files. The `santhosh-tekuri/jsonschema/v6` library used by the validator can resolve `$ref` if referenced schemas are pre-registered via `compiler.AddResource()`.

**Phase 1**: Create shared definition schemas under `.wave/contracts/_defs/` containing common sub-schemas.

**Phase 2**: Enhance the `jsonSchemaValidator` to auto-load `_defs/` schemas into the compiler before compiling the target schema, enabling cross-file `$ref` resolution.

**Phase 3**: Refactor existing schemas to `$ref` shared definitions instead of inline duplicates. Normalize enums during this process.

**Phase 4**: Wire input-side `schema_path` into key pipeline YAMLs for artifact chain validation.

**Phase 5**: Add integration tests that validate step A's output against step B's expected input schema.

### Shared Definition Schemas

Create these files under `.wave/contracts/_defs/`:

1. **`severity.schema.json`** — Canonical severity enum: `critical/high/medium/low/info` (the most widely used scale). Include a `review_severity` variant: `critical/major/minor/suggestion` for review-specific schemas.

2. **`finding.schema.json`** — Base finding item from `shared-findings`: `{type, severity, package?, file?, line?, item?, description?, evidence?, recommendation?}`. The `aggregated-findings` schema extends this with `source_audit`.

3. **`pr-reference.schema.json`** — Common PR fields: `{pr_number: integer, pr_url: uri-string}`.

4. **`issue-reference.schema.json`** — Common issue reference: `{issue_number: integer, repository: pattern, issue_url?: uri-string}`.

5. **`github-result.schema.json`** — Common GitHub operation result fields: `{repository?: pattern, summary?: string}`.

### Enum Normalization Strategy

- **Severity**: Standardize on `critical/high/medium/low/info` for findings schemas. Keep `critical/major/minor/suggestion` for review-verdict schemas (different semantic domain). Document the mapping: `high→major`, `medium→minor`, `low/info→suggestion`.
- **Verdict**: Leave verdict enums domain-specific (`APPROVE/REQUEST_CHANGES/COMMENT/REJECT` for reviews, `pass/fail` for gates, `approved/changes_requested` for fix-review). These serve different pipeline stages with different semantics — forcing unification would lose information.

### Input Validation Wiring

Add `schema_path` to `inject_artifacts` entries in these high-value chains:
- `impl-issue.yaml`: assessment→plan, plan→implement
- `impl-issue-core.yaml`: assessment→plan
- `ops-pr-review-core.yaml`: diff→reviews
- `ops-pr-fix-review.yaml`: review-findings→triage

## File Mapping

### Create
| Path | Purpose |
|---|---|
| `.wave/contracts/_defs/severity.schema.json` | Canonical severity enum definitions |
| `.wave/contracts/_defs/finding.schema.json` | Base finding item schema |
| `.wave/contracts/_defs/pr-reference.schema.json` | Common PR reference fields |
| `.wave/contracts/_defs/issue-reference.schema.json` | Common issue reference object |
| `internal/contract/jsonschema_refs_test.go` | Tests for $ref resolution |
| `internal/contract/chain_test.go` | Integration tests for artifact chaining |

### Modify
| Path | Purpose |
|---|---|
| `internal/contract/jsonschema.go` | Pre-load `_defs/` schemas into compiler for $ref resolution |
| `internal/contract/input_validator.go` | Pre-load `_defs/` schemas (same pattern) |
| `.wave/contracts/shared-findings.schema.json` | $ref finding and severity from `_defs/` |
| `.wave/contracts/aggregated-findings.schema.json` | $ref finding base, add source_audit extension |
| `.wave/contracts/shared-review-verdict.schema.json` | $ref review severity from `_defs/` |
| `.wave/contracts/review-findings.schema.json` | $ref pr-reference from `_defs/` |
| `.wave/contracts/pr-result.schema.json` | $ref pr-reference from `_defs/` |
| `.wave/contracts/gh-pr-comment-result.schema.json` | $ref pr-reference from `_defs/` |
| `.wave/contracts/comment-result.schema.json` | $ref issue-reference from `_defs/` |
| `.wave/contracts/research-findings.schema.json` | $ref issue-reference from `_defs/` |
| `.wave/contracts/research-report.schema.json` | $ref issue-reference from `_defs/` |
| `.wave/contracts/triage-verdict.schema.json` | $ref pr-reference from `_defs/` |
| `internal/defaults/pipelines/impl-issue.yaml` | Add input schema_path for artifact chains |
| `internal/defaults/pipelines/impl-issue-core.yaml` | Add input schema_path for artifact chains |
| `internal/defaults/pipelines/ops-pr-review-core.yaml` | Add input schema_path for diff→review chain |
| `internal/defaults/pipelines/ops-pr-fix-review.yaml` | Add input schema_path for review→triage chain |

## Architecture Decisions

1. **Cross-file `$ref` over inline `$defs`**: Using separate files under `_defs/` is cleaner than copying `$defs` into every schema. Requires a small validator enhancement but centralizes definitions properly.

2. **`_defs/` naming convention**: Underscore prefix signals these are internal shared definitions, not standalone contract schemas. They are never referenced directly by `schema_path` in pipeline YAML.

3. **Keep verdict enums domain-specific**: Review verdicts (`APPROVE`/`REQUEST_CHANGES`), gate verdicts (`pass`/`fail`), and fix-review verdicts (`approved`/`changes_requested`) serve different pipeline stages. Forcing a single enum would lose semantic precision.

4. **Two severity scales**: `critical/high/medium/low/info` for scan/findings (5-level) and `critical/major/minor/suggestion` for reviews (4-level). Document the mapping but don't force unification — they serve different purposes.

5. **Input validation opt-in**: Wire `schema_path` on `inject_artifacts` only for high-value chains where type mismatches have caused issues. Don't add it everywhere — that would slow all pipelines for marginal benefit.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| `$ref` resolution breaks existing pipelines | High — all schema validation fails | Test `$ref` resolution in isolation first; keep original schemas as backup until verified |
| `santhosh-tekuri/jsonschema/v6` doesn't resolve relative file `$ref` as expected | Medium — need different approach | Validate with unit test before refactoring schemas; fallback to `compiler.AddResource()` with explicit URIs |
| Input validation blocks pipelines on minor schema drift | Medium — pipelines fail on previously-passing artifacts | Use `must_pass: false` initially for input validation; promote to `must_pass: true` after burn-in |
| Schema changes break existing artifact producers (AI prompts) | Low — schemas are loosened not tightened | All changes use `$ref` to existing definitions; no fields are removed or enum values changed |

## Testing Strategy

1. **Unit tests** (`internal/contract/jsonschema_refs_test.go`):
   - Verify `$ref` resolution works with `_defs/` schemas pre-loaded
   - Test that refactored schemas validate the same artifacts as originals
   - Test invalid artifacts still fail validation

2. **Chain integration tests** (`internal/contract/chain_test.go`):
   - Create sample artifacts matching step A's output schema
   - Validate them against step B's input/expected schema
   - Cover: assessment→plan, plan→implement, diff→review, review→triage chains

3. **Regression**: Run `go test ./internal/contract/...` after each schema modification to catch breakage early

4. **Pipeline smoke test**: Run `wave run ops-hello-world` to verify the validator enhancement doesn't break basic pipeline execution
