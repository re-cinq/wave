# Tasks

## Phase 1: Shared Definition Schemas

- [X] Task 1.1: Create `.wave/contracts/_defs/severity.schema.json` with canonical severity enums (`findings_severity`: critical/high/medium/low/info; `review_severity`: critical/major/minor/suggestion)
- [X] Task 1.2: Create `.wave/contracts/_defs/finding.schema.json` with base finding item schema extracted from `shared-findings.schema.json`
- [X] Task 1.3: Create `.wave/contracts/_defs/pr-reference.schema.json` with `{pr_number, pr_url}` extracted from `pr-result.schema.json`
- [X] Task 1.4: Create `.wave/contracts/_defs/issue-reference.schema.json` with `{issue_number, repository, issue_url}` extracted from `comment-result.schema.json`

## Phase 2: Validator Enhancement

- [X] Task 2.1: Modify `internal/contract/jsonschema.go` — add helper function `preloadSharedDefs(compiler, workspacePath)` that scans `.wave/contracts/_defs/*.schema.json` and registers each as a compiler resource with a canonical URI (e.g., `_defs/severity.schema.json`)
- [X] Task 2.2: Call `preloadSharedDefs` in `jsonSchemaValidator.Validate()` before `compiler.Compile()`
- [X] Task 2.3: Apply same `preloadSharedDefs` pattern in `internal/contract/input_validator.go` for `validateSingleInputArtifact()`
- [X] Task 2.4: Write unit tests in `internal/contract/jsonschema_refs_test.go` verifying `$ref` resolution with `_defs/` schemas [P]

## Phase 3: Schema Refactoring

- [X] Task 3.1: Refactor `shared-findings.schema.json` — `$ref` severity and finding item from `_defs/` [P]
- [X] Task 3.2: Refactor `aggregated-findings.schema.json` — `$ref` severity from `_defs/` [P]
- [X] Task 3.3: Refactor `shared-review-verdict.schema.json` — `$ref` review severity from `_defs/` [P]
- [X] Task 3.4: Refactor `review-findings.schema.json` — `$ref` pr-reference from `_defs/` [P]
- [X] Task 3.5: Refactor `pr-result.schema.json` — `$ref` pr-reference from `_defs/` [P]
- [X] Task 3.6: Refactor `gh-pr-comment-result.schema.json` — `$ref` pr-reference from `_defs/` [P]
- [X] Task 3.7: Refactor `comment-result.schema.json` — `$ref` issue-reference from `_defs/` [P]
- [X] Task 3.8: Refactor `research-findings.schema.json` — `$ref` issue-reference from `_defs/` [P]
- [X] Task 3.9: Refactor `research-report.schema.json` — `$ref` issue-reference from `_defs/` [P]
- [X] Task 3.10: Refactor `triage-verdict.schema.json` — `$ref` pr-reference from `_defs/` [P]

## Phase 4: Input Validation Wiring

- [X] Task 4.1: Add `schema_path` to `inject_artifacts` in `internal/defaults/pipelines/impl-issue.yaml` for assessment→plan and plan→implement chains [P]
- [X] Task 4.2: Add `schema_path` to `inject_artifacts` in `internal/defaults/pipelines/impl-issue-core.yaml` for assessment→plan chain [P]
- [X] Task 4.3: Add `schema_path` to `inject_artifacts` in `internal/defaults/pipelines/ops-pr-review-core.yaml` for diff→review chain [P]
- [X] Task 4.4: Add `schema_path` to `inject_artifacts` in `internal/defaults/pipelines/ops-pr-fix-review.yaml` for review→triage chain [P]

## Phase 5: Testing & Validation

- [X] Task 5.1: Write chain integration tests in `internal/contract/chain_test.go` — validate step A output schema compatibility with step B input schema for key chains
- [X] Task 5.2: Run `go test ./internal/contract/...` to verify all tests pass
- [X] Task 5.3: Verify refactored schemas validate existing test fixtures (no regression)
- [X] Task 5.4: Run `go vet ./internal/contract/...` for static analysis
