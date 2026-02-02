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

When Claude adapter runs:
1. Creates `.claude/settings.json` with persona temperature and tools
2. Projects persona system prompt to `CLAUDE.md`
3. Copies `project_files` to workspace root

### CLI Invocation

```bash
claude -p --allowedTools "Read,Write" --output-format json --temperature 0.7 --no-continue "prompt"
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

Adapters inherit environment variables from the parent process:

| Variable | Purpose |
|----------|---------|
| `ANTHROPIC_API_KEY` | Claude API authentication |

::: warning
Credentials are **never** written to disk. They flow via process environment only.
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
