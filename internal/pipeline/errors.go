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

// ReworkError wraps a step failure with rework branching context.
// It indicates that a step failed and rework was attempted (or could not be attempted).
type ReworkError struct {
	OriginalStepID string
	TargetStep     string
	TargetPipeline string
	ReworkDepth    int
	Err            error
}

func (e *ReworkError) Error() string {
	target := e.TargetStep
	if target == "" {
		target = "pipeline:" + e.TargetPipeline
	}
	return fmt.Sprintf("rework from step %q to %q failed (depth %d): %v",
		e.OriginalStepID, target, e.ReworkDepth, e.Err)
}

func (e *ReworkError) Unwrap() error {
	return e.Err
}
