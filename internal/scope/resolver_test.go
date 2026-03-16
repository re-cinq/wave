package scope

import (
	"testing"

	"github.com/recinq/wave/internal/forge"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name      string
		forgeType forge.ForgeType
		scope     TokenScope
		want      []string
		wantErr   bool
	}{
		// GitHub mappings
		{"github issues:read", forge.ForgeGitHub, TokenScope{Resource: "issues", Permission: "read"}, []string{"repo"}, false},
		{"github issues:write", forge.ForgeGitHub, TokenScope{Resource: "issues", Permission: "write"}, []string{"repo"}, false},
		{"github pulls:read", forge.ForgeGitHub, TokenScope{Resource: "pulls", Permission: "read"}, []string{"repo"}, false},
		{"github repos:admin", forge.ForgeGitHub, TokenScope{Resource: "repos", Permission: "admin"}, []string{"repo"}, false},
		{"github actions:write", forge.ForgeGitHub, TokenScope{Resource: "actions", Permission: "write"}, []string{"repo"}, false},
		{"github packages:read", forge.ForgeGitHub, TokenScope{Resource: "packages", Permission: "read"}, []string{"read:packages"}, false},
		{"github packages:write", forge.ForgeGitHub, TokenScope{Resource: "packages", Permission: "write"}, []string{"write:packages"}, false},

		// GitLab mappings
		{"gitlab issues:read", forge.ForgeGitLab, TokenScope{Resource: "issues", Permission: "read"}, []string{"read_api"}, false},
		{"gitlab issues:write", forge.ForgeGitLab, TokenScope{Resource: "issues", Permission: "write"}, []string{"api"}, false},
		{"gitlab repos:read", forge.ForgeGitLab, TokenScope{Resource: "repos", Permission: "read"}, []string{"read_repository"}, false},
		{"gitlab repos:write", forge.ForgeGitLab, TokenScope{Resource: "repos", Permission: "write"}, []string{"write_repository"}, false},
		{"gitlab packages:read", forge.ForgeGitLab, TokenScope{Resource: "packages", Permission: "read"}, []string{"read_api"}, false},

		// Gitea mappings
		{"gitea issues:read", forge.ForgeGitea, TokenScope{Resource: "issues", Permission: "read"}, []string{"read:issue"}, false},
		{"gitea issues:write", forge.ForgeGitea, TokenScope{Resource: "issues", Permission: "write"}, []string{"write:issue"}, false},
		{"gitea pulls:read", forge.ForgeGitea, TokenScope{Resource: "pulls", Permission: "read"}, []string{"read:issue"}, false},
		{"gitea repos:read", forge.ForgeGitea, TokenScope{Resource: "repos", Permission: "read"}, []string{"read:repository"}, false},
		{"gitea repos:write", forge.ForgeGitea, TokenScope{Resource: "repos", Permission: "write"}, []string{"write:repository"}, false},
		{"gitea packages:read", forge.ForgeGitea, TokenScope{Resource: "packages", Permission: "read"}, []string{"read:package"}, false},
		{"gitea packages:write", forge.ForgeGitea, TokenScope{Resource: "packages", Permission: "write"}, []string{"write:package"}, false},
		{"gitea actions:read", forge.ForgeGitea, TokenScope{Resource: "actions", Permission: "read"}, []string{"read:repository"}, false},

		// Bitbucket — error
		{"bitbucket issues:read", forge.ForgeBitbucket, TokenScope{Resource: "issues", Permission: "read"}, nil, true},

		// Unknown forge — error
		{"unknown forge", forge.ForgeUnknown, TokenScope{Resource: "issues", Permission: "read"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResolver(tt.forgeType)
			got, err := r.Resolve(tt.scope)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("Resolve() = %v, want %v", got, tt.want)
					return
				}
				for i, g := range got {
					if g != tt.want[i] {
						t.Errorf("Resolve()[%d] = %q, want %q", i, g, tt.want[i])
					}
				}
			}
		})
	}
}

func TestResolverUnknownResource(t *testing.T) {
	r := NewResolver(forge.ForgeGitHub)
	got, err := r.Resolve(TokenScope{Resource: "custom", Permission: "read"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unknown resources pass through
	if len(got) != 1 || got[0] != "custom:read" {
		t.Errorf("got %v, want [custom:read]", got)
	}
}
