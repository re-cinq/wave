package webui

import "testing"

func TestFormatTokensShort(t *testing.T) {
	tests := []struct {
		name string
		n    int64
		want string
	}{
		{"zero", 0, ""},
		{"one", 1, "1"},
		{"999", 999, "999"},
		{"1000", 1000, "1.0k"},
		{"1500", 1500, "1.5k"},
		{"10000", 10_000, "10.0k"},
		{"999999", 999_999, "1000.0k"},
		{"1M", 1_000_000, "1.0M"},
		{"1_5M", 1_500_000, "1.5M"},
		{"10M", 10_000_000, "10.0M"},
		{"negative_small", -1, "-1"},
		{"500", 500, "500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokensShort(tt.n)
			if got != tt.want {
				t.Errorf("formatTokensShort(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
