# Tasks

## Phase 1: Error Types and Classification

- [ ] Task 1.1: Create `internal/adapter/errors.go` with `StepError` type and classification constants
- [ ] Task 1.2: Add `FailureReason` field to `AdapterResult` in `internal/adapter/adapter.go`
- [ ] Task 1.3: Write unit tests for `StepError` classification and remediation in `internal/adapter/errors_test.go`

## Phase 2: NDJSON Result Parsing

- [ ] Task 2.1: Extend `parseOutput` in `internal/adapter/claude.go` to extract `subtype` from result events [P]
- [ ] Task 2.2: Extend `parseStreamLine` to capture `subtype` in `StreamEvent` [P]
- [ ] Task 2.3: Add `Subtype` field to `StreamEvent` in `internal/adapter/adapter.go` [P]
- [ ] Task 2.4: Write tests for subtype extraction in `internal/adapter/claude_test.go`

## Phase 3: Graceful Termination and Timeout Error Enhancement

- [ ] Task 3.1: Change `killProcessGroup` in `internal/adapter/adapter.go` from SIGKILL to SIGTERM with 3-second grace period
- [ ] Task 3.2: Parse buffered output on timeout in `ClaudeAdapter.Run` before returning error (capture token usage + subtype from partial output)
- [ ] Task 3.3: Return classified `StepError` from `ClaudeAdapter.Run` on timeout and context exhaustion instead of raw `ctx.Err()`
- [ ] Task 3.4: Write tests for graceful termination and timeout error classification

## Phase 4: Executor Integration and Event Enrichment

- [ ] Task 4.1: Add `FailureReason` and `Remediation` fields to `Event` struct in `internal/event/emitter.go`
- [ ] Task 4.2: Update `runStepExecution` in `internal/pipeline/executor.go` to unwrap `StepError` and emit enriched failure events with token usage and remediation
- [ ] Task 4.3: Reduce default relay compaction threshold from 80% to 70% in `internal/relay/relay.go`
- [ ] Task 4.4: Add context utilization percentage to progress events when token data available

## Phase 5: Testing and Validation

- [ ] Task 5.1: Write integration tests for error classification flow through executor [P]
- [ ] Task 5.2: Verify all existing tests pass with `go test ./...` [P]
- [ ] Task 5.3: Run tests with race detector `go test -race ./...`
