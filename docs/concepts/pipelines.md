# Pipelines

A pipeline is a multi-step AI workflow where each step runs one persona in an isolated workspace. Pipelines enable complex AI workflows by breaking tasks into focused steps with clear boundaries.

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: code-review
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
```

</div>

Use pipelines when you need coordinated AI tasks that build on each other's outputs.

## Pipeline Structure

Every pipeline has three main sections:

| Section | Purpose |
|---------|---------|
| `metadata` | Name, description, and pipeline identity |
| `input` | How the pipeline receives its input |
| `steps` | The sequence of AI tasks to execute |

## Dependency Patterns

### Linear Dependencies

Steps execute in sequence when dependencies are specified:

<div v-pre>

```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze the codebase for: {{ input }}"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json

  - id: implement
    persona: craftsman
    dependencies: [analyze]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Implement based on the analysis."
```

</div>

### Parallel Execution (Fan-Out)

Steps without mutual dependencies run in parallel. This pattern is useful when you need multiple perspectives on the same input:

<div v-pre>

```yaml
steps:
  - id: navigate
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json

  - id: security
    persona: auditor
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Security review"
    output_artifacts:
      - name: findings
        path: .wave/output/security.md
        type: markdown

  - id: quality
    persona: auditor
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Quality review"
    output_artifacts:
      - name: findings
        path: .wave/output/quality.md
        type: markdown
```

</div>

In this example, `security` and `quality` run in parallel after `navigate` completes.

### Convergence (Fan-In)

Multiple parallel steps can feed into a single summary step:

```yaml
steps:
  # ... parallel steps above ...

  - id: summary
    persona: summarizer
    dependencies: [security, quality]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: security
          artifact: findings
          as: security_report
        - step: quality
          artifact: findings
          as: quality_report
    exec:
      type: prompt
      source: "Synthesize all findings into a final report"
```

### Dependency Visualization

The following diagram shows how dependencies create the execution flow:

```mermaid
flowchart TD
  navigate[navigate<br/><small>navigator</small>]
  security[security<br/><small>auditor</small>]
  quality[quality<br/><small>auditor</small>]
  summary[summary<br/><small>summarizer</small>]

  navigate --> security
  navigate --> quality
  security --> summary
  quality --> summary

  navigate -.->|"analysis"| security
  navigate -.->|"analysis"| quality
  security -.->|"findings"| summary
  quality -.->|"findings"| summary

  style navigate fill:#4a90d9,color:#fff
  style security fill:#d94a4a,color:#fff
  style quality fill:#d94a4a,color:#fff
  style summary fill:#9a4ad9,color:#fff
```

## Artifact Patterns

Artifacts are the primary mechanism for passing data between steps.

### Producing Artifacts

Declare what a step outputs:

```yaml
output_artifacts:
  - name: analysis        # Artifact identifier
    path: .wave/output/data.json  # Where the step writes it
    type: json             # Type hint for consumers
```

### Consuming Artifacts

Inject artifacts from previous steps:

```yaml
memory:
  strategy: fresh
  inject_artifacts:
    - step: analyze      # Source step
      artifact: analysis  # Artifact name
      as: context         # Mount name in workspace
```

The artifact appears at `.wave/artifacts/<as-name>` in the step's workspace.

### Artifact Types

| Type | Description | Best For |
|------|-------------|----------|
| `json` | Structured data | Analysis results, configs |
| `markdown` | Formatted text | Reports, documentation |
| `file` | Single file | Code, configs |
| `directory` | Folder | Multiple files, assets |

### Multi-Artifact Injection

A step can consume multiple artifacts:

```yaml
memory:
  strategy: fresh
  inject_artifacts:
    - step: analyze
      artifact: code_analysis
      as: code
    - step: security
      artifact: findings
      as: security
    - step: quality
      artifact: findings
      as: quality
```

All artifacts are available under `.wave/artifacts/`:
- `.wave/artifacts/code`
- `.wave/artifacts/security`
- `.wave/artifacts/quality`

## Memory Strategies

Control how context flows between steps:

| Strategy | Behavior | Use When |
|----------|----------|----------|
| `fresh` | Clean slate, only injected artifacts | Most cases (recommended) |
| `inherit` | Carry forward previous context | Continuation tasks |

Fresh memory is recommended to prevent context pollution and ensure reproducible results.

## Running Pipelines

Execute a pipeline with input:

```bash
wave run code-review "Review authentication changes"
```

Check pipeline status:

```bash
wave status code-review
```

View artifacts from a run:

```bash
wave artifacts <run-id>
```

## Complete Example

A production-ready code review pipeline:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Multi-perspective code review with security and quality checks"

input:
  source: cli

steps:
  - id: diff-analysis
    persona: navigator
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze the changes: {{ input }}
        Output as JSON with files, modules, and breaking changes.
    output_artifacts:
      - name: diff
        path: .wave/output/diff.json
        type: json

  - id: security-review
    persona: auditor
    dependencies: [diff-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: diff-analysis
          artifact: diff
          as: changes
    exec:
      type: prompt
      source: "Review .wave/artifacts/changes for security vulnerabilities"
    output_artifacts:
      - name: security
        path: .wave/output/security.md
        type: markdown

  - id: quality-review
    persona: auditor
    dependencies: [diff-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: diff-analysis
          artifact: diff
          as: changes
    exec:
      type: prompt
      source: "Review .wave/artifacts/changes for code quality issues"
    output_artifacts:
      - name: quality
        path: .wave/output/quality.md
        type: markdown

  - id: final-verdict
    persona: summarizer
    dependencies: [security-review, quality-review]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: security-review
          artifact: security
          as: security_findings
        - step: quality-review
          artifact: quality
          as: quality_findings
    exec:
      type: prompt
      source: |
        Synthesize findings into: APPROVE / REQUEST_CHANGES / NEEDS_DISCUSSION
    output_artifacts:
      - name: verdict
        path: .wave/output/verdict.md
        type: markdown
```

</div>

## Next Steps

- [Personas](/concepts/personas) - Configure the AI agents that run in each step
- [Contracts](/concepts/contracts) - Validate step outputs before handover
- [Artifacts](/concepts/artifacts) - Deep dive into artifact passing
- [Pipeline Configuration Guide](/guides/pipeline-configuration) - Step-by-step configuration guide
- [Pipeline Schema Reference](/reference/pipeline-schema) - Complete field reference
