# Enterprise Patterns

Wave in larger organizations. This guide covers patterns that scale.

## Multi-Project Setup

### Shared Configuration Repository

For consistent standards across projects:

```
org-wave-config/
├── personas/
│   ├── navigator.md
│   ├── auditor.md
│   └── craftsman.md
├── contracts/
│   ├── gh-pr-review.schema.json
│   └── security-report.schema.json
└── README.md
```

Projects reference via git submodule:

```bash
git submodule add git@github.com:org/org-wave-config.git .wave/shared
```

Then in `wave.yaml`:

```yaml
skill_mounts:
  - path: .wave/shared/
```

### Project-Specific Overrides

Each project can extend shared configuration:

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

## Security Considerations

### Permission Controls

Personas enforce what AI can do:

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

Enable comprehensive logging:

```yaml
runtime:
  audit:
    log_all_tool_calls: true
    log_all_file_operations: true
    log_dir: .wave/traces/
```

Logs are stored per-run for review.

### API Key Management

Use environment variables, not files:

```bash
export ANTHROPIC_API_KEY=sk-...
```

For CI/CD, use secrets management:

```yaml
# GitHub Actions
env:
  ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Wave Pipeline
on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Wave
        run: |
          curl -L https://wave.dev/install.sh | sh

      - name: Run Review
        run: wave run gh-pr-review "${{ github.event.pull_request.title }}"
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: review
          path: .wave/workspaces/*/output/
```

### GitLab CI

```yaml
wave-review:
  stage: review
  script:
    - curl -L https://wave.dev/install.sh | sh
    - wave run gh-pr-review "$CI_MERGE_REQUEST_TITLE"
  artifacts:
    paths:
      - .wave/workspaces/*/output/
```

## Scaling Patterns

### Parallel Step Execution

Wave runs independent steps in parallel:

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

Configure concurrency:

```yaml
runtime:
  max_concurrent_workers: 5
```

### Large Codebases

Use workspace mounts to limit scope:

```yaml
workspace:
  mount:
    - source: ./src/auth  # Only auth module
      target: /code
      mode: readonly
```

### Timeout Configuration

Set appropriate timeouts:

```yaml
runtime:
  default_timeout_minutes: 30
  meta_pipeline:
    timeout_minutes: 60
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

### Token Limits

Control token usage in runtime config:

```yaml
runtime:
  meta_pipeline:
    max_total_tokens: 500000
```

### Step Timeouts

Prevent runaway executions:

```yaml
runtime:
  default_timeout_minutes: 10
```

## Best Practices

### 1. Centralize Standards

Shared personas and contracts ensure consistency.

### 2. Start Small

Begin with one team, one pipeline. Expand based on results.

### 3. Review Changes

Pipeline configurations go through code review like any code.

### 4. Monitor Usage

Track which pipelines run, how often, and outcomes.

### 5. Document Everything

Clear descriptions in pipeline metadata.

## Compliance

### Audit Requirements

Wave's logging provides:
- Tool calls made
- Files accessed
- Outputs generated
- Execution timestamps

### Data Handling

Workspaces are ephemeral by default. Configure retention:

```bash
# Clean up workspaces
wave clean

# Keep last N runs
wave clean --keep 10
```

## Next Steps

- [Team Adoption](/migration/team-adoption) - Getting started with teams
- [Creating Pipelines](/workflows/creating-workflows) - Pipeline reference
- [Pipeline Execution](/concepts/pipeline-execution) - How Wave runs
