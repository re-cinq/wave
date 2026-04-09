package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HealthDetailModel is the right pane model for the Health view.
type HealthDetailModel struct {
	width          int
	height         int
	focused        bool
	viewport       viewport.Model
	selected       *HealthCheck
	guidedComplete bool // true when all health checks resolved in guided mode
	guidedErrors   bool // true when health errors exist in guided mode
}

// NewHealthDetailModel creates a new health detail model.
func NewHealthDetailModel() HealthDetailModel {
	return HealthDetailModel{
		viewport: viewport.New(0, 0),
	}
}

// Init returns nil.
func (m HealthDetailModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the model dimensions.
func (m *HealthDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if m.selected != nil {
		m.viewport.SetContent(renderHealthDetail(m.selected, w))
	}
}

// SetFocused updates the focused state.
func (m *HealthDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetCheck sets the selected health check and updates the viewport.
func (m *HealthDetailModel) SetCheck(check *HealthCheck) {
	m.selected = check
	m.viewport.SetContent(renderHealthDetail(check, m.width))
	m.viewport.GotoTop()
}

// Update handles messages.
func (m HealthDetailModel) Update(msg tea.Msg) (HealthDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case HealthCheckResultMsg:
		if m.selected != nil && msg.Name == m.selected.Name {
			m.selected.Status = msg.Status
			m.selected.Message = msg.Message
			m.selected.Details = msg.Details
			m.viewport.SetContent(renderHealthDetail(m.selected, m.width))
		}
		return m, nil

	case HealthAllCompleteMsg:
		m.guidedComplete = true
		m.guidedErrors = msg.HasErrors
		if m.selected != nil {
			m.viewport.SetContent(renderHealthDetail(m.selected, m.width))
		}
		return m, nil

	case tea.KeyMsg:
		if m.focused {
			// Handle 'y' key to continue despite health errors in guided mode
			if msg.String() == "y" && m.guidedComplete && m.guidedErrors {
				return m, func() tea.Msg { return HealthContinueMsg{} }
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the detail pane.
func (m HealthDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if m.selected == nil {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Select a health check to view details")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return m.viewport.View()
}

func renderHealthDetail(check *HealthCheck, _ int) string {
	if check == nil {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render(check.Name))
	sb.WriteString("\n")

	// Status with icon and color
	sb.WriteString("\n")
	var statusStr string
	switch check.Status {
	case HealthCheckOK:
		statusStr = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("● OK")
	case HealthCheckWarn:
		statusStr = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("▲ Warning")
	case HealthCheckErr:
		statusStr = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("✗ Failed")
	case HealthCheckChecking:
		statusStr = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("… Checking")
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), statusStr))

	if check.Message != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Message:"), check.Message))
	}

	if len(check.Details) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Details:"))
		sb.WriteString("\n")

		// Sort keys for deterministic output
		keys := make([]string, 0, len(check.Details))
		for k := range check.Details {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render(k+":"), check.Details[k]))
		}
	}

	if !check.LastChecked.IsZero() {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Last checked:"), check.LastChecked.Format("2006-01-02 15:04:05")))
	}

	return sb.String()
}
