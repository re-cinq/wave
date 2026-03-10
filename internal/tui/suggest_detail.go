package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SuggestDetailModel is the right pane for the Suggest view.
type SuggestDetailModel struct {
	width    int
	height   int
	focused  bool
	selected *SuggestProposedPipeline
}

// NewSuggestDetailModel creates a new suggest detail model.
func NewSuggestDetailModel() SuggestDetailModel {
	return SuggestDetailModel{}
}

// SetSize updates dimensions.
func (m *SuggestDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state.
func (m *SuggestDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// Update handles messages.
func (m SuggestDetailModel) Update(msg tea.Msg) (SuggestDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case SuggestSelectedMsg:
		p := msg.Pipeline
		m.selected = &p
	}
	return m, nil
}

// View renders the suggest detail.
func (m SuggestDetailModel) View() string {
	if m.selected == nil {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Select a suggestion to view details")
	}

	p := m.selected
	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle := lipgloss.NewStyle().Bold(true)

	sb.WriteString(titleStyle.Render(p.Name))
	sb.WriteString("\n\n")

	sb.WriteString(labelStyle.Render("Priority: "))
	sb.WriteString(fmt.Sprintf("%d\n", p.Priority))

	sb.WriteString(labelStyle.Render("Reason: "))
	sb.WriteString(p.Reason)
	sb.WriteString("\n")

	if p.Input != "" {
		sb.WriteString(labelStyle.Render("Input: "))
		sb.WriteString(p.Input)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Press Enter to launch"))

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(0, 1).
		Render(sb.String())
}
