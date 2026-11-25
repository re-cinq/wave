package security

import (
	"fmt"
	"strings"
)

// SecurityValidationError represents a structured error for security validation failures
type SecurityValidationError struct {
	Type         string                 `json:"type"`
	Message      string                 `json:"message"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Retryable    bool                   `json:"retryable"`
	SuggestedFix string                 `json:"suggested_fix,omitempty"`
}

// Error implements the error interface
func (sve *SecurityValidationError) Error() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s] %s", sve.Type, sve.Message))

	if sve.SuggestedFix != "" {
		parts = append(parts, fmt.Sprintf("Fix: %s", sve.SuggestedFix))
	}

	return strings.Join(parts, " - ")
}

// IsRetryable returns true if the operation can be retried
func (sve *SecurityValidationError) IsRetryable() bool {
	return sve.Retryable
}

// NewPathTraversalError creates an error for path traversal attempts
func NewPathTraversalError(path string, approved []string) *SecurityValidationError {
	return &SecurityValidationError{
		Type:    string(ViolationPathTraversal),
		Message: "path traversal detected in path",
		Details: map[string]interface{}{
			"approved_directories": approved,
			"path_length":          len(path),
		},
		Retryable:    false,
		SuggestedFix: fmt.Sprintf("Use a path within approved directories: %v", approved),
	}
}

// NewPromptInjectionError creates an error for prompt injection attempts
func NewPromptInjectionError(inputType string, detected []string) *SecurityValidationError {
	return &SecurityValidationError{
		Type:    string(ViolationPromptInjection),
		Message: "prompt injection patterns detected in input",
		Details: map[string]interface{}{
			"input_type":        inputType,
			"patterns_detected": len(detected),
		},
		Retryable:    true,
		SuggestedFix: "Remove instruction override attempts from input and retry",
	}
}

// NewInputValidationError creates an error for input validation failures
func NewInputValidationError(inputType, reason string) *SecurityValidationError {
	return &SecurityValidationError{
		Type:    string(ViolationInputValidation),
		Message: fmt.Sprintf("input validation failed for %s: %s", inputType, reason),
		Details: map[string]interface{}{
			"input_type": inputType,
			"reason":     reason,
		},
		Retryable:    true,
		SuggestedFix: "Ensure input conforms to expected format and size limits",
	}
}
