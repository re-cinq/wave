# Pipeline Naming Conventions

This document defines the naming conventions for Wave pipelines. All new pipelines should follow these rules.

## Rules

1. **Pattern**: Use `<category>-<verb>` in kebab-case. Every pipeline must have a category prefix.
2. **Max length**: 25 characters.
3. **Clarity**: A developer should be able to guess what the pipeline does from the name alone.
4. **Grouping**: Related pipelines share a common category prefix. See [Taxonomy Categories](#taxonomy-categories) below.
5. **No abbreviations** unless universally understood (e.g., `adr`, `gh`).

## Taxonomy Categories

All built-in pipelines are grouped into 6 primary categories plus 2 integration-specific prefixes:

| Prefix | Purpose | Destructive? | Description |
|--------|---------|-------------|-------------|
| `audit-*` | Analysis and reporting | No | Read-only pipelines that analyze code, docs, or processes and produce reports |
| `plan-*` | Planning and specification | No | Pre-implementation pipelines that produce specs, plans, and task breakdowns |
| `impl-*` | Code implementation | Yes | Pipelines that modify source code, create PRs, or apply fixes |
| `doc-*` | Documentation generation | Varies | Pipelines that generate, audit, or fix documentation |
| `test-*` | Testing and verification | No | Pipelines that run tests, verify configurations, or check coverage |
| `ops-*` | Operational and meta-pipelines | Varies | Utility pipelines for orchestration, demos, and operational tasks |
| `gh-*` | GitHub operations | Varies | GitHub-specific operations (issues, PRs, comments) |
| `wave-*` | Wave self-evolution | Varies | Wave's own self-improvement pipelines |

## Current Pipeline Inventory

| Name | Description |
|---|---|
| `audit-docs` | Audit documentation for consistency and create issues for drift |
| `audit-security` | Scan the codebase for security vulnerabilities |
| `audit-supervise` | Supervise and coordinate multi-agent pipeline execution |
| `doc-adr` | Create an Architecture Decision Record |
| `doc-changelog` | Generate a changelog from recent commits |
| `doc-explain` | Explain a piece of code or architecture |
| `doc-fix` | Scan documentation for inconsistencies, fix them, and create a PR |
| `doc-onboard` | Generate onboarding documentation for new contributors |
| `gh-implement` | Implement a GitHub issue end-to-end with PR creation |
| `gh-pr-review` | Review code changes and provide feedback |
| `gh-research` | Research a GitHub issue and post findings as a comment |
| `gh-rewrite` | Analyze and rewrite poorly documented GitHub issues |
| `gh-refresh` | Refresh a stale GitHub issue against recent codebase changes |
| `impl-dead-code` | Detect and report unused code |
| `impl-debug` | Debug a failing test or runtime issue |
| `impl-feature` | Implement a new feature end-to-end |
| `impl-hotfix` | Apply a targeted bug fix |
| `impl-improve` | Improve existing code quality |
| `impl-prototype` | Rapidly prototype a feature or concept |
| `impl-recinq` | Rethink and simplify code using divergent-convergent analysis |
| `impl-refactor` | Refactor code for better structure and maintainability |
| `ops-hello-world` | Example pipeline for testing and demos |
| `plan-feature` | Create an implementation plan for a task |
| `plan-speckit` | Specification-driven feature development using the speckit workflow |
| `test-gen` | Generate tests for existing code |
| `test-smoke` | Run smoke tests to verify basic functionality |

## Examples

Good names:
- `gh-implement` — GitHub scope, implement verb
- `audit-docs` — audit category, documentation scope
- `audit-security` — audit category, security scope
- `impl-refactor` — implementation category, refactor verb

Avoid:
- Abbreviations that aren't universally understood (e.g., `sync`)
- Verbose prefixes that repeat scope (e.g., `gh-issue-` when `gh-` suffices)
- Abstract nouns that don't convey action (e.g., `loop`, `flow`)
- Unprefixed names — every pipeline must have a category prefix
