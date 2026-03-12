# Improve dead-code detection pipeline: multi-mode output, PR review comments, and issue creation

**Issue**: [#135](https://github.com/re-cinq/wave/issues/135)
**Labels**: enhancement, code-quality, pipeline
**Author**: nextlevelshit

## Problem

The current dead-code pipeline lacks visibility into AI-generated code quality. Since no human reviews every line of produced code, several risks accumulate silently:

- **Dead code and unused exports** go undetected
- **Duplicate implementations** of the same functionality across packages
- **Incorrect or redundant tests** that test the wrong behavior
- **Stale glue code** that once connected components but is now orphaned

These issues compound over time and degrade codebase health without any automated feedback loop.

## Current State

A `dead-code` pipeline already exists at `.wave/pipelines/dead-code.yaml` with four steps:

1. **scan** — navigator persona scans for dead code (unused exports, unreachable code, orphaned files, redundant code, stale tests, unused imports, commented-out code) and produces structured JSON output
2. **clean** — craftsman persona removes high-confidence findings on an isolated worktree branch
3. **verify** — reviewer persona verifies the removals were safe
4. **create-pr** — craftsman persona pushes the branch and creates a pull request

A contract schema exists at `.wave/contracts/dead-code-scan.schema.json` covering: `unused_export`, `unreachable`, `orphaned_file`, `redundant`, `stale_test`, `unused_import`, `commented_code` finding types with confidence scores and safe-to-remove flags.

This pipeline currently operates in a **single mode**: scan → clean → PR. The remaining work is to add alternative output modes and bring the pipeline in line with current conventions.

## Remaining Work

### 1. Add multi-mode output support

The pipeline currently only supports the full scan-clean-verify-PR flow. Add alternative modes:

- **PR review comment mode**: Given a PR number, scan only the changed files and post a review comment summarizing findings. This should be a separate pipeline (e.g., `dead-code-review`).
- **Issue creation mode**: On-demand or scheduled scan that creates a GitHub issue with the full structured report instead of cleaning code. This should be a separate pipeline (e.g., `dead-code-issue`).
- **Local report mode**: The current scan step already writes JSON to `.wave/output/dead-code-scan.json`. This mode is effectively supported — document it as the default for local use.

### 2. Add `requires.tools` field

Recent pipeline convention added `requires.tools` for preflight validation. The `dead-code` pipeline and new variant pipelines should declare their tool requirements (e.g., `go`, `gh`) to enable preflight checks.

### 3. Expand detection categories

The existing scan prompt and schema cover the core categories. Add:
- **`duplicate_signature`** — Duplicate function signatures across packages (already mentioned in the prompt as "redundant" but benefits from a dedicated type)

### 4. Add `suggested_action` to schema

Each finding should include a `suggested_action` field with an enum: `remove`, `consolidate`, `investigate`. The current schema has `safe_to_remove` (boolean) and `removal_note` (string) but no explicit action enum.

## Acceptance Criteria

- [x] Pipeline produces valid JSON output matching a defined contract schema
- [x] At least the following categories are detected: unused exports, duplicate implementations, orphaned tests
- [ ] PR review comment mode posts a summary comment on an existing pull request
- [ ] On-demand trigger mode creates a GitHub issue with findings
- [x] Pipeline definition exists in `.wave/pipelines/dead-code.yaml`
- [x] Contract schema exists in `.wave/contracts/dead-code-scan.schema.json`
- [ ] Pipeline declares `requires.tools` for preflight validation
- [ ] `suggested_action` enum added to scan schema
- [ ] `duplicate_signature` type added to scan schema

## Out of Scope

- A `heal-code` pipeline that automatically fixes detected issues (separate issue)
- Unit tests for detection logic (blocked until heuristics extracted to Go)
