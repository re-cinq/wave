# Contracts

A contract validates that a step's output meets requirements before the next step begins. Contracts catch malformed artifacts early, preventing wasted work downstream.

```yaml
handover:
  contract:
    type: test_suite
    command: "go test ./..."
```

Use contracts when you need guaranteed output quality - structure validation, type checking, or test verification.

## Simple: Test Suite Contract

Run tests to validate implementation:

```yaml
steps:
  - id: implement
    persona: craftsman
    exec:
      type: prompt
      source: "Implement the feature"
    handover:
      contract:
        type: test_suite
        command: "npm test"
```

## Intermediate: JSON Schema Contract

Validate output structure against a schema:

```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json
    handover:
      contract:
        type: json_schema
        schema_path: .wave/contracts/analysis.schema.json
        source: output/analysis.json
```

Example schema file (`.wave/contracts/analysis.schema.json`):

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["files", "summary"],
  "properties": {
    "files": { "type": "array", "items": { "type": "string" } },
    "summary": { "type": "string" }
  }
}
```

## Advanced: Retry on Failure

Configure automatic retry when validation fails:

```yaml
handover:
  contract:
    type: json_schema
    schema_path: .wave/contracts/spec.schema.json
    source: output/spec.json
    on_failure: retry
    max_retries: 3
```

## Contract Types

| Type | Validates | Use When |
|------|-----------|----------|
| `test_suite` | Command exit code | Verifying code works |
| `json_schema` | JSON structure | Ensuring data format |
| `typescript_interface` | TypeScript compiles | Validating generated types |
| `markdown_spec` | Markdown structure | Checking documentation |

## Contract Fields

### Required Fields
- `type` - Contract type (see table above)

### Optional Fields
- `must_pass: true/false` - Whether validation must pass for step to succeed (default: true)
- `on_failure: retry|halt` - Behavior when validation fails
- `max_retries: N` - Maximum retry attempts (default: 0)
- `source` - Path to artifact being validated (for schema contracts)
- `schema_path` - Path to schema file (for `json_schema` type)
- `command` - Test command to run (for `test_suite` type)

## Failure Handling

| Setting | Behavior |
|---------|----------|
| `on_failure: retry` | Re-run step with fresh workspace |
| `on_failure: halt` | Stop pipeline immediately |

After `max_retries` is exceeded, the step fails regardless of `on_failure` setting.

## Next Steps

- [Artifacts](/concepts/artifacts) - Output files validated by contracts
- [Pipelines](/concepts/pipelines) - Use contracts in multi-step workflows
- [Contract Types Reference](/reference/contract-types) - Complete contract options
