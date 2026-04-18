# Concepts

Wave is a multi-agent pipeline orchestrator. Understanding these core concepts will help you design effective AI workflows.

<script setup>
const conceptCards = [
  {
    title: 'Pipelines',
    description: 'Multi-step AI workflows with dependencies and artifact passing',
    link: '/concepts/pipelines',
    icon: '🔀'
  },
  {
    title: 'Personas',
    description: 'Specialized AI agents with scoped permissions and system prompts',
    link: '/concepts/personas',
    icon: '🎭'
  },
  {
    title: 'Contracts',
    description: 'Output validation rules ensuring AI outputs meet requirements',
    link: '/concepts/contracts',
    icon: '📜'
  },
  {
    title: 'Artifacts',
    description: 'Files passed between pipeline steps for data sharing',
    link: '/concepts/artifacts',
    icon: '📦'
  },
  {
    title: 'Workspaces',
    description: 'Git-native worktrees that give every pipeline a real checkout on a dedicated branch',
    link: '/concepts/workspaces',
    icon: '📁'
  },
  {
    title: 'Execution',
    description: 'How pipelines run, with fresh memory at step boundaries',
    link: '/concepts/pipelines',
    icon: '⚡'
  }
]
</script>

<CardGrid :cards="conceptCards" />

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
        path: .agents/output/analysis.json
    handover:
      contract:              # Validates output
        type: json_schema
        schema_path: .agents/contracts/analysis.schema.json
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                     Wave Manifest                        │
│                      (wave.yaml)                         │
└─────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│   Personas    │  │   Pipelines   │  │   Contracts   │
│  (AI agents)  │  │  (workflows)  │  │ (validation)  │
└───────────────┘  └───────────────┘  └───────────────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           ▼
                ┌───────────────────┐
                │    Execution      │
                │   (orchestrator)  │
                └───────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│   Workspace   │  │   Artifacts   │  │  Audit Logs   │
│  (isolated)   │  │   (output)    │  │  (tracking)   │
└───────────────┘  └───────────────┘  └───────────────┘
```

## Next Steps

- [Pipelines](/concepts/pipelines) - Start with the core execution model
- [Quick Start](/quickstart) - Run your first pipeline in 60 seconds
- [Use Cases](/use-cases/) - See complete working examples
