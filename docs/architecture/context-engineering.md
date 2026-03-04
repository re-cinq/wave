# Context Engineering

Context engineering is how Wave controls **what each AI agent knows** when it starts
working. Unlike a human developer who remembers everything from the day, each Wave
agent starts with a completely blank slate — it has no memory of what happened before.
The only information it receives is what Wave explicitly provides.

This is by design: it prevents agents from being confused by irrelevant context,
ensures reproducible behavior, and maintains security boundaries between steps.

## Artifact Flow Between Steps

Artifacts are the files that steps produce as output and pass to downstream steps.
They are the primary mechanism for inter-step communication.

```mermaid
graph LR
    subgraph Step1["Step 1: Analyze"]
        S1Work["Agent works<br/>in isolation"]
        S1Out["📤 Output:<br/>analysis.json"]
    end

    subgraph Step2["Step 2: Plan"]
        S2In["📥 Input:<br/>analysis.json"]
        S2Work["Agent works<br/>in isolation"]
        S2Out["📤 Output:<br/>plan.md"]
    end

    subgraph Step3["Step 3: Implement"]
        S3In["📥 Inputs:<br/>analysis.json<br/>plan.md"]
        S3Work["Agent works<br/>in isolation"]
        S3Out["📤 Output:<br/>code changes"]
    end

    S1Work --> S1Out
    S1Out -->|"copied to<br/>.wave/artifacts/"| S2In
    S2In --> S2Work --> S2Out
    S1Out -->|"copied to<br/>.wave/artifacts/"| S3In
    S2Out -->|"copied to<br/>.wave/artifacts/"| S3In
    S3In --> S3Work --> S3Out

    style S1Out fill:#ffd93d,color:#333
    style S2In fill:#4a9eff,color:#fff
    style S2Out fill:#ffd93d,color:#333
    style S3In fill:#4a9eff,color:#fff
    style S3Out fill:#6bcb77,color:#fff
```

### How Artifact Injection Works

1. A step completes and produces an output artifact (a file)
2. Wave saves this artifact and records its location
3. When a downstream step starts, Wave checks which artifacts it needs
4. Those artifacts are **copied** into the new step's workspace under `.wave/artifacts/`
5. The agent reads these files to understand what the previous step produced
6. If a schema is specified, Wave validates the artifact format before injection

Only explicitly declared dependencies are injected — a step cannot access artifacts
from steps it does not depend on.

## What Each Agent Sees

When an agent starts, it receives exactly three things — nothing more, nothing less:

```mermaid
graph TD
    subgraph Context["What the Agent Receives"]
        direction TB
        CLAUDE["📄 CLAUDE.md<br/>Assembled instruction document<br/>(base protocol + persona + contract + restrictions)"]
        Artifacts["📥 Injected Artifacts<br/>Output files from<br/>dependency steps only"]
        Workspace["📁 Clean Workspace<br/>Isolated directory with<br/>no prior history"]
    end

    subgraph Absent["What the Agent Does NOT Have"]
        direction TB
        NoPrior["❌ No prior chat history"]
        NoOther["❌ No artifacts from<br/>non-dependency steps"]
        NoEnv["❌ No host environment<br/>variables (except allowed)"]
    end

    style CLAUDE fill:#4a9eff,color:#fff
    style Artifacts fill:#ffd93d,color:#333
    style Workspace fill:#6bcb77,color:#fff
    style NoPrior fill:#ffcccc,color:#333
    style NoOther fill:#ffcccc,color:#333
    style NoEnv fill:#ffcccc,color:#333
```

## Fresh Memory Principle

Each step begins with **zero memory** of what happened before. This is the "fresh
memory at step boundaries" principle. Even if the same persona runs in two consecutive
steps, it starts clean each time.

**Why?** Because:
- It prevents context pollution — agents cannot be confused by irrelevant history
- It ensures reproducibility — the same inputs always produce the same behavior
- It enforces security — an agent cannot leak information from one step to another
- It keeps token usage efficient — agents do not waste tokens on irrelevant context

## Relay: Handling Long-Running Tasks

Sometimes a single step requires more work than fits in the AI model's context window
(roughly 200,000 tokens). The **Relay Monitor** watches token usage and triggers
compaction when needed.

```mermaid
graph TD
    Agent["🎭 Agent Working"]
    Monitor["🔄 Relay Monitor<br/>Watches token usage"]
    Threshold{"Token usage<br/>exceeds 70%?"}
    Summarizer["📝 Summarizer Persona<br/>Condenses conversation"]
    Checkpoint["💾 Checkpoint<br/>Summary saved"]
    Resume["🔄 Fresh Agent<br/>Resumes with summary"]

    Agent --> Monitor
    Monitor --> Threshold
    Threshold -->|"No"| Agent
    Threshold -->|"Yes"| Summarizer
    Summarizer --> Checkpoint
    Checkpoint --> Resume
    Resume --> Agent

    style Agent fill:#ffd93d,color:#333
    style Monitor fill:#4a9eff,color:#fff
    style Threshold fill:#ff8c42,color:#fff
    style Summarizer fill:#a855f7,color:#fff
    style Checkpoint fill:#6bcb77,color:#fff
    style Resume fill:#64748b,color:#fff
```

### How Relay Works

1. The Relay Monitor tracks how many tokens the agent has used
2. When usage exceeds ~70% of the context window, compaction triggers
3. A specialized **Summarizer** persona reads the conversation and produces a concise summary
4. This summary is saved as a **checkpoint**
5. A fresh agent instance starts with the checkpoint summary instead of the full history
6. The agent continues working from where it left off, but with a much smaller context
