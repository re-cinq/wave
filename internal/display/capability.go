package display

import (
	"github.com/recinq/wave/internal/config"
)

// DetectANSISupport determines if ANSI escape sequences can be used.
// Returns true if:
//   - Output is to a TTY
//   - TERM is not "dumb"
//   - NO_COLOR environment variable is not set
func DetectANSISupport() bool {
	ti := NewTerminalInfo()
	return ti.SupportsANSI()
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
	var useColors bool
	switch colorMode {
	case "on":
		useColors = true
	case "off":
		useColors = false
	case "auto":
		// Auto mode: use colors if supported and not disabled
		useColors = DetectANSISupport() && config.FromEnv().NoColor == ""
	default:
		useColors = DetectANSISupport() && config.FromEnv().NoColor == ""
	}

	if !useColors {
		return AsciiOnlyColorScheme
	}

	return DefaultColorScheme
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
