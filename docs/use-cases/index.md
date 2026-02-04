# Use Cases

Find the right pipeline for your task. Each use case includes a complete, runnable pipeline you can copy and use immediately.

## Development Workflows

| Use Case | Description | Pipeline |
|----------|-------------|----------|
| [Code Review](/use-cases/code-review) | Automated PR review with security checks and quality analysis | `code-review` |
| [Security Audit](/use-cases/security-audit) | Vulnerability scanning, dependency checks, compliance verification | `code-review` (or custom) |
| [Documentation](/use-cases/docs-generation) | Generate API docs, README files, and usage guides from code | `docs` |
| [Test Generation](/use-cases/test-generation) | Analyze coverage gaps and generate comprehensive tests | `test-gen` |

## Quick Start

Run any built-in pipeline immediately:

```bash
cd your-project
wave init
wave run code-review "review the authentication module"
```

Expected output:

```
[10:00:01] started   diff-analysis     (navigator)              Starting step
[10:00:25] completed diff-analysis     (navigator)   24s   2.5k Analysis complete
[10:00:26] started   security-review   (auditor)                Starting step
[10:00:26] started   quality-review    (auditor)                Starting step
[10:00:45] completed security-review   (auditor)     19s   1.8k Review complete
[10:00:48] completed quality-review    (auditor)     22s   2.1k Review complete
[10:00:49] started   summary           (summarizer)             Starting step
[10:01:05] completed summary           (summarizer)  16s   1.2k Summary complete

Pipeline code-review completed in 64s
Artifacts: output/review-summary.md
```

## Pipeline Structure

Every use-case pipeline follows the same pattern:

```yaml
kind: WavePipeline
metadata:
  name: pipeline-name
  description: "What this pipeline does"

steps:
  - id: analyze
    persona: navigator
    exec:
      source: "Analyze the codebase for: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json

  - id: execute
    persona: craftsman
    dependencies: [analyze]
    exec:
      source: "Implement based on analysis"
    output_artifacts:
      - name: result
        path: output/result.md
        type: markdown
```

## Create Custom Pipelines

Need something specific? Start with an ad-hoc task:

```bash
# Quick task without a pipeline file
wave do "refactor the database connection handling"

# Save the generated pipeline for reuse
wave do "refactor the database connection handling" --save .wave/pipelines/db-refactor.yaml
```

## Next Steps

- [Quickstart](/quickstart) - Get Wave running in 60 seconds
- [Concepts: Pipelines](/concepts/pipelines) - Understand pipeline structure in depth
- [CLI Reference](/reference/cli) - Complete command documentation
