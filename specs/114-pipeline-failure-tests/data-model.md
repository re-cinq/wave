# Data Model: Pipeline Failure Mode Test Coverage

**Branch**: `114-pipeline-failure-tests` | **Date**: 2026-02-20

## Overview

This document defines the data structures and relationships relevant to pipeline failure mode testing. These entities already exist in the codebase; this document maps them to the test scenarios.

## Core Entities

### 1. ValidationError

**Location**: `internal/contract/contract.go:34-42`

**Purpose**: Represents a contract validation failure with structured details for debugging and retry decisions.

```go
type ValidationError struct {
    ContractType string   // "json_schema", "typescript_interface", "test_suite", etc.
    Message      string   // Human-readable error description
    Details      []string // Specific validation failures (e.g., "field 'name' is required")
    Retryable    bool     // Whether the step should be retried
    Attempt      int      // Current attempt number
    MaxRetries   int      // Maximum configured retries
}
```

**Test Relevance**:
- User Story 1 (Contract Schema Mismatch): Validates `ContractType` and `Details` are populated correctly
- User Story 7 (False-Positive Prevention): Validates that malformed input produces non-nil error

**Invariants**:
- `ContractType` must match the validator that produced the error
- `Details` should be non-empty for schema validation failures
- `Attempt <= MaxRetries` always

### 2. StepError

**Location**: `internal/adapter/errors.go`

**Purpose**: Wraps adapter execution failures with classification for remediation guidance.

```go
type StepError struct {
    FailureReason string // Classification constant
    Cause         error  // Underlying error (supports errors.Unwrap)
    TokensUsed    int    // Tokens consumed before failure
    Subtype       string // Adapter-specific subtype
    Remediation   string // User-facing remediation guidance
}
```

**Failure Reason Constants**:
```go
const (
    FailureReasonTimeout            = "timeout"
    FailureReasonContextExhaustion  = "context_exhaustion"
    FailureReasonRateLimit          = "rate_limit"
    FailureReasonGeneralError       = "general_error"
)
```

**Test Relevance**:
- User Story 2 (Timeout): Validates `FailureReason == "timeout"` on deadline exceeded
- User Story 6 (Exit Code): Validates non-zero exit creates `StepError`

**Invariants**:
- `FailureReason` must be one of the defined constants
- `Remediation` is derived from `FailureReason` via `remediationForReason()`

### 3. AdapterResult

**Location**: `internal/adapter/adapter.go:59-67`

**Purpose**: Encapsulates the complete result of an adapter execution.

```go
type AdapterResult struct {
    ExitCode      int       // Process exit code (0 = success)
    Stdout        io.Reader // Captured standard output
    TokensUsed    int       // Estimated token consumption
    Artifacts     []string  // Produced artifact paths
    ResultContent string    // Extracted response content
    FailureReason string    // Classification if failed
    Subtype       string    // NDJSON result subtype
}
```

**Test Relevance**:
- User Story 6 (Exit Code): `ExitCode != 0` should result in step failure
- User Story 7 (False-Positive): `ExitCode` must be checked before contract validation

**Invariants**:
- `ExitCode == 0` implies no process-level failure
- `FailureReason` is set when `ExitCode != 0` or context deadline exceeded

### 4. QualityViolation

**Location**: `internal/contract/quality_gate.go`

**Purpose**: Represents a quality gate threshold violation.

```go
type QualityViolation struct {
    Gate      string   // Gate name (e.g., "coverage", "complexity")
    Severity  string   // "error", "warning", "info"
    Message   string   // Human-readable description
    Score     int      // Achieved score (0-100)
    Threshold int      // Required minimum
    Details   []string // Specific violations
}
```

**Test Relevance**:
- User Story 1 (Contract Validation): Quality gates are optional validation layer
- Success Criteria SC-002/SC-003: Coverage gates enforce 80% thresholds

**Invariants**:
- `Score < Threshold` for any violation
- `Severity` must be "error", "warning", or "info"

## Supporting Types

### ContractConfig

**Location**: `internal/contract/contract.go:10-32`

```go
type ContractConfig struct {
    Type        string   // Validator type
    Source      string   // Inline schema or reference
    Schema      string   // JSON Schema content
    SchemaPath  string   // Path to schema file
    Command     string   // Test suite command
    CommandArgs []string // Test suite arguments
    Dir         string   // Working directory
    MustPass    bool     // Whether failure blocks pipeline
    MaxRetries  int      // Retry count before failure
    // ... quality gate and recovery settings
}
```

**Test Relevance**:
- User Story 1: `Type == "json_schema"` with `Schema` or `SchemaPath`
- User Story 3: `Command` execution with exit code checking

### PipelineStatus

**Location**: `internal/pipeline/executor.go:35-44`

```go
type PipelineStatus struct {
    ID             string     // Runtime ID with hash suffix
    PipelineName   string     // Logical pipeline name
    State          string     // "pending", "running", "completed", "failed", "retrying"
    CurrentStep    string     // Active step ID
    CompletedSteps []string   // Successfully finished steps
    FailedSteps    []string   // Steps that failed
    StartedAt      time.Time
    CompletedAt    *time.Time
}
```

**Test Relevance**:
- All failure scenarios should result in `State == "failed"`
- `FailedSteps` should contain the step ID that caused failure

## Entity Relationships

```
Pipeline
    └── Step[]
            ├── Persona (permissions, tools)
            ├── HandoverConfig
            │       └── ContractConfig
            │               └── QualityGateConfig[]
            ├── Memory
            │       └── InjectArtifacts[]
            └── OutputArtifacts[]

Execution
    ├── PipelineStatus
    ├── ArtifactPaths (map[stepID:artifactName] → path)
    └── Results (map[stepID] → map[string]interface{})

Adapter Execution
    ├── AdapterRunConfig → AdapterResult
    └── On failure → StepError
                         └── wraps → Cause (error)

Contract Validation
    ├── ContractConfig → Validator.Validate()
    └── On failure → ValidationError
                         └── contains → Details[]
```

## Error Propagation Flow

```
1. Adapter execution
   └── context.DeadlineExceeded → StepError{FailureReason: "timeout"}
   └── exitCode != 0 → StepError{FailureReason: "general_error"}
   └── SIGKILL (137) → StepError{FailureReason: "general_error"}

2. Artifact injection
   └── missing artifact → error (NEEDS IMPLEMENTATION)
   └── path is directory → error (NEEDS IMPLEMENTATION)

3. Contract validation
   └── schema mismatch → ValidationError{Details: [...]}
   └── test suite fails → ValidationError{ContractType: "test_suite"}
   └── quality gate fails → ValidationError{Details: QualityViolation[]}

4. Pipeline orchestration
   └── any step error → PipelineStatus{State: "failed", FailedSteps: [...]}
```

## Test Data Requirements

### JSON Schema Test Fixtures

```json
// Valid schema for testing
{
  "type": "object",
  "properties": {
    "status": {"type": "string"},
    "count": {"type": "integer", "minimum": 0}
  },
  "required": ["status", "count"],
  "additionalProperties": false
}

// Invalid inputs to test rejection
{"status": 123, "count": "not-a-number"}  // Type mismatch
{"status": "ok"}                           // Missing required
{"status": "ok", "count": 5, "extra": 1}   // Additional properties
{"status": "ok", "count": -1}              // Minimum violation
[1, 2, 3]                                  // Array instead of object
{not valid json}                           // Malformed
```

### Mock Adapter Configurations

```go
// Timeout scenario
AdapterRunConfig{
    Adapter: "sleep",
    Prompt:  "60",
    Timeout: 100 * time.Millisecond,
}

// Non-zero exit scenario
AdapterRunConfig{
    Adapter: "false", // Always returns exit code 1
}

// SIGKILL scenario (exit code 137)
// Requires process that ignores SIGTERM
```

## Implementation Notes

### Missing Artifact Detection (CL-003)

Current implementation silently ignores missing artifacts. Required change:

```go
func (e *DefaultPipelineExecutor) injectArtifacts(...) error {
    var missing []string
    for _, ref := range step.Memory.InjectArtifacts {
        key := ref.Step + ":" + ref.Artifact
        if _, ok := execution.ArtifactPaths[key]; !ok {
            // Also check fallback
            if _, exists := execution.Results[ref.Step]; !exists {
                missing = append(missing, fmt.Sprintf("%s from step %s", ref.Artifact, ref.Step))
            }
        }
    }
    if len(missing) > 0 {
        return fmt.Errorf("missing artifacts: %s", strings.Join(missing, ", "))
    }
    // ... rest of injection logic
}
```

### Exit Code Precedence (FR-002)

Ensure exit code is checked before contract validation:

```go
// In step execution
result, err := runner.Run(ctx, cfg)
if err != nil {
    return wrapStepError(err)
}
if result.ExitCode != 0 {
    return &StepError{
        FailureReason: FailureReasonGeneralError,
        Cause:         fmt.Errorf("adapter exited with code %d", result.ExitCode),
    }
}
// Only then run contract validation
```
