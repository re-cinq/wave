package humanize

import (
	"testing"
)

func TestDurationMs(t *testing.T) {
	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{"zero", 0, "-"},
		{"negative", -100, "-"},
		{"sub-second", 250, "<1s"},
		{"just under second", 999, "<1s"},
		{"one second", 1000, "1s"},
		{"seconds", 5_000, "5s"},
		{"minute boundary", 60_000, "1m0s"},
		{"minutes and seconds", 195_000, "3m15s"},
		{"hour boundary", 3_600_000, "1h0m"},
		{"hours and minutes", 9_000_000, "2h30m"},
		{"large value", 36_000_000, "10h0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DurationMs(tt.ms); got != tt.want {
				t.Errorf("DurationMs(%d) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}
