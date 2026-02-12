# Integrations Overview

Wave integrates seamlessly with CI/CD platforms, enabling automated AI pipelines as part of your development workflow.

## Architecture

Wave runs as a single static binary with no runtime dependencies. This makes CI/CD integration straightforward:

```
CI Runner
    │
    ▼
┌───────────────┐
│  Wave Binary  │
└───────────────┘
    │
    ▼
┌───────────────┐
│  LLM Adapter  │ (e.g., claude, opencode)
└───────────────┘
    │
    ▼
┌───────────────┐
│    LLM API    │
└───────────────┘
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

For general troubleshooting, see [Troubleshooting Reference](/reference/troubleshooting).

For error code details, see [Error Code Reference](/reference/error-codes).

## Next Steps

- [Error Codes Reference](/reference/error-codes) - Complete error code catalog
- [CLI Reference](/reference/cli) - Complete command documentation
