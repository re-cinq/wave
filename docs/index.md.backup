---
layout: home
hero:
  name: Wave
  text: Infrastructure as Code for AI
  tagline: Define reproducible AI workflows with declarative config files. Version control them like infrastructure, share them like Docker Compose.
  actions:
    - theme: brand
      text: Create Workflow
      link: /workflows/creating-workflows
    - theme: alt
      text: View Examples
      link: /workflows/examples
    - theme: alt
      text: GitHub
      link: https://github.com/recinq/wave
features:
  - icon: 📋
    title: Declarative Workflows
    details: Define multi-step AI workflows in config files. Like Kubernetes manifests or Docker Compose, but for AI tasks with guaranteed outputs.
    link: /workflows/creating-workflows
  - icon: 🤝
    title: Guaranteed Contracts
    details: Every workflow step validates its output against schemas. No more unpredictable AI responses — get exactly what you specify.
    link: /concepts/contracts
  - icon: 🔄
    title: Version Controlled & Shareable
    details: Workflows are files in git. Share them with your team, deploy them to environments, track changes like any infrastructure code.
    link: /workflows/sharing-workflows
  - icon: 🏗️
    title: Familiar Patterns
    details: "If you know Terraform, Kubernetes, or Docker Compose, you already understand Wave's declarative approach to AI automation."
    link: /paradigm/infrastructure-parallels
  - icon: 🎯
    title: Reproducible Deliverables
    details: "Same input, same workflow → identical outputs. No variance, no surprises. AI becomes as predictable as your build pipeline."
    link: /paradigm/deliverables-contracts
  - icon: 🔧
    title: Ready-to-Use Library
    details: "Start with built-in workflows: code review, refactoring, documentation, testing. Customize or create your own patterns."
    link: /workflows/community-library
---

## AI Workflows as Code

Define AI automation like you define infrastructure — declarative, version-controlled, shareable.

```yaml
# workflow.yaml
name: code-review
description: Automated PR review with security and quality checks

steps:
  - name: analyze-diff
    persona: navigator
    input: "${pr.diff}"
    output:
      type: json
      schema: analysis-schema.json

  - name: security-review
    persona: auditor
    depends: [analyze-diff]
    input: "${steps.analyze-diff.output}"
    contracts:
      - type: test-suite
        path: ./tests/security-checks.js

  - name: summary
    persona: summarizer
    depends: [security-review]
    output:
      type: markdown
      deliverable: pr-review-comment.md
```

```bash
# Run like any infrastructure tool
wave apply workflow.yaml --input pr=123

# Version control your AI automation
git add workflow.yaml
git commit -m "Add automated security review workflow"
git push origin main

# Share with your team
wave run workflow.yaml --input pr=456
```

## Infrastructure Parallels

If you know these tools, you already understand Wave's approach:

| Tool Pattern | Wave Equivalent | Shared Concept |
|--------------|-----------------|----------------|
| `docker-compose.yml` | `workflow.yaml` | Declarative service orchestration |
| Kubernetes manifests | Wave workflows | Multi-step deployment with dependencies |
| Terraform configs | Wave pipelines | State management with guaranteed outputs |
| CI/CD pipelines | Wave automation | Reproducible, version-controlled execution |

## Guaranteed Deliverables

Unlike traditional AI tools, Wave enforces **contracts** at every step:

```yaml
# Traditional AI: unpredictable outputs
prompt: "Review this code for security issues"
# Result: ¯\_(ツ)_/¯ (might be thorough, might miss issues)

# Wave: guaranteed deliverables
steps:
  - name: security-review
    contracts:
      - type: json-schema
        schema: security-findings.schema.json
      - type: test-suite
        tests: ./validate-security-report.js
    output:
      format: structured-report
      required_fields: [vulnerabilities, risk_score, recommendations]
# Result: Always gets validated JSON with required security fields
```

## Version Control Your AI

```bash
# Create workflow
wave init security-review

# Test and iterate
wave run security-review.yaml --input ./src

# Commit like infrastructure
git add security-review.yaml
git commit -m "Add automated security review workflow"

# Deploy to CI/CD
wave run security-review.yaml --input ${{ github.workspace }}

# Share with team
git clone repo && wave run workflows/security-review.yaml
```

Wave makes AI automation as **predictable**, **shareable**, and **maintainable** as your infrastructure code.
