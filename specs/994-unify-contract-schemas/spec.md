# refactor(contract): unify contract schemas to enable cross-pipeline artifact chaining

**Issue**: [#994](https://github.com/re-cinq/wave/issues/994)
**Original issue**: [#654](https://github.com/re-cinq/wave/issues/654)
**Labels**: audit
**Fidelity category**: partial (6 of 9 acceptance criteria unverified)

## Context

Issue #654 established the contract chaining architecture: shared schemas (`shared-findings.schema.json`, `shared-review-verdict.schema.json`), input validation via `schema_path` on `inject_artifacts`, and the `docs/guides/contract-chaining.md` guide. PR #676 merged this work.

A subsequent `wave-audit` pipeline found that 6 of the original 9 acceptance criteria remain unverified in the codebase.

## Audit Evidence

- `.wave/contracts/` exists with shared schemas
- `go test ./internal/contract/...` passes
- `docs/guides/contract-chaining.md` exists

## Unverified Acceptance Criteria

The following criteria were claimed complete but not verified by the audit scanner:

1. **Audit all existing contract schemas and identify overlapping/redundant definitions**
   - Severity enum fragmentation: 5+ incompatible severity vocabularies across schemas
   - `pr_url` defined independently in 4+ schemas
   - `findings` array redefined with incompatible shapes in 15+ schemas
   - `dead-code-scan.schema.json` is unreferenced by any pipeline (dead artifact)
   - Zero `$ref` cross-file references exist anywhere

2. **Define shared base schemas for common artifact types (PR URL, review result...)**
   - Only 2 shared schemas created (`shared-findings`, `shared-review-verdict`)
   - No `shared-pr-result.schema.json` despite `pr_url`/`pr_number`/`branch` repeating in 4+ schemas
   - No shared severity enum definition
   - `.wave/contracts/` and `internal/defaults/contracts/` copies have diverged:
     - `shared-findings`: `.wave/` has 4 extra type values, `additionalProperties: false`, `format: date-time`
     - 5 schemas in `.wave/contracts/` not promoted to `internal/defaults/contracts/`

3. **Ensure contract output of each pipeline step can be validated as input by the next step**
   - `input_validator.go` exists and works, but `schema_path` on `inject_artifacts` is used in only 1 pipeline (`full-impl-cycle.yaml`)
   - All other cross-step artifact injections have no input schema validation
   - Audit, review, and scope pipelines chain artifacts without input-side validation

4. **Additional unverified criteria** (truncated in audit issue):
   - Cross-pipeline artifact type compatibility not enforced
   - No migration path documented for existing custom schemas to adopt shared base types
   - Schema documentation/registry not maintained

## Acceptance Criteria (for this remediation)

- [ ] Complete schema audit: identify all overlapping/redundant definitions with a documented report
- [ ] Sync `.wave/contracts/` and `internal/defaults/contracts/` — single source of truth
- [ ] Define shared severity enum and extract into reusable `$defs` block or convention
- [ ] Create `shared-pr-result.schema.json` for PR URL/number/branch pattern
- [ ] Add `schema_path` on `inject_artifacts` for all cross-step artifact injections in audit/review/scope pipelines
- [ ] Remove or deprecate unreferenced schemas (e.g. `dead-code-scan.schema.json`)
- [ ] Update `docs/guides/contract-chaining.md` with new shared schemas and input validation guidance
- [ ] All existing tests pass (`go test ./internal/contract/...` and `go test ./...`)
- [ ] Add tests for input validation with shared schemas
