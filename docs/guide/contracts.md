# Contracts Guide

Handover contracts validate step output before dependent steps begin. They catch malformed artifacts early, preventing wasted work downstream.

## What is a Contract?

A contract validates that a step produced correct output. Contracts can check:

- **Structure** - JSON schema compliance
- **Types** - TypeScript compilation
- **Behavior** - Test suite results

## Configuration

Contracts are defined in a step's `handover` section:

```yaml
steps:
  - id: navigate
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/navigation.schema.json
        source: output/analysis.json
        on_failure: retry
        max_retries: 2
```

| Field | Default | Description |
|-------|---------|-------------|
| `type` | - | `json_schema`, `typescript_interface`, or `test_suite` |
| `schema` | - | Schema file path (for json_schema) |
| `source` | - | File to validate |
| `command` | - | Test command (for test_suite) |
| `must_pass` | `true` | Whether failure blocks progression |
| `on_failure` | `retry` | Action: `retry` or `halt` |
| `max_retries` | `2` | Maximum retry attempts |

## Contract Types

### JSON Schema

Validates output structure:

```yaml
handover:
  contract:
    type: json_schema
    schema: .wave/contracts/navigation.schema.json
    source: output/analysis.json
```

Example schema:
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["files", "summary"],
  "properties": {
    "files": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["path", "purpose"],
        "properties": {
          "path": { "type": "string" },
          "purpose": { "type": "string" }
        }
      }
    },
    "summary": { "type": "string" }
  }
}
```

### TypeScript Interface

Validates generated TypeScript compiles:

```yaml
handover:
  contract:
    type: typescript_interface
    source: output/types.ts
    validate: true
```

If `tsc` is unavailable, degrades to syntax-only checking.

### Test Suite

Validates by running tests:

```yaml
handover:
  contract:
    type: test_suite
    command: "go test ./..."
    must_pass: true
    max_retries: 3
```

## Failure Handling

### Retry Behavior

When `on_failure: retry`:
1. Step transitions to `retrying`
2. Re-executes with fresh context
3. Validates again
4. After `max_retries` failures, transitions to `failed`

### Halt Behavior

When `on_failure: halt`:
1. Step immediately fails
2. Pipeline stops
3. Error includes validation details

### Optional Contracts

Use `must_pass: false` for advisory checks:

```yaml
handover:
  contract:
    type: test_suite
    command: "npm run lint"
    must_pass: false    # Log but don't block
```

## Common Patterns

### Navigation Contract

```yaml
- id: navigate
  output_artifacts:
    - name: analysis
      path: output/analysis.json
  handover:
    contract:
      type: json_schema
      schema: .wave/contracts/navigation.schema.json
      source: output/analysis.json
```

### Implementation Contract

```yaml
- id: implement
  handover:
    contract:
      type: test_suite
      command: "go build ./... && go test ./..."
      max_retries: 3
```

### Chained Validation

Use a script for multiple checks:

```yaml
handover:
  contract:
    type: test_suite
    command: ".wave/scripts/validate.sh"
```

```bash
#!/bin/bash
set -e
npx ajv validate -s schema.json -d output.json
npm test
```

## Schema Organization

```
.wave/
├── contracts/
│   ├── navigation.schema.json
│   ├── specification.schema.json
│   └── implementation.schema.json
└── pipelines/
    └── feature-flow.yaml
```

## Debugging Failures

Check audit logs:
```bash
cat .wave/traces/<pipeline-id>.jsonl | jq 'select(.type == "contract_failure")'
```

## Related Topics

- [Pipeline Schema Reference](/reference/pipeline-schema) - Contract field reference
- [Pipelines Guide](/guide/pipelines) - Using contracts in pipelines
