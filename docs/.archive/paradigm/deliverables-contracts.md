# Contracts and Guaranteed Outputs

Traditional AI interactions are unpredictable - outputs vary in format, completeness, and quality. Wave solves this through **contracts**: automated validation gates that ensure step outputs meet your requirements before the pipeline continues.

## The Reliability Problem

Without contracts, AI outputs are accepted on faith:

```
Developer: "Generate an API schema in JSON format"
AI: Returns markdown instead
Developer: "No, I need JSON"
AI: Returns malformed JSON
Developer: "Please follow the exact structure I need"
AI: Returns valid JSON but missing required fields
```

This is manual, error-prone, and doesn't scale.

## How Contracts Work

Contracts validate step outputs before handover to the next step. If validation fails, Wave automatically retries with a fresh context.

```yaml
steps:
  - id: generate-schema
    persona: documenter
    exec:
      type: prompt
      source: "Generate OpenAPI schema for the endpoints in {{ input }}"
    output_artifacts:
      - name: schema
        path: .wave/output/api.json
        type: json
    handover:
      contract:
        type: jsonschema
        schema_path: .wave/contracts/openapi.schema.json
        on_failure: retry
        max_retries: 2
```

**Execution flow:**
1. Step completes, producing `.wave/output/api.json`
2. Wave validates output against the JSON Schema
3. **Pass**: Step marked complete, artifact available to dependent steps
4. **Fail**: Fresh workspace created, step re-executes (up to `max_retries`)

## Contract Types

Wave supports four contract types, matching common validation patterns:

### JSON Schema Validation

Validates JSON output against a JSON Schema specification.

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/response.schema.json
```

**Schema example:**
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["files", "summary"],
  "properties": {
    "files": {
      "type": "array",
      "items": { "type": "string" }
    },
    "summary": {
      "type": "string",
      "minLength": 10
    }
  }
}
```

**Use for:** Structured data outputs, API responses, configuration files.

### TypeScript Compilation

Validates that generated TypeScript code compiles without errors.

```yaml
handover:
  contract:
    type: typescript
    source: .wave/output/types.ts
    validate: true
```

**Use for:** Generated type definitions, API client code, typed configurations.

### Test Suite Execution

Runs a test command and validates it passes.

```yaml
handover:
  contract:
    type: testsuite
    command: "npm test -- --testPathPattern=generated"
    must_pass: true
```

**Use for:** Generated code that must be functionally correct, implementations that need to pass existing tests.

### Markdown Specification

Validates markdown structure and content requirements.

```yaml
handover:
  contract:
    type: markdownspec
    source: .wave/output/documentation.md
```

**Use for:** Documentation outputs, README files, structured reports.

## Failure Handling

When contracts fail, Wave provides specific feedback:

```json
{
  "contract_failure": {
    "type": "jsonschema",
    "schema_path": ".wave/contracts/api-spec.schema.json",
    "validation_errors": [
      {
        "path": "$.endpoints[0].responses",
        "message": "Required property '200' is missing"
      },
      {
        "path": "$.version",
        "message": "Expected string, got number"
      }
    ],
    "retry_count": 1,
    "max_retries": 2
  }
}
```

### Retry Behavior

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/output.schema.json
    on_failure: retry
    max_retries: 3
```

On failure:
1. Wave creates a fresh workspace (no contamination from failed attempt)
2. Step re-executes with the same inputs
3. New output is validated
4. Process repeats until success or `max_retries` exceeded

## Practical Examples

### Code Review Pipeline

Ensure structured analysis before generating review:

```yaml
kind: WavePipeline
metadata:
  name: code-review

steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: |
        Analyze changes: {{ input }}

        Output JSON with:
        - files_changed: array of file paths
        - risk_level: low, medium, high, or critical
        - test_coverage: percentage as number
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json
    handover:
      contract:
        type: jsonschema
        schema_path: .wave/contracts/analysis.schema.json
        on_failure: retry
        max_retries: 2

  - id: review
    persona: auditor
    dependencies: [analyze]
    memory:
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: context
```

**Contract schema:**
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["files_changed", "risk_level"],
  "properties": {
    "files_changed": {
      "type": "array",
      "items": { "type": "string" }
    },
    "risk_level": {
      "type": "string",
      "enum": ["low", "medium", "high", "critical"]
    },
    "test_coverage": {
      "type": "number",
      "minimum": 0,
      "maximum": 100
    }
  }
}
```

### Generated Code Pipeline

Ensure generated code compiles and passes tests:

```yaml
kind: WavePipeline
metadata:
  name: generate-client

steps:
  - id: generate
    persona: craftsman
    exec:
      type: prompt
      source: "Generate TypeScript API client from {{ input }}"
    output_artifacts:
      - name: client
        path: .wave/output/client.ts
        type: typescript
    handover:
      contract:
        type: typescript
        source: .wave/output/client.ts
        validate: true

  - id: test
    persona: craftsman
    dependencies: [generate]
    memory:
      inject_artifacts:
        - step: generate
          artifact: client
          as: code
    handover:
      contract:
        type: testsuite
        command: "npm test -- --testPathPattern=client"
        must_pass: true
```

## Benefits

### Predictable Outputs

Without contracts:
- "The AI might generate what we need"
- Manual checking of every output
- Failed outputs require complete restart

With contracts:
- "The AI will generate exactly what we specified, or retry"
- Automated validation at every step
- Automatic retry on failure

### Team Consistency

Without contracts:
- Each developer gets different quality outputs
- Tribal knowledge about "good prompts"
- Inconsistent downstream processing

With contracts:
- Same quality guarantees for everyone
- Quality requirements explicit in configuration
- Downstream steps can rely on validated inputs

### Enterprise Adoption

Without contracts:
- AI outputs too unreliable for production
- Human oversight required at every step
- Limited scalability

With contracts:
- Quality gates built into the pipeline
- Production-ready outputs
- Scalable AI automation

## Best Practices

### Start Simple

Begin with basic structure validation, add complexity as needed:

```yaml
# Start here
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/basic.schema.json

# Add later if needed
handover:
  contract:
    type: testsuite
    command: "npm test"
    must_pass: true
```

### Schema Design

- **Required fields only**: Don't over-constrain
- **Reasonable limits**: `minLength`, `maximum`, etc.
- **Clear enum values**: Constrain categorical data
- **Pattern matching**: Validate formats (emails, UUIDs, etc.)

### Retry Strategy

- **Low retries for simple tasks**: 1-2 retries
- **Higher retries for complex generation**: 3-5 retries
- **Monitor retry rates**: High retry rates indicate unclear prompts

## Contract Files Organization

Store contracts in a dedicated directory:

```
.wave/
├── contracts/
│   ├── analysis.schema.json
│   ├── review.schema.json
│   └── implementation.schema.json
├── pipelines/
│   └── code-review.yaml
└── personas/
    ├── navigator.md
    └── auditor.md
```

## Next Steps

- [AI as Code](/paradigm/ai-as-code) - The foundational paradigm
- [Infrastructure Parallels](/paradigm/infrastructure-parallels) - IaC pattern comparisons
- [Contracts Reference](/concepts/contracts) - Complete contract specification
- [Pipeline Execution](/concepts/pipeline-execution) - How contracts integrate with execution
