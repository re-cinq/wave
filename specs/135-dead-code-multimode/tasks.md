# Tasks

## Phase 1: Schema Enhancement

- [X] Task 1.1: Add `suggested_action` enum to `dead-code-scan.schema.json`
  - Add optional `suggested_action` field to finding items with enum `["remove", "consolidate", "investigate"]`
  - Field is optional to maintain backward compatibility with existing scan outputs
  - File: `.wave/contracts/dead-code-scan.schema.json`

- [X] Task 1.2: Add `duplicate_signature` to finding type enum in `dead-code-scan.schema.json`
  - Extend the `type` enum from 7 to 8 values by adding `"duplicate_signature"`
  - File: `.wave/contracts/dead-code-scan.schema.json`

## Phase 2: Existing Pipeline Update

- [X] Task 2.1: Add `requires.tools` to `dead-code.yaml`
  - Add `requires: tools: [go]` section after `metadata` block
  - Follow convention from `gh-pr-review.yaml`, `gh-implement.yaml`, etc.
  - File: `.wave/pipelines/dead-code.yaml`

## Phase 3: PR Review Pipeline

- [X] Task 3.1: Create `dead-code-review.yaml` pipeline
  - 3-step pipeline: `scan` → `compose` → `publish`
  - `scan` step: navigator persona, readonly mount, scans only PR-changed files (PR number from `{{ input }}`)
  - `compose` step: summarizer persona, composes a markdown review comment from findings
  - `publish` step: github-commenter persona, posts comment via `gh pr comment`
  - Declare `requires: tools: [go, gh]`
  - Reuse `dead-code-scan.schema.json` for scan output contract
  - Reuse `gh-pr-comment-result.schema.json` for publish output contract
  - File: `.wave/pipelines/dead-code-review.yaml`

## Phase 4: Issue Creation Pipeline

- [X] Task 4.1: Create `dead-code-issue-result.schema.json` contract
  - Based on `doc-issue-result.schema.json` pattern
  - Required fields: `success`, `repository`, `timestamp`
  - Optional fields: `issue` (with `number`, `url`, `title`, `finding_count`), `skipped`, `error`
  - File: `.wave/contracts/dead-code-issue-result.schema.json`

- [X] Task 4.2: Create `dead-code-issue.yaml` pipeline [P]
  - 3-step pipeline: `scan` → `compose-report` → `create-issue`
  - `scan` step: navigator persona, readonly mount, full codebase scan
  - `compose-report` step: navigator persona, composes a markdown issue body from findings
  - `create-issue` step: craftsman persona, creates GitHub issue via `gh issue create --body-file`
  - Declare `requires: tools: [go, gh]`
  - Reuse `dead-code-scan.schema.json` for scan output contract
  - Use `dead-code-issue-result.schema.json` for create-issue output contract
  - Include `outcomes` with type `issue` for deliverable tracking
  - File: `.wave/pipelines/dead-code-issue.yaml`

## Phase 5: Validation

- [X] Task 5.1: Run `go test ./...` to verify pipeline YAML parses correctly
- [X] Task 5.2: Validate all JSON schemas are well-formed JSON Schema draft-07
- [X] Task 5.3: Verify the updated `dead-code-scan.schema.json` accepts both old and new format findings
