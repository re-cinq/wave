# Implementation Plan — Issue #134

## Objective

Fix 6 documentation inconsistencies between the Wave codebase and its reference/concept documentation. The changes are documentation-only: correcting contract type names, adding the undocumented `outcomes` feature and `slash_command` exec type, and ensuring pipeline-schema.md lists all contract types.

## Approach

Work through each DOC item sequentially, starting with the most impactful (DOC-001 critical, DOC-002/003 high) and finishing with medium-severity corrections. Since contract-types.md is already up to date, the bulk of the work is in pipeline-schema.md (3 fixes), a new outcomes concept doc, and a small addition to pipelines.md.

## File Mapping

| File | Action | DOC Items |
|------|--------|-----------|
| `docs/reference/pipeline-schema.md` | modify | DOC-001, DOC-004, DOC-005, DOC-006 |
| `docs/concepts/outcomes.md` | create | DOC-002 |
| `docs/concepts/pipelines.md` | modify | DOC-003 |

**Note**: `docs/reference/contract-types.md` already documents `format` and `non_empty_file` correctly — no changes needed there.

## Architecture Decisions

1. **DOC-006 scope adjustment**: The original issue references a `template` contract type, but it no longer exists in `internal/contract/contract.go`. The actual undocumented types in pipeline-schema.md are `format` and `non_empty_file` (both already covered in contract-types.md). The fix is to add these to pipeline-schema.md's Contract Fields table.

2. **Outcomes section placement in pipeline-schema.md**: Add a new `## Outcomes` section after `## Output Artifacts` (around line 224) since outcomes are semantically related to step output.

3. **Slash command section**: Add a `### Slash Command Execution` subsection under `## Exec Configuration` alongside the existing prompt and command subsections.

4. **Outcomes concept doc structure**: Follow the same pattern as other concept docs (e.g., `artifacts.md`, `contracts.md`) — introduction, YAML examples, field reference table, cross-links.

5. **Contract type examples in pipeline-schema.md**: The existing Contracts section (lines 294–340) uses wrong type names in individual examples too (`testsuite` on line 303, `typescript` on line 326). These must also be corrected.

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Stale line numbers in issue | Low | Verify all referenced lines against current files before editing |
| Cross-reference breakage | Low | Verify all inter-doc links after changes |
| Incorrect `OutcomeDef` field documentation | Medium | Cross-reference with `internal/pipeline/types.go` struct definition |
| VitePress rendering issues with new doc | Low | Follow existing doc patterns (div v-pre wrappers, etc.) |

## Testing Strategy

- Run `go test ./...` to verify no code changes were accidentally introduced
- Manually verify all cross-references between docs
- Verify the new `outcomes.md` file follows existing concept doc conventions
- Check that contract type names in pipeline-schema.md match `internal/contract/contract.go` exactly
