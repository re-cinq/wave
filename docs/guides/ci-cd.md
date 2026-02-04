# CI/CD Integration

Automate Wave pipelines in your CI/CD workflows. Run code reviews, security audits, and documentation generation on every pull request.

## Prerequisites

- Wave installed on CI runner
- API key stored as CI secret
- Pipeline definitions committed to repository

## GitHub Actions

### Basic Setup

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
        run: wave run code-review "${{ github.event.pull_request.title }}"
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

### With Artifact Upload

```yaml
name: Wave Review
on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Wave
        run: curl -L https://wave.dev/install.sh | sh

      - name: Run Review
        run: wave run code-review "${{ github.event.pull_request.title }}"
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Upload Results
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: review-results
          path: .wave/workspaces/*/output/

      - name: Upload Audit Logs
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: audit-logs
          path: .wave/traces/
```

### Multiple Pipelines

```yaml
name: Wave CI
on: [pull_request]

jobs:
  code-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: curl -L https://wave.dev/install.sh | sh
      - run: wave run code-review "${{ github.event.pull_request.title }}"
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

  security-audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: curl -L https://wave.dev/install.sh | sh
      - run: wave run security-audit
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

## GitLab CI

### Basic Setup

```yaml
stages:
  - review

wave-review:
  stage: review
  script:
    - curl -L https://wave.dev/install.sh | sh
    - wave run code-review "$CI_MERGE_REQUEST_TITLE"
  artifacts:
    paths:
      - .wave/workspaces/*/output/
    when: always
  rules:
    - if: $CI_MERGE_REQUEST_IID
```

### With Variables

```yaml
variables:
  ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY

wave-review:
  stage: review
  script:
    - curl -L https://wave.dev/install.sh | sh
    - wave run code-review "$CI_MERGE_REQUEST_TITLE"
  artifacts:
    paths:
      - .wave/workspaces/*/output/
      - .wave/traces/
```

### Parallel Jobs

```yaml
stages:
  - analyze

code-review:
  stage: analyze
  script:
    - curl -L https://wave.dev/install.sh | sh
    - wave run code-review "$CI_MERGE_REQUEST_TITLE"

security-audit:
  stage: analyze
  script:
    - curl -L https://wave.dev/install.sh | sh
    - wave run security-audit
```

## Complete Example

A production-ready GitHub Actions workflow:

```yaml
name: Wave Analysis
on:
  pull_request:
    types: [opened, synchronize]

env:
  ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Wave
        run: curl -L https://wave.dev/install.sh | sh

      - name: Validate Configuration
        run: wave validate

  review:
    needs: validate
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for diff

      - name: Install Wave
        run: curl -L https://wave.dev/install.sh | sh

      - name: Run Code Review
        run: |
          wave run code-review "${{ github.event.pull_request.title }}"

      - name: Check Status
        run: wave status

      - name: Get Artifacts
        run: wave artifacts --latest

      - name: Upload Results
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: wave-results
          path: |
            .wave/workspaces/*/output/
            .wave/traces/
```

## Best Practices

### Store API Keys as Secrets

Never commit API keys. Use CI secret management:

```yaml
# GitHub: Settings > Secrets > Actions
ANTHROPIC_API_KEY: sk-ant-...

# GitLab: Settings > CI/CD > Variables
ANTHROPIC_API_KEY: sk-ant-... (masked)
```

### Enable Audit Logging

```yaml
# In wave.yaml
runtime:
  audit:
    log_all_tool_calls: true
    log_dir: .wave/traces/
```

### Upload Artifacts on Failure

Use `if: always()` or `when: always` to capture results even when pipelines fail:

```yaml
# GitHub Actions
- uses: actions/upload-artifact@v4
  if: always()

# GitLab CI
artifacts:
  when: always
```

### Cache Wave Binary

Speed up CI runs by caching the Wave installation:

```yaml
# GitHub Actions
- uses: actions/cache@v4
  with:
    path: ~/.wave
    key: wave-${{ runner.os }}
```

## Troubleshooting

### API Key Not Found

Ensure the secret is set and available:

```yaml
# GitHub: Check secret name matches
env:
  ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

# GitLab: Check variable is not protected-only
variables:
  ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
```

### Pipeline Not Found

Verify the pipeline exists and wave.yaml is valid:

```yaml
- name: Debug
  run: |
    ls -la .wave/pipelines/
    wave validate
```

### Timeout Issues

Set appropriate timeouts in both CI and Wave:

```yaml
# CI timeout
jobs:
  review:
    timeout-minutes: 30

# Wave timeout (wave.yaml)
runtime:
  default_timeout_minutes: 20
```

## Next Steps

- [Enterprise Patterns](/guides/enterprise) - Scale Wave to your organization
- [Audit Logging](/guides/audit-logging) - Track all pipeline activity
