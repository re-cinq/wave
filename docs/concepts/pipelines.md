# Pipelines

A pipeline is a multi-step AI workflow where each step runs one persona in an isolated workspace.

```yaml
kind: WavePipeline
metadata:
  name: code-review
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
```

Use pipelines when you need coordinated AI tasks that build on each other's outputs.

## Adding Dependencies

Steps can depend on other steps. Dependencies run first, and their artifacts are available to dependent steps.

```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze the codebase for: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json

  - id: implement
    persona: craftsman
    dependencies: [analyze]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Implement based on the analysis."
```

## Parallel Execution

Steps without mutual dependencies run in parallel:

```yaml
steps:
  - id: navigate
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"

  - id: security
    persona: auditor
    dependencies: [navigate]
    exec:
      type: prompt
      source: "Security review"

  - id: quality
    persona: auditor
    dependencies: [navigate]
    exec:
      type: prompt
      source: "Quality review"

  - id: summary
    persona: navigator
    dependencies: [security, quality]
    exec:
      type: prompt
      source: "Summarize all findings"
```

In this example, `security` and `quality` run in parallel after `navigate` completes.

## Running Pipelines

```bash
wave run --pipeline code-review --input "Review authentication changes"
```

## Next Steps

- [Personas](/concepts/personas) - Configure the AI agents that run in each step
- [Contracts](/concepts/contracts) - Validate step outputs before handover
- [Artifacts](/concepts/artifacts) - Pass data between pipeline steps
- [Pipeline Schema Reference](/reference/pipeline-schema) - Complete field reference
