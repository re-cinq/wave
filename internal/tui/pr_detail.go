package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PRDetailModel is the right pane model for the Pull Requests view.
type PRDetailModel struct {
	width    int
	height   int
	focused  bool
	viewport viewport.Model
	selected *PRData
}

// NewPRDetailModel creates a new PR detail model.
func NewPRDetailModel() PRDetailModel {
	return PRDetailModel{
		viewport: viewport.New(0, 0),
	}
}

// Init returns nil.
func (m PRDetailModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the model dimensions.
func (m *PRDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if m.selected != nil {
		m.updateContent()
	}
}

// SetFocused updates the focused state.
func (m *PRDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetPR sets the selected pull request and updates the viewport.
func (m *PRDetailModel) SetPR(pr *PRData) {
	m.selected = pr
	m.updateContent()
	m.viewport.GotoTop()
}

// Update handles messages.
func (m PRDetailModel) Update(msg tea.Msg) (PRDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		// Scroll viewport
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the detail pane.
func (m PRDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if m.selected == nil {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Select a pull request to view details")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return m.viewport.View()
}

func (m *PRDetailModel) updateContent() {
	content := m.renderPRDetail()
	if m.width > 0 {
		content = lipgloss.NewStyle().Width(m.width).Render(content)
	}
	m.viewport.SetContent(content)
}

func (m PRDetailModel) renderPRDetail() string {
	if m.selected == nil {
		return ""
	}

	pr := m.selected
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))

	var sb strings.Builder

	// Title
	sb.WriteString(titleStyle.Render(fmt.Sprintf("PR #%d: %s", pr.Number, pr.Title)))
	sb.WriteString("\n")

	// Metadata
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("State:"), m.statusLabel(pr)))
	if pr.Author != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Author:"), pr.Author))
	}
	if len(pr.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Labels:"), strings.Join(pr.Labels, ", ")))
	}
	if pr.HeadBranch != "" && pr.BaseBranch != "" {
		sb.WriteString(fmt.Sprintf("%s %s → %s\n", labelStyle.Render("Branches:"), pr.HeadBranch, pr.BaseBranch))
	}
	if pr.Additions > 0 || pr.Deletions > 0 || pr.ChangedFiles > 0 {
		sb.WriteString(fmt.Sprintf("%s +%d/-%d in %d files\n",
			labelStyle.Render("Changes:"), pr.Additions, pr.Deletions, pr.ChangedFiles))
	}
	if !pr.CreatedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Created:"), pr.CreatedAt.Format("2006-01-02")))
	}
	if pr.Comments > 0 {
		sb.WriteString(fmt.Sprintf("%s %d\n", labelStyle.Render("Comments:"), pr.Comments))
	}
	if pr.Commits > 0 {
		sb.WriteString(fmt.Sprintf("%s %d\n", labelStyle.Render("Commits:"), pr.Commits))
	}

	// Body
	if pr.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Description:"))
		sb.WriteString("\n")
		sb.WriteString(pr.Body)
		sb.WriteString("\n")
	}

	return sb.String()
}

// statusLabel returns a human-readable status for the PR.
func (m PRDetailModel) statusLabel(pr *PRData) string {
	if pr.Draft {
		return "Draft"
	}
	if pr.Merged {
		return "Merged"
	}
	if pr.State == "closed" {
		return "Closed"
	}
	return "Open"
}
