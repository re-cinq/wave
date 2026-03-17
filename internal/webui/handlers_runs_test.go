package webui

import (
	"testing"
	"time"
)

func TestFormatDurationValue(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, "<1s"},
		{"sub-second", 500 * time.Millisecond, "<1s"},
		{"one second", time.Second, "1s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"exactly 1 minute", time.Minute, "1m"},
		{"1 minute 30 seconds", 90 * time.Second, "1m 30s"},
		{"2 minutes 34 seconds", 154 * time.Second, "2m 34s"},
		{"10 minutes", 10 * time.Minute, "10m"},
		{"exactly 1 hour", time.Hour, "1h"},
		{"1 hour 5 minutes", time.Hour + 5*time.Minute, "1h 5m"},
		{"2 hours 30 minutes", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"large value", 25*time.Hour + 15*time.Minute, "25h 15m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDurationValue(tt.duration)
			if got != tt.want {
				t.Errorf("formatDurationValue(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}
