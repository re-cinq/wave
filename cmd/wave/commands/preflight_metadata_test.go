package commands

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/recinq/wave/internal/preflight"
	"github.com/recinq/wave/internal/recovery"
)

func TestExtractPreflightMetadata(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want *recovery.PreflightMetadata
	}{
		{
			name: "nil error returns nil",
			err:  nil,
			want: nil,
		},
		{
			name: "non-preflight error returns nil",
			err:  errors.New("generic error"),
			want: nil,
		},
		{
			name: "skill error extracts missing skills",
			err: &preflight.SkillError{
				MissingSkills: []string{"speckit", "testkit"},
			},
			want: &recovery.PreflightMetadata{
				MissingSkills: []string{"speckit", "testkit"},
			},
		},
		{
			name: "tool error extracts missing tools",
			err: &preflight.ToolError{
				MissingTools: []string{"jq", "yq"},
			},
			want: &recovery.PreflightMetadata{
				MissingTools: []string{"jq", "yq"},
			},
		},
		{
			name: "errors.Join with skill error extracts missing skills",
			err: errors.Join(
				errors.New("preflight check failed"),
				&preflight.SkillError{
					MissingSkills: []string{"speckit"},
				},
			),
			want: &recovery.PreflightMetadata{
				MissingSkills: []string{"speckit"},
			},
		},
		{
			name: "errors.Join with tool error extracts missing tools",
			err: errors.Join(
				errors.New("preflight check failed"),
				&preflight.ToolError{
					MissingTools: []string{"jq"},
				},
			),
			want: &recovery.PreflightMetadata{
				MissingTools: []string{"jq"},
			},
		},
		{
			name: "errors.Join with both skill and tool errors extracts both",
			err: errors.Join(
				&preflight.SkillError{MissingSkills: []string{"speckit"}},
				&preflight.ToolError{MissingTools: []string{"jq"}},
			),
			want: &recovery.PreflightMetadata{
				MissingSkills: []string{"speckit"},
				MissingTools:  []string{"jq"},
			},
		},
		{
			name: "fmt.Errorf %w wrapping skill error extracts missing skills",
			err: fmt.Errorf("preflight failed: %w", &preflight.SkillError{
				MissingSkills: []string{"speckit", "testkit"},
			}),
			want: &recovery.PreflightMetadata{
				MissingSkills: []string{"speckit", "testkit"},
			},
		},
		{
			name: "fmt.Errorf %w wrapping tool error extracts missing tools",
			err: fmt.Errorf("preflight failed: %w", &preflight.ToolError{
				MissingTools: []string{"jq"},
			}),
			want: &recovery.PreflightMetadata{
				MissingTools: []string{"jq"},
			},
		},
		{
			name: "double-wrapped fmt.Errorf %w skill error still extracts missing skills",
			err: fmt.Errorf("outer: %w",
				fmt.Errorf("inner: %w", &preflight.SkillError{
					MissingSkills: []string{"speckit"},
				}),
			),
			want: &recovery.PreflightMetadata{
				MissingSkills: []string{"speckit"},
			},
		},
		{
			name: "skill error with empty MissingSkills returns nil",
			err: &preflight.SkillError{
				MissingSkills: nil,
			},
			want: nil,
		},
		{
			name: "tool error with empty MissingTools returns nil",
			err: &preflight.ToolError{
				MissingTools: []string{},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPreflightMetadata(tt.err)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractPreflightMetadata(%v) = %+v, want %+v", tt.err, got, tt.want)
			}
		})
	}
}
