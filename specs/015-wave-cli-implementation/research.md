# Research: Wave CLI Implementation (Hardening)

**Branch**: `015-wave-cli-implementation`
**Date**: 2026-02-02

## R1: Test Strategy for CLI Commands

**Decision**: Use Cobra's testing utilities with testable command execution pattern

**Rationale**: The existing `cmd/wave/commands/` package has no tests. Cobra
provides `cmd.SetArgs()` and `cmd.Execute()` for testing commands without
spawning processes. Tests should:
1. Create a temp directory with test fixtures (wave.yaml, pipelines)
2. Set command args programmatically
3. Capture stdout/stderr
4. Assert exit codes and output content

**Implementation approach**:
- Create `cmd/wave/commands/commands_test.go` as the main test file
- Use `testify/assert` for assertions (already in go.mod)
- Create test fixtures in `testdata/` subdirectory
- Test both success paths and error conditions

## R2: State Store Test Strategy

**Decision**: Use in-memory SQLite for unit tests, file-based for integration tests

**Rationale**: `internal/state/store.go` has no tests. SQLite supports
`:memory:` databases that are fast and isolated. Tests should cover:
1. Pipeline state CRUD operations
2. Step state transitions
3. Concurrent access from matrix workers
4. Resume from various states

**Implementation approach**:
- Create `internal/state/store_test.go`
- Use `:memory:` for fast unit tests
- Use temp file DB for concurrency tests
- Test all state transitions in the minimal state machine

## R3: Error Message Quality Standards

**Decision**: Structured errors with context using wrapped errors

**Rationale**: Current error messages lack context. Production-ready errors need:
1. File paths for configuration errors
2. Line numbers for YAML parsing errors
3. Step IDs for pipeline execution errors
4. Actionable suggestions where possible

**Implementation approach**:
- Use `fmt.Errorf` with `%w` for error wrapping
- Add context at each layer (command → executor → adapter)
- Include suggestions in validation errors (e.g., "run wave init first")
- Keep errors human-readable while being grep-able

## R4: Graceful Degradation for Missing Tools

**Decision**: Feature detection with fallback behavior

**Rationale**: TypeScript contract validation requires `tsc`, but it may not
be installed. The system should:
1. Check for tool availability at validation time
2. Emit a warning (not error) if tool is missing
3. Fall back to syntax-only validation (parse, don't compile)
4. Log which validation level was actually performed

**Implementation approach**:
- Add `exec.LookPath("tsc")` check in `internal/contract/typescript.go`
- Return degraded validation result with warning
- Document degradation in progress events
- Same pattern for other optional tools

## R5: Subprocess Timeout Implementation

**Decision**: Process group timeout with graceful shutdown

**Rationale**: Hanging subprocesses must be killed. Current implementation
may not handle this correctly. The timeout strategy:
1. Set deadline using `context.WithTimeout`
2. Create process group with `Setpgid: true`
3. On timeout, send SIGTERM to process group
4. Wait 5 seconds, then SIGKILL
5. Mark step as Failed/Retrying

**Implementation approach**:
- Verify `internal/adapter/adapter.go` uses process groups
- Add graceful shutdown sequence (TERM → wait → KILL)
- Test timeout behavior with mock adapter that hangs
- Ensure workspace cleanup on timeout

## R6: Concurrent Pipeline Execution

**Decision**: Allow concurrent pipelines with workspace isolation

**Rationale**: Multiple `wave run` invocations may happen simultaneously
(CI parallelism). Each pipeline instance needs:
1. Unique pipeline_id (UUID)
2. Isolated workspace directory
3. Separate SQLite database connection
4. Non-conflicting state entries

**Implementation approach**:
- Verify pipeline_id generation is UUID-based
- Verify workspace paths include pipeline_id
- Test concurrent execution with race detector
- Document concurrency behavior

## R7: Credential Scrubbing Patterns

**Decision**: Regex-based scrubbing with configurable patterns

**Rationale**: Audit logs must not contain credentials. The scrubbing needs:
1. Known patterns: ANTHROPIC_API_KEY, OPENAI_API_KEY, AWS_*, etc.
2. Generic patterns: *_TOKEN, *_SECRET, *_KEY, *_PASSWORD
3. Scrub both keys and values
4. Apply to audit logs and error messages

**Implementation approach**:
- Verify `internal/audit/logger.go` has scrubbing
- Add comprehensive pattern list
- Test with various credential formats
- Document which patterns are scrubbed

## R8: CLI Flag Completion

**Decision**: Add missing flags to match spec requirements

**Rationale**: The spec defines flags not yet implemented:
- `wave validate --verbose` (verbose output)
- `wave do --save <path>` (save generated pipeline)
- `wave clean --keep-last <n>` (preserve recent workspaces)
- `wave clean --dry-run` (preview deletions)
- `wave run --dry-run` (preview execution plan)

**Implementation approach**:
- Audit each command against spec acceptance scenarios
- Add missing flags with Cobra flag definitions
- Implement flag behavior
- Test each flag combination
