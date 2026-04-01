# Adapters Reference

Adapters wrap LLM CLI tools for subprocess invocation. Wave ships with support for Claude Code, OpenCode, Gemini Code, and Codex.

## Adapter Selection

Wave uses a 4-tier precedence system to determine which adapter a step runs with (strongest to weakest):

1. **CLI `--adapter` flag** — e.g., `wave run my-pipeline --adapter opencode` overrides all steps
2. **Step-level `adapter:`** — specified per step in pipeline YAML
3. **Persona-level `adapter:`** — set on the persona in `wave.yaml`
4. **Adapter default** — from the adapter definition in `wave.yaml`

This means you can set a project-wide default via persona configuration, override individual steps in the pipeline, and still force a specific adapter at the CLI for one-off runs.

## Claude Code Adapter

The primary adapter for the `claude` CLI.

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
      allowed_tools: ["Read", "Write", "Edit", "Bash", "Glob", "Grep"]
      deny: []
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `binary` | `string` | — | CLI binary name. Must resolve via `$PATH`. |
| `mode` | `string` | — | Always `headless` for subprocess execution. |
| `output_format` | `string` | `json` | Output format. Only `json` supported. |
| `project_files` | `[]string` | `[]` | Files copied to workspace. Supports globs. |
| `default_permissions` | `Permissions` | allow all | Default tool permissions for personas. |

### Workspace Setup

When the Claude adapter runs, it generates two configuration files in the workspace:

1. **`.claude/settings.json`** — Claude Code reads this for permissions, model, and sandbox config
2. **`CLAUDE.md`** — Claude Code reads this as a system prompt with restriction directives

Both are derived from the manifest's persona and runtime configuration.

### Generated `settings.json` Structure

```json
{
  "model": "opus",
  "temperature": 0.7,
  "output_format": "stream-json",
  "permissions": {
    "allow": ["Read", "Write", "Edit", "Bash", "Glob", "Grep"],
    "deny": ["Bash(rm -rf /*)"]
  },
  "sandbox": {
    "enabled": true,
    "allowUnsandboxedCommands": false,
    "autoAllowBashIfSandboxed": true,
    "network": {
      "allowedDomains": ["api.anthropic.com", "github.com"]
    }
  }
}
```

| Field | Source | Description |
|-------|--------|-------------|
| `permissions.allow` | `persona.permissions.allowed_tools` | Tools the persona can use |
| `permissions.deny` | `persona.permissions.deny` | Tools explicitly denied |
| `sandbox.enabled` | Present when `allowed_domains` configured | Enables Claude Code's built-in sandbox |
| `sandbox.network.allowedDomains` | `persona.sandbox.allowed_domains` or `runtime.sandbox.default_allowed_domains` | Network domain allowlist |

### Generated `CLAUDE.md` Structure

The adapter writes the persona's system prompt followed by a restriction section:

```markdown
# Navigator

You are the navigator persona...

---

## Restrictions

The following restrictions are enforced by the pipeline orchestrator.

### Denied Tools

- `Write(*)`
- `Edit(*)`

### Allowed Tools

You may ONLY use the following tools:

- `Read`
- `Glob`
- `Grep`

### Network Access

Network requests are restricted to:

- `api.anthropic.com`
```

### Environment Hygiene

The adapter passes a **curated environment** to subprocesses instead of the full host environment. Only these variables are included:

- `HOME`, `PATH`, `TERM`, `TMPDIR` (base)
- `DISABLE_TELEMETRY`, `DISABLE_ERROR_REPORTING` (telemetry suppression)
- Variables listed in `runtime.sandbox.env_passthrough` (explicit passthrough)
- Step-specific env vars from pipeline config

This prevents credential leakage from unrelated host environment variables (e.g., `AWS_SECRET_ACCESS_KEY`).

### CLI Invocation

```bash
claude -p --model opus --allowedTools "Read,Write" --output-format stream-json \
  --verbose --dangerously-skip-permissions --no-session-persistence "prompt"
```

---

## Gemini Code Adapter

Adapter for Google's Gemini Code CLI (`gemini`), v0.34.0+.

```yaml
adapters:
  gemini:
    binary: gemini
    mode: headless
    output_format: json
    project_files:
      - GEMINI.md
    default_permissions:
      allowed_tools:
        - Read
        - Bash
      deny: []
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `binary` | `string` | — | CLI binary name. Must resolve via `$PATH`. |
| `mode` | `string` | — | Always `headless` for subprocess execution. |
| `output_format` | `string` | `json` | Output format. Only `json` supported. |
| `project_files` | `[]string` | `[]` | Files copied to workspace. Supports globs. |
| `default_permissions` | `Permissions` | allow all | Default tool permissions for personas. |

### Model Format

Gemini uses **plain model names** — no provider prefix:

| Model | Description |
|-------|-------------|
| `gemini-2.0-pro` | Gemini 2.0 Pro |
| `gemini-3-flash-preview` | Gemini 3 Flash (preview) |

If no model is specified, the Gemini binary uses its own default.

### Workspace Setup

When the Gemini adapter runs, it generates a `GEMINI.md` file in the workspace containing the persona's system prompt and restriction directives. Gemini Code reads this file for context.

### Output Format

The adapter parses an NDJSON stream with these event types:

| Event `type` | Description |
|---------------|-------------|
| `tool_use` | Tool invocation with name and input |
| `text` | Text content delta |
| `result` | Final result with content and usage stats (`input_tokens`, `output_tokens`) |

### CLI Invocation

```bash
gemini --yolo --output-format stream-json -p "prompt"
gemini --model gemini-2.0-pro --yolo --output-format stream-json -p "prompt"
```

---

## Codex Adapter

Adapter for the OpenAI Codex CLI (`codex`).

```yaml
adapters:
  codex:
    binary: codex
    mode: headless
    output_format: json
    project_files:
      - AGENTS.md
    default_permissions:
      allowed_tools:
        - Read
        - Bash
      deny: []
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `binary` | `string` | — | CLI binary name. Must resolve via `$PATH`. |
| `mode` | `string` | — | Always `headless` for subprocess execution. |
| `output_format` | `string` | `json` | Output format. Only `json` supported. |
| `project_files` | `[]string` | `[]` | Files copied to workspace. Supports globs. |
| `default_permissions` | `Permissions` | allow all | Default tool permissions for personas. |

### Model Format

Codex supports both **plain model names** for OpenAI models and `provider/model` format for other providers:

| Model | Description |
|-------|-------------|
| `openai/gpt-4o` | OpenAI GPT-4o (explicit provider) |
| `gpt-4o` | OpenAI GPT-4o (plain, inferred as OpenAI) |

### Workspace Setup

When the Codex adapter runs, it generates an `AGENTS.md` file in the workspace containing the persona's system prompt. The Codex CLI reads this file for agent instructions.

---

## OpenCode Adapter

Alternative adapter for the `opencode` CLI, supporting multiple model providers.

```yaml
adapters:
  opencode:
    binary: opencode
    mode: headless
    output_format: json
    default_permissions:
      allowed_tools: ["Read", "Write", "Edit"]
      deny: ["Bash(rm *)"]
```

### Model Configuration

OpenCode uses a `provider/model` format to select which LLM backend to use. Specify the model on the persona:

```yaml
personas:
  openai-coder:
    adapter: opencode
    model: openai/gpt-4o
```

The model string is split on the **first `/`** only:

| `model` value | Provider | Model |
|---------------|----------|-------|
| `openai/gpt-4o` | `openai` | `gpt-4o` |
| `anthropic/claude-sonnet-4-20250514` | `anthropic` | `claude-sonnet-4-20250514` |
| `google/gemini-2.0-flash` | `google` | `gemini-2.0-flash` |
| `openai/org/gpt-4o` | `openai` | `org/gpt-4o` (multi-slash: splits on first `/` only) |
| _(not set)_ | `anthropic` | `claude-sonnet-4-20250514` (default) |

If no model is configured, OpenCode defaults to `anthropic/claude-sonnet-4-20250514`.

### Environment Passthrough

OpenCode uses the same **curated environment** as the Claude adapter. Only base variables and those explicitly listed in `runtime.sandbox.env_passthrough` are passed to the subprocess. Configure API keys for your chosen provider:

```yaml
runtime:
  sandbox:
    env_passthrough:
      - OPENAI_API_KEY
      - GOOGLE_API_KEY
      - ANTHROPIC_API_KEY
```

Variables not listed here are not visible to the OpenCode subprocess, preventing credential leakage from unrelated host environment variables.

### Workspace Setup

When the OpenCode adapter runs:
1. Creates `.opencode/config.json` with provider and model settings derived from the persona's `model` field
2. Projects the persona system prompt to `AGENTS.md`

### Complete Persona Example

```yaml
adapters:
  opencode:
    binary: opencode
    mode: headless
    output_format: json
    default_permissions:
      allowed_tools: ["Read", "Write", "Edit", "Bash"]
      deny: []

personas:
  openai-coder:
    adapter: opencode
    model: openai/gpt-4o
    system_prompt: |
      You are a senior software engineer focused on clean implementation.

runtime:
  sandbox:
    env_passthrough:
      - OPENAI_API_KEY
```

### CLI Invocation

```bash
opencode --prompt "prompt" --output-format json --non-interactive
```

---

## GitHub Adapter

The GitHub adapter wraps the GitHub API for direct repository operations. Unlike the Claude and OpenCode adapters, it does not invoke a subprocess CLI — it makes GitHub API calls directly using the `GITHUB_TOKEN` or `GH_TOKEN` environment variable.

### Purpose

The GitHub adapter enables pipelines to perform GitHub API operations:

- **Issue management** — list, analyze, retrieve, and update issues
- **Pull request creation** — create PRs from pipeline-generated branches
- **Repository queries** — retrieve repository metadata
- **Branch creation** — create feature branches for pipeline workflows

### Configuration

The GitHub adapter is configured internally and does not require an entry in the `adapters:` section of `wave.yaml`. It is used automatically by GitHub-related pipelines.

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub personal access token. The adapter reads this via `os.Getenv("GITHUB_TOKEN")`. |
| `GH_TOKEN` | Alternative token variable. Pass through via `runtime.sandbox.env_passthrough`. |

At least one of these must be set for the adapter to authenticate with the GitHub API.

### Supported Operations

| Operation | Description |
|-----------|-------------|
| `list_issues` | List open issues for a repository |
| `analyze_issues` | Analyze issues for quality (below a configurable threshold) |
| `get_issue` | Retrieve a single issue by number |
| `update_issue` | Update an issue's title, body, state, or labels |
| `create_pr` | Create a pull request |
| `get_repo` | Retrieve repository metadata |
| `create_branch` | Create a new branch from a reference |

### How It Differs from LLM Adapters

| Aspect | LLM Adapters (Claude, OpenCode) | GitHub Adapter |
|--------|--------------------------------|----------------|
| Execution | Subprocess CLI invocation | Direct API calls |
| Output | LLM-generated text | Structured JSON data |
| Workspace | Generates config files (`.claude/settings.json`, `CLAUDE.md`) | No workspace setup |
| Permissions | Tool allow/deny lists | GitHub API token scope |

### Usage Context

The GitHub adapter is used by pipelines that interact with GitHub repositories:

- `plan-research` — scan and analyze issue quality
- `ops-rewrite` — enhance poorly written issues
- `impl-issue` — implement features from GitHub issues

---

## Multiple Adapters

A project can define multiple adapters in `wave.yaml` and switch between them at runtime via the 4-tier precedence system.

```yaml
adapters:
  claude:
    binary: claude
    mode: headless
  opencode:
    binary: opencode
    mode: headless
  gemini:
    binary: gemini
    mode: headless
  codex:
    binary: codex
    mode: headless

personas:
  navigator:
    adapter: claude
  reviewer:
    adapter: gemini
    model: gemini-2.0-pro
```

### Mixing Adapters in a Pipeline

Each step can specify its own adapter, enabling different steps to use different LLM backends:

```yaml
steps:
  - id: analyze
    persona: navigator
    adapter: opencode
    model: "zai-coding-plan/glm-5-turbo"
  - id: implement
    persona: craftsman
    # Uses persona's adapter (claude) by default
  - id: review
    persona: reviewer
    adapter: gemini
    model: "gemini-2.0-pro"
```

### CLI Overrides

```bash
wave run my-pipeline --adapter opencode                    # Override all steps to use opencode
wave run my-pipeline --adapter opencode --model "zai-coding-plan/glm-5-turbo"  # Override adapter + model
wave run my-pipeline --model "gemini-2.0-pro"              # Override model only (persona's adapter stays)
```

---

## Environment and Credentials

Adapters receive a **curated environment** — not the full host environment. Only explicitly allowed variables are passed through.

| Variable | Source | Purpose |
|----------|--------|---------|
| `HOME`, `PATH`, `TERM`, `TMPDIR` | Always included | Base operation |
| `ANTHROPIC_API_KEY` | Via `runtime.sandbox.env_passthrough` | Claude API authentication |
| `GH_TOKEN` | Via `runtime.sandbox.env_passthrough` | GitHub CLI authentication |

Configure which variables are passed to adapter subprocesses:

```yaml
runtime:
  sandbox:
    env_passthrough:
      - ANTHROPIC_API_KEY
      - GH_TOKEN
```

::: warning
Credentials are **never** written to disk. They flow via curated process environment only. Variables not in `env_passthrough` are not visible to adapter subprocesses.
:::

---

## Timeout Handling

- Default timeout: 5 minutes per step
- Override via `runtime.default_timeout_minutes` or `--timeout` flag
- On timeout, entire process group receives `SIGKILL`
- Timeout counts as step failure, triggering retry logic

```yaml
runtime:
  default_timeout_minutes: 30
```

---

## Model Format

Model format varies by adapter:

| Adapter | Model Format | Example |
|---------|-------------|---------|
| claude | Short names or full IDs | `sonnet`, `haiku`, `claude-opus-4-5-20251101` |
| opencode | `provider/model` for multi-provider, or short names | `zai-coding-plan/glm-5-turbo`, `openai/gpt-4o`, `anthropic/claude-sonnet-4-20250514` |
| gemini | Plain model names | `gemini-2.0-pro`, `gemini-3-flash-preview` |
| codex | `provider/model` or plain names | `openai/gpt-4o` |

---

## Validation

```bash
$ wave validate --verbose
  Adapter 'claude' binary found: /usr/local/bin/claude
  Adapter 'opencode' binary found: /usr/local/bin/opencode
  Adapter 'gemini' binary found: /usr/local/bin/gemini
  Adapter 'codex' binary not found on PATH  # Warning only
```

Binary warnings do not block validation - the binary may be available at runtime.
