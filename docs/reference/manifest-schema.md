# Manifest Schema Reference

Complete field reference for `wave.yaml` — the single source of truth for all Wave orchestration behavior.

## Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | `string` | **yes** | Schema version. Currently `"v1"`. |
| `kind` | `string` | **yes** | Must be `"WaveManifest"`. |
| `metadata` | [`Metadata`](#metadata) | **yes** | Project metadata. |
| `adapters` | `map[string]`[`Adapter`](#adapter) | **yes** | Named adapter configurations. |
| `personas` | `map[string]`[`Persona`](#persona) | **yes** | Named persona configurations. |
| `runtime` | [`Runtime`](#runtime) | **yes** | Global runtime settings. |
| `project` | [`Project`](#project) | no | Project metadata for language, test commands, and source globs. |
| `skills` | `map[string]`[`SkillConfig`](#skillconfig) | no | Named skill configurations with install, check, and provisioning settings. |

### Minimal Example

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
```

---

## Metadata

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | `string` | **yes** | — | Project name. Used in event output and workspace paths. |
| `description` | `string` | no | `""` | Human-readable project description. |
| `repo` | `string` | no | `""` | Repository URL. Used in audit logs and generated documentation. |

```yaml
metadata:
  name: acme-backend
  description: "Acme Corp backend API service"
  repo: https://github.com/acme/backend
```

---


## Project

Optional project-level settings used for build, test, and source discovery.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `language` | `string` | no | `""` | Project language (e.g., "go", "typescript"). Used for adapter heuristics. |
| `test_command` | `string` | no | `""` | Command to run tests (e.g., "go test ./..."). Used by contract validation. |
| `lint_command` | `string` | no | `""` | Command to run linting. |
| `build_command` | `string` | no | `""` | Command to build the project. |
| `source_glob` | `string` | no | `""` | Glob pattern matching source files (e.g., "**/*.go"). |

```yaml
project:
  language: go
  test_command: "go test ./..."
  lint_command: "golangci-lint run"
  build_command: "go build ./..."
  source_glob: "**/*.go"
```

---

## Adapter

Wraps a specific LLM CLI for subprocess invocation. Each adapter defines how Wave communicates with one LLM tool.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `binary` | `string` | **yes** | — | CLI binary name. Must be resolvable on `$PATH`. |
| `mode` | `string` | **yes** | — | Execution mode. Currently only `"headless"` (always subprocess, never interactive). |
| `output_format` | `string` | no | `"json"` | Expected output format from the CLI. |
| `project_files` | `[]string` | no | `[]` | Files to project (copy) into every workspace using this adapter. Supports glob patterns. |
| `default_permissions` | [`Permissions`](#permissions) | no | allow all | Default tool permissions applied to all personas using this adapter. Persona-level permissions override these. |
| `hooks_template` | `string` | no | `""` | Directory containing hook script templates. Scripts are copied into workspaces. |

### Adapter Example

```yaml
adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
    project_files:
      - CLAUDE.md
      - .claude/settings.json
    default_permissions:
      allowed_tools: ["Read", "Write", "Bash"]
      deny: []
    hooks_template: .wave/hooks/claude/

  opencode:
    binary: opencode
    mode: headless
    output_format: json
    default_permissions:
      allowed_tools: ["Read", "Write"]
      deny: ["Bash(rm *)"]
```

### Binary Resolution

The `binary` field is resolved against `$PATH` at validation time. If the binary is not found, `wave validate` emits a **warning** (not an error) — the binary may be available at runtime but not at validation time (e.g., in CI).

---

## Persona

Agent configuration binding an adapter to a specific role. Personas enforce separation of concerns — each persona has distinct permissions, behavior, and purpose.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `adapter` | `string` | **yes** | — | References a key in `adapters`. |
| `description` | `string` | no | `""` | Human-readable purpose description. |
| `system_prompt_file` | `string` | **yes** | — | Path to markdown file containing the persona's system prompt. Relative to project root. |
| `temperature` | `float` | no | adapter default | LLM temperature setting. Range: `0.0` to `1.0`. Lower values produce more deterministic output. |
| `permissions` | [`Permissions`](#permissions) | no | inherit from adapter | Tool permission overrides. Merged with adapter defaults; persona-level `deny` always takes precedence. |
| `model` | `string` | no | adapter default | LLM model override for this persona (e.g., "opus", "sonnet"). |
| `hooks` | [`HookConfig`](#hookconfig) | no | `{}` | Pre/post tool use hook definitions. |

### Built-in Persona Archetypes

```yaml
personas:
  # Read-only codebase analysis
  navigator:
    adapter: claude
    description: "Codebase exploration and analysis"
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Glob", "Grep", "Bash(git log*)", "Bash(git status*)"]
      deny: ["Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)"]

  # Design and specification
  philosopher:
    adapter: claude
    description: "Architecture design and specification"
    system_prompt_file: .wave/personas/philosopher.md
    temperature: 0.3
    permissions:
      allowed_tools: ["Read", "Write(.wave/specs/*)"]
      deny: ["Bash(*)"]

  # Implementation with full write access
  craftsman:
    adapter: claude
    description: "Code implementation and testing"
    system_prompt_file: .wave/personas/craftsman.md
    temperature: 0.7
    permissions:
      allowed_tools: ["Read", "Write", "Edit", "Bash"]
      deny: ["Bash(rm -rf /*)"]

  # Security and quality review
  auditor:
    adapter: claude
    description: "Security review and quality assurance"
    system_prompt_file: .wave/personas/auditor.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Grep", "Bash(npm audit*)", "Bash(go vet*)"]
      deny: ["Write(*)", "Edit(*)"]

  # Context relay summarizer
  summarizer:
    adapter: claude
    description: "Context compaction for relay handoffs"
    system_prompt_file: .wave/personas/summarizer.md
    temperature: 0.0
    permissions:
      allowed_tools: ["Read"]
      deny: ["Write(*)", "Bash(*)"]
```

---

## Permissions

Tool access control applied to an adapter or persona.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `allowed_tools` | `[]string` | no | `["*"]` (all) | Glob patterns for allowed tool calls. |
| `deny` | `[]string` | no | `[]` | Glob patterns for denied tool calls. **Always takes precedence** over `allowed_tools`. |

### Evaluation Order

1. Check `deny` patterns first. If **any** deny pattern matches → **blocked**.
2. Check `allowed_tools`. If **any** allowed pattern matches → **permitted**.
3. If no pattern matches → **blocked** (implicit deny).

### Pattern Syntax

Patterns use glob matching against tool call signatures:

| Pattern | Matches |
|---------|---------|
| `Read` | All Read tool calls |
| `Write(*)` | All Write tool calls (any path) |
| `Write(src/*.ts)` | Write calls to TypeScript files in `src/` |
| `Bash(git *)` | Bash calls starting with `git` |
| `Bash(npm test*)` | Bash calls starting with `npm test` |
| `*` | All tool calls |

### Permission Inheritance

```
Adapter default_permissions
    ↓ (base)
Persona permissions
    ↓ (override)
Effective permissions
```

Persona `deny` patterns are **additive** — they combine with adapter-level denies. Persona `allowed_tools` **replace** adapter-level allowed tools when specified.

---

## HookConfig

Pre/post tool execution hooks. Hooks execute shell commands triggered by tool call patterns.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `PreToolUse` | [`[]HookRule`](#hookrule) | no | `[]` | Hooks that fire **before** a tool call executes. |
| `PostToolUse` | [`[]HookRule`](#hookrule) | no | `[]` | Hooks that fire **after** a tool call completes. |

### HookRule

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `matcher` | `string` | **yes** | Glob pattern matching tool call signatures. Same syntax as [Permissions](#pattern-syntax). |
| `command` | `string` | **yes** | Shell command to execute. Must be a script on disk or inline command. |

### Hook Behavior

- **PreToolUse**: If the command exits **non-zero**, the tool call is **blocked**. The agent receives a permission denial message.
- **PostToolUse**: Informational only. Exit code is logged but does **not** block execution.

### Hook Example

```yaml
personas:
  craftsman:
    adapter: claude
    system_prompt_file: .wave/personas/craftsman.md
    hooks:
      PreToolUse:
        # Block destructive filesystem operations
        - matcher: "Bash(rm -rf *)"
          command: ".wave/hooks/block-destructive.sh"
        # Require linting before any commit
        - matcher: "Bash(git commit*)"
          command: ".wave/hooks/pre-commit-lint.sh"
      PostToolUse:
        # Run tests after file writes
        - matcher: "Write(src/**)"
          command: "npm test --silent"
        # Log all bash invocations
        - matcher: "Bash(*)"
          command: ".wave/hooks/log-bash.sh"
```

---

## Runtime

Global runtime settings governing execution behavior.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `workspace_root` | `string` | no | `".wave/workspaces"` | Root directory for ephemeral workspaces. Each pipeline run creates subdirectories here. |
| `max_concurrent_workers` | `int` | no | `5` | Maximum parallel matrix strategy workers. Range: `1`–`10`. |
| `max_step_concurrency` | `int` | no | `10` | Global cap on per-step concurrency. Limits how many parallel agents a single step can spawn via `concurrency`. Range: `1`–`10`. |
| `default_timeout_minutes` | `int` | no | `5` | Default per-step timeout. Steps exceeding this are killed (entire process group). |
| `relay` | [`RelayConfig`](#relayconfig) | no | see defaults | Context relay/compaction settings. |
| `audit` | [`AuditConfig`](#auditconfig) | no | see defaults | Audit logging settings. |
| `meta_pipeline` | [`MetaPipelineConfig`](#metapipelineconfig) | no | see defaults | Meta-pipeline recursion and resource limits. |
| `routing` | [`RoutingConfig`](#routingconfig) | no | see defaults | Pipeline routing rules for matching inputs to pipelines. |
| `sandbox` | [`RuntimeSandbox`](#runtimesandbox) | no | see defaults | Sandbox settings including env passthrough and domain allowlisting. |
| `artifacts` | [`RuntimeArtifactsConfig`](#runtimeartifactsconfig) | no | see defaults | Global artifact handling configuration. |
| `pipeline_id_hash_length` | `int` | no | `4` | Length of hash suffix appended to pipeline workspace IDs. |

### RelayConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `token_threshold_percent` | `int` | no | `80` | Context utilization percentage that triggers relay. Range: `50`–`95`. |
| `strategy` | `string` | no | `"summarize_to_checkpoint"` | Compaction strategy. Currently only `"summarize_to_checkpoint"`. |

### AuditConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `log_dir` | `string` | no | `".wave/traces/"` | Directory for audit trail files. Created automatically. |
| `log_all_tool_calls` | `bool` | no | `false` | Log every tool invocation with arguments and results. |
| `log_all_file_operations` | `bool` | no | `false` | Log every file read, write, and delete with paths. |

::: warning Credential Scrubbing
Audit logs **never** capture environment variable values or credential content. Known credential patterns (`*_KEY`, `*_TOKEN`, `*_SECRET`, `*_PASSWORD`) are automatically redacted.
:::

### MetaPipelineConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `max_depth` | `int` | no | `2` | Maximum recursion depth for meta-pipeline generation. A meta-pipeline spawning another meta-pipeline counts as depth+1. |
| `max_total_steps` | `int` | no | `20` | Maximum total steps across all recursion levels. |
| `max_total_tokens` | `int` | no | `500000` | Maximum total token consumption across all meta-pipeline levels. |
| `timeout_minutes` | `int` | no | `60` | Hard timeout for entire meta-pipeline tree. |

### RoutingConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `default` | `string` | no | `""` | Pipeline to use when no routing rules match. |
| `rules` | `[]RoutingRule` | no | `[]` | Routing rules evaluated in priority order. |

#### RoutingRule

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `pattern` | `string` | no | `""` | Glob pattern for matching input strings. Supports `*`, `?`, `[abc]`, `[a-z]`. |
| `pipeline` | `string` | **yes** | — | Pipeline name to route to when this rule matches. |
| `priority` | `int` | no | `0` | Evaluation order. Higher priority rules are evaluated first. |

### RuntimeSandbox

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | `bool` | no | `false` | Enable sandbox mode for adapter subprocesses. |
| `default_allowed_domains` | `[]string` | no | `[]` | Network domain allowlist applied to all personas. |
| `env_passthrough` | `[]string` | no | `[]` | Environment variables passed through to adapter subprocesses. |

### RuntimeArtifactsConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `max_stdout_size` | `int` | no | `10485760` | Maximum bytes to capture from stdout (default: 10MB). |
| `default_artifact_dir` | `string` | no | `".wave/artifacts"` | Base directory for artifacts. |

### Full Runtime Example

```yaml
runtime:
  workspace_root: .wave/workspaces
  max_concurrent_workers: 5
  default_timeout_minutes: 5
  relay:
    token_threshold_percent: 80
    strategy: summarize_to_checkpoint
  audit:
    log_dir: .wave/traces/
    log_all_tool_calls: true
    log_all_file_operations: true
  meta_pipeline:
    max_depth: 2
    max_total_steps: 20
    max_total_tokens: 500000
    timeout_minutes: 60
```

---

## SkillConfig

Declares an external skill with install, check, and provisioning commands. Skills are referenced by pipelines via the [`requires`](#pipeline-requires) block and validated at preflight time before execution begins.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `install` | `string` | no | `""` | Shell command to install the skill. Executed via `sh -c` when preflight detects the skill is missing. |
| `init` | `string` | no | `""` | Shell command to initialize the skill after installation (e.g., create config files). |
| `check` | `string` | **yes** | — | Shell command to verify the skill is installed. Exit code 0 = installed. |
| `commands_glob` | `string` | no | `.claude/commands/<name>.*.md` | Glob pattern for skill command files provisioned into workspaces. |

### SkillConfig Example

```yaml
skills:
  speckit:
    install: "npm install -g @anthropic/speckit"
    init: "speckit init --non-interactive"
    check: "speckit --version"
    commands_glob: ".claude/commands/speckit.*.md"

  golangci-lint:
    install: "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
    check: "golangci-lint --version"
```

### Preflight Flow

When a pipeline declares `requires.skills`, the executor runs preflight validation before any step executes:

1. For each required skill, run its `check` command.
2. If the check fails and `install` is configured, run the install command.
3. If `init` is configured, run it after successful install.
4. Re-run the `check` command to verify installation succeeded.
5. If any skill remains unavailable, the pipeline fails with a preflight error.

---

## Pipeline Requires

Pipelines can declare tool and skill dependencies via a `requires` block. These are validated at preflight time before any step executes.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `requires.skills` | `[]string` | no | Skill names that must be installed. Each name must match a key in the manifest `skills` map. |
| `requires.tools` | `[]string` | no | CLI tool names that must be available on `$PATH` (checked via `exec.LookPath`). |

### Requires Example

```yaml
kind: WavePipeline
metadata:
  name: speckit-flow
requires:
  skills: [speckit]
  tools: [git, go]
steps:
  - id: specify
    # ...
```

---

## Workspace Types

Steps can use different workspace isolation strategies via the `workspace.type` field.

| Type | Description |
|------|-------------|
| *(empty)* | Default directory-based workspace under `runtime.workspace_root`. Supports `mount` for bind-mounting source directories. |
| `worktree` | Creates a git worktree for full repository isolation. Each step gets its own branch and working copy. |

### Worktree Workspace Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `workspace.type` | `string` | no | `""` | Set to `"worktree"` for git worktree isolation. |
| `workspace.branch` | `string` | no | `wave/<pipelineID>/<stepID>` | Branch name for the worktree. Supports `{{ }}` placeholder resolution. |

### Worktree Example

```yaml
steps:
  - id: implement
    persona: craftsman
    workspace:
      type: worktree
      branch: "feat/{{ pipeline_name }}"
    exec:
      type: prompt
      source: "Implement the feature on this isolated branch."
```

---

## Agent Concurrency

Steps can configure the maximum number of concurrent sub-agents the persona may spawn via `max_concurrent_agents`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `max_concurrent_agents` | `int` | no | `0` | Maximum concurrent sub-agents for this step. Values > 1 inject a concurrency hint into the persona's system prompt. Capped at 10. |

When `max_concurrent_agents` is set to a value greater than 1, the generated CLAUDE.md includes:

```
You may spawn up to N concurrent sub-agents or workers for this step.
```

Values of 0 or 1 produce no hint (default single-agent behavior).

### Agent Concurrency Example

```yaml
steps:
  - id: implement
    persona: craftsman
    max_concurrent_agents: 6
    exec:
      type: prompt
      source: "Implement the feature using parallel sub-agents."
```

---

## Step Concurrency

Steps can spawn multiple parallel agent instances via `concurrency`. Unlike `max_concurrent_agents` (which hints to the agent about internal sub-agent spawning), `concurrency` causes the **executor** to fork N parallel adapter processes for the same step.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `concurrency` | `int` | no | `1` | Number of parallel agent instances the executor spawns. Each agent gets an isolated workspace with the same prompt and input artifacts. Results are merged into an indexed artifact set. Capped at `runtime.max_step_concurrency` (default 10). |

### Key Behaviors

- **Workspace isolation**: Each agent gets its own workspace at `<step_workspace>_agent_<N>/`
- **Same prompt**: All agents receive the same prompt and input artifacts. Work partitioning is the prompt author's responsibility
- **Result aggregation**: JSON artifacts are merged into an array; text artifacts are concatenated with agent index headers
- **Fail-fast**: If any agent fails, the step fails immediately (errgroup cancellation)
- **Mutual exclusion**: `concurrency` is mutually exclusive with `strategy` and `iterate`

### Step Concurrency vs Agent Concurrency

| Feature | `concurrency` | `max_concurrent_agents` |
|---------|--------------|-------------------------|
| Level | Executor (Wave) | Agent (internal) |
| What it does | Spawns N adapter processes | Tells agent it may use N sub-agents |
| Isolation | Each agent gets own workspace | All sub-agents share one workspace |
| Can coexist? | Yes — orthogonal | Yes — orthogonal |

A step with `concurrency: 3, max_concurrent_agents: 5` spawns 3 parallel adapter processes, each of which is allowed to use 5 sub-agents internally.

### Step Concurrency Example

```yaml
steps:
  - id: process-items
    persona: worker
    concurrency: 3
    exec:
      type: prompt
      source: "Process the assigned work items in parallel."
```

### With Global Cap

```yaml
runtime:
  max_step_concurrency: 5  # No step can spawn more than 5 agents

steps:
  - id: analyze
    persona: navigator
    concurrency: 8  # Capped at 5 by runtime setting
    exec:
      type: prompt
      source: "Analyze the codebase."
```

---

## Slash Command Exec Type

Steps can invoke skill slash commands instead of inline prompts via `exec.type: slash_command`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `exec.type` | `string` | **yes** | Set to `"slash_command"` to invoke a skill command. |
| `exec.command` | `string` | **yes** | Slash command name (e.g., `speckit.specify`). Automatically prefixed with `/` if missing. |
| `exec.args` | `string` | no | Arguments passed to the slash command. Supports `{{ input }}` placeholder. |

### Slash Command Example

```yaml
steps:
  - id: specify
    persona: implementer
    exec:
      type: slash_command
      command: speckit.specify
      args: "{{ input }}"
```

---

## Validation Rules

`wave validate` checks the following rules:

| # | Rule | Severity | Description |
|---|------|----------|-------------|
| 1 | Adapter reference | **error** | Every persona must reference a defined adapter key. |
| 2 | System prompt exists | **error** | Every persona's `system_prompt_file` must exist on disk. |
| 3 | Hook scripts exist | **error** | Every hook `command` script must exist on disk (if it's a file path, not inline). |
| 4 | Binary on PATH | **warning** | Adapter `binary` should be resolvable on `$PATH`. |
| 5 | No circular refs | **error** | No circular persona or pipeline references. |
| 6 | Required fields | **error** | All required fields must be present and non-empty. |
| 7 | Type correctness | **error** | Fields must match expected types (string, int, float, array, map). |
| 8 | Value ranges | **error** | Numeric fields must be within valid ranges (e.g., temperature 0.0–1.0). |

---

## Complete Example

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: acme-backend
  description: "Acme Corp backend API — Go microservices"
  repo: https://github.com/acme/backend

adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
    project_files:
      - CLAUDE.md
      - .claude/settings.json
    default_permissions:
      allowed_tools: ["Read", "Write", "Edit", "Bash"]
      deny: []
    hooks_template: .wave/hooks/claude/

personas:
  navigator:
    adapter: claude
    description: "Read-only codebase exploration"
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Glob", "Grep", "Bash(git *)"]
      deny: ["Write(*)", "Edit(*)", "Bash(rm *)"]

  philosopher:
    adapter: claude
    description: "Architecture and specification design"
    system_prompt_file: .wave/personas/philosopher.md
    temperature: 0.3
    permissions:
      allowed_tools: ["Read", "Write(.wave/specs/*)"]
      deny: ["Bash(*)"]

  craftsman:
    adapter: claude
    description: "Implementation and testing"
    system_prompt_file: .wave/personas/craftsman.md
    temperature: 0.7
    permissions:
      allowed_tools: ["Read", "Write", "Edit", "Bash"]
      deny: ["Bash(rm -rf /*)"]
    hooks:
      PreToolUse:
        - matcher: "Bash(git commit*)"
          command: ".wave/hooks/pre-commit-lint.sh"
      PostToolUse:
        - matcher: "Write(src/**)"
          command: "go test ./..."

  auditor:
    adapter: claude
    description: "Security and quality review"
    system_prompt_file: .wave/personas/auditor.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Grep", "Bash(go vet*)"]
      deny: ["Write(*)", "Edit(*)"]

  summarizer:
    adapter: claude
    description: "Relay checkpoint summarizer"
    system_prompt_file: .wave/personas/summarizer.md
    temperature: 0.0
    permissions:
      allowed_tools: ["Read"]
      deny: ["Write(*)", "Bash(*)"]

runtime:
  workspace_root: .wave/workspaces
  max_concurrent_workers: 5
  default_timeout_minutes: 5
  relay:
    token_threshold_percent: 80
    strategy: summarize_to_checkpoint
  audit:
    log_dir: .wave/traces/
    log_all_tool_calls: true
    log_all_file_operations: false
  meta_pipeline:
    max_depth: 2
    max_total_steps: 20
    max_total_tokens: 500000
    timeout_minutes: 60

skills:
  speckit:
    install: "npm install -g @anthropic/speckit"
    check: "speckit --version"
```
