package display

import (
	"fmt"
	"strings"
	"time"
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

// NewFormatterWithConfig creates a formatter with specific configuration.
func NewFormatterWithConfig(colorMode string, asciiOnly bool) *Formatter {
	return &Formatter{
		codec:   NewANSICodecWithConfig(colorMode, asciiOnly),
		charSet: GetUnicodeCharSet(),
		enabled: DetectANSISupport() && !asciiOnly,
	}
}

// ANSI Control Sequences

// ClearLine clears the current line.
func (f *Formatter) ClearLine() string {
	return f.codec.ClearLine()
}

// ClearScreen clears the entire screen.
func (f *Formatter) ClearScreen() string {
	if !f.enabled {
		return ""
	}
	return "\033[2J"
}

// ClearToEndOfLine clears from cursor to end of line.
func (f *Formatter) ClearToEndOfLine() string {
	if !f.enabled {
		return ""
	}
	return "\033[K"
}

// ClearToStartOfLine clears from cursor to start of line.
func (f *Formatter) ClearToStartOfLine() string {
	if !f.enabled {
		return ""
	}
	return "\033[1K"
}

// MoveCursorUp moves cursor up N lines.
func (f *Formatter) MoveCursorUp(n int) string {
	if !f.enabled || n <= 0 {
		return ""
	}
	return fmt.Sprintf("\033[%dA", n)
}

// MoveCursorDown moves cursor down N lines.
func (f *Formatter) MoveCursorDown(n int) string {
	if !f.enabled || n <= 0 {
		return ""
	}
	return fmt.Sprintf("\033[%dB", n)
}

// MoveCursorForward moves cursor forward N columns.
func (f *Formatter) MoveCursorForward(n int) string {
	if !f.enabled || n <= 0 {
		return ""
	}
	return fmt.Sprintf("\033[%dC", n)
}

// MoveCursorBack moves cursor back N columns.
func (f *Formatter) MoveCursorBack(n int) string {
	if !f.enabled || n <= 0 {
		return ""
	}
	return fmt.Sprintf("\033[%dD", n)
}

// MoveCursorToColumn moves cursor to specific column.
func (f *Formatter) MoveCursorToColumn(col int) string {
	if !f.enabled || col < 0 {
		return ""
	}
	return fmt.Sprintf("\033[%dG", col)
}

// MoveCursorToPosition moves cursor to specific row and column.
func (f *Formatter) MoveCursorToPosition(row, col int) string {
	if !f.enabled || row < 0 || col < 0 {
		return ""
	}
	return fmt.Sprintf("\033[%d;%dH", row, col)
}

// SaveCursorPosition saves current cursor position.
func (f *Formatter) SaveCursorPosition() string {
	return f.codec.SaveCursorPosition()
}

// RestoreCursorPosition restores saved cursor position.
func (f *Formatter) RestoreCursorPosition() string {
	return f.codec.RestoreCursorPosition()
}

// HideCursor hides the cursor.
func (f *Formatter) HideCursor() string {
	return f.codec.HideCursor()
}

// ShowCursor shows the cursor.
func (f *Formatter) ShowCursor() string {
	return f.codec.ShowCursor()
}

// Text Formatting

// Bold formats text as bold.
func (f *Formatter) Bold(text string) string {
	return f.codec.Bold(text)
}

// Dim formats text as dim/faint.
func (f *Formatter) Dim(text string) string {
	return f.codec.Dim(text)
}

// Italic formats text as italic.
func (f *Formatter) Italic(text string) string {
	if !f.enabled {
		return text
	}
	return "\033[3m" + text + "\033[0m"
}

// Underline formats text as underlined.
func (f *Formatter) Underline(text string) string {
	return f.codec.Underline(text)
}

// Strikethrough formats text with strikethrough.
func (f *Formatter) Strikethrough(text string) string {
	if !f.enabled {
		return text
	}
	return "\033[9m" + text + "\033[0m"
}

// Inverse formats text with inverted colors.
func (f *Formatter) Inverse(text string) string {
	if !f.enabled {
		return text
	}
	return "\033[7m" + text + "\033[0m"
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

// ProgressBar renders a progress bar with percentage.
func (f *Formatter) ProgressBar(percent int, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// Calculate filled and empty portions
	filled := (percent * width) / 100
	empty := width - filled

	var bar strings.Builder
	bar.WriteString("[")

	// Draw filled portion
	for i := 0; i < filled; i++ {
		bar.WriteString(f.charSet.Block)
	}

	// Draw empty portion
	for i := 0; i < empty; i++ {
		bar.WriteString(f.charSet.LightBlock)
	}

	bar.WriteString("]")

	// Add percentage
	percentStr := fmt.Sprintf(" %3d%%", percent)

	return f.Primary(bar.String()) + " " + f.Muted(percentStr)
}

// ProgressBarWithLabel renders a progress bar with a label.
func (f *Formatter) ProgressBarWithLabel(label string, percent int, width int) string {
	bar := f.ProgressBar(percent, width)
	return fmt.Sprintf("%s %s", f.Bold(label), bar)
}

// Spinner returns the current spinner frame based on tick count.
func (f *Formatter) Spinner(tick int) string {
	frame := tick % len(f.charSet.Spinner)
	return f.Primary(f.charSet.Spinner[frame])
}

// Duration Formatting

// FormatDuration formats a duration in milliseconds to human-readable form.
func (f *Formatter) FormatDuration(durationMs int64) string {
	if durationMs < 0 {
		return "0s"
	}

	d := time.Duration(durationMs) * time.Millisecond

	if d < time.Second {
		return fmt.Sprintf("%dms", durationMs)
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// FormatETA formats estimated time remaining.
func (f *Formatter) FormatETA(etaMs int64) string {
	if etaMs <= 0 {
		return "calculating..."
	}
	duration := f.FormatDuration(etaMs)
	return fmt.Sprintf("ETA: %s", duration)
}

// FormatTimestamp formats a Unix timestamp to human-readable form.
func (f *Formatter) FormatTimestamp(timestamp int64) string {
	t := time.Unix(timestamp, 0)
	return t.Format("15:04:05")
}

// State Formatting

// FormatState returns a formatted state string with appropriate colors.
func (f *Formatter) FormatState(state ProgressState) string {
	tcc := NewTerminalColorContext()
	return tcc.FormatState(state)
}

// GetStateIcon returns an appropriate icon for the given state.
func (f *Formatter) GetStateIcon(state ProgressState) string {
	tcc := NewTerminalColorContext()
	return tcc.GetStateIcon(state)
}

// Size Formatting

// FormatBytes formats a byte count to human-readable form.
func (f *Formatter) FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// Token Formatting

// FormatTokens formats a token count to human-readable form.
func (f *Formatter) FormatTokens(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d tokens", tokens)
	}
	return fmt.Sprintf("%.1fk tokens", float64(tokens)/1000.0)
}

// Table Formatting

// TableRow formats a row with columns of specific widths.
func (f *Formatter) TableRow(columns []string, widths []int) string {
	if len(columns) != len(widths) {
		return strings.Join(columns, " ")
	}

	var parts []string
	for i, col := range columns {
		if widths[i] > 0 {
			// Truncate or pad to width
			if len(col) > widths[i] {
				col = col[:widths[i]-3] + "..."
			} else {
				col = col + strings.Repeat(" ", widths[i]-len(col))
			}
		}
		parts = append(parts, col)
	}

	return strings.Join(parts, " ")
}

// Truncate truncates text to specified width with ellipsis.
func (f *Formatter) Truncate(text string, width int) string {
	if len(text) <= width {
		return text
	}
	if width <= 3 {
		return text[:width]
	}
	return text[:width-3] + f.charSet.Ellipsis
}

// Wrap wraps text to fit within specified width.
func (f *Formatter) Wrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		// If adding this word would exceed width, start new line
		if currentLine.Len() > 0 && currentLine.Len()+1+len(word) > width {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		}

		// Add space before word if not first word in line
		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)
	}

	// Add last line
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// Box Formatting

// Box draws a box around text with optional title.
func (f *Formatter) Box(content string, title string, width int) string {
	var result strings.Builder

	// Top border
	if title != "" {
		titleLen := len(title) + 2 // Add spaces around title
		if width < titleLen+4 {
			width = titleLen + 4
		}
		leftPad := (width - titleLen) / 2
		rightPad := width - titleLen - leftPad
		result.WriteString("┌" + strings.Repeat("─", leftPad-1))
		result.WriteString(" " + f.Bold(title) + " ")
		result.WriteString(strings.Repeat("─", rightPad-1) + "┐\n")
	} else {
		result.WriteString("┌" + strings.Repeat("─", width-2) + "┐\n")
	}

	// Content lines
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Pad or truncate line to fit width
		padding := width - len(line) - 4 // Account for borders and spaces
		if padding < 0 {
			line = line[:width-6] + "..."
			padding = 0
		}
		result.WriteString("│ " + line + strings.Repeat(" ", padding) + " │\n")
	}

	// Bottom border
	result.WriteString("└" + strings.Repeat("─", width-2) + "┘")

	return result.String()
}

// List Formatting

// BulletList formats items as a bulleted list.
func (f *Formatter) BulletList(items []string) string {
	var result strings.Builder
	for _, item := range items {
		result.WriteString(f.Muted(f.charSet.Bullet))
		result.WriteString(" ")
		result.WriteString(item)
		result.WriteString("\n")
	}
	return result.String()
}

// NumberedList formats items as a numbered list.
func (f *Formatter) NumberedList(items []string) string {
	var result strings.Builder
	for i, item := range items {
		result.WriteString(f.Muted(fmt.Sprintf("%d.", i+1)))
		result.WriteString(" ")
		result.WriteString(item)
		result.WriteString("\n")
	}
	return result.String()
}

// Horizontal Rule

// HorizontalRule draws a horizontal line of specified width.
func (f *Formatter) HorizontalRule(width int) string {
	if width <= 0 {
		width = 80
	}
	return f.Muted(strings.Repeat("─", width))
}

// Reset resets all formatting.
func (f *Formatter) Reset() string {
	if !f.enabled {
		return ""
	}
	return "\033[0m"
}

// =============================================================================
// Package-level Helper Functions
// =============================================================================

// FormatTokenCount formats a token count to human-readable form.
func FormatTokenCount(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	return fmt.Sprintf("%.1fk", float64(tokens)/1000.0)
}

// FormatFileCount formats a file count to human-readable form.
func FormatFileCount(files int) string {
	if files == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", files)
}

// FormatDuration formats a duration in milliseconds to human-readable form.
func FormatDuration(durationMs int64) string {
	if durationMs < 0 {
		return "0s"
	}

	d := time.Duration(durationMs) * time.Millisecond

	if d < time.Second {
		return fmt.Sprintf("%dms", durationMs)
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}
