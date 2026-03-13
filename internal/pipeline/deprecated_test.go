package pipeline

import "testing"

func TestResolveDeprecatedName(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantResolved   string
		wantDeprecated bool
	}{
		{"gh prefix", "gh-implement", "implement", true},
		{"gl prefix", "gl-research", "research", true},
		{"bb prefix", "bb-scope", "scope", true},
		{"gt prefix", "gt-refresh", "refresh", true},
		{"gh with hyphenated base", "gh-pr-review", "pr-review", true},
		{"no prefix", "speckit-flow", "speckit-flow", false},
		{"simple name", "debug", "debug", false},
		{"wave prefix is not forge", "wave-evolve", "wave-evolve", false},
		{"already unified", "implement", "implement", false},
		{"partial prefix no match", "g-test", "g-test", false},
		{"empty string", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, deprecated := ResolveDeprecatedName(tt.input)
			if resolved != tt.wantResolved {
				t.Errorf("ResolveDeprecatedName(%q) resolved = %q, want %q", tt.input, resolved, tt.wantResolved)
			}
			if deprecated != tt.wantDeprecated {
				t.Errorf("ResolveDeprecatedName(%q) deprecated = %v, want %v", tt.input, deprecated, tt.wantDeprecated)
			}
		})
	}
}
