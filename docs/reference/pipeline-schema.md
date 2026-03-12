# Pipeline Schema Reference

Pipeline YAML files define multi-step AI workflows. Store pipelines in `.wave/pipelines/`.

## Minimal Pipeline

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: simple-task
steps:
  - id: execute
    persona: craftsman
    exec:
      type: prompt
      source: "Execute: {{ input }}"
```

</div>

Copy this to `.wave/pipelines/simple-task.yaml` and run with `wave run simple-task "your task"`.

---

## Complete Example

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: gh-pr-review
  description: "Automated code review pipeline"

input:
  source: cli

steps:
  - id: analyze
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      type: worktree
      branch: "{{ pipeline_id }}"
    exec:
      type: prompt
      source: "Analyze the codebase for: {{ input }}"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json
    handover:
      contract:
        type: json_schema
        schema_path: .wave/contracts/analysis.schema.json
        source: .wave/output/analysis.json

  - id: review
    persona: auditor
    dependencies: [analyze]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Review the code for security issues."
    output_artifacts:
      - name: findings
        path: .wave/output/findings.md
        type: markdown
    handover:
      contract:
        type: test_suite
        command: "go vet ./..."
```

</div>

---

## Top-Level Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `kind` | **yes** | - | Must be `WavePipeline` |
| `metadata.name` | **yes** | - | Pipeline identifier |
| `metadata.description` | no | `""` | Human-readable description |
| `input.source` | no | `cli` | Input source: `cli`, `file`, `stdin` |
| `input.path` | no | - | File path when `source: file` |
| `steps` | **yes** | - | Array of step definitions |

---

## Step Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `id` | **yes** | - | Unique step identifier |
| `persona` | **yes** | - | Persona from wave.yaml |
| `exec.type` | **yes** | - | `prompt`, `command`, or `slash_command` |
| `exec.source` | **yes** | - | Prompt template or shell command |
| `dependencies` | no | `[]` | Step IDs that must complete first |
| `memory.strategy` | no | `fresh` | Memory strategy (always `fresh`) |
| `memory.inject_artifacts` | no | `[]` | Artifacts from prior steps |
| `workspace.type` | no | - | `worktree` for git worktree workspaces |
| `workspace.branch` | no | auto | Branch name for worktree (supports templates) |
| `workspace.mount` | no | `[]` | Source mounts (alternative to worktree) |
| `output_artifacts` | no | `[]` | Files produced by this step |
| `outcomes` | no | `[]` | Structured results to extract from artifacts |
| `handover.contract` | no | - | Output validation |
| `handover.compaction` | no | - | Context relay settings |
| `strategy` | no | - | Matrix fan-out configuration |
| `validation` | no | `[]` | Pre-execution checks |

---

## Step Definition

### Basic Step

<div v-pre>

```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
```

</div>

### Step with Dependencies

```yaml
steps:
  - id: implement
    persona: craftsman
    dependencies: [analyze, plan]
    exec:
      type: prompt
      source: "Implement the feature"
```

### Step with Artifact Injection

```yaml
steps:
  - id: review
    persona: auditor
    dependencies: [implement]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: implement
          artifact: code
          as: changes
    exec:
      type: prompt
      source: "Review the changes"
```

---

## Exec Configuration

### Prompt Execution

<div v-pre>

```yaml
exec:
  type: prompt
  source: |
    Analyze the codebase for {{ input }}.
    Report file paths and architectural patterns.
```

</div>

### Command Execution

```yaml
exec:
  type: command
  source: "go test -v ./..."
```

### Slash Command Execution

```yaml
exec:
  type: slash_command
  command: review-pr
  args: "123"
```

Slash command execution invokes a Claude Code slash command (e.g., `/review-pr`) within the adapter session. The `command` field specifies the slash command name (without the leading `/`), and `args` provides the arguments.

| Field | Required | Description |
|-------|----------|-------------|
| `command` | **yes** | Slash command name (without `/` prefix) |
| `args` | no | Arguments to pass to the slash command |

### Template Variables

| Variable | Scope | Description |
|----------|-------|-------------|
| `{{ input }}` | All steps | Pipeline input from `--input` |
| `{{ task }}` | Matrix steps | Current matrix item |

---

## Output Artifacts

Declare files produced by a step:

```yaml
output_artifacts:
  - name: analysis
    path: .wave/output/analysis.json
    type: json
  - name: report
    path: .wave/output/report.md
    type: markdown
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | **yes** | Artifact identifier |
| `path` | **yes** | File path relative to workspace |
| `type` | no | `json`, `markdown`, `file`, `directory` |

---

## Outcomes

Outcomes extract structured results from step artifacts into the pipeline output summary. Use outcomes to surface PR URLs, issue links, deployment URLs, or other key results.

```yaml
outcomes:
  - type: pr
    extract_from: output/publish-result.json
    json_path: ".pr_url"
    label: "Pull Request"
  - type: url
    extract_from: output/publish-result.json
    json_path: ".deploy_urls[*]"
    json_path_label: ".label"
    label: "Deployment"
```

### Outcome Fields

| Field | Required | Description |
|-------|----------|-------------|
| `type` | **yes** | Outcome type: `pr`, `issue`, `url`, `deployment` |
| `extract_from` | **yes** | Artifact path relative to workspace (e.g., `output/publish-result.json`) |
| `json_path` | **yes** | Dot notation path to extract the value (e.g., `.pr_url`, `.comment_url`) |
| `json_path_label` | no | Label extraction path for array items (used with `[*]` in `json_path`) |
| `label` | no | Human-readable label for display in the output summary |

### Supported Outcome Types

| Type | Description |
|------|-------------|
| `pr` | Pull request URL |
| `issue` | Issue URL |
| `url` | Generic URL |
| `deployment` | Deployment URL |

---

## Artifact Injection

Import artifacts from prior steps:

```yaml
memory:
  strategy: fresh
  inject_artifacts:
    - step: analyze
      artifact: analysis
      as: context
    - step: plan
      artifact: tasks
      as: task_list
```

| Field | Required | Description |
|-------|----------|-------------|
| `step` | **yes** | Source step ID |
| `artifact` | **yes** | Artifact name from source step |
| `as` | **yes** | Name in current workspace |

Artifacts are copied to `.wave/artifacts/<as>/` in the step workspace.

---

## Workspace Configuration

### Worktree Workspace (Recommended)

<div v-pre>

```yaml
workspace:
  type: worktree
  branch: "{{ pipeline_id }}"
```

</div>

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | no | - | `worktree` for git worktree workspaces |
| `branch` | no | auto | Branch name for the worktree. Supports template variables. Steps sharing the same branch share the same worktree. |

When `type` is `worktree`, Wave creates a git worktree via `git worktree add` on the specified branch. If the branch doesn't exist, it's created from HEAD. Multiple steps with the same resolved branch reuse the same worktree directory.

### Mount Workspace

```yaml
workspace:
  mount:
    - source: ./src
      target: /code
      mode: readonly
    - source: ./test-fixtures
      target: /fixtures
      mode: readonly
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `mount[].source` | **yes** | - | Source directory |
| `mount[].target` | **yes** | - | Mount point in workspace |
| `mount[].mode` | no | `readonly` | `readonly` or `readwrite` |

---

## Contracts

Validate step output before proceeding.

### Test Suite Contract

```yaml
handover:
  contract:
    type: test_suite
    command: "npm test"
```

### JSON Schema Contract

```yaml
handover:
  contract:
    type: json_schema
    schema_path: .wave/contracts/analysis.schema.json
    source: .wave/output/analysis.json
    on_failure: retry
    max_retries: 2
```

### TypeScript Contract

```yaml
handover:
  contract:
    type: typescript_interface
    source: .wave/output/types.ts
    validate: true
```

### Contract Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | **yes** | - | `test_suite`, `json_schema`, `typescript_interface`, `markdown_spec`, `format`, `non_empty_file` |
| `command` | depends | - | Test command (for `test_suite`) |
| `schema` | depends | - | Schema path (for `json_schema`) |
| `source` | depends | - | File to validate |
| `must_pass` | no | `true` | Whether failure blocks progression |
| `on_failure` | no | `retry` | `retry` or `halt` |
| `max_retries` | no | `2` | Maximum retry attempts |

---

## Compaction

Configure context relay for long-running steps.

```yaml
handover:
  compaction:
    trigger: "token_limit_80%"
    persona: summarizer
```

| Field | Default | Description |
|-------|---------|-------------|
| `trigger` | `token_limit_80%` | When to trigger relay |
| `persona` | `summarizer` | Persona for checkpoints |

---

## Matrix Strategy

Fan-out parallel execution from a task list.

```yaml
steps:
  - id: plan
    persona: philosopher
    exec:
      type: prompt
      source: "Break down into tasks. Output: {\"tasks\": [...]}"
    output_artifacts:
      - name: tasks
        path: .wave/output/tasks.json

  - id: execute
    persona: craftsman
    dependencies: [plan]
    strategy:
      type: matrix
      items_source: plan/tasks.json
      item_key: task
      max_concurrency: 4
    exec:
      type: prompt
      source: "Execute: {{ task }}"
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | **yes** | - | Must be `matrix` |
| `items_source` | **yes** | - | Path to JSON task list |
| `item_key` | **yes** | - | JSON key for task items |
| `max_concurrency` | no | runtime default | Parallel workers |

---

## Pre-Execution Validation

Check conditions before step runs.

```yaml
validation:
  - type: file_exists
    target: src/models/user.go
    message: "User model required"
  - type: command
    target: "go build ./..."
    message: "Project must compile"
```

| Field | Required | Description |
|-------|----------|-------------|
| `type` | **yes** | `file_exists`, `command`, `schema` |
| `target` | **yes** | File path or command |
| `message` | no | Custom error message |

---

## DAG Rules

Pipeline steps form a directed acyclic graph (DAG).

**Enforced rules:**
- No circular dependencies
- All `dependencies` must reference valid step IDs
- All `persona` values must exist in wave.yaml
- Independent steps may run in parallel

```yaml
steps:
  - id: analyze        # Runs first
    persona: navigator

  - id: security       # Parallel with quality
    persona: auditor
    dependencies: [analyze]

  - id: quality        # Parallel with security
    persona: auditor
    dependencies: [analyze]

  - id: summary        # Waits for both
    persona: navigator
    dependencies: [security, quality]
```

---

## Step States

| State | Description |
|-------|-------------|
| `pending` | Waiting for dependencies |
| `running` | Currently executing |
| `completed` | Finished successfully |
| `retrying` | Failed, attempting retry |
| `failed` | Max retries exceeded |

---

## Next Steps

- [Pipelines](/concepts/pipelines) - Pipeline concepts
- [Outcomes](/concepts/outcomes) - Extracting structured results from pipelines
- [Contracts](/concepts/contracts) - Output validation
- [Contract Types](/reference/contract-types) - All contract options
- [Manifest Reference](/reference/manifest) - Project configuration
