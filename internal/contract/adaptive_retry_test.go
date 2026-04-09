package contract

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateWithAdaptiveRetry_SuccessFirstAttempt(t *testing.T) {
	workspacePath := t.TempDir()
	writeTestArtifact(t, workspacePath, []byte(`{"name": "valid"}`))

	cfg := ContractConfig{
		Type:       "json_schema",
		Schema:     `{"type": "object", "properties": {"name": {"type": "string"}}}`,
		MaxRetries: 3,
	}

	result, err := ValidateWithAdaptiveRetry(cfg, workspacePath)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", result.Attempts)
	}
	if len(result.FailureTypes) != 0 {
		t.Errorf("expected no failure types, got %v", result.FailureTypes)
	}
	if result.TotalDuration <= 0 {
		t.Error("expected positive TotalDuration")
	}
}

func TestValidateWithAdaptiveRetry_ExhaustsRetries(t *testing.T) {
	workspacePath := t.TempDir()
	writeTestArtifact(t, workspacePath, []byte(`{"name": 123}`)) // Invalid

	cfg := ContractConfig{
		Type:       "json_schema",
		Schema:     `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
		MaxRetries: 2,
	}

	result, err := ValidateWithAdaptiveRetry(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.Attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", result.Attempts)
	}
	if len(result.FailureTypes) == 0 {
		t.Error("expected at least one failure type recorded")
	}
	if result.FinalError == nil {
		t.Error("expected FinalError to be set")
	}
}

func TestValidateWithAdaptiveRetry_DefaultRetries(t *testing.T) {
	workspacePath := t.TempDir()
	writeTestArtifact(t, workspacePath, []byte(`{"name": 123}`))

	cfg := ContractConfig{
		Type:       "json_schema",
		Schema:     `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
		MaxRetries: 0, // Should default to 3
	}

	result, err := ValidateWithAdaptiveRetry(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for invalid artifact")
	}
	if result.Attempts != 3 {
		t.Errorf("expected 3 default attempts, got %d", result.Attempts)
	}
}

func TestValidateWithAdaptiveRetry_UnknownValidatorType(t *testing.T) {
	cfg := ContractConfig{
		Type:       "unknown_type",
		MaxRetries: 3,
	}

	result, err := ValidateWithAdaptiveRetry(cfg, t.TempDir())
	if err != nil {
		t.Fatalf("expected nil error for unknown validator type, got: %v", err)
	}
	if !result.Success {
		t.Error("expected success for unknown validator type (no validator means no failure)")
	}
	if result.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", result.Attempts)
	}
}

func TestGetRepairGuidance(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		attempt     int
		maxRetries  int
		wantContain []string
	}{
		{
			name:        "schema mismatch error",
			err:         errors.New("schema does not match"),
			attempt:     1,
			maxRetries:  3,
			wantContain: []string{"VALIDATION FAILURE", "schema"},
		},
		{
			name:        "format error",
			err:         errors.New("invalid JSON parse error"),
			attempt:     2,
			maxRetries:  3,
			wantContain: []string{"VALIDATION FAILURE", "retry attempt 2"},
		},
		{
			name:        "missing content error",
			err:         errors.New("required field missing"),
			attempt:     1,
			maxRetries:  3,
			wantContain: []string{"VALIDATION FAILURE", "required"},
		},
		{
			name:       "nil error returns empty string",
			err:        nil,
			attempt:    1,
			maxRetries: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guidance := GetRepairGuidance(tt.err, tt.attempt, tt.maxRetries)

			if tt.err == nil {
				if guidance != "" {
					t.Errorf("expected empty guidance for nil error, got: %s", guidance)
				}
				return
			}

			for _, s := range tt.wantContain {
				if !strings.Contains(strings.ToLower(guidance), strings.ToLower(s)) {
					t.Errorf("guidance should contain %q, got:\n%s", s, guidance)
				}
			}
		})
	}
}

func TestRetryResult_FormatSummary(t *testing.T) {
	tests := []struct {
		name     string
		result   RetryResult
		contains []string
	}{
		{
			name: "success first attempt",
			result: RetryResult{
				Success:  true,
				Attempts: 1,
			},
			contains: []string{"first attempt"},
		},
		{
			name: "success after retries",
			result: RetryResult{
				Success:  true,
				Attempts: 3,
			},
			contains: []string{"3 attempts"},
		},
		{
			name: "failure with types",
			result: RetryResult{
				Success:      false,
				Attempts:     3,
				FailureTypes: []FailureType{FailureTypeSchemaMismatch, FailureTypeFormatError, FailureTypeSchemaMismatch},
				FinalError:   errors.New("validation failed"),
			},
			contains: []string{"Failed after 3 attempts", "Failure progression", "schema_mismatch", "format_error", "Final error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.result.FormatSummary()
			for _, s := range tt.contains {
				if !strings.Contains(summary, s) {
					t.Errorf("summary should contain %q, got:\n%s", s, summary)
				}
			}
		})
	}
}

func TestFailureClassifier_Classify(t *testing.T) {
	classifier := &FailureClassifier{}

	tests := []struct {
		name      string
		err       error
		wantType  FailureType
		wantRetry bool
		wantNil   bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
		{
			name:      "schema mismatch",
			err:       errors.New("schema does not match expected format"),
			wantType:  FailureTypeSchemaMismatch,
			wantRetry: true,
		},
		{
			name:      "quality gate failure",
			err:       errors.New("quality gate check failed"),
			wantType:  FailureTypeQualityGate,
			wantRetry: true,
		},
		{
			name:      "missing content",
			err:       errors.New("required field is missing"),
			wantType:  FailureTypeMissingContent,
			wantRetry: true,
		},
		{
			name:      "JSON parse error",
			err:       errors.New("invalid JSON syntax"),
			wantType:  FailureTypeFormatError,
			wantRetry: true,
		},
		{
			name:      "structure error",
			err:       errors.New("document structure is wrong"),
			wantType:  FailureTypeStructure,
			wantRetry: true,
		},
		{
			name:      "unknown error type",
			err:       errors.New("something completely unrelated happened"),
			wantType:  FailureTypeUnknown,
			wantRetry: false,
		},
		{
			name: "ValidationError json_schema with parse",
			err: &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to parse JSON",
				Retryable:    true,
			},
			wantType:  FailureTypeFormatError,
			wantRetry: true,
		},
		{
			name: "ValidationError json_schema without parse",
			err: &ValidationError{
				ContractType: "json_schema",
				Message:      "field type mismatch",
				Retryable:    true,
			},
			wantType:  FailureTypeSchemaMismatch,
			wantRetry: true,
		},
		{
			name: "ValidationError markdown_spec",
			err: &ValidationError{
				ContractType: "markdown_spec",
				Message:      "structure issue",
				Retryable:    true,
			},
			wantType:  FailureTypeStructure,
			wantRetry: true,
		},
		{
			name: "ValidationError test_suite",
			err: &ValidationError{
				ContractType: "test_suite",
				Message:      "tests failed",
				Retryable:    true,
			},
			wantType:  FailureTypeQualityGate,
			wantRetry: true,
		},
		{
			name: "ValidationError unknown contract type",
			err: &ValidationError{
				ContractType: "custom",
				Message:      "custom failure",
				Retryable:    false,
			},
			wantType:  FailureTypeUnknown,
			wantRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.err)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", result.Type, tt.wantType)
			}
			if result.Retryable != tt.wantRetry {
				t.Errorf("Retryable = %v, want %v", result.Retryable, tt.wantRetry)
			}
			if result.Message == "" {
				t.Error("expected non-empty Message")
			}
		})
	}
}

func TestAdaptiveRetryStrategy_ShouldRetry(t *testing.T) {
	strategy := NewAdaptiveRetryStrategy(3)

	tests := []struct {
		name    string
		attempt int
		err     error
		want    bool
	}{
		{"retryable error under max", 1, errors.New("JSON parse error"), true},
		{"retryable error at max", 3, errors.New("JSON parse error"), false},
		{"non-retryable error", 1, errors.New("completely unknown error"), false},
		{"nil error", 1, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.ShouldRetry(tt.attempt, tt.err)
			if got != tt.want {
				t.Errorf("ShouldRetry(%d, %v) = %v, want %v", tt.attempt, tt.err, got, tt.want)
			}
		})
	}
}

func TestAdaptiveRetryStrategy_GetRetryDelay(t *testing.T) {
	strategy := NewAdaptiveRetryStrategy(5)

	// Verify exponential backoff with reasonable bounds
	delay1 := strategy.GetRetryDelay(1)
	delay2 := strategy.GetRetryDelay(2)
	delay3 := strategy.GetRetryDelay(3)

	if delay1 <= 0 {
		t.Error("delay should be positive")
	}
	// With jitter, delay2 should generally be larger than delay1 but not always,
	// so we check reasonable bounds instead
	if delay2 > strategy.MaxDelay {
		t.Errorf("delay2 %v exceeds MaxDelay %v", delay2, strategy.MaxDelay)
	}
	if delay3 > strategy.MaxDelay {
		t.Errorf("delay3 %v exceeds MaxDelay %v", delay3, strategy.MaxDelay)
	}

	// Very high attempt should be capped near MaxDelay (with ±25% jitter)
	delayHigh := strategy.GetRetryDelay(100)
	maxWithJitter := time.Duration(float64(strategy.MaxDelay) * 1.25)
	if delayHigh > maxWithJitter {
		t.Errorf("delay for high attempt %v should be within MaxDelay + jitter %v", delayHigh, maxWithJitter)
	}
}

func TestAdaptiveRetryStrategy_GenerateRepairPrompt(t *testing.T) {
	strategy := NewAdaptiveRetryStrategy(3)

	tests := []struct {
		name     string
		err      error
		attempt  int
		contains []string
	}{
		{
			name:     "schema mismatch prompt",
			err:      errors.New("schema does not match"),
			attempt:  1,
			contains: []string{"VALIDATION FAILURE", "schema", "required fields"},
		},
		{
			name:     "format error prompt",
			err:      errors.New("invalid JSON syntax"),
			attempt:  1,
			contains: []string{"VALIDATION FAILURE", "valid JSON"},
		},
		{
			name:     "quality gate prompt",
			err:      errors.New("quality gate failed"),
			attempt:  1,
			contains: []string{"VALIDATION FAILURE", "quality"},
		},
		{
			name:     "structure prompt",
			err:      errors.New("document structure invalid"),
			attempt:  1,
			contains: []string{"VALIDATION FAILURE", "structure"},
		},
		{
			name:     "retry attempt 2 adds extra warning",
			err:      errors.New("JSON parse error"),
			attempt:  2,
			contains: []string{"retry attempt 2"},
		},
		{
			name:     "missing content prompt",
			err:      errors.New("required field is missing"),
			attempt:  1,
			contains: []string{"required fields"},
		},
		{
			name: "validation error with details",
			err: &ValidationError{
				ContractType: "json_schema",
				Message:      "field type wrong",
				Details:      []string{"name must be string", "age must be integer"},
				Retryable:    true,
			},
			attempt:  1,
			contains: []string{"Details", "name must be string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := strategy.GenerateRepairPrompt(tt.err, tt.attempt)

			for _, s := range tt.contains {
				if !strings.Contains(strings.ToLower(prompt), strings.ToLower(s)) {
					t.Errorf("prompt should contain %q, got:\n%s", s, prompt)
				}
			}
		})
	}

	// Test nil error returns empty prompt
	if prompt := strategy.GenerateRepairPrompt(nil, 1); prompt != "" {
		t.Errorf("expected empty prompt for nil error, got: %s", prompt)
	}
}

func TestNewAdaptiveRetryStrategy_Defaults(t *testing.T) {
	strategy := NewAdaptiveRetryStrategy(5)

	if strategy.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", strategy.MaxRetries)
	}
	if strategy.BaseDelay != 1*time.Second {
		t.Errorf("BaseDelay = %v, want 1s", strategy.BaseDelay)
	}
	if strategy.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", strategy.MaxDelay)
	}
	if strategy.BackoffFactor != 2.0 {
		t.Errorf("BackoffFactor = %f, want 2.0", strategy.BackoffFactor)
	}
	if strategy.Classifier == nil {
		t.Error("Classifier should not be nil")
	}
}

// Test that ValidateWithAdaptiveRetry correctly uses the Dir field for test_suite
func TestValidateWithAdaptiveRetry_TestSuiteSuccess(t *testing.T) {
	workspacePath := t.TempDir()

	cfg := ContractConfig{
		Type:       "test_suite",
		Command:    "true",
		Dir:        workspacePath,
		MaxRetries: 1,
	}

	result, err := ValidateWithAdaptiveRetry(cfg, workspacePath)
	if err != nil {
		t.Fatalf("expected success for 'true' command, got: %v", err)
	}
	if !result.Success {
		t.Error("expected Success for 'true' command")
	}
}

func TestValidateWithAdaptiveRetry_TestSuiteFailure(t *testing.T) {
	workspacePath := t.TempDir()

	// Create a script that always fails
	script := filepath.Join(workspacePath, "fail.sh")
	_ = os.WriteFile(script, []byte("#!/bin/sh\nexit 1"), 0755)

	cfg := ContractConfig{
		Type:       "test_suite",
		Command:    script,
		Dir:        workspacePath,
		MaxRetries: 1,
	}

	result, err := ValidateWithAdaptiveRetry(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for failing test suite")
	}
	if result.Success {
		t.Error("expected failure for failing test suite")
	}
}

func TestValidateWithAdaptiveRetry_MissingArtifact(t *testing.T) {
	workspacePath := t.TempDir()
	// Don't create artifact.json — validation should fail

	cfg := ContractConfig{
		Type:       "json_schema",
		Schema:     `{"type": "object"}`,
		MaxRetries: 1,
	}

	result, err := ValidateWithAdaptiveRetry(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for missing artifact")
	}
	if result.Success {
		t.Error("expected failure for missing artifact")
	}
	if result.FinalError == nil {
		t.Error("expected FinalError to be set")
	}
}

func TestClassifiedFailure_ConfidenceRange(t *testing.T) {
	classifier := &FailureClassifier{}

	errorMessages := []string{
		"schema does not match",
		"quality gate failed",
		"required field missing",
		"JSON parse error",
		"structure hierarchy wrong",
		"completely unknown error",
	}

	for _, msg := range errorMessages {
		t.Run(msg, func(t *testing.T) {
			result := classifier.Classify(fmt.Errorf("%s", msg))
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Confidence < 0 || result.Confidence > 1.0 {
				t.Errorf("Confidence %f should be between 0 and 1", result.Confidence)
			}
		})
	}
}
