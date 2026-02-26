# Tasks

## Phase 1: Schema Updates

- [X] Task 1.1: Expand `dead-code-scan.schema.json` with new detection categories (`duplicate`, `stale_glue`, `hardcoded_value`), `line_range` object, and `suggested_action` field
- [X] Task 1.2: Create `dead-code-report.schema.json` for formatted markdown report output (title, body markdown, findings_count, categories_found)
- [X] Task 1.3: Create `dead-code-pr-result.schema.json` for PR comment posting result (comment_url, pr_number, findings_summary) [P]
- [X] Task 1.4: Create `dead-code-issue-result.schema.json` for GitHub issue creation result (issue_url, issue_number, findings_summary) [P]

## Phase 2: Core Pipeline Rewrite

- [X] Task 2.1: Rewrite the scan step in `dead-code.yaml` with enhanced detection prompts covering all required categories (unused exports, duplicates, orphaned tests, stale glue, hardcoded values)
- [X] Task 2.2: Update the scan step contract reference to match the expanded schema
- [X] Task 2.3: Remove clean/verify/create-pr steps from `dead-code.yaml` (local-only mode: scan + report)
- [X] Task 2.4: Add a format step to `dead-code.yaml` that transforms raw JSON scan results into a human-readable summary report

## Phase 3: Output Mode Pipelines

- [X] Task 3.1: Create `dead-code-pr.yaml` pipeline with scan + format + publish-pr-comment steps [P]
- [X] Task 3.2: Create `dead-code-issue.yaml` pipeline with scan + format + create-github-issue steps [P]
- [X] Task 3.3: Create `dead-code-heal.yaml` pipeline preserving the existing clean/verify/create-pr flow, consuming scan results as input

## Phase 4: Persona Permissions

- [X] Task 4.1: Verify `github-commenter` persona has required permissions for `gh pr comment` (currently only allows `gh issue comment*`); extend if needed
- [X] Task 4.2: Add `requires.tools` declarations to `dead-code-pr.yaml` and `dead-code-issue.yaml` for `gh` CLI preflight check

## Phase 5: Testing

- [X] Task 5.1: Write schema validation tests for expanded `dead-code-scan.schema.json` with sample payloads covering all new categories [P]
- [X] Task 5.2: Write schema validation tests for new schemas (`dead-code-report`, `dead-code-pr-result`, `dead-code-issue-result`) [P]
- [X] Task 5.3: Write pipeline YAML parse tests verifying all new/modified pipelines load correctly and resolve dependencies
- [X] Task 5.4: Verify backward compatibility — existing valid payloads still validate against updated schema

## Phase 6: Documentation and Polish

- [X] Task 6.1: Update pipeline descriptions and metadata in all new/modified YAML files
- [X] Task 6.2: Verify all persona references in pipeline steps exist in `wave.yaml`
