# Verification & Validation Patterns

Wave provides built-in contract types that verify step outputs before execution continues. This guide covers each contract type, common composition patterns, and how to combine them effectively.

## Contract Types Reference

| Type | Purpose | Requires LLM | Rework-able |
|------|---------|:---:|:---:|
| `non_empty_file` | Assert a file exists and is non-empty | No | No |
| `json_schema` | Validate JSON output against a JSON Schema | No | Yes |
| `format` | Check file format (JSON, YAML, etc.) | No | Yes |
| `llm_judge` | LLM evaluates output against criteria | Yes | Yes |
| `source_diff` | Verify meaningful source changes were made | No | No |
| `agent_review` | Full agent review of step output | Yes | Yes |
| `spec_derived_test` | Generate and run tests from a specification | Yes | Yes |
| `test_suite` | Run an existing test suite | No | Yes |
| `markdown_spec` | Validate markdown structure | No | Yes |
| `typescript_interface` | Check TypeScript interface conformance | No | Yes |

## Basic Contract Usage

Every step can declare a `handover.contract` block:

```yaml
steps:
  - id: generate-code
    persona: implementer
    handover:
      contract:
        type: json_schema
        source: .agents/output/result.json
        schema_path: .agents/contracts/result.schema.json
        on_failure: rework   # or: fail, warn
        max_rework: 3
```

### on_failure Modes

- **`fail`** — Stop the pipeline immediately. Use for hard gates.
- **`rework`** — Re-run the step with the validation error as feedback. The step sees what went wrong and can fix it.
- **`warn`** — Log the violation but continue. Use for advisory checks.

## Rework Loops

When `on_failure: rework` is set, Wave feeds the contract error back into the step's prompt and re-executes. This creates a self-correcting loop:

```
Step executes → Contract validates → Fail? → Feed error back → Re-execute
                                       ↓
                                    Pass? → Continue to next step
```

Configure bounds with `max_rework`:

```yaml
handover:
  contract:
    type: json_schema
    source: output.json
    schema_path: schema.json
    on_failure: rework
    max_rework: 3        # give up after 3 attempts
```

Each rework attempt is recorded as a `step_attempt` in the state database, giving full visibility into retry history.

## Gate Patterns

Gates pause pipeline execution until a condition is met:

```yaml
- id: human-review
  gate:
    type: approval
    message: "Review the generated code before continuing"
    timeout: "2h"
    choices:
      - label: "Approve"
        key: "a"
      - label: "Reject"
        key: "r"
        target: "_fail"
```

### Gate Types

| Type | Behavior |
|------|----------|
| `approval` | Wait for human choice (CLI, TUI, or WebUI) |
| `timer` | Auto-resolve after a duration |
| `pr_merge` | Poll GitHub until a PR is merged |
| `ci_pass` | Poll CI status on open PRs |

Gates can be auto-approved for CI/batch contexts:

```yaml
gate:
  type: approval
  auto: true
  timeout: "30m"
  default: "a"
```

## LLM Judge

The `llm_judge` contract uses a separate LLM call to evaluate output quality:

```yaml
handover:
  contract:
    type: llm_judge
    source: .agents/output/review.md
    criteria: |
      - Contains at least 3 specific findings
      - Each finding references a file path
      - Severity levels are assigned (low/medium/high/critical)
    model: cheapest
    on_failure: rework
    max_rework: 2
```

The judge runs independently from the step's LLM — it evaluates the output file against the criteria and returns pass/fail with reasoning.

## Composition V&V

When composing pipelines via `branch:` or command steps, contracts bridge the boundary:

### Output Contract → Input Expectation

The producing pipeline's final step should write a well-defined artifact:

```yaml
# Producer pipeline
- id: analyze
  persona: auditor
  output_artifacts:
    - name: findings
      path: .agents/output/findings.json
  handover:
    contract:
      type: json_schema
      source: .agents/output/findings.json
      schema_path: .agents/contracts/shared-findings.schema.json
      on_failure: rework
```

The consuming pipeline can then rely on that schema:

```yaml
# Consumer pipeline
- id: triage
  persona: navigator
  exec:
    type: prompt
    source: |
      Read .agents/output/findings.json and prioritize the findings...
```

### Shared Schemas

Place reusable schemas in `.agents/contracts/shared-*.schema.json`. See the [Contract Chaining](/guides/contract-chaining) guide for details on shared schema conventions.

## Combining Multiple Contracts

A step can only have one `handover.contract` block. To apply multiple validations, use a script contract that runs several checks:

```yaml
handover:
  contract:
    type: test_suite
    source: ./scripts/validate-output.sh
    on_failure: fail
```

Or chain steps — one produces output, the next validates it with a different contract type.

## Decision Logging

Every contract evaluation is recorded in the `decision_log` table with category `"contract"`. Query decision history with:

```bash
wave analyze --decisions
```

This shows contract pass/fail rates per pipeline, helping identify steps that frequently need rework.

## Best Practices

1. **Start with `non_empty_file`** — cheapest possible gate. Catches steps that produce nothing.
2. **Add `json_schema` for structured output** — catches format drift without LLM cost.
3. **Reserve `llm_judge` for semantic quality** — use when structure alone can't verify correctness.
4. **Set `max_rework` conservatively** — 2-3 attempts is usually enough. More suggests a prompt problem.
5. **Use `warn` during development** — switch to `fail` or `rework` once the pipeline is stable.
6. **Use `cheapest` model for judges** — validation doesn't need the strongest model.
