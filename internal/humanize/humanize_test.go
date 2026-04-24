package humanize

import (
	"testing"
	"time"
)

func TestDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "-"},
		{"seconds", 5 * time.Second, "5s"},
		{"sub-minute", 59 * time.Second, "59s"},
		{"exact minute", time.Minute, "1m0s"},
		{"minutes and seconds", 3*time.Minute + 15*time.Second, "3m15s"},
		{"just under hour", 59*time.Minute + 30*time.Second, "59m30s"},
		{"hour", time.Hour, "1h0m"},
		{"hours and minutes", 2*time.Hour + 30*time.Minute, "2h30m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Duration(tt.d); got != tt.want {
				t.Errorf("Duration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestTokenCount(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{"zero", 0, "-"},
		{"small", 42, "42"},
		{"999", 999, "999"},
		{"1k", 1000, "1k"},
		{"45k", 45_000, "45k"},
		{"1.5M", 1_500_000, "1.5M"},
		{"12.3M", 12_345_678, "12.3M"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TokenCount(tt.n); got != tt.want {
				t.Errorf("TokenCount(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestFileSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero", 0, "-"},
		{"bytes", 512, "512 B"},
		{"just under KB", 1023, "1023 B"},
		{"KB", 2 * 1024, "2.0 KB"},
		{"MB", 1536 * 1024, "1.5 MB"},
		{"GB", 2 * 1024 * 1024 * 1024, "2.0 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileSize(tt.bytes); got != tt.want {
				t.Errorf("FileSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
