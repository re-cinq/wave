# audit: partial — refactor(contract): unify contract schemas to enable cross-pipeline artifact chaining

**Issue**: [#994](https://github.com/re-cinq/wave/issues/994)
**Original Issue**: [#654](https://github.com/re-cinq/wave/issues/654) — refactor(contract): unify contract schemas to enable cross-pipeline artifact chaining
**Repository**: re-cinq/wave
**Labels**: audit
**Fidelity category**: partial
**Author**: nextlevelshit

## Description

Audit follow-up for the incomplete contract schema unification work from #654. The contract system has 87 JSON Schema files under `.wave/contracts/` and a working output-side validation system, but 6 of 9 acceptance criteria from the original issue remain unverified.

The core problem: contract schemas evolved independently per-pipeline, creating redundant inline definitions for common types (findings, severity enums, PR references, issue references) and inconsistent enum values across schemas that serve the same semantic purpose. Additionally, input-side artifact validation infrastructure exists in code (`internal/contract/input_validator.go`) but is not wired into any pipeline YAML.

## Evidence

- `.wave/contracts/` contains 87 schema files referenced across 41 of 51 pipeline YAMLs
- `internal/contract/input_validator.go` provides `ValidateInputArtifacts()` — fully implemented but unused in any pipeline YAML
- `go test ./internal/contract/...` passes — existing validator tests cover output-side validation

## Unverified Acceptance Criteria (6 of 9)

1. **Audit all existing contract schemas and identify overlapping/redundant definitions**
   - 5 overlap groups identified: findings items, severity enums, review verdicts, PR/GitHub references, issue references
2. **Define shared base schemas for common artifact types (PR URL, review result)**
   - No shared `$defs` or `$ref`-based reuse exists; all schemas define types inline
3. **Ensure contract output of each pipeline step can be validated as input by the next**
   - Output validation works; input-side `schema_path` on `inject_artifacts` is never set in YAML
4. **Normalize severity enum values across schemas**
   - 4 incompatible severity scales: `critical/high/medium/low/info`, `critical/major/minor/suggestion`, `critical/major/minor`, and UPPERCASE variants
5. **Deduplicate issue_reference object across research-findings, research-report, and comment-result schemas**
   - Same `{issue_number, repository, issue_url}` object defined inline in 3+ schemas
6. **Validate cross-pipeline artifact chaining end-to-end**
   - No integration test verifies that step A's output schema is compatible with step B's input schema

## Overlap Analysis

### Group A: Findings Item Shape
- `shared-findings.schema.json` — base finding: `{type, severity, package?, file?, line?, item?, description?, evidence?, recommendation?}`
- `aggregated-findings.schema.json` — identical item shape + `source_audit` required field
- Both use severity `critical/high/medium/low/info` and recommendation enum `remove/merge/keep/wire/document/investigate/fix/refactor`

### Group B: Severity Enum Inconsistency
| Schema | Severity Values |
|---|---|
| shared-findings, aggregated-findings, review-findings | `critical/high/medium/low/info` |
| shared-review-verdict | `critical/major/minor/suggestion` |
| review-findings (finding-level) | `critical/high/medium/low/info` |
| rework-gate-verdict | (uses counts, not per-finding severity) |

### Group C: Review Verdict Inconsistency
| Schema | Verdict Values |
|---|---|
| shared-review-verdict | `APPROVE/REQUEST_CHANGES/COMMENT/REJECT` |
| review-findings | `approved/changes_requested` |
| rework-gate-verdict | `pass/fail` |

### Group D: PR/GitHub References
- `pr_number` + `pr_url` defined inline in: `pr-result`, `gh-pr-comment-result`, `review-findings`, `triage-verdict`, `diff-analysis`
- `repository` pattern `^[^/]+/[^/]+$` defined inline in: `gh-pr-comment-result`, `comment-result`

### Group E: Issue Reference
- `{issue_number, repository, issue_url}` object defined inline in: `research-findings`, `research-report`, `comment-result`

## Links

- Original Issue #654: https://github.com/re-cinq/wave/issues/654
- This Issue #994: https://github.com/re-cinq/wave/issues/994
- Repository: re-cinq/wave
