# Prompt Engineering Layers

Every AI agent in Wave receives a carefully assembled instruction document called
**CLAUDE.md**. This document tells the agent who it is, what it should do, what rules
it must follow, and what output format is expected. It is built from four layers,
stacked on top of each other like a sandwich.

This ensures every agent receives consistent operational rules while also getting
role-specific guidance for its particular task.

## CLAUDE.md Assembly

This diagram shows how the four layers combine to form the complete instruction
document for each pipeline step.

```mermaid
graph TD
    subgraph Assembly["CLAUDE.md Assembly"]
        direction TB
        L1["📜 Layer 1: Base Protocol<br/>Shared rules for all agents"]
        L2["🎭 Layer 2: Persona Prompt<br/>Role-specific behavior & constraints"]
        L3["📝 Layer 3: Contract Compliance<br/>Output format & validation rules"]
        L4["🔒 Layer 4: Restrictions<br/>Tool permissions & network access"]

        L1 --> L2
        L2 --> L3
        L3 --> L4
    end

    L4 --> Final["📄 Final CLAUDE.md<br/>Written to workspace<br/>before agent starts"]

    style L1 fill:#4a9eff,color:#fff
    style L2 fill:#ffd93d,color:#333
    style L3 fill:#ff8c42,color:#fff
    style L4 fill:#ff6b6b,color:#fff
    style Final fill:#6bcb77,color:#fff
```

### What Each Layer Contains

| Layer | Source | Purpose |
|-------|--------|---------|
| **Base Protocol** | `.wave/personas/base-protocol.md` | Universal rules: fresh context per step, artifact I/O conventions, workspace isolation, no memory of prior steps |
| **Persona Prompt** | `.wave/personas/<name>.md` | Role definition: responsibilities, anti-patterns, quality checklist (e.g., "You are a senior developer focused on clean implementation") |
| **Contract Compliance** | Auto-generated from step config | Output requirements: where to write artifacts, expected JSON schema, validation commands |
| **Restrictions** | Auto-generated from manifest permissions | Security constraints: which tools are denied, which are allowed, which network domains are accessible |

## Persona Definition

Each persona is a named agent role defined in the manifest (`wave.yaml`). Beyond the
system prompt, a persona carries configuration that controls how it behaves at runtime.

```mermaid
graph TD
    Persona["🎭 Persona Definition"]

    Persona --> Prompt["📜 System Prompt<br/>Role, responsibilities,<br/>constraints, anti-patterns"]
    Persona --> Permissions["🔐 Permissions<br/>Allowed & denied<br/>tool lists"]
    Persona --> Model["🧠 Model<br/>Which AI model<br/>to use"]
    Persona --> Temp["🌡️ Temperature<br/>Creativity vs<br/>precision setting"]
    Persona --> Sandbox["🏖️ Sandbox<br/>Network domain<br/>allowlist"]
    Persona --> Hooks["🪝 Hooks<br/>Pre/post tool-use<br/>actions"]

    Permissions --> Enforce["⚡ Enforced at Runtime<br/>Written to settings.json<br/>AND CLAUDE.md restrictions"]
    Sandbox --> Enforce

    style Persona fill:#ffd93d,color:#333
    style Prompt fill:#4a9eff,color:#fff
    style Permissions fill:#ff6b6b,color:#fff
    style Model fill:#a855f7,color:#fff
    style Temp fill:#ff8c42,color:#fff
    style Sandbox fill:#6bcb77,color:#fff
    style Hooks fill:#64748b,color:#fff
    style Enforce fill:#ff6b6b,color:#fff
```

## Built-in Personas

Wave ships with several built-in personas, each designed for a specific role in the
development workflow:

| Persona | Role | Key Trait |
|---------|------|-----------|
| **Navigator** | Codebase explorer | Read-only — never modifies files |
| **Craftsman** | Senior developer | Writes production code, runs tests |
| **Implementer** | Code executor | Applies changes, builds, and validates |
| **Reviewer** | Quality auditor | Reviews code without modifying it |
| **Planner** | Task decomposer | Creates plans, never writes code |
| **Summarizer** | Context compactor | Condenses long conversations for relay |

Each persona's permissions are strictly enforced — a navigator cannot write files,
and a reviewer cannot modify source code, regardless of what they are asked to do.
