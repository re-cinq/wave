# Implementation Plan: Spec-Derived Test Contract

## Objective

Add a new `spec_derived_test` contract type that takes a spec artifact and independently generates test cases via a separate persona, then validates the implementation against those tests. This enforces persona separation between test author and implementer.

## Approach

Follow the established contract pattern (similar to `agent_review`):
1. Add config fields to `ContractConfig` in `contract.go`
2. Create a new validator in `spec_derived.go` that requires an adapter runner (like `agent_review`)
3. Register in the `NewValidator` switch
4. The validator reads the spec artifact, builds a prompt for the test persona, runs the persona to generate tests, then executes those tests against the implementation

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/contract/contract.go` | modify | Add `SpecDerivedConfig` fields to `ContractConfig`, register `spec_derived_test` in `NewValidator` |
| `internal/contract/spec_derived.go` | create | New validator implementation |
| `internal/contract/spec_derived_test.go` | create | Unit tests for the new contract type |

## Architecture Decisions

1. **Config fields on `ContractConfig`**: Following the flat struct pattern used by all other contract types (source_diff, agent_review, llm_judge all add their fields directly to `ContractConfig`). The issue says "Add SpecDerivedConfig" — we'll add the fields with a comment grouping them, matching the existing pattern (e.g., `// source_diff contract fields`, `// Agent review settings`).

2. **Runner-dependent validator**: Like `agent_review`, `spec_derived_test` needs an adapter runner to invoke the test persona. `NewValidator` returns `nil` for it; the executor calls a dedicated function (similar to `ValidateWithRunner`).

3. **Persona separation enforcement**: The validator checks at validation time that the configured `test_persona` is not the same as the persona that ran the implementation step. This is a hard error, not a warning.

4. **Two-phase validation**: Phase 1 — invoke test persona with spec artifact to generate test cases. Phase 2 — execute those tests against the workspace. This mirrors the "independently generates test cases, then validates" requirement.

## Risks

| Risk | Mitigation |
|------|-----------|
| Circular import with adapter package | Use same pattern as `agent_review` — accept `adapter.AdapterRunner` interface |
| Test persona generates invalid tests | Structured output format with JSON schema for test definitions |
| Spec artifact missing or empty | Validate existence and non-empty before invoking persona |

## Testing Strategy

- **Unit tests** for config validation (missing fields, persona equality)
- **Unit tests** for spec artifact loading (missing file, path traversal)
- **Unit tests** for test result parsing
- **Unit tests** for `NewValidator` returning nil for `spec_derived_test`
- **Integration**: Existing `go test ./internal/contract/...` must pass
