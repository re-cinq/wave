# Implementation Plan: Dead-Code Pipeline Multi-Mode Output

## Objective

Extend the dead-code detection system with two new pipeline variants (`dead-code-review` for PR comments, `dead-code-issue` for issue creation), enhance the contract schema with `suggested_action` and `duplicate_signature`, and add `requires.tools` preflight declarations to all dead-code pipelines.

## Approach

Create new pipeline YAML files that reuse the existing scan step pattern from `dead-code.yaml` but diverge at the output step. The `dead-code-review` pipeline scans PR-changed files and posts a review comment (modeled on `gh-pr-review.yaml`). The `dead-code-issue` pipeline runs a full scan and creates a GitHub issue (modeled on `doc-audit.yaml`). Schema changes are additive and backward-compatible.

## File Mapping

### New Files

| File | Purpose |
|------|---------|
| `.wave/pipelines/dead-code-review.yaml` | PR review comment pipeline variant |
| `.wave/pipelines/dead-code-issue.yaml` | Issue creation pipeline variant |
| `.wave/contracts/dead-code-issue-result.schema.json` | Contract for the issue creation step output |

### Modified Files

| File | Change |
|------|--------|
| `.wave/pipelines/dead-code.yaml` | Add `requires.tools` section (`go`) |
| `.wave/contracts/dead-code-scan.schema.json` | Add `suggested_action` enum and `duplicate_signature` type to findings |

## Architecture Decisions

### 1. Separate pipelines vs. modes within one pipeline

**Decision**: Separate pipeline files (`dead-code-review.yaml`, `dead-code-issue.yaml`).

**Rationale**: Wave pipelines are static DAGs. A single pipeline cannot conditionally skip steps based on runtime flags. Separate pipelines match the existing convention (e.g., `gh-pr-review.yaml` is separate from `gh-implement.yaml`). Each pipeline has different step graphs and different input expectations.

### 2. Schema changes are additive

**Decision**: Add `suggested_action` as an optional enum field and `duplicate_signature` as a new enum value for `type`.

**Rationale**: Existing scan results remain valid. The `dead-code.yaml` scan step already filters to high/medium confidence, so existing prompts will naturally start producing the new fields once the schema allows them. No migration needed.

### 3. PR review pipeline reuses `gh-pr-comment-result.schema.json`

**Decision**: Reuse the existing `gh-pr-comment-result.schema.json` contract for the publish step.

**Rationale**: The schema already defines `comment_url`, `pr_number`, `repository`, and `summary` — exactly what the dead-code review publish step needs. This avoids schema duplication.

### 4. Issue creation pipeline adapts `doc-issue-result.schema.json` pattern

**Decision**: Create a new `dead-code-issue-result.schema.json` based on the `doc-issue-result.schema.json` pattern but tailored with `finding_count` instead of `inconsistency_count`.

**Rationale**: The structure is nearly identical but the domain-specific field name should reflect dead-code findings, not documentation inconsistencies.

### 5. Personas for new pipelines

**Decision**:
- `dead-code-review`: `navigator` (scan), `summarizer` (compose), `github-commenter` (publish)
- `dead-code-issue`: `navigator` (scan), `navigator` (compose report), `craftsman` (create issue)

**Rationale**: Follows existing patterns. `github-commenter` is already used in `gh-pr-review.yaml` for posting PR comments. `craftsman` is used in `doc-audit.yaml` for issue creation (needs `Bash` for `gh` commands).

## Risks

| Risk | Mitigation |
|------|------------|
| New `duplicate_signature` type may produce low-quality results | Keep as optional detection; AI prompt already handles "redundant" category |
| PR review pipeline needs PR number from input parsing | Follow `gh-pr-review.yaml` pattern: pass PR URL/number as `{{ input }}` |
| Schema expansion breaks existing scan output validation | All new fields are optional; `type` enum is additive |
| `github-commenter` persona may not have `gh` tool access | Verify persona permissions; add `gh` to `requires.tools` |

## Testing Strategy

Since all changes are pipeline YAML definitions and JSON schemas:

1. **Schema validation**: Ensure the updated `dead-code-scan.schema.json` validates both old-format (without `suggested_action`) and new-format findings
2. **Pipeline structure**: Verify YAML is well-formed and parseable by running `go test ./...` (the pipeline loader validates structure)
3. **Contract compliance**: Ensure new schemas are valid JSON Schema draft-07
4. **Integration**: Manual pipeline execution tests (documented in tasks)
