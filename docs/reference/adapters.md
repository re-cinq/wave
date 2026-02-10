# Adapters Reference

Adapters wrap LLM CLI tools for subprocess invocation. Wave ships with support for Claude Code and OpenCode.

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

## OpenCode Adapter

Alternative adapter for the `opencode` CLI.

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

### Workspace Setup

When OpenCode adapter runs:
1. Creates `.opencode/config.json` with provider and model settings
2. Projects persona system prompt to `AGENTS.md`

### CLI Invocation

```bash
opencode --prompt "prompt" --output-format json --non-interactive
```

---

## Multiple Adapters

Define multiple adapters for different use cases:

```yaml
adapters:
  claude:
    binary: claude
    mode: headless
  opencode:
    binary: opencode
    mode: headless

personas:
  navigator:
    adapter: claude   # Uses Claude
  reviewer:
    adapter: opencode # Uses OpenCode
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

- Default timeout: 10 minutes per step
- Override via `runtime.default_timeout_minutes` or `--timeout` flag
- On timeout, entire process group receives `SIGKILL`
- Timeout counts as step failure, triggering retry logic

```yaml
runtime:
  default_timeout_minutes: 30
```

---

## Validation

```bash
$ wave validate --verbose
 Adapter 'claude' binary found: /usr/local/bin/claude
 Adapter 'opencode' binary not found on PATH  # Warning only
```

Binary warnings do not block validation - the binary may be available at runtime.
