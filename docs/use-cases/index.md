---
title: Use Cases
description: Explore Wave pipelines for code review, security audits, documentation, testing, and more
---

<script setup>
import UseCaseGallery from '../.vitepress/theme/components/UseCaseGallery.vue'

const useCases = [
  {
    id: 'code-review',
    title: 'Code Review',
    description: 'Automated PR reviews with security checks, quality analysis, and actionable feedback.',
    category: 'code-quality',
    complexity: 'beginner',
    personas: ['navigator', 'auditor', 'summarizer'],
    tags: ['PR review', 'quality'],
    link: '/use-cases/code-review'
  },
  {
    id: 'security-audit',
    title: 'Security Audit',
    description: 'Comprehensive vulnerability scanning, dependency checks, and compliance verification.',
    category: 'security',
    complexity: 'intermediate',
    personas: ['navigator', 'auditor', 'summarizer'],
    tags: ['vulnerabilities', 'compliance'],
    link: '/use-cases/security-audit'
  },
  {
    id: 'documentation-generation',
    title: 'Documentation Generation',
    description: 'Generate API docs, README files, and usage guides automatically from your code.',
    category: 'documentation',
    complexity: 'beginner',
    personas: ['navigator', 'philosopher', 'auditor'],
    tags: ['API docs', 'README'],
    link: '/use-cases/documentation-generation'
  },
  {
    id: 'test-generation',
    title: 'Test Generation',
    description: 'Analyze coverage gaps and generate comprehensive tests with edge case handling.',
    category: 'testing',
    complexity: 'intermediate',
    personas: ['navigator', 'craftsman', 'auditor'],
    tags: ['coverage', 'unit tests'],
    link: '/use-cases/test-generation'
  },
  {
    id: 'refactoring',
    title: 'Refactoring',
    description: 'Systematic code refactoring with analysis, implementation, and verification steps.',
    category: 'code-quality',
    complexity: 'intermediate',
    personas: ['navigator', 'craftsman', 'auditor'],
    tags: ['code quality', 'maintainability'],
    link: '/use-cases/refactoring'
  },
  {
    id: 'multi-agent-review',
    title: 'Multi-Agent Review',
    description: 'Parallel specialized reviews combining security, performance, and architecture analysis.',
    category: 'code-quality',
    complexity: 'advanced',
    personas: ['navigator', 'auditor', 'philosopher', 'summarizer'],
    tags: ['parallel', 'comprehensive'],
    link: '/use-cases/multi-agent-review'
  },
  {
    id: 'incident-response',
    title: 'Incident Response',
    description: 'Rapid investigation and remediation workflows for production incidents.',
    category: 'devops',
    complexity: 'advanced',
    personas: ['navigator', 'auditor', 'craftsman', 'summarizer'],
    tags: ['debugging', 'root cause'],
    link: '/use-cases/incident-response'
  },
  {
    id: 'onboarding',
    title: 'Developer Onboarding',
    description: 'Generate onboarding materials and codebase exploration guides for new team members.',
    category: 'onboarding',
    complexity: 'beginner',
    personas: ['navigator', 'philosopher'],
    tags: ['documentation', 'knowledge transfer'],
    link: '/use-cases/onboarding'
  },
  {
    id: 'api-design',
    title: 'API Design',
    description: 'Design and validate APIs with schema generation, documentation, and contract testing.',
    category: 'documentation',
    complexity: 'intermediate',
    personas: ['philosopher', 'craftsman', 'auditor'],
    tags: ['OpenAPI', 'contracts'],
    link: '/use-cases/api-design'
  },
  {
    id: 'migration',
    title: 'Migration',
    description: 'Plan and execute codebase migrations with dependency analysis and verification.',
    category: 'devops',
    complexity: 'advanced',
    personas: ['navigator', 'craftsman', 'auditor', 'summarizer'],
    tags: ['upgrades', 'dependencies'],
    link: '/use-cases/migration'
  }
]
</script>

# Use Cases

Explore real-world Wave pipelines for common development tasks. Each use case includes a complete, runnable pipeline you can copy and customize.

<UseCaseGallery :use-cases="useCases" :show-filters="true" />

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

## Complexity Levels

Use cases are categorized by complexity to help you find the right starting point:

| Level | Description | Best For |
|-------|-------------|----------|
| **Beginner** | Single or simple multi-step pipelines with minimal configuration | Getting started, common tasks |
| **Intermediate** | Multi-step pipelines with artifacts, contracts, and parallel execution | Regular development workflows |
| **Advanced** | Complex orchestration with multiple personas, conditional logic, and custom contracts | Enterprise workflows, critical systems |

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
