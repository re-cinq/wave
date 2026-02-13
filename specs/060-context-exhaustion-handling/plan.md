# Implementation Plan: Context Exhaustion Handling

## Objective

Distinguish between Go context timeouts and Claude Code context window exhaustion so that Wave provides accurate, actionable error messages instead of the generic `context deadline exceeded`.

## Approach

Implement a three-layer solution:

1. **Error classification** - Parse the NDJSON result event's `subtype` field and error content to classify failures into timeout, context exhaustion, or general error.
2. **Graceful process termination** - Replace SIGKILL with SIGTERM + grace period so the Claude Code process can flush its final result event before being killed on timeout.
3. **Actionable error messages** - Include remediation suggestions and token usage data in error messages.

## File Mapping

### Modified Files

| File | Changes |
|------|---------|
| `internal/adapter/adapter.go` | Add `StepError` type with error classification; change `killProcessGroup` from SIGKILL to SIGTERM+grace; add `FailureReason` field to `AdapterResult` |
| `internal/adapter/claude.go` | Parse result `subtype` field in `parseOutput`; parse buffered output on timeout before returning error; return classified `StepError` instead of raw `ctx.Err()` |
| `internal/adapter/claude_test.go` | Add tests for error classification, result subtype parsing, timeout output parsing |
| `internal/pipeline/executor.go` | Unwrap `StepError` to emit token usage on failure; include remediation in error events |
| `internal/event/emitter.go` | Add `FailureReason` and `Remediation` fields to `Event` struct |
| `internal/relay/relay.go` | Reduce default threshold from 80% to 70% |

### New Files

| File | Purpose |
|------|---------|
| `internal/adapter/errors.go` | `StepError` type, error classification constants, remediation text builder |
| `internal/adapter/errors_test.go` | Tests for `StepError` classification and remediation messages |

## Architecture Decisions

### AD-1: Error Classification via NDJSON Parsing (not exit codes)

**Decision**: Classify errors by parsing the `subtype` field in the stream-json `result` event, not by exit codes.

**Rationale**: Claude Code does not have dedicated exit codes for context exhaustion. The NDJSON result event includes `subtype` (`success`, `error_max_turns`, `error_during_execution`) and the result text may contain `prompt is too long`. This gives us reliable three-way classification.

### AD-2: SIGTERM with Grace Period (not SIGKILL)

**Decision**: Replace immediate SIGKILL with SIGTERM followed by a 3-second grace period, then SIGKILL.

**Rationale**: SIGKILL prevents Claude Code from flushing its final result event. With SIGTERM, the process can write its result (including usage data and subtype) before termination. Go 1.20+ `cmd.Cancel` and `cmd.WaitDelay` provide this pattern natively, but we keep manual process group management for backward compatibility and control.

### AD-3: StepError Type (not sentinel errors)

**Decision**: Create a structured `StepError` type with classification, token usage, and remediation fields.

**Rationale**: The executor needs to extract diagnostic data from adapter errors. A structured error type allows `errors.As()` unwrapping and carries all context needed for rich error reporting.

### AD-4: Relay Threshold Reduction to 70%

**Decision**: Lower the default relay compaction threshold from 80% to 70%.

**Rationale**: Claude Code auto-compacts at ~95% of the 200K context window. At 80%, Wave's compaction and Claude's internal compaction may race. At 70%, Wave triggers compaction well before Claude's internal limit, giving the relay system a chance to summarize before context pressure builds.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| SIGTERM may not be handled by Claude Code subprocess | Low | Medium | Falls back to SIGKILL after 3-second grace period |
| Result event parsing may miss new Claude Code output formats | Medium | Low | Use defensive parsing; fall back to "unknown" classification |
| Relay threshold change may cause more frequent compaction | Low | Low | Compaction is already best-effort; extra runs have minimal cost |
| `subtype` field may not always be present in result events | Medium | Low | Default to "unknown" when field missing; still better than current |

## Testing Strategy

### Unit Tests
- `TestStepErrorClassification` - Verify three-way classification from result subtype
- `TestStepErrorRemediation` - Verify remediation messages for each classification
- `TestParseOutputWithSubtype` - Verify subtype extraction from NDJSON result events
- `TestTimeoutOutputParsing` - Verify buffered output is parsed on timeout
- `TestGracefulTermination` - Verify SIGTERM + SIGKILL fallback behavior
- `TestRelayThresholdDefault` - Verify new 70% default

### Integration Tests
- `TestStepErrorPropagation` - Verify error classification flows through executor to events
- `TestTimeoutWithOutputCapture` - Verify token data captured even on timeout

### Existing Test Compatibility
- All existing `TestParseStreamLine` tests must continue passing
- All existing `TestParseOutput` behavior preserved (new fields are additive)
