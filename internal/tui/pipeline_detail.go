package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PipelineDetailModel is the Bubble Tea model for the right pane.
type PipelineDetailModel struct {
	width    int
	height   int
	focused  bool
	viewport viewport.Model

	selectedName  string
	selectedKind  itemKind
	selectedRunID string

	availableDetail *AvailableDetail
	finishedDetail  *FinishedDetail
	branchDeleted   bool
	loading         bool
	errorMsg        string

	provider DetailDataProvider
}

// NewPipelineDetailModel creates a new pipeline detail model with the given provider.
func NewPipelineDetailModel(provider DetailDataProvider) PipelineDetailModel {
	return PipelineDetailModel{
		viewport: viewport.New(0, 0),
		provider: provider,
	}
}

// SetSize updates the model dimensions and re-renders content if data exists.
func (m *PipelineDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	m.updateViewportContent()
}

// SetFocused updates the focused state.
func (m *PipelineDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// Init implements tea.Model. Returns nil.
func (m PipelineDetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages to update model state.
func (m PipelineDetailModel) Update(msg tea.Msg) (PipelineDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case PipelineSelectedMsg:
		m.selectedName = msg.Name
		m.selectedKind = msg.Kind
		m.selectedRunID = msg.RunID
		m.branchDeleted = msg.BranchDeleted
		m.availableDetail = nil
		m.finishedDetail = nil
		m.errorMsg = ""

		if msg.Kind == itemKindSectionHeader {
			m.selectedName = ""
			m.loading = false
			m.viewport.SetContent("")
			return m, nil
		}

		if msg.Kind == itemKindRunning {
			m.loading = false
			m.updateViewportContent()
			return m, nil
		}

		m.loading = true

		if msg.Kind == itemKindAvailable {
			name := msg.Name
			provider := m.provider
			return m, func() tea.Msg {
				detail, err := provider.FetchAvailableDetail(name)
				return DetailDataMsg{AvailableDetail: detail, Err: err}
			}
		}

		if msg.Kind == itemKindFinished {
			runID := msg.RunID
			provider := m.provider
			return m, func() tea.Msg {
				detail, err := provider.FetchFinishedDetail(runID)
				return DetailDataMsg{FinishedDetail: detail, Err: err}
			}
		}

	case DetailDataMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMsg = msg.Err.Error()
		} else {
			m.availableDetail = msg.AvailableDetail
			m.finishedDetail = msg.FinishedDetail
		}
		m.updateViewportContent()
		m.viewport.GotoTop()
		return m, nil

	case tea.KeyMsg:
		if m.focused {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// updateViewportContent re-renders the appropriate content and sets it on the viewport.
func (m *PipelineDetailModel) updateViewportContent() {
	if m.availableDetail != nil {
		m.viewport.SetContent(renderAvailableDetail(m.availableDetail, m.width))
	} else if m.finishedDetail != nil {
		m.viewport.SetContent(renderFinishedDetail(m.finishedDetail, m.width, m.branchDeleted))
	} else if m.selectedKind == itemKindRunning && m.selectedName != "" {
		m.viewport.SetContent(renderRunningInfo(m.selectedName, m.width))
	}
}

// View renders the detail pane.
func (m PipelineDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if m.selectedName == "" {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Select a pipeline to view details")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	if m.loading {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Loading...")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	if m.errorMsg != "" {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Render(fmt.Sprintf("Failed to load pipeline details: %s", m.errorMsg))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return m.viewport.View()
}

// renderAvailableDetail renders the detail view for an available pipeline.
func renderAvailableDetail(detail *AvailableDetail, width int) string {
	_ = width
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render(detail.Name))
	sb.WriteString("\n")

	if detail.Description != "" {
		sb.WriteString("\n")
		sb.WriteString(detail.Description)
		sb.WriteString("\n")
	}

	if detail.Category != "" {
		sb.WriteString("\n")
		sb.WriteString(labelStyle.Render("Category: "))
		sb.WriteString(detail.Category)
		sb.WriteString("\n")
	}

	if detail.StepCount > 0 || len(detail.Steps) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render(fmt.Sprintf("Steps (%d):", detail.StepCount)))
		sb.WriteString("\n")
		for i, step := range detail.Steps {
			sb.WriteString(fmt.Sprintf("  %d. %s (%s)\n", i+1, step.ID, step.Persona))
		}
	}

	if detail.InputSource != "" || detail.InputExample != "" {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Input:"))
		sb.WriteString("\n")
		if detail.InputSource != "" {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Source:"), detail.InputSource))
		}
		if detail.InputExample != "" {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Example:"), detail.InputExample))
		}
	}

	if len(detail.Artifacts) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Artifacts:"))
		sb.WriteString("\n")
		for _, a := range detail.Artifacts {
			sb.WriteString(fmt.Sprintf("  • %s\n", a))
		}
	}

	if len(detail.Skills) > 0 || len(detail.Tools) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Dependencies:"))
		sb.WriteString("\n")
		if len(detail.Skills) > 0 {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Skills:"), strings.Join(detail.Skills, ", ")))
		}
		if len(detail.Tools) > 0 {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Tools:"), strings.Join(detail.Tools, ", ")))
		}
	}

	return sb.String()
}

// renderFinishedDetail renders the detail view for a finished pipeline run.
func renderFinishedDetail(detail *FinishedDetail, width int, branchDeleted bool) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render(detail.Name))
	sb.WriteString("\n\n")

	// Status
	var statusStr string
	switch detail.Status {
	case "completed":
		statusStr = greenStyle.Render("\u2713 completed")
	case "failed":
		statusStr = redStyle.Render("\u2717 failed")
	case "cancelled":
		statusStr = yellowStyle.Render("\u2717 cancelled")
	default:
		statusStr = detail.Status
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), statusStr))

	// Duration
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Duration:"), formatDuration(detail.Duration)))

	// Branch
	if detail.BranchName != "" {
		branchName := detail.BranchName
		if branchDeleted {
			branchName = mutedStyle.Render(branchName + " (deleted)")
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Branch:"), branchName))
	}

	// Times
	if !detail.StartedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Started:"), detail.StartedAt.Format("2006-01-02 15:04:05")))
	}
	if !detail.CompletedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Completed:"), detail.CompletedAt.Format("2006-01-02 15:04:05")))
	}

	// Error info
	if detail.ErrorMessage != "" {
		sb.WriteString("\n")
		errWidth := width - 4
		if errWidth < 10 {
			errWidth = 10
		}
		sb.WriteString(redStyle.Render("Error: "))
		sb.WriteString(redStyle.Width(errWidth).Render(detail.ErrorMessage))
		sb.WriteString("\n")
	}
	if detail.FailedStep != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Failed step:"), detail.FailedStep))
	}

	// Steps
	if len(detail.Steps) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Steps:"))
		sb.WriteString("\n")
		for _, step := range detail.Steps {
			var iconStr string
			switch step.Status {
			case "completed":
				iconStr = greenStyle.Render("\u2713")
			case "failed":
				iconStr = redStyle.Render("\u2717")
			default:
				iconStr = mutedStyle.Render("\u2014")
			}
			sb.WriteString(fmt.Sprintf("  %s %-20s  %s  (%s)\n",
				iconStr,
				step.ID,
				formatDuration(step.Duration),
				step.Persona,
			))
		}
	}

	// Artifacts
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("Artifacts:"))
	sb.WriteString("\n")
	if len(detail.Artifacts) == 0 {
		sb.WriteString(fmt.Sprintf("  %s\n", mutedStyle.Render("No artifacts produced")))
	} else {
		for _, a := range detail.Artifacts {
			sb.WriteString(fmt.Sprintf("  * %s  %s  (%s)\n", a.Name, a.Path, a.Type))
		}
	}

	// Action hints
	sb.WriteString("\n")
	enterHint := mutedStyle.Render("[Enter] Open chat")
	branchHint := mutedStyle.Render("[b] Checkout branch")
	if branchDeleted {
		branchHint = mutedStyle.Faint(true).Render("[b] Checkout branch")
	}
	diffHint := mutedStyle.Render("[d] View diff")
	escHint := mutedStyle.Render("[Esc] Back")
	sb.WriteString(fmt.Sprintf("%s  %s  %s  %s\n", enterHint, branchHint, diffHint, escHint))

	return sb.String()
}

// renderRunningInfo renders a brief info view for a running pipeline.
func renderRunningInfo(name string, width int) string {
	_ = width
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render(name))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), greenStyle.Render("\u25b6 Running")))
	sb.WriteString("\n")
	sb.WriteString("Real-time progress monitoring planned for #258.\n")

	return sb.String()
}
