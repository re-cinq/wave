package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// SuggestListModel is the left pane model for the Suggest view.
type SuggestListModel struct {
	width         int
	height        int
	proposals     []SuggestProposedPipeline
	cursor        int
	focused       bool
	filtering     bool
	loaded        bool
	provider      SuggestDataProvider
	errMsg        string
	selected      map[int]bool // Multi-select state: index -> selected
	healthSummary string
	skipped       map[int]bool
	inputOverlay  *textinput.Model
	overlayTarget int
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

// IsInputActive returns true if the input overlay is active.
func (m SuggestListModel) IsInputActive() bool {
	return m.inputOverlay != nil
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

		// Handle input overlay keys when active
		if m.inputOverlay != nil {
			switch msg.Type {
			case tea.KeyEnter:
				// Confirm modification
				m.proposals[m.overlayTarget].Input = m.inputOverlay.Value()
				m.inputOverlay = nil
				return m, m.emitSelection()
			case tea.KeyEscape:
				// Cancel modification
				m.inputOverlay = nil
				return m, nil
			default:
				var cmd tea.Cmd
				*m.inputOverlay, cmd = m.inputOverlay.Update(msg)
				return m, cmd
			}
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
		case " ":
			// Toggle multi-select on current item
			if m.cursor < len(m.proposals) {
				if m.selected == nil {
					m.selected = make(map[int]bool)
				}
				if m.selected[m.cursor] {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = true
				}
				return m, m.emitSelection()
			}
		case "s":
			// Toggle skip on current item
			if m.cursor < len(m.proposals) {
				if m.skipped == nil {
					m.skipped = make(map[int]bool)
				}
				if m.skipped[m.cursor] {
					delete(m.skipped, m.cursor)
				} else {
					m.skipped[m.cursor] = true
				}
				return m, m.emitSelection()
			}
		case "m":
			// Activate input modification overlay
			if m.cursor < len(m.proposals) {
				ti := textinput.New()
				ti.Placeholder = "Pipeline input..."
				ti.SetValue(m.proposals[m.cursor].Input)
				ti.Focus()
				ti.CharLimit = 500
				m.inputOverlay = &ti
				m.overlayTarget = m.cursor
				return m, ti.Cursor.BlinkCmd()
			}
		case "enter":
			if len(m.selected) > 1 {
				// Multi-select: emit SuggestComposeMsg, filtering out skipped
				var pipelines []SuggestProposedPipeline
				for i, p := range m.proposals {
					if m.selected[i] && !m.skipped[i] {
						pipelines = append(pipelines, p)
					}
				}
				if len(pipelines) > 1 {
					return m, func() tea.Msg {
						return SuggestComposeMsg{Pipelines: pipelines}
					}
				}
				// If filtering reduced to 1, treat as single launch
				if len(pipelines) == 1 {
					p := pipelines[0]
					return m, func() tea.Msg {
						return SuggestLaunchMsg{Pipeline: p}
					}
				}
			}
			if m.cursor < len(m.proposals) && !m.skipped[m.cursor] {
				p := m.proposals[m.cursor]
				return m, func() tea.Msg {
					return SuggestLaunchMsg{Pipeline: p}
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
			Render("No pipeline recommendations\n\nn: manual launch  Tab: fleet view")
	}

	var sb strings.Builder

	// Selection count header
	selCount := len(m.selected)
	if selCount > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render(
			fmt.Sprintf("  %d proposals — %d selected", len(m.proposals), selCount)))
		sb.WriteString("\n")
	}

	if m.healthSummary != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(
			"  " + m.healthSummary))
		sb.WriteString("\n")
	}

	for i, p := range m.proposals {
		prefix := "  "
		isSelected := i == m.cursor

		// Selection marker — plain text when cursor is on this item so
		// SelectionStyle controls all colors (inner ANSI codes break the highlight).
		selMarker := " "
		if m.selected[i] {
			if isSelected {
				selMarker = "●"
			} else {
				selMarker = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render("●")
			}
		}

		// Type badge for sequence/parallel proposals
		typeBadge := ""
		if p.Type == "sequence" {
			if isSelected {
				typeBadge = "[seq] "
			} else {
				typeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("[seq]") + " "
			}
		} else if p.Type == "parallel" {
			if isSelected {
				typeBadge = "[par] "
			} else {
				typeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render("[par]") + " "
			}
		}

		skipped := m.skipped[i]

		line := fmt.Sprintf("%s%s %s[P%d] %s", prefix, selMarker, typeBadge, p.Priority, p.Name)
		if isSelected {
			style := SelectionStyle(m.focused).Width(m.width)
			sb.WriteString(style.Render(line))
		} else if skipped {
			sb.WriteString(lipgloss.NewStyle().Faint(true).Width(m.width).Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	if m.inputOverlay != nil {
		overlay := fmt.Sprintf("  Modify input: %s", m.inputOverlay.View())
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(overlay))
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
	var multi []SuggestProposedPipeline
	if len(m.selected) > 0 {
		for i, prop := range m.proposals {
			if m.selected[i] {
				multi = append(multi, prop)
			}
		}
	}
	return func() tea.Msg {
		return SuggestSelectedMsg{Pipeline: p, MultiSelected: multi}
	}
}
