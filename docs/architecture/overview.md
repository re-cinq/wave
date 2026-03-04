# Wave Architecture Overview

Wave is a **multi-agent pipeline orchestrator** — it coordinates specialized AI agents
to perform complex software engineering tasks as a series of connected steps. Think of
it as an assembly line where each station has a specialist who receives materials from
the previous station, does their part, and passes their work forward.

This diagram shows all the major components and how they relate to each other.

```mermaid
graph TD
    Manifest["📋 Manifest<br/>(wave.yaml)"]
    Executor["⚙️ Pipeline Executor"]
    Preflight["✅ Preflight Checker"]
    DAG["🔀 DAG Resolver"]
    Personas["🎭 Personas"]
    Workspaces["📁 Workspaces"]
    Adapters["🔌 Adapters"]
    Contracts["📝 Contracts"]
    StateStore["💾 State Store"]
    Events["📡 Event System"]
    Relay["🔄 Relay Monitor"]
    Audit["📜 Audit Logger"]
    CLI["🖥️ Claude Code CLI"]

    Manifest -->|"defines pipelines,<br/>personas, contracts"| Executor
    Executor -->|"checks tool &<br/>skill availability"| Preflight
    Executor -->|"resolves step<br/>dependencies"| DAG
    DAG -->|"sorted execution<br/>order"| Executor

    Executor -->|"assigns persona<br/>to each step"| Personas
    Executor -->|"creates isolated<br/>directory per step"| Workspaces
    Executor -->|"validates output<br/>after each step"| Contracts

    Personas -->|"defines behavior,<br/>permissions, role"| Adapters
    Adapters -->|"launches subprocess"| CLI
    CLI -->|"runs in isolated<br/>workspace"| Workspaces

    Executor -->|"persists progress<br/>& enables resume"| StateStore
    Executor -->|"emits progress<br/>updates"| Events
    Executor -->|"monitors token usage,<br/>triggers compaction"| Relay
    CLI -->|"logs tool calls,<br/>scrubs credentials"| Audit

    style Manifest fill:#4a9eff,color:#fff,stroke:#2d7ad6
    style Executor fill:#ff6b6b,color:#fff,stroke:#d64545
    style Personas fill:#ffd93d,color:#333,stroke:#d4b52e
    style Workspaces fill:#6bcb77,color:#fff,stroke:#4a9e5a
    style Contracts fill:#ff8c42,color:#fff,stroke:#d6712e
    style StateStore fill:#a855f7,color:#fff,stroke:#8b3fd6
    style CLI fill:#64748b,color:#fff,stroke:#475569
```

## What Each Component Does

| Component | Purpose |
|-----------|---------|
| **Manifest** | The configuration file (`wave.yaml`) that defines everything: which pipelines exist, which personas are available, and how contracts validate output |
| **Pipeline Executor** | The orchestration engine — reads the manifest, resolves the order of steps, and runs them one by one (or in parallel where possible) |
| **Preflight Checker** | Verifies all required tools and skills are available before the pipeline starts |
| **DAG Resolver** | Determines the correct execution order by analyzing step dependencies — ensures no step runs before its prerequisites are complete |
| **Personas** | Specialized AI agent roles (e.g., navigator, craftsman, reviewer) — each has a defined behavior, permissions, and system prompt |
| **Workspaces** | Isolated directories where each step runs — prevents steps from interfering with each other |
| **Adapters** | The bridge between Wave and the AI CLI tool (Claude Code) — handles subprocess launching and output parsing |
| **Contracts** | Validation rules that check whether a step's output meets quality requirements (JSON schema, test suites, etc.) |
| **State Store** | SQLite database that tracks pipeline progress — enables pausing and resuming pipelines |
| **Event System** | Real-time progress notifications — powers the terminal dashboard and monitoring |
| **Relay Monitor** | Watches token usage during long tasks — triggers context compaction when approaching limits |
| **Audit Logger** | Records all tool calls and actions with credential scrubbing for security and traceability |
