# Implementation Plan: Audit Trace Context Enrichment

## 1. Objective

Extend the `AuditLogger` interface and `TraceLogger` implementation to capture rich execution context (duration, exit code, output size, errors, contract results, token usage) at step boundaries, enabling post-mortem debugging and pipeline observability.

## 2. Approach

The strategy is to **extend** the existing audit logger with new structured log methods while preserving the current `[TOOL]` and `[FILE]` trace format for backward compatibility. New trace entry types (`[STEP_START]`, `[STEP_END]`, `[CONTRACT]`) will be added alongside existing entries. The executor will call these new methods at the appropriate lifecycle points.

Key design decisions:
- **Same log file, new event types**: New entries use the same key=value format as existing `[TOOL]` entries, just with different event type tags and additional fields.
- **Interface extension**: Add `LogStepStart`, `LogStepEnd`, and `LogContractResult` methods to `AuditLogger`. This is a breaking interface change, but the codebase has only two implementations (TraceLogger and any mocks in tests) and we're in prototype phase with no backward compatibility constraint.
- **Token split**: The adapter already surfaces `TokensIn`/`TokensOut` via `StreamEvent` at the result level, and `TokensUsed` (combined) via `AdapterResult`. We'll log `tokens_used` (combined) since that's what the executor has available. Splitting into `tokens_in`/`tokens_out` would require plumbing `parseOutputResult` details through `AdapterResult`, which is a separate concern.
- **Credential scrubbing**: All new fields pass through the existing `scrub()` method before writing.

## 3. File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/audit/logger.go` | **modify** | Extend `AuditLogger` interface with `LogStepStart`, `LogStepEnd`, `LogContractResult`; implement on `TraceLogger` |
| `internal/audit/logger_test.go` | **modify** | Add tests for new methods, verify credential scrubbing on error messages, verify trace format |
| `internal/pipeline/executor.go` | **modify** | Add audit log calls at step start, step end (success/failure), and contract validation result points in `runStepExecution` |
| `internal/pipeline/executor_test.go` | **modify** | Update mock logger implementations in tests to satisfy extended interface |
| `internal/pipeline/executor_enhanced.go` | **modify** | May need mock logger updates if tests use the enhanced executor path |

## 4. Architecture Decisions

### 4.1 New AuditLogger methods

```go
type AuditLogger interface {
    LogToolCall(pipelineID, stepID, tool, args string) error
    LogFileOp(pipelineID, stepID, op, path string) error
    LogStepStart(pipelineID, stepID, persona string, injectedArtifacts []string) error
    LogStepEnd(pipelineID, stepID, status string, duration time.Duration, exitCode int, outputBytes int, tokensUsed int, errMsg string) error
    LogContractResult(pipelineID, stepID, contractType, result string) error
    Close() error
}
```

### 4.2 Trace format examples

```
2026-02-02T18:17:04.933Z [STEP_START] pipeline=migrate step=impact-analysis persona=navigator artifacts=spec.md,plan.md
2026-02-02T18:17:04.933Z [TOOL] pipeline=migrate step=impact-analysis tool=adapter.Run args=persona=navigator prompt_len=449
2026-02-02T18:17:12.456Z [CONTRACT] pipeline=migrate step=impact-analysis type=json_schema result=pass
2026-02-02T18:17:12.457Z [STEP_END] pipeline=migrate step=impact-analysis status=success duration=7.523s exit_code=0 output_bytes=2048 tokens_used=761
```

On failure:
```
2026-02-02T18:17:12.457Z [STEP_END] pipeline=migrate step=impact-analysis status=failed duration=7.523s exit_code=1 output_bytes=0 tokens_used=761 error="adapter execution failed: context deadline exceeded"
```

### 4.3 Integration points in executor.go

- **`runStepExecution` entry**: Call `LogStepStart` after resolving persona and artifacts but before adapter execution.
- **`runStepExecution` exit (success)**: Call `LogStepEnd` with status="success" after contract validation.
- **`runStepExecution` exit (failure)**: Call `LogStepEnd` with status="failed" and error message at each error return path.
- **Contract validation**: Call `LogContractResult` after contract.Validate returns (pass, fail, or soft_fail).

## 5. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking `AuditLogger` interface | Compilation errors in tests using mock loggers | Update all mock implementations; grep for `AuditLogger` to find all usage sites |
| Error messages leaking credentials | Security violation | All error strings pass through `scrub()` before writing to trace file |
| Noisy trace output | Harder to read manually | New entries are additive; existing `[TOOL]` entries unchanged. Filtering by event type is straightforward |
| Token split unavailable | Issue example shows `tokens_in`/`tokens_out` but we only have `tokens_used` | Document this as a limitation; log combined total. Can be split in a follow-up if adapter exposes breakdown |

## 6. Testing Strategy

### Unit tests (internal/audit/logger_test.go)
- `TestLogStepStart`: Verify format, credential scrubbing on persona/artifact names
- `TestLogStepEnd_Success`: Verify all fields present, duration formatting
- `TestLogStepEnd_Failure`: Verify error message included and scrubbed
- `TestLogContractResult`: Verify contract type and result logged
- `TestLogStepEnd_CredentialScrubbing`: Verify secrets in error messages are redacted

### Integration tests (internal/pipeline/executor_test.go)
- Verify that existing executor tests still compile with updated mock logger
- Verify trace log calls are made at correct lifecycle points (if test infrastructure supports it)

### Manual validation
- Run a pipeline with `--debug` and inspect `.wave/traces/` output
- Confirm backward compatibility: existing `[TOOL]` format preserved alongside new entries
