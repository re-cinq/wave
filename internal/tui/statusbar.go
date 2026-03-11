package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarModel is the bottom status bar component.
type StatusBarModel struct {
	width                int
	contextLabel         string
	focusPane            FocusPane
	formActive           bool
	composeActive        bool
	liveOutputActive     bool
	finishedDetailActive bool
	runningInfoActive    bool
	currentView          ViewType
}

// NewStatusBarModel creates a new status bar model with default context.
func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{
		contextLabel: "Pipelines",
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
	case ComposeActiveMsg:
		m.composeActive = msg.Active
	case FinishedDetailActiveMsg:
		m.finishedDetailActive = msg.Active
	case LiveOutputActiveMsg:
		m.liveOutputActive = msg.Active
	case RunningInfoActiveMsg:
		m.runningInfoActive = msg.Active
	case ViewChangedMsg:
		m.currentView = msg.View
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

	viewLabel := m.contextLabel
	if m.currentView > 0 {
		viewLabel = m.currentView.String()
	}
	label := labelStyle.Render(viewLabel)

	var hintsText string
	if m.formActive && m.focusPane == FocusPaneRight {
		hintsText = "Tab: next  Shift+Tab: prev  Enter: launch  Esc: cancel"
	} else if m.composeActive {
		hintsText = "a: add  x: remove  Shift+↑↓: reorder  Enter: start  Esc: cancel"
	} else if m.liveOutputActive && m.focusPane == FocusPaneRight {
		hintsText = "v: verbose  d: debug  o: output-only  l: log  c: cancel  ↑↓: scroll  Esc: back"
	} else if m.runningInfoActive && m.focusPane == FocusPaneRight {
		hintsText = "c: dismiss  l: logs  ↑↓: scroll  Esc: back"
	} else if m.finishedDetailActive && m.focusPane == FocusPaneRight {
		hintsText = "[Enter] Chat  [b] Branch  [d] Diff  [l] Logs  [Esc] Back"
	} else if m.focusPane == FocusPaneRight {
		hintsText = "↑↓: scroll  Esc: back  q: quit  ctrl+c: exit"
	} else if m.currentView == ViewHealth {
		hintsText = "↑↓: navigate  r: recheck  Enter: view  Tab/Shift+Tab: views  q: quit"
	} else if m.currentView == ViewIssues && m.focusPane == FocusPaneLeft {
		hintsText = "↑↓: navigate  Enter: view  /: filter  Tab/Shift+Tab: views  q: quit"
	} else if m.currentView == ViewIssues && m.focusPane == FocusPaneRight {
		hintsText = "Enter: launch pipeline  ↑↓: scroll  Esc: back"
	} else {
		hintsText = "↑↓: navigate  Enter: view  /: filter  s: compose  c: cancel  Tab/Shift+Tab: views  q: quit"
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
