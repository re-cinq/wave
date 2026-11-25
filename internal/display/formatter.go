package display

import (
	"fmt"
)

// Formatter provides ANSI escape sequence management and text formatting utilities.
type Formatter struct {
	codec   *ANSICodec
	charSet UnicodeCharSet
	enabled bool
}

// NewFormatter creates a new formatter with detected capabilities.
func NewFormatter() *Formatter {
	return &Formatter{
		codec:   NewANSICodec(),
		charSet: GetUnicodeCharSet(),
		enabled: DetectANSISupport(),
	}
}

// ANSI Control Sequences

// Text Formatting

// Bold formats text as bold.
func (f *Formatter) Bold(text string) string {
	return f.codec.Bold(text)
}

// Color Formatting

// Primary wraps text in primary color (cyan).
func (f *Formatter) Primary(text string) string {
	return f.codec.Primary(text)
}

// Success wraps text in success color (green).
func (f *Formatter) Success(text string) string {
	return f.codec.Success(text)
}

// Error wraps text in error color (red).
func (f *Formatter) Error(text string) string {
	return f.codec.Error(text)
}

// Warning wraps text in warning color (yellow).
func (f *Formatter) Warning(text string) string {
	return f.codec.Warning(text)
}

// Muted wraps text in muted color (gray).
func (f *Formatter) Muted(text string) string {
	return f.codec.Muted(text)
}

// Colorize wraps text with custom ANSI color code.
func (f *Formatter) Colorize(text string, colorCode string) string {
	return f.codec.Colorize(text, colorCode)
}

// Progress Bar Formatting

// Table Formatting

// Box Formatting

// List Formatting

// Horizontal Rule

// =============================================================================
// Package-level Helper Functions
// =============================================================================

// FormatTokenCount formats a token count to human-readable form.
func FormatTokenCount(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	if tokens < 1_000_000 {
		return fmt.Sprintf("%.1fk", float64(tokens)/1_000.0)
	}
	if tokens < 1_000_000_000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000.0)
	}
	return fmt.Sprintf("%.1fB", float64(tokens)/1_000_000_000.0)
}
