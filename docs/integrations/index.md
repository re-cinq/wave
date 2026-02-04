# Integrations Overview

Wave integrates seamlessly with CI/CD platforms, enabling automated AI pipelines as part of your development workflow.

## Supported Platforms

| Platform | Status | Guide |
|----------|--------|-------|
| GitHub Actions | Supported | [GitHub Actions Guide](./github-actions.md) |
| GitLab CI/CD | Supported | [GitLab CI Guide](./gitlab-ci.md) |
| Jenkins | Community | Coming soon |
| CircleCI | Community | Coming soon |

## Quick Start

All CI/CD integrations follow the same pattern:

1. **Install Wave** - Download and install the Wave binary
2. **Configure Secrets** - Set up API keys as CI secrets
3. **Run Pipelines** - Execute Wave pipelines in your CI jobs

### Minimal Example

```yaml
# Any CI platform
- curl -fsSL https://wave.dev/install.sh | sh
- wave run code-review
```

## Architecture

Wave runs as a single static binary with no runtime dependencies. This makes CI/CD integration straightforward:

```
CI Runner
    |
    v
+---------------+
|  Wave Binary  |
+---------------+
    |
    v
+---------------+
|  LLM Adapter  | (e.g., claude, opencode)
+---------------+
    |
    v
+---------------+
|    LLM API    |
+---------------+
```

## Prerequisites

### API Keys

Wave requires API keys for the LLM providers you use. Configure these as CI secrets:

| Provider | Environment Variable | Required |
|----------|---------------------|----------|
| Anthropic | `ANTHROPIC_API_KEY` | For Claude adapter |
| OpenAI | `OPENAI_API_KEY` | For OpenAI adapter |

### Adapter Binaries

Wave wraps LLM CLIs via subprocess execution. Ensure the required adapter binaries are available:

```bash
# Claude Code
npm install -g @anthropic-ai/claude-code

# Verify
which claude
```

## Common Patterns

### Pipeline Execution

Run a named pipeline:

```bash
wave run code-review
```

Run with input:

```bash
wave run code-review --input "Review the authentication module"
```

### Validation Before Run

Validate manifests before execution:

```bash
wave validate && wave run code-review
```

### Dry Run

Test pipeline structure without execution:

```bash
wave run code-review --dry-run
```

### Timeout Configuration

Set execution timeout:

```bash
wave run code-review --timeout 30
```

## Security Considerations

### API Key Management

- **Never** commit API keys to version control
- Use CI platform secret management
- Rotate keys regularly (every 90 days recommended)

### Permission Boundaries

Wave enforces persona permissions at runtime. In CI, ensure:

- Personas have minimal required permissions
- Deny patterns block destructive operations
- Audit logging is enabled for compliance

### Workspace Isolation

Each pipeline run creates an isolated workspace. Configure cleanup:

```yaml
runtime:
  workspace_root: /tmp/wave
```

## Troubleshooting

See the platform-specific guides for detailed troubleshooting:

- [GitHub Actions Troubleshooting](./github-actions.md#troubleshooting)
- [GitLab CI Troubleshooting](./gitlab-ci.md#troubleshooting)

For general troubleshooting, see [Troubleshooting Reference](/reference/troubleshooting).

For error code details, see [Error Code Reference](/reference/error-codes).

## Next Steps

- [GitHub Actions Guide](./github-actions.md) - Full setup for GitHub workflows
- [GitLab CI Guide](./gitlab-ci.md) - Full setup for GitLab pipelines
- [Error Codes Reference](/reference/error-codes) - Complete error code catalog
