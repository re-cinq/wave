package relay

import (
	"testing"
)

// =============================================================================
// Benchmarks for Performance-Critical Functions
// =============================================================================

func BenchmarkRelayMonitor_ShouldCompact(b *testing.B) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      200000,
	}, adapter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ShouldCompact(160000, 80)
	}
}

func BenchmarkRelayMonitor_ShouldCompactWithWindow(b *testing.B) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      200000,
	}, adapter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ShouldCompactWithWindow(160000, 200000, 80)
	}
}
