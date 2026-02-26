# GitLab CI/CD Integration

Run Wave pipelines in GitLab CI/CD for automated code review, security audits, documentation generation, and more.

## Quick Start

Add this to `.gitlab-ci.yml`:

```yaml
wave-pipeline:
  image: ubuntu:latest
  script:
    - curl -fsSL https://wave.dev/install.sh | sh
    - wave run gh-pr-review
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
```

## Complete Examples

### Code Review on Merge Requests

```yaml
stages:
  - review

gh-pr-review:
  stage: review
  image: node:20
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  before_script:
    - curl -fsSL https://wave.dev/install.sh | sh
    - npm install -g @anthropic-ai/claude-code
  script:
    - wave run gh-pr-review --input "Review changes in MR !$CI_MERGE_REQUEST_IID"
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
  artifacts:
    paths:
      - .wave/workspaces/
    expire_in: 1 week
    when: always
```

### Security Audit Pipeline

```yaml
stages:
  - audit

security-audit:
  stage: audit
  image: node:20
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"
    - if: $CI_PIPELINE_SOURCE == "web"
  before_script:
    - curl -fsSL https://wave.dev/install.sh | sh
    - npm install -g @anthropic-ai/claude-code
  script:
    - wave run security-audit --timeout 60
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
  artifacts:
    paths:
      - .wave/workspaces/**/output/*.json
      - .wave/workspaces/**/output/*.md
    expire_in: 30 days
```

### Documentation Generation

```yaml
stages:
  - docs

generate-docs:
  stage: docs
  image: node:20
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
      changes:
        - src/**/*
        - lib/**/*
  before_script:
    - curl -fsSL https://wave.dev/install.sh | sh
    - npm install -g @anthropic-ai/claude-code
    - git config --global user.email "ci@gitlab.com"
    - git config --global user.name "GitLab CI"
  script:
    - wave run generate-docs
    - git add docs/
    - git diff --staged --quiet || git commit -m "docs: auto-generated documentation"
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
```

### Multi-Pipeline Matrix

```yaml
stages:
  - wave

.wave-template:
  image: node:20
  before_script:
    - curl -fsSL https://wave.dev/install.sh | sh
    - npm install -g @anthropic-ai/claude-code
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
  artifacts:
    paths:
      - .wave/workspaces/
    expire_in: 1 week

gh-pr-review:
  extends: .wave-template
  stage: wave
  script:
    - wave run gh-pr-review

lint-check:
  extends: .wave-template
  stage: wave
  script:
    - wave run lint-check

test-generation:
  extends: .wave-template
  stage: wave
  script:
    - wave run test-generation
```

### Parent-Child Pipeline

```yaml
# .gitlab-ci.yml
stages:
  - trigger

trigger-wave:
  stage: trigger
  trigger:
    include: .gitlab/ci/wave.yml
    strategy: depend
```

```yaml
# .gitlab/ci/wave.yml
stages:
  - analyze
  - implement
  - verify

analyze:
  stage: analyze
  image: node:20
  before_script:
    - curl -fsSL https://wave.dev/install.sh | sh
    - npm install -g @anthropic-ai/claude-code
  script:
    - wave run analyze
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
  artifacts:
    paths:
      - .wave/workspaces/
    expire_in: 1 day

implement:
  stage: implement
  image: node:20
  needs: [analyze]
  before_script:
    - curl -fsSL https://wave.dev/install.sh | sh
    - npm install -g @anthropic-ai/claude-code
  script:
    - wave run implement --resume
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
  artifacts:
    paths:
      - .wave/workspaces/
    expire_in: 1 day

verify:
  stage: verify
  image: node:20
  needs: [implement]
  before_script:
    - curl -fsSL https://wave.dev/install.sh | sh
    - npm install -g @anthropic-ai/claude-code
  script:
    - wave run verify
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
```

## Configuration

### Setting Up CI/CD Variables

1. Go to your project Settings
2. Navigate to CI/CD > Variables
3. Click "Add variable"
4. Add `ANTHROPIC_API_KEY`:
   - **Key**: `ANTHROPIC_API_KEY`
   - **Value**: Your API key
   - **Type**: Variable
   - **Flags**: Mask variable, Protect variable (recommended)

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude | Yes (for Claude adapter) |
| `OPENAI_API_KEY` | OpenAI API key | Yes (for OpenAI adapter) |

### Protected and Masked Variables

For security, configure variables as:

- **Masked**: Prevents value from appearing in job logs
- **Protected**: Only available to protected branches/tags

```yaml
variables:
  ANTHROPIC_API_KEY:
    value: $ANTHROPIC_API_KEY
    masked: true
    protected: true
```

## Artifact Caching

### Cache Wave Installation

```yaml
.wave-cache:
  cache:
    key: wave-${CI_COMMIT_REF_SLUG}
    paths:
      - .wave-cache/
    policy: pull-push
  before_script:
    - |
      if [ ! -f .wave-cache/wave ]; then
        mkdir -p .wave-cache
        curl -fsSL https://wave.dev/install.sh | sh
        cp ~/.local/bin/wave .wave-cache/
      else
        cp .wave-cache/wave ~/.local/bin/
      fi
    - export PATH="$HOME/.local/bin:$PATH"
```

### Cache Node Modules

```yaml
.node-cache:
  cache:
    key: node-${CI_COMMIT_REF_SLUG}
    paths:
      - node_modules/
      - .npm/
    policy: pull-push
  before_script:
    - npm config set cache .npm
    - npm install -g @anthropic-ai/claude-code
```

### Cache Pipeline Outputs

```yaml
wave-job:
  artifacts:
    paths:
      - .wave/workspaces/
    expire_in: 1 week
  cache:
    key: wave-workspaces-${CI_COMMIT_SHA}
    paths:
      - .wave/workspaces/
```

## Troubleshooting

### API Key Configuration

**Problem**: `ANTHROPIC_API_KEY is not set`

**Solution**:
1. Verify variable is configured in CI/CD settings
2. Check variable name matches exactly (case-sensitive)
3. Ensure variable is not protected if running on unprotected branch
4. Verify job has access to the variable:
   ```yaml
   variables:
     ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
   ```

**Problem**: `Invalid API key`

**Solution**:
1. Regenerate your API key at [console.anthropic.com](https://console.anthropic.com)
2. Update the CI/CD variable with the new key
3. Verify no leading/trailing whitespace in the variable value
4. Check the variable is not expired or revoked

### Permission Issues

**Problem**: `Permission denied: Write(src/main.go) blocked`

**Solution**:
1. Check the persona has write permissions in `wave.yaml`:
   ```yaml
   personas:
     writer:
       permissions:
         allowed_tools: ["Write", "Edit"]
   ```
2. Use the correct persona for the operation
3. Review deny patterns in manifest

**Problem**: `Cannot push to protected branch`

**Solution**:
1. Use a deploy token with write access
2. Configure git credentials:
   ```yaml
   before_script:
     - git config --global user.email "ci@gitlab.com"
     - git config --global user.name "GitLab CI"
     - git remote set-url origin https://oauth2:${GITLAB_TOKEN}@gitlab.com/${CI_PROJECT_PATH}.git
   ```

### Timeout Handling

**Problem**: `context deadline exceeded` or job timeout

**Solution**:
1. Increase Wave timeout:
   ```bash
   wave run pipeline --timeout 60
   ```

2. Increase GitLab job timeout:
   ```yaml
   wave-job:
     timeout: 1 hour
   ```

3. Break complex pipelines into smaller jobs:
   ```yaml
   analyze:
     script:
       - wave run pipeline --stop-after analyze

   implement:
     needs: [analyze]
     script:
       - wave run pipeline --resume
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
3. Use GitLab resource groups to limit parallel jobs:
   ```yaml
   wave-job:
     resource_group: wave-api
   ```

### Artifact Issues

**Problem**: Artifacts not saved

**Solution**:
1. Use `when: always` to save on failure:
   ```yaml
   artifacts:
     paths:
       - .wave/workspaces/
     when: always
   ```

2. Check path pattern matches output location:
   ```yaml
   artifacts:
     paths:
       - .wave/workspaces/**/output/*.json
       - .wave/workspaces/**/output/*.md
   ```

3. Verify artifact size is within limits:
   ```yaml
   artifacts:
     paths:
       - .wave/workspaces/
     expire_in: 1 week
   ```

### Binary Not Found

**Problem**: `adapter binary 'claude' not found on PATH`

**Solution**:
1. Install the adapter in `before_script`:
   ```yaml
   before_script:
     - npm install -g @anthropic-ai/claude-code
   ```

2. Verify installation:
   ```yaml
   script:
     - which claude && claude --version
     - wave run gh-pr-review
   ```

3. Use a custom image with pre-installed tools:
   ```yaml
   image: your-registry/wave-runner:latest
   ```

### Docker-in-Docker Issues

**Problem**: Wave needs Docker but GitLab runner doesn't support it

**Solution**:
Use shell executor or enable Docker-in-Docker:
```yaml
wave-job:
  image: docker:latest
  services:
    - docker:dind
  variables:
    DOCKER_HOST: tcp://docker:2375
    DOCKER_TLS_CERTDIR: ""
```

## Advanced Patterns

### Conditional Pipeline Execution

```yaml
gh-pr-review:
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
      when: always
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
      when: never
    - when: manual
  script:
    - |
      CHANGED_FILES=$(git diff --name-only origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME...HEAD | tr '\n' ' ')
      wave run gh-pr-review --input "Review these files: $CHANGED_FILES"
```

### Using GitLab Services

```yaml
wave-with-db:
  services:
    - postgres:15
  variables:
    POSTGRES_DB: test
    POSTGRES_USER: test
    POSTGRES_PASSWORD: test
    DATABASE_URL: postgres://test:test@postgres:5432/test
  script:
    - wave run integration-test
```

### Auto DevOps Integration

```yaml
include:
  - template: Auto-DevOps.gitlab-ci.yml

wave-review:
  stage: review
  extends: .wave-template
  script:
    - wave run gh-pr-review
  rules:
    - if: $CI_MERGE_REQUEST_ID
```

### Self-Managed Runners

For better performance and security:

```yaml
wave-job:
  tags:
    - wave-runner
    - self-hosted
  script:
    - wave run gh-pr-review
```

## See Also

- [CI/CD Integration](/guides/ci-cd)
- [GitHub Actions Guide](./github-actions.md)
- [Error Codes Reference](/reference/error-codes)
- [Troubleshooting Reference](/reference/troubleshooting)
