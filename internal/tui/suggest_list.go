package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SuggestListModel is the left pane model for the Suggest view.
type SuggestListModel struct {
	width      int
	height     int
	proposals  []SuggestProposedPipeline
	cursor     int
	focused    bool
	filtering  bool
	loaded     bool
	provider   SuggestDataProvider
	errMsg     string
}

// NewSuggestListModel creates a new suggest list model.
func NewSuggestListModel(provider SuggestDataProvider) SuggestListModel {
	return SuggestListModel{
		provider: provider,
		focused:  true,
	}
}

// Init returns the command to fetch suggestions.
func (m SuggestListModel) Init() tea.Cmd {
	return m.fetchSuggestions
}

// SetSize updates the list dimensions.
func (m *SuggestListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state.
func (m *SuggestListModel) SetFocused(focused bool) {
	m.focused = focused
}

// Update handles messages.
func (m SuggestListModel) Update(msg tea.Msg) (SuggestListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case SuggestDataMsg:
		if msg.Err != nil {
			m.errMsg = msg.Err.Error()
			m.loaded = true
			return m, nil
		}
		if msg.Proposal != nil {
			m.proposals = msg.Proposal.Pipelines
		}
		m.loaded = true
		m.cursor = 0
		return m, m.emitSelection()

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				return m, m.emitSelection()
			}
		case "down", "j":
			if m.cursor < len(m.proposals)-1 {
				m.cursor++
				return m, m.emitSelection()
			}
		case "enter":
			if m.cursor < len(m.proposals) {
				return m, func() tea.Msg {
					return SuggestLaunchMsg{Pipeline: m.proposals[m.cursor]}
				}
			}
		}
	}
	return m, nil
}

// View renders the suggest list.
func (m SuggestListModel) View() string {
	if !m.loaded {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Loading suggestions...")
	}

	if m.errMsg != "" {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Render(fmt.Sprintf("Error: %s", m.errMsg))
	}

	if len(m.proposals) == 0 {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("No suggestions available")
	}

	var sb strings.Builder
	for i, p := range m.proposals {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		style := lipgloss.NewStyle()
		if i == m.cursor && m.focused {
			style = style.Bold(true).Foreground(lipgloss.Color("12"))
		}

		line := fmt.Sprintf("%s[P%d] %s", cursor, p.Priority, p.Name)
		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(sb.String())
}

func (m SuggestListModel) fetchSuggestions() tea.Msg {
	if m.provider == nil {
		return SuggestDataMsg{Err: fmt.Errorf("no suggest provider")}
	}
	proposal, err := m.provider.FetchSuggestions()
	return SuggestDataMsg{Proposal: proposal, Err: err}
}

func (m SuggestListModel) emitSelection() tea.Cmd {
	if len(m.proposals) == 0 || m.cursor >= len(m.proposals) {
		return nil
	}
	p := m.proposals[m.cursor]
	return func() tea.Msg {
		return SuggestSelectedMsg{Pipeline: p}
	}
}
