# Environment Variables & Credentials

Reference for all environment variables that control Muzzle behavior, and the credential handling model.

## Muzzle Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `MUZZLE_DEBUG` | `bool` | `false` | Enable debug logging. Outputs verbose execution traces to stderr. |
| `MUZZLE_WORKSPACE_ROOT` | `string` | `/tmp/muzzle` | Override the default workspace root. Takes precedence over `runtime.workspace_root` in the manifest. |
| `MUZZLE_LOG_FORMAT` | `string` | `json` | Event output format: `json` (NDJSON, machine-parseable) or `text` (human-friendly with color). |
| `MUZZLE_MANIFEST` | `string` | `muzzle.yaml` | Default manifest file path. Overridden by `--manifest` flag. |
| `MUZZLE_NO_COLOR` | `bool` | `false` | Disable colored output in text log format. Also respects `NO_COLOR` standard. |

### Precedence Order

Configuration values are resolved in this order (highest priority first):

1. CLI flags (`--manifest`, `--pipeline`, etc.)
2. Environment variables (`MUZZLE_*`)
3. Manifest values (`muzzle.yaml`)
4. Built-in defaults

## Credential Handling

Muzzle enforces a strict credential model: **credentials never touch disk**.

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
    Muzzle Process (inherits all env vars)
         │
         ▼
    Adapter Subprocess (inherits all env vars)
         │
         ▼
    LLM CLI (e.g., claude) uses ANTHROPIC_API_KEY
```

Adapter subprocesses inherit the full environment from the Muzzle parent process. This means:

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

Muzzle itself requires no environment variables. However, adapters typically need credentials:

| Adapter | Required Variables | Description |
|---------|-------------------|-------------|
| Claude Code | `ANTHROPIC_API_KEY` | Anthropic API key for Claude. |
| OpenCode | varies | Depends on configured LLM provider. |

### CI/CD Configuration

```yaml
# GitHub Actions example
jobs:
  muzzle-pipeline:
    runs-on: ubuntu-latest
    env:
      ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
      MUZZLE_LOG_FORMAT: json
      MUZZLE_WORKSPACE_ROOT: /tmp/muzzle-ci
    steps:
      - uses: actions/checkout@v4
      - run: muzzle run --pipeline .muzzle/pipelines/ci-flow.yaml --input "CI run"
```

```yaml
# GitLab CI example
muzzle-pipeline:
  script:
    - muzzle run --pipeline .muzzle/pipelines/ci-flow.yaml --input "CI run"
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
    MUZZLE_LOG_FORMAT: json
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
