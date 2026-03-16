package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SuggestDetailModel is the right pane for the Suggest view.
type SuggestDetailModel struct {
	width         int
	height        int
	focused       bool
	selected      *SuggestProposedPipeline
	multiSelected []SuggestProposedPipeline // Set when multi-select is active
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
		m.multiSelected = msg.MultiSelected
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

	var sb strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle := lipgloss.NewStyle().Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Multi-select execution plan
	if len(m.multiSelected) > 1 {
		sb.WriteString(titleStyle.Render("Execution Plan"))
		sb.WriteString("\n\n")
		sb.WriteString(labelStyle.Render(fmt.Sprintf("Selected: %d pipelines", len(m.multiSelected))))
		sb.WriteString("\n\n")
		for i, p := range m.multiSelected {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, p.Name))
			if p.Reason != "" {
				sb.WriteString(fmt.Sprintf("     %s\n", mutedStyle.Render(p.Reason)))
			}
		}
		sb.WriteString("\n")
		sb.WriteString(mutedStyle.Render("Press Enter to compose sequence"))

		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Padding(0, 1).
			Render(sb.String())
	}

	// Single proposal detail
	p := m.selected
	sb.WriteString(titleStyle.Render(p.Name))
	sb.WriteString("\n\n")

	sb.WriteString(labelStyle.Render("Priority: "))
	sb.WriteString(fmt.Sprintf("%d\n", p.Priority))

	if p.Type != "" && p.Type != "single" {
		sb.WriteString(labelStyle.Render("Type: "))
		sb.WriteString(p.Type)
		sb.WriteString("\n")
	}

	if len(p.Sequence) > 0 {
		sb.WriteString(labelStyle.Render("Sequence: "))
		sb.WriteString(strings.Join(p.Sequence, " → "))
		sb.WriteString("\n")
	}

	sb.WriteString(labelStyle.Render("Reason: "))
	sb.WriteString(p.Reason)
	sb.WriteString("\n")

	if p.Input != "" {
		sb.WriteString(labelStyle.Render("Input: "))
		sb.WriteString(p.Input)
		sb.WriteString("\n")
	}

	// DAG preview for sequence/parallel proposals
	if dag := RenderDAG(*p); dag != "" {
		sb.WriteString("\n")
		sb.WriteString(labelStyle.Render("Execution Flow:"))
		sb.WriteString("\n")
		sb.WriteString(dag)
	}

	sb.WriteString("\n")
	sb.WriteString(mutedStyle.Render("Press Enter to launch  Space to select  m: modify  s: skip"))

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(0, 1).
		Render(sb.String())
}
