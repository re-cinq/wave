package contract

import (
	"fmt"
	"strings"
)

// ContractConfig defines the configuration for contract validation.
type ContractConfig struct {
	Type        string   `json:"type"`
	Source      string   `json:"source,omitempty"`
	Schema      string   `json:"schema,omitempty"`
	SchemaPath  string   `json:"schemaPath,omitempty"`
	Command     string   `json:"command,omitempty"`
	CommandArgs []string `json:"commandArgs,omitempty"`
	StrictMode  bool     `json:"strictMode,omitempty"`
	MaxRetries  int      `json:"maxRetries,omitempty"`
}

// ValidationError provides detailed information about contract validation failures.
type ValidationError struct {
	ContractType string
	Message      string
	Details      []string
	Retryable    bool
	Attempt      int
	MaxRetries   int
}

func (e *ValidationError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("contract validation failed [%s]", e.ContractType))

	if e.MaxRetries > 0 {
		sb.WriteString(fmt.Sprintf(" (attempt %d/%d)", e.Attempt, e.MaxRetries))
	}

	sb.WriteString(": ")
	sb.WriteString(e.Message)

	if len(e.Details) > 0 {
		sb.WriteString("\n  Details:")
		for _, detail := range e.Details {
			sb.WriteString("\n    - ")
			sb.WriteString(detail)
		}
	}

	return sb.String()
}

// ContractValidator defines the interface for contract validation.
type ContractValidator interface {
	Validate(cfg ContractConfig, workspacePath string) error
}

// NewValidator creates a new contract validator based on the configuration type.
func NewValidator(cfg ContractConfig) ContractValidator {
	switch cfg.Type {
	case "json_schema":
		return &jsonSchemaValidator{}
	case "typescript_interface":
		return &typeScriptValidator{}
	case "test_suite":
		return &testSuiteValidator{}
	default:
		return nil
	}
}

// Validate runs the appropriate validator for the given configuration.
func Validate(cfg ContractConfig, workspacePath string) error {
	validator := NewValidator(cfg)
	if validator == nil {
		return nil
	}
	return validator.Validate(cfg, workspacePath)
}

// ValidateWithRetries runs validation with retry logic.
// It returns a ValidationError when max retries are exhausted.
func ValidateWithRetries(cfg ContractConfig, workspacePath string) error {
	validator := NewValidator(cfg)
	if validator == nil {
		return nil
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := validator.Validate(cfg, workspacePath)
		if err == nil {
			return nil
		}
		lastErr = err

		// If this is the last attempt, return a detailed error
		if attempt == maxRetries {
			return &ValidationError{
				ContractType: cfg.Type,
				Message:      fmt.Sprintf("validation failed after %d attempt(s)", maxRetries),
				Details:      []string{lastErr.Error()},
				Retryable:    false,
				Attempt:      attempt,
				MaxRetries:   maxRetries,
			}
		}
	}

	return lastErr
}

// WrapValidationError wraps a regular error into a ValidationError with details.
func WrapValidationError(contractType string, err error, details ...string) *ValidationError {
	return &ValidationError{
		ContractType: contractType,
		Message:      err.Error(),
		Details:      details,
		Retryable:    true,
	}
}
