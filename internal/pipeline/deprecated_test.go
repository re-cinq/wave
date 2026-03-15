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
		{"gh prefix", "gh-implement", "impl-issue", true},
		{"gl prefix", "gl-research", "plan-research", true},
		{"bb prefix", "bb-scope", "plan-scope", true},
		{"gt prefix", "gt-refresh", "ops-refresh", true},
		{"gh with hyphenated base", "gh-pr-review", "ops-pr-review", true},

		// Taxonomy resolution
		{"old debug", "debug", "ops-debug", true},
		{"old hotfix", "hotfix", "impl-hotfix", true},
		{"old adr", "adr", "plan-adr", true},
		{"old changelog", "changelog", "doc-changelog", true},
		{"old dead-code", "dead-code", "audit-dead-code", true},
		{"old security-scan", "security-scan", "audit-security", true},
		{"old speckit-flow", "speckit-flow", "impl-speckit", true},
		{"old smoke-test", "smoke-test", "test-smoke", true},
		{"old plan", "plan", "plan-task", true},
		{"old prototype", "prototype", "impl-prototype", true},
		{"old supervise", "supervise", "ops-supervise", true},
		{"old plan-speckit", "plan-speckit", "impl-speckit", true},
		{"old plan-prototype", "plan-prototype", "impl-prototype", true},
		{"old consolidate", "consolidate", "audit-consolidate", true},
		{"old ops-consolidate", "ops-consolidate", "audit-consolidate", true},
		{"old implement", "implement", "impl-issue", true},
		{"old pr-review", "pr-review", "ops-pr-review", true},
		{"old refresh", "refresh", "ops-refresh", true},
		{"old research", "research", "plan-research", true},
		{"old rewrite", "rewrite", "ops-rewrite", true},
		{"old scope", "scope", "plan-scope", true},

		// Forge + taxonomy (double resolution)
		{"gh-debug", "gh-debug", "ops-debug", true},
		{"gh-hotfix", "gh-hotfix", "impl-hotfix", true},

		// No change needed
		{"wave prefix is not forge", "wave-evolve", "wave-evolve", false},
		{"already taxonomy", "ops-debug", "ops-debug", false},
		{"already taxonomy impl", "impl-hotfix", "impl-hotfix", false},
		{"already taxonomy impl-issue", "impl-issue", "impl-issue", false},
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
