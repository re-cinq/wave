# Team Adoption

Share Wave pipelines across your team through git. Configuration files live alongside your code, so teammates get pipelines automatically with `git pull`.

## Prerequisites

- Wave installed locally
- Git repository for your project
- API key set in environment

## Step-by-Step

### 1. Initialize Wave

```bash
cd your-project
wave init
```

This creates the `.wave/` directory:

```
.wave/
├── wave.yaml          # Manifest with adapters and personas
├── pipelines/         # Pipeline definitions
├── personas/          # System prompts
└── contracts/         # Output schemas
```

### 2. Create Your First Pipeline

```yaml
# .wave/pipelines/code-review.yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Review changes against team standards"

input:
  source: cli

steps:
  - id: review
    persona: navigator
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Review the changes: {{ input }}
        Check for team coding standards.
    output_artifacts:
      - name: review
        path: output/review.md
        type: markdown
```

### 3. Test Locally

```bash
wave run code-review "Add user authentication"
```

### 4. Commit and Share

```bash
git add .wave/
git commit -m "Add Wave configuration"
git push
```

Teammates now get the same pipelines:

```bash
git pull
wave run code-review "Fix login bug"
```

## Rollout Strategy

### Phase 1: Pilot (1 Week)

One developer creates and tests pipelines:

1. Initialize Wave in the repository
2. Create pipelines for common tasks
3. Test with real work
4. Document what works

### Phase 2: Team Trial (2-4 Weeks)

Share with the team for feedback:

1. Team members pull the configuration
2. Everyone tries pipelines on actual work
3. Collect feedback in standup
4. Iterate on prompts and contracts

### Phase 3: Standard Practice

Integrate into workflow:

1. Add pipelines to PR checklist
2. Consider CI/CD integration
3. Review pipeline changes in PRs
4. Maintain and improve over time

## Common Team Pipelines

### Code Review

```yaml
kind: WavePipeline
metadata:
  name: code-review

steps:
  - id: review
    persona: navigator
    exec:
      type: prompt
      source: "Review these changes: {{ input }}"
    output_artifacts:
      - name: review
        path: output/review.md
```

Run: `wave run code-review "Add caching layer"`

### Documentation

```yaml
kind: WavePipeline
metadata:
  name: docs

steps:
  - id: generate
    persona: navigator
    workspace:
      mount:
        - source: ./src
          target: /code
          mode: readonly
    exec:
      type: prompt
      source: "Generate documentation for: {{ input }}"
    output_artifacts:
      - name: docs
        path: output/docs.md
```

Run: `wave run docs "the auth module"`

### Test Generation

```yaml
kind: WavePipeline
metadata:
  name: generate-tests

steps:
  - id: tests
    persona: navigator
    workspace:
      mount:
        - source: ./src
          target: /code
          mode: readonly
    exec:
      type: prompt
      source: "Generate tests for: {{ input }}"
    output_artifacts:
      - name: tests
        path: output/tests.go
```

Run: `wave run generate-tests "user service"`

## Complete Example

A team-ready manifest with multiple personas:

```yaml
# wave.yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project

adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json

personas:
  navigator:
    adapter: claude
    description: "Read-only analysis"
    system_prompt_file: .wave/personas/navigator.md
    permissions:
      allowed_tools: [Read, Glob, Grep]
      deny: [Write, Edit, Bash]

  auditor:
    adapter: claude
    description: "Security review"
    system_prompt_file: .wave/personas/auditor.md
    permissions:
      allowed_tools: [Read, Grep]
      deny: [Write, Edit, Bash]

runtime:
  workspace_root: .wave/workspaces
  audit:
    log_all_tool_calls: true
    log_dir: .wave/traces/
```

## Tips

### Start Simple

Single-step pipelines are easier to maintain. Add complexity only when needed.

### Add Descriptions

Help teammates understand pipelines:

```yaml
metadata:
  name: code-review
  description: |
    Usage: wave run code-review "description"
    Output: output/review.md
```

### Use Contracts for Structured Output

When you need consistent output format:

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/review.schema.json
```

### Review Pipeline Changes

Treat `.wave/` as code. Review changes in PRs.

## Troubleshooting

### "Pipeline not found"

Check the file exists and name matches:

```bash
ls .wave/pipelines/
grep "name:" .wave/pipelines/code-review.yaml
```

### "Persona not defined"

Add the persona to `wave.yaml`:

```yaml
personas:
  navigator:  # Must exist
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
```

### API Key Issues

Set the environment variable:

```bash
export ANTHROPIC_API_KEY=your-key
```

Each team member needs their own key.

## Next Steps

- [CI/CD Integration](/guides/ci-cd) - Automate pipelines in your build
- [Enterprise Patterns](/guides/enterprise) - Scale to larger organizations
- [Contracts](/concepts/contracts) - Validate pipeline outputs
