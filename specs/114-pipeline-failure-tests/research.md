# Research: Pipeline Failure Mode Test Coverage

**Branch**: `114-pipeline-failure-tests` | **Date**: 2026-02-20
**Input**: Feature specification from `specs/114-pipeline-failure-tests/spec.md`

## Executive Summary

This research analyzes the existing Wave codebase to understand current failure handling, test patterns, and coverage gaps. The goal is to inform implementation of comprehensive failure mode tests as specified in Issue #114.

## Current State Analysis

### Coverage Metrics (Baseline)

| Package | Current Coverage | Target (SC-002/003) |
|---------|------------------|---------------------|
| `internal/contract/` | 53.6% | 80% |
| `internal/pipeline/` | 65.8% | 80% |

**Gap Analysis**: Contract package needs +26.4 percentage points, pipeline package needs +14.2 percentage points.

### Existing Test Patterns

#### Table-Driven Tests
The codebase consistently uses table-driven tests with clear structure:
- `contract_test.go`: `TestJSONSchemaValidator_ValidationFailure_TableDriven` (lines 64-165)
- `errors_test.go`: `TestClassifyFailure` (lines 10-98)
- `dag_test.go`: Comprehensive cycle detection and topological sort tests

**Pattern**:
```go
tests := []struct {
    name          string
    // inputs
    expectError   bool
    errorContains string
}{...}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {...})
}
```

#### Integration Tests
Integration tests use build tags (`//go:build integration`) and test full execution paths:
- `adapter_test.go`: Timeout, non-zero exit, process group tests
- `error_handling_integration_test.go`: Pipeline execution with validation

### Failure Handling Analysis

#### 1. Contract Schema Mismatch (User Story 1)
**Location**: `internal/contract/jsonschema.go`

**Current Implementation**:
- `ValidationError` struct captures contract type, message, details, retryability
- JSON Schema validation uses `github.com/santhosh-tekuri/jsonschema/v6`
- Table-driven tests cover: type mismatch, missing required fields, `additionalProperties: false`

**Test Coverage Assessment**: Good coverage exists. Need to add:
- Boundary value tests (minimum/maximum constraints)
- Malformed JSON rejection (trailing commas, comments)
- Explicit type coercion prevention tests

#### 2. Step Timeout Handling (User Story 2)
**Location**: `internal/adapter/adapter.go:141-156`

**Current Implementation**:
```go
func killProcessGroup(process *os.Process) {
    _ = syscall.Kill(-process.Pid, syscall.SIGTERM)
    go func() {
        time.Sleep(3 * time.Second) // Hardcoded grace period
        _ = syscall.Kill(-process.Pid, syscall.SIGKILL)
    }()
}
```

**Key Finding**: Grace period is hardcoded at 3 seconds (confirmed in CL-001).

**Test Coverage Assessment**: `adapter_test.go` has comprehensive timeout tests (lines 1053-1402):
- Basic timeout, graceful vs forced termination
- Concurrent timeouts, sequential timeouts
- Output before timeout

**Gap**: No test for child process group termination verification.

#### 3. Missing Artifact Detection (User Story 3)
**Location**: `internal/pipeline/executor.go:1035-1089`

**Current Implementation**:
```go
func (e *DefaultPipelineExecutor) injectArtifacts(...) error {
    // Silently continues if artifact not found!
    if artifactPath, ok := execution.ArtifactPaths[key]; ok {
        // ... inject
    }
    // No error returned if missing
}
```

**Critical Gap**: Missing artifacts are **silently ignored**. This violates FR-005 and User Story 3.

**Required Implementation Change**:
- Accumulate all missing artifacts before returning error
- Report all missing artifacts in single error message (CL-003 resolution)

#### 4. Permission Denial (User Story 4)
**Location**: `internal/security/`, CLAUDE.md injection

**Current Implementation**:
- Deny rules projected into `settings.json` AND `CLAUDE.md` restriction section
- Permission enforcement handled by adapter (Claude Code)
- `internal/security/` provides path validation and sanitization

**Gap**: No integration tests verifying that denied tool calls result in step failure.

#### 5. Workspace Corruption (User Story 5)
**Location**: `internal/workspace/`

**Current Implementation**:
- Ephemeral workspace management exists
- No explicit corruption detection at step boundaries

**Gap**: No tests for workspace validation (directory exists, writable) at step boundaries.

#### 6. Non-Zero Exit Code (User Story 6)
**Location**: `internal/adapter/adapter.go:158-165`

**Current Implementation**:
```go
func exitCodeFromError(err error) int {
    if exitErr, ok := err.(*exec.ExitError); ok {
        if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
            return status.ExitStatus()
        }
    }
    return -1
}
```

**Test Coverage Assessment**: `TestProcessGroupRunner_Run_NonZeroExit` exists but doesn't verify exit code precedence over partial output.

**Gap**: Need test verifying exit code failure takes precedence over contract validation of partial output.

#### 7. Contract Validator False-Positives (User Story 7)
**Location**: `internal/contract/`

**Current Tests**:
- Type mismatch (string vs integer)
- Missing required fields
- `additionalProperties: false`

**Gaps**:
- Array vs object type coercion
- Boundary conditions (`minimum`, `maximum`)
- Malformed JSON with trailing commas/comments

### Error Types and Structures

#### ValidationError (`internal/contract/contract.go:34-42`)
```go
type ValidationError struct {
    ContractType string
    Message      string
    Details      []string
    Retryable    bool
    Attempt      int
    MaxRetries   int
}
```

#### StepError (`internal/adapter/errors.go`)
```go
type StepError struct {
    FailureReason string // timeout, context_exhaustion, rate_limit, general_error
    Cause         error
    TokensUsed    int
    Subtype       string
    Remediation   string
}
```

#### AdapterResult (`internal/adapter/adapter.go:59-67`)
```go
type AdapterResult struct {
    ExitCode      int
    Stdout        io.Reader
    TokensUsed    int
    Artifacts     []string
    ResultContent string
    FailureReason string
    Subtype       string
}
```

## Technology Decisions

### Decision 1: Test Organization
**Decision**: Unit tests in package `_test.go` files, integration tests with `//go:build integration` tag.
**Rationale**: Follows existing codebase pattern. Integration tests can be run separately with `-tags=integration`.
**Alternatives Rejected**: Separate `/tests/` directory (doesn't match existing structure).

### Decision 2: Mock Adapter Usage
**Decision**: Use `adapter.NewMockAdapter()` for unit tests, real adapters for integration tests.
**Rationale**: MockAdapter provides deterministic behavior for testing failure paths.
**Alternatives Rejected**: Always use real adapters (slow, non-deterministic).

### Decision 3: Coverage Measurement
**Decision**: Use `go test -cover` statement coverage per CL-002 resolution.
**Rationale**: Standard Go tooling, matches industry practice.
**Alternatives Rejected**: Branch coverage (requires additional tooling not in scope).

### Decision 4: Artifact Accumulation Fix
**Decision**: Modify `injectArtifacts()` to collect all missing artifacts before failing.
**Rationale**: CL-003 specifies tests should drive implementation (TDD approach).
**Alternatives Rejected**: Fail on first missing artifact (violates User Story 3.2).

## Test Implementation Strategy

### Phase 1: Contract Package (Target: 80% coverage)

| Test File | New Tests Needed |
|-----------|-----------------|
| `contract_test.go` | Boundary values, JSON coercion prevention |
| `jsonschema.go` | Malformed JSON rejection |
| `testsuite_test.go` | Exit code verification |

### Phase 2: Pipeline Package (Target: 80% coverage)

| Test File | New Tests Needed |
|-----------|-----------------|
| `executor.go` | Missing artifact accumulation (requires code change) |
| `error_handling_integration_test.go` | Exit code precedence, workspace validation |

### Phase 3: Adapter Package

| Test File | New Tests Needed |
|-----------|-----------------|
| `adapter_test.go` | Child process group termination |
| `errors_test.go` | SIGKILL (137) classification |

### Phase 4: Integration Tests

| Pipeline | Failure Modes to Test |
|----------|----------------------|
| `speckit-flow` | Contract failures, timeout |
| `gh-issue-implement` | Workspace failures, permission denial |
| `doc-sync` or `gh-issue-rewrite` | Missing artifact, exit code |

## Risk Assessment

### High Risk
- **Missing artifact silent failure**: Requires implementation change, not just tests
- **Coverage targets**: 26.4% increase for contract package is significant

### Medium Risk
- **Permission denial testing**: Depends on adapter behavior, may need mocking
- **Workspace corruption simulation**: Requires careful test setup/teardown

### Low Risk
- **Timeout tests**: Well-covered existing patterns
- **Exit code tests**: Straightforward additions

## Dependencies

- `github.com/santhosh-tekuri/jsonschema/v6` - JSON Schema validation
- `syscall` package - Process group management (SIGTERM/SIGKILL)
- `testing` package - Standard Go testing

## Recommendations

1. **Priority 1**: Fix `injectArtifacts()` to accumulate missing artifacts (blocks User Story 3)
2. **Priority 2**: Add false-positive prevention tests for JSON Schema validator
3. **Priority 3**: Add exit code precedence tests
4. **Priority 4**: Add workspace validation at step boundaries
5. **Priority 5**: Create integration test suite for named pipelines

## References

- `internal/contract/contract_test.go` - Existing validation tests
- `internal/adapter/adapter_test.go` - Timeout and process management tests
- `internal/pipeline/error_handling_integration_test.go` - Integration test patterns
- Constitution v2.1.0 Principle 13: Test Ownership for Core Primitives
