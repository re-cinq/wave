package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PersonaDataMsg carries fetched persona data from the provider.
type PersonaDataMsg struct {
	Personas []PersonaInfo
	Err      error
}

// PersonaSelectedMsg signals that a persona was selected in the list.
type PersonaSelectedMsg struct {
	Name  string
	Index int
}

// PersonaListModel is the left pane model for the Personas view.
type PersonaListModel struct {
	width        int
	height       int
	items        []PersonaInfo
	cursor       int
	navigable    []PersonaInfo
	filtering    bool
	filterInput  textinput.Model
	filterQuery  string
	focused      bool
	scrollOffset int
	provider     PersonaDataProvider
	loaded       bool
}

// NewPersonaListModel creates a new persona list model.
func NewPersonaListModel(provider PersonaDataProvider) PersonaListModel {
	ti := textinput.New()
	ti.Placeholder = "Filter personas..."
	ti.CharLimit = 100

	return PersonaListModel{
		provider:    provider,
		filterInput: ti,
		focused:     true,
	}
}

// Init returns the command to fetch persona data.
func (m PersonaListModel) Init() tea.Cmd {
	return m.fetchPersonaData
}

// SetSize updates the list dimensions.
func (m *PersonaListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state.
func (m *PersonaListModel) SetFocused(focused bool) {
	m.focused = focused
}

// Update handles messages to update list state.
func (m PersonaListModel) Update(msg tea.Msg) (PersonaListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case PersonaDataMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.items = msg.Personas
		m.loaded = true
		m.buildNavigableItems()

		if len(m.navigable) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(m.navigable) {
			m.cursor = len(m.navigable) - 1
		}

		// Emit initial selection
		if cmd := m.emitSelectionMsg(); cmd != nil {
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

// View renders the persona list.
func (m PersonaListModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	var lines []string
	visibleHeight := m.height

	if m.filtering {
		filterLine := "/ " + m.filterInput.View()
		lines = append(lines, filterLine)
		visibleHeight--
	}

	if len(m.navigable) == 0 {
		emptyMsg := "No personas configured"
		if m.filtering && m.filterQuery != "" {
			emptyMsg = "No matching personas"
		}
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Width(m.width)
		placeholder := style.Render(emptyMsg)

		if m.filtering {
			lines = append(lines, placeholder)
		} else {
			lines = append(lines, lipgloss.Place(m.width, visibleHeight, lipgloss.Center, lipgloss.Center, placeholder))
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	m.adjustScrollOffset(visibleHeight)

	endOffset := m.scrollOffset + visibleHeight
	if endOffset > len(m.navigable) {
		endOffset = len(m.navigable)
	}

	for i := m.scrollOffset; i < endOffset; i++ {
		persona := m.navigable[i]
		isSelected := i == m.cursor

		name := truncateName(persona.Name, m.width-3)
		text := "  " + name
		if isSelected {
			style := SelectionStyle(m.focused).
				Width(m.width)
			lines = append(lines, style.Render(text))
		} else {
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

func (m PersonaListModel) handleKeyMsg(msg tea.KeyMsg) (PersonaListModel, tea.Cmd) {
	if m.filtering {
		switch msg.Type {
		case tea.KeyEscape:
			m.filtering = false
			m.filterQuery = ""
			m.filterInput.SetValue("")
			m.buildNavigableItems()
			m.cursor = 0
			return m, nil

		case tea.KeyUp, tea.KeyDown:
			return m.handleNavigation(msg)

		case tea.KeyEnter:
			m.filtering = false
			return m, nil

		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			newQuery := m.filterInput.Value()
			if newQuery != m.filterQuery {
				m.filterQuery = newQuery
				oldCursor := m.cursor
				m.buildNavigableItems()
				if oldCursor >= len(m.navigable) {
					m.cursor = 0
				}
			}
			return m, cmd
		}
	}

	switch msg.Type {
	case tea.KeyUp, tea.KeyDown:
		return m.handleNavigation(msg)
	default:
		if msg.String() == "/" {
			m.filtering = true
			m.filterInput.SetValue("")
			m.filterQuery = ""
			m.filterInput.Focus()
			return m, m.filterInput.Cursor.BlinkCmd()
		}
	}

	return m, nil
}

func (m PersonaListModel) handleNavigation(msg tea.KeyMsg) (PersonaListModel, tea.Cmd) {
	if len(m.navigable) == 0 {
		return m, nil
	}

	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.navigable)-1 {
			m.cursor++
		}
	}

	return m, m.emitSelectionMsg()
}

func (m PersonaListModel) emitSelectionMsg() tea.Cmd {
	if len(m.navigable) == 0 || m.cursor >= len(m.navigable) {
		return nil
	}

	persona := m.navigable[m.cursor]
	return func() tea.Msg {
		return PersonaSelectedMsg{Name: persona.Name, Index: m.cursor}
	}
}

func (m *PersonaListModel) buildNavigableItems() {
	query := strings.ToLower(m.filterQuery)
	m.navigable = nil

	for _, p := range m.items {
		if query == "" || strings.Contains(strings.ToLower(p.Name), query) {
			m.navigable = append(m.navigable, p)
		}
	}
}

func (m *PersonaListModel) adjustScrollOffset(visibleHeight int) {
	if visibleHeight <= 0 {
		return
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.cursor - visibleHeight + 1
	}
	maxOffset := len(m.navigable) - visibleHeight
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

func (m PersonaListModel) fetchPersonaData() tea.Msg {
	if m.provider == nil {
		return PersonaDataMsg{Err: nil}
	}

	personas, err := m.provider.FetchPersonas()
	return PersonaDataMsg{Personas: personas, Err: err}
}
