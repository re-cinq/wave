package pipeline

import "testing"

func TestResolveDeprecatedName(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantResolved   string
		wantDeprecated bool
	}{
		// Forge prefix resolution
		{"gh prefix", "gh-implement", "implement", true},
		{"gl prefix", "gl-research", "research", true},
		{"bb prefix", "bb-scope", "scope", true},
		{"gt prefix", "gt-refresh", "refresh", true},
		{"gh with hyphenated base", "gh-pr-review", "pr-review", true},

		// Taxonomy resolution
		{"old debug", "debug", "ops-debug", true},
		{"old hotfix", "hotfix", "impl-hotfix", true},
		{"old adr", "adr", "plan-adr", true},
		{"old changelog", "changelog", "doc-changelog", true},
		{"old dead-code", "dead-code", "audit-dead-code", true},
		{"old security-scan", "security-scan", "audit-security", true},
		{"old speckit-flow", "speckit-flow", "plan-speckit", true},
		{"old smoke-test", "smoke-test", "test-smoke", true},
		{"old plan", "plan", "plan-task", true},
		{"old prototype", "prototype", "plan-prototype", true},
		{"old supervise", "supervise", "ops-supervise", true},

		// Forge + taxonomy (double resolution)
		{"gh-debug", "gh-debug", "ops-debug", true},
		{"gh-hotfix", "gh-hotfix", "impl-hotfix", true},

		// No change needed
		{"wave prefix is not forge", "wave-evolve", "wave-evolve", false},
		{"already unified", "implement", "implement", false},
		{"already taxonomy", "ops-debug", "ops-debug", false},
		{"already taxonomy impl", "impl-hotfix", "impl-hotfix", false},
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
