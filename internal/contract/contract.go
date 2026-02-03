package contract

import (
	"fmt"
	"strings"
	"time"
)

// ContractConfig defines the configuration for contract validation.
type ContractConfig struct {
	Type        string   `json:"type"`
	Source      string   `json:"source,omitempty"`
	Schema      string   `json:"schema,omitempty"`
	SchemaPath  string   `json:"schemaPath,omitempty"`
	Command     string   `json:"command,omitempty"`
	CommandArgs []string `json:"commandArgs,omitempty"`
	StrictMode  bool     `json:"strictMode,omitempty"` // Deprecated: use MustPass instead
	MustPass    bool     `json:"must_pass,omitempty"`   // New: determines if validation failure blocks pipeline
	MaxRetries  int      `json:"maxRetries,omitempty"`
	QualityGates []QualityGateConfig `json:"quality_gates,omitempty"` // Quality gates to enforce
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
	case "markdown_spec":
		return &markdownSpecValidator{}
	case "template":
		return &TemplateValidator{}
	case "format":
		return &FormatValidator{}
	default:
		return nil
	}
}

// Validate runs the appropriate validator for the given configuration.
// It also runs any configured quality gates.
func Validate(cfg ContractConfig, workspacePath string) error {
	// First run the primary contract validator
	validator := NewValidator(cfg)
	if validator != nil {
		if err := validator.Validate(cfg, workspacePath); err != nil {
			return err
		}
	}

	// Then run quality gates if configured
	if len(cfg.QualityGates) > 0 {
		runner := NewQualityGateRunner()
		result, err := runner.RunGates(workspacePath, cfg.QualityGates)
		if err != nil {
			return fmt.Errorf("quality gate execution failed: %w", err)
		}

		if !result.Passed {
			return &ValidationError{
				ContractType: cfg.Type,
				Message:      fmt.Sprintf("quality gates failed (score: %d/100)", result.Score),
				Details:      formatQualityViolations(result.Violations),
				Retryable:    true,
			}
		}
	}

	return nil
}

// formatQualityViolations converts quality violations to string details
func formatQualityViolations(violations []QualityViolation) []string {
	details := make([]string, 0, len(violations))
	for _, v := range violations {
		msg := fmt.Sprintf("[%s] %s: %s", v.Severity, v.Gate, v.Message)
		details = append(details, msg)
		for _, detail := range v.Details {
			details = append(details, fmt.Sprintf("  - %s", detail))
		}
	}
	return details
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

// ValidateWithAdaptiveRetry runs validation with intelligent retry strategy.
// It analyzes failures and generates targeted repair prompts for the AI.
func ValidateWithAdaptiveRetry(cfg ContractConfig, workspacePath string) (*RetryResult, error) {
	strategy := NewAdaptiveRetryStrategy(cfg.MaxRetries)
	if cfg.MaxRetries <= 0 {
		strategy.MaxRetries = 3 // Default to 3 retries
	}

	result := &RetryResult{
		Attempts:     0,
		FailureTypes: make([]FailureType, 0),
	}

	startTime := time.Now()

	for attempt := 1; attempt <= strategy.MaxRetries; attempt++ {
		result.Attempts = attempt

		// Run validation
		err := Validate(cfg, workspacePath)

		if err == nil {
			// Success
			result.Success = true
			result.TotalDuration = time.Since(startTime)
			return result, nil
		}

		// Classify the failure
		classified := strategy.Classifier.Classify(err)
		if classified != nil {
			result.FailureTypes = append(result.FailureTypes, classified.Type)
		}

		// Check if we should retry
		if !strategy.ShouldRetry(attempt, err) {
			result.FinalError = err
			result.TotalDuration = time.Since(startTime)
			return result, err
		}

		// If not the last attempt, wait before retrying
		if attempt < strategy.MaxRetries {
			delay := strategy.GetRetryDelay(attempt)
			time.Sleep(delay)
		}
	}

	// All retries exhausted
	result.Success = false
	result.TotalDuration = time.Since(startTime)
	return result, result.FinalError
}

// GetRepairGuidance generates targeted guidance for fixing validation failures.
// This is intended to be injected into the AI prompt for retry attempts.
func GetRepairGuidance(err error, attempt int, maxRetries int) string {
	strategy := NewAdaptiveRetryStrategy(maxRetries)
	return strategy.GenerateRepairPrompt(err, attempt)
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
