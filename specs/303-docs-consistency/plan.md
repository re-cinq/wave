# Implementation Plan: Documentation Consistency Report (#303)

## 1. Objective

Resolve 11 active documentation inconsistencies identified in the doc-audit report (DOC-008 already resolved by commit f5987f2). The fixes span CLI reference docs, architecture docs, contract docs, environment docs, quick-start guide, and potentially Go source code (DOC-002 non_empty_file validator).

## 2. Approach

Work through items in severity order (Critical → High → Medium → Low). Group related changes by file to minimize context switching. DOC-002 requires a design decision: implement a `non_empty_file` validator in Go or remove the contract type from pipeline YAMLs.

**Recommended approach for DOC-002**: Implement a `non_empty_file` validator since 32 pipeline files already reference it. This is safer than modifying 32 YAML files and avoids breaking existing pipeline definitions.

## 3. File Mapping

### Files to Modify

| File | Items | Action |
|------|-------|--------|
| `docs/reference/cli.md` | DOC-001, DOC-003, DOC-004, DOC-010 | Add compose/doctor/suggest docs, add global flags, add --model, remove duplicate chat |
| `docs/concepts/architecture.md` | DOC-005, DOC-006, DOC-007 | Fix workspace path, update OpenCode status, add missing contract types |
| `docs/concepts/contracts.md` | DOC-009 | Fix max_retries default |
| `docs/reference/contract-types.md` | DOC-002 (docs), DOC-009 | Add non_empty_file docs, fix max_retries default |
| `docs/reference/environment.md` | DOC-011 | Clarify CLAUDE_CODE_MODEL usage |
| `docs/guide/quick-start.md` | DOC-012 | Update wave init output example |

### Files to Create

| File | Item | Purpose |
|------|------|---------|
| `internal/contract/non_empty_file.go` | DOC-002 | non_empty_file validator implementation |
| `internal/contract/non_empty_file_test.go` | DOC-002 | Validator tests |

### Files to Modify (Code)

| File | Item | Action |
|------|------|--------|
| `internal/contract/contract.go` | DOC-002 | Add `non_empty_file` case to `NewValidator()` switch |

## 4. Architecture Decisions

### DOC-002: non_empty_file Validator

**Decision**: Implement a new `non_empty_file` validator rather than removing the contract type from 32 pipeline YAMLs.

**Rationale**:
- 32 files reference this type — removing it is high churn and error-prone
- The concept is straightforward: check that `source` file exists and is non-empty
- Aligns with how pipelines use it: validating that a persona wrote output
- Follows existing validator pattern (implements `ContractValidator` interface)

**Behavior**:
- Resolve `source` path relative to workspace
- Check file exists (error if not)
- Check file size > 0 (error if empty)
- Support `must_pass`, `on_failure`, `max_retries` like other validators

### DOC-009: max_retries Default

The code has two paths with different defaults:
- `Validate()` defaults to 1 attempt (no retries)
- `ValidateWithRetries()` defaults to 3 retries

Document `max_retries` default as `2` in `contract-types.md` (which describes YAML config behavior) since the retry path is what pipelines use. Document as `0` in `contracts.md` conceptual doc to mean "no explicit retry configuration" which delegates to the validator's default.

**Revised approach**: Align both docs to say default is `2` for `test_suite`/`json_schema`/`typescript_interface`/`markdown_spec`/`format` since that's what `contract-types.md` already says and matches the YAML-facing behavior.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| DOC-002 validator could break existing pipelines if behavior differs from expectations | Simple validator with clear semantics: file exists + non-empty. Add comprehensive tests |
| Incorrect line references in issue may not match current code | Always verify by reading current file state, not relying on line numbers |
| DOC-008 already fixed — must not revert | Skip DOC-008 entirely |
| Quick-start output is dynamic (depends on selected pipelines) | Show a representative example rather than exact output |

## 6. Testing Strategy

- **DOC-002**: Table-driven tests for `non_empty_file` validator: missing file, empty file, non-empty file, relative/absolute paths
- **Full suite**: `go test ./...` to verify no regressions from contract.go changes
- **Manual verification**: Review each modified doc section for accuracy against code
