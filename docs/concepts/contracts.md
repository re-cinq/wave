# Handover Contracts

Contracts are validation gates at step boundaries. They ensure that a step's output meets quality requirements before the next step begins. Without contracts, a poorly-formed artifact propagates through the entire pipeline before anyone notices.

## Why Contracts?

Consider a pipeline where the navigator produces a codebase analysis, and the craftsman implements based on it. If the analysis is missing critical file paths, the craftsman wastes tokens implementing the wrong thing. Contracts catch this at the boundary.

```mermaid
graph LR
    S1[Step A Output] --> V{Contract Validation}
    V -->|Pass| S2[Step B Starts]
    V -->|Fail| R[Retry Step A]
    R --> V
```

## Contract Types

Muzzle supports three contract types, each validating at a different level:

### JSON Schema

Validates output structure against a JSON Schema definition. Best for checking that artifacts contain required fields and correct types.

```yaml
handover:
  contract:
    type: json_schema
    schema: .muzzle/contracts/navigation.schema.json
    source: output/analysis.json
    on_failure: retry
    max_retries: 2
```

Example schema:
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["files", "patterns", "dependencies"],
  "properties": {
    "files": {
      "type": "array",
      "items": { "type": "string" },
      "minItems": 1
    },
    "patterns": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "description"],
        "properties": {
          "name": { "type": "string" },
          "description": { "type": "string" }
        }
      }
    },
    "dependencies": { "type": "object" }
  }
}
```

### TypeScript Interface

Validates that generated TypeScript compiles against a declared interface. Best for checking that generated types or contracts are syntactically valid.

```yaml
handover:
  contract:
    type: typescript_interface
    source: output/types.ts
    validate: true
    on_failure: retry
    max_retries: 2
```

If `tsc` is not available in the environment, this degrades to syntax-only validation.

### Test Suite

Validates step output by running a test command. The most flexible contract type — any executable check can serve as validation.

```yaml
handover:
  contract:
    type: test_suite
    command: "npm test -- --testPathPattern=profile.test"
    must_pass: true
    on_failure: retry
    max_retries: 3
```

## Failure Handling

When a contract fails:

| `on_failure` | Behavior |
|-------------|----------|
| `retry` | Re-run the step with a fresh workspace. Retry count increments. |
| `halt` | Stop the pipeline immediately. Step transitions to `failed`. |

```yaml
handover:
  contract:
    on_failure: retry    # Try again
    max_retries: 3       # Up to 3 attempts
```

After `max_retries` is exceeded, the step transitions to `failed` regardless of `on_failure` setting.

### Retry Behavior

- Each retry gets a **fresh workspace** — no leftover state from the failed attempt.
- The retry budget is **per-step**, not per-pipeline.
- Retries use **exponential backoff** between attempts.
- Subprocess crashes and timeouts count as failures and use the same retry mechanism.

## Contract Placement

Contracts live on the **producing** step (the step whose output is validated), not the consuming step:

```yaml
steps:
  - id: navigate
    # ... execution config ...
    handover:
      contract:          # ← Validates navigate's output
        type: json_schema
        schema: .muzzle/contracts/nav.schema.json
        source: output/analysis.json

  - id: implement
    dependencies: [navigate]
    # If we get here, navigate's output is guaranteed valid
```

## Contract Design Patterns

### Progressive Validation

Use stricter contracts as the pipeline progresses:

```yaml
# Early steps: structural checks only
- id: navigate
  handover:
    contract:
      type: json_schema        # "Does the output have the right shape?"

# Middle steps: compilation checks
- id: specify
  handover:
    contract:
      type: typescript_interface  # "Does the output compile?"

# Late steps: behavioral checks
- id: implement
  handover:
    contract:
      type: test_suite           # "Does the output work correctly?"
```

### Schema Reuse

Store contract schemas in `.muzzle/contracts/` and reference them across pipelines:

```
.muzzle/contracts/
├── navigation.schema.json
├── specification.schema.json
├── implementation.schema.json
└── review.schema.json
```

## Further Reading

- [Pipeline Schema — HandoverConfig](/reference/pipeline-schema#handoverconfig) — complete field reference
- [Pipelines](/concepts/pipelines) — how contracts fit into step execution
- [Speckit Flow Example](/examples/speckit-flow) — contracts in a real pipeline
