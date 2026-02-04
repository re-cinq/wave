# GitHub Actions Integration

Run Wave pipelines in GitHub Actions for automated code review, security audits, documentation generation, and more.

## Quick Start

Add this workflow to `.github/workflows/wave.yml`:

```yaml
name: Wave Pipeline
on: [push, pull_request]

jobs:
  wave:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install Wave
        run: curl -fsSL https://wave.dev/install.sh | sh
      - name: Run Pipeline
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: wave run code-review
```

## Complete Examples

### Code Review on Pull Requests

```yaml
name: Code Review
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  review:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for diff analysis

      - name: Install Wave
        run: curl -fsSL https://wave.dev/install.sh | sh

      - name: Install Claude Code
        run: npm install -g @anthropic-ai/claude-code

      - name: Run Code Review
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          wave run code-review --input "Review changes in PR #${{ github.event.pull_request.number }}"

      - name: Upload Review Artifacts
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: wave-review-${{ github.run_id }}
          path: .wave/workspaces/
          retention-days: 7
```

### Security Audit Pipeline

```yaml
name: Security Audit
on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly on Sunday
  workflow_dispatch:  # Manual trigger

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Wave
        run: curl -fsSL https://wave.dev/install.sh | sh

      - name: Install Claude Code
        run: npm install -g @anthropic-ai/claude-code

      - name: Run Security Audit
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: wave run security-audit --timeout 60

      - name: Upload Audit Report
        uses: actions/upload-artifact@v4
        with:
          name: security-audit-${{ github.run_id }}
          path: |
            .wave/workspaces/**/output/*.json
            .wave/workspaces/**/output/*.md
          retention-days: 30
```

### Documentation Generation

```yaml
name: Generate Docs
on:
  push:
    branches: [main]
    paths:
      - 'src/**'
      - 'lib/**'

jobs:
  docs:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4

      - name: Install Wave
        run: curl -fsSL https://wave.dev/install.sh | sh

      - name: Install Claude Code
        run: npm install -g @anthropic-ai/claude-code

      - name: Generate Documentation
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: wave run generate-docs

      - name: Commit Documentation
        run: |
          git config --local user.email "action@github.com"
          git config --local user.name "GitHub Action"
          git add docs/
          git diff --staged --quiet || git commit -m "docs: auto-generated documentation"
          git push
```

### Matrix Strategy for Multiple Pipelines

```yaml
name: Wave Matrix
on: [push]

jobs:
  wave:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        pipeline: [code-review, lint-check, test-generation]
      fail-fast: false
    steps:
      - uses: actions/checkout@v4

      - name: Install Wave
        run: curl -fsSL https://wave.dev/install.sh | sh

      - name: Install Claude Code
        run: npm install -g @anthropic-ai/claude-code

      - name: Run ${{ matrix.pipeline }}
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: wave run ${{ matrix.pipeline }}
```

## Configuration

### Setting Up Secrets

1. Go to your repository Settings
2. Navigate to Secrets and variables > Actions
3. Click "New repository secret"
4. Add `ANTHROPIC_API_KEY` with your API key

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude | Yes (for Claude adapter) |
| `OPENAI_API_KEY` | OpenAI API key | Yes (for OpenAI adapter) |
| `WAVE_DEBUG` | Enable debug logging | No |
| `WAVE_WORKSPACE_ROOT` | Custom workspace directory | No |

### Workflow Permissions

Configure permissions based on your pipeline needs:

```yaml
permissions:
  contents: read        # Read repository files
  pull-requests: write  # Comment on PRs
  issues: write         # Create/update issues
  actions: read         # Read workflow status
```

## Artifact Caching

### Cache Wave Installation

```yaml
- name: Cache Wave
  uses: actions/cache@v4
  with:
    path: ~/.local/bin/wave
    key: wave-${{ runner.os }}-${{ hashFiles('wave.yaml') }}

- name: Install Wave
  if: steps.cache.outputs.cache-hit != 'true'
  run: curl -fsSL https://wave.dev/install.sh | sh
```

### Cache Adapter Installation

```yaml
- name: Cache Claude Code
  uses: actions/cache@v4
  with:
    path: ~/.npm
    key: npm-${{ runner.os }}-claude-code

- name: Install Claude Code
  run: npm install -g @anthropic-ai/claude-code
```

### Cache Pipeline Outputs

```yaml
- name: Cache Wave Workspaces
  uses: actions/cache@v4
  with:
    path: .wave/workspaces
    key: wave-workspaces-${{ github.sha }}
    restore-keys: |
      wave-workspaces-
```

## Troubleshooting

### API Key Configuration

**Problem**: `ANTHROPIC_API_KEY is not set`

**Solution**:
1. Verify secret is configured in repository settings
2. Check secret name matches exactly (case-sensitive)
3. Ensure workflow has correct syntax:

```yaml
env:
  ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

**Problem**: `Invalid API key`

**Solution**:
1. Regenerate your API key at [console.anthropic.com](https://console.anthropic.com)
2. Update the repository secret with the new key
3. Verify no leading/trailing whitespace in the secret value

### Permission Issues

**Problem**: `Permission denied: Write(src/main.go) blocked`

**Solution**:
1. Check the persona has write permissions:
   ```yaml
   personas:
     writer:
       permissions:
         allowed_tools: ["Write", "Edit"]
   ```
2. Use the correct persona for the operation
3. Review deny patterns in manifest

**Problem**: `Repository permissions insufficient`

**Solution**:
Add required permissions to workflow:
```yaml
permissions:
  contents: write
  pull-requests: write
```

### Timeout Handling

**Problem**: `context deadline exceeded` or workflow timeout

**Solution**:
1. Increase Wave timeout:
   ```bash
   wave run pipeline --timeout 60
   ```

2. Increase GitHub Actions timeout:
   ```yaml
   jobs:
     wave:
       timeout-minutes: 60
   ```

3. Break complex pipelines into smaller steps:
   ```yaml
   - name: Step 1
     run: wave run analyze --stop-after analyze
   - name: Step 2
     run: wave run implement --resume
   ```

**Problem**: Rate limiting from LLM provider

**Solution**:
1. Add delays between API calls in wave.yaml:
   ```yaml
   runtime:
     rate_limit_delay_ms: 1000
   ```
2. Reduce concurrent workers:
   ```yaml
   runtime:
     max_concurrent_workers: 2
   ```

### Artifact Issues

**Problem**: Artifacts not uploading

**Solution**:
1. Use `if: always()` to upload on failure:
   ```yaml
   - uses: actions/upload-artifact@v4
     if: always()
     with:
       name: wave-output
       path: .wave/workspaces/
   ```

2. Check path pattern matches output location:
   ```yaml
   path: |
     .wave/workspaces/**/output/*.json
     .wave/workspaces/**/output/*.md
   ```

### Binary Not Found

**Problem**: `adapter binary 'claude' not found on PATH`

**Solution**:
1. Install the adapter before running Wave:
   ```yaml
   - name: Install Claude Code
     run: npm install -g @anthropic-ai/claude-code
   ```

2. Verify installation:
   ```yaml
   - name: Verify Claude
     run: which claude && claude --version
   ```

3. Add to PATH if installed in custom location:
   ```yaml
   - name: Add to PATH
     run: echo "$HOME/.local/bin" >> $GITHUB_PATH
   ```

## Advanced Patterns

### Conditional Pipeline Execution

```yaml
- name: Run Review on Changed Files
  if: github.event_name == 'pull_request'
  run: |
    CHANGED_FILES=$(git diff --name-only origin/main...HEAD | tr '\n' ' ')
    wave run code-review --input "Review these files: $CHANGED_FILES"
```

### Parallel Jobs with Dependencies

```yaml
jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: curl -fsSL https://wave.dev/install.sh | sh
      - run: wave run analyze
      - uses: actions/upload-artifact@v4
        with:
          name: analysis
          path: .wave/workspaces/**/output/

  implement:
    needs: analyze
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/download-artifact@v4
        with:
          name: analysis
      - run: curl -fsSL https://wave.dev/install.sh | sh
      - run: wave run implement
```

### Self-Hosted Runners

For better performance and cost control:

```yaml
jobs:
  wave:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4
      - name: Run Pipeline
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: wave run code-review
```

## See Also

- [Integrations Overview](./index.md)
- [GitLab CI Guide](./gitlab-ci.md)
- [Error Codes Reference](/reference/error-codes)
- [Troubleshooting Reference](/reference/troubleshooting)
