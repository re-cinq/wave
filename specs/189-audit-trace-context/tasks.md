# Tasks

## Phase 1: Extend Audit Logger Interface and Implementation

- [X] Task 1.1: Add `LogStepStart`, `LogStepEnd`, `LogContractResult` methods to `AuditLogger` interface in `internal/audit/logger.go`
- [X] Task 1.2: Implement `LogStepStart` on `TraceLogger` — writes `[STEP_START]` line with pipeline, step, persona, and injected artifact names (all scrubbed)
- [X] Task 1.3: Implement `LogStepEnd` on `TraceLogger` — writes `[STEP_END]` line with status, duration, exit_code, output_bytes, tokens_used, and optional error (all scrubbed)
- [X] Task 1.4: Implement `LogContractResult` on `TraceLogger` — writes `[CONTRACT]` line with contract type and result (pass/fail/soft_fail/skip)

## Phase 2: Integrate with Pipeline Executor

- [X] Task 2.1: Add `LogStepStart` call at the beginning of `runStepExecution` in `internal/pipeline/executor.go` — after persona/artifacts resolved, before adapter run. Include injected artifact names from `step.Memory.InjectArtifacts`
- [X] Task 2.2: Add `LogStepEnd` calls at all exit paths of `runStepExecution` — success path (after contract validation), and each error return (adapter failure, rate limit, contract hard failure). Compute output_bytes from stdout length
- [X] Task 2.3: Add `LogContractResult` call after `contract.Validate` in `runStepExecution` — log pass, fail, or soft_fail based on validation outcome. Log "skip" when no contract is configured

## Phase 3: Update Tests

- [X] Task 3.1: Add unit tests for `LogStepStart`, `LogStepEnd`, `LogContractResult` in `internal/audit/logger_test.go` — verify trace format, field presence, and credential scrubbing on error messages [P]
- [X] Task 3.2: Update mock `AuditLogger` implementations in `internal/pipeline/executor_test.go` and any other test files to satisfy the extended interface (add no-op implementations of new methods) [P]
- [X] Task 3.3: Run `go test ./...` to verify all tests pass with the interface change

## Phase 4: Validation

- [X] Task 4.1: Run `go test -race ./...` to verify no data races in new logging code
- [X] Task 4.2: Verify backward compatibility — confirm `[TOOL]` and `[FILE]` trace entries are unchanged in format
