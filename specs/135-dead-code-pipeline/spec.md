# Improve dead-code detection pipeline: structured output, PR comments, and issue creation

> GitHub Issue: [#135](https://github.com/re-cinq/wave/issues/135)
> Labels: `enhancement`, `code-quality`, `pipeline`
> Author: nextlevelshit
> State: OPEN

## Problem

The current dead-code pipeline lacks visibility into AI-generated code quality. Since no human reviews every line of produced code, several risks accumulate silently:

- **Dead code and unused exports** go undetected
- **Duplicate implementations** of the same functionality across packages
- **Incorrect or redundant tests** that test the wrong behavior
- **Stale glue code** that once connected components but is now orphaned

These issues compound over time and degrade codebase health without any automated feedback loop.

## Proposal

Rewrite the dead-code detection pipeline to provide structured, actionable output. This issue covers the **detection and reporting** side. A separate issue should be created for the code-healing pipeline (see below).

### Dead-Code Detection Pipeline

The pipeline should:

1. **Scan for high-signal tokens** indicating potential dead code:
   - Unused exports and unexported symbols
   - Identify hard-coded values and propose configuration, env vars, options etc.
   - Functions with zero callers (beyond tests)
   - Duplicate function signatures across packages
   - Orphaned glue code (adapter/wrapper functions with no consumers)
   - Test files testing removed or renamed functions

2. **Produce structured output** in a defined JSON schema containing:
   - File path, line range, and symbol name
   - Category (dead code, duplicate, orphaned test, stale glue)
   - Confidence score (high/medium/low)
   - Suggested action (remove, consolidate, investigate)

3. **Support multiple output modes** depending on trigger context:
   - **PR trigger**: Post a review comment on the pull request summarizing findings
   - **Scheduled/on-demand trigger**: Create a GitHub issue with the full report
   - **Local trigger**: Write a JSON report to the filesystem

### Out of Scope (Separate Issue)

A `heal-code` pipeline that automatically fixes detected issues should be tracked separately. That pipeline would consume the dead-code report as input and apply safe automated fixes.

## Acceptance Criteria

- [ ] Pipeline produces valid JSON output matching a defined contract schema
- [ ] At least the following categories are detected: unused exports, duplicate implementations, orphaned tests
- [ ] PR trigger mode posts a summary comment on the pull request
- [ ] On-demand trigger mode creates a GitHub issue with findings
- [ ] Pipeline definition is added/updated in `.wave/pipelines/`
- [ ] Contract schema is added in `.wave/contracts/`
- [ ] Unit tests cover the core detection logic

## Existing Baseline

The current `dead-code.yaml` pipeline has four steps:
1. **scan** (navigator) - Scans for dead code with a prompt-based approach
2. **clean** (craftsman) - Removes high-confidence findings
3. **verify** (auditor) - Verifies removals were safe
4. **create-pr** (craftsman) - Creates a PR with the changes

The existing contract schema (`dead-code-scan.schema.json`) supports types: `unused_export`, `unreachable`, `orphaned_file`, `redundant`, `stale_test`, `unused_import`, `commented_code`.

### What needs to change

- The scan step schema needs new categories: `duplicate`, `stale_glue`, `hardcoded_value`
- The scan step schema needs new fields: `line_range`, `suggested_action`
- The pipeline needs to support three output modes (PR comment, GitHub issue, local JSON)
- New report formatting steps are needed for PR and issue modes
- The existing clean/verify/create-pr steps should be preserved as an optional downstream flow

## References

- Existing pipeline: `.wave/pipelines/dead-code.yaml`
- Existing contract: `.wave/contracts/dead-code-scan.schema.json`
- GitHub client: `internal/github/client.go` (has `CreateIssueComment`, `CreatePullRequest`)
- Personas available: `navigator`, `craftsman`, `auditor`, `github-commenter`
- Reference pipeline with PR comments: `.wave/pipelines/code-review.yaml`
