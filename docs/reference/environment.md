# Environment Variables & Credentials

Reference for all environment variables that control Wave behavior, and the credential handling model.

## Wave Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `WAVE_FORCE_TTY` | `bool` | _(auto)_ | Override TTY detection for `-o auto` mode. Set `1` to force TUI, `0` to force plain text. Useful in CI and testing. |
| `WAVE_SERVE_TOKEN` | `string` | `""` | Authentication token for `wave serve` dashboard. Can also be set via `--token` flag. Auto-generated when binding to non-localhost addresses. |
| `WAVE_MIGRATION_ENABLED` | `bool` | `true` | Enable the database migration system. |
| `WAVE_AUTO_MIGRATE` | `bool` | `true` | Automatically apply pending migrations on startup. |
| `WAVE_SKIP_MIGRATION_VALIDATION` | `bool` | `false` | Skip migration checksum validation (development only). |
| `WAVE_MAX_MIGRATION_VERSION` | `int` | `0` | Limit migrations to this version (0 = unlimited). Useful for gradual rollout. |
| `NO_COLOR` | `string` | _(unset)_ | Disable colored output. Any non-empty value disables color. Follows the [NO_COLOR](https://no-color.org) standard. |

### Precedence Order

Configuration values are resolved in this order (highest priority first):

1. CLI flags (`--manifest`, `--pipeline`, etc.)
2. Environment variables (`WAVE_*`)
3. Manifest values (`wave.yaml`)
4. Built-in defaults

### Default Step Timeout

The default step timeout is `5` minutes (from `internal/timeouts.StepDefault`).
Override via `runtime.timeouts.step_default_minutes` in `wave.yaml`, or per-run with `--timeout`.

## Display & Terminal Variables

These variables influence Wave's terminal display behavior:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `COLUMNS` | `int` | _(auto)_ | Override terminal width detection. |
| `LINES` | `int` | _(auto)_ | Override terminal height detection. |
| `COLORTERM` | `string` | _(auto)_ | Color support hint. `truecolor` or `24bit` enables 24-bit color. |
| `NO_UNICODE` | `string` | `""` | Disable Unicode characters in output. Any non-empty value enables ASCII fallback. |
| `NERD_FONT` | `string` | `""` | Enable Nerd Font icons in deliverable output. Set to `"1"` to enable. Also auto-detected for kitty, alacritty, and JetBrains terminals. |

## CI/CD Detection Variables

Wave automatically detects CI/CD environments by checking for these variables (read-only, no configuration needed):

| Variable | CI System |
|----------|-----------|
| `CI` | Generic CI |
| `CONTINUOUS_INTEGRATION` | Generic CI |
| `BUILD_ID` | Jenkins / Generic |
| `BUILD_NUMBER` | Jenkins |
| `RUN_ID` | Generic |
| `GITHUB_ACTIONS` | GitHub Actions |
| `GITLAB_CI` | GitLab CI |
| `CIRCLECI` | CircleCI |
| `TRAVIS` | Travis CI |
| `DRONE` | Drone CI |

When a CI environment is detected, Wave adjusts display behavior (disables interactive TUI, uses plain text output).

## Hook Environment Variables

When a lifecycle hook executes, Wave injects these variables into the hook subprocess environment:

| Variable | Description |
|----------|-------------|
| `WAVE_HOOK_EVENT` | The lifecycle event type (e.g., `pipeline.start`, `step.complete`, `pipeline.success`). |
| `WAVE_HOOK_PIPELINE` | The pipeline ID for the current run. |
| `WAVE_HOOK_STEP_ID` | The step ID that triggered the hook (empty for pipeline-level events). |
| `WAVE_HOOK_WORKSPACE` | Absolute path to the workspace directory for the current run. |

These variables are available inside hook scripts and HTTP hook payloads. Only these WAVE_HOOK_* variables and base system variables (HOME, PATH, TERM, TMPDIR) are injected — the full host environment is not passed.

## Credential Handling

Wave enforces a strict credential model: **credentials never touch disk**.

### How Credentials Flow

```
Shell Environment
    │
    ├── ANTHROPIC_API_KEY
    ├── GH_TOKEN
    ├── DATABASE_URL (blocked)
    └── ...
         │
         ▼
    Wave Process
         │  env_passthrough filter
         ▼
    Adapter Subprocess (curated env only)
         │
         ▼
    LLM CLI (e.g., claude)
```

The Claude adapter constructs a curated environment for each subprocess. Only base variables (HOME, PATH, TERM, TMPDIR) and those explicitly listed in `runtime.sandbox.env_passthrough` are passed. Other host environment variables (e.g., `AWS_SECRET_ACCESS_KEY`, `DATABASE_PASSWORD`) are never inherited.

- **API keys** must be listed in `runtime.sandbox.env_passthrough` to reach adapter subprocesses.
- **No manifest entries** for credentials — they are never written to YAML files.
- **No checkpoint entries** — credentials are never serialized in relay checkpoints.
- **No audit log entries** — credential values are scrubbed from all logs.

**Note:** The `ProcessGroupRunner` (used for non-Claude adapters) currently inherits the full host environment. The curated model only applies to the Claude adapter.

### Credential Scrubbing

Audit logs automatically redact values for environment variables matching these patterns:

| Pattern (regex) | Examples |
|---------|----------|
| `API[_-]?KEY` | `ANTHROPIC_API_KEY`, `OPENAI_APIKEY` |
| `TOKEN` | `GITHUB_TOKEN`, `NPM_TOKEN` |
| `SECRET` | `AWS_SECRET_ACCESS_KEY`, `JWT_SECRET` |
| `PASSWORD` | `DATABASE_PASSWORD`, `SMTP_PASSWORD` |
| `CREDENTIAL` | `GCP_CREDENTIAL_FILE` |
| `AUTH` | `BASIC_AUTH`, `AUTH_TOKEN` |
| `PRIVATE[_-]?KEY` | `SSH_PRIVATE_KEY` |
| `ACCESS[_-]?KEY` | `AWS_ACCESS_KEY` |

Redacted values appear as `[REDACTED]` in audit logs.

### Required Environment Variables

Wave itself requires no environment variables. However, adapters typically need credentials:

| Adapter | Required Variables | Description |
|---------|-------------------|-------------|
| Claude Code | `ANTHROPIC_API_KEY` | Anthropic API key for Claude. |
| OpenCode | varies | Depends on configured LLM provider. |
| GitHub | `GH_TOKEN` or `GITHUB_TOKEN` | GitHub personal access token for the GitHub adapter. Required for GitHub-related pipelines (`plan-research`, `ops-rewrite`, `impl-issue`). `GH_TOKEN` is checked first, then `GITHUB_TOKEN` as fallback. If neither is set, Wave falls back to `gh auth token`. |

**Note:** For the Claude adapter, `ANTHROPIC_API_KEY` must be included in `runtime.sandbox.env_passthrough` in your `wave.yaml` manifest. Without this entry, the key will not reach the adapter subprocess even if it is set in your shell environment.

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
| `CLAUDE_CODE_MODEL` | Model override (e.g., `claude-sonnet-4-20250514`). Must be listed in `runtime.sandbox.env_passthrough` to reach adapter subprocesses. Wave's `--model` flag is the preferred mechanism for model selection. |
| `DISABLE_TELEMETRY` | Set to `1` to disable Claude Code telemetry. Wave sets this automatically in adapter subprocesses. |
| `DISABLE_ERROR_REPORTING` | Set to `1` to disable Claude Code error reporting. Wave sets this automatically. |
| `CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY` | Set to `1` to suppress the feedback survey prompt. Wave sets this automatically. |
| `DISABLE_BUG_COMMAND` | Set to `1` to disable the `/bug` command in Claude Code. Wave sets this automatically. |

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
