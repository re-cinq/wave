package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/suggest"
)

// HealthCheck holds the state and result of a single health check.
type HealthCheck struct {
	Name        string
	Description string
	Status      suggest.Status
	Message     string
	Details     map[string]string
	LastChecked time.Time
}

// HealthSelectedMsg signals that a health check was selected in the list.
type HealthSelectedMsg struct {
	Name  string
	Index int
}

// HealthListModel is the left pane model for the Health view.
type HealthListModel struct {
	width        int
	height       int
	checks       []HealthCheck
	cursor       int
	focused      bool
	scrollOffset int
	provider     HealthDataProvider
}

// NewHealthListModel creates a new health list model.
func NewHealthListModel(provider HealthDataProvider) HealthListModel {
	names := provider.CheckNames()
	checks := make([]HealthCheck, len(names))
	for i, name := range names {
		checks[i] = HealthCheck{
			Name:   name,
			Status: HealthCheckChecking,
		}
	}

	return HealthListModel{
		checks:   checks,
		provider: provider,
		focused:  true,
	}
}

// Init returns batch commands to run all health checks.
func (m HealthListModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.checks))
	for i, check := range m.checks {
		name := check.Name
		provider := m.provider
		cmds[i] = func() tea.Msg {
			return provider.RunCheck(name)
		}
	}
	return tea.Batch(cmds...)
}

// SetSize updates the list dimensions.
func (m *HealthListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state.
func (m *HealthListModel) SetFocused(focused bool) {
	m.focused = focused
}

// Update handles messages.
func (m HealthListModel) Update(msg tea.Msg) (HealthListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case HealthCheckResultMsg:
		for i := range m.checks {
			if m.checks[i].Name == msg.Name {
				m.checks[i].Status = msg.Status
				m.checks[i].Message = msg.Message
				m.checks[i].Details = msg.Details
				m.checks[i].LastChecked = time.Now()
				break
			}
		}
		// Check if all health checks have completed
		if cmd := m.checkAllComplete(); cmd != nil {
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

func (m HealthListModel) handleKeyMsg(msg tea.KeyMsg) (HealthListModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, m.emitSelectionMsg()
	case tea.KeyDown:
		if m.cursor < len(m.checks)-1 {
			m.cursor++
		}
		return m, m.emitSelectionMsg()
	default:
		if msg.String() == "r" {
			// Re-run all checks
			cmds := make([]tea.Cmd, len(m.checks))
			for i := range m.checks {
				m.checks[i].Status = HealthCheckChecking
				name := m.checks[i].Name
				provider := m.provider
				cmds[i] = func() tea.Msg {
					return provider.RunCheck(name)
				}
			}
			return m, tea.Batch(cmds...)
		}
	}

	return m, nil
}

func (m HealthListModel) emitSelectionMsg() tea.Cmd {
	if len(m.checks) == 0 || m.cursor >= len(m.checks) {
		return nil
	}

	check := m.checks[m.cursor]
	return func() tea.Msg {
		return HealthSelectedMsg{Name: check.Name, Index: m.cursor}
	}
}

// View renders the health check list.
func (m HealthListModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	var lines []string
	visibleHeight := m.height

	m.adjustScrollOffset(visibleHeight)

	endOffset := m.scrollOffset + visibleHeight
	if endOffset > len(m.checks) {
		endOffset = len(m.checks)
	}

	for i := m.scrollOffset; i < endOffset; i++ {
		check := m.checks[i]
		isSelected := i == m.cursor

		var icon string
		var iconStyle lipgloss.Style
		switch check.Status {
		case suggest.StatusOK:
			icon = "●"
			iconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		case suggest.StatusWarn:
			icon = "▲"
			iconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
		case suggest.StatusErr:
			icon = "✗"
			iconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		case HealthCheckChecking:
			icon = "…"
			iconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		}

		nameMaxWidth := m.width - 6 // prefix (3) + icon (1) + space (1) + padding
		name := truncateName(check.Name, nameMaxWidth)

		if isSelected {
			// Plain text when selected — inner ANSI codes break the highlight background.
			text := "  " + icon + " " + name
			style := SelectionStyle(m.focused).
				Width(m.width)
			lines = append(lines, style.Render(text))
		} else {
			styledIcon := iconStyle.Render(icon)
			text := "  " + styledIcon + " " + name
			style := lipgloss.NewStyle().
				Width(m.width)
			lines = append(lines, style.Render(text))
		}
	}

	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// checkAllComplete returns a command emitting HealthAllCompleteMsg if all checks have resolved.
func (m HealthListModel) checkAllComplete() tea.Cmd {
	if len(m.checks) == 0 {
		return nil
	}
	hasErrors := false
	for _, check := range m.checks {
		if check.Status == HealthCheckChecking {
			return nil
		}
		if check.Status == suggest.StatusErr {
			hasErrors = true
		}
	}
	return func() tea.Msg {
		return HealthAllCompleteMsg{HasErrors: hasErrors}
	}
}

func (m *HealthListModel) adjustScrollOffset(visibleHeight int) {
	if visibleHeight <= 0 {
		return
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.cursor - visibleHeight + 1
	}
	maxOffset := len(m.checks) - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}
