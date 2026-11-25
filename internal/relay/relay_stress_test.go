package relay

import (
	"testing"
)

// =============================================================================
// Stress Tests for Large Payloads and Edge Cases
// =============================================================================

func TestRelayMonitor_LargeTokenCounts(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      1000000, // 1M context window
	}, adapter)

	testCases := []struct {
		name       string
		tokensUsed int
		expected   bool
	}{
		{"very large token count", 950000, true}, // 95% of 1M
		{"maximum token count", 1000000, true},   // 100% of 1M
		{"over context window", 1500000, true},   // 150% of 1M
		{"just under threshold", 799999, false},  // Just under 80%
		{"exactly at threshold", 800000, true},   // Exactly 80%
		{"extreme token count", 10000000, true},  // 10M tokens
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := m.ShouldCompact(tc.tokensUsed, 80)
			if result != tc.expected {
				t.Errorf("expected %v for %d tokens, got %v", tc.expected, tc.tokensUsed, result)
			}
		})
	}
}
