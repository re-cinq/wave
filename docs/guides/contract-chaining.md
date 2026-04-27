# Contract Chaining for Cross-Pipeline Composition

When composing multiple pipelines in sequence, the output contract of one pipeline must be structurally compatible with the input expectations of the next. Wave provides **shared contract schemas** to make these handover points explicit and validated.

## Shared Schemas

Shared schemas live in `.agents/contracts/shared-*.schema.json` and are designed to be referenced by multiple pipelines:

| Schema | Purpose | Used by |
|--------|---------|---------|
| `shared-findings.schema.json` | Standardized audit/analysis findings | All `audit-*` pipelines |
| `shared-review-verdict.schema.json` | Review outcome with structured verdict | `ops-pr-review` (verdict step; publish step gated by `config.env.profile`) |
| `shared-pr-result.schema.json` | PR URL/number/branch/summary output | PR-producing pipelines |

### shared-findings

Array of findings with `type`, `severity`, `package`, `file`, `item`, `description`, `evidence`, and `recommendation`. Any pipeline that produces analysis results should use this schema so downstream aggregation or triage steps can consume them uniformly.

### shared-review-verdict

Structured review output with `verdict` (APPROVE / REQUEST_CHANGES / COMMENT / REJECT), `summary`, `findings` array, `pr_url`, and `reviewed_at`. Enables composition patterns like implement-review-triage loops where the verdict drives conditional routing.

### shared-pr-result

Standard output for steps that produce a pull request: `pr_url` (required, format: uri), `pr_number` (integer), `branch`, `summary`, and `title`. Extracts the common PR output pattern repeated across `pr-result`, `review-findings`, and `triage-verdict` schemas.

## Canonical Severity Conventions

Two severity vocabularies are used intentionally:

| Context | Enum values | Used in |
|---------|-------------|---------|
| Scan/audit severity | `critical`, `high`, `medium`, `low`, `info` | `shared-findings`, `security-scan`, `doc-fix-scan`, `review-findings` |
| Review severity | `critical`, `major`, `minor`, `suggestion` | `shared-review-verdict` |

All schemas use **lowercase** severity values. When mapping between vocabularies (e.g., in the rework gate), `high` maps to `major`, `medium` to `minor`, and `low`+`info` to `suggestion`.

## Input Validation with `schema_path`

When a pipeline step injects artifacts from a previous step, you can add input-side schema validation using `schema_path` on `inject_artifacts`:

```yaml
memory:
  inject_artifacts:
    - step: scan
      artifact: findings
      as: scan_findings
      schema_path: .agents/contracts/shared-findings.schema.json
```

This validates the injected artifact against the schema before the step executes. If validation fails, the step is skipped with a validation error. Use this to catch schema drift early in cross-step artifact chains.

### Which pipelines use input validation

All audit pipelines that chain findings between scan and report steps now validate injected artifacts:

- `audit-architecture`, `audit-tests`, `audit-correctness` — validate `scan_findings` against `shared-findings.schema.json`
- `audit-doc` — validates `scan` against `doc-scan-results.schema.json`
- `audit-dead-code-issue` — validates `findings` against `shared-findings.schema.json`
- `ops-pr-fix-review` — validates `raw_findings` against `review-findings.schema.json`

## Schema Sync

`.agents/contracts/` is the authoritative source. `internal/defaults/contracts/` contains embedded copies distributed with the binary. Changes flow `.agents/` → `internal/defaults/`. A test in `internal/contract/sync_test.go` verifies that all `shared-*.schema.json` files match between the two directories.

## Composition Example

The `impl-review-loop` pipeline chains three sub-pipelines:

```
impl-issue-core  -->  wave-land  -->  ops-pr-review (profile=core)
     |                    |                  |
  (implements)      (creates PR)      (reviews, produces verdict JSON)
                                             |
                                    verdict == 'APPROVE'? --> done
                                             |
                                          fix loop
```

The `ops-pr-review` summary step produces `.agents/output/review-verdict.json` validated against `shared-review-verdict.schema.json`. Callers that want the verdict only (no PR comment) pass `config.env.profile: core`, which gates the trailing publish step via `branch`. The loop condition reads <code v-pre>{{ review-fix.output.verdict == 'APPROVE' }}</code> from this structured output.

```yaml
- id: review
  pipeline: ops-pr-review
  config:
    env:
      profile: core   # skip publish step; produce verdict artifact only
  input: "{{ input }}"
```

## Naming Conventions

- `shared-*.schema.json` -- schemas intended for cross-pipeline use
- `<domain>.schema.json` -- schemas specific to a single pipeline family (e.g., `bug-investigation.schema.json` for hotfix/bugfix pipelines)
- Avoid generic names like `findings.schema.json` -- prefix with the domain to prevent confusion

## Creating New Shared Schemas

1. Define the schema in `.agents/contracts/shared-<name>.schema.json`
2. Reference it in pipeline steps via `schema_path:`
3. Ensure both the producing and consuming pipelines agree on the schema
4. Use `on_failure: skip` during development, graduate to `retry` when stable
