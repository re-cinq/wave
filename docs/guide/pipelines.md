# Pipelines Guide

Pipelines are DAGs (Directed Acyclic Graphs) that orchestrate multi-step agent workflows. Each step executes one persona in an isolated workspace, passing artifacts to dependent steps.

## Basic Structure

```yaml
kind: WavePipeline
metadata:
  name: feature-flow
  description: "Implement a feature from analysis to review"

steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Analyze the codebase for: {{ input }}"

  - id: implement
    persona: craftsman
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: navigation_report
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

### Execution Types

**Prompt** - Send to LLM:
```yaml
exec:
  type: prompt
  source: "Analyze: {{ input }}"
```

**Command** - Run shell:
```yaml
exec:
  type: command
  source: "go test -v ./..."
```

### Template Variables

| Variable | Description |
|----------|-------------|
| `{{ input }}` | Pipeline input from `--input` flag |
| `{{ task }}` | Current task in matrix strategy |

## Dependencies and DAG

Steps execute in dependency order. Independent steps run in parallel:

```yaml
steps:
  - id: navigate        # Runs first
  - id: specify
    dependencies: [navigate]
  - id: implement
    dependencies: [specify]
  - id: test
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

## Workspace Configuration

Each step gets isolated workspace:

```yaml
workspace:
  mount:
    - source: ./
      target: /src
      mode: readonly    # or readwrite
```

Default structure:
```
/tmp/wave/<pipeline-id>/<step-id>/
├── src/          # Mounted source
├── artifacts/    # Injected from dependencies
└── output/       # Step output
```

## Running Pipelines

```bash
# Run with input
wave run --pipeline .wave/pipelines/flow.yaml --input "add auth"

# Dry run
wave run --pipeline flow.yaml --dry-run

# Resume from step
wave run --pipeline flow.yaml --from-step implement
```

## Step States

```
Pending -> Running -> Completed
              |-> Retrying -> Running
              \-> Failed
```

## Common Patterns

### Speckit Flow

```yaml
steps:
  - id: navigate
    persona: navigator
  - id: specify
    persona: philosopher
    dependencies: [navigate]
  - id: implement
    persona: craftsman
    dependencies: [specify]
  - id: review
    persona: auditor
    dependencies: [implement]
```

### Ad-Hoc Execution

For quick tasks without pipeline files:

```bash
wave do "fix the bug" --persona craftsman
```

Generates a 2-step pipeline (navigate -> execute) automatically.

## Related Topics

- [Pipeline Schema Reference](/reference/pipeline-schema) - Full field reference
- [Contracts Guide](/guide/contracts) - Step output validation
- [Relay Guide](/guide/relay) - Context compaction
