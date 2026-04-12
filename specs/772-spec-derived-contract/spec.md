# WS2.1 Spec-Derived Test Contract

**Issue**: [re-cinq/wave#772](https://github.com/re-cinq/wave/issues/772)
**Labels**: epic
**Author**: nextlevelshit
**State**: CLOSED

## Description

Create a new contract type `spec_derived_test` in `internal/contract/spec_derived.go`. This contract takes a specification artifact (from a spec step) and independently generates test cases, then validates the implementation against those tests. The test author persona must be explicitly different from the implementer persona.

## Requirements

1. Add `SpecDerivedConfig` to `internal/contract/contract.go` with fields:
   - `spec_artifact` (string) — reference to the specification artifact
   - `test_persona` (string) — persona that generates tests (must differ from implementer)
   - `implementation_step` (string) — step ID of the implementation to validate

2. Create `internal/contract/spec_derived.go` implementing the `spec_derived_test` contract type

3. Register `spec_derived_test` in the contract validator registry (`NewValidator` switch)

4. Include unit tests

## Acceptance Criteria

- [ ] `SpecDerivedConfig` struct exists in `contract.go` with the three required fields
- [ ] `spec_derived_test` contract type is registered in `NewValidator`
- [ ] Validator enforces that `test_persona` differs from the implementer persona
- [ ] Validator reads the spec artifact and uses it for test generation context
- [ ] Unit tests cover: missing spec artifact, empty persona, persona equality check, successful validation path
- [ ] All existing tests continue to pass (`go test ./internal/contract/...`)
