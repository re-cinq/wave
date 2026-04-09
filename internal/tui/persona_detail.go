package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PersonaStatsMsg carries fetched persona stats from the provider.
type PersonaStatsMsg struct {
	Name  string
	Stats *PersonaStats
	Err   error
}

// PersonaDetailModel is the right pane model for the Personas view.
type PersonaDetailModel struct {
	width        int
	height       int
	focused      bool
	viewport     viewport.Model
	selected     *PersonaInfo
	stats        *PersonaStats
	statsLoading bool
	provider     PersonaDataProvider
}

// NewPersonaDetailModel creates a new persona detail model.
func NewPersonaDetailModel(provider PersonaDataProvider) PersonaDetailModel {
	return PersonaDetailModel{
		viewport: viewport.New(0, 0),
		provider: provider,
	}
}

// Init returns nil.
func (m PersonaDetailModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the model dimensions.
func (m *PersonaDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if m.selected != nil {
		m.viewport.SetContent(renderPersonaDetail(m.selected, m.stats, w))
	}
}

// SetFocused updates the focused state.
func (m *PersonaDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// Update handles messages.
func (m PersonaDetailModel) Update(msg tea.Msg) (PersonaDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case PersonaSelectedMsg:
		// Find matching persona
		m.stats = nil
		m.statsLoading = true

		// We'll receive the persona data through the list model
		// Find matching persona from current selection
		name := msg.Name
		provider := m.provider
		return m, func() tea.Msg {
			stats, err := provider.FetchPersonaStats(name)
			return PersonaStatsMsg{Name: name, Stats: stats, Err: err}
		}

	case PersonaStatsMsg:
		if m.selected != nil && msg.Name == m.selected.Name {
			m.stats = msg.Stats
			m.statsLoading = false
			m.viewport.SetContent(renderPersonaDetail(m.selected, m.stats, m.width))
		}
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

// SetPersona sets the selected persona and updates the viewport.
func (m *PersonaDetailModel) SetPersona(info *PersonaInfo) {
	m.selected = info
	m.stats = nil
	m.statsLoading = true
	m.viewport.SetContent(renderPersonaDetail(info, nil, m.width))
	m.viewport.GotoTop()
}

// View renders the detail pane.
func (m PersonaDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if m.selected == nil {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Select a persona to view details")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return m.viewport.View()
}

func renderPersonaDetail(info *PersonaInfo, stats *PersonaStats, _ int) string {
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

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Adapter:"), info.Adapter))

	if info.Model != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Model:"), info.Model))
	}

	if len(info.AllowedTools) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Allowed Tools:"))
		sb.WriteString("\n")
		for _, tool := range info.AllowedTools {
			sb.WriteString(fmt.Sprintf("  • %s\n", tool))
		}
	}

	if len(info.DeniedTools) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Denied Tools:"))
		sb.WriteString("\n")
		for _, tool := range info.DeniedTools {
			sb.WriteString(fmt.Sprintf("  • %s\n", tool))
		}
	}

	if len(info.PipelineUsage) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Pipeline Usage:"))
		sb.WriteString("\n")
		for _, ref := range info.PipelineUsage {
			sb.WriteString(fmt.Sprintf("  • %s / %s\n", ref.PipelineName, ref.StepID))
		}
	}

	// Run stats section
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("Run Stats:"))
	sb.WriteString("\n")
	if stats == nil {
		sb.WriteString(fmt.Sprintf("  %s\n", labelStyle.Render("No runs recorded")))
	} else {
		sb.WriteString(fmt.Sprintf("  %s %d\n", labelStyle.Render("Total runs:"), stats.TotalRuns))
		if stats.TotalRuns > 0 {
			successRate := float64(stats.SuccessfulRuns) / float64(stats.TotalRuns) * 100
			sb.WriteString(fmt.Sprintf("  %s %.0f%%\n", labelStyle.Render("Success rate:"), successRate))
		}
		sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Avg duration:"), formatDuration(time.Duration(stats.AvgDurationMs)*time.Millisecond)))
		if !stats.LastRunAt.IsZero() {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Last run:"), stats.LastRunAt.Format("2006-01-02 15:04:05")))
		}
	}

	return sb.String()
}
