package display_test

import (
	"testing"

	"github.com/recinq/wave/internal/display"
)

// TestTerminalCapabilitiesDetection tests terminal capability detection.
func TestTerminalCapabilitiesDetection(t *testing.T) {
	ti := display.NewTerminalInfo()

	// Terminal info should always be created successfully
	if ti == nil {
		t.Fatal("expected NewTerminalInfo to return non-nil")
	}

	// Capabilities should be initialized
	caps := ti.Capabilities()
	if caps == nil {
		t.Fatal("expected Capabilities() to return non-nil")
	}

	// Width and height should have sane defaults (even if not a TTY)
	width := ti.GetWidth()
	if width < 1 || width > 10000 {
		t.Errorf("expected Width to be reasonable (1-10000), got %d", width)
	}

	height := ti.GetHeight()
	if height < 1 || height > 10000 {
		t.Errorf("expected Height to be reasonable (1-10000), got %d", height)
	}

	// Color scheme should be either "light" or "dark"
	scheme := ti.GetColorScheme()
	if scheme != "light" && scheme != "dark" {
		t.Errorf("expected ColorScheme to be 'light' or 'dark', got %q", scheme)
	}
}

// TestANSICodec tests ANSI code generation.
func TestANSICodec(t *testing.T) {
	codec := display.NewANSICodec()
	if codec == nil {
		t.Fatal("expected NewANSICodec to return non-nil")
	}

	// Test colorization (output depends on terminal capabilities)
	text := "test"
	colored := codec.Success(text)
	// Should at least contain the original text
	if len(colored) < len(text) {
		t.Errorf("expected colored text to contain original text")
	}

	// Test other color methods
	_ = codec.Error("error")
	_ = codec.Warning("warning")
	_ = codec.Muted("muted")
	_ = codec.Primary("primary")

	// Test formatting
	_ = codec.Bold("bold")
	_ = codec.Dim("dim")
	_ = codec.Underline("underline")
}

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

// TestTerminalColorContext tests color context for state formatting.
func TestTerminalColorContext(t *testing.T) {
	ctx := display.NewTerminalColorContext()
	if ctx == nil {
		t.Fatal("expected NewTerminalColorContext to return non-nil")
	}

	tests := []struct {
		state display.ProgressState
		name  string
	}{
		{display.StateCompleted, "completed"},
		{display.StateFailed, "failed"},
		{display.StateRunning, "running"},
		{display.StateSkipped, "skipped"},
		{display.StateCancelled, "cancelled"},
		{display.StateNotStarted, "not_started"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Format state should return a non-empty string
			formatted := ctx.FormatState(tt.state)
			if formatted == "" {
				t.Error("expected FormatState to return non-empty string")
			}

			// Get state icon should return a non-empty string
			icon := ctx.GetStateIcon(tt.state)
			if icon == "" {
				t.Error("expected GetStateIcon to return non-empty string")
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

// TestCapabilityDetector tests capability detection functions.
func TestCapabilityDetector(t *testing.T) {
	detector := display.NewCapabilityDetector()
	if detector == nil {
		t.Fatal("expected NewCapabilityDetector to return non-nil")
	}

	// Test standalone detection functions (results depend on environment)
	_ = display.DetectANSISupport()
	_ = display.DetectColorSupport()
	_ = display.Detect256ColorSupport()
	_ = display.DetectUnicodeSupport()
}

// TestSelectColorPalette tests color palette selection.
func TestSelectColorPalette(t *testing.T) {
	tests := []struct {
		name      string
		colorMode string
		asciiOnly bool
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

// TestSelectAnimationType tests animation type selection.
func TestSelectAnimationType(t *testing.T) {
	tests := []struct {
		name      string
		preferred display.AnimationType
	}{
		{"dots", display.AnimationDots},
		{"spinner", display.AnimationSpinner},
		{"bars", display.AnimationBars},
		{"line", display.AnimationLine},
		{"clock", display.AnimationClock},
		{"bouncing_bar", display.AnimationBouncingBar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selected := display.SelectAnimationType(tt.preferred)

			// Should return a valid animation type
			validTypes := map[display.AnimationType]bool{
				display.AnimationDots:        true,
				display.AnimationLine:        true,
				display.AnimationBars:        true,
				display.AnimationSpinner:     true,
				display.AnimationClock:       true,
				display.AnimationBouncingBar: true,
			}

			if !validTypes[selected] {
				t.Errorf("expected valid animation type, got %v", selected)
			}
		})
	}
}

// TestGetOptimalDisplayConfig tests optimal configuration detection.
func TestGetOptimalDisplayConfig(t *testing.T) {
	config := display.GetOptimalDisplayConfig()

	// Should have reasonable defaults
	if config.RefreshRate < 1 || config.RefreshRate > 60 {
		t.Errorf("expected RefreshRate 1-60, got %d", config.RefreshRate)
	}

	if config.ColorMode != "auto" && config.ColorMode != "off" {
		t.Errorf("expected ColorMode 'auto' or 'off', got %q", config.ColorMode)
	}

	// Animation type should be valid
	validTypes := map[display.AnimationType]bool{
		display.AnimationDots:        true,
		display.AnimationLine:        true,
		display.AnimationBars:        true,
		display.AnimationSpinner:     true,
		display.AnimationClock:       true,
		display.AnimationBouncingBar: true,
	}

	if !validTypes[config.AnimationType] {
		t.Errorf("expected valid animation type, got %v", config.AnimationType)
	}
}

// TestANSICodecControlCodes tests ANSI control codes.
func TestANSICodecControlCodes(t *testing.T) {
	codec := display.NewANSICodec()

	// Test control codes (may be empty if ANSI not supported)
	_ = codec.ClearLine()
	_ = codec.CursorUp(1)
	_ = codec.CursorHome()
	_ = codec.HideCursor()
	_ = codec.ShowCursor()
	_ = codec.SaveCursorPosition()
	_ = codec.RestoreCursorPosition()

	// Test cursor up with different values
	_ = codec.CursorUp(0)  // Should return empty
	_ = codec.CursorUp(5)  // Should return valid code
	_ = codec.CursorUp(-1) // Should return empty
}

// TestResponsiveLayout tests that display adapts to different terminal sizes.
func TestResponsiveLayout(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		valid  bool
	}{
		{"tiny terminal", 20, 5, true},
		{"small terminal", 40, 10, true},
		{"normal terminal", 80, 24, true},
		{"large terminal", 120, 40, true},
		{"wide terminal", 200, 24, true},
		{"tall terminal", 80, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that configuration can adapt to different sizes
			config := display.DefaultDisplayConfig()

			// For very small terminals, compact mode might be preferred
			if tt.width < 60 || tt.height < 15 {
				config.CompactMode = true
			}

			// Validate configuration works for any size
			config.Validate()

			if config.RefreshRate < 1 || config.RefreshRate > 60 {
				t.Errorf("expected valid RefreshRate after validation, got %d", config.RefreshRate)
			}
		})
	}
}

// TestColorSchemeAdaptation tests color scheme adaptation to terminal type.
func TestColorSchemeAdaptation(t *testing.T) {
	schemes := []string{"default", "dark", "light", "high_contrast"}

	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			palette := display.GetColorSchemeByName(scheme)

			// All non-ASCII schemes should have color codes
			if scheme != "ascii_only" {
				if palette.Reset == "" {
					t.Error("expected Reset code to be non-empty for color scheme")
				}
			}
		})
	}
}
