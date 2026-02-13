package recovery

import (
	"errors"
	"fmt"
	"testing"

	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/security"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorClass
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ClassUnknown,
		},
		{
			name: "direct contract validation error",
			err: &contract.ValidationError{
				ContractType: "json_schema",
				Message:      "missing required field",
			},
			expected: ClassContractValidation,
		},
		{
			name: "wrapped contract validation error",
			err: fmt.Errorf("step failed: %w", &contract.ValidationError{
				ContractType: "json_schema",
				Message:      "missing field",
			}),
			expected: ClassContractValidation,
		},
		{
			name: "direct security error",
			err: &security.SecurityValidationError{
				Type:    "path_traversal",
				Message: "path traversal detected",
			},
			expected: ClassSecurityViolation,
		},
		{
			name: "wrapped security error",
			err: fmt.Errorf("step failed: %w", &security.SecurityValidationError{
				Type:    "prompt_injection",
				Message: "injection detected",
			}),
			expected: ClassSecurityViolation,
		},
		{
			name:     "generic error with message",
			err:      errors.New("adapter crashed"),
			expected: ClassRuntimeError,
		},
		{
			name:     "wrapped generic error",
			err:      fmt.Errorf("step failed: %w", errors.New("timeout")),
			expected: ClassRuntimeError,
		},
		{
			name:     "bare exit status is unknown",
			err:      errors.New("exit status 1"),
			expected: ClassUnknown,
		},
		{
			name:     "signal error is unknown",
			err:      errors.New("signal: killed"),
			expected: ClassUnknown,
		},
		{
			name: "multi-wrapped contract error",
			err: fmt.Errorf("pipeline failed: %w",
				fmt.Errorf("step failed: %w", &contract.ValidationError{
					ContractType: "test",
					Message:      "test failed",
				})),
			expected: ClassContractValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError(tt.err)
			if got != tt.expected {
				t.Errorf("ClassifyError() = %q, want %q", got, tt.expected)
			}
		})
	}
}
