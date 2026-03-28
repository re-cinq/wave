package cost

import "testing"

func TestLookupContextWindow(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{"claude-opus", 1_000_000},
		{"claude-opus-4-6", 1_000_000},
		{"claude-sonnet", 200_000},
		{"claude-haiku-4-5", 200_000},
		{"gpt-4o", 128_000},
		{"gpt-4o-mini", 128_000},
		{"gemini-2.5-pro", 1_000_000},
		{"unknown-model", DefaultContextWindow},
		{"", DefaultContextWindow},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := LookupContextWindow(tt.model)
			if got != tt.want {
				t.Errorf("LookupContextWindow(%q) = %d, want %d", tt.model, got, tt.want)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		bytes int
		want  int
	}{
		{0, 0},
		{400, 100},
		{4000, 1000},
		{800_000, 200_000},
	}

	for _, tt := range tests {
		got := EstimateTokens(tt.bytes)
		if got != tt.want {
			t.Errorf("EstimateTokens(%d) = %d, want %d", tt.bytes, got, tt.want)
		}
	}
}

func TestCheckIronRule(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		promptBytes int
		wantStatus  IronRuleStatus
	}{
		{
			name:        "small prompt OK",
			model:       "claude-opus",
			promptBytes: 100_000, // ~25k tokens, well under 1M window
			wantStatus:  IronRuleOK,
		},
		{
			name:        "80% warning for sonnet",
			model:       "claude-sonnet",
			promptBytes: 680_000, // ~170k tokens, 85% of 200k window
			wantStatus:  IronRuleWarning,
		},
		{
			name:        "95% fail for sonnet",
			model:       "claude-sonnet",
			promptBytes: 780_000, // ~195k tokens, 97.5% of 200k window
			wantStatus:  IronRuleFail,
		},
		{
			name:        "large prompt OK for opus (1M window)",
			model:       "claude-opus",
			promptBytes: 2_000_000, // ~500k tokens, 50% of 1M window
			wantStatus:  IronRuleOK,
		},
		{
			name:        "zero bytes OK",
			model:       "claude-opus",
			promptBytes: 0,
			wantStatus:  IronRuleOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, msg := CheckIronRule(tt.model, tt.promptBytes)
			if status != tt.wantStatus {
				t.Errorf("CheckIronRule() status = %d, want %d (msg: %s)", status, tt.wantStatus, msg)
			}
			if status == IronRuleOK && msg != "" {
				t.Errorf("expected empty message for OK status, got: %s", msg)
			}
			if status != IronRuleOK && msg == "" {
				t.Errorf("expected non-empty message for non-OK status")
			}
		})
	}
}
