package display

import (
	"strings"
	"testing"
)

func TestSelectColorPalette(t *testing.T) {
	tests := []struct {
		name      string
		colorMode string
		asciiOnly bool
		wantEmpty bool
	}{
		{"ascii only mode", "auto", true, true},
		{"color off mode", "off", false, true},
		{"color on mode", "on", false, false},
		{"auto mode", "auto", false, false}, // May or may not be empty depending on terminal
		{"unknown mode", "unknown", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette := SelectColorPalette(tt.colorMode, tt.asciiOnly)
			if tt.wantEmpty {
				if palette.Primary != "" || palette.Success != "" {
					t.Error("Expected empty color palette for ascii-only or color-off mode")
				}
			}
		})
	}
}

func TestNewANSICodec(t *testing.T) {
	codec := NewANSICodec()
	if codec == nil {
		t.Fatal("NewANSICodec returned nil")
	}
}

func TestNewANSICodecWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		colorMode string
		asciiOnly bool
	}{
		{"auto mode", "auto", false},
		{"on mode", "on", false},
		{"off mode", "off", false},
		{"ascii only", "auto", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewANSICodecWithConfig(tt.colorMode, tt.asciiOnly)
			if codec == nil {
				t.Fatal("NewANSICodecWithConfig returned nil")
			}
		})
	}
}

func TestANSICodec_Colorize(t *testing.T) {
	codec := NewANSICodecWithConfig("on", false)
	text := "test text"

	tests := []struct {
		name      string
		colorCode string
		wantCode  bool
	}{
		{"with color code", "\033[34m", true},
		{"empty color code", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codec.Colorize(text, tt.colorCode)
			if !strings.Contains(result, text) {
				t.Error("Result should contain original text")
			}
			if tt.wantCode && codec.ansi && !strings.Contains(result, tt.colorCode) {
				t.Error("Result should contain color code when ANSI is enabled")
			}
		})
	}
}

func TestANSICodec_ColorMethods(t *testing.T) {
	codec := NewANSICodecWithConfig("on", false)
	text := "test"

	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"Success", codec.Success},
		{"Error", codec.Error},
		{"Warning", codec.Warning},
		{"Muted", codec.Muted},
		{"Primary", codec.Primary},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(text)
			if !strings.Contains(result, text) {
				t.Errorf("%s should contain original text", tt.name)
			}
		})
	}
}

func TestGetUnicodeCharSet(t *testing.T) {
	charSet := GetUnicodeCharSet()

	// CharSet should have non-empty values
	if charSet.CheckMark == "" {
		t.Error("CheckMark should not be empty")
	}
	if charSet.CrossMark == "" {
		t.Error("CrossMark should not be empty")
	}
	if charSet.Block == "" {
		t.Error("Block should not be empty")
	}
	if charSet.Spinner[0] == "" {
		t.Error("Spinner frames should not be empty")
	}
}

func TestUnicodeCharSetFull(t *testing.T) {
	cs := UnicodeCharSetFull

	if cs.CheckMark != "✓" {
		t.Errorf("Full charset CheckMark = %q, want %q", cs.CheckMark, "✓")
	}
	if cs.CrossMark != "✗" {
		t.Errorf("Full charset CrossMark = %q, want %q", cs.CrossMark, "✗")
	}
	if cs.Block != "█" {
		t.Errorf("Full charset Block = %q, want %q", cs.Block, "█")
	}
}

func TestUnicodeCharSetASCII(t *testing.T) {
	cs := UnicodeCharSetASCII

	if cs.CheckMark != "[OK]" {
		t.Errorf("ASCII charset CheckMark = %q, want %q", cs.CheckMark, "[OK]")
	}
	if cs.CrossMark != "[X]" {
		t.Errorf("ASCII charset CrossMark = %q, want %q", cs.CrossMark, "[X]")
	}
	if cs.Block != "#" {
		t.Errorf("ASCII charset Block = %q, want %q", cs.Block, "#")
	}
}

func TestAsciiOnlyColorScheme(t *testing.T) {
	if AsciiOnlyColorScheme.Primary != "" {
		t.Error("AsciiOnlyColorScheme Primary should be empty")
	}
	if AsciiOnlyColorScheme.Success != "" {
		t.Error("AsciiOnlyColorScheme Success should be empty")
	}
	if AsciiOnlyColorScheme.Reset != "" {
		t.Error("AsciiOnlyColorScheme Reset should be empty")
	}
}
