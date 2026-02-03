# Concepts

Wave is a multi-agent pipeline orchestrator. Understanding these core concepts will help you design effective AI workflows.

## Core Concepts

| Concept | What It Is | When You Need It |
|---------|-----------|------------------|
| [Pipelines](/concepts/pipelines) | Multi-step AI workflows | Running coordinated AI tasks |
| [Personas](/concepts/personas) | Specialized AI agents with permissions | Controlling what AI can access |
| [Contracts](/concepts/contracts) | Output validation rules | Ensuring AI outputs meet requirements |
| [Artifacts](/concepts/artifacts) | Files passed between steps | Sharing data across pipeline steps |
| [Execution](/concepts/execution) | How pipelines run | Understanding and debugging runs |

## How They Fit Together

```yaml
kind: WavePipeline
metadata:
  name: example
steps:
  - id: analyze              # Pipeline step
    persona: navigator       # Uses navigator persona
    exec:
      type: prompt
      source: "Analyze the codebase"
    output_artifacts:        # Produces artifacts
      - name: analysis
        path: output/analysis.json
    handover:
      contract:              # Validates output
        type: jsonschema
        schema: .wave/contracts/analysis.schema.json
```

## Next Steps

- [Pipelines](/concepts/pipelines) - Start with the core execution model
- [Quick Start](/quickstart) - Run your first pipeline in 60 seconds
- [Use Cases](/use-cases/) - See complete working examples
