# Sharing Pipelines

Wave pipelines are plain YAML files in your repository. Share them with your team the same way you share any code: through git.

## Git-Based Sharing

Pipelines live in your project's `.wave/` directory and are version controlled alongside your code.

### Project Structure

```
your-project/
├── .wave/
│   ├── wave.yaml              # Project manifest
│   ├── pipelines/
│   │   ├── gh-pr-review.yaml
│   │   ├── documentation.yaml
│   │   └── testing.yaml
│   ├── personas/
│   │   ├── navigator.md
│   │   ├── auditor.md
│   │   └── craftsman.md
│   └── contracts/
│       ├── analysis.schema.json
│       └── review.schema.json
├── src/
└── README.md
```

### Team Workflow

**Developer creates a pipeline:**
```bash
# Create new pipeline
vim .wave/pipelines/feature-development.yaml

# Test it
wave run feature-development "Add user authentication"

# Commit
git add .wave/pipelines/feature-development.yaml
git commit -m "Add feature development pipeline"
git push
```

**Teammate uses it:**
```bash
git pull
wave run feature-development "Add payment processing"
```

### Pull Request Review

Pipeline changes go through normal code review:

```bash
# Create feature branch
git checkout -b improve-gh-pr-review-pipeline

# Modify pipeline
vim .wave/pipelines/gh-pr-review.yaml

# Test changes
wave run gh-pr-review "Test the changes"

# Submit PR
git add .wave/
git commit -m "Add security scanning step to code review"
git push origin improve-gh-pr-review-pipeline
gh pr create
```

Reviewers see exactly what changed:

```diff
   - id: security-review
     persona: auditor
+    dependencies: [diff-analysis]
+    memory:
+      inject_artifacts:
+        - step: diff-analysis
+          artifact: diff
+          as: changes
```

## Reproducibility

Wave guarantees identical behavior when the same pipeline runs on different machines.

### How Reproducibility Works

1. **Configuration is complete**: All behavior defined in YAML
2. **Fresh memory**: Each step starts clean, no hidden state
3. **Explicit artifacts**: Data flow declared, not implicit
4. **Contract validation**: Outputs verified against schemas

### Same Config, Same Results

```bash
# Developer A
wave run gh-pr-review "Review auth changes"
# Output: output/review-summary.md

# Developer B (same repository state)
git checkout same-commit
wave run gh-pr-review "Review auth changes"
# Output: identical output/review-summary.md
```

### Version Control Benefits

```bash
# See pipeline evolution
git log --oneline .wave/pipelines/gh-pr-review.yaml

# Compare versions
git diff HEAD~5 .wave/pipelines/gh-pr-review.yaml

# Revert problematic changes
git revert <commit>
```

## Team Patterns

### Standard Pipelines

Create team-standard pipelines that encode your practices:

```yaml
# .wave/pipelines/pr-review.yaml
kind: WavePipeline
metadata:
  name: pr-review
  description: "Team standard PR review process"

steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: |
        Analyze PR according to team standards:
        - Check coding conventions
        - Verify test coverage
        - Review security implications
        {{ input }}
```

### Persona Libraries

Share persona configurations across projects:

```yaml
# .wave/personas/team-reviewer.md
You are a code reviewer following our team's standards:

## Coding Standards
- Follow project style guide
- Prefer immutability
- Handle errors explicitly

## Review Priorities
1. Security vulnerabilities
2. Performance issues
3. Test coverage
4. Code clarity
```

### Contract Standards

Define organization-wide output schemas:

```json
// .wave/contracts/review-finding.schema.json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["severity", "category", "description"],
  "properties": {
    "severity": {
      "enum": ["critical", "high", "medium", "low"]
    },
    "category": {
      "enum": ["security", "performance", "maintainability", "testing"]
    },
    "description": {
      "type": "string",
      "minLength": 10
    }
  }
}
```

## Cross-Project Sharing

### Git Submodules

Share pipelines across multiple projects:

```bash
# Create shared pipeline repository
mkdir wave-pipelines && cd wave-pipelines
git init

# Add to projects as submodule
cd your-project
git submodule add git@github.com:org/wave-pipelines.git .wave/shared
```

Reference shared pipelines:

```yaml
# In your project's wave.yaml
skill_mounts:
  - path: .wave/shared/pipelines/
```

### Copy and Customize

For project-specific needs, copy and modify:

```bash
# Copy from another project
cp ../other-project/.wave/pipelines/useful.yaml .wave/pipelines/

# Customize for this project
vim .wave/pipelines/useful.yaml

# Commit as your own
git add .wave/pipelines/useful.yaml
git commit -m "Add customized pipeline from other-project"
```

## CI/CD Integration

### GitHub Actions

Run pipelines in CI:

```yaml
# .github/workflows/wave-review.yml
name: AI Code Review
on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Wave
        run: |
          curl -L https://github.com/wave/releases/latest/download/wave-linux.tar.gz | tar xz
          sudo mv wave /usr/local/bin/

      - name: Run Code Review
        run: |
          wave run gh-pr-review "$(git diff origin/main..HEAD)"
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Upload Review
        uses: actions/upload-artifact@v3
        with:
          name: gh-pr-review
          path: .wave/workspaces/*/output/
```

### GitLab CI

```yaml
# .gitlab-ci.yml
ai-review:
  stage: review
  script:
    - curl -L https://wave.dev/install | sh
    - wave run gh-pr-review "$CI_MERGE_REQUEST_DIFF_BASE_SHA"
  artifacts:
    paths:
      - .wave/workspaces/*/output/
```

## Best Practices

### 1. Document Pipelines

Add descriptions and comments:

```yaml
kind: WavePipeline
metadata:
  name: gh-pr-review
  description: |
    Security-focused code review pipeline.

    Usage: wave run gh-pr-review "description of changes"

    Outputs:
    - Security findings (output/security.md)
    - Quality issues (output/quality.md)
    - Review summary (output/summary.md)
```

### 2. Version Personas Carefully

Changes to persona prompts affect all pipelines using them. Test thoroughly:

```bash
# Before changing a persona
wave run gh-pr-review "test input" > before.txt

# After changing
vim .wave/personas/auditor.md
wave run gh-pr-review "test input" > after.txt

# Compare
diff before.txt after.txt
```

### 3. Use Contracts for Stability

Contracts catch breaking changes:

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/output.schema.json
```

If pipeline output changes, contract validation will catch it.

### 4. Maintain Backwards Compatibility

When modifying shared pipelines:

- Add optional fields, don't remove required ones
- Keep existing step IDs
- Document breaking changes in commit messages

### 5. Organize by Purpose

```
.wave/pipelines/
├── review/
│   ├── gh-pr-review.yaml
│   └── security-review.yaml
├── development/
│   ├── feature.yaml
│   └── bugfix.yaml
└── documentation/
    ├── api-docs.yaml
    └── readme-update.yaml
```

## Troubleshooting

### Pipeline Not Found

```
Error: pipeline "gh-pr-review" not found
```

Check:
1. File exists: `ls .wave/pipelines/gh-pr-review.yaml`
2. Metadata name matches: `grep "name:" .wave/pipelines/gh-pr-review.yaml`

### Persona Not Found

```
Error: persona "navigator" not defined in manifest
```

Check wave.yaml:
```yaml
personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
```

### Contract Schema Not Found

```
Error: schema ".wave/contracts/output.schema.json" not found
```

Ensure the schema file exists and path is correct relative to project root.

## Next Steps

- [Creating Pipelines](/workflows/creating-workflows) - Pipeline structure guide
- [Pipeline Execution](/concepts/pipeline-execution) - How execution works
- [Contracts](/paradigm/deliverables-contracts) - Output validation
