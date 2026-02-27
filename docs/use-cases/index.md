---
title: Use Cases
description: Explore Wave pipelines for code review, security audits, documentation, testing, and more
---

# Use Cases

Explore real-world Wave pipelines for common development tasks. Each use case includes a complete, runnable pipeline you can copy and customize.

## Browse Use Cases

<script setup>
import UseCaseGallery from '../.vitepress/theme/components/UseCaseGallery.vue'

const useCases = [
  {
    id: 'gh-pr-review',
    title: 'Code Review',
    description: 'Automated PR reviews with security checks, quality analysis, and actionable feedback.',
    category: 'code-quality',
    complexity: 'beginner',
    personas: ['navigator', 'auditor', 'summarizer'],
    tags: ['PR review', 'quality'],
    link: '/use-cases/gh-pr-review'
  },
  {
    id: 'audit-pipelines',
    title: 'Audit Pipelines',
    description: 'Reusable audit pipelines for code quality, security, dependency health, and common flaws with unified JSON output.',
    category: 'code-quality',
    complexity: 'beginner',
    personas: ['navigator', 'auditor', 'summarizer'],
    tags: ['audit', 'quality', 'security', 'dependencies'],
    link: '/use-cases/audit-pipelines'
  },
  {
    id: 'doc-audit',
    title: 'Documentation Consistency',
    description: 'Pre-PR gate that scans code changes, cross-references docs, and creates a GitHub issue for inconsistencies.',
    category: 'documentation',
    complexity: 'intermediate',
    personas: ['navigator', 'reviewer', 'github-analyst'],
    tags: ['docs', 'consistency', 'gh'],
    link: '/use-cases/doc-audit'
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
  {
    id: 'supervise',
    title: 'Work Supervision',
    description: 'Review output quality and process quality of completed work, including AI session transcripts.',
    category: 'code-quality',
    complexity: 'intermediate',
    personas: ['supervisor', 'reviewer'],
    tags: ['quality', 'process review', 'claudit'],
    link: '/use-cases/supervise'
  },
  {
    id: 'recinq',
    title: 'Recinq',
    description: 'Rethink and simplify code using divergent-convergent thinking (Double Diamond model).',
    category: 'code-quality',
    complexity: 'advanced',
    personas: ['provocateur', 'planner', 'craftsman'],
    tags: ['simplification', 'complexity', 'refactoring'],
    link: '/use-cases/recinq'
  },
]
</script>

<UseCaseGallery :use-cases="useCases" :show-filters="true" />
