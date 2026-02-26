# Team Adoption

Rolling out Wave to your team is straightforward: it's configuration files in git. This guide covers practical steps for team adoption.

## Getting Started

### 1. Initialize Wave

```bash
cd your-project
wave init
```

This creates:
```
.wave/
├── wave.yaml          # Manifest with adapters and personas
├── pipelines/         # Pipeline definitions
├── personas/          # System prompts
└── contracts/         # Output schemas
```

### 2. Configure Your Manifest

Edit `wave.yaml` with your team's personas:

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: your-project

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
      deny: [Write, Edit]

  craftsman:
    adapter: claude
    description: "Code implementation"
    system_prompt_file: .wave/personas/craftsman.md
    permissions:
      allowed_tools: [Read, Write, Edit, Bash]

runtime:
  workspace_root: .wave/workspaces
  audit:
    log_all_tool_calls: true
    log_dir: .wave/traces/
```

### 3. Create Standard Pipelines

Start with pipelines that solve real team problems:

```yaml
# .wave/pipelines/gh-pr-review.yaml
kind: WavePipeline
metadata:
  name: gh-pr-review
  description: "Team code review"

input:
  source: cli

steps:
  - id: analyze
    persona: navigator
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze changes: {{ input }}

        Check against team standards.
    output_artifacts:
      - name: review
        path: .wave/output/review.md
        type: markdown
```

### 4. Commit and Share

```bash
git add .wave/
git commit -m "Add Wave configuration"
git push
```

Teammates get the configuration with `git pull`.

## Rollout Strategy

### Phase 1: Pilot

1. One developer creates initial configuration
2. Test with real tasks
3. Iterate on personas and pipelines
4. Document what works

### Phase 2: Team Trial

1. Share configuration via git
2. Team tries pipelines on actual work
3. Collect feedback
4. Refine based on usage

### Phase 3: Standard Practice

1. Add pipelines to development workflow
2. Integrate with CI if useful
3. Maintain and improve over time

## Common Pipelines for Teams

### Code Review

```yaml
kind: WavePipeline
metadata:
  name: gh-pr-review

steps:
  - id: review
    persona: auditor
    exec:
      type: prompt
      source: "Review: {{ input }}"
    output_artifacts:
      - name: review
        path: .wave/output/review.md
```

### Documentation

```yaml
kind: WavePipeline
metadata:
  name: docs

steps:
  - id: generate
    persona: documenter
    workspace:
      mount:
        - source: ./src
          target: /code
          mode: readonly
    exec:
      type: prompt
      source: "Document: {{ input }}"
    output_artifacts:
      - name: docs
        path: .wave/output/docs.md
```

## Tips

### Keep It Simple

Start with single-step pipelines. Add complexity only when needed.

### Use Contracts

Add contracts for structured outputs:

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/output.schema.json
```

### Document Your Pipelines

Add descriptions:

```yaml
metadata:
  name: gh-pr-review
  description: |
    Usage: wave run gh-pr-review "description of changes"
    Output: .wave/output/review.md
```

### Version Control Everything

The `.wave/` directory is code. Review changes in PRs.

## Troubleshooting

### "Pipeline not found"

Check that the file exists and `metadata.name` matches:

```bash
ls .wave/pipelines/
grep "name:" .wave/pipelines/gh-pr-review.yaml
```

### "Persona not defined"

Ensure the persona is in `wave.yaml`:

```yaml
personas:
  navigator:  # This must exist
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
```

### API Key Issues

Set the environment variable:

```bash
export ANTHROPIC_API_KEY=your-key
```

## Next Steps

- [Creating Pipelines](/workflows/creating-workflows) - Pipeline guide
- [Enterprise Patterns](/migration/enterprise-patterns) - Larger organizations
