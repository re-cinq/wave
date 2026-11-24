# Pipelines Guide

Pipelines are DAGs (Directed Acyclic Graphs) that orchestrate multi-step agent workflows. Each step executes one persona in an isolated workspace, passing artifacts to dependent steps.

## Built-in Pipelines

`wave init` installs these pipelines into `.agents/pipelines/`:

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `ops-hello-world` | greet -> verify | Smoke test of the contract loop |
| `impl-issue` | fetch-assess -> plan -> implement -> fix-implement -> create-pr | Implement a GitHub issue end to end |
| `ops-pr-review` | diff-analysis -> security-review, quality-review, slop-review -> summary -> publish | Review a pull request |
| `plan-task` | explore -> breakdown -> review | Break a feature into ordered tasks |
| `plan-research` | fetch-issue -> analyze-topics -> research-topics -> synthesize-report -> post-comment | Research an issue and post findings |
| `plan-scope` | fetch-epic -> scope-and-create -> verify-report | Split an epic into child issues |
| `audit-security` | scan -> deep-dive -> report | Security audit with typed findings |
| `audit-architecture` | scan -> report | Architecture audit with typed findings |
| `ops-parallel-audit` | run-audits (iterate) -> merge-findings (aggregate) -> report | Run both audits in parallel and merge the findings |

Need something else? `wave meta "<task>"` generates a pipeline, see [Meta-Pipelines](/guide/meta-pipelines).

## Running Pipelines

```bash
# Run with input
wave run impl-issue "https://github.com/org/repo/issues/42"

# Preview execution plan
wave run plan-task --dry-run

# Start from specific step
wave run impl-issue --from-step implement

# Custom timeout
wave run audit-security --timeout 120
```

## Pipeline Structure

<div v-pre>

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
        path: .agents/output/analysis.json
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

</div>

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
    path: .agents/output/analysis.json
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
    schema_path: .agents/contracts/analysis.schema.json
    source: .agents/output/analysis.json
    on_failure: retry
    max_retries: 2
```

Contract types:
- `json_schema` — Validate against JSON Schema
- `typescript_interface` — Validate against TypeScript interface
- `test_suite` — Run test command, must pass
- `markdown_spec` — Validate Markdown structure

## Template Variables

| Variable | Description |
|----------|-------------|
| <code v-pre>{{ input }}</code> | Pipeline input from `--input` flag |
| <code v-pre>{{ task }}</code> | Current task in matrix strategy |

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
.agents/workspaces/<pipeline-id>/<step-id>/
├── src/               # Mounted source
├── .agents/artifacts/   # Injected from dependencies
└── .agents/output/      # Step output
```

## Matrix Strategy (Parallel Fan-Out)

Spawn parallel instances from a task list:

```yaml
- id: plan
  output_artifacts:
    - name: tasks
      path: .agents/output/tasks.json

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

## Related Topics

- [Pipeline Schema Reference](/reference/pipeline-schema)
- [Contracts Guide](/guide/contracts)
- [Personas Guide](/guide/personas)
