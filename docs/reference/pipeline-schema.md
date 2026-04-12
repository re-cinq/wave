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
  name: ops-pr-review
  description: "Automated code review pipeline"
  category: ops

input:
  source: cli

hooks:
  - name: notify-start
    event: run_start
    type: command
    command: "echo 'Pipeline started'"

pipeline_outputs:
  review_url:
    step: publish
    artifact: result
    field: ".pr_url"

chat_context:
  artifact_summaries: [findings]
  suggested_questions:
    - "What security issues were found?"
  focus_areas: [security, performance]

steps:
  - id: analyze
    persona: navigator
    model: balanced
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
    thread: review-thread
    fidelity: compact
    contexts: [security, api]
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
    outcomes:
      - type: pr
        extract_from: output/findings.json
        json_path: ".pr_url"
        label: "Review PR"
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
| `metadata.category` | no | `""` | Pipeline category (e.g., `impl`, `audit`, `ops`) |
| `metadata.release` | no | `false` | Whether this pipeline is a released (stable) pipeline |
| `metadata.disabled` | no | `false` | Disable the pipeline without deleting it |
| `input.source` | no | `cli` | Input source: `cli`, `file`, `stdin` |
| `input.path` | no | - | File path when `source: file` |
| `input.schema` | no | - | Input schema for validation |
| `input.example` | no | - | Example input for documentation |
| `input.label_filter` | no | - | Label filter for issue-based inputs |
| `input.batch_size` | no | - | Batch size for multi-item inputs |
| `steps` | **yes** | - | Array of step definitions |
| `hooks` | no | `[]` | [Lifecycle hooks](#hooks) triggered on pipeline events |
| `pipeline_outputs` | no | `{}` | [Named output aliases](#pipeline-outputs) for composability |
| `chat_context` | no | - | [Post-pipeline chat](#chat-context) session configuration |
| `skills` | no | `[]` | Declarative [skill](#skills) references |
| `requires` | no | - | Pipeline [dependency declarations](#requires) |
| `max_step_visits` | no | `50` | [Graph-level limit](#max-step-visits) on total step visits |

---

## Step Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `id` | **yes** | - | Unique step identifier |
| `persona` | conditional | - | Persona from wave.yaml (required for prompt steps) |
| `adapter` | no | - | Step-level adapter override (e.g., `codex`, `gemini`) |
| `model` | no | - | Step-level model tier or name (e.g., `balanced`, `strongest`, `claude-haiku-4-5`) |
| `exec.type` | conditional | - | `prompt`, `command`, or `slash_command` |
| `exec.source` | conditional | - | Prompt template or shell command |
| `exec.source_path` | no | - | Path to a prompt file (alternative to inline `source`) |
| `dependencies` | no | `[]` | Step IDs that must complete first |
| `timeout_minutes` | no | - | Step-level timeout in minutes |
| `optional` | no | `false` | If true, step failure does not block the pipeline |
| `memory.strategy` | no | `fresh` | Memory strategy (always `fresh`) |
| `memory.inject_artifacts` | no | `[]` | Artifacts from prior steps |
| `workspace.type` | no | - | `worktree` for git worktree workspaces |
| `workspace.branch` | no | auto | Branch name for worktree (supports templates) |
| `workspace.mount` | no | `[]` | Source mounts (alternative to worktree) |
| `workspace.ref` | no | - | Reference another step's workspace (shared worktree) |
| `output_artifacts` | no | `[]` | Files produced by this step |
| `outcomes` | no | `[]` | Structured results to extract from artifacts |
| `handover.contract` | no | - | Output validation |
| `handover.contracts` | no | `[]` | Multiple output validations (takes precedence over singular `contract`) |
| `handover.compaction` | no | - | Context relay settings |
| `strategy` | no | - | Matrix fan-out configuration |
| `validation` | no | `[]` | Pre-execution checks |
| `retry` | no | - | [Retry and rework](#retry-and-rework) configuration |
| `rework_only` | no | `false` | Only runs via rework trigger, not normal DAG scheduling |
| `concurrency` | no | - | Max parallel agent instances for this step |
| `max_concurrent_agents` | no | - | Alias for `concurrency` |
| `thread` | no | - | [Thread group](#threads) ID for conversation continuity |
| `fidelity` | no | auto | [Context fidelity](#threads): `full`, `compact`, `summary`, `fresh` |
| `contexts` | no | `[]` | [Ontology context](#contexts) filter for bounded contexts |
| `type` | no | - | Step type: `conditional`, `command`, or empty (prompt) |
| `edges` | no | `[]` | [Graph edges](#edges) for conditional routing |
| `max_visits` | no | `10` | Max visits to this step in a [loop](#graph-loops) |
| `script` | no | - | Shell script for `command` type steps |
| `pipeline` | no | - | Child pipeline name for [sub-pipeline](#sub-pipelines) steps |
| `input` | no | - | Input template for child pipeline |
| `config` | no | - | [Sub-pipeline configuration](#sub-pipelines) |
| `iterate` | no | - | [Iterate](#iterate) over items (parallel fan-out) |
| `branch` | no | - | [Branch](#branch) for conditional pipeline selection |
| `gate` | no | - | [Gate](#gates) for approval or polling |
| `loop` | no | - | [Loop](#loop) for feedback loops |
| `aggregate` | no | - | [Aggregate](#aggregate) for output collection |

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

### Prompt from File

<div v-pre>

```yaml
exec:
  type: prompt
  source_path: .wave/prompts/analyze.md
```

</div>

Use `source_path` to keep long prompts in separate files. The file path is relative to the project root.

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
| <code v-pre>{{ input }}</code> | All steps | Pipeline input from `--input` |
| <code v-pre>{{ task }}</code> | Matrix steps | Current matrix item |
| <code v-pre>{{ pipeline_id }}</code> | All steps | Unique pipeline run ID |
| <code v-pre>{{ project.test_command }}</code> | All steps | Test command from wave.yaml |
| <code v-pre>{{ project.contract_test_command }}</code> | All steps | Contract test command from wave.yaml |
| <code v-pre>{{ forge.cli_tool }}</code> | All steps | Detected forge CLI (`gh`, `glab`) |
| <code v-pre>{{ forge.type }}</code> | All steps | Forge type (`github`, `gitlab`) |
| <code v-pre>{{ forge.pr_term }}</code> | All steps | PR terminology (`pull request`, `merge request`) |
| <code v-pre>{{ forge.pr_command }}</code> | All steps | PR command (`pr`, `mr`) |

---

## Model Routing

Override the model tier or specific model at the step level. See the [Model Routing Guide](/guide/model-routing) for full details.

```yaml
steps:
  - id: triage
    persona: navigator
    model: balanced
    exec:
      type: prompt
      source: "Classify the issue"

  - id: implement
    persona: craftsman
    model: strongest
    exec:
      type: prompt
      source: "Implement the solution"
```

Valid model tiers: `cheapest`, `balanced`, `strongest`. You can also specify exact model names (e.g., `claude-haiku-4-5`).

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
  - name: stdout-capture
    source: stdout
    type: json
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | **yes** | - | Artifact identifier |
| `path` | conditional | - | File path relative to workspace. Optional when `source: stdout`. |
| `type` | no | `file` | `json`, `markdown`, `file`, `binary`, `directory` |
| `source` | no | `file` | `file` (default) or `stdout` to capture from standard output |
| `required` | no | `false` | If true, missing artifact fails the step |

---

## Outcomes

Outcomes extract structured results from step artifacts into the pipeline output summary. Use outcomes to surface PR URLs, issue links, deployment URLs, or other key results. See the [Outcomes Guide](/guide/outcomes) for patterns.

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
  - type: file
    extract_from: output/report.md
    label: "Analysis Report"
```

### Outcome Fields

| Field | Required | Description |
|-------|----------|-------------|
| `type` | **yes** | Outcome type: `pr`, `issue`, `url`, `deployment`, `file`, `artifact` |
| `extract_from` | **yes** | Artifact path relative to workspace (e.g., `output/publish-result.json`) |
| `json_path` | conditional | Dot notation path to extract the value. Required for `pr`, `issue`, `url`, `deployment`. |
| `json_path_label` | no | Label extraction path for array items (used with `[*]` in `json_path`) |
| `label` | no | Human-readable label for display in the output summary |

### Supported Outcome Types

| Type | Description |
|------|-------------|
| `pr` | Pull request URL |
| `issue` | Issue URL |
| `url` | Generic URL |
| `deployment` | Deployment URL |
| `file` | File deliverable (uses `extract_from` as path) |
| `artifact` | Artifact deliverable (uses `extract_from` as path) |

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
      optional: true
    - pipeline: other-pipeline
      artifact: report
      as: upstream_report
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `step` | conditional | - | Source step ID (mutually exclusive with `pipeline`) |
| `pipeline` | conditional | - | Cross-pipeline artifact source name |
| `artifact` | **yes** | - | Artifact name from source step or pipeline |
| `as` | **yes** | - | Name in current workspace |
| `type` | no | - | Expected artifact type for validation |
| `schema_path` | no | - | JSON schema path for input validation |
| `optional` | no | `false` | If true, missing artifact does not fail the step |

Artifacts are copied to `.wave/artifacts/<as>/` in the step workspace.

---

## Workspace Configuration

### Worktree Workspace (Recommended)

<div v-pre>

```yaml
workspace:
  type: worktree
  branch: "{{ pipeline_id }}"
  base: main
```

</div>

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | no | - | `worktree` for git worktree workspaces |
| `branch` | no | auto | Branch name for the worktree. Supports template variables. Steps sharing the same branch share the same worktree. |
| `base` | no | HEAD | Start point for the worktree (e.g., `main`) |
| `ref` | no | - | Reference another step's workspace (shared worktree) |

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

### Basic Directory Workspace

```yaml
workspace:
  root: ./
```

Creates an empty workspace directory. The `root` field is the base path (relative to project root).

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

### Multiple Contracts

When a step requires multiple validations, use the plural `contracts` field. It takes precedence over the singular `contract`.

```yaml
handover:
  contracts:
    - type: json_schema
      schema_path: .wave/contracts/output.schema.json
      source: .wave/output/result.json
    - type: test_suite
      command: "go test ./..."
      dir: project_root
```

### LLM Judge Contract

```yaml
handover:
  contract:
    type: llm_judge
    model: claude-haiku-4-5
    criteria:
      - "Output is well-structured JSON"
      - "All required fields are present"
    threshold: 0.8
    source: .wave/output/result.json
```

### Agent Review Contract

```yaml
handover:
  contract:
    type: agent_review
    persona: auditor
    criteria_path: .wave/contracts/review-criteria.md
    context:
      - source: git_diff
      - source: artifact
        artifact: implementation
    token_budget: 50000
    timeout: "120s"
    on_failure: rework
    rework_step: fix-implementation
```

### Contract Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | **yes** | - | `test_suite`, `json_schema`, `typescript_interface`, `markdown_spec`, `format`, `non_empty_file`, `llm_judge`, `agent_review` |
| `command` | depends | - | Test command (for `test_suite`) |
| `schema_path` | depends | - | Schema path (for `json_schema`) |
| `source` | depends | - | File to validate |
| `dir` | no | workspace | Working directory: `project_root`, absolute path, or empty for workspace |
| `must_pass` | no | `true` | Whether failure blocks progression |
| `on_failure` | no | `retry` | `retry`, `halt`, `rework`, `warn` |
| `max_retries` | no | `2` | Maximum retry attempts |
| `model` | no | - | LLM model (for `llm_judge`) |
| `criteria` | no | - | Evaluation criteria list (for `llm_judge`) |
| `threshold` | no | `1.0` | Pass threshold 0.0-1.0 (for `llm_judge`) |
| `persona` | no | - | Reviewer persona (for `agent_review`) |
| `criteria_path` | no | - | Review criteria file (for `agent_review`) |
| `context` | no | - | Context sources for reviewer (for `agent_review`) |
| `token_budget` | no | unlimited | Max tokens for review agent |
| `timeout` | no | - | Duration string for review timeout (e.g., `60s`) |
| `rework_step` | no | - | Step to run on review failure with `on_failure: rework` |

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
| `item_id_key` | no | - | JSON key for unique item identifiers |
| `dependency_key` | no | - | JSON key for inter-item dependencies |
| `child_pipeline` | no | - | Pipeline name to invoke per item (instead of inline step) |
| `input_template` | no | - | Template for child pipeline input |
| `stacked` | no | `false` | If true, items share cumulative context |

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

## Threads

Steps sharing the same `thread` value participate in a conversation thread. Each step receives transcripts from prior steps in the same thread, enabling multi-step reasoning chains. See the [Threads Guide](/guide/threads) for patterns.

<div v-pre>

```yaml
steps:
  - id: research
    persona: navigator
    thread: analysis
    fidelity: full
    exec:
      type: prompt
      source: "Research: {{ input }}"

  - id: synthesize
    persona: navigator
    thread: analysis
    fidelity: compact
    dependencies: [research]
    exec:
      type: prompt
      source: "Synthesize findings"

  - id: implement
    persona: craftsman
    thread: impl
    dependencies: [synthesize]
    exec:
      type: prompt
      source: "Implement based on synthesis"
```

</div>

### Fidelity Levels

| Level | Description |
|-------|-------------|
| `full` | Complete conversation history (default when `thread` is set) |
| `compact` | Step ID + status + truncated content summary |
| `summary` | LLM-generated summary via compaction adapter |
| `fresh` | No prior context (default when no `thread`) |

### Thread Fields

| Field | Default | Description |
|-------|---------|-------------|
| `thread` | - | Thread group ID. Steps with the same thread share conversation context. |
| `fidelity` | `full` (if thread set), `fresh` (if no thread) | How much prior context to inject. |

---

## Contexts

Filter which ontology bounded contexts are injected into a step. When set, only the specified contexts are provided to the persona, reducing noise for focused steps.

```yaml
steps:
  - id: security-review
    persona: auditor
    contexts: [security, authentication]
    exec:
      type: prompt
      source: "Review for security vulnerabilities"

  - id: api-design
    persona: navigator
    contexts: [api, contracts]
    exec:
      type: prompt
      source: "Design the API surface"
```

When `contexts` is omitted, the step receives all available context.

---

## Edges

Edges define conditional routing between steps in graph-mode pipelines. Use edges to create loops, branching, and conditional execution. See the [Graph Loops Guide](/guide/graph-loops) for patterns.

```yaml
steps:
  - id: implement
    persona: craftsman
    thread: impl
    exec:
      type: prompt
      source: "Implement the feature"

  - id: test
    type: command
    dependencies: [implement]
    script: "go test ./..."

  - id: check
    type: conditional
    dependencies: [test]
    edges:
      - target: finalize
        condition: "outcome=success"
      - target: implement

  - id: finalize
    persona: navigator
    dependencies: [check]
```

### Edge Fields

| Field | Required | Description |
|-------|----------|-------------|
| `target` | **yes** | Target step ID to route to |
| `condition` | no | Condition for this edge (e.g., `outcome=success`). The first edge without a condition is the default fallback. |

### Step Types for Graph Mode

| Type | Purpose | Needs Persona? |
|------|---------|----------------|
| *(empty)* | LLM persona execution | Yes |
| `command` | Shell script execution | No |
| `conditional` | Route based on prior step outcome | No |

### Max Visits

Prevent infinite loops with visit limits:

```yaml
steps:
  - id: fix
    persona: craftsman
    max_visits: 5
    exec:
      type: prompt
      source: "Fix the failing tests"
```

| Field | Default | Description |
|-------|---------|-------------|
| `max_visits` | `10` | Max times a step can be visited in a graph loop |
| `max_step_visits` | `50` | Pipeline-level total visit limit across all steps |

---

## Gates

Gate steps pause pipeline execution for human decisions, CI events, or timers. See the [Human Gates Guide](/guide/human-gates) for patterns.

### Approval Gate

```yaml
steps:
  - id: approve
    gate:
      type: approval
      prompt: "Review the implementation plan"
      choices:
        - label: "Approve"
          key: "a"
          target: implement
        - label: "Revise"
          key: "r"
          target: plan
        - label: "Abort"
          key: "q"
          target: _fail
      freeform: true
      default: "a"
      timeout: "1h"
    dependencies: [plan]
```

### PR Merge Gate

```yaml
steps:
  - id: wait-merge
    gate:
      type: pr_merge
      pr_number: 123
      repo: "owner/repo"
      interval: "30s"
      timeout: "2h"
    dependencies: [create-pr]
```

### CI Pass Gate

```yaml
steps:
  - id: wait-ci
    gate:
      type: ci_pass
      branch: feature-branch
      interval: "30s"
      timeout: "30m"
    dependencies: [push]
```

### Timer Gate

```yaml
steps:
  - id: cooldown
    gate:
      type: timer
      timeout: "5m"
      message: "Cooling down before next phase"
    dependencies: [deploy]
```

### Gate Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | **yes** | - | `approval`, `pr_merge`, `ci_pass`, `timer` |
| `timeout` | no | - | Duration before auto-resolving (e.g., `30m`, `2h`) |
| `message` | no | - | Display message while waiting |
| `auto` | no | `false` | Auto-approve (for CI/testing) |
| `prompt` | no | - | Prompt text for approval gates |
| `choices` | no | - | Interactive choice options for approval gates |
| `freeform` | no | `false` | Allow freeform text input alongside choices |
| `default` | no | - | Default choice key (used on timeout or auto-approve) |
| `pr_number` | no | - | PR number for `pr_merge` gates |
| `repo` | no | auto-detect | `owner/repo` slug for poll gates |
| `branch` | no | auto-detect | Branch name for `ci_pass` gates |
| `interval` | no | `30s` | Poll interval for `pr_merge` and `ci_pass` gates |

### Gate Choice Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | **yes** | Human-readable label (e.g., "Approve") |
| `key` | **yes** | Keyboard shortcut key (e.g., "a") |
| `target` | no | Target step ID on selection, or `_fail` to abort the pipeline |

---

## Iterate

Iterate over a collection of items, executing a child pipeline for each. Use iterate for parallel fan-out over dynamic item lists. See the [Composition Guide](/guide/composition) for patterns.

```yaml
steps:
  - id: process-items
    iterate:
      over: "{{ steps.plan.artifacts.items }}"
      mode: parallel
      max_concurrent: 3
    pipeline: process-single-item
    input: "{{ item }}"
    config:
      inject: [context]
      extract: [result]
    dependencies: [plan]
```

### Iterate Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `over` | **yes** | - | Template expression resolving to a JSON array |
| `mode` | **yes** | - | `sequential` or `parallel` |
| `max_concurrent` | no | - | Max parallel workers (only for `parallel` mode) |

---

## Branch

Conditional pipeline selection based on a runtime value. Use branch to route execution to different pipelines based on step results. See the [Composition Guide](/guide/composition) for patterns.

```yaml
steps:
  - id: classify
    persona: navigator
    exec:
      type: prompt
      source: "Classify the issue as: bug, feature, or docs"
    output_artifacts:
      - name: classification
        path: .wave/output/classification.json
        type: json

  - id: route
    branch:
      on: "{{ steps.classify.artifacts.classification.type }}"
      cases:
        bug: impl-bugfix
        feature: impl-feature
        docs: doc-update
        _default: skip
    dependencies: [classify]
```

### Branch Fields

| Field | Required | Description |
|-------|----------|-------------|
| `on` | **yes** | Template expression to evaluate |
| `cases` | **yes** | Map of value to pipeline name. Use `skip` for no-op. |

---

## Loop

Feedback loops execute sub-steps repeatedly until a condition is met or the iteration limit is reached. See the [Composition Guide](/guide/composition) for patterns.

```yaml
steps:
  - id: refine
    loop:
      max_iterations: 5
      until: "{{ steps.validate.outcome == 'success' }}"
      steps:
        - id: improve
          persona: craftsman
          exec:
            type: prompt
            source: "Improve the implementation"
        - id: validate
          type: command
          script: "go test ./..."
          dependencies: [improve]
    dependencies: [initial-impl]
```

### Loop Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `max_iterations` | **yes** | - | Hard limit on iterations |
| `until` | no | - | Template condition for early exit |
| `steps` | no | - | Sub-steps to execute per iteration |

---

## Aggregate

Collect and merge outputs from prior steps (typically after iterate or matrix fan-out). See the [Composition Guide](/guide/composition) for patterns.

```yaml
steps:
  - id: collect
    aggregate:
      from: "{{ steps.process-items.results }}"
      into: .wave/output/combined.json
      strategy: merge_arrays
      key: findings          # extract .findings from each JSON object before merging
    dependencies: [process-items]
```

### Aggregate Fields

| Field | Required | Description |
|-------|----------|-------------|
| `from` | **yes** | Template expression for source data |
| `into` | **yes** | Output file path |
| `strategy` | **yes** | `merge_arrays`, `concat`, or `reduce` |
| `key` | no | JSON object key to extract before merging (`merge_arrays` only). When set, each element is expected to be an object and the value at this key (which must be an array) is extracted and merged. |

### Aggregation Strategies

| Strategy | Description |
|----------|-------------|
| `merge_arrays` | Merge JSON arrays from all items into one array. When `key` is set, extracts the named field from each JSON object before merging. |
| `concat` | Concatenate text outputs |
| `reduce` | Custom reduction (requires reduce template) |

---

## Sub-Pipelines

Execute a child pipeline as a step. Use sub-pipelines for reusable workflow components. See the [Composition Guide](/guide/composition) for patterns.

```yaml
steps:
  - id: run-tests
    pipeline: test-suite
    input: "{{ input }}"
    config:
      inject: [implementation]
      extract: [test-results]
      timeout: "3600s"
      max_cycles: 10
      stop_condition: "{{ child.status == 'all_pass' }}"
    dependencies: [implement]
```

### Sub-Pipeline Config Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `pipeline` | **yes** | - | Child pipeline name |
| `input` | no | - | Input template for the child pipeline |
| `config.inject` | no | `[]` | Parent artifact names to inject into the child |
| `config.extract` | no | `[]` | Child artifact names to extract back to parent |
| `config.timeout` | no | - | Hard timeout for child execution (e.g., `3600s`) |
| `config.max_cycles` | no | - | Max iterations for child loop steps |
| `config.stop_condition` | no | - | Template expression for early termination |

---

## Hooks

Lifecycle hooks trigger actions on pipeline events. Hooks run shell commands, HTTP requests, LLM evaluations, or scripts at defined points in the pipeline lifecycle.

```yaml
hooks:
  - name: notify-start
    event: run_start
    type: command
    command: "echo 'Pipeline {{ pipeline_id }} started'"

  - name: slack-notification
    event: run_completed
    type: http
    url: "https://hooks.slack.com/services/T.../B.../..."
    timeout: "10s"
    fail_open: true

  - name: quality-check
    event: step_completed
    type: llm_judge
    model: claude-haiku-4-5
    prompt: "Evaluate the quality of the step output"
    matcher: "step_id=implement"
    blocking: true

  - name: cleanup
    event: run_failed
    type: script
    script: |
      rm -rf /tmp/wave-cache
      echo "Cleaned up"
```

### Hook Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | **yes** | - | Hook identifier |
| `event` | **yes** | - | Lifecycle event to trigger on |
| `type` | **yes** | - | `command`, `http`, `llm_judge`, `script` |
| `command` | conditional | - | Shell command (for `command` type) |
| `url` | conditional | - | HTTP endpoint (for `http` type) |
| `model` | conditional | - | LLM model (for `llm_judge` type) |
| `prompt` | conditional | - | Evaluation prompt (for `llm_judge` type) |
| `script` | conditional | - | Shell script (for `script` type) |
| `matcher` | no | - | Filter which steps trigger this hook (e.g., `step_id=implement`) |
| `blocking` | no | event-dependent | Whether the hook blocks pipeline execution on failure |
| `fail_open` | no | type-dependent | If true, hook errors do not block the pipeline |
| `timeout` | no | type-dependent | Duration string (defaults: command 30s, http 10s, llm_judge 60s, script 30s) |

### Lifecycle Events

| Event | Scope | Description |
|-------|-------|-------------|
| `run_start` | Pipeline | Fires when the pipeline run begins |
| `run_completed` | Pipeline | Fires when the pipeline completes successfully |
| `run_failed` | Pipeline | Fires when the pipeline fails |
| `step_start` | Step | Fires before a step executes |
| `step_completed` | Step | Fires after a step completes successfully |
| `step_failed` | Step | Fires when a step fails |
| `step_retrying` | Step | Fires when a step is about to retry |
| `contract_validated` | Step | Fires after a contract passes validation |
| `artifact_created` | Step | Fires when an output artifact is written |
| `workspace_created` | Step | Fires when a workspace is provisioned |

---

## Pipeline Outputs

Named output aliases expose pipeline results for composition with other pipelines. Parent pipelines can reference these outputs when using sub-pipelines.

```yaml
pipeline_outputs:
  review_url:
    step: publish
    artifact: result
    field: ".pr_url"
  summary:
    step: analyze
    artifact: report
```

### Pipeline Output Fields

| Field | Required | Description |
|-------|----------|-------------|
| `step` | **yes** | Source step ID |
| `artifact` | **yes** | Artifact name from the source step |
| `field` | no | Optional JSON field extraction (dot notation) |

---

## Chat Context

Configure what context to inject into post-pipeline interactive chat sessions. When a pipeline completes, Wave can start a chat session pre-loaded with pipeline results.

```yaml
chat_context:
  artifact_summaries:
    - analysis
    - findings
  suggested_questions:
    - "What were the main security findings?"
    - "Which files need the most attention?"
  focus_areas:
    - security
    - performance
    - architecture
  max_context_tokens: 12000
```

### Chat Context Fields

| Field | Default | Description |
|-------|---------|-------------|
| `artifact_summaries` | `[]` | Artifact names to summarize in the chat context |
| `suggested_questions` | `[]` | Opening questions displayed to the user |
| `focus_areas` | `[]` | Areas to highlight in the chat session |
| `max_context_tokens` | `8000` | Token budget for injected context |

---

## Skills

Declarative skill references ensure required skills are available before the pipeline runs. Skills provide domain-specific capabilities to personas.

```yaml
skills:
  - golang
  - docker

requires:
  skills:
    golang:
      install: "go install github.com/example/skill@latest"
      check: "which go-skill"
    docker:
      check: "docker version"
  tools:
    - gh
    - jq
```

### Pipeline-Level `skills`

A list of skill names that the pipeline uses. Wave validates these are available at runtime.

### Requires Block

| Field | Description |
|-------|-------------|
| `requires.skills` | Map of skill name to config (install, init, check commands) |
| `requires.tools` | List of CLI tool names that must be on PATH |

### Skill Config Fields

| Field | Description |
|-------|-------------|
| `install` | Command to install the skill |
| `init` | Command to initialize the skill after install |
| `check` | Command to verify the skill is available |
| `commands_glob` | Glob pattern for skill command files |

See the [Skill Authoring Guide](/guide/skills) for creating custom skills.

---

## Max Step Visits

Pipeline-level limit on total step visits across all steps in graph-mode pipelines. Prevents runaway loops.

```yaml
kind: WavePipeline
metadata:
  name: iterative-fix
max_step_visits: 30

steps:
  - id: fix
    persona: craftsman
    max_visits: 10
    # ...
```

| Field | Level | Default | Description |
|-------|-------|---------|-------------|
| `max_step_visits` | Pipeline | `50` | Total visits across all steps in the pipeline |
| `max_visits` | Step | `10` | Max visits for a single step |

When either limit is reached, the pipeline halts with an error indicating the loop limit was exceeded.

---

## DAG Rules

Pipeline steps form a directed acyclic graph (DAG). In graph-mode pipelines (using edges), cycles are permitted.

**Enforced rules:**
- No circular dependencies in DAG mode (cycles allowed only via edges in graph mode)
- All `dependencies` must reference valid step IDs
- All `persona` values must exist in wave.yaml (for prompt steps)
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

## Retry and Rework

Control what happens when a step fails after exhausting its retry attempts.

### Retry Configuration

```yaml
steps:
  - id: flaky-step
    persona: craftsman
    exec:
      type: prompt
      source: "Implement feature"
    retry:
      max_attempts: 3
      backoff: exponential
      base_delay: "2s"
      max_delay: "30s"
      adapt_prompt: true
      on_failure: fail
```

### Retry Policy Presets

Use named policies instead of configuring individual fields:

```yaml
retry:
  policy: standard
```

| Policy | Attempts | Backoff | Base Delay | Max Delay |
|--------|----------|---------|------------|-----------|
| `none` | 1 | fixed | 0s | 0s |
| `standard` | 3 | exponential | 1s | 30s |
| `aggressive` | 5 | exponential | 200ms | 30s |
| `patient` | 3 | exponential | 5s | 90s |

Explicit fields override policy defaults.

### Retry Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `policy` | no | - | Named preset: `none`, `standard`, `aggressive`, `patient` |
| `max_attempts` | no | `1` | Total number of attempts (1 = no retry) |
| `backoff` | no | `linear` | `fixed`, `linear`, or `exponential` |
| `base_delay` | no | `1s` | Base delay between retries (e.g., `2s`, `500ms`) |
| `max_delay` | no | `30s` | Maximum delay cap |
| `adapt_prompt` | no | `false` | Inject prior failure context into retry prompt |
| `on_failure` | no | `fail` | Action when all attempts are exhausted: `fail`, `skip`, `continue`, `rework` |
| `rework_step` | conditional | - | Step ID to execute when `on_failure: rework`. Required when `on_failure` is `rework`. |

### On-Failure Actions

| Action | Description |
|--------|-------------|
| `fail` | Halt the pipeline (default) |
| `skip` | Mark step as skipped, continue pipeline |
| `continue` | Mark step as failed, continue pipeline |
| `rework` | Execute an alternative step (`rework_step`) as a fallback |

### Rework Branching

When `on_failure: rework` is set, the executor redirects to an alternative step after all retry attempts are exhausted:

```yaml
steps:
  - id: complex-impl
    persona: craftsman
    exec:
      type: prompt
      source: "Implement the complex feature"
    retry:
      max_attempts: 2
      on_failure: rework
      rework_step: simple-impl

  - id: simple-impl
    persona: craftsman
    rework_only: true
    exec:
      type: prompt
      source: "Implement a simpler fallback"
```

**Rework behavior:**
1. The failed step is marked as `failed`
2. Failure context (error, duration, partial artifacts) is injected into the rework step's prompt
3. The rework step executes with the failure context
4. On success, the rework step's artifacts replace the failed step's artifacts for downstream steps
5. If the rework step itself fails, its own `on_failure` policy applies

**DAG validation rules for rework targets:**
- The rework target must be an existing step in the pipeline
- The rework target cannot be an upstream dependency of the failing step
- The failing step cannot be a dependency of the rework target
- A step cannot rework to itself

---

## Step States

| State | Description |
|-------|-------------|
| `pending` | Waiting for dependencies |
| `running` | Currently executing |
| `completed` | Finished successfully |
| `retrying` | Failed, attempting retry |
| `reworking` | Rework step executing after failure |
| `failed` | Max retries exceeded |
| `skipped` | Skipped (dependency failed or on_failure: skip) |

---

## Next Steps

- [Pipelines](/concepts/pipelines) - Pipeline concepts
- [Graph Loops](/guide/graph-loops) - Conditional routing and feedback loops
- [Human Gates](/guide/human-gates) - Approval and polling gates
- [Threads](/guide/threads) - Thread continuity and fidelity levels
- [Composition](/guide/composition) - Sub-pipelines, iterate, branch, loop, aggregate
- [Pipeline Outputs](/guide/pipeline-outputs) - Named output aliases for composability
- [Outcomes](/guide/outcomes) - Deliverable extraction from pipelines
- [Chat Context](/guide/chat-context) - Post-pipeline chat experience
- [Model Routing](/guide/model-routing) - Model tier selection
- [Skills](/guide/skills) - Skill authoring and configuration
- [Outcomes](/concepts/outcomes) - Extracting structured results from pipelines
- [Contracts](/concepts/contracts) - Output validation
- [Contract Types](/reference/contract-types) - All contract options
- [Manifest Reference](/reference/manifest) - Project configuration
