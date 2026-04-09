package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SuggestListModel is the left pane model for the Suggest view.
type SuggestListModel struct {
	width     int
	height    int
	proposals []SuggestProposedPipeline
	cursor    int
	focused   bool
	filtering bool
	loaded    bool
	provider  SuggestDataProvider
	errMsg    string
	selected  map[int]bool    // Multi-select state: index -> selected
	launched  map[string]bool // Tracks which proposals have been launched by name
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
	case SuggestLaunchedMsg:
		if m.launched == nil {
			m.launched = make(map[string]bool)
		}
		m.launched[msg.Name] = true
		return m, nil

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
		case "enter":
			if len(m.selected) > 1 {
				// Multi-select: emit SuggestComposeMsg
				var pipelines []SuggestProposedPipeline
				for i, p := range m.proposals {
					if m.selected[i] {
						pipelines = append(pipelines, p)
					}
				}
				return m, func() tea.Msg {
					return SuggestComposeMsg{Pipelines: pipelines}
				}
			}
			if m.cursor < len(m.proposals) {
				return m, func() tea.Msg {
					return SuggestLaunchMsg{Pipeline: m.proposals[m.cursor]}
				}
			}
		case "s":
			// Skip/dismiss the current proposal
			if m.cursor < len(m.proposals) {
				m.proposals = append(m.proposals[:m.cursor], m.proposals[m.cursor+1:]...)
				// Rebuild selected map with adjusted indices
				newSelected := make(map[int]bool)
				for idx, sel := range m.selected {
					if idx < m.cursor {
						newSelected[idx] = sel
					} else if idx > m.cursor {
						newSelected[idx-1] = sel
					}
				}
				m.selected = newSelected
				// Adjust cursor
				if m.cursor >= len(m.proposals) && m.cursor > 0 {
					m.cursor--
				}
				return m, m.emitSelection()
			}
		case "m":
			// Modify input before launch
			if m.cursor < len(m.proposals) {
				p := m.proposals[m.cursor]
				return m, func() tea.Msg {
					return SuggestModifyMsg{Pipeline: p}
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

	// Selection count header
	selCount := len(m.selected)
	if selCount > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render(
			fmt.Sprintf("  %d selected", selCount)))
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

		// Launched badge
		launchedBadge := ""
		if m.launched[p.Name] {
			if isSelected {
				launchedBadge = " ✓"
			} else {
				launchedBadge = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("✓")
			}
		}

		line := fmt.Sprintf("%s%s %s[P%d] %s%s", prefix, selMarker, typeBadge, p.Priority, p.Name, launchedBadge)
		if isSelected {
			style := SelectionStyle(m.focused).Width(m.width)
			sb.WriteString(style.Render(line))
		} else {
			sb.WriteString(line)
		}
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
