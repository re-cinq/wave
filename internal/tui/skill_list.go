package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SkillDataMsg carries fetched skill data from the provider.
type SkillDataMsg struct {
	Skills []SkillInfo
	Err    error
}

// SkillSelectedMsg signals that a skill was selected in the list.
type SkillSelectedMsg struct {
	Name  string
	Index int
}

// SkillListModel is the left pane model for the Skills view.
type SkillListModel struct {
	width        int
	height       int
	items        []SkillInfo
	cursor       int
	navigable    []SkillInfo
	filtering    bool
	filterInput  textinput.Model
	filterQuery  string
	focused      bool
	scrollOffset int
	provider     SkillDataProvider
	loaded       bool
}

// NewSkillListModel creates a new skill list model.
func NewSkillListModel(provider SkillDataProvider) SkillListModel {
	ti := textinput.New()
	ti.Placeholder = "Filter skills..."
	ti.CharLimit = 100

	return SkillListModel{
		provider:    provider,
		filterInput: ti,
		focused:     true,
	}
}

// Init returns the command to fetch skill data.
func (m SkillListModel) Init() tea.Cmd {
	return m.fetchSkillData
}

// SetSize updates the list dimensions.
func (m *SkillListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state.
func (m *SkillListModel) SetFocused(focused bool) {
	m.focused = focused
}

// Update handles messages.
func (m SkillListModel) Update(msg tea.Msg) (SkillListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case SkillDataMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.items = msg.Skills
		m.loaded = true
		m.buildNavigableItems()

		if len(m.navigable) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(m.navigable) {
			m.cursor = len(m.navigable) - 1
		}

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

// View renders the skill list.
func (m SkillListModel) View() string {
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
		emptyMsg := "No skills configured"
		if m.filtering && m.filterQuery != "" {
			emptyMsg = "No matching skills"
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
		skill := m.navigable[i]
		isSelected := i == m.cursor

		badge := fmt.Sprintf("(%d cmds)", len(skill.CommandFiles))
		nameMaxWidth := m.width - 3 - len(badge) - 1
		name := truncateName(skill.Name, nameMaxWidth)

		if isSelected {
			spacer := m.width - lipgloss.Width("▶ "+name) - lipgloss.Width(badge) - 1
			if spacer < 1 {
				spacer = 1
			}
			text := fmt.Sprintf("▶ %s%s%s", name, strings.Repeat(" ", spacer), badge)
			style := lipgloss.NewStyle().
				Foreground(lipgloss.Color("6")).
				Width(m.width)
			lines = append(lines, style.Render(text))
		} else {
			spacer := m.width - lipgloss.Width("  "+name) - lipgloss.Width(badge) - 1
			if spacer < 1 {
				spacer = 1
			}
			text := fmt.Sprintf("  %s%s%s", name, strings.Repeat(" ", spacer), badge)
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

func (m SkillListModel) handleKeyMsg(msg tea.KeyMsg) (SkillListModel, tea.Cmd) {
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

func (m SkillListModel) handleNavigation(msg tea.KeyMsg) (SkillListModel, tea.Cmd) {
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

func (m SkillListModel) emitSelectionMsg() tea.Cmd {
	if len(m.navigable) == 0 || m.cursor >= len(m.navigable) {
		return nil
	}

	skill := m.navigable[m.cursor]
	return func() tea.Msg {
		return SkillSelectedMsg{Name: skill.Name, Index: m.cursor}
	}
}

func (m *SkillListModel) buildNavigableItems() {
	query := strings.ToLower(m.filterQuery)
	m.navigable = nil

	for _, s := range m.items {
		if query == "" || strings.Contains(strings.ToLower(s.Name), query) {
			m.navigable = append(m.navigable, s)
		}
	}
}

func (m *SkillListModel) adjustScrollOffset(visibleHeight int) {
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

func (m SkillListModel) fetchSkillData() tea.Msg {
	if m.provider == nil {
		return SkillDataMsg{Err: nil}
	}

	skills, err := m.provider.FetchSkills()
	return SkillDataMsg{Skills: skills, Err: err}
}
