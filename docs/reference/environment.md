# Environment Variables & Credentials

Reference for all environment variables that control Wave behavior, and the credential handling model.

## Wave Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `WAVE_DEBUG` | `bool` | `false` | Enable debug logging. Outputs verbose execution traces to stderr. |
| `WAVE_WORKSPACE_ROOT` | `string` | `/tmp/wave` | Override the default workspace root. Takes precedence over `runtime.workspace_root` in the manifest. |
| `WAVE_LOG_FORMAT` | `string` | `json` | Event output format: `json` (NDJSON, machine-parseable) or `text` (human-friendly with color). |
| `WAVE_MANIFEST` | `string` | `wave.yaml` | Default manifest file path. Overridden by `--manifest` flag. |
| `WAVE_NO_COLOR` | `bool` | `false` | Disable colored output in text log format. Also respects `NO_COLOR` standard. |

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
      WAVE_LOG_FORMAT: json
      WAVE_WORKSPACE_ROOT: /tmp/wave-ci
    steps:
      - uses: actions/checkout@v4
      - run: wave run .wave/pipelines/ci-flow.yaml "CI run"
```

> **Note:** You can also use the shorthand `wave run ci-flow "CI run"`. Wave will automatically look for pipelines in the `.wave/pipelines/` directory.

```yaml
# GitLab CI example
wave-pipeline:
  script:
    - wave run .wave/pipelines/ci-flow.yaml "CI run"
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
    WAVE_LOG_FORMAT: json
```

> **Note:** Shorthand `wave run ci-flow "CI run"` also works.

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
