package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarModel is the bottom status bar component.
type StatusBarModel struct {
	width        int
	contextLabel string
	focusPane    FocusPane
	formActive   bool
}

// NewStatusBarModel creates a new status bar model with default context.
func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{
		contextLabel: "Dashboard",
	}
}

// SetWidth updates the status bar width for reflow.
func (m *StatusBarModel) SetWidth(w int) {
	m.width = w
}

// Update handles messages to update status bar state.
func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case FocusChangedMsg:
		m.focusPane = msg.Pane
	case FormActiveMsg:
		m.formActive = msg.Active
	}
	return m, nil
}

// View renders the status bar as a single line.
func (m StatusBarModel) View() string {
	bg := lipgloss.Color("237")

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).
		Background(bg).
		PaddingLeft(1)

	hintsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(bg).
		PaddingRight(1)

	label := labelStyle.Render(m.contextLabel)

	var hintsText string
	if m.formActive && m.focusPane == FocusPaneRight {
		hintsText = "Tab: next  Shift+Tab: prev  Enter: launch  Esc: cancel"
	} else if m.focusPane == FocusPaneRight {
		hintsText = "↑↓: scroll  Esc: back  q: quit  ctrl+c: exit"
	} else {
		hintsText = "↑↓: navigate  Enter: view  /: filter  q: quit  ctrl+c: exit"
	}
	hints := hintsStyle.Render(hintsText)

	// Calculate spacing between label and hints
	labelWidth := lipgloss.Width(label)
	hintsWidth := lipgloss.Width(hints)
	spacerWidth := m.width - labelWidth - hintsWidth
	if spacerWidth < 0 {
		spacerWidth = 0
	}

	spacer := lipgloss.NewStyle().
		Background(bg).
		Render(strings.Repeat(" ", spacerWidth))

	return label + spacer + hints
}
