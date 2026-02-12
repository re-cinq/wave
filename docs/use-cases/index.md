---
title: Use Cases
description: Explore Wave pipelines for code review, security audits, documentation, testing, and more
---

# Use Cases

Explore real-world Wave pipelines for common development tasks. Each use case includes a complete, runnable pipeline you can copy and customize.

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

<div v-pre>

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

</div>

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
```

## Next Steps

- [Quickstart](/quickstart) - Get Wave running in 60 seconds
- [Concepts: Pipelines](/concepts/pipelines) - Understand pipeline structure in depth
- [CLI Reference](/reference/cli) - Complete command documentation

---

## Browse Use Cases

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
    id: 'doc-loop',
    title: 'Documentation Consistency',
    description: 'Pre-PR gate that scans code changes, cross-references docs, and creates a GitHub issue for inconsistencies.',
    category: 'documentation',
    complexity: 'intermediate',
    personas: ['navigator', 'reviewer', 'github-analyst'],
    tags: ['docs', 'consistency', 'gh'],
    link: '/use-cases/doc-loop'
  },
  {
    id: 'github-issue-enhancer',
    title: 'Issue Enhancement',
    description: 'Scan GitHub issues for poor documentation, generate improvements, and apply them automatically.',
    category: 'github',
    complexity: 'intermediate',
    personas: ['github-analyst', 'github-enhancer'],
    tags: ['issues', 'quality', 'gh'],
    link: '/use-cases/github-issue-enhancer'
  },
  {
    id: 'issue-research',
    title: 'Issue Research',
    description: 'Research a GitHub issue with web search, synthesize findings, and post a structured comment.',
    category: 'github',
    complexity: 'intermediate',
    personas: ['github-analyst', 'researcher', 'summarizer', 'github-commenter'],
    tags: ['research', 'issues', 'gh'],
    link: '/use-cases/issue-research'
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
]
</script>

<UseCaseGallery :use-cases="useCases" :show-filters="true" />
