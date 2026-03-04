# Pipeline Execution Lifecycle

A pipeline in Wave is a series of connected steps that form a **directed acyclic graph**
(DAG) — a fancy way of saying "steps have dependencies, but there are no circular loops."
The Pipeline Executor processes these steps in the right order, running independent steps
in parallel when possible.

This diagram shows what happens from the moment a pipeline is triggered until it completes.

```mermaid
sequenceDiagram
    participant User
    participant Executor as Pipeline Executor
    participant DAG as DAG Resolver
    participant Preflight as Preflight Checker
    participant State as State Store
    participant WS as Workspace Manager
    participant Adapter as Adapter
    participant CLI as Claude Code CLI
    participant Contract as Contract Validator
    participant Events as Event System

    User->>Executor: Start pipeline

    rect rgb(240, 248, 255)
        Note over Executor,Preflight: Initialization
        Executor->>DAG: Resolve step dependencies
        DAG-->>Executor: Sorted execution order
        Executor->>Preflight: Check required tools & skills
        Preflight-->>Executor: All prerequisites met
        Executor->>State: Create pipeline run record
        Executor->>Events: Emit "pipeline started"
    end

    loop For each step (in dependency order)
        rect rgb(245, 255, 245)
            Note over Executor,Events: Step Execution
            Executor->>Events: Emit "step started"

            Executor->>WS: Create isolated workspace
            WS-->>Executor: Workspace directory ready

            Executor->>WS: Inject artifacts from prior steps
            WS-->>Executor: Artifacts copied to .wave/artifacts/

            Executor->>Adapter: Prepare persona & prompt
            Note over Adapter: Assembles CLAUDE.md:<br/>base protocol + persona<br/>+ contract + restrictions

            Adapter->>CLI: Launch subprocess
            CLI-->>Adapter: Stream progress (NDJSON)
            Adapter->>Events: Forward activity events
            CLI-->>Adapter: Final result + token usage

            Executor->>Contract: Validate step output
            alt Validation passes
                Contract-->>Executor: Output meets requirements
            else Validation fails (strict)
                Contract-->>Executor: Block — step failed
                Executor->>Events: Emit "step failed"
            end

            Executor->>State: Save step result & artifacts
            Executor->>Events: Emit "step completed"
        end
    end

    Executor->>State: Mark pipeline completed
    Executor->>Events: Emit "pipeline completed"
    Executor-->>User: Pipeline finished
```

## Step Execution in Detail

Each step in the pipeline goes through these stages:

```mermaid
graph LR
    A["🔀 Find Ready<br/>Steps"] --> B["📁 Create<br/>Workspace"]
    B --> C["📥 Inject<br/>Artifacts"]
    C --> D["🎭 Bind<br/>Persona"]
    D --> E["🚀 Run<br/>Adapter"]
    E --> F["📝 Validate<br/>Contract"]
    F --> G["💾 Save<br/>State"]
    G --> H["📤 Register<br/>Artifacts"]

    style A fill:#4a9eff,color:#fff
    style B fill:#6bcb77,color:#fff
    style C fill:#ffd93d,color:#333
    style D fill:#ff8c42,color:#fff
    style E fill:#ff6b6b,color:#fff
    style F fill:#a855f7,color:#fff
    style G fill:#64748b,color:#fff
    style H fill:#4a9eff,color:#fff
```

## Key Concepts

### Dependency Resolution
Steps declare which other steps they depend on. The DAG Resolver sorts them so that
a step never runs before its prerequisites. Steps with no mutual dependencies can run
in parallel.

### Artifact Flow
When a step completes, its output (called an **artifact**) is saved. Downstream steps
that depend on it receive those artifacts automatically — they appear as files in the
step's workspace under `.wave/artifacts/`.

### Retry and Resume
If a step fails, the executor can retry it (up to a configurable limit). If the entire
pipeline is interrupted, it can be resumed from the last successful step — the State
Store tracks exactly where execution left off.

### Contract Validation
After each step runs, its output is validated against a **contract** — a set of rules
defining what valid output looks like. This catches errors early, before they propagate
to downstream steps.
