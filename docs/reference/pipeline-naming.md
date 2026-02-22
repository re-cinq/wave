# Pipeline Naming Conventions

This document defines the naming conventions for Wave pipelines. All new pipelines should follow these rules.

## Rules

1. **Pattern**: Use `<verb>` or `<scope>-<verb>` in kebab-case.
2. **Max length**: 20 characters.
3. **Clarity**: A developer should be able to guess what the pipeline does from the name alone.
4. **Grouping**: Related pipelines share a common prefix (e.g., `gh-` for GitHub operations, `doc-` for documentation).
5. **No abbreviations** unless universally understood (e.g., `adr`, `gh`).

## Current Pipeline Inventory

| Name | Description |
|---|---|
| `adr` | Create an Architecture Decision Record |
| `changelog` | Generate a changelog from recent commits |
| `code-review` | Review code changes and provide feedback |
| `dead-code` | Detect and report unused code |
| `debug` | Debug a failing test or runtime issue |
| `doc-audit` | Audit documentation for consistency and create issues for drift |
| `doc-fix` | Scan documentation for inconsistencies, fix them, and create a PR |
| `explain` | Explain a piece of code or architecture |
| `feature` | Implement a new feature end-to-end |
| `gh-implement` | Implement a GitHub issue end-to-end with PR creation |
| `gh-research` | Research a GitHub issue and post findings as a comment |
| `gh-rewrite` | Analyze and rewrite poorly documented GitHub issues |
| `gh-refresh` | Refresh a stale GitHub issue against recent codebase changes |
| `hello-world` | Example pipeline for testing and demos |
| `hotfix` | Apply a targeted bug fix |
| `improve` | Improve existing code quality |
| `onboard` | Generate onboarding documentation for new contributors |
| `plan` | Create an implementation plan for a task |
| `prototype` | Rapidly prototype a feature or concept |
| `refactor` | Refactor code for better structure and maintainability |
| `security-scan` | Scan the codebase for security vulnerabilities |
| `recinq` | Rethink and simplify code using divergent-convergent analysis |
| `smoke-test` | Run smoke tests to verify basic functionality |
| `speckit-flow` | Specification-driven feature development using the speckit workflow |
| `supervise` | Supervise and coordinate multi-agent pipeline execution |
| `test-gen` | Generate tests for existing code |

## Prefix Groups

| Prefix | Scope |
|---|---|
| `gh-` | GitHub API operations (issues, PRs, comments) |
| `doc-` | Documentation analysis and maintenance |
| `spec-` | Specification-driven workflows |
| _(none)_ | General development operations |

## Examples

Good names:
- `gh-implement` — GitHub scope, implement verb
- `doc-audit` — documentation scope, audit verb
- `security-scan` — security scope, scan verb
- `refactor` — single verb, self-explanatory

Bad names:
- `gh-issue-impl` — abbreviation (`impl`), verbose prefix (`gh-issue-`)
- `doc-loop` — unclear what "loop" means in context
- `speckit-flow` — trademark pipeline name (exception to naming rules)
