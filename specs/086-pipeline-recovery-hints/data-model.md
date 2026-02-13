# Data Model: Pipeline Recovery Hints

**Feature Branch**: `086-pipeline-recovery-hints`
**Date**: 2026-02-13

## New Types

### RecoveryHint

A single recovery action suggested to the user after a pipeline step failure.

```go
// Package: internal/recovery (new package)

// HintType identifies the category of recovery hint.
type HintType string

const (
    HintResume    HintType = "resume"    // --from-step resume command
    HintForce     HintType = "force"     // --from-step --force variant
    HintWorkspace HintType = "workspace" // Workspace path for inspection
    HintDebug     HintType = "debug"     // Suggestion to use --debug
)

// RecoveryHint represents a single suggested recovery action.
type RecoveryHint struct {
    Label   string   `json:"label"`   // Human-readable description (e.g., "Resume from failed step")
    Command string   `json:"command"` // Shell command to execute (e.g., "wave run feature 'add auth' --from-step implement")
    Type    HintType `json:"type"`    // Hint category for filtering/ordering
}
```

### RecoveryBlock

An ordered collection of hints for a specific failure, plus context metadata.

```go
// RecoveryBlock holds all recovery hints for a single step failure.
type RecoveryBlock struct {
    PipelineName    string         // Pipeline that failed
    StepID          string         // Step that failed
    Input           string         // Original input string
    WorkspacePath   string         // Path pattern to workspace directory
    ErrorClass      ErrorClass     // Classification of the failure
    Hints           []RecoveryHint // Ordered list of hints
}
```

### ErrorClass

The classification of a pipeline failure, determining which hints to show.

```go
// ErrorClass categorizes a pipeline failure for hint selection.
type ErrorClass string

const (
    ClassContractValidation ErrorClass = "contract_validation"
    ClassSecurityViolation  ErrorClass = "security_violation"
    ClassRuntimeError       ErrorClass = "runtime_error"
    ClassUnknown            ErrorClass = "unknown"
)
```

### StepError (new, in internal/pipeline)

A structured error type that preserves the failed step ID in the error chain,
replacing the current `fmt.Errorf("step %q failed: %w", ...)` pattern.

```go
// Package: internal/pipeline

// StepError wraps a step execution error with the step ID for programmatic access.
type StepError struct {
    StepID string // ID of the step that failed
    Err    error  // Underlying error
}

func (e *StepError) Error() string {
    return fmt.Sprintf("step %q failed: %v", e.StepID, e.Err)
}

func (e *StepError) Unwrap() error {
    return e.Err
}
```

## Modified Types

### event.Event (extended)

Add `RecoveryHints` field for JSON output mode.

```go
// In internal/event/emitter.go, add to Event struct:

RecoveryHints []RecoveryHint `json:"recovery_hints,omitempty"` // Recovery hints on failure
```

Note: `RecoveryHint` here refers to a simple struct with `Label`, `Command`, `Type` fields
that mirrors `recovery.RecoveryHint`. To avoid a circular import between `event` and `recovery`,
define a `RecoveryHintJSON` struct directly in the `event` package:

```go
// RecoveryHintJSON is the JSON-serializable representation of a recovery hint.
type RecoveryHintJSON struct {
    Label   string `json:"label"`
    Command string `json:"command"`
    Type    string `json:"type"`
}
```

## Entity Relationships

```
runRun() error path
    │
    ├── execErr: error
    │   └── unwrap → StepError{StepID, Err}
    │                    └── unwrap → *contract.ValidationError
    │                              OR *security.SecurityValidationError
    │                              OR generic error
    │
    ├── ClassifyError(err) → ErrorClass
    │
    ├── BuildRecoveryBlock(pipelineName, input, stepID, runID, errorClass) → RecoveryBlock
    │   ├── Always: RecoveryHint{Type: "resume"}
    │   ├── If contract_validation: RecoveryHint{Type: "force"}
    │   ├── Always: RecoveryHint{Type: "workspace"}
    │   └── If runtime_error or unknown: RecoveryHint{Type: "debug"}
    │
    └── Render:
        ├── text/auto/quiet mode → FormatRecoveryBlock() → stderr
        └── json mode → populate Event.RecoveryHints → stdout via emitter
```

## Workspace Path Pattern

```
.wave/workspaces/<runID>/<stepID>/
```

Where:
- `runID` = `pipeline.GenerateRunID(pipelineName, hashLength)` — available in `runRun()`
- `stepID` = extracted from `StepError` via `errors.As()`
