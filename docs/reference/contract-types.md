# Contract Types Reference

Contracts validate step output before dependent steps begin. This page documents all available contract types.

## Quick Reference

| Type | Validates | Use When |
|------|-----------|----------|
| `test_suite` | Command exit code | Verifying code compiles and tests pass |
| `json_schema` | JSON structure | Ensuring data format and required fields |
| `typescript_interface` | TypeScript compiles | Validating generated type definitions |
| `markdown_spec` | Markdown structure | Checking documentation format |

---

## test_suite

Run a command and validate exit code.

```yaml
handover:
  contract:
    type: test_suite
    command: "npm test"
```

**Use when:** Verifying implementation correctness through tests.

### Full Configuration

```yaml
handover:
  contract:
    type: test_suite
    command: "go test ./... && go vet ./..."
    dir: project_root
    must_pass: true
    on_failure: retry
    max_retries: 3
```

### Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `command` | **yes** | - | Shell command to execute |
| `dir` | no | workspace | Working directory: `project_root`, absolute path, or relative to workspace |
| `must_pass` | no | `true` | Whether failure blocks progression |
| `on_failure` | no | `retry` | `retry` or `halt` |
| `max_retries` | no | `2` | Maximum retry attempts |

### Working Directory

By default, `test_suite` commands run in the step's workspace directory. Since workspaces are ephemeral and isolated, commands like `go test ./...` will fail if they expect project files (e.g., `go.mod`).

Use `dir` to control where the command runs:

| Value | Resolves to |
|-------|-------------|
| _(empty)_ | Step workspace (default) |
| `project_root` | Git repository root (`git rev-parse --show-toplevel`) |
| `/absolute/path` | Used as-is |
| `relative/path` | Relative to workspace |

### Examples

**Go project (run tests at project root):**
```yaml
handover:
  contract:
    type: test_suite
    command: "go build ./... && go test ./..."
    dir: project_root
```

**Node.js project:**
```yaml
handover:
  contract:
    type: test_suite
    command: "npm test"
```

**Python project:**
```yaml
handover:
  contract:
    type: test_suite
    command: "pytest"
```

**Multi-command validation:**
```yaml
handover:
  contract:
    type: test_suite
    command: ".wave/scripts/validate.sh"
```

---

## json_schema

Validate JSON output against a JSON Schema.

```yaml
handover:
  contract:
    type: json_schema
    schema_path: .wave/contracts/analysis.schema.json
    source: output/analysis.json
```

**Use when:** Ensuring structured output with specific fields and types.

### Full Configuration

```yaml
handover:
  contract:
    type: json_schema
    schema_path: .wave/contracts/analysis.schema.json
    source: output/analysis.json
    must_pass: true
    on_failure: retry
    max_retries: 2
```

### Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `schema_path` | **yes** | - | Path to JSON Schema file |
| `source` | **yes** | - | Path to JSON file to validate |
| `must_pass` | no | `true` | Whether failure blocks progression |
| `on_failure` | no | `retry` | `retry` or `halt` |
| `max_retries` | no | `2` | Maximum retry attempts |

### Example Schema

`.wave/contracts/analysis.schema.json`:
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
    "summary": { "type": "string", "minLength": 10 }
  }
}
```

### Common Patterns

**Navigation output:**
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["files", "patterns", "summary"],
  "properties": {
    "files": { "type": "array", "items": { "type": "string" } },
    "patterns": { "type": "array", "items": { "type": "string" } },
    "summary": { "type": "string" }
  }
}
```

**Task list output:**
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["tasks"],
  "properties": {
    "tasks": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["task"],
        "properties": {
          "task": { "type": "string" },
          "priority": { "type": "string", "enum": ["high", "medium", "low"] }
        }
      }
    }
  }
}
```

---

## typescript_interface

Validate that generated TypeScript compiles successfully.

```yaml
handover:
  contract:
    type: typescript_interface
    source: output/types.ts
```

**Use when:** Ensuring generated TypeScript definitions are valid.

### Full Configuration

```yaml
handover:
  contract:
    type: typescript_interface
    source: output/types.ts
    validate: true
    must_pass: true
    on_failure: retry
    max_retries: 2
```

### Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `source` | **yes** | - | Path to TypeScript file |
| `validate` | no | `true` | Run type checking |
| `must_pass` | no | `true` | Whether failure blocks progression |
| `on_failure` | no | `retry` | `retry` or `halt` |
| `max_retries` | no | `2` | Maximum retry attempts |

### Behavior

1. Checks TypeScript syntax
2. If `tsc` is available, runs full type checking
3. If `tsc` is unavailable, performs syntax-only validation

### Examples

**Type definitions:**
```yaml
steps:
  - id: generate-types
    persona: craftsman
    exec:
      type: prompt
      source: "Generate TypeScript interfaces for the API"
    output_artifacts:
      - name: types
        path: output/api.types.ts
    handover:
      contract:
        type: typescript_interface
        source: output/api.types.ts
```

---

## markdown_spec

Validate Markdown document structure.

```yaml
handover:
  contract:
    type: markdown_spec
    source: output/spec.md
```

**Use when:** Ensuring documentation follows required format.

### Full Configuration

```yaml
handover:
  contract:
    type: markdown_spec
    source: output/spec.md
    must_pass: true
    on_failure: retry
    max_retries: 2
```

### Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `source` | **yes** | - | Path to Markdown file |
| `must_pass` | no | `true` | Whether failure blocks progression |
| `on_failure` | no | `retry` | `retry` or `halt` |
| `max_retries` | no | `2` | Maximum retry attempts |

### Validation Checks

- Valid Markdown syntax
- Required sections present (configurable)
- Proper heading hierarchy

### Examples

**Specification document:**
```yaml
steps:
  - id: specify
    persona: philosopher
    exec:
      type: prompt
      source: "Create a feature specification"
    output_artifacts:
      - name: spec
        path: output/spec.md
    handover:
      contract:
        type: markdown_spec
        source: output/spec.md
```

---

## Failure Handling

### Retry Behavior

When `on_failure: retry`:

1. Step state changes to `retrying`
2. Fresh workspace is created
3. Step re-executes from scratch
4. Validation runs again
5. After `max_retries`, step fails

```yaml
handover:
  contract:
    type: json_schema
    schema_path: .wave/contracts/output.schema.json
    source: output/data.json
    on_failure: retry
    max_retries: 3
```

### Halt Behavior

When `on_failure: halt`:

1. Step immediately fails
2. Pipeline stops
3. Error includes validation details

```yaml
handover:
  contract:
    type: test_suite
    command: "npm test"
    on_failure: halt
```

### Advisory Contracts

Use `must_pass: false` for warnings that don't block:

```yaml
handover:
  contract:
    type: test_suite
    command: "npm run lint"
    must_pass: false
```

Validation runs and logs results, but step completes regardless.

---

## Chained Validation

Use a shell script for multiple validation steps:

```yaml
handover:
  contract:
    type: test_suite
    command: ".wave/scripts/validate-all.sh"
```

`.wave/scripts/validate-all.sh`:
```bash
#!/bin/bash
set -e

echo "Validating JSON schema..."
npx ajv validate -s .wave/contracts/output.schema.json -d output/data.json

echo "Running tests..."
npm test

echo "Checking TypeScript..."
npx tsc --noEmit output/types.ts

echo "All validations passed"
```

---

## Contract Organization

Recommended directory structure:

```
.wave/
├── contracts/
│   ├── navigation.schema.json
│   ├── specification.schema.json
│   ├── task-list.schema.json
│   └── review.schema.json
├── scripts/
│   └── validate.sh
└── pipelines/
    └── code-review.yaml
```

---

## Debugging Failures

View contract validation errors:

```bash
wave logs run-abc123 --errors
```

**Output:**
```
[14:32:15] contract_failure  analyze  json_schema
  Error: Missing required property 'summary'
  File: output/analysis.json
  Schema: .wave/contracts/analysis.schema.json
```

Check audit logs for details:

```bash
cat .wave/traces/run-abc123.jsonl | grep contract
```

---

## Next Steps

- [Contracts](/concepts/contracts) - Contract concepts
- [Pipeline Schema](/reference/pipeline-schema) - Full step configuration
- [Artifacts](/concepts/artifacts) - Output files validated by contracts
