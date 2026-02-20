package display

import (
	"os"
	"strings"
	"testing"
)

func TestNewCapabilityDetector(t *testing.T) {
	cd := NewCapabilityDetector()
	if cd == nil {
		t.Fatal("NewCapabilityDetector returned nil")
	}
}

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

func TestSelectAnimationType(t *testing.T) {
	tests := []struct {
		name      string
		preferred AnimationType
	}{
		{"empty preferred", ""},
		{"dots preferred", AnimationDots},
		{"spinner preferred", AnimationSpinner},
		{"line preferred", AnimationLine},
		{"bars preferred", AnimationBars},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectAnimationType(tt.preferred)
			if result == "" {
				t.Error("SelectAnimationType should not return empty string")
			}
		})
	}
}

func TestGetOptimalDisplayConfig(t *testing.T) {
	config := GetOptimalDisplayConfig()

	// Verify config has valid values
	if config.RefreshRate < 1 || config.RefreshRate > 60 {
		t.Errorf("RefreshRate should be between 1 and 60, got %d", config.RefreshRate)
	}

	if config.ColorMode != "auto" && config.ColorMode != "on" && config.ColorMode != "off" {
		t.Errorf("ColorMode should be auto/on/off, got %q", config.ColorMode)
	}
}

func TestGetOptimalDisplayConfig_WithNoColor(t *testing.T) {
	// Save and restore NO_COLOR
	oldNoColor := os.Getenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", oldNoColor)

	os.Setenv("NO_COLOR", "1")
	config := GetOptimalDisplayConfig()

	if config.ColorMode != "off" {
		t.Errorf("ColorMode should be 'off' when NO_COLOR is set, got %q", config.ColorMode)
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

func TestANSICodec_TextFormatting(t *testing.T) {
	codec := NewANSICodecWithConfig("on", false)
	text := "formatted"

	tests := []struct {
		name    string
		fn      func(string) string
		wantSeq string
	}{
		{"Bold", codec.Bold, "\033[1m"},
		{"Dim", codec.Dim, "\033[2m"},
		{"Underline", codec.Underline, "\033[4m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(text)
			if codec.ansi {
				if !strings.Contains(result, tt.wantSeq) {
					t.Errorf("Result should contain formatting sequence %s", tt.wantSeq)
				}
			}
			if !strings.Contains(result, text) {
				t.Error("Result should contain original text")
			}
		})
	}
}

func TestANSICodec_ClearLine(t *testing.T) {
	codec := NewANSICodecWithConfig("on", false)
	result := codec.ClearLine()
	if codec.ansi && result != "\033[2K" {
		t.Errorf("ClearLine() = %q, want %q", result, "\033[2K")
	}
}

func TestANSICodec_CursorUp(t *testing.T) {
	codec := NewANSICodecWithConfig("on", false)

	tests := []struct {
		name  string
		lines int
		want  string
	}{
		{"one line", 1, "\033[A"},
		{"multiple lines", 5, "\033[5A"},
		{"zero lines", 0, ""},
		{"negative lines", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codec.CursorUp(tt.lines)
			if codec.ansi && result != tt.want {
				t.Errorf("CursorUp(%d) = %q, want %q", tt.lines, result, tt.want)
			}
		})
	}
}

func TestANSICodec_CursorHome(t *testing.T) {
	codec := NewANSICodecWithConfig("on", false)
	result := codec.CursorHome()
	if codec.ansi && result != "\033[H" {
		t.Errorf("CursorHome() = %q, want %q", result, "\033[H")
	}
}

func TestANSICodec_CursorVisibility(t *testing.T) {
	codec := NewANSICodecWithConfig("on", false)

	t.Run("HideCursor", func(t *testing.T) {
		result := codec.HideCursor()
		if codec.ansi && result != "\033[?25l" {
			t.Errorf("HideCursor() = %q, want %q", result, "\033[?25l")
		}
	})

	t.Run("ShowCursor", func(t *testing.T) {
		result := codec.ShowCursor()
		if codec.ansi && result != "\033[?25h" {
			t.Errorf("ShowCursor() = %q, want %q", result, "\033[?25h")
		}
	})
}

func TestANSICodec_CursorPosition(t *testing.T) {
	codec := NewANSICodecWithConfig("on", false)

	t.Run("SaveCursorPosition", func(t *testing.T) {
		result := codec.SaveCursorPosition()
		if codec.ansi && result != "\033[s" {
			t.Errorf("SaveCursorPosition() = %q, want %q", result, "\033[s")
		}
	})

	t.Run("RestoreCursorPosition", func(t *testing.T) {
		result := codec.RestoreCursorPosition()
		if codec.ansi && result != "\033[u" {
			t.Errorf("RestoreCursorPosition() = %q, want %q", result, "\033[u")
		}
	})
}

func TestANSICodec_Reset(t *testing.T) {
	codec := NewANSICodecWithConfig("on", false)
	result := codec.Reset()
	// Reset should return the reset code from the color palette
	if result == "" && codec.colors.Reset != "" {
		t.Error("Reset should return the color palette reset code")
	}
}

func TestANSICodec_DisabledMode(t *testing.T) {
	codec := NewANSICodecWithConfig("on", true) // ASCII only disables ANSI

	text := "test"

	// All formatting should return plain text when ANSI is disabled
	if codec.Bold(text) != text && !codec.ansi {
		t.Error("Bold should return plain text when ANSI disabled")
	}

	if codec.ClearLine() != "" && !codec.ansi {
		t.Error("ClearLine should return empty when ANSI disabled")
	}

	if codec.HideCursor() != "" && !codec.ansi {
		t.Error("HideCursor should return empty when ANSI disabled")
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

func TestNewTerminalColorContext(t *testing.T) {
	tcc := NewTerminalColorContext()
	if tcc == nil {
		t.Fatal("NewTerminalColorContext returned nil")
	}
	if tcc.codec == nil {
		t.Error("TerminalColorContext should have codec initialized")
	}
}

func TestTerminalColorContext_FormatState(t *testing.T) {
	tcc := NewTerminalColorContext()

	tests := []struct {
		state ProgressState
	}{
		{StateCompleted},
		{StateFailed},
		{StateRunning},
		{StateSkipped},
		{StateCancelled},
		{StateNotStarted},
		{"unknown_state"}, // Test default case
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			result := tcc.FormatState(tt.state)
			if result == "" {
				t.Errorf("FormatState(%s) should not return empty", tt.state)
			}
			// Result should contain the state name
			if !strings.Contains(result, string(tt.state)) {
				t.Errorf("FormatState(%s) result should contain state name", tt.state)
			}
		})
	}
}

func TestTerminalColorContext_GetStateIcon(t *testing.T) {
	tcc := NewTerminalColorContext()

	tests := []struct {
		state ProgressState
	}{
		{StateCompleted},
		{StateFailed},
		{StateRunning},
		{StateSkipped},
		{StateCancelled},
		{StateNotStarted},
		{"unknown_state"}, // Test default case
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			icon := tcc.GetStateIcon(tt.state)
			if icon == "" {
				t.Errorf("GetStateIcon(%s) should not return empty", tt.state)
			}
		})
	}
}

func TestDefaultColorSchemes(t *testing.T) {
	schemes := []struct {
		name   string
		scheme ColorPalette
	}{
		{"DefaultColorScheme", DefaultColorScheme},
		{"DarkColorScheme", DarkColorScheme},
		{"LightColorScheme", LightColorScheme},
		{"HighContrastColorScheme", HighContrastColorScheme},
	}

	for _, tt := range schemes {
		t.Run(tt.name, func(t *testing.T) {
			if tt.scheme.Reset != "\033[0m" {
				t.Errorf("%s Reset = %q, want %q", tt.name, tt.scheme.Reset, "\033[0m")
			}
			// All color codes should be non-empty for non-ASCII schemes
			if tt.scheme.Primary == "" {
				t.Errorf("%s Primary should not be empty", tt.name)
			}
		})
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

func TestGetColorSchemeByName(t *testing.T) {
	tests := []struct {
		theme    string
		expected ColorPalette
	}{
		{"dark", DarkColorScheme},
		{"light", LightColorScheme},
		{"high_contrast", HighContrastColorScheme},
		{"default", DefaultColorScheme},
		{"unknown", DefaultColorScheme},
		{"", DefaultColorScheme},
	}

	for _, tt := range tests {
		t.Run(tt.theme, func(t *testing.T) {
			result := GetColorSchemeByName(tt.theme)
			if result.Primary != tt.expected.Primary {
				t.Errorf("GetColorSchemeByName(%q) Primary = %q, want %q",
					tt.theme, result.Primary, tt.expected.Primary)
			}
		})
	}
}
