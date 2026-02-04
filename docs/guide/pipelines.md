# Pipelines Guide

Pipelines are DAGs (Directed Acyclic Graphs) that orchestrate multi-step agent workflows. Each step executes one persona in an isolated workspace, passing artifacts to dependent steps.

## Built-in Pipelines

Wave ships with 9 pipelines organized by use case:

### Development

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `speckit-flow` | navigate → specify → plan → implement → review | Feature development |
| `hotfix` | investigate → fix → verify | Production bugs |
| `refactor` | analyze → test-baseline → refactor → verify | Safe refactoring |

### Quality

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `code-review` | diff → security + quality → summary | PR reviews |
| `test-gen` | analyze-coverage → generate → verify | Test coverage |
| `debug` | reproduce → hypothesize → investigate → fix | Root cause analysis |

### Planning & Documentation

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `plan` | explore → breakdown → review | Task planning |
| `docs` | discover → generate → review | Documentation |
| `migrate` | impact → plan → implement → review | Migrations |

## Running Pipelines

```bash
# Run with input
wave run --pipeline speckit-flow --input "add user authentication"

# Preview execution plan
wave run --pipeline hotfix --dry-run

# Start from specific step
wave run --pipeline speckit-flow --from-step implement

# Custom timeout
wave run --pipeline migrate --timeout 120
```

## Pipeline Structure

```yaml
kind: WavePipeline
metadata:
  name: my-pipeline
  description: "What this pipeline does"

steps:
  - id: first-step
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json

  - id: second-step
    persona: craftsman
    dependencies: [first-step]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: first-step
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Implement based on the analysis."
```

## Step Configuration

| Field | Required | Description |
|-------|----------|-------------|
| `id` | yes | Unique step identifier |
| `persona` | yes | References persona in wave.yaml |
| `memory.strategy` | yes | Always `fresh` (clean context) |
| `exec.type` | yes | `prompt` or `command` |
| `exec.source` | yes | Prompt template or shell command |
| `dependencies` | no | Step IDs that must complete first |
| `output_artifacts` | no | Files produced by this step |
| `handover.contract` | no | Validation for step output |

## Dependencies and DAG

Steps execute in dependency order. Independent steps run in parallel:

```yaml
steps:
  - id: navigate        # Runs first
  - id: specify
    dependencies: [navigate]
  - id: implement
    dependencies: [specify]
  - id: test            # Parallel with review
    dependencies: [implement]
  - id: review          # Parallel with test
    dependencies: [implement]
```

## Artifacts

### Declaring Output

```yaml
output_artifacts:
  - name: analysis
    path: output/analysis.json
    type: json
```

### Injecting into Steps

```yaml
memory:
  strategy: fresh
  inject_artifacts:
    - step: navigate
      artifact: analysis
      as: codebase_analysis
```

## Contracts

Validate step output before proceeding:

```yaml
handover:
  contract:
    type: json_schema
    schema_path: .wave/contracts/analysis.schema.json
    source: output/analysis.json
    on_failure: retry
    max_retries: 2
```

Contract types:
- `json_schema` — Validate against JSON Schema
- `typescript` — Validate against TypeScript interface
- `test_suite` — Run test command, must pass

## Template Variables

| Variable | Description |
|----------|-------------|
| `{{ input }}` | Pipeline input from `--input` flag |
| `{{ task }}` | Current task in matrix strategy |

## Workspace Configuration

Each step gets an isolated workspace:

```yaml
workspace:
  mount:
    - source: ./
      target: /src
      mode: readonly    # or readwrite
```

Default structure:
```
.wave/workspaces/<pipeline-id>/<step-id>/
├── src/          # Mounted source
├── artifacts/    # Injected from dependencies
└── output/       # Step output
```

## Matrix Strategy (Parallel Fan-Out)

Spawn parallel instances from a task list:

```yaml
- id: plan
  output_artifacts:
    - name: tasks
      path: output/tasks.json

- id: execute
  dependencies: [plan]
  strategy:
    type: matrix
    items_source: plan/tasks.json
    item_key: task
    max_concurrency: 4
  exec:
    type: prompt
    source: "Execute: {{ task }}"
```

## Pipeline Examples

### speckit-flow

Full feature development workflow:

```yaml
steps:
  - id: navigate
    persona: navigator
    exec:
      source: "Analyze the codebase for: {{ input }}"

  - id: specify
    persona: philosopher
    dependencies: [navigate]
    exec:
      source: "Create specification for: {{ input }}"

  - id: plan
    persona: philosopher
    dependencies: [specify]
    exec:
      source: "Create implementation plan"

  - id: implement
    persona: craftsman
    dependencies: [plan]
    exec:
      source: "Implement according to plan"

  - id: review
    persona: auditor
    dependencies: [implement]
    exec:
      source: "Review for security and quality"
```

### hotfix

Fast-track bug fix:

```yaml
steps:
  - id: investigate
    persona: navigator
    exec:
      source: "Investigate: {{ input }}"

  - id: fix
    persona: craftsman
    dependencies: [investigate]
    exec:
      source: "Fix the issue with regression test"

  - id: verify
    persona: auditor
    dependencies: [fix]
    exec:
      source: "Verify fix is safe for production"
```

### debug

Systematic debugging:

```yaml
steps:
  - id: reproduce
    persona: debugger
    exec:
      source: "Reproduce: {{ input }}"

  - id: hypothesize
    persona: debugger
    dependencies: [reproduce]
    exec:
      source: "Form hypotheses about root cause"

  - id: investigate
    persona: debugger
    dependencies: [hypothesize]
    exec:
      source: "Test each hypothesis"

  - id: fix
    persona: craftsman
    dependencies: [investigate]
    exec:
      source: "Implement fix with regression test"
```

## Ad-Hoc Execution

For quick tasks without pipeline files:

```bash
wave do "fix the bug" --persona craftsman
```

Generates a 2-step pipeline (navigate → execute) automatically.

## Related Topics

- [Pipeline Schema Reference](/reference/pipeline-schema)
- [Contracts Guide](/guide/contracts)
- [Personas Guide](/guide/personas)
- [Relay Guide](/guide/relay)
