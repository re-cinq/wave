# Contract Chaining for Cross-Pipeline Composition

When composing multiple pipelines in sequence, the output contract of one pipeline must be structurally compatible with the input expectations of the next. Wave provides **shared contract schemas** to make these handover points explicit and validated.

## Shared Schemas

Shared schemas live in `.wave/contracts/shared-*.schema.json` and are designed to be referenced by multiple pipelines:

| Schema | Purpose | Used by |
|--------|---------|---------|
| `shared-findings.schema.json` | Standardized audit/analysis findings | All `audit-*` pipelines |
| `shared-review-verdict.schema.json` | Review outcome with structured verdict | `ops-pr-review`, `ops-pr-review-core` |

### shared-findings

Array of findings with `type`, `severity`, `package`, `file`, `item`, `description`, `evidence`, and `recommendation`. Any pipeline that produces analysis results should use this schema so downstream aggregation or triage steps can consume them uniformly.

### shared-review-verdict

Structured review output with `verdict` (APPROVE / REQUEST_CHANGES / COMMENT / REJECT), `summary`, `findings` array, `pr_url`, and `reviewed_at`. Enables composition patterns like implement-review-triage loops where the verdict drives conditional routing.

## Composition Example

The `impl-review-loop` pipeline chains three sub-pipelines:

```
impl-issue-core  -->  wave-land  -->  ops-pr-review-core
     |                    |                  |
  (implements)      (creates PR)      (reviews, produces verdict JSON)
                                             |
                                    verdict == 'APPROVE'? --> done
                                             |
                                          fix loop
```

The `ops-pr-review-core` summary step produces `.wave/output/review-verdict.json` validated against `shared-review-verdict.schema.json`. The loop condition reads `{{ review-fix.output.verdict == 'APPROVE' }}` from this structured output.

## Naming Conventions

- `shared-*.schema.json` -- schemas intended for cross-pipeline use
- `<domain>.schema.json` -- schemas specific to a single pipeline family (e.g., `bug-investigation.schema.json` for hotfix/bugfix pipelines)
- Avoid generic names like `findings.schema.json` -- prefix with the domain to prevent confusion

## Creating New Shared Schemas

1. Define the schema in `.wave/contracts/shared-<name>.schema.json`
2. Reference it in pipeline steps via `schema_path:`
3. Ensure both the producing and consuming pipelines agree on the schema
4. Use `on_failure: skip` during development, graduate to `retry` when stable
