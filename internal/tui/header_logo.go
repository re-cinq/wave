package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// logoText is the raw Wave ASCII art (3 lines, no margins or styling).
const logoText = "╦ ╦╔═╗╦  ╦╔═╗\n║║║╠═╣╚╗╔╝║╣\n╚╩╝╩ ╩ ╚╝ ╚═╝"

// LogoAnimator manages logo foreground color cycling.
type LogoAnimator struct {
	palette    []lipgloss.Color
	colorIndex int
	active     bool
}

// NewLogoAnimator creates a LogoAnimator with the default color palette.
func NewLogoAnimator() LogoAnimator {
	return LogoAnimator{
		palette: []lipgloss.Color{"6", "4", "5"}, // cyan, blue, magenta
	}
}

// View renders the logo with the current palette color.
func (l LogoAnimator) View() string {
	color := l.palette[l.colorIndex]
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Render(logoText)
}

// SetActive starts or stops animation. When deactivated, resets to cyan (index 0).
func (l *LogoAnimator) SetActive(active bool) {
	l.active = active
	if !active {
		l.colorIndex = 0
	}
}

// IsActive returns whether the animation is active.
func (l LogoAnimator) IsActive() bool {
	return l.active
}

// Advance moves to the next color in the palette.
func (l *LogoAnimator) Advance() {
	l.colorIndex = (l.colorIndex + 1) % len(l.palette)
}

// Tick returns a tea.Cmd that fires a LogoTickMsg after 200ms.
func (l LogoAnimator) Tick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return LogoTickMsg{}
	})
}
