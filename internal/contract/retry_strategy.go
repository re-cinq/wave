package contract

import (
	"fmt"
	"strings"
	"time"
)

// RetryStrategy defines how to retry failed validations
type RetryStrategy interface {
	// ShouldRetry determines if a retry should be attempted
	ShouldRetry(attempt int, err error) bool
	// GetRetryDelay returns the delay before the next retry
	GetRetryDelay(attempt int) time.Duration
	// GenerateRepairPrompt creates a targeted prompt to fix the validation failure
	GenerateRepairPrompt(err error, attempt int) string
}

// FailureClassifier analyzes validation errors to determine failure type
type FailureClassifier struct{}

// FailureType categorizes different types of validation failures
type FailureType string

const (
	FailureTypeSchemaMismatch   FailureType = "schema_mismatch"
	FailureTypeMissingContent   FailureType = "missing_content"
	FailureTypeFormatError      FailureType = "format_error"
	FailureTypeQualityGate      FailureType = "quality_gate"
	FailureTypeStructure        FailureType = "structure"
	FailureTypeUnknown          FailureType = "unknown"
)

// ClassifiedFailure contains details about a validation failure
type ClassifiedFailure struct {
	Type        FailureType
	Message     string
	Details     []string
	Retryable   bool
	Confidence  float64 // 0.0 to 1.0
	Suggestions []string
}

// Classify analyzes a validation error and determines its type
func (c *FailureClassifier) Classify(err error) *ClassifiedFailure {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)

	// Check for validation error with details
	if validationErr, ok := err.(*ValidationError); ok {
		return c.classifyValidationError(validationErr)
	}

	// Check for quality gate failures
	if strings.Contains(errMsgLower, "quality gate") {
		return &ClassifiedFailure{
			Type:       FailureTypeQualityGate,
			Message:    errMsg,
			Retryable:  true,
			Confidence: 0.9,
			Suggestions: []string{
				"Review quality gate requirements",
				"Improve content completeness and formatting",
			},
		}
	}

	// Check for schema mismatches
	if strings.Contains(errMsgLower, "schema") || strings.Contains(errMsgLower, "does not match") {
		return &ClassifiedFailure{
			Type:       FailureTypeSchemaMismatch,
			Message:    errMsg,
			Retryable:  true,
			Confidence: 0.85,
			Suggestions: []string{
				"Review the JSON schema requirements",
				"Ensure all required fields are present",
				"Check field types match schema",
			},
		}
	}

	// Check for missing content
	if strings.Contains(errMsgLower, "missing") || strings.Contains(errMsgLower, "required") {
		return &ClassifiedFailure{
			Type:       FailureTypeMissingContent,
			Message:    errMsg,
			Retryable:  true,
			Confidence: 0.9,
			Suggestions: []string{
				"Add all required fields and sections",
				"Check for empty or placeholder values",
			},
		}
	}

	// Check for format errors
	if strings.Contains(errMsgLower, "json") || strings.Contains(errMsgLower, "parse") || strings.Contains(errMsgLower, "syntax") {
		return &ClassifiedFailure{
			Type:       FailureTypeFormatError,
			Message:    errMsg,
			Retryable:  true,
			Confidence: 0.95,
			Suggestions: []string{
				"Fix JSON syntax errors",
				"Ensure valid JSON structure",
				"Check for missing commas, brackets, or quotes",
			},
		}
	}

	// Check for structure issues
	if strings.Contains(errMsgLower, "structure") || strings.Contains(errMsgLower, "hierarchy") {
		return &ClassifiedFailure{
			Type:       FailureTypeStructure,
			Message:    errMsg,
			Retryable:  true,
			Confidence: 0.8,
			Suggestions: []string{
				"Review document structure requirements",
				"Ensure proper heading hierarchy",
			},
		}
	}

	// Unknown error type
	return &ClassifiedFailure{
		Type:       FailureTypeUnknown,
		Message:    errMsg,
		Retryable:  false,
		Confidence: 0.5,
		Suggestions: []string{
			"Review the error message for specific guidance",
		},
	}
}

func (c *FailureClassifier) classifyValidationError(err *ValidationError) *ClassifiedFailure {
	failure := &ClassifiedFailure{
		Message:    err.Message,
		Details:    err.Details,
		Retryable:  err.Retryable,
		Confidence: 0.95,
	}

	// Determine type from contract type and message
	switch err.ContractType {
	case "json_schema":
		if strings.Contains(strings.ToLower(err.Message), "parse") {
			failure.Type = FailureTypeFormatError
			failure.Suggestions = []string{
				"Fix JSON syntax errors",
				"Ensure the output is valid JSON",
				"Remove any markdown code blocks or explanatory text",
			}
		} else {
			failure.Type = FailureTypeSchemaMismatch
			failure.Suggestions = []string{
				"Review the schema requirements carefully",
				"Ensure all required fields are included",
				"Verify field types match the schema",
				"Check enum values are valid",
			}
		}

	case "markdown_spec":
		failure.Type = FailureTypeStructure
		failure.Suggestions = []string{
			"Ensure proper markdown structure",
			"Include all required sections",
			"Use correct heading hierarchy",
		}

	case "test_suite":
		failure.Type = FailureTypeQualityGate
		failure.Suggestions = []string{
			"Review test failures",
			"Fix any failing test cases",
			"Ensure code quality meets standards",
		}

	default:
		failure.Type = FailureTypeUnknown
	}

	return failure
}

// AdaptiveRetryStrategy implements intelligent retry logic based on failure type
type AdaptiveRetryStrategy struct {
	MaxRetries     int
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
	Classifier     *FailureClassifier
}

// NewAdaptiveRetryStrategy creates a new adaptive retry strategy
func NewAdaptiveRetryStrategy(maxRetries int) *AdaptiveRetryStrategy {
	return &AdaptiveRetryStrategy{
		MaxRetries:    maxRetries,
		BaseDelay:     1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Classifier:    &FailureClassifier{},
	}
}

// ShouldRetry determines if another retry attempt should be made
func (s *AdaptiveRetryStrategy) ShouldRetry(attempt int, err error) bool {
	if attempt >= s.MaxRetries {
		return false
	}

	// Classify the failure
	classified := s.Classifier.Classify(err)
	if classified == nil {
		return false
	}

	// Only retry if the failure is retryable
	return classified.Retryable
}

// GetRetryDelay calculates the delay before the next retry with exponential backoff and jitter
func (s *AdaptiveRetryStrategy) GetRetryDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * (backoffFactor ^ attempt)
	delay := float64(s.BaseDelay) * pow(s.BackoffFactor, float64(attempt))

	// Cap at max delay
	if delay > float64(s.MaxDelay) {
		delay = float64(s.MaxDelay)
	}

	// Add jitter (±25%)
	jitter := delay * 0.25
	delay = delay - jitter + (jitter * 2.0 * rand())

	return time.Duration(delay)
}

// GenerateRepairPrompt creates a specific prompt to help fix the validation failure
func (s *AdaptiveRetryStrategy) GenerateRepairPrompt(err error, attempt int) string {
	classified := s.Classifier.Classify(err)
	if classified == nil {
		return ""
	}

	var prompt strings.Builder

	prompt.WriteString("VALIDATION FAILURE - RETRY REQUIRED\n\n")
	prompt.WriteString(fmt.Sprintf("Attempt %d of %d\n\n", attempt, s.MaxRetries))
	prompt.WriteString(fmt.Sprintf("Failure Type: %s\n", classified.Type))
	prompt.WriteString(fmt.Sprintf("Error: %s\n\n", classified.Message))

	if len(classified.Details) > 0 {
		prompt.WriteString("Details:\n")
		for _, detail := range classified.Details {
			prompt.WriteString(fmt.Sprintf("  - %s\n", detail))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("CRITICAL REQUIREMENTS:\n")

	switch classified.Type {
	case FailureTypeSchemaMismatch:
		prompt.WriteString("1. Review the JSON schema carefully - every field is important\n")
		prompt.WriteString("2. Ensure ALL required fields are present with correct types\n")
		prompt.WriteString("3. Check that enum values match allowed options exactly\n")
		prompt.WriteString("4. Verify nested object structures match the schema\n")
		prompt.WriteString("5. Do NOT add any extra fields not defined in the schema\n\n")
		prompt.WriteString("Output ONLY valid JSON matching the schema - no markdown, no explanations.\n")

	case FailureTypeMissingContent:
		prompt.WriteString("1. Add ALL required fields and sections\n")
		prompt.WriteString("2. Replace any placeholder text with real content\n")
		prompt.WriteString("3. Ensure no fields are empty or null unless explicitly allowed\n")
		prompt.WriteString("4. Provide complete, meaningful values for all fields\n\n")

	case FailureTypeFormatError:
		prompt.WriteString("1. Output ONLY valid JSON - no markdown code blocks\n")
		prompt.WriteString("2. Start with { or [ and end with } or ]\n")
		prompt.WriteString("3. Do NOT include any explanatory text before or after the JSON\n")
		prompt.WriteString("4. Ensure all strings are properly quoted\n")
		prompt.WriteString("5. Check for missing commas between array/object elements\n")
		prompt.WriteString("6. Verify all brackets and braces are balanced\n\n")

	case FailureTypeQualityGate:
		prompt.WriteString("1. Review quality requirements carefully\n")
		prompt.WriteString("2. Ensure content is complete and well-formatted\n")
		prompt.WriteString("3. Verify all required sections are present\n")
		prompt.WriteString("4. Remove placeholder or TODO content\n")
		prompt.WriteString("5. Meet minimum quality thresholds\n\n")

	case FailureTypeStructure:
		prompt.WriteString("1. Follow proper document structure\n")
		prompt.WriteString("2. Use correct heading hierarchy (h1, h2, h3 in order)\n")
		prompt.WriteString("3. Include all required sections\n")
		prompt.WriteString("4. Ensure consistent formatting throughout\n\n")
	}

	if len(classified.Suggestions) > 0 {
		prompt.WriteString("Specific Suggestions:\n")
		for i, suggestion := range classified.Suggestions {
			prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, suggestion))
		}
		prompt.WriteString("\n")
	}

	if attempt > 1 {
		prompt.WriteString(fmt.Sprintf("⚠ This is retry attempt %d - be extra careful to address the specific errors above.\n\n", attempt))
	}

	prompt.WriteString("Please correct the issues and generate a valid output that passes all validation checks.\n")

	return prompt.String()
}

// Helper functions

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// Simple pseudo-random for jitter (not cryptographically secure)
func rand() float64 {
	// Use current nanosecond as seed
	seed := time.Now().UnixNano()
	// Simple LCG (Linear Congruential Generator)
	return float64((seed*1103515245+12345)&0x7fffffff) / float64(0x7fffffff)
}

// RetryResult captures the outcome of a retry sequence
type RetryResult struct {
	Success       bool
	Attempts      int
	FailureTypes  []FailureType
	TotalDuration time.Duration
	FinalError    error
}

// FormatSummary returns a human-readable summary of the retry result
func (r *RetryResult) FormatSummary() string {
	if r.Success {
		if r.Attempts == 1 {
			return "✓ Passed validation on first attempt"
		}
		return fmt.Sprintf("✓ Passed validation after %d attempts", r.Attempts)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✗ Failed after %d attempts\n", r.Attempts))

	if len(r.FailureTypes) > 0 {
		sb.WriteString("\nFailure progression:\n")
		for i, ft := range r.FailureTypes {
			sb.WriteString(fmt.Sprintf("  Attempt %d: %s\n", i+1, ft))
		}
	}

	if r.FinalError != nil {
		sb.WriteString(fmt.Sprintf("\nFinal error: %s\n", r.FinalError.Error()))
	}

	return sb.String()
}
