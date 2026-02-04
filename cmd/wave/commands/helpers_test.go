package commands

import (
	"testing"
	"time"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"bytes", 100, "100 B"},
		{"max bytes", 1023, "1023 B"},
		{"one KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"one MB", 1024 * 1024, "1.0 MB"},
		{"1.5 MB", 1536 * 1024, "1.5 MB"},
		{"one GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"1.5 GB", 1536 * 1024 * 1024, "1.5 GB"},
		{"one TB", 1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.expected {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"empty string", "", 0, false},
		{"hours", "2h", 2 * time.Hour, false},
		{"minutes", "30m", 30 * time.Minute, false},
		{"seconds", "45s", 45 * time.Second, false},
		{"hours and minutes", "1h30m", 90 * time.Minute, false},
		{"days", "7d", 7 * 24 * time.Hour, false},
		{"days and hours", "1d12h", 36 * time.Hour, false},
		{"days and minutes", "2d30m", 48*time.Hour + 30*time.Minute, false},
		{"invalid", "invalid", 0, true},
		{"invalid days", "xd", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name     string
		tokens   int
		expected string
	}{
		{"zero", 0, "0"},
		{"small", 500, "500"},
		{"under thousand", 999, "999"},
		{"one thousand", 1000, "1k"},
		{"thousands", 45000, "45k"},
		{"under million", 999999, "999k"},
		{"one million", 1000000, "1M"},
		{"millions", 12000000, "12M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokens(tt.tokens)
			if got != tt.expected {
				t.Errorf("formatTokens(%d) = %q, want %q", tt.tokens, got, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"max of 4", "hello", 4, "h..."},
		{"empty string", "", 10, ""},
		{"unicode", "hello 世界", 8, "hello..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0m0s"},
		{"seconds only", 45 * time.Second, "0m45s"},
		{"one minute", time.Minute, "1m0s"},
		{"minutes and seconds", 2*time.Minute + 30*time.Second, "2m30s"},
		{"one hour", time.Hour, "1h0m"},
		{"hours and minutes", 2*time.Hour + 15*time.Minute, "2h15m"},
		{"negative duration", -30 * time.Second, "0m30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatElapsed(tt.duration)
			if got != tt.expected {
				t.Errorf("formatElapsed(%v) = %q, want %q", tt.duration, got, tt.expected)
			}
		})
	}
}
