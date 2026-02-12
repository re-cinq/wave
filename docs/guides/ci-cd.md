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
          curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh

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
        run: curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh

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
      - run: curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
      - run: wave run code-review "${{ github.event.pull_request.title }}"
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

  security-audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
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
    - curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
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
    - curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
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
    - curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
    - wave run code-review "$CI_MERGE_REQUEST_TITLE"

security-audit:
  stage: analyze
  script:
    - curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
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
        run: curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh

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
        run: curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh

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

## Automated Versioning

Wave uses conventional commit messages to automate semantic versioning. On every push to `main`, the release workflow:

1. Analyzes commit messages since the last `v*` tag
2. Determines the version bump type
3. Creates and pushes a new tag
4. GoReleaser builds binaries and publishes a GitHub Release

### Bump Rules

| Commit prefix | Bump | Example |
|---------------|------|---------|
| `fix:`, `docs:`, `refactor:`, `test:`, `chore:` | **patch** (0.0.X) | v0.1.0 → v0.1.1 |
| `feat:` | **minor** (0.X.0) | v0.1.1 → v0.2.0 |
| `BREAKING CHANGE:` or `!:` (e.g. `feat!:`) | **major** (X.0.0) | v0.2.0 → v1.0.0 |

When multiple commits are present, the highest bump wins. For example, if a merge contains both `fix:` and `feat:` commits, the version gets a minor bump.

### Release Pipeline

The release workflow (`.github/workflows/release.yml`) has three jobs:

- **validate** — runs on all branches and PRs: `go test -race ./...`, goreleaser config check, snapshot build
- **auto-tag** — runs only on `main` after validate passes: determines bump, creates and pushes tag
- **release** — triggered by the new `v*` tag: GoReleaser builds binaries, creates GitHub Release with archives, `.deb` package, and Homebrew cask

### Manual Override

To skip the auto-tag and create a specific version:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Next Steps

- [Enterprise Patterns](/guides/enterprise) - Scale Wave to your organization
- [Audit Logging](/guides/audit-logging) - Track all pipeline activity
