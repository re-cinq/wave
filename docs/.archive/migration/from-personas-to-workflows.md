# From Ad-hoc AI to Pipelines

This guide helps you transition from ad-hoc AI interactions to structured Wave pipelines.

## The Transition

### Before: Ad-hoc AI Usage

```
Developer: "Review this code for security issues"
AI: [Reviews code]
Developer: "Now check the test coverage"
AI: [Checks tests]
Developer: "Generate a summary"
AI: [Creates summary]
```

Problems:
- No reproducibility
- Context lost between sessions
- Manual orchestration
- No quality guarantees

### After: Wave Pipelines

```yaml
kind: WavePipeline
metadata:
  name: gh-pr-review

steps:
  - id: security
    persona: auditor
    exec:
      type: prompt
      source: "Review for security issues: {{ input }}"
    output_artifacts:
      - name: security-report
        path: .wave/output/security.md
        type: markdown

  - id: coverage
    persona: navigator
    dependencies: [security]
    exec:
      type: prompt
      source: "Check test coverage"

  - id: summary
    persona: summarizer
    dependencies: [security, coverage]
    memory:
      inject_artifacts:
        - step: security
          artifact: security-report
          as: security
    exec:
      type: prompt
      source: "Summarize: {{ artifacts.security }}"
```

Benefits:
- Reproducible results
- Explicit data flow
- Version controlled
- Contract validation

## Migration Steps

### 1. Document Your Current Process

What AI tasks do you perform regularly?

```
Task: Code Review
Steps:
1. Analyze changes (read-only)
2. Check security (specialized knowledge)
3. Verify tests (run commands)
4. Generate summary
```

### 2. Map to Wave Concepts

| Your Process | Wave Concept |
|--------------|--------------|
| Separate tasks | Steps |
| Different modes (read-only, write) | Personas |
| Output requirements | Contracts |
| Data passing | Artifacts |

### 3. Create Personas

Define personas in `wave.yaml`:

```yaml
personas:
  navigator:
    adapter: claude
    description: "Read-only codebase analysis"
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
      deny: []
```

### 4. Create Your Pipeline

```yaml
kind: WavePipeline
metadata:
  name: gh-pr-review

input:
  source: cli

steps:
  - id: analyze
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json

  - id: review
    persona: auditor
    dependencies: [analyze]
    memory:
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Review based on: {{ artifacts.context }}"
```

### 5. Add Contracts

Ensure output quality:

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/analysis.schema.json
    on_failure: retry
    max_retries: 2
```

### 6. Test and Iterate

```bash
# Validate configuration
wave validate

# Test with real input
wave run gh-pr-review "Review auth changes"

# Check output
cat .wave/workspaces/*/.wave/output/
```

## Common Migrations

### Single-Shot Prompt → Single-Step Pipeline

**Before:**
```
"Analyze this code for performance issues"
```

**After:**
```yaml
kind: WavePipeline
metadata:
  name: perf-analysis

steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze for performance issues: {{ input }}"
    output_artifacts:
      - name: report
        path: .wave/output/performance.md
        type: markdown
```

### Multi-Turn Conversation → Multi-Step Pipeline

**Before:**
```
Turn 1: "Analyze the codebase"
Turn 2: "Based on that, suggest improvements"
Turn 3: "Create implementation plan"
```

**After:**
```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json

  - id: suggest
    persona: philosopher
    dependencies: [analyze]
    memory:
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Suggest improvements for: {{ artifacts.context }}"
    output_artifacts:
      - name: suggestions
        path: .wave/output/suggestions.json
        type: json

  - id: plan
    persona: planner
    dependencies: [suggest]
    memory:
      inject_artifacts:
        - step: suggest
          artifact: suggestions
          as: ideas
    exec:
      type: prompt
      source: "Create plan for: {{ artifacts.ideas }}"
```

## Key Differences

| Ad-hoc | Pipeline |
|--------|----------|
| Implicit context | Explicit artifacts |
| Manual ordering | Declared dependencies |
| No validation | Contract enforcement |
| Lost on close | Version controlled |
| One developer | Team shareable |

## Next Steps

- [Creating Pipelines](/workflows/creating-workflows) - Full pipeline guide
- [Contracts](/paradigm/deliverables-contracts) - Output validation
- [Team Adoption](/migration/team-adoption) - Roll out to your team
