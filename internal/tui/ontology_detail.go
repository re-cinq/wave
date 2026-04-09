package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// OntologyDetailModel is the right pane model for the Ontology view.
type OntologyDetailModel struct {
	width    int
	height   int
	focused  bool
	viewport viewport.Model
	selected *OntologyInfo
}

// NewOntologyDetailModel creates a new ontology detail model.
func NewOntologyDetailModel() OntologyDetailModel {
	return OntologyDetailModel{
		viewport: viewport.New(0, 0),
	}
}

// Init returns nil.
func (m OntologyDetailModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the model dimensions.
func (m *OntologyDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if m.selected != nil {
		m.viewport.SetContent(renderOntologyDetail(m.selected, w))
	}
}

// SetFocused updates the focused state.
func (m *OntologyDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetContext sets the selected context and updates the viewport.
func (m *OntologyDetailModel) SetContext(info *OntologyInfo) {
	m.selected = info
	m.viewport.SetContent(renderOntologyDetail(info, m.width))
	m.viewport.GotoTop()
}

// Update handles messages.
func (m OntologyDetailModel) Update(msg tea.Msg) (OntologyDetailModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.focused {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the detail pane.
func (m OntologyDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if m.selected == nil {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Select a context to view details")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return m.viewport.View()
}

func renderOntologyDetail(info *OntologyInfo, _ int) string {
	if info == nil {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render(info.Name))
	sb.WriteString("\n")

	if info.Description != "" {
		sb.WriteString("\n")
		sb.WriteString(info.Description)
		sb.WriteString("\n")
	}

	// Invariants
	if len(info.Invariants) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Invariants:"))
		sb.WriteString("\n")
		for _, inv := range info.Invariants {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("*"), inv))
		}
	}

	// Lineage stats
	if info.HasLineage {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Pipeline Lineage:"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s %d\n", labelStyle.Render("Total runs:"), info.TotalRuns))
		sb.WriteString(fmt.Sprintf("  %s %d  %s %d\n",
			labelStyle.Render("Successes:"), info.Successes,
			labelStyle.Render("Failures:"), info.Failures))
		sb.WriteString(fmt.Sprintf("  %s %.0f%%\n", labelStyle.Render("Success rate:"), info.SuccessRate))
		if !info.LastUsed.IsZero() {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Last used:"), info.LastUsed.Format("2006-01-02 15:04")))
		}
	}

	// Skill file
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("Context Skill:"))
	sb.WriteString("\n")
	if info.HasSkill {
		sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Path:"), info.SkillPath))

		// Read and display SKILL.md content
		if data, err := os.ReadFile(info.SkillPath); err == nil {
			sb.WriteString("\n")
			content := string(data)
			// Truncate if very long
			if len(content) > 2000 {
				content = content[:2000] + "\n... (truncated)"
			}
			sb.WriteString(labelStyle.Render(content))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString(fmt.Sprintf("  %s\n", labelStyle.Render("No context skill file. Run 'wave analyze' to generate.")))
	}

	return sb.String()
}
