# Work Items

## Phase 1: Core Implementation
- [X] Item 1.1: Fix Finding 1 — Convert introspection error warnings to ScopeViolation in `validator.go:151-153`
- [X] Item 1.2: Fix Finding 2 — Convert resolver error warnings to ScopeViolation in `validator.go:133-137`
- [X] Item 1.3: Fix Finding 3 — Add fine-grained PAT remediation hint in violation message

## Phase 2: Testing
- [X] Item 2.1: Update `TestValidatePersonas_IntrospectionFailure` to expect violations [P]
- [X] Item 2.2: Add new test for Bitbucket/unknown forge violation [P]
- [X] Item 2.3: Add new test for fine-grained PAT hint [P]
- [X] Item 2.4: Run all existing scope tests to verify no regressions

## Phase 3: Validation
- [X] Item 3.1: Run `go build` to verify compilation
- [X] Item 3.2: Run `go test ./internal/scope/...` to verify all tests pass
