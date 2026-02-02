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
		Message: fmt.Sprintf("path traversal detected in path"),
		Details: map[string]interface{}{
			"approved_directories": approved,
			"path_length":         len(path),
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

// NewInvalidPersonaError creates an error for invalid persona references
func NewInvalidPersonaError(persona string, available []string) *SecurityValidationError {
	return &SecurityValidationError{
		Type:    string(ViolationInvalidPersona),
		Message: fmt.Sprintf("persona '%s' not found in manifest", persona),
		Details: map[string]interface{}{
			"requested_persona":   persona,
			"available_personas":  available,
			"available_count":     len(available),
		},
		Retryable:    true,
		SuggestedFix: fmt.Sprintf("Use one of the available personas: %v", available),
	}
}

// NewMalformedJSONError creates an error for malformed JSON content
func NewMalformedJSONError(issue string) *SecurityValidationError {
	return &SecurityValidationError{
		Type:         string(ViolationMalformedJSON),
		Message:      fmt.Sprintf("malformed JSON detected: %s", issue),
		Retryable:    true,
		SuggestedFix: "Ensure JSON output follows valid JSON format without comments",
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

// NewConfigurationError creates an error for security configuration issues
func NewConfigurationError(setting, reason string) *SecurityValidationError {
	return &SecurityValidationError{
		Type:         "configuration_error",
		Message:      fmt.Sprintf("security configuration error in %s: %s", setting, reason),
		Retryable:    false,
		SuggestedFix: "Check security configuration in manifest and fix the specified setting",
	}
}

// IsSecurityError checks if an error is a security validation error
func IsSecurityError(err error) bool {
	_, ok := err.(*SecurityValidationError)
	return ok
}

// GetSecurityError extracts a SecurityValidationError from an error
func GetSecurityError(err error) (*SecurityValidationError, bool) {
	secErr, ok := err.(*SecurityValidationError)
	return secErr, ok
}