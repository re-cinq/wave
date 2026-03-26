package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// OntologyDataMsg carries fetched ontology data from the provider.
type OntologyDataMsg struct {
	Overview *OntologyOverview
	Err      error
}

// OntologySelectedMsg signals that a context was selected in the list.
type OntologySelectedMsg struct {
	Name  string
	Index int
}

// OntologyListModel is the left pane model for the Ontology view.
type OntologyListModel struct {
	width        int
	height       int
	items        []OntologyInfo
	telos        string
	stale        bool
	cursor       int
	navigable    []OntologyInfo
	filtering    bool
	filterInput  textinput.Model
	filterQuery  string
	focused      bool
	scrollOffset int
	provider     OntologyDataProvider
	loaded       bool
}

// NewOntologyListModel creates a new ontology list model.
func NewOntologyListModel(provider OntologyDataProvider) OntologyListModel {
	ti := textinput.New()
	ti.Placeholder = "Filter contexts..."
	ti.CharLimit = 100

	return OntologyListModel{
		provider:    provider,
		filterInput: ti,
		focused:     true,
	}
}

// Init returns the command to fetch ontology data.
func (m OntologyListModel) Init() tea.Cmd {
	return m.fetchOntologyData
}

// SetSize updates the list dimensions.
func (m *OntologyListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state.
func (m *OntologyListModel) SetFocused(focused bool) {
	m.focused = focused
}

// Update handles messages.
func (m OntologyListModel) Update(msg tea.Msg) (OntologyListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case OntologyDataMsg:
		if msg.Err != nil {
			return m, nil
		}
		if msg.Overview != nil {
			m.telos = msg.Overview.Telos
			m.stale = msg.Overview.Stale
			m.items = msg.Overview.Contexts
		}
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

// View renders the ontology list.
func (m OntologyListModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	var lines []string
	visibleHeight := m.height

	// Staleness warning
	if m.stale {
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
		lines = append(lines, warnStyle.Render("! stale — run wave analyze"))
		visibleHeight--
	}

	// Telos line takes one row when present
	if m.telos != "" {
		telosLine := m.telos
		if len(telosLine) > m.width-2 {
			telosLine = telosLine[:m.width-5] + "..."
		}
		mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		lines = append(lines, mutedStyle.Render(telosLine))
		visibleHeight--
	}

	if m.filtering {
		filterLine := "/ " + m.filterInput.View()
		lines = append(lines, filterLine)
		visibleHeight--
	}

	if len(m.navigable) == 0 {
		emptyMsg := "No contexts defined"
		if m.filtering && m.filterQuery != "" {
			emptyMsg = "No matching contexts"
		}
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Width(m.width)
		placeholder := style.Render(emptyMsg)

		if m.filtering || m.telos != "" {
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

	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	staleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

	for i := m.scrollOffset; i < endOffset; i++ {
		ctx := m.navigable[i]
		isSelected := i == m.cursor

		name := ctx.Name
		var suffix string

		if ctx.HasSkill {
			age := time.Since(ctx.LastUpdated)
			if age < 24*time.Hour {
				suffix += mutedStyle.Render(fmt.Sprintf(" %s", formatAge(age)))
			} else {
				suffix += staleStyle.Render(fmt.Sprintf(" %s!", formatAge(age)))
			}
		}

		if len(ctx.Invariants) > 0 {
			suffix += mutedStyle.Render(fmt.Sprintf(" (%d inv)", len(ctx.Invariants)))
		}

		if ctx.HasLineage {
			rateStyle := mutedStyle
			if ctx.SuccessRate < 50 {
				rateStyle = staleStyle
			}
			suffix += rateStyle.Render(fmt.Sprintf(" %d runs %.0f%%", ctx.TotalRuns, ctx.SuccessRate))
		}

		nameMaxWidth := m.width - 3 - lipgloss.Width(suffix)
		name = truncateName(name, nameMaxWidth)

		text := fmt.Sprintf("  %s%s", name, suffix)
		if isSelected {
			style := SelectionStyle(m.focused).Width(m.width)
			lines = append(lines, style.Render(text))
		} else {
			style := lipgloss.NewStyle().Width(m.width)
			lines = append(lines, style.Render(text))
		}
	}

	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m OntologyListModel) handleKeyMsg(msg tea.KeyMsg) (OntologyListModel, tea.Cmd) {
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

func (m OntologyListModel) handleNavigation(msg tea.KeyMsg) (OntologyListModel, tea.Cmd) {
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

func (m OntologyListModel) emitSelectionMsg() tea.Cmd {
	if len(m.navigable) == 0 || m.cursor >= len(m.navigable) {
		return nil
	}

	ctx := m.navigable[m.cursor]
	return func() tea.Msg {
		return OntologySelectedMsg{Name: ctx.Name, Index: m.cursor}
	}
}

func (m *OntologyListModel) buildNavigableItems() {
	query := strings.ToLower(m.filterQuery)
	m.navigable = nil

	for _, item := range m.items {
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) ||
			strings.Contains(strings.ToLower(item.Description), query) {
			m.navigable = append(m.navigable, item)
		}
	}
}

func (m *OntologyListModel) adjustScrollOffset(visibleHeight int) {
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

func (m OntologyListModel) fetchOntologyData() tea.Msg {
	if m.provider == nil {
		return OntologyDataMsg{Err: nil}
	}

	overview, err := m.provider.FetchOntology()
	return OntologyDataMsg{Overview: overview, Err: err}
}

func formatAge(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
