package forge

import (
	"reflect"
	"testing"
)

func TestResolvePersonaCandidates(t *testing.T) {
	tests := []struct {
		name     string
		persona  string
		fi       ForgeInfo
		expected []string
	}{
		{"no template returns persona unchanged", "navigator", ForgeInfo{Type: ForgeGitHub}, []string{"navigator"}},
		{"known forge resolves spaced template", "{{ forge.type }}-analyst", ForgeInfo{Type: ForgeGitHub}, []string{"github-analyst"}},
		{"known forge resolves compact template", "{{forge.type}}-analyst", ForgeInfo{Type: ForgeGitLab}, []string{"gitlab-analyst"}},
		{"unknown forge expands to all 4 variants", "{{ forge.type }}-analyst", ForgeInfo{Type: ForgeUnknown}, []string{"github-analyst", "gitlab-analyst", "gitea-analyst", "bitbucket-analyst"}},
		{"empty forge type expands to all 4 variants", "{{ forge.type }}-analyst", ForgeInfo{}, []string{"github-analyst", "gitlab-analyst", "gitea-analyst", "bitbucket-analyst"}},
		{"local forge (no remote) expands to all 4 variants", "{{ forge.type }}-analyst", ForgeInfo{Type: ForgeLocal}, []string{"github-analyst", "gitlab-analyst", "gitea-analyst", "bitbucket-analyst"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolvePersonaCandidates(tt.persona, tt.fi); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ResolvePersonaCandidates(%q) = %v, want %v", tt.persona, got, tt.expected)
			}
		})
	}
}

func TestIsForgeTemplatedPersona(t *testing.T) {
	for persona, want := range map[string]bool{
		"navigator":                  false,
		"{{ forge.type }}-commenter": true,
		"{{forge.type}}-commenter":   true,
		"{{ pipeline_id }}":          false,
	} {
		if got := IsForgeTemplatedPersona(persona); got != want {
			t.Errorf("IsForgeTemplatedPersona(%q) = %v, want %v", persona, got, want)
		}
	}
}
