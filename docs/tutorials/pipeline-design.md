# Pipeline Design Tutorial

Learn to design effective multi-step pipelines with DAG execution, artifact handoff, and contract validation.

## Pipeline Fundamentals

A pipeline is a DAG where:
- **Steps** execute personas in isolated workspaces
- **Dependencies** determine execution order
- **Artifacts** flow between steps
- **Contracts** validate output quality

## Example: Feature Implementation Pipeline

Create `.wave/pipelines/feature-flow.yaml`:

```yaml
kind: WavePipeline
metadata:
  name: feature-flow
input:
  source: cli
steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: Analyze the codebase for: {{ input }}
    output_artifacts:
      - name: analysis
        path: output/analysis.json
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/navigation.schema.json
        source: output/analysis.json

  - id: specify
    persona: philosopher
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: codebase_analysis
    exec:
      type: prompt
      source: Create a specification for: {{ input }}
    output_artifacts:
      - name: spec
        path: output/spec.md

  - id: implement
    persona: craftsman
    dependencies: [specify]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: specify
          artifact: spec
          as: feature_spec
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readwrite
    exec:
      type: prompt
      source: Implement per specification.
    handover:
      contract:
        type: test_suite
        command: "go test ./..."
        max_retries: 3

  - id: review
    persona: auditor
    dependencies: [implement]
    exec:
      type: prompt
      source: Review for security and quality issues.
```

## Key Patterns

**Read-only navigation:** Navigator cannot accidentally modify files.

**Artifact injection:** Pass outputs between steps:
```yaml
inject_artifacts:
  - step: navigate
    artifact: analysis
    as: codebase_analysis
```

**Contract validation:** Ensure quality at boundaries:
```yaml
handover:
  contract:
    type: test_suite
    command: "go test ./..."
```

## Matrix Strategy

For parallel sub-tasks:

```yaml
  - id: execute
    dependencies: [plan]
    strategy:
      type: matrix
      items_source: plan/tasks.json
      item_key: task
      max_concurrency: 4
    exec:
      type: prompt
      source: Complete this task: {{ task }}
```

## Validate and Run

```bash
wave validate --pipeline feature-flow.yaml
wave run --pipeline feature-flow.yaml --dry-run
wave run --pipeline feature-flow.yaml --input "Add user profile"
```

## Best Practices

1. **Start with navigation** - provides context for downstream steps
2. **Use contracts liberally** - catch errors early
3. **Keep steps focused** - one task per step
4. **Design for resumability** - `wave resume --from-step implement`

## Next Steps

- [Handover contracts](/reference/pipeline-schema#handoverconfig)
- [Custom personas](/tutorials/custom-personas)
