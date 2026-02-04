---
layout: home
hero:
  name: Wave
  text: AI Pipelines as Code
  tagline: Define multi-step AI workflows in YAML. Run them with validation, isolation, and reproducible results.
  actions:
    - theme: brand
      text: Get Started in 60 Seconds
      link: /quickstart
    - theme: alt
      text: Use Cases
      link: /use-cases/
    - theme: alt
      text: GitHub
      link: https://github.com/recinq/wave
features:
  - icon: ">"
    title: Pipelines as Code
    details: Define multi-step AI workflows in YAML. Version control them, share them, run them anywhere.
    link: /concepts/pipelines
  - icon: "#"
    title: Contract Validation
    details: Every step validates its output against schemas. Get structured, predictable results every time.
    link: /concepts/contracts
  - icon: "~"
    title: Step Isolation
    details: Each step runs with fresh memory in an ephemeral workspace. No context bleed between steps.
    link: /concepts/workspaces
  - icon: "@"
    title: Ready-to-Run Pipelines
    details: Built-in pipelines for code review, security audits, documentation, and test generation.
    link: /use-cases/
---

## What is Wave?

Wave is a pipeline orchestrator that runs AI workflows defined in YAML files. You define a sequence of steps, each executed by an AI agent with specific permissions. Wave handles isolation between steps, validates outputs against schemas, and passes artifacts through the pipeline. The result: repeatable, auditable AI automation that you can version control and share.

```
                        wave.yaml
                            |
                            v
                    +---------------+
                    |  Wave Engine  |
                    +---------------+
                            |
        +-------------------+-------------------+
        |                   |                   |
        v                   v                   v
   +--------+          +--------+          +--------+
   | Step 1 |    ->    | Step 2 |    ->    | Step 3 |
   | analyze|          | review |          | report |
   +--------+          +--------+          +--------+
        |                   |                   |
        v                   v                   v
    artifact            artifact            artifact
    (JSON)             (markdown)           (summary)
```

Each step runs in complete isolation with fresh memory. Artifacts flow between steps automatically. Contracts validate outputs before the next step begins.

## Your First Pipeline

```bash
# Install and initialize
curl -L https://github.com/recinq/wave/releases/latest/download/wave-linux-amd64 -o wave
chmod +x wave && sudo mv wave /usr/local/bin/
cd your-project && wave init

# Run your first pipeline
wave run --pipeline hello-world --input "testing Wave"
```

## Example: Code Review Pipeline

```yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Automated code review"

steps:
  - id: analyze
    persona: navigator
    exec:
      source: "Analyze the code changes: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json

  - id: review
    persona: auditor
    dependencies: [analyze]
    exec:
      source: "Review for security and quality issues"
    output_artifacts:
      - name: review
        path: output/review.md
        type: markdown
```

```bash
wave run --pipeline code-review --input "authentication module"
```

[Get started in 60 seconds](/quickstart) or explore [use cases](/use-cases/).
