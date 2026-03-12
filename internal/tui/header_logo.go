package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// logoLines are the raw Wave ASCII art lines.
var logoLines = []string{
	"╦ ╦╔═╗╦  ╦╔═╗",
	"║║║╠═╣╚╗╔╝║╣",
	"╚╩╝╩ ╩ ╚╝ ╚═╝",
}

// logoText is the raw Wave ASCII art (3 lines, no margins or styling).
const logoText = "╦ ╦╔═╗╦  ╦╔═╗\n║║║╠═╣╚╗╔╝║╣\n╚╩╝╩ ╩ ╚╝ ╚═╝"

// LogoAnimator manages logo foreground color cycling with a walking glow effect.
type LogoAnimator struct {
	frame  int
	active bool
}

// NewLogoAnimator creates a LogoAnimator.
func NewLogoAnimator() LogoAnimator {
	return LogoAnimator{}
}

// View renders the logo with a walking glow effect when active.
func (l LogoAnimator) View() string {
	if !l.active {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("2")).
			Render(logoText)
	}

	// Walking glow: a bright spot moves across the logo characters
	glowColors := []lipgloss.Color{"240", "244", "250", "15", "250", "244", "240"}
	baseColor := lipgloss.Color("2")
	glowCenter := l.frame % 18 // logo is ~14 chars wide, sweep across with padding

	var result strings.Builder
	for lineIdx, line := range logoLines {
		runes := []rune(line)
		for i, r := range runes {
			dist := glowCenter - i
			if dist < 0 {
				dist = -dist
			}
			if dist < len(glowColors) {
				style := lipgloss.NewStyle().Bold(true).Foreground(glowColors[dist])
				result.WriteString(style.Render(string(r)))
			} else {
				style := lipgloss.NewStyle().Bold(true).Foreground(baseColor)
				result.WriteString(style.Render(string(r)))
			}
		}
		if lineIdx < len(logoLines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// SetActive starts or stops animation. When deactivated, resets frame to 0.
func (l *LogoAnimator) SetActive(active bool) {
	l.active = active
	if !active {
		l.frame = 0
	}
}

// IsActive returns whether the animation is active.
func (l LogoAnimator) IsActive() bool {
	return l.active
}

// Advance moves to the next frame in the walking glow animation.
func (l *LogoAnimator) Advance() {
	l.frame++
}

// Tick returns a tea.Cmd that fires a LogoTickMsg after 200ms.
func (l LogoAnimator) Tick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return LogoTickMsg{}
	})
}
