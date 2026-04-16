# Tasks

## Phase 1: Schema Audit and Cleanup

- [X] Task 1.1: Delete unreferenced `dead-code-scan.schema.json` from both `.wave/contracts/` and `internal/defaults/contracts/`
- [X] Task 1.2: Sync `shared-findings.schema.json` — copy `.wave/contracts/` version to `internal/defaults/contracts/` (adds 4 type values, `additionalProperties: false`, `format: date-time`)
- [X] Task 1.3: Add `additionalProperties: false` to `shared-review-verdict.schema.json` in both locations
- [X] Task 1.4: Promote 5 `.wave/`-only schemas to `internal/defaults/contracts/`: `aggregated-findings`, `rework-gate-verdict`, `config-check`, `smoke-review`, `wave-smoke-test` [P]

## Phase 2: Severity Standardization

- [X] Task 2.1: Normalize `security-scan.schema.json` severity from uppercase `["CRITICAL","HIGH","MEDIUM","LOW"]` to lowercase `["critical","high","medium","low","info"]` [P]
- [X] Task 2.2: Normalize `doc-fix-scan.schema.json` severity from uppercase to lowercase enum [P]
- [X] Task 2.3: Add missing severity values to `triage-verdict.schema.json` (add `"high"`, `"low"`, `"info"`) [P]
- [X] Task 2.4: Add missing severity values to `feature-exploration.schema.json` risk levels (add `"critical"`, `"info"`) [P]
- [X] Task 2.5: Sync all modified schemas to `internal/defaults/contracts/` [P]

## Phase 3: Shared PR Result Schema

- [X] Task 3.1: Create `shared-pr-result.schema.json` with `pr_url`, `pr_number`, `branch`, `summary`, `title` fields
- [X] Task 3.2: Copy to `internal/defaults/contracts/shared-pr-result.schema.json`
- [X] Task 3.3: Migrate `pr-result.schema.json` to reference shared fields (align property names, keep backward compat by keeping existing required fields) [P]

## Phase 4: Input Validation Wiring

- [X] Task 4.1: Add `schema_path: .wave/contracts/shared-findings.schema.json` to `inject_artifacts` in `audit-architecture.yaml` (report step, scan_findings artifact) [P]
- [X] Task 4.2: Add `schema_path` to `inject_artifacts` in `audit-tests.yaml` (report step) [P]
- [X] Task 4.3: Add `schema_path` to `inject_artifacts` in `audit-correctness.yaml` (report step) [P]
- [X] Task 4.4: Add `schema_path` to `inject_artifacts` in `audit-duplicates.yaml` (report step) [P]
- [X] Task 4.5: Add `schema_path` to `inject_artifacts` in `audit-doc-scan.yaml` (report step) [P]
- [X] Task 4.6: Add `schema_path` to `inject_artifacts` in `audit-doc.yaml` (normalize and publish steps) [P]
- [X] Task 4.7: Add `schema_path` to `inject_artifacts` in `audit-consolidate.yaml` (consolidate step) [P]
- [X] Task 4.8: Add `schema_path` to `inject_artifacts` in `audit-dead-code-issue.yaml` (verify step) [P]
- [X] Task 4.9: Add `schema_path` to `inject_artifacts` in `ops-pr-fix-review.yaml` (triage step, review-findings schema) [P]

## Phase 5: Testing

- [X] Task 5.1: Add sync test to verify shared schemas match between `.wave/contracts/` and `internal/defaults/contracts/`
- [X] Task 5.2: Add input validation test cases using shared schemas in `internal/contract/input_validator_test.go`
- [X] Task 5.3: Run `go test ./internal/contract/...` and fix any failures
- [X] Task 5.4: Run `go test ./...` full suite and fix any regressions

## Phase 6: Documentation and Polish

- [X] Task 6.1: Update `docs/guides/contract-chaining.md` — add shared-pr-result, severity conventions, input validation wiring section
- [X] Task 6.2: Update `internal/contract/README.md` with input validation examples
- [X] Task 6.3: Final validation — verify all schema files are valid JSON Schema Draft-07
