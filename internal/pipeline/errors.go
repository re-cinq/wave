package pipeline

import "fmt"

// StepExecutionError wraps a step execution error with the step ID for programmatic access.
// It preserves the same error message format as the previous fmt.Errorf pattern
// while making the step ID extractable via errors.As().
type StepExecutionError struct {
	StepID string
	Err    error
}

func (e *StepExecutionError) Error() string {
	return fmt.Sprintf("step %q failed: %v", e.StepID, e.Err)
}

func (e *StepExecutionError) Unwrap() error {
	return e.Err
}

// ContractRejectionError signals a *design rejection* — a contract with
// on_failure: rejected fired because the persona output deliberately marked
// the work as non-actionable (e.g. fetch-assess setting `implementable:
// false` because the issue is already implemented or superseded).
//
// This is structurally a step failure (the contract did not pass), but
// semantically it is the persona telling the orchestrator "there is no work
// here." Callers up the stack (the executor, the CLI's runOnce, the webui
// status renderer) detect this error via errors.As and route it to the
// dedicated `rejected` terminal state instead of `failed`. The CLI exits 0
// because there was no malfunction — the answer was simply "no".
type ContractRejectionError struct {
	StepID       string
	ContractType string
	Reason       string // human-readable summary of the rejection (validator error message)
}

func (e *ContractRejectionError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("step %q rejected by contract (%s): %s", e.StepID, e.ContractType, e.Reason)
	}
	return fmt.Sprintf("step %q rejected by contract (%s)", e.StepID, e.ContractType)
}

// gateAbortError is returned when a gate step selects a choice targeting _fail,
// signaling that the pipeline should abort.
type gateAbortError struct {
	StepID string
	Choice string
}

func (e *gateAbortError) Error() string {
	return fmt.Sprintf("gate %q aborted with choice %q", e.StepID, e.Choice)
}
