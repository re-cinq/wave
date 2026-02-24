package commands

import (
	"errors"
	"testing"

	"github.com/recinq/wave/internal/preflight"
)

func TestExtractPreflightMetadata(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		wantSkills       []string
		wantTools        []string
		wantNil          bool
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
			meta := extractPreflightMetadata(tt.err)

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
