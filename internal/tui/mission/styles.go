package mission

import "github.com/charmbracelet/lipgloss"

// Colors — bright ANSI, matching ProgressModel's palette
var (
	colorPrimary = lipgloss.Color("14")  // Bright cyan
	colorWhite   = lipgloss.Color("7")   // Light gray / white
	colorMuted   = lipgloss.Color("244") // Medium gray
	colorSuccess = lipgloss.Color("10")  // Bright green
	colorError   = lipgloss.Color("9")   // Bright red
	colorYellow  = lipgloss.Color("3")   // Yellow
	colorBlue    = lipgloss.Color("4")   // Blue
)

// Status glyphs
const (
	glyphRunning   = "●"
	glyphQueued    = "◐"
	glyphCompleted = "✓"
	glyphFailed    = "✗"
	glyphCancelled = "⊘"
	glyphStale     = "?"
)

// Shared styles
var (
	styleStatusRunning = lipgloss.NewStyle().
				Foreground(colorPrimary)

	styleStatusQueued = lipgloss.NewStyle().
				Foreground(colorYellow)

	styleStatusCompleted = lipgloss.NewStyle().
				Foreground(colorSuccess)

	styleStatusFailed = lipgloss.NewStyle().
				Foreground(colorError)

	styleStatusCancelled = lipgloss.NewStyle().
				Foreground(colorMuted)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleCursor = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleOverlayBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(1, 2)

	styleWarning = lipgloss.NewStyle().
			Foreground(colorYellow)
)

// statusGlyph returns the glyph and style for a run status.
func statusGlyph(status string) (string, lipgloss.Style) {
	switch status {
	case "running":
		return glyphRunning, styleStatusRunning
	case "queued", "pending":
		return glyphQueued, styleStatusQueued
	case "completed":
		return glyphCompleted, styleStatusCompleted
	case "failed":
		return glyphFailed, styleStatusFailed
	case "cancelled":
		return glyphCancelled, styleStatusCancelled
	case "stale":
		return glyphStale, styleMuted
	default:
		return "?", styleMuted
	}
}
