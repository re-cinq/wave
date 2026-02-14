package adapter

import (
	"context"
	"fmt"
	"strings"
)

// Failure reason constants for error classification.
const (
	FailureReasonTimeout            = "timeout"
	FailureReasonContextExhaustion  = "context_exhaustion"
	FailureReasonRateLimit          = "rate_limit"
	FailureReasonGeneralError       = "general_error"
)

// StepError is a structured error type that carries diagnostic data
// from adapter execution failures. It enables the executor to extract
// token usage, failure classification, and remediation suggestions
// via errors.As().
type StepError struct {
	FailureReason string
	TokensUsed    int
	Subtype       string
	Remediation   string
	Cause         error
}

// Error implements the error interface with a descriptive message.
func (e *StepError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.FailureReason, e.Cause)
	}
	return e.FailureReason
}

// Unwrap returns the underlying cause for errors.Is/errors.As support.
func (e *StepError) Unwrap() error {
	return e.Cause
}

// NewStepError creates a StepError with the given classification and
// automatically fills in the remediation message.
func NewStepError(reason string, cause error, tokensUsed int, subtype string) *StepError {
	return &StepError{
		FailureReason: reason,
		TokensUsed:    tokensUsed,
		Subtype:       subtype,
		Remediation:   remediationFor(reason),
		Cause:         cause,
	}
}

// ClassifyFailure determines the failure reason from the result subtype,
// result content, and context error. This implements three-way classification:
//   - timeout: Go context deadline was exceeded
//   - context_exhaustion: Claude Code ran out of context window
//   - general_error: any other failure
func ClassifyFailure(subtype string, resultContent string, ctxErr error) string {
	if ctxErr == context.DeadlineExceeded {
		return FailureReasonTimeout
	}
	if subtype == "error_max_turns" {
		return FailureReasonContextExhaustion
	}
	if strings.Contains(strings.ToLower(resultContent), "prompt is too long") {
		return FailureReasonContextExhaustion
	}
	lowerContent := strings.ToLower(resultContent)
	if strings.Contains(lowerContent, "you've hit your limit") ||
		strings.Contains(lowerContent, "rate limit") ||
		strings.Contains(lowerContent, "too many requests") {
		return FailureReasonRateLimit
	}
	return FailureReasonGeneralError
}

// remediationFor returns an actionable remediation message for the given
// failure reason.
func remediationFor(reason string) string {
	switch reason {
	case FailureReasonTimeout:
		return "Consider increasing the step timeout with --timeout or breaking the task into smaller steps."
	case FailureReasonContextExhaustion:
		return "The context window was exhausted. Consider breaking the task into smaller steps or adjusting relay compaction thresholds (relay.token_threshold_percent)."
	case FailureReasonRateLimit:
		return "API rate limit reached. Wait for the limit to reset and retry."
	case FailureReasonGeneralError:
		return "Check the adapter output and logs for details."
	default:
		return "Check the adapter output and logs for details."
	}
}
