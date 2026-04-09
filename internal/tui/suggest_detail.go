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
	if msg, ok := msg.(SuggestSelectedMsg); ok {
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
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))

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

	// -- Header --
	sb.WriteString(titleStyle.Render(p.Name))
	sb.WriteString("\n")

	if p.Description != "" {
		sb.WriteString(mutedStyle.Render(p.Description))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// -- Metadata --
	sb.WriteString(labelStyle.Render("Priority: "))
	sb.WriteString(fmt.Sprintf("%d\n", p.Priority))

	if p.Type != "" && p.Type != "single" {
		sb.WriteString(labelStyle.Render("Type: "))
		sb.WriteString(p.Type)
		sb.WriteString("\n")
	}

	if p.Complexity != "" {
		sb.WriteString(labelStyle.Render("Complexity: "))
		sb.WriteString(renderComplexity(p.Complexity))
		sb.WriteString("\n")
	}

	if len(p.Sequence) > 0 {
		sb.WriteString(labelStyle.Render("Sequence: "))
		sb.WriteString(strings.Join(p.Sequence, " -> "))
		sb.WriteString("\n")
	}

	if p.Input != "" {
		sb.WriteString(labelStyle.Render("Input: "))
		sb.WriteString(p.Input)
		sb.WriteString("\n")
	}

	// -- Why Suggested --
	if p.Reason != "" {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Why Suggested"))
		sb.WriteString("\n")
		sb.WriteString(p.Reason)
		sb.WriteString("\n")
	}

	// -- Pipeline Steps --
	if len(p.Steps) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Pipeline Steps"))
		sb.WriteString("\n")
		for i, step := range p.Steps {
			var connector string
			if i < len(p.Steps)-1 {
				connector = "├─"
			} else {
				connector = "└─"
			}
			sb.WriteString(fmt.Sprintf("  %s %s\n", connector, step))
		}
	}

	// -- DAG preview for sequence/parallel proposals --
	if dag := RenderDAG(*p); dag != "" {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Execution Flow"))
		sb.WriteString("\n")
		sb.WriteString(dag)
	}

	sb.WriteString("\n")
	sb.WriteString(mutedStyle.Render(m.footerHints()))

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(0, 1).
		Render(sb.String())
}

// footerHints returns key hints for the detail pane.
func (m SuggestDetailModel) footerHints() string {
	return "Press Enter to launch  Space to select  m: modify  s: skip"
}

// renderComplexity returns a styled complexity indicator.
func renderComplexity(complexity string) string {
	switch strings.ToLower(complexity) {
	case "low":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("Low")
	case "medium":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Medium")
	case "high":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("High")
	default:
		return complexity
	}
}
