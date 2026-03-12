# docs: documentation consistency report — outcomes feature gaps, contract type corrections, and new contract/exec types

**Issue**: [#134](https://github.com/re-cinq/wave/issues/134)
**Labels**: documentation, enhancement, priority: high
**Author**: nextlevelshit
**State**: OPEN

## Summary

Documentation consistency report identifying 6 gaps between the codebase and documentation. Covers undocumented `outcomes` pipeline feature, incorrect contract type names in pipeline-schema.md, undocumented `slash_command` exec type, and missing contract type documentation.

## Severity Breakdown

| Severity | Count | Description |
|----------|-------|-------------|
| Critical | 1 | Undocumented `outcomes` schema field in pipeline-schema.md |
| High | 2 | Missing outcomes concept documentation |
| Medium | 3 | Contract type name mismatches, missing contract types, incomplete exec type docs |

## Detailed Tasks

### DOC-001 — `outcomes` field missing from pipeline-schema.md Step Fields table (Critical)

The `Step` struct in `internal/pipeline/types.go:128` includes an `Outcomes []OutcomeDef` field with types `pr`, `issue`, `url`, and `deployment`. This field is actively used in production pipelines. However, `docs/reference/pipeline-schema.md` does not list `outcomes` in the Step Fields table (lines 100–118), nor does it document `OutcomeDef` fields (`type`, `extract_from`, `json_path`, `label`).

**Fix**: Add `outcomes` to the Step Fields table and add a new "## Outcomes" section documenting the `OutcomeDef` sub-fields.

### DOC-002 — No outcomes concept doc (High)

There is no `docs/concepts/outcomes.md`. Outcomes are a first-class pipeline feature for extracting structured results (PR URLs, issue links, deployment URLs) from step artifacts into the pipeline output summary.

**Fix**: Create `docs/concepts/outcomes.md` explaining the concept, showing examples, and linking to the schema reference.

### DOC-003 — Pipelines concept doc does not reference outcomes (High)

`docs/concepts/pipelines.md` covers artifacts, memory strategies, and contracts but never mentions outcomes.

**Fix**: Add an "Outcomes" subsection to `docs/concepts/pipelines.md` with a brief explanation and cross-link to the concept doc.

### DOC-004 — Contract type names in pipeline-schema.md are wrong (Medium)

`docs/reference/pipeline-schema.md` line 333 lists contract types as `testsuite`, `jsonschema`, `typescript`, `markdownspec`. The actual type strings in `internal/contract/contract.go` (lines 73–89) are `json_schema`, `test_suite`, `typescript_interface`, `markdown_spec`.

**Fix**: Update the Contract Fields table in `pipeline-schema.md` to use the correct type names: `json_schema`, `test_suite`, `typescript_interface`, `markdown_spec`.

### DOC-005 — `slash_command` exec type not in pipeline-schema.md (Medium)

`docs/reference/pipeline-schema.md` line 106 lists exec types as only `prompt` or `command`. The `ExecConfig` struct in `internal/pipeline/types.go:204` supports `slash_command` as a third type, with dedicated `Command` and `Args` fields.

**Fix**: Add `slash_command` to the exec type list in `pipeline-schema.md` and document the `command` and `args` sub-fields under Exec Configuration.

### DOC-006 — `format` and `non_empty_file` contract types missing from pipeline-schema.md (Medium)

`internal/contract/contract.go` registers `format` and `non_empty_file` contract validator types. While `docs/reference/contract-types.md` already documents both, `docs/reference/pipeline-schema.md` line 333 does not list them in the Contract Fields table.

**Note**: The original issue referenced a `template` contract type, but this type no longer exists in the current codebase. The `non_empty_file` type is the correct additional undocumented type in pipeline-schema.md.

**Fix**: Add `format` and `non_empty_file` to the Contract Fields table in `pipeline-schema.md`.

## Acceptance Criteria

- All six documentation items (DOC-001 through DOC-006) resolved
- `go test ./...` passes after changes
- No new documentation inconsistencies introduced
- Cross-references between new and existing docs are consistent

## Source Code References

- `internal/pipeline/types.go` — `Step` struct (line 128), `OutcomeDef` (line 281), `ExecConfig` (line 204)
- `internal/contract/contract.go` — `NewValidator` switch (lines 72–88)
- `docs/reference/pipeline-schema.md` — Step Fields table (lines 100–118), Contract Fields (line 333)
- `docs/concepts/pipelines.md` — Pipeline concepts (no outcomes section)
- `docs/reference/contract-types.md` — Already documents `format` and `non_empty_file`
