package pipeline

import "testing"

func TestResolveDeprecatedName(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantName       string
		wantDeprecated bool
	}{
		{
			name:           "gh-implement resolves to implement",
			input:          "gh-implement",
			wantName:       "implement",
			wantDeprecated: true,
		},
		{
			name:           "gl-implement resolves to implement",
			input:          "gl-implement",
			wantName:       "implement",
			wantDeprecated: true,
		},
		{
			name:           "bb-implement resolves to implement",
			input:          "bb-implement",
			wantName:       "implement",
			wantDeprecated: true,
		},
		{
			name:           "gt-implement resolves to implement",
			input:          "gt-implement",
			wantName:       "implement",
			wantDeprecated: true,
		},
		{
			name:           "gh-scope resolves to scope",
			input:          "gh-scope",
			wantName:       "scope",
			wantDeprecated: true,
		},
		{
			name:           "gh-pr-review resolves to pr-review",
			input:          "gh-pr-review",
			wantName:       "pr-review",
			wantDeprecated: true,
		},
		{
			name:           "gh-research resolves to research",
			input:          "gh-research",
			wantName:       "research",
			wantDeprecated: true,
		},
		{
			name:           "gh-refresh resolves to refresh",
			input:          "gh-refresh",
			wantName:       "refresh",
			wantDeprecated: true,
		},
		{
			name:           "gh-rewrite resolves to rewrite",
			input:          "gh-rewrite",
			wantName:       "rewrite",
			wantDeprecated: true,
		},
		{
			name:           "implement is already unified",
			input:          "implement",
			wantName:       "implement",
			wantDeprecated: false,
		},
		{
			name:           "speckit-flow is not deprecated",
			input:          "speckit-flow",
			wantName:       "speckit-flow",
			wantDeprecated: false,
		},
		{
			name:           "debug is not deprecated",
			input:          "debug",
			wantName:       "debug",
			wantDeprecated: false,
		},
		{
			name:           "wave-evolve is not deprecated",
			input:          "wave-evolve",
			wantName:       "wave-evolve",
			wantDeprecated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotDeprecated := ResolveDeprecatedName(tt.input)
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotDeprecated != tt.wantDeprecated {
				t.Errorf("deprecated = %v, want %v", gotDeprecated, tt.wantDeprecated)
			}
		})
	}
}

func TestIsDeprecatedPipelineName(t *testing.T) {
	if !IsDeprecatedPipelineName("gh-implement") {
		t.Error("expected gh-implement to be deprecated")
	}
	if IsDeprecatedPipelineName("implement") {
		t.Error("expected implement to NOT be deprecated")
	}
	if IsDeprecatedPipelineName("speckit-flow") {
		t.Error("expected speckit-flow to NOT be deprecated")
	}
}

func TestStripForgePrefix(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantBase     string
		wantStripped bool
	}{
		{"gh prefix", "gh-implement", "implement", true},
		{"gl prefix", "gl-scope", "scope", true},
		{"bb prefix", "bb-research", "research", true},
		{"gt prefix", "gt-rewrite", "rewrite", true},
		{"no prefix", "implement", "implement", false},
		{"wave prefix not stripped", "wave-evolve", "wave-evolve", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBase, gotStripped := StripForgePrefix(tt.input)
			if gotBase != tt.wantBase {
				t.Errorf("base = %q, want %q", gotBase, tt.wantBase)
			}
			if gotStripped != tt.wantStripped {
				t.Errorf("stripped = %v, want %v", gotStripped, tt.wantStripped)
			}
		})
	}
}
