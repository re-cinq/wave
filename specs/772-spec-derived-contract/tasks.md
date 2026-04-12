# Tasks

## Phase 1: Config & Registration
- [X] Task 1.1: Add spec_derived_test config fields to `ContractConfig` in `internal/contract/contract.go` — add `SpecArtifact`, `TestPersona`, `ImplementationStep` fields with JSON tags and a comment block
- [X] Task 1.2: Register `spec_derived_test` in `NewValidator` switch — return nil (runner-dependent, like agent_review)

## Phase 2: Core Implementation
- [X] Task 2.1: Create `internal/contract/spec_derived.go` with `specDerivedValidator` struct [P]
- [X] Task 2.2: Implement `Validate()` method (returns error directing callers to use runner-based validation) [P]
- [X] Task 2.3: Implement `ValidateSpecDerived()` function — loads spec artifact, enforces persona separation, builds test generation prompt, invokes test persona, parses test output, runs tests
- [X] Task 2.4: Implement spec artifact loading with path traversal protection
- [X] Task 2.5: Implement persona separation check (test_persona != implementer persona)
- [X] Task 2.6: Implement test result parsing from persona output

## Phase 3: Testing
- [X] Task 3.1: Write unit tests for config validation — missing spec_artifact, missing test_persona, missing implementation_step [P]
- [X] Task 3.2: Write unit tests for persona separation enforcement — same persona rejected, different persona accepted [P]
- [X] Task 3.3: Write unit tests for spec artifact loading — missing file, path traversal blocked, valid file [P]
- [X] Task 3.4: Write unit tests for `NewValidator` returning nil for `spec_derived_test` type
- [X] Task 3.5: Write unit test for `Validate()` returning error (runner required)
- [X] Task 3.6: Run `go test ./internal/contract/...` to verify no regressions

## Phase 4: Polish
- [X] Task 4.1: Verify all existing contract tests still pass
- [X] Task 4.2: Run `go vet ./internal/contract/...`
