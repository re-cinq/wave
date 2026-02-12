# Enterprise Patterns

Scale Wave across multiple projects and teams with shared configuration, centralized standards, and comprehensive audit trails.

## Prerequisites

- Wave deployed to teams
- Git repository for shared configuration
- CI/CD pipeline access

## Step-by-Step

### 1. Create Shared Configuration Repository

Centralize personas and contracts for consistency:

```
org-wave-config/
├── personas/
│   ├── navigator.md
│   ├── auditor.md
│   └── craftsman.md
├── contracts/
│   ├── code-review.schema.json
│   └── security-report.schema.json
└── README.md
```

### 2. Reference Shared Config

Projects include shared configuration via git submodule:

```bash
git submodule add git@github.com:org/org-wave-config.git .wave/shared
```

### 3. Override Per-Project

Each project extends shared configuration:

```yaml
# wave.yaml
personas:
  # Use shared navigator
  navigator:
    adapter: claude
    system_prompt_file: .wave/shared/personas/navigator.md
    permissions:
      allowed_tools: [Read, Glob, Grep]
      deny: [Write, Edit, Bash]

  # Project-specific persona
  domain-expert:
    adapter: claude
    system_prompt_file: .wave/personas/domain-expert.md
    permissions:
      allowed_tools: [Read, Glob]
      deny: [Write, Edit, Bash]
```

## Security Controls

### Permission Enforcement

Personas define what AI can and cannot do:

```yaml
personas:
  reader:
    permissions:
      allowed_tools: [Read, Glob, Grep]
      deny: [Write, Edit, Bash]

  writer:
    permissions:
      allowed_tools: [Read, Write, Edit]
      deny:
        - Bash(rm -rf *)
        - Write(/etc/*)
```

### Audit Logging

Enable comprehensive logging for compliance:

```yaml
runtime:
  audit:
    log_all_tool_calls: true
    log_all_file_operations: true
    log_dir: .wave/traces/
```

Logs capture:
- Tool calls made
- Files accessed
- Outputs generated
- Execution timestamps

### API Key Management

Use environment variables, never files:

```bash
export ANTHROPIC_API_KEY=sk-...
```

In CI/CD, use secrets management:

```yaml
# GitHub Actions
env:
  ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

## Scaling Patterns

### Parallel Step Execution

Wave runs independent steps concurrently:

```yaml
steps:
  - id: analyze
    # Runs first

  - id: security
    dependencies: [analyze]
    # Runs in parallel with quality

  - id: quality
    dependencies: [analyze]
    # Runs in parallel with security

  - id: summary
    dependencies: [security, quality]
    # Waits for both
```

### Large Codebases

Limit scope with workspace mounts:

```yaml
workspace:
  mount:
    - source: ./src/auth  # Only auth module
      target: /code
      mode: readonly
```

### Timeout Configuration

Set appropriate limits:

```yaml
runtime:
  default_timeout_minutes: 30
```

## Complete Example

Enterprise-ready manifest:

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: enterprise-project

adapters:
  claude:
    binary: claude
    mode: headless

personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/shared/personas/navigator.md
    permissions:
      allowed_tools: [Read, Glob, Grep]
      deny: [Write, Edit, Bash]

  auditor:
    adapter: claude
    system_prompt_file: .wave/shared/personas/auditor.md
    permissions:
      allowed_tools: [Read, Grep]
      deny: [Write, Edit, Bash]

  craftsman:
    adapter: claude
    system_prompt_file: .wave/personas/craftsman.md
    permissions:
      allowed_tools: [Read, Write, Edit]
      deny:
        - Bash(rm -rf *)
        - Write(/etc/*)

runtime:
  workspace_root: .wave/workspaces
  default_timeout_minutes: 30
  audit:
    log_all_tool_calls: true
    log_all_file_operations: true
    log_dir: .wave/traces/
```

## Monitoring

### Execution Status

```bash
# List runs
wave list

# Check specific run
wave status <run-id>

# View logs
wave logs <run-id>
```

### Audit Trail

All executions logged to `runtime.audit.log_dir`:

```
.wave/traces/
├── run-abc123/
│   ├── step-analyze.log
│   └── step-review.log
└── run-def456/
    └── ...
```

## Cost Management

### Workspace Cleanup

Clean up old runs:

```bash
# Remove all workspaces
wave clean

# Keep last N runs
wave clean --keep 10
```

## Best Practices

1. **Centralize Standards** - Shared personas and contracts ensure consistency
2. **Start Small** - Begin with one team, one pipeline; expand based on results
3. **Review Changes** - Pipeline configurations go through code review
4. **Monitor Usage** - Track which pipelines run, how often, and outcomes
5. **Document Everything** - Clear descriptions in pipeline metadata

## Compliance

Wave's logging provides audit requirements:

- Tool calls made
- Files accessed
- Outputs generated
- Execution timestamps

Workspaces are ephemeral by default. Monitor disk usage and configure retention as needed.

## Next Steps

- [Audit Logging](/guides/audit-logging) - Detailed audit configuration
- [CI/CD Integration](/guides/ci-cd) - Automate pipelines in builds
- [Contracts](/concepts/contracts) - Validate pipeline outputs
