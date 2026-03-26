# Pipeline Naming Conventions

All Wave pipelines use a mandatory taxonomy prefix that guarantees the pipeline's behavior category.

## Taxonomy Prefixes

| Prefix | Behavior | Description |
|--------|----------|-------------|
| `audit-` | Read-only | Analysis, scanning, reporting — never modifies code |
| `doc-` | Documentation | Creates or fixes documentation |
| `impl-` | Writes code | Implements features, fixes bugs, creates PRs |
| `ops-` | Operational | Orchestration, maintenance, issue management, reviews |
| `plan-` | Read-only | Planning, research, scoping — never modifies code |
| `test-` | Testing | Test generation and validation |
| `wave-` | Self-evolution | Wave's own development and maintenance pipelines |

## Rules

1. **Every pipeline must have a prefix** from the table above.
2. **Pattern**: `<prefix><name>` in kebab-case (e.g., `impl-issue`, `ops-pr-review`).
3. **Max length**: 25 characters.
4. **Prefix guarantees behavior**: `plan-*` and `audit-*` pipelines never write code. `impl-*` pipelines always produce code changes.
5. **No abbreviations** unless universally understood (e.g., `adr`, `pr`, `dx`).

## Current Pipeline Inventory

### `audit-` — Analysis and Scanning

| Name | Description |
|------|-------------|
| `audit-consolidate` | Read-only codebase consolidation scan |
| `audit-dead-code` | Detect and report unused code |
| `audit-dead-code-issue` | Dead code scan with GitHub issue creation |
| `audit-dead-code-review` | Dead code scan with PR comment |
| `audit-doc` | Documentation consistency audit |
| `audit-dual` | Parallel code quality and security analysis |
| `audit-dx` | Developer experience analysis |
| `audit-junk-code` | Junk code detection |
| `audit-quality-loop` | Iterative quality improvement analysis |
| `audit-security` | Security vulnerability scanning |
| `audit-ux` | User experience analysis |

### `doc-` — Documentation

| Name | Description |
|------|-------------|
| `doc-changelog` | Generate changelog from recent commits |
| `doc-explain` | Explain code or architecture |
| `doc-fix` | Fix documentation inconsistencies and create PR |
| `doc-onboard` | Generate onboarding documentation |

### `impl-` — Implementation

| Name | Description |
|------|-------------|
| `impl-feature` | Plan, implement, test, and commit a feature |
| `impl-hotfix` | Apply a targeted bug fix |
| `impl-improve` | Improve existing code quality |
| `impl-issue` | Implement an issue end-to-end with PR creation |
| `impl-prototype` | Prototype-driven implementation (spec → docs → dummy → implement) |
| `impl-recinq` | Divergent-convergent code simplification |
| `impl-refactor` | Refactor code for better structure |
| `impl-research` | Research then implement (composition pipeline) |
| `impl-speckit` | Specification-driven implementation (specify → plan → tasks → implement) |

### `ops-` — Operational

| Name | Description |
|------|-------------|
| `ops-debug` | Debug a failing test or runtime issue |
| `ops-epic-runner` | Scope an epic and implement child issues sequentially |
| `ops-hello-world` | Example pipeline for testing and demos |
| `ops-implement-epic` | Implement all subissues from a scoped epic |
| `ops-pr-review` | Pull request code review with security and quality analysis |
| `ops-refresh` | Refresh a stale issue against recent codebase changes |
| `ops-release-harden` | Harden a release with targeted fixes |
| `ops-rewrite` | Rewrite poorly documented issues |
| `ops-supervise` | Supervise multi-agent pipeline execution |

### `plan-` — Planning and Research

| Name | Description |
|------|-------------|
| `plan-adr` | Create an Architecture Decision Record |
| `plan-research` | Research an issue and post findings |
| `plan-scope` | Decompose an epic into well-scoped child issues |
| `plan-task` | Create an implementation plan for a task |

### `test-` — Testing

| Name | Description |
|------|-------------|
| `test-gen` | Generate comprehensive test coverage |
| `test-smoke` | Minimal pipeline for testing contracts and artifacts |

### `wave-` — Self-Evolution

| Name | Description |
|------|-------------|
| `wave-audit` | Audit Wave's own codebase |
| `wave-bugfix` | Fix bugs in Wave itself |
| `wave-evolve` | Evolve Wave's capabilities |
| `wave-land` | Land changes to Wave's main branch |
| `wave-review` | Review Wave's own PRs |
| `wave-security-audit` | Security audit of Wave itself |
| `wave-test-hardening` | Harden Wave's test suite |

