package display

import (
	"os"
)

// CapabilityDetector detects and manages terminal capabilities with caching.
type CapabilityDetector struct {
	cached *TerminalCapabilities
}

// NewCapabilityDetector creates a new capability detector.
func NewCapabilityDetector() *CapabilityDetector {
	return &CapabilityDetector{
		cached: nil,
	}
}

// DetectANSISupport determines if ANSI escape sequences can be used.
// Returns true if:
//   - Output is to a TTY
//   - TERM is not "dumb"
//   - NO_COLOR environment variable is not set
func DetectANSISupport() bool {
	ti := NewTerminalInfo()
	return ti.SupportsANSI()
}

// DetectColorSupport determines if 24-bit RGB color can be used.
// Returns true if:
//   - ANSI is supported
//   - COLORTERM is "truecolor" or "24bit"
//   - TERM contains "256color" or "truecolor"
//   - Running in a supported CI environment
func DetectColorSupport() bool {
	ti := NewTerminalInfo()
	return ti.SupportsColor()
}

// Detect256ColorSupport determines if 256-color palette can be used.
// Returns true if:
//   - ANSI is supported
//   - TERM indicates 256-color support
//   - 24-bit color is supported (superset)
func Detect256ColorSupport() bool {
	ti := NewTerminalInfo()
	return ti.Supports256Colors()
}

// DetectUnicodeSupport determines if Unicode characters can be displayed.
// Returns true if:
//   - LANG or LC_ALL contains "UTF-8"
//   - Running on a TTY (most modern terminals support Unicode)
//   - NO_UNICODE environment variable is not set
func DetectUnicodeSupport() bool {
	ti := NewTerminalInfo()
	return ti.SupportsUnicode()
}

// SelectColorPalette returns appropriate colors based on terminal capabilities.
// If colors are disabled (via NO_COLOR or ASCII-only mode), returns empty strings.
func SelectColorPalette(colorMode string, asciiOnly bool) ColorPalette {
	// ASCII-only mode disables all colors
	if asciiOnly {
		return AsciiOnlyColorScheme
	}

	// Determine if colors should be used
	useColors := false
	switch colorMode {
	case "on":
		useColors = true
	case "off":
		useColors = false
	case "auto":
		// Auto mode: use colors if supported and not disabled
		useColors = DetectANSISupport() && os.Getenv("NO_COLOR") == ""
	default:
		useColors = DetectANSISupport() && os.Getenv("NO_COLOR") == ""
	}

	if !useColors {
		return AsciiOnlyColorScheme
	}

	return DefaultColorScheme
}

// SelectAnimationType returns an appropriate animation based on capabilities.
func SelectAnimationType(preferred AnimationType) AnimationType {
	// If Unicode is not supported, use dots animation (ASCII-safe)
	if !DetectUnicodeSupport() {
		return AnimationDots
	}

	// If ANSI is not supported, use simple ASCII animation
	if !DetectANSISupport() {
		return AnimationDots
	}

	// Use preferred type if specified and not disabled
	if preferred != "" && preferred != AnimationDots {
		return preferred
	}

	// Default animation
	return AnimationSpinner
}

// GetOptimalDisplayConfig returns a display configuration optimized for the terminal.
func GetOptimalDisplayConfig() DisplayConfig {
	ti := NewTerminalInfo()

	// Determine color mode
	colorMode := "auto"
	if os.Getenv("NO_COLOR") != "" {
		colorMode = "off"
	}

	// ASCII-only mode for non-Unicode terminals
	asciiOnly := !ti.SupportsUnicode()

	// Select animation based on capabilities
	animation := AnimationSpinner
	if asciiOnly {
		animation = AnimationDots
	}

	// Refresh rate: smooth animations for modern CLI feel
	refreshRate := 30 // 30 FPS for smooth animations like btop+/opencode
	if !ti.IsTTY() {
		refreshRate = 1 // Still slow for non-interactive
	}

	return DisplayConfig{
		Enabled:          ti.IsTTY() && ti.SupportsANSI(),
		AnimationType:    animation,
		RefreshRate:      refreshRate,
		ShowDetails:      true,
		ShowArtifacts:    true,
		CompactMode:      false,
		ColorMode:        colorMode,
		AsciiOnly:        asciiOnly,
		MaxHistoryLines:  100,
		EnableTimestamps: true,
		VerboseOutput:    false,
	}
}

// ANSICodec provides functions to generate ANSI control codes.
type ANSICodec struct {
	colors ColorPalette
	ansi   bool
}

// NewANSICodec creates a new ANSI codec with detected capabilities.
func NewANSICodec() *ANSICodec {
	return NewANSICodecWithConfig("auto", false)
}

// NewANSICodecWithConfig creates a new ANSI codec with specified configuration.
func NewANSICodecWithConfig(colorMode string, asciiOnly bool) *ANSICodec {
	return &ANSICodec{
		colors: SelectColorPalette(colorMode, asciiOnly),
		ansi:   DetectANSISupport() && !asciiOnly,
	}
}

// Colorize wraps text with color codes.
func (ac *ANSICodec) Colorize(text string, colorCode string) string {
	if !ac.ansi || colorCode == "" {
		return text
	}
	return colorCode + text + ac.colors.Reset
}

// Success wraps text in success color (green).
func (ac *ANSICodec) Success(text string) string {
	return ac.Colorize(text, ac.colors.Success)
}

// Error wraps text in error color (red).
func (ac *ANSICodec) Error(text string) string {
	return ac.Colorize(text, ac.colors.Error)
}

// Warning wraps text in warning color (yellow).
func (ac *ANSICodec) Warning(text string) string {
	return ac.Colorize(text, ac.colors.Warning)
}

// Muted wraps text in muted color (gray).
func (ac *ANSICodec) Muted(text string) string {
	return ac.Colorize(text, ac.colors.Muted)
}

// Primary wraps text in primary color (cyan).
func (ac *ANSICodec) Primary(text string) string {
	return ac.Colorize(text, ac.colors.Primary)
}

// Bold returns the ANSI code for bold text.
func (ac *ANSICodec) Bold(text string) string {
	if !ac.ansi {
		return text
	}
	return "\033[1m" + text + ac.colors.Reset
}

// Dim returns the ANSI code for dim text.
func (ac *ANSICodec) Dim(text string) string {
	if !ac.ansi {
		return text
	}
	return "\033[2m" + text + ac.colors.Reset
}

// Underline returns the ANSI code for underlined text.
func (ac *ANSICodec) Underline(text string) string {
	if !ac.ansi {
		return text
	}
	return "\033[4m" + text + ac.colors.Reset
}

// ClearLine returns ANSI code to clear the current line.
func (ac *ANSICodec) ClearLine() string {
	if !ac.ansi {
		return ""
	}
	return "\033[2K"
}

// CursorUp returns ANSI code to move cursor up N lines.
func (ac *ANSICodec) CursorUp(lines int) string {
	if !ac.ansi || lines <= 0 {
		return ""
	}
	if lines == 1 {
		return "\033[A"
	}
	return "\033[" + string(rune(lines)) + "A"
}

// CursorHome returns ANSI code to move cursor to beginning of line.
func (ac *ANSICodec) CursorHome() string {
	if !ac.ansi {
		return ""
	}
	return "\033[H"
}

// HideCursor returns ANSI code to hide the cursor.
func (ac *ANSICodec) HideCursor() string {
	if !ac.ansi {
		return ""
	}
	return "\033[?25l"
}

// ShowCursor returns ANSI code to show the cursor.
func (ac *ANSICodec) ShowCursor() string {
	if !ac.ansi {
		return ""
	}
	return "\033[?25h"
}

// SaveCursorPosition returns ANSI code to save cursor position.
func (ac *ANSICodec) SaveCursorPosition() string {
	if !ac.ansi {
		return ""
	}
	return "\033[s"
}

// RestoreCursorPosition returns ANSI code to restore cursor position.
func (ac *ANSICodec) RestoreCursorPosition() string {
	if !ac.ansi {
		return ""
	}
	return "\033[u"
}

// Reset returns the ANSI reset code to clear all formatting.
func (ac *ANSICodec) Reset() string {
	return ac.colors.Reset
}

// GetUnicodeCharSet returns appropriate characters based on Unicode support.
func GetUnicodeCharSet() UnicodeCharSet {
	if DetectUnicodeSupport() {
		return UnicodeCharSetFull
	}
	return UnicodeCharSetASCII
}

// UnicodeCharSet defines characters for various display elements.
type UnicodeCharSet struct {
	CheckMark   string
	CrossMark   string
	Bullet      string
	RightArrow  string
	LeftArrow   string
	Ellipsis    string
	Bar         string
	Block       string
	LightBlock  string
	Spinner     [4]string
	ProgressBar [5]string
}

// UnicodeCharSetFull uses Unicode characters for better appearance.
var UnicodeCharSetFull = UnicodeCharSet{
	CheckMark:   "✓",
	CrossMark:   "✗",
	Bullet:      "•",
	RightArrow:  "→",
	LeftArrow:   "←",
	Ellipsis:    "…",
	Bar:         "▄",
	Block:       "█",
	LightBlock:  "░",
	Spinner:     [4]string{"⠋", "⠙", "⠹", "⠸"},
	ProgressBar: [5]string{"░", "▏", "▎", "▍", "█"},
}

// UnicodeCharSetASCII uses ASCII characters for fallback.
var UnicodeCharSetASCII = UnicodeCharSet{
	CheckMark:   "[OK]",
	CrossMark:   "[X]",
	Bullet:      "*",
	RightArrow:  "->",
	LeftArrow:   "<-",
	Ellipsis:    "...",
	Bar:         "-",
	Block:       "#",
	LightBlock:  "-",
	Spinner:     [4]string{"|", "/", "-", "\\"},
	ProgressBar: [5]string{".", ".", ".", ".", "#"},
}

// TerminalColorContext holds color context for consistent styling.
type TerminalColorContext struct {
	codec   *ANSICodec
	charSet UnicodeCharSet
}

// NewTerminalColorContext creates a new context with detected capabilities.
func NewTerminalColorContext() *TerminalColorContext {
	return &TerminalColorContext{
		codec:   NewANSICodec(),
		charSet: GetUnicodeCharSet(),
	}
}

// FormatState returns a formatted state string with appropriate colors.
func (tcc *TerminalColorContext) FormatState(state ProgressState) string {
	switch state {
	case StateCompleted:
		return tcc.codec.Success(string(state))
	case StateFailed:
		return tcc.codec.Error(string(state))
	case StateRunning:
		return tcc.codec.Primary(string(state))
	case StateSkipped:
		return tcc.codec.Muted(string(state))
	case StateCancelled:
		return tcc.codec.Warning(string(state))
	default:
		return tcc.codec.Muted(string(state))
	}
}

// GetStateIcon returns an appropriate icon for the given state.
func (tcc *TerminalColorContext) GetStateIcon(state ProgressState) string {
	switch state {
	case StateCompleted:
		return tcc.codec.Success(tcc.charSet.CheckMark)
	case StateFailed:
		return tcc.codec.Error(tcc.charSet.CrossMark)
	case StateRunning:
		return tcc.codec.Primary("⟳") // Fallback for spinner
	case StateSkipped:
		return tcc.codec.Muted("⊘")
	case StateCancelled:
		return tcc.codec.Warning("⊛")
	default:
		return tcc.codec.Muted("○")
	}
}
