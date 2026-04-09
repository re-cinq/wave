package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// IssueDetailModel is the right pane model for the Issues view.
type IssueDetailModel struct {
	width         int
	height        int
	focused       bool
	viewport      viewport.Model
	selected      *IssueData
	pipelines     []PipelineInfo
	chooserActive bool
	chooserCursor int
}

// NewIssueDetailModel creates a new issue detail model.
func NewIssueDetailModel() IssueDetailModel {
	return IssueDetailModel{
		viewport: viewport.New(0, 0),
	}
}

// Init returns nil.
func (m IssueDetailModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the model dimensions.
func (m *IssueDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if m.selected != nil {
		m.updateContent()
	}
}

// SetFocused updates the focused state.
func (m *IssueDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetIssue sets the selected issue, ranks pipelines by relevance, and updates the viewport.
func (m *IssueDetailModel) SetIssue(issue *IssueData) {
	m.selected = issue
	m.chooserActive = false
	m.chooserCursor = 0
	m.sortPipelinesByRelevance()
	m.updateContent()
	m.viewport.GotoTop()
}

// SetPipelines sets the available pipelines for the chooser.
func (m *IssueDetailModel) SetPipelines(pipelines []PipelineInfo) {
	m.pipelines = pipelines
}

// Update handles messages.
func (m IssueDetailModel) Update(msg tea.Msg) (IssueDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		if m.chooserActive {
			return m.handleChooserKey(msg)
		}

		// Enter activates the pipeline chooser when an issue is selected
		if msg.Type == tea.KeyEnter && m.selected != nil && len(m.pipelines) > 0 {
			m.chooserActive = true
			m.chooserCursor = 0
			m.updateContent()
			return m, nil
		}

		// Default: scroll viewport
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m IssueDetailModel) handleChooserKey(msg tea.KeyMsg) (IssueDetailModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.chooserCursor > 0 {
			m.chooserCursor--
			m.updateContent()
		}
		return m, nil

	case tea.KeyDown:
		if m.chooserCursor < len(m.pipelines)-1 {
			m.chooserCursor++
			m.updateContent()
		}
		return m, nil

	case tea.KeyEnter:
		if m.chooserCursor < len(m.pipelines) && m.selected != nil {
			pipeline := m.pipelines[m.chooserCursor]
			issueURL := m.selected.HTMLURL
			m.chooserActive = false
			m.updateContent()
			return m, func() tea.Msg {
				return IssueLaunchMsg{
					PipelineName: pipeline.Name,
					IssueURL:     issueURL,
				}
			}
		}
		return m, nil

	case tea.KeyEscape:
		m.chooserActive = false
		m.updateContent()
		return m, nil
	}

	return m, nil
}

// View renders the detail pane.
func (m IssueDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if m.selected == nil {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Select an issue to view details")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return m.viewport.View()
}

func (m *IssueDetailModel) updateContent() {
	content := m.renderIssueDetail()
	// Wrap content to viewport width to prevent horizontal overflow
	// that corrupts the split-pane layout when joined horizontally.
	if m.width > 0 {
		content = lipgloss.NewStyle().Width(m.width).Render(content)
	}
	m.viewport.SetContent(content)
}

func (m IssueDetailModel) renderIssueDetail() string {
	if m.selected == nil {
		return ""
	}

	issue := m.selected
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))

	var sb strings.Builder

	// Title
	sb.WriteString(titleStyle.Render(fmt.Sprintf("Issue #%d: %s", issue.Number, issue.Title)))
	sb.WriteString("\n")

	// Metadata
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("State:"), issue.State))
	if issue.Author != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Author:"), issue.Author))
	}
	if len(issue.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Labels:"), strings.Join(issue.Labels, ", ")))
	}
	if len(issue.Assignees) > 0 {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Assignees:"), strings.Join(issue.Assignees, ", ")))
	}
	if !issue.CreatedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Created:"), issue.CreatedAt.Format("2006-01-02")))
	}
	if issue.Comments > 0 {
		sb.WriteString(fmt.Sprintf("%s %d\n", labelStyle.Render("Comments:"), issue.Comments))
	}

	// Body
	if issue.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Body:"))
		sb.WriteString("\n")
		sb.WriteString(issue.Body)
		sb.WriteString("\n")
	}

	// Pipeline chooser
	if len(m.pipelines) > 0 {
		sb.WriteString("\n")
		dividerWidth := m.width - 2
		if dividerWidth < 10 {
			dividerWidth = 10
		}
		divider := strings.Repeat("\u2500", dividerWidth)
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(divider))
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Launch Pipeline"))
		sb.WriteString("\n\n")

		for i, p := range m.pipelines {
			if m.chooserActive && i == m.chooserCursor {
				line := lipgloss.NewStyle().
					Foreground(lipgloss.Color("6")).
					Bold(true).
					Render(fmt.Sprintf("  > %s", p.Name))
				sb.WriteString(line)
			} else {
				sb.WriteString(fmt.Sprintf("    %s", p.Name))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
		if m.chooserActive {
			sb.WriteString(labelStyle.Render("[Enter] Launch with issue URL  [Esc] Cancel"))
		} else {
			sb.WriteString(labelStyle.Render("[Enter] Open pipeline chooser"))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// sortPipelinesByRelevance reorders pipelines by keyword relevance to the selected issue.
// Pipelines whose name, description, or category match issue title/labels score higher.
func (m *IssueDetailModel) sortPipelinesByRelevance() {
	if m.selected == nil || len(m.pipelines) == 0 {
		return
	}
	scores := make(map[int]int, len(m.pipelines))
	for i, p := range m.pipelines {
		scores[i] = pipelineRelevanceScore(p, m.selected)
	}
	sort.SliceStable(m.pipelines, func(i, j int) bool {
		return scores[i] > scores[j]
	})
}

// pipelineRelevanceScore computes a simple keyword-overlap score between a pipeline and an issue.
func pipelineRelevanceScore(p PipelineInfo, issue *IssueData) int {
	score := 0
	titleLower := strings.ToLower(issue.Title)
	nameLower := strings.ToLower(p.Name)
	descLower := strings.ToLower(p.Description)
	catLower := strings.ToLower(p.Category)

	// Pipeline name tokens match issue title
	for _, tok := range strings.FieldsFunc(nameLower, func(r rune) bool { return r == '-' || r == '_' || r == ' ' }) {
		if len(tok) >= 3 && strings.Contains(titleLower, tok) {
			score += 3
		}
	}

	// Pipeline description words match issue title
	for _, tok := range strings.Fields(descLower) {
		if len(tok) >= 4 && strings.Contains(titleLower, tok) {
			score += 1
		}
	}

	// Label matches against pipeline name, description, or category
	for _, label := range issue.Labels {
		labelLower := strings.ToLower(label)
		if strings.Contains(nameLower, labelLower) {
			score += 5
		}
		if strings.Contains(descLower, labelLower) {
			score += 2
		}
		if catLower != "" && strings.Contains(catLower, labelLower) {
			score += 4
		}
	}

	return score
}
