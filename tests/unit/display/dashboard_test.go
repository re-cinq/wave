package display_test

import (
	"testing"

	"github.com/recinq/wave/internal/display"
)

// TestANSICodecWithConfig tests ANSI codec with different configurations.
func TestANSICodecWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		colorMode string
		asciiOnly bool
	}{
		{"auto mode", "auto", false},
		{"color on", "on", false},
		{"color off", "off", false},
		{"ascii only", "auto", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := display.NewANSICodecWithConfig(tt.colorMode, tt.asciiOnly)
			if codec == nil {
				t.Fatal("expected NewANSICodecWithConfig to return non-nil")
			}

			text := "test"
			colored := codec.Success(text)

			// ASCII-only or color-off should not add escape codes
			if tt.asciiOnly || tt.colorMode == "off" {
				if colored != text {
					t.Errorf("expected no color codes in ASCII-only/off mode, got %q", colored)
				}
			}
		})
	}
}

// TestUnicodeCharSets tests Unicode character set selection.
func TestUnicodeCharSets(t *testing.T) {
	charSet := display.GetUnicodeCharSet()

	// Should return a valid character set
	if charSet.CheckMark == "" {
		t.Error("expected CheckMark to be non-empty")
	}

	if charSet.CrossMark == "" {
		t.Error("expected CrossMark to be non-empty")
	}

	if charSet.Bullet == "" {
		t.Error("expected Bullet to be non-empty")
	}

	if len(charSet.Spinner) != 4 {
		t.Errorf("expected Spinner to have 4 frames, got %d", len(charSet.Spinner))
	}

	if len(charSet.ProgressBar) != 5 {
		t.Errorf("expected ProgressBar to have 5 characters, got %d", len(charSet.ProgressBar))
	}
}

// TestSelectColorPalette tests color palette selection.
func TestSelectColorPalette(t *testing.T) {
	tests := []struct {
		name        string
		colorMode   string
		asciiOnly   bool
		expectASCII bool
	}{
		{"ascii only", "auto", true, true},
		{"color off", "off", false, true},
		{"color on", "on", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette := display.SelectColorPalette(tt.colorMode, tt.asciiOnly)

			if tt.expectASCII {
				// ASCII-only should have empty color codes
				if palette.Primary != "" || palette.Success != "" {
					t.Error("expected ASCII-only palette to have empty color codes")
				}
			}
		})
	}
}
