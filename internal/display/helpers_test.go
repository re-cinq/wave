package display

import (
	"testing"
)

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
		want   string
	}{
		{"zero", 0, "0"},
		{"small", 842, "842"},
		{"just under 1k", 999, "999"},
		{"exactly 1k", 1000, "1.0k"},
		{"over 1k", 1500, "1.5k"},
		{"large", 10000, "10.0k"},
		{"very large", 100000, "100.0k"},
		{"just_under_1M", 999999, "1000.0k"},
		{"exactly_1M", 1000000, "1.0M"},
		{"1.5M", 1500000, "1.5M"},
		{"just_under_1B", 999999999, "1000.0M"},
		{"exactly_1B", 1000000000, "1.0B"},
		{"2.3B", 2300000000, "2.3B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTokenCount(tt.tokens)
			if got != tt.want {
				t.Errorf("FormatTokenCount(%d) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkFormatTokenCount_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatTokenCount(500)
	}
}

func BenchmarkFormatTokenCount_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatTokenCount(150000)
	}
}

func BenchmarkFormatter_Bold(b *testing.B) {
	f := NewFormatter()
	text := "sample text"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Bold(text)
	}
}

func BenchmarkFormatter_Colorize(b *testing.B) {
	f := NewFormatter()
	text := "sample text"
	color := "\033[34m"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Colorize(text, color)
	}
}

func BenchmarkANSICodec_Colorize(b *testing.B) {
	codec := NewANSICodec()
	text := "sample text"
	color := "\033[34m"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Colorize(text, color)
	}
}

func BenchmarkGetUnicodeCharSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetUnicodeCharSet()
	}
}
