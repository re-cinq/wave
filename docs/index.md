---
layout: home
hero:
  name: Wave
  text: Multi-Agent Pipelines
  tagline: Orchestrate LLM agents with scoped permissions, validated contracts, and isolated workspaces.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/quick-start
    - theme: alt
      text: View Pipelines
      link: /guide/pipelines
    - theme: alt
      text: GitHub
      link: https://github.com/recinq/wave
features:
  - icon: 🛡️
    title: Persona-Scoped Safety
    details: Each agent runs with explicit permissions. Navigator can't write. Craftsman can't push. Auditor can't fix. Deny patterns always win.
    link: /concepts/personas
  - icon: 🔀
    title: Pipeline DAGs
    details: Multi-step workflows with dependency resolution, parallel execution, and artifact injection between steps.
    link: /concepts/pipelines
  - icon: 📄
    title: Validated Contracts
    details: JSON Schema, TypeScript, or test suite validation at every boundary. Malformed output triggers retry or halt.
    link: /concepts/contracts
  - icon: ⚡
    title: 9 Built-in Pipelines
    details: "speckit-flow, hotfix, code-review, refactor, debug, test-gen, docs, plan, migrate — ready to use."
    link: /guide/pipelines
  - icon: 👥
    title: 7 Specialized Personas
    details: "navigator, philosopher, planner, craftsman, debugger, auditor, summarizer — each with role-specific permissions."
    link: /guide/personas
  - icon: 🧠
    title: Context Relay
    details: Automatic compaction when approaching token limits. Summarizer creates checkpoints for seamless handoffs.
    link: /guides/relay-compaction
---

## Install

```bash
go install github.com/recinq/wave/cmd/wave@latest
```

## Quick Start

```bash
# Initialize project
wave init

# Run a pipeline
wave run --pipeline speckit-flow --input "add user authentication"

# Or quick ad-hoc tasks
wave do "fix the failing test"
```

## Pipelines at a Glance

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `speckit-flow` | navigate → specify → plan → implement → review | Feature development |
| `hotfix` | investigate → fix → verify | Production bugs |
| `code-review` | diff → security + quality → summary | PR reviews |
| `refactor` | analyze → baseline → refactor → verify | Safe refactoring |
| `debug` | reproduce → hypothesize → investigate → fix | Root cause analysis |
| `test-gen` | analyze → generate → verify | Test coverage |
| `docs` | discover → generate → review | Documentation |
| `plan` | explore → breakdown → review | Task planning |
| `migrate` | impact → plan → implement → review | Migrations |

## Personas at a Glance

| Persona | Temperature | Purpose |
|---------|-------------|---------|
| `navigator` | 0.1 | Read-only codebase exploration |
| `philosopher` | 0.3 | Architecture and specification |
| `planner` | 0.3 | Task breakdown and planning |
| `craftsman` | 0.7 | Implementation and testing |
| `debugger` | 0.2 | Systematic issue diagnosis |
| `auditor` | 0.1 | Security and quality review |
| `summarizer` | 0.0 | Context compaction |

## How It Works

```
wave.yaml → Pipeline DAG → Step Execution → Artifacts
                │                 │
            Dependency        Persona binding
            resolution        Workspace isolation
            Parallelism       Contract validation
```

Every step gets fresh context, explicit permissions, and validated handover contracts. State persists for resumption after interruptions.
