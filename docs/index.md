---
layout: home
hero:
  name: Wave
  text: Multi-Agent Orchestrator for Claude Code
  tagline: Pipeline DAGs, persona-scoped safety, handover contracts, and context relay — all from a single YAML manifest.
  image:
    src: /logo.svg
    alt: Wave
  actions:
    - theme: brand
      text: Get Started
      link: /guide/quick-start
    - theme: alt
      text: CLI Reference
      link: /reference/cli
    - theme: alt
      text: View on GitHub
      link: https://github.com/recinq/wave
features:
  - icon: 📋
    title: Manifest-Driven
    details: One wave.yaml declares adapters, personas, runtime settings, and skill mounts. The manifest is the single source of truth — version it, validate it, share it.
    link: /concepts/manifests
    linkText: Learn about manifests
  - icon: 🔀
    title: Pipeline DAGs
    details: Define multi-step workflows as directed acyclic graphs. Steps execute in dependency order with automatic parallelism, artifact injection, and contract validation at every boundary.
    link: /concepts/pipelines
    linkText: How pipelines work
  - icon: 🛡️
    title: Persona-Scoped Safety
    details: Each agent runs with explicit permissions, hooks, and tool restrictions. A navigator can't write files. A craftsman can't push to remote. Deny patterns always win.
    link: /concepts/personas
    linkText: Understand personas
  - icon: 📄
    title: Handover Contracts
    details: JSON Schema, TypeScript interface, or test suite validation at every step boundary. Malformed artifacts never propagate — failed contracts trigger retries or halt the pipeline.
    link: /concepts/contracts
    linkText: Contract system
  - icon: 🧠
    title: Context Relay
    details: Automatic compaction when agents approach token limits. A summarizer persona creates structured checkpoints, and fresh instances resume without repeating work.
    link: /guides/relay-compaction
    linkText: Relay mechanism
  - icon: ⚡
    title: Ad-Hoc Execution
    details: "Run wave do 'fix the bug' for quick tasks. Wave generates a 2-step pipeline (navigate → execute) with full safety model — no YAML required."
    link: /reference/cli#wave-do
    linkText: Ad-hoc commands
---

## Quick Start

```bash
# Install
curl -L https://github.com/recinq/wave/releases/latest/download/wave-linux-amd64 -o wave
chmod +x wave && sudo mv wave /usr/local/bin/

# Initialize project
cd your-project && wave init

# Validate configuration
wave validate

# Run a pipeline
wave run --pipeline .wave/pipelines/speckit-flow.yaml \
  --input "add user authentication"

# Or just do a quick task
wave do "fix the broken login test"
```

## How It Works

Wave wraps LLM CLIs (like Claude Code) as subprocess adapters, then orchestrates them through pipeline DAGs where each step binds a persona to an ephemeral workspace.

```
wave.yaml → Pipeline DAG → Step Execution → Artifacts
                  │                 │
              Dependency        Persona binding
              resolution        Workspace isolation
              Cycle detection   Contract validation
              Parallelism       State persistence
```

Every step gets fresh context, explicit permissions, and validated handover contracts. Pipeline state is persisted in SQLite for resumption after interruptions.
