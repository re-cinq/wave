# Pipeline Schema Reference

Complete field reference for Wave pipeline YAML files — DAG definitions that orchestrate multi-step agent workflows.

Pipeline files are stored in `.wave/pipelines/` by convention and referenced from `wave run --pipeline <path>`.

## Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | `string` | **yes** | Must be `"WavePipeline"`. |
| `metadata` | [`PipelineMetadata`](#pipelinemetadata) | **yes** | Pipeline identification. |
| `input` | [`InputConfig`](#inputconfig) | no | Work item source configuration. |
| `steps` | [`[]Step`](#step) | **yes** | Ordered list of pipeline steps forming a DAG. |

### Minimal Example

```yaml
kind: WavePipeline
metadata:
  name: simple-fix
steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      root: ./
    exec:
      type: prompt
      source: "Analyze the codebase structure."
```

---

## PipelineMetadata

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | `string` | **yes** | — | Pipeline name. Used in events, state persistence, and workspace paths. Must be unique within a project. |
| `description` | `string` | no | `""` | Human-readable pipeline purpose. |

---

## InputConfig

Configuration for how the pipeline receives its initial work item.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `source` | `string` | no | `"cli"` | Input source: `"cli"` (from `--input` flag), `"file"` (read from path), `"stdin"`. |
| `path` | `string` | no | — | File path when `source: file`. |

```yaml
input:
  source: cli  # Accepts --input "task description"
```

---

## Step

A single unit of work in the pipeline DAG. Each step executes one persona in one ephemeral workspace.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | `string` | **yes** | — | Unique step identifier within this pipeline. Used in dependency references, events, and workspace paths. |
| `persona` | `string` | **yes** | — | References a persona key defined in the project manifest (`wave.yaml`). |
| `dependencies` | `[]string` | no | `[]` | Step IDs that must complete successfully before this step starts. Forms the DAG edges. |
| `memory` | [`MemoryConfig`](#memoryconfig) | **yes** | — | Memory strategy and artifact injection. |
| `workspace` | [`WorkspaceConfig`](#workspaceconfig) | no | auto-generated | Workspace directory and mount configuration. |
| `exec` | [`ExecConfig`](#execconfig) | **yes** | — | What this step actually does — prompt or command. |
| `output_artifacts` | [`[]ArtifactDef`](#artifactdef) | no | `[]` | Expected output files/directories produced by this step. |
| `handover` | [`HandoverConfig`](#handoverconfig) | no | `{}` | Contract validation and compaction settings at step boundary. |
| `strategy` | [`MatrixStrategy`](#matrixstrategy) | no | `null` | Fan-out parallel execution configuration. |
| `validation` | [`[]ValidationRule`](#validationrule) | no | `[]` | Pre-execution validation checks. |

### Step State Machine

```
Pending ──→ Running ──→ Completed
                ├──→ Retrying ──→ Running (retry attempt)
                └──→ Failed (max retries exceeded)
```

| State | Description |
|-------|-------------|
| `pending` | Step is queued, waiting for dependencies to complete. |
| `running` | Step is actively executing. Relay/compaction is a sub-state of running. |
| `completed` | Step finished successfully. Output artifacts are available. |
| `retrying` | Step failed (contract violation, crash, or timeout) and is retrying. |
| `failed` | Step exceeded `max_retries`. Pipeline halts. |

Only `pending` and `failed` steps are resumable via `wave resume`.

---

## MemoryConfig

Controls how context is managed for a step.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `strategy` | `string` | **yes** | — | Memory strategy. Must be `"fresh"` — each step always starts with a clean context. This is a core design principle. |
| `inject_artifacts` | [`[]ArtifactRef`](#artifactref) | no | `[]` | Artifacts from prior steps to inject into this step's context. |

### ArtifactRef

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `step` | `string` | **yes** | Source step ID that produced the artifact. |
| `artifact` | `string` | **yes** | Artifact name from the source step's `output_artifacts`. |
| `as` | `string` | **yes** | Name for this artifact in the current step's workspace. |

```yaml
memory:
  strategy: fresh
  inject_artifacts:
    - step: navigate
      artifact: analysis
      as: navigation_report
    - step: specify
      artifact: spec
      as: feature_spec
```

---

## WorkspaceConfig

Ephemeral workspace configuration for a step.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `root` | `string` | no | auto-generated | Workspace root directory path template. Default: `<runtime.workspace_root>/<pipeline_id>/<step_id>/` |
| `mount` | [`[]Mount`](#mount) | no | `[]` | Filesystem mounts into the workspace. |

### Mount

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `source` | `string` | **yes** | — | Source directory path (absolute or relative to project root). |
| `target` | `string` | **yes** | — | Mount point within the workspace. |
| `mode` | `string` | no | `"readonly"` | Access mode: `"readonly"` or `"readwrite"`. |

```yaml
workspace:
  root: ./
  mount:
    - source: ./
      target: /src
      mode: readonly
    - source: ./test-fixtures
      target: /fixtures
      mode: readonly
```

### Workspace Directory Structure

Each step gets an isolated workspace:

```
/tmp/wave/<pipeline-id>/<step-id>/
├── src/              # Mounted from repo (readonly by default)
├── artifacts/        # Injected artifacts from dependencies
├── output/           # Step output artifacts
├── .claude/          # Adapter configuration
└── CLAUDE.md         # Persona system prompt
```

Workspaces persist until explicitly cleaned with `wave clean`. They are **never** auto-deleted.

---

## ExecConfig

Defines what a step executes.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | `string` | **yes** | Execution type: `"prompt"` (send text to LLM) or `"command"` (run shell command). |
| `source` | `string` | **yes** | The prompt template or shell command to execute. Supports `{{ input }}` and `{{ task }}` template variables. |

### Prompt Execution

```yaml
exec:
  type: prompt
  source: |
    Analyze the codebase structure for {{ input }}.
    Report file paths, patterns, and architectural decisions.
```

### Command Execution

```yaml
exec:
  type: command
  source: "go test -run TestMemoryLeak -v -count=1"
```

### Template Variables

| Variable | Scope | Description |
|----------|-------|-------------|
| `{{ input }}` | All steps | Pipeline input from `--input` flag or input config. |
| `{{ task }}` | Matrix steps | Current task item from matrix strategy. |

---

## ArtifactDef

Declares expected output artifacts from a step.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | **yes** | Artifact identifier. Referenced by downstream steps in `inject_artifacts`. |
| `path` | `string` | **yes** | File or directory path relative to the step's workspace output directory. |
| `type` | `string` | no | Content type hint: `"file"`, `"directory"`, `"json"`, `"markdown"`. |

```yaml
output_artifacts:
  - name: analysis
    path: output/analysis.json
    type: json
  - name: source_map
    path: output/files.md
    type: markdown
```

---

## HandoverConfig

Contract validation and compaction settings applied at step boundaries.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `contract` | [`ContractConfig`](#contractconfig) | no | Validation rules for step output. |
| `compaction` | [`CompactionConfig`](#compactionconfig) | no | Context relay settings for this step. |

### ContractConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | `string` | no | — | Contract type: `"json_schema"`, `"typescript_interface"`, `"test_suite"`. |
| `schema` | `string` | no | — | Inline JSON schema or path to schema file. Used with `type: json_schema`. |
| `source` | `string` | no | — | File to validate against the contract. Relative to workspace. |
| `validate` | `bool` | no | `true` | Whether to run compile-time validation. Used with `type: typescript_interface`. |
| `command` | `string` | no | — | Test command to execute. Used with `type: test_suite`. |
| `must_pass` | `bool` | no | `true` | Whether contract failure blocks progression. |
| `on_failure` | `string` | no | `"retry"` | Action on failure: `"retry"` (re-run step) or `"halt"` (stop pipeline). |
| `max_retries` | `int` | no | `2` | Maximum retry attempts before transitioning to `failed`. |

### Contract Types

#### JSON Schema

Validates output structure against a JSON Schema definition.

```yaml
handover:
  contract:
    type: json_schema
    schema: .wave/contracts/navigation-output.schema.json
    source: output/analysis.json
    on_failure: retry
    max_retries: 2
```

#### TypeScript Interface

Validates that generated TypeScript compiles against a declared interface.

```yaml
handover:
  contract:
    type: typescript_interface
    source: output/types.ts
    validate: true
    on_failure: retry
    max_retries: 2
```

::: tip
If TypeScript compilation tools (`tsc`) are not available in the environment, this degrades to syntax-only validation.
:::

#### Test Suite

Validates step output by running a test command.

```yaml
handover:
  contract:
    type: test_suite
    command: "npm test -- --testPathPattern=profile.test"
    must_pass: true
    on_failure: retry
    max_retries: 3
```

### CompactionConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `trigger` | `string` | no | `"token_limit_80%"` | Condition that triggers relay. Format: `"token_limit_<N>%"`. |
| `persona` | `string` | no | `"summarizer"` | Persona to use for checkpoint generation. Must be defined in the manifest. |

```yaml
handover:
  compaction:
    trigger: "token_limit_80%"
    persona: summarizer
```

---

## MatrixStrategy

Fan-out parallel execution. Spawns multiple instances of a step, each processing one item from a task list.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | `string` | **yes** | — | Must be `"matrix"`. |
| `items_source` | `string` | **yes** | — | Path to JSON file containing the task list. Relative to the dependency step's output. Format: `<step_id>/<artifact_path>`. |
| `item_key` | `string` | **yes** | — | JSON key name for individual task items. Available as `{{ task }}` in exec source. |
| `max_concurrency` | `int` | no | `runtime.max_concurrent_workers` | Maximum parallel workers for this matrix. |

### Matrix Example

```yaml
steps:
  - id: plan
    persona: philosopher
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: |
        Break down the task into independent sub-tasks.
        Output as JSON: {"tasks": [{"task": "description"}, ...]}
    output_artifacts:
      - name: tasks
        path: output/tasks.json
        type: json

  - id: execute
    persona: craftsman
    dependencies: [plan]
    strategy:
      type: matrix
      items_source: plan/tasks.json
      item_key: task
      max_concurrency: 4
    memory:
      strategy: fresh
      inject_artifacts:
        - step: plan
          artifact: tasks
          as: task_list
    exec:
      type: prompt
      source: |
        Execute your assigned task: {{ task }}
        Follow the project's coding standards.
```

### Matrix Conflict Detection

When multiple matrix workers modify the same file, the merge phase detects and reports conflicts rather than silently overwriting. Conflicts halt the pipeline with a clear error listing the affected files and workers.

---

## ValidationRule

Pre-execution validation checks run before the step starts.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | `string` | **yes** | Validation type: `"file_exists"`, `"command"`, `"schema"`. |
| `target` | `string` | **yes** | File path or command to validate. |
| `message` | `string` | no | Custom error message on failure. |

```yaml
validation:
  - type: file_exists
    target: src/models/user.go
    message: "User model must exist before profile feature"
  - type: command
    target: "go build ./..."
    message: "Project must compile before implementation step"
```

---

## DAG Rules

Pipeline steps form a Directed Acyclic Graph (DAG). Wave enforces:

1. **No cycles** — Circular dependencies are detected at load time and rejected.
2. **Valid references** — All `dependencies` entries must reference existing step IDs.
3. **Persona references** — All `persona` values must reference personas defined in the manifest.
4. **Topological ordering** — Steps execute in dependency order. Independent steps may execute in parallel (subject to `max_concurrent_workers`).

---

## Complete Pipeline Example

```yaml
kind: WavePipeline
metadata:
  name: speckit-flow
  description: "Full specification-driven development pipeline"

input:
  source: cli

steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      root: ./
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze the codebase for: {{ input }}
        Find relevant files, patterns, and dependencies.
        Output a structured analysis report.
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/navigation.schema.json
        source: output/analysis.json
        on_failure: retry
        max_retries: 2

  - id: specify
    persona: philosopher
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: navigation_report
    exec:
      type: prompt
      source: |
        Based on the navigation report, create a specification for: {{ input }}
        Include data model, API design, and acceptance criteria.
    output_artifacts:
      - name: spec
        path: output/spec.md
        type: markdown
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/specification.schema.json
        source: output/spec.json
        on_failure: retry
        max_retries: 2

  - id: implement
    persona: craftsman
    dependencies: [specify]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: specify
          artifact: spec
          as: feature_spec
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readwrite
    exec:
      type: prompt
      source: |
        Implement the feature according to the specification.
        Write code, tests, and documentation.
    handover:
      contract:
        type: test_suite
        command: "go test ./..."
        must_pass: true
        on_failure: retry
        max_retries: 3
      compaction:
        trigger: "token_limit_80%"
        persona: summarizer

  - id: review
    persona: auditor
    dependencies: [implement]
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: |
        Review the implementation for:
        - Security vulnerabilities (OWASP top 10)
        - Performance regressions
        - Test coverage gaps
        - Code quality issues
    output_artifacts:
      - name: review
        path: output/review.md
        type: markdown
```
