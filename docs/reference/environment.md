# Environment Variables & Credentials

Reference for all environment variables that control Wave behavior, and the credential handling model.

## Wave Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `WAVE_FORCE_TTY` | `bool` | _(auto)_ | Override TTY detection for `-o auto` mode. Set `1` to force TUI, `0` to force plain text. Useful in CI and testing. |
| `WAVE_MIGRATION_ENABLED` | `bool` | `true` | Enable the database migration system. |
| `WAVE_AUTO_MIGRATE` | `bool` | `true` | Automatically apply pending migrations on startup. |
| `WAVE_SKIP_MIGRATION_VALIDATION` | `bool` | `false` | Skip migration checksum validation (development only). |
| `WAVE_MAX_MIGRATION_VERSION` | `int` | `0` | Limit migrations to this version (0 = unlimited). Useful for gradual rollout. |
| `NO_COLOR` | `bool` | `false` | Disable colored output. Follows the [NO_COLOR](https://no-color.org) standard. |

### Precedence Order

Configuration values are resolved in this order (highest priority first):

1. CLI flags (`--manifest`, `--pipeline`, etc.)
2. Environment variables (`WAVE_*`)
3. Manifest values (`wave.yaml`)
4. Built-in defaults

## Credential Handling

Wave enforces a strict credential model: **credentials never touch disk**.

### How Credentials Flow

```
Shell Environment
    │
    ├── ANTHROPIC_API_KEY
    ├── GITHUB_TOKEN
    ├── DATABASE_URL
    └── ...
         │
         ▼
    Wave Process (inherits all env vars)
         │
         ▼
    Adapter Subprocess (inherits all env vars)
         │
         ▼
    LLM CLI (e.g., claude) uses ANTHROPIC_API_KEY
```

Adapter subprocesses inherit the full environment from the Wave parent process. This means:

- **API keys** (e.g., `ANTHROPIC_API_KEY`) are available to adapter subprocesses without configuration.
- **No manifest entries** for credentials — they are never written to YAML files.
- **No checkpoint entries** — credentials are never serialized in relay checkpoints.
- **No audit log entries** — credential values are scrubbed from all logs.

### Credential Scrubbing

Audit logs automatically redact values for environment variables matching these patterns:

| Pattern | Examples |
|---------|----------|
| `*_KEY` | `ANTHROPIC_API_KEY`, `AWS_ACCESS_KEY` |
| `*_TOKEN` | `GITHUB_TOKEN`, `NPM_TOKEN` |
| `*_SECRET` | `AWS_SECRET_ACCESS_KEY`, `JWT_SECRET` |
| `*_PASSWORD` | `DATABASE_PASSWORD`, `SMTP_PASSWORD` |
| `*_CREDENTIAL*` | `GCP_CREDENTIAL_FILE` |

Redacted values appear as `[REDACTED]` in audit logs.

### Required Environment Variables

Wave itself requires no environment variables. However, adapters typically need credentials:

| Adapter | Required Variables | Description |
|---------|-------------------|-------------|
| Claude Code | `ANTHROPIC_API_KEY` | Anthropic API key for Claude. |
| OpenCode | varies | Depends on configured LLM provider. |

### CI/CD Configuration

```yaml
# GitHub Actions example
jobs:
  wave-pipeline:
    runs-on: ubuntu-latest
    env:
      ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
    steps:
      - uses: actions/checkout@v4
      - run: wave run ci-flow "CI run" -o json
```

```yaml
# GitLab CI example
wave-pipeline:
  script:
    - wave run ci-flow "CI run" -o json
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
```

## Adapter Environment Variables

Adapters may use additional environment variables for configuration:

### Claude Code Adapter

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | API key for Claude. |
| `CLAUDE_CODE_MAX_TURNS` | Maximum agentic turns per invocation. |
| `CLAUDE_CODE_MODEL` | Model override (e.g., `claude-sonnet-4-20250514`). |

### Custom Adapters

Custom adapters can use any environment variables. Document required variables in the adapter's description field:

```yaml
adapters:
  custom-llm:
    binary: my-llm-cli
    mode: headless
    # Document required env vars in description
    # Requires: MY_LLM_API_KEY, MY_LLM_ENDPOINT
```
