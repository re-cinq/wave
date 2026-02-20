# Feature Specification: Pipeline Failure Mode Test Coverage

**Feature Branch**: `114-pipeline-failure-tests`
**Created**: 2026-02-20
**Status**: Draft
**Input**: GitHub Issue #114 - Add integration tests covering pipeline failure modes and false-positive detection

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Contract Schema Mismatch Detection (Priority: P1)

A pipeline step produces JSON output that doesn't conform to the expected schema. The pipeline must detect this mismatch, report a meaningful error, and exit with a non-zero code rather than silently accepting malformed output.

**Why this priority**: Contract validation is the primary defense against false-positive pipeline success. If contracts pass malformed data, the entire system's reliability is compromised.

**Independent Test**: Can be fully tested by creating a step with a strict JSON schema, having the step produce non-conforming output, and verifying the pipeline fails with a clear validation error.

**Acceptance Scenarios**:

1. **Given** a pipeline step with a JSON schema contract requiring `{"status": "string", "count": "integer"}`, **When** the step outputs `{"status": 123, "count": "not-a-number"}`, **Then** the pipeline exits with non-zero code and error message identifies the type mismatches.
2. **Given** a pipeline step with a required field contract, **When** the step outputs JSON missing required fields, **Then** the pipeline fails with an error specifying which fields are missing.
3. **Given** a pipeline step with additional property restrictions, **When** the step outputs JSON with unexpected extra fields, **Then** the pipeline fails if `additionalProperties: false` is set.

---

### User Story 2 - Step Timeout Handling (Priority: P1)

A pipeline step exceeds its configured timeout duration. The pipeline must abort the step cleanly, terminate the underlying process, and exit with a non-zero code without leaving orphaned processes.

**Why this priority**: Runaway steps can block CI pipelines indefinitely and consume resources. Clean timeout handling is critical for production reliability.

**Independent Test**: Can be tested by configuring a step with a short timeout (e.g., 5 seconds), having it run a command that hangs or runs forever, and verifying clean termination.

**Acceptance Scenarios**:

1. **Given** a step configured with `timeout: 5s`, **When** the step execution exceeds 5 seconds, **Then** the pipeline emits a timeout event, terminates the adapter process, and exits with non-zero code.
2. **Given** a step that spawns child processes and times out, **When** the timeout triggers, **Then** all processes in the process group are terminated (no orphaned processes).
3. **Given** a step timeout during retry attempts, **When** the timeout occurs mid-retry, **Then** the pipeline does not attempt further retries and fails cleanly.

---

### User Story 3 - Missing Artifact Detection (Priority: P1)

A downstream step depends on an artifact that a previous step was supposed to produce but didn't. The pipeline must detect this missing dependency and fail with a clear error before attempting execution.

**Why this priority**: Missing artifacts cause confusing downstream failures. Early detection with clear errors saves debugging time and prevents cascading failures.

**Independent Test**: Can be tested by configuring step dependencies where the upstream step doesn't produce the expected artifact file, and verifying the downstream step fails at injection time.

**Acceptance Scenarios**:

1. **Given** step B depends on artifact "analysis.json" from step A, **When** step A completes without producing "analysis.json", **Then** step B fails at artifact injection with error identifying the missing artifact.
2. **Given** step B depends on multiple artifacts from different steps, **When** any required artifact is missing, **Then** the error identifies all missing artifacts, not just the first one.
3. **Given** an artifact path that resolves to a directory instead of a file, **When** artifact injection is attempted, **Then** the pipeline fails with a clear path type error.

---

### User Story 4 - Permission Denial Enforcement (Priority: P2)

A step attempts to use a tool or access a path that is explicitly denied in its persona permissions. The pipeline must block the action and fail the step rather than allowing unauthorized operations.

**Why this priority**: Permission enforcement is a security control. Violations should be hard failures, not warnings, to maintain the security model.

**Independent Test**: Can be tested by defining a persona with explicit tool denials (e.g., `deny: ["Bash(rm -rf /*)"]`), having the step attempt the denied action, and verifying rejection.

**Acceptance Scenarios**:

1. **Given** a persona with `deny: ["Bash(sudo *)"]`, **When** the step attempts to run `sudo apt-get install`, **Then** the tool call is rejected and the step fails.
2. **Given** a persona with path restrictions, **When** the step attempts to write outside allowed directories, **Then** the write is blocked and the step fails with a permission error.
3. **Given** multiple permission rules (deny and allow), **When** a step matches both a deny and allow pattern, **Then** the deny rule takes precedence (fail-secure).

---

### User Story 5 - Workspace Corruption Recovery (Priority: P2)

The workspace directory becomes corrupted or enters an invalid state during pipeline execution (e.g., unexpected deletion, permission changes, disk full). The pipeline must detect this and fail gracefully rather than producing undefined behavior.

**Why this priority**: Workspace issues are rare but catastrophic when they occur. Clean failure prevents data corruption and helps with debugging.

**Independent Test**: Can be tested by deleting or corrupting the workspace directory mid-execution (using test hooks) and verifying the pipeline detects the issue.

**Acceptance Scenarios**:

1. **Given** a multi-step pipeline in progress, **When** the workspace directory is deleted between steps, **Then** the next step fails with a workspace validation error.
2. **Given** a step writing artifacts, **When** the workspace becomes read-only mid-execution, **Then** the step fails with a clear I/O error rather than succeeding without artifacts.
3. **Given** a workspace with insufficient disk space, **When** artifact writing fails due to space, **Then** the error message identifies disk space as the issue.

---

### User Story 6 - Non-Zero Adapter Exit Code Handling (Priority: P2)

The underlying adapter CLI (e.g., Claude Code) exits with an error code indicating a crash or internal failure. The pipeline must not treat this as success even if partial output exists.

**Why this priority**: Adapter crashes can leave partial output that passes naive validation. Exit codes are a signal of execution integrity.

**Independent Test**: Can be tested by mocking the adapter to exit with various error codes and verifying pipeline behavior.

**Acceptance Scenarios**:

1. **Given** an adapter that exits with code 1, **When** no artifact is produced, **Then** the pipeline step fails with the exit code in the error.
2. **Given** an adapter that exits with code 1 but produces partial output, **When** contract validation runs, **Then** the exit code failure takes precedence over partial output validation.
3. **Given** an adapter killed by SIGKILL (exit code 137), **When** the step completes, **Then** the pipeline reports a process termination error.

---

### User Story 7 - Contract Validator False-Positive Prevention (Priority: P1)

The contract validator itself must not pass malformed output due to bugs in validation logic. This requires testing the validators with known-bad inputs to ensure they correctly reject them.

**Why this priority**: Validator bugs are the most dangerous failure mode - they create a false sense of security while allowing bad data through.

**Independent Test**: Can be tested by feeding known-invalid JSON to JSON schema validators and ensuring rejection, testing TypeScript validators with syntax errors, etc.

**Acceptance Scenarios**:

1. **Given** a JSON schema requiring `type: "object"`, **When** the validator receives a JSON array, **Then** validation fails (not passes due to type coercion bugs).
2. **Given** a JSON schema with `minimum: 1`, **When** the validator receives `{"value": 0}`, **Then** validation fails (boundary conditions work correctly).
3. **Given** malformed JSON with trailing commas or comments, **When** validation runs without recovery mode, **Then** validation fails (doesn't silently fix input).
4. **Given** a test suite contract with a failing test, **When** the validator runs, **Then** the contract fails (test exit codes are checked correctly).

---

### Edge Cases

- **Concurrent step failures**: When multiple parallel steps fail simultaneously, all failures should be collected and reported in the final error, not just the first one.
- **Retry exhaustion**: After max retries are exhausted, the step state should clearly indicate retry exhaustion with attempt count.
- **Context cancellation**: External cancellation (SIGINT, CI timeout) should trigger graceful shutdown with cleanup of all running steps.
- **Empty artifact content**: An empty file (0 bytes) should be treated as a valid artifact, distinguishable from a missing artifact.
- **Circular dependency detection**: DAG validation should detect cycles at pipeline load time, not at execution time.
- **Unicode/binary artifacts**: Artifact paths and content should handle UTF-8 correctly; binary artifacts should be copied verbatim.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: Pipeline execution MUST return a non-zero exit code when any contract validation fails with `must_pass: true`.
- **FR-002**: Pipeline execution MUST return a non-zero exit code when any step exits with an error (non-zero adapter exit code).
- **FR-003**: Contract validators MUST reject JSON that doesn't match the specified schema, including type mismatches, missing required fields, and constraint violations.
- **FR-004**: Step timeout MUST terminate the adapter process and all child processes within a 3-second grace period (SIGTERM followed by SIGKILL after grace period).
- **FR-005**: Missing artifact detection MUST occur before step execution begins, with clear identification of which artifacts are missing from which steps.
- **FR-006**: Permission denials MUST be enforced by the orchestrator layer (via settings.json and CLAUDE.md injection), and violations detected by the adapter MUST result in step failure.
- **FR-007**: Workspace validation MUST detect missing workspace directories at step boundaries.
- **FR-008**: All failure scenarios MUST emit structured events that can be captured for monitoring and debugging.
- **FR-009**: Test suite contracts MUST fail when the test command returns a non-zero exit code.
- **FR-010**: All tests MUST pass under `go test -race ./...` to ensure thread safety.

### Key Entities

- **ValidationError**: Represents a contract validation failure with contract type, message, details array, and retryability flag.
- **StepError**: Represents a step execution failure wrapping the underlying error with step ID and context.
- **AdapterResult**: Contains exit code, stdout, stderr, duration, and error information from adapter execution.
- **QualityViolation**: Represents a quality gate violation with gate name, severity, score, threshold, and suggestions.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All seven failure scenarios listed in the issue have corresponding tests in `tests/` or `internal/*/`.
- **SC-002**: Test coverage for `internal/contract/` package is at least 80% (measured by `go test -cover`).
- **SC-003**: Test coverage for `internal/pipeline/` package is at least 80% (measured by `go test -cover`).
- **SC-004**: Zero test failures when running `go test -race ./...` on the full test suite.
- **SC-005**: Integration tests exist for at least 3 of the 5 named pipelines (`gh-issue-rewrite`, `doc-sync`, `dead-code`, `speckit-flow`, `gh-issue-implement`).
- **SC-006**: No false-positive scenarios exist where a pipeline reports success when a failure condition was triggered (verified by adversarial test cases).
- **SC-007**: All tests complete within CI timeout limits (individual tests under 30 seconds, full suite under 10 minutes).

## Clarifications _(added by clarify step)_

### CL-001: Grace Period Configuration for Timeout Termination

**Question**: FR-004 specifies a "configurable" grace period (default 3 seconds), but the codebase shows this is hardcoded in `internal/adapter/adapter.go:153`. Should tests verify the current hardcoded behavior or expect configurability?

**Resolution**: Tests should verify the current hardcoded 3-second grace period behavior. The spec uses "configurable" aspirationally; actual configurability is out of scope for this test coverage issue. The grace period implementation in `killProcessGroup()` uses `time.Sleep(3 * time.Second)` before SIGKILL.

**Rationale**: The issue scope is test coverage, not new features. Making the grace period configurable would require manifest schema changes and is a separate enhancement.

### CL-002: Coverage Metric Type for SC-002/SC-003

**Question**: The 80% coverage thresholds don't specify line vs. statement vs. branch coverage.

**Resolution**: Coverage refers to **statement coverage** as reported by Go's standard `go test -cover` tool. This is the default and most commonly used metric in the Go ecosystem.

**Rationale**: Industry standard for Go projects. Branch coverage requires additional tooling (`go test -covermode=count` with external analysis) that isn't mentioned in the issue.

### CL-003: Multiple Missing Artifacts Accumulation

**Question**: User Story 3, Scenario 2 requires reporting "all missing artifacts, not just the first one." The current `injectArtifacts` implementation processes sequentially. Should tests expect this behavior or verify current behavior?

**Resolution**: Tests should verify the **desired behavior** (accumulating all missing artifacts). This may require implementation changes to `injectArtifacts()` to collect all errors before failing. The test should drive the implementation.

**Rationale**: TDD approach - write the test for the correct behavior, then fix the implementation if it fails. The current sequential approach is a bug, not a feature.

### CL-004: JSON Schema additionalProperties Default

**Question**: User Story 1, Scenario 3 mentions failing "if `additionalProperties: false` is set." What is the expected behavior when `additionalProperties` is not specified?

**Resolution**: Per JSON Schema specification (draft-07 and later), `additionalProperties` defaults to `true` (any additional properties allowed). Tests should:
1. Verify that schemas **without** `additionalProperties` allow extra fields
2. Verify that schemas **with** `additionalProperties: false` reject extra fields

**Rationale**: Wave uses `github.com/santhosh-tekuri/jsonschema/v6` which follows the JSON Schema specification. The spec scenario correctly states "if set", implying the default allows additional properties.

### CL-005: Named Pipeline Integration Test Distribution

**Question**: SC-005 requires "at least 3 of the 5 named pipelines" have integration tests. Is any combination of 3 acceptable, or are some pipelines higher priority?

**Resolution**: Any combination of 3 pipelines is acceptable, with preference for testing pipelines that exercise different failure modes:
- `speckit-flow` - multi-step with contracts (exercises contract failures)
- `gh-issue-implement` - worktree workspace (exercises workspace failures)
- Either `gh-issue-rewrite`, `doc-sync`, or `dead-code` for variety

**Rationale**: The goal is broad coverage of failure modes, not comprehensive pipeline testing. Three diverse pipelines provide sufficient confidence without excessive test maintenance burden.
