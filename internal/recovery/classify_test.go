package recovery

import (
	"errors"
	"fmt"
	"testing"

	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/preflight"
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
		{
			name: "direct skill error",
			err: &preflight.SkillError{
				MissingSkills: []string{"speckit"},
				Err:           errors.New("missing required skills: speckit"),
			},
			expected: ClassPreflight,
		},
		{
			name: "wrapped skill error",
			err: fmt.Errorf("preflight check failed: %w", &preflight.SkillError{
				MissingSkills: []string{"speckit", "testkit"},
				Err:           errors.New("missing required skills: speckit, testkit"),
			}),
			expected: ClassPreflight,
		},
		{
			name: "direct tool error",
			err: &preflight.ToolError{
				MissingTools: []string{"gh"},
				Err:          errors.New("missing required tools: gh"),
			},
			expected: ClassPreflight,
		},
		{
			name: "wrapped tool error",
			err: fmt.Errorf("preflight check failed: %w", &preflight.ToolError{
				MissingTools: []string{"gh", "jq"},
				Err:          errors.New("missing required tools: gh, jq"),
			}),
			expected: ClassPreflight,
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

func TestExtractPreflightMetadata(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSkills []string
		wantTools  []string
		wantNil    bool
	}{
		{
			name:    "nil error returns nil",
			err:     nil,
			wantNil: true,
		},
		{
			name:    "non-preflight error returns nil",
			err:     errors.New("generic error"),
			wantNil: true,
		},
		{
			name: "skill error extracts missing skills",
			err: &preflight.SkillError{
				MissingSkills: []string{"speckit", "testkit"},
			},
			wantSkills: []string{"speckit", "testkit"},
		},
		{
			name: "tool error extracts missing tools",
			err: &preflight.ToolError{
				MissingTools: []string{"jq", "yq"},
			},
			wantTools: []string{"jq", "yq"},
		},
		{
			name: "wrapped skill error extracts missing skills",
			err: errors.Join(
				errors.New("preflight check failed"),
				&preflight.SkillError{
					MissingSkills: []string{"speckit"},
				},
			),
			wantSkills: []string{"speckit"},
		},
		{
			name: "wrapped tool error extracts missing tools",
			err: errors.Join(
				errors.New("preflight check failed"),
				&preflight.ToolError{
					MissingTools: []string{"jq"},
				},
			),
			wantTools: []string{"jq"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := ExtractPreflightMetadata(tt.err)

			if tt.wantNil {
				if meta != nil {
					t.Errorf("expected nil metadata, got %+v", meta)
				}
				return
			}

			if meta == nil {
				t.Fatal("expected non-nil metadata")
			}

			if len(tt.wantSkills) > 0 {
				if len(meta.MissingSkills) != len(tt.wantSkills) {
					t.Errorf("MissingSkills count = %d, want %d", len(meta.MissingSkills), len(tt.wantSkills))
				}
				for i, skill := range tt.wantSkills {
					if i >= len(meta.MissingSkills) || meta.MissingSkills[i] != skill {
						t.Errorf("MissingSkills[%d] = %q, want %q", i, meta.MissingSkills[i], skill)
					}
				}
			}

			if len(tt.wantTools) > 0 {
				if len(meta.MissingTools) != len(tt.wantTools) {
					t.Errorf("MissingTools count = %d, want %d", len(meta.MissingTools), len(tt.wantTools))
				}
				for i, tool := range tt.wantTools {
					if i >= len(meta.MissingTools) || meta.MissingTools[i] != tool {
						t.Errorf("MissingTools[%d] = %q, want %q", i, meta.MissingTools[i], tool)
					}
				}
			}
		})
	}
}
