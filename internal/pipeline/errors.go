package pipeline

import "fmt"

// StepError wraps a step execution error with the step ID for programmatic access.
// It preserves the same error message format as the previous fmt.Errorf pattern
// while making the step ID extractable via errors.As().
type StepError struct {
	StepID string
	Err    error
}

func (e *StepError) Error() string {
	return fmt.Sprintf("step %q failed: %v", e.StepID, e.Err)
}

func (e *StepError) Unwrap() error {
	return e.Err
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
