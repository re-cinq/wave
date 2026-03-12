# Tasks

## Phase 1: Setup

- [X] Task 1.1: Audit existing failure-mode tests across `internal/pipeline/executor_test.go`, `contract_integration_test.go`, and `internal/contract/contract_test.go` to confirm coverage gaps
- [X] Task 1.2: Create `internal/pipeline/failure_modes_test.go` with package declaration, imports, and shared test helpers (reuse `createTestManifest`, `newTestEventCollector`, `stepAwareAdapter`)

## Phase 2: Core Implementation — Pipeline Failure Mode Tests

- [X] Task 2.1: `TestFailureMode_ContractSchemaMismatch` — Mock adapter writes output not matching JSON schema with `must_pass: true`. Assert: `Execute()` returns error containing "contract validation failed", event with state "contract_failed" emitted, no "completed" event for the step [P]
- [X] Task 2.2: `TestFailureMode_StepTimeout` — Mock adapter with `SimulatedDelay` exceeding context deadline. Assert: `Execute()` returns error, error wraps `context.DeadlineExceeded`, pipeline state is "failed" [P]
- [X] Task 2.3: `TestFailureMode_MissingArtifact` — Two-step pipeline where step2 requires artifact from step1 but step1 produces no output artifacts. Assert: `Execute()` returns error containing "required artifact" and "not found" [P]
- [X] Task 2.4: `TestFailureMode_MalformedArtifact` — Step produces artifact but content is empty/truncated. Downstream step with contract validation fails. Assert: validation error propagates as pipeline failure [P]
- [X] Task 2.5: `TestFailureMode_WorkspaceCorruption` — Use a custom workspace manager mock that returns error on Create(). Assert: `Execute()` returns error containing "workspace" [P]
- [X] Task 2.6: `TestFailureMode_NonZeroExitCode` — Mock adapter returns `ExitCode: 1` but no error (simulating Claude Code JS crash after tool calls). With a strict contract, pipeline should still fail if artifact is missing/invalid. Without contract, pipeline should emit warning but complete [P]
- [X] Task 2.7: `TestFailureMode_AdapterError` — Mock adapter returns error via `WithFailure()`. Assert: `Execute()` returns error, event with "failed" state emitted, error wraps adapter error

## Phase 3: Contract False-Positive Detection Tests

- [X] Task 3.1: Create `internal/contract/false_positive_test.go` with table-driven tests for JSON schema validator edge cases [P]
- [X] Task 3.2: `TestFalsePositive_TruncatedJSON` — Validate that truncated JSON (`{"name": "te`) is rejected, not silently accepted
- [X] Task 3.3: `TestFalsePositive_WrongTypesMasquerading` — String "123" for integer field, "true" for boolean, nested object where array expected
- [X] Task 3.4: `TestFalsePositive_EmptyObjectPassesRequired` — Empty `{}` must fail when required fields are specified
- [X] Task 3.5: `TestFalsePositive_NullValues` — `{"name": null}` should fail when `name` is required string type
- [X] Task 3.6: `TestFalsePositive_ExtraFieldsWithStrictSchema` — Object with `additionalProperties: false` must reject extra fields

## Phase 4: Integration Validation

- [X] Task 4.1: Run `go test -race ./internal/pipeline/...` to verify all new tests pass alongside existing tests
- [X] Task 4.2: Run `go test -race ./internal/contract/...` to verify contract tests
- [X] Task 4.3: Run `go test -race ./...` full suite to confirm no regressions
- [X] Task 4.4: Verify test count increase — new tests should add at minimum 13 test functions (7 pipeline + 6 contract)
