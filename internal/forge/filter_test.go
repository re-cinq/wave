package forge

import (
	"reflect"
	"testing"
)

func TestFilterPipelines(t *testing.T) {
	allPipelines := []string{
		"gh-implement",
		"gh-pr-review",
		"gh-issue-triage",
		"gl-merge-request",
		"gl-deploy",
		"bb-pull-request",
		"gt-issue-flow",
		"prototype",
		"hotfix",
		"speckit-flow",
	}

	tests := []struct {
		name      string
		forgeType ForgeType
		input     []string
		want      []string
	}{
		{
			name:      "GitHub filter",
			forgeType: GitHub,
			input:     allPipelines,
			want:      []string{"gh-implement", "gh-pr-review", "gh-issue-triage", "prototype", "hotfix", "speckit-flow"},
		},
		{
			name:      "GitLab filter",
			forgeType: GitLab,
			input:     allPipelines,
			want:      []string{"gl-merge-request", "gl-deploy", "prototype", "hotfix", "speckit-flow"},
		},
		{
			name:      "Bitbucket filter",
			forgeType: Bitbucket,
			input:     allPipelines,
			want:      []string{"bb-pull-request", "prototype", "hotfix", "speckit-flow"},
		},
		{
			name:      "Gitea filter",
			forgeType: Gitea,
			input:     allPipelines,
			want:      []string{"gt-issue-flow", "prototype", "hotfix", "speckit-flow"},
		},
		{
			name:      "Unknown returns all",
			forgeType: Unknown,
			input:     allPipelines,
			want:      allPipelines,
		},
		{
			name:      "Empty input",
			forgeType: GitHub,
			input:     []string{},
			want:      nil,
		},
		{
			name:      "Nil input",
			forgeType: GitHub,
			input:     nil,
			want:      nil,
		},
		{
			name:      "Only universal pipelines",
			forgeType: GitHub,
			input:     []string{"prototype", "hotfix"},
			want:      []string{"prototype", "hotfix"},
		},
		{
			name:      "Only forge pipelines no match",
			forgeType: GitHub,
			input:     []string{"gl-deploy", "bb-pull-request"},
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterPipelines(tt.forgeType, tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterPipelines(%q) = %v, want %v", tt.forgeType, got, tt.want)
			}
		})
	}
}

func TestHasForgePrefix(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"gh-implement", true},
		{"gl-deploy", true},
		{"bb-pull-request", true},
		{"gt-issue-flow", true},
		{"prototype", false},
		{"hotfix", false},
		{"speckit-flow", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasForgePrefix(tt.name)
			if got != tt.want {
				t.Errorf("hasForgePrefix(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
