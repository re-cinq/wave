# Wave Architecture Overview

This document provides a comprehensive visual overview of Wave's architecture based on codebase analysis.

## Core Concepts

```mermaid
flowchart TB
    subgraph Manifest["wave.yaml (Manifest)"]
        direction TB
        ADAPTERS[Adapters<br/><small>claude, opencode</small>]
        PERSONAS[Personas<br/><small>navigator, implementer, reviewer</small>]
        RUNTIME[Runtime Config<br/><small>workspace, relay, audit</small>]
    end

    subgraph Pipeline["Pipeline"]
        direction TB
        META[Metadata<br/><small>name, description</small>]

        subgraph Steps["Steps (DAG)"]
            direction LR
            S1[Step 1<br/><small>persona: navigator</small>]
            S2[Step 2<br/><small>persona: implementer</small>]
            S3[Step 3<br/><small>persona: reviewer</small>]
            S1 --> S2 --> S3
        end
    end

    subgraph StepDetail["Step Execution"]
        direction TB
        WS[Workspace<br/><small>ephemeral, isolated</small>]
        EXEC[Exec<br/><small>prompt template</small>]

        subgraph Handover["Handover"]
            ART[Artifacts<br/><small>output files</small>]
            CONT[Contract<br/><small>validation schema</small>]
        end
    end

    Manifest --> Pipeline
    PERSONAS -.->|binds| Steps
    Pipeline --> StepDetail
    ART -->|inject into next step| Steps

    style Manifest fill:#e0f2fe,stroke:#0284c7
    style Pipeline fill:#fef3c7,stroke:#f59e0b
    style StepDetail fill:#dcfce7,stroke:#22c55e
    style Handover fill:#f3e8ff,stroke:#a855f7
```

## Concept Relationships

```mermaid
erDiagram
    MANIFEST ||--o{ PERSONA : defines
    MANIFEST ||--o{ ADAPTER : configures
    MANIFEST ||--|| RUNTIME : sets

    PIPELINE ||--o{ STEP : contains
    PIPELINE }|--|| MANIFEST : "reads from"

    STEP }|--|| PERSONA : "uses"
    STEP ||--o{ ARTIFACT : "produces"
    STEP ||--o| CONTRACT : "validates with"
    STEP ||--o{ STEP : "depends on"

    ARTIFACT }o--|| STEP : "injected into"

    PERSONA }|--|| ADAPTER : "runs via"
    PERSONA ||--o{ PERMISSION : "has"

    CONTRACT ||--o{ QUALITY_GATE : "includes"
```

## Simplified Flow

```mermaid
flowchart LR
    subgraph Input
        USER[User Input]
        YAML[Pipeline YAML]
    end

    subgraph Execution
        direction TB
        STEP1[Step 1] --> ART1[Artifact]
        ART1 --> STEP2[Step 2]
        STEP2 --> ART2[Artifact]
        ART2 --> STEP3[Step 3]
    end

    subgraph Validation
        CONTRACT[Contract Check]
    end

    subgraph Output
        RESULT[Pipeline Result]
    end

    USER --> Execution
    YAML --> Execution
    STEP1 & STEP2 & STEP3 --> CONTRACT
    CONTRACT --> RESULT

    style Input fill:#f0f9ff
    style Execution fill:#fefce8
    style Validation fill:#fdf4ff
    style Output fill:#f0fdf4
```

## High-Level System Architecture

```mermaid
graph TB
    subgraph CLI["CLI Layer"]
        CMD[wave CLI]
        RUN[wave run]
        OPS[wave ops]
        MIGRATE[wave migrate]
    end

    subgraph Core["Core Engine"]
        MANIFEST[Manifest Loader]
        ROUTER[Pipeline Router]
        EXECUTOR[Pipeline Executor]
        DAG[DAG Validator]
    end

    subgraph Execution["Execution Layer"]
        WORKSPACE[Workspace Manager]
        ADAPTER[Adapter Runner]
        CONTRACT[Contract Validator]
        MATRIX[Matrix Executor]
    end

    subgraph Persistence["Persistence Layer"]
        STATE[(SQLite State)]
        FS[Filesystem]
        TRACES[Audit Traces]
    end

    subgraph Security["Security Layer"]
        SANITIZE[Input Sanitizer]
        PERMS[Permission Checker]
        SCRUB[Credential Scrubber]
    end

    CMD --> RUN & OPS & MIGRATE
    RUN --> MANIFEST
    MANIFEST --> ROUTER
    ROUTER --> EXECUTOR
    EXECUTOR --> DAG
    DAG --> WORKSPACE
    WORKSPACE --> ADAPTER
    ADAPTER --> CONTRACT
    EXECUTOR --> MATRIX

    EXECUTOR --> STATE
    WORKSPACE --> FS
    ADAPTER --> TRACES

    SANITIZE --> EXECUTOR
    PERMS --> ADAPTER
    SCRUB --> TRACES
```

## Pipeline Execution Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Manifest
    participant Router
    participant Executor
    participant DAG
    participant Workspace
    participant Adapter
    participant Contract
    participant State

    User->>CLI: wave run pipeline "input"
    CLI->>Manifest: Load wave.yaml
    Manifest->>Router: Route(input, labels)
    Router->>Executor: Execute(pipeline, input)

    Executor->>DAG: ValidateDAG(pipeline)
    DAG->>DAG: TopologicalSort(steps)
    DAG-->>Executor: sorted steps[]

    Executor->>State: SavePipelineState(running)

    loop For each step
        Executor->>State: SaveStepState(running)
        Executor->>Workspace: Create(config)
        Workspace->>Workspace: InjectArtifacts()
        Executor->>Adapter: Run(persona, prompt)
        Adapter-->>Executor: result

        alt Has Contract
            Executor->>Contract: Validate(output)
            Contract-->>Executor: pass/fail
        end

        Executor->>State: SaveStepState(completed)
    end

    Executor->>State: SavePipelineState(completed)
    Executor-->>CLI: success
    CLI-->>User: Pipeline completed
```

## Step Execution Detail

```mermaid
flowchart TD
    START([Step Start]) --> CREATE[Create Workspace]
    CREATE --> INJECT[Inject Artifacts]
    INJECT --> BUILD[Build Prompt]
    BUILD --> RESOLVE[Resolve Templates]
    RESOLVE --> SANITIZE[Sanitize Input]

    SANITIZE --> PERMS{Check Permissions}
    PERMS -->|Denied| FAIL([Step Failed])
    PERMS -->|Allowed| RUN[Run Adapter]

    RUN --> RESULT{Exit Code}
    RESULT -->|Error| RETRY{Retries Left?}
    RETRY -->|Yes| RUN
    RETRY -->|No| FAIL

    RESULT -->|Success| ARTIFACTS[Register Artifacts]
    ARTIFACTS --> CONTRACT{Has Contract?}

    CONTRACT -->|No| COMPLETE([Step Complete])
    CONTRACT -->|Yes| VALIDATE[Validate Contract]

    VALIDATE --> VALID{Valid?}
    VALID -->|Yes| COMPLETE
    VALID -->|No & must_pass| FAIL
    VALID -->|No & !must_pass| WARN[Log Warning]
    WARN --> COMPLETE
```

## Persona Permission Model

```mermaid
flowchart TD
    subgraph Personas
        NAV[Navigator<br/>Read-only exploration]
        IMPL[Implementer<br/>Full write access]
        REV[Reviewer<br/>Limited write]
        AUD[Auditor<br/>Read + audit tools]
        CRAFT[Craftsman<br/>Full access]
        PHIL[Philosopher<br/>Spec writing only]
        PLAN[Planner<br/>No tools]
    end

    subgraph Tools
        READ[Read]
        WRITE[Write]
        EDIT[Edit]
        BASH[Bash]
        GLOB[Glob]
        GREP[Grep]
    end

    NAV --> READ & GLOB & GREP
    IMPL --> READ & WRITE & EDIT & BASH & GLOB & GREP
    REV --> READ & GLOB & GREP
    AUD --> READ & GLOB & GREP
    CRAFT --> READ & WRITE & EDIT & BASH & GLOB & GREP
    PHIL --> READ & GLOB & GREP
    PLAN --> READ & GLOB & GREP

    style NAV fill:#e1f5fe
    style IMPL fill:#c8e6c9
    style REV fill:#fff3e0
    style AUD fill:#fce4ec
    style CRAFT fill:#c8e6c9
    style PHIL fill:#f3e5f5
    style PLAN fill:#eceff1
```

## Permission Check Flow

```mermaid
flowchart TD
    REQ[Permission Request<br/>tool + argument] --> DENY{Match Deny Pattern?}
    DENY -->|Yes| BLOCKED([DENIED])
    DENY -->|No| ALLOW_DEF{Allow Patterns Defined?}
    ALLOW_DEF -->|No| PERMITTED([ALLOWED])
    ALLOW_DEF -->|Yes| ALLOW{Match Allow Pattern?}
    ALLOW -->|Yes| PERMITTED
    ALLOW -->|No| BLOCKED

    style BLOCKED fill:#ffcdd2
    style PERMITTED fill:#c8e6c9
```

## Contract Validation System

```mermaid
flowchart TD
    subgraph Types["Contract Types"]
        JSON[JSON Schema]
        TS[TypeScript Interface]
        TEST[Test Suite]
        MD[Markdown Spec]
    end

    OUTPUT[Step Output] --> DETECT[Detect Contract Type]
    DETECT --> JSON & TS & TEST & MD

    JSON --> JSONV[JSON Schema Validator<br/>RFC 7159]
    TS --> TSV[TypeScript Validator<br/>tsc --noEmit]
    TEST --> TESTV[Test Suite Validator<br/>Command execution]
    MD --> MDV[Markdown Validator<br/>Structure check]

    JSONV & TSV & TESTV & MDV --> RESULT{Valid?}

    RESULT -->|Yes| GATES[Quality Gates]
    RESULT -->|No| RECOVERY{Recovery Enabled?}

    RECOVERY -->|Yes| RECOVER[JSON Recovery]
    RECOVER --> RESULT
    RECOVERY -->|No| FAIL([Validation Failed])

    GATES --> PASS([Contract Passed])
```

## State Management Schema

```mermaid
erDiagram
    pipeline_run {
        text run_id PK
        text pipeline_name
        text status
        text input
        text current_step
        int total_tokens
        text tags_json
        timestamp started_at
        timestamp completed_at
    }

    step_state {
        text step_id PK
        text pipeline_id FK
        text state
        int retry_count
        text workspace_path
        text error_message
        timestamp started_at
        timestamp completed_at
    }

    event_log {
        int id PK
        text run_id FK
        text step_id
        text state
        text persona
        text message
        int tokens_used
        int duration_ms
        timestamp timestamp
    }

    artifact {
        int id PK
        text run_id FK
        text step_id
        text name
        text path
        text type
        int size_bytes
        timestamp created_at
    }

    performance_metric {
        int id PK
        text run_id FK
        text step_id
        text persona
        int duration_ms
        int tokens_used
        int files_modified
        bool success
    }

    pipeline_run ||--o{ step_state : contains
    pipeline_run ||--o{ event_log : logs
    pipeline_run ||--o{ artifact : produces
    pipeline_run ||--o{ performance_metric : tracks
```

## Workspace Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Created: Create workspace
    Created --> Mounted: Mount sources
    Mounted --> Injected: Inject artifacts
    Injected --> Executing: Adapter runs
    Executing --> Artifacts: Write outputs
    Artifacts --> Validated: Contract check
    Validated --> Preserved: Keep for dependents
    Preserved --> [*]: Pipeline complete

    note right of Created
        .wave/workspaces/
        {pipeline}/{step}/
    end note

    note right of Injected
        artifacts/
        {step}_{artifact}
    end note
```

## Artifact Flow Between Steps

```mermaid
flowchart LR
    subgraph Step1["Step: spec"]
        S1OUT[output_artifacts:<br/>- spec.md]
    end

    subgraph Step2["Step: docs"]
        S2IN[inject_artifacts:<br/>- step: spec<br/>  artifact: spec]
        S2OUT[output_artifacts:<br/>- docs.md]
    end

    subgraph Step3["Step: implement"]
        S3IN[inject_artifacts:<br/>- step: spec<br/>- step: docs]
    end

    S1OUT -->|"artifacts/spec_spec.md"| S2IN
    S1OUT -->|"artifacts/spec_spec.md"| S3IN
    S2OUT -->|"artifacts/docs_docs.md"| S3IN
```

## Matrix Strategy Execution

```mermaid
flowchart TD
    MATRIX[Matrix Step] --> LOAD[Load items_source JSON]
    LOAD --> EXTRACT[Extract items via item_key]
    EXTRACT --> SPAWN[Spawn Workers]

    subgraph Workers["Parallel Workers (max_concurrency)"]
        W1[Worker 1<br/>item: task1]
        W2[Worker 2<br/>item: task2]
        W3[Worker 3<br/>item: task3]
        WN[Worker N<br/>item: taskN]
    end

    SPAWN --> W1 & W2 & W3 & WN

    W1 & W2 & W3 & WN --> COLLECT[Collect Results]
    COLLECT --> CONFLICT{File Conflicts?}
    CONFLICT -->|Yes| FAIL([Abort Pipeline])
    CONFLICT -->|No| AGGREGATE[Aggregate Results]
    AGGREGATE --> DONE([Matrix Complete])
```

## Security Validation Flow

```mermaid
flowchart TD
    INPUT[User Input] --> SANITIZE[Input Sanitizer]

    subgraph Sanitization
        PROMPT[Prompt Injection Detection]
        LENGTH[Length Validation]
        CONTENT[Content Cleaning]
    end

    SANITIZE --> PROMPT & LENGTH & CONTENT

    PROMPT & LENGTH & CONTENT --> RISK[Risk Score Calculation]
    RISK --> PATH[Path Validation]

    subgraph PathValidation
        TRAVERSE[Traversal Detection]
        APPROVED[Approved Directory Check]
        SYMLINK[Symlink Detection]
    end

    PATH --> TRAVERSE & APPROVED & SYMLINK

    TRAVERSE & APPROVED & SYMLINK --> PERMS[Permission Check]
    PERMS --> SCRUB[Credential Scrubbing]
    SCRUB --> LOG[Audit Log]

    LOG --> TRACE[".wave/traces/<br/>trace-TIMESTAMP.log"]
```

## Resume Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant ResumeManager
    participant StateStore
    participant Filesystem
    participant Executor

    User->>CLI: wave run pipeline --resume-from step3
    CLI->>ResumeManager: ResumeFromStep(pipeline, step3)

    ResumeManager->>ResumeManager: ValidateResumePoint()
    ResumeManager->>Filesystem: Scan .wave/workspaces/{pipeline}/
    Filesystem-->>ResumeManager: completed step workspaces

    ResumeManager->>StateStore: GetStepStates(pipeline)
    StateStore-->>ResumeManager: step states

    ResumeManager->>ResumeManager: LoadResumeState()
    Note over ResumeManager: Index artifact paths<br/>from completed steps

    ResumeManager->>ResumeManager: CreateResumeSubpipeline()
    Note over ResumeManager: steps[step3:]

    ResumeManager->>Executor: ExecuteResumedPipeline()
    Executor->>Executor: Inject preserved artifacts
    Executor->>Executor: Execute remaining steps
    Executor-->>CLI: Pipeline resumed and completed
```

## Event Emission

```mermaid
flowchart LR
    subgraph Events["Event Types"]
        START[pipeline_started]
        STEP[step_started/completed/failed]
        PROG[step_progress]
        CONTRACT[contract_passed/failed]
        COMP[compaction_triggered]
        END[pipeline_completed]
    end

    subgraph Output["Output Modes"]
        JSON[NDJSON to stdout]
        HUMAN[Human-readable to stderr]
        PROGRESS[Progress spinner]
    end

    START & STEP & PROG & CONTRACT & COMP & END --> EMITTER[Event Emitter]
    EMITTER --> JSON & HUMAN & PROGRESS
```

## Complete Data Flow

```mermaid
flowchart TB
    subgraph Input
        YAML[wave.yaml<br/>Manifest]
        PIPELINE[pipeline.yaml<br/>Pipeline Definition]
        PERSONA[.wave/personas/<br/>System Prompts]
        CONTRACT[.wave/contracts/<br/>Schemas]
    end

    subgraph Processing
        LOAD[Load & Parse]
        ROUTE[Route to Pipeline]
        SORT[Topological Sort]
        EXEC[Execute Steps]
    end

    subgraph Runtime
        WS[Workspace<br/>Isolation]
        ADAPT[Adapter<br/>Subprocess]
        VALID[Contract<br/>Validation]
    end

    subgraph Output
        STATE[(SQLite<br/>State)]
        ARTIFACTS[Artifacts<br/>Filesystem]
        TRACES[Audit<br/>Traces]
        EVENTS[Progress<br/>Events]
    end

    YAML & PIPELINE & PERSONA & CONTRACT --> LOAD
    LOAD --> ROUTE --> SORT --> EXEC
    EXEC --> WS --> ADAPT --> VALID
    VALID --> STATE & ARTIFACTS & TRACES & EVENTS
```
