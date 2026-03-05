package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// HeaderModel is the header bar component showing Wave branding and pipeline status.
type HeaderModel struct {
	width int
}

// NewHeaderModel creates a new header model.
func NewHeaderModel() HeaderModel {
	return HeaderModel{}
}

// SetWidth updates the header width for reflow.
func (m *HeaderModel) SetWidth(w int) {
	m.width = w
}

// View renders the header bar as a 3-line string.
func (m HeaderModel) View() string {
	logoText := "╦ ╦╔═╗╦  ╦╔═╗\n║║║╠═╣╚╗╔╝║╣\n╚╩╝╩ ╩ ╚╝ ╚═╝"

	logoStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6"))

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).
		Bold(true).
		PaddingLeft(2)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	logo := logoStyle.Render(logoText)

	infoBlock := titleStyle.Render("Pipeline Orchestrator") + "\n" +
		"\n" +
		statusStyle.Render("  [no pipeline running]")

	header := lipgloss.JoinHorizontal(lipgloss.Top, logo, infoBlock)

	if m.width > 0 {
		header = lipgloss.NewStyle().Width(m.width).Render(header)
	}

	return header
}
