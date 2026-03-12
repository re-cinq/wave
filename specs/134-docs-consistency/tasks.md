# Tasks

## Phase 1: Critical — Outcomes in pipeline-schema.md (DOC-001)

- [X] Task 1.1: Add `outcomes` row to the Step Fields table in `docs/reference/pipeline-schema.md` (after `output_artifacts`, before `handover.contract`)
- [X] Task 1.2: Add a new `## Outcomes` section in `docs/reference/pipeline-schema.md` after `## Output Artifacts` documenting `OutcomeDef` sub-fields (`type`, `extract_from`, `json_path`, `json_path_label`, `label`) with a YAML example

## Phase 2: High — Outcomes concept documentation (DOC-002, DOC-003)

- [X] Task 2.1: Create `docs/concepts/outcomes.md` with concept explanation, supported outcome types (`pr`, `issue`, `url`, `deployment`), YAML examples, field reference table, and cross-links to pipeline-schema.md [P]
- [X] Task 2.2: Add an "Outcomes" subsection to `docs/concepts/pipelines.md` after the "Memory Strategies" section, with brief explanation and cross-link to the new concept doc [P]

## Phase 3: Medium — Contract type corrections and exec type docs (DOC-004, DOC-005, DOC-006)

- [X] Task 3.1: Fix contract type names in `docs/reference/pipeline-schema.md` Contract Fields table (line 333): `testsuite` → `test_suite`, `jsonschema` → `json_schema`, `typescript` → `typescript_interface`, `markdownspec` → `markdown_spec` [P]
- [X] Task 3.2: Fix contract type names in pipeline-schema.md Contract section examples: `testsuite` → `test_suite` (line 303), `typescript` → `typescript_interface` (line 326) [P]
- [X] Task 3.3: Add `format` and `non_empty_file` to the Contract Fields type list in pipeline-schema.md [P]
- [X] Task 3.4: Add `slash_command` to the exec type description in the Step Fields table (`exec.type` row, line 106) [P]
- [X] Task 3.5: Add a `### Slash Command Execution` subsection under `## Exec Configuration` documenting `type: slash_command` with `command` and `args` fields [P]

## Phase 4: Validation

- [X] Task 4.1: Run `go test ./...` to confirm no regressions
- [X] Task 4.2: Verify all cross-references between new and existing docs are consistent
- [X] Task 4.3: Verify contract type names in pipeline-schema.md match `internal/contract/contract.go` exactly
