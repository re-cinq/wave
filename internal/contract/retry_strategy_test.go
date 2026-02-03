package contract

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailureClassifier(t *testing.T) {
	classifier := &FailureClassifier{}

	tests := []struct {
		name           string
		err            error
		expectedType   FailureType
		expectedRetry  bool
		minConfidence  float64
	}{
		{
			name:           "schema mismatch error",
			err:            errors.New("artifact does not match schema: missing required field 'name'"),
			expectedType:   FailureTypeSchemaMismatch,
			expectedRetry:  true,
			minConfidence:  0.8,
		},
		{
			name:           "missing content error",
			err:            errors.New("required field 'description' is missing"),
			expectedType:   FailureTypeMissingContent,
			expectedRetry:  true,
			minConfidence:  0.8,
		},
		{
			name:           "JSON parse error",
			err:            errors.New("failed to parse JSON: unexpected token at position 10"),
			expectedType:   FailureTypeFormatError,
			expectedRetry:  true,
			minConfidence:  0.9,
		},
		{
			name:           "structure error",
			err:            errors.New("markdown structure invalid: heading hierarchy broken"),
			expectedType:   FailureTypeStructure,
			expectedRetry:  true,
			minConfidence:  0.7,
		},
		{
			name:           "quality gate error",
			err:            errors.New("quality gate failed: content completeness below threshold"),
			expectedType:   FailureTypeQualityGate,
			expectedRetry:  true,
			minConfidence:  0.8,
		},
		{
			name:           "unknown error",
			err:            errors.New("something went wrong"),
			expectedType:   FailureTypeUnknown,
			expectedRetry:  false,
			minConfidence:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classified := classifier.Classify(tt.err)
			require.NotNil(t, classified)
			assert.Equal(t, tt.expectedType, classified.Type)
			assert.Equal(t, tt.expectedRetry, classified.Retryable)
			assert.GreaterOrEqual(t, classified.Confidence, tt.minConfidence)
			assert.NotEmpty(t, classified.Suggestions, "Should provide suggestions")
		})
	}
}

func TestFailureClassifierWithValidationError(t *testing.T) {
	classifier := &FailureClassifier{}

	tests := []struct {
		name         string
		validationErr *ValidationError
		expectedType FailureType
	}{
		{
			name: "json_schema parse error",
			validationErr: &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to parse artifact JSON",
				Details:      []string{"unexpected character"},
				Retryable:    true,
			},
			expectedType: FailureTypeFormatError,
		},
		{
			name: "json_schema validation error",
			validationErr: &ValidationError{
				ContractType: "json_schema",
				Message:      "artifact does not match schema",
				Details:      []string{"missing field: name"},
				Retryable:    true,
			},
			expectedType: FailureTypeSchemaMismatch,
		},
		{
			name: "markdown_spec error",
			validationErr: &ValidationError{
				ContractType: "markdown_spec",
				Message:      "spec validation failed",
				Details:      []string{"missing required sections"},
				Retryable:    true,
			},
			expectedType: FailureTypeStructure,
		},
		{
			name: "test_suite error",
			validationErr: &ValidationError{
				ContractType: "test_suite",
				Message:      "tests failed",
				Details:      []string{"3 tests failed"},
				Retryable:    true,
			},
			expectedType: FailureTypeQualityGate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classified := classifier.Classify(tt.validationErr)
			require.NotNil(t, classified)
			assert.Equal(t, tt.expectedType, classified.Type)
			assert.True(t, classified.Retryable)
			assert.GreaterOrEqual(t, classified.Confidence, 0.9)
			assert.NotEmpty(t, classified.Suggestions)
		})
	}
}

func TestAdaptiveRetryStrategy(t *testing.T) {
	strategy := NewAdaptiveRetryStrategy(3)

	t.Run("should retry on retryable errors", func(t *testing.T) {
		err := errors.New("artifact does not match schema")
		assert.True(t, strategy.ShouldRetry(1, err))
		assert.True(t, strategy.ShouldRetry(2, err))
		assert.False(t, strategy.ShouldRetry(3, err), "Should not retry after max attempts")
	})

	t.Run("should not retry on non-retryable errors", func(t *testing.T) {
		err := errors.New("unknown catastrophic failure")
		assert.False(t, strategy.ShouldRetry(1, err))
	})

	t.Run("should calculate exponential backoff", func(t *testing.T) {
		delay1 := strategy.GetRetryDelay(1)
		delay2 := strategy.GetRetryDelay(2)
		delay3 := strategy.GetRetryDelay(3)

		// Delays should increase
		assert.Greater(t, delay2, delay1)
		assert.Greater(t, delay3, delay2)

		// Should respect max delay
		assert.LessOrEqual(t, delay3, strategy.MaxDelay)
	})

	t.Run("should generate appropriate repair prompts", func(t *testing.T) {
		tests := []struct {
			name          string
			err           error
			attempt       int
			shouldContain []string
		}{
			{
				name:    "schema mismatch prompt",
				err:     errors.New("does not match schema"),
				attempt: 1,
				shouldContain: []string{
					"VALIDATION FAILURE",
					"schema",
					"required fields",
					"valid JSON",
				},
			},
			{
				name:    "format error prompt",
				err:     errors.New("failed to parse JSON"),
				attempt: 1,
				shouldContain: []string{
					"VALIDATION FAILURE",
					"JSON",
					"no markdown",
					"properly quoted",
				},
			},
			{
				name:    "retry attempt warning",
				err:     errors.New("does not match schema"),
				attempt: 2,
				shouldContain: []string{
					"retry attempt 2",
					"extra careful",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				prompt := strategy.GenerateRepairPrompt(tt.err, tt.attempt)
				assert.NotEmpty(t, prompt)

				for _, expected := range tt.shouldContain {
					assert.Contains(t, strings.ToLower(prompt), strings.ToLower(expected),
						"Prompt should contain guidance about %s", expected)
				}
			})
		}
	})
}

func TestRetryResultFormatting(t *testing.T) {
	t.Run("successful first attempt", func(t *testing.T) {
		result := &RetryResult{
			Success:  true,
			Attempts: 1,
		}

		summary := result.FormatSummary()
		assert.Contains(t, summary, "first attempt")
		assert.Contains(t, summary, "✓")
	})

	t.Run("successful after retries", func(t *testing.T) {
		result := &RetryResult{
			Success:  true,
			Attempts: 3,
		}

		summary := result.FormatSummary()
		assert.Contains(t, summary, "after 3 attempts")
		assert.Contains(t, summary, "✓")
	})

	t.Run("failed after retries", func(t *testing.T) {
		result := &RetryResult{
			Success:  false,
			Attempts: 3,
			FailureTypes: []FailureType{
				FailureTypeFormatError,
				FailureTypeSchemaMismatch,
				FailureTypeSchemaMismatch,
			},
			FinalError: errors.New("final validation error"),
		}

		summary := result.FormatSummary()
		assert.Contains(t, summary, "Failed after 3 attempts")
		assert.Contains(t, summary, "✗")
		assert.Contains(t, summary, "Failure progression")
		assert.Contains(t, summary, "format_error")
		assert.Contains(t, summary, "schema_mismatch")
		assert.Contains(t, summary, "final validation error")
	})
}

func TestRetryDelayWithJitter(t *testing.T) {
	strategy := NewAdaptiveRetryStrategy(5)

	// Run multiple times to verify jitter creates variance
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = strategy.GetRetryDelay(2)
	}

	// Check that we have some variance (not all delays are exactly the same)
	// This is a probabilistic test - it could theoretically fail if we're very unlucky
	uniqueDelays := make(map[time.Duration]bool)
	for _, d := range delays {
		uniqueDelays[d] = true
	}

	// With jitter, we should have at least a few different values
	assert.Greater(t, len(uniqueDelays), 1, "Jitter should create variance in delays")
}

func TestPowHelper(t *testing.T) {
	tests := []struct {
		base     float64
		exp      float64
		expected float64
	}{
		{2.0, 0.0, 1.0},
		{2.0, 1.0, 2.0},
		{2.0, 2.0, 4.0},
		{2.0, 3.0, 8.0},
		{3.0, 2.0, 9.0},
	}

	for _, tt := range tests {
		result := pow(tt.base, tt.exp)
		assert.Equal(t, tt.expected, result)
	}
}

func TestClassifiedFailureGuidance(t *testing.T) {
	classifier := &FailureClassifier{}

	err := &ValidationError{
		ContractType: "json_schema",
		Message:      "validation failed",
		Details: []string{
			"missing required field: 'name'",
			"field 'age' has wrong type: expected number, got string",
		},
		Retryable: true,
	}

	classified := classifier.Classify(err)
	require.NotNil(t, classified)

	// Verify classification
	assert.Equal(t, FailureTypeSchemaMismatch, classified.Type)
	assert.True(t, classified.Retryable)
	assert.Len(t, classified.Details, 2)

	// Verify suggestions are actionable
	assert.NotEmpty(t, classified.Suggestions)
	for _, suggestion := range classified.Suggestions {
		assert.NotEmpty(t, suggestion)
		// Suggestions should be actionable sentences
		assert.True(t, len(suggestion) > 10, "Suggestions should be detailed")
	}
}
