package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PRListModel is the left pane model for the Pull Requests view.
type PRListModel struct {
	width        int
	height       int
	prs          []PRData
	navigable    []int // indices into prs slice after filtering
	cursor       int
	filtering    bool
	filterInput  textinput.Model
	filterQuery  string
	focused      bool
	scrollOffset int
	provider     PRDataProvider
	loaded       bool
}

// NewPRListModel creates a new PR list model.
func NewPRListModel(provider PRDataProvider) PRListModel {
	ti := textinput.New()
	ti.Placeholder = "Filter pull requests..."
	ti.CharLimit = 100

	return PRListModel{
		provider:    provider,
		filterInput: ti,
		focused:     true,
	}
}

// Init returns the command to fetch PR data.
func (m PRListModel) Init() tea.Cmd {
	return m.fetchPRData
}

// SetSize updates the list dimensions.
func (m *PRListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state.
func (m *PRListModel) SetFocused(focused bool) {
	m.focused = focused
}

// IsFiltering returns true if the list is in filter mode.
func (m PRListModel) IsFiltering() bool {
	return m.filtering
}

// Update handles messages to update list state.
func (m PRListModel) Update(msg tea.Msg) (PRListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case PRDataMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.prs = msg.PRs
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

// View renders the PR list.
func (m PRListModel) View() string {
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
		emptyMsg := "No pull requests found"
		if m.filtering && m.filterQuery != "" {
			emptyMsg = "No matching pull requests"
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
		prIdx := m.navigable[i]
		isSelected := i == m.cursor
		lines = append(lines, m.renderPRLine(&m.prs[prIdx], isSelected))
	}

	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m PRListModel) renderPRLine(pr *PRData, isSelected bool) string {
	number := fmt.Sprintf("#%d", pr.Number)
	badge := m.statusBadge(pr)

	// Fixed portions: "  " prefix + number + " " + badge + " "
	fixedWidth := 2 + lipgloss.Width(number) + 1 + lipgloss.Width(badge) + 1
	availableWidth := m.width - fixedWidth
	if availableWidth < 0 {
		availableWidth = 0
	}

	// Give title at least 60% of available space, labels get the rest
	minTitleWidth := availableWidth * 60 / 100
	if minTitleWidth < 10 {
		minTitleWidth = availableWidth
	}

	// Build truncated labels
	labelBudget := availableWidth - minTitleWidth - 1
	var labelBadges string
	if labelBudget > 4 && len(pr.Labels) > 0 {
		var parts []string
		used := 0
		for _, l := range pr.Labels {
			lb := "[" + l + "]"
			need := lipgloss.Width(lb)
			if used > 0 {
				need++
			}
			if used+need > labelBudget {
				break
			}
			parts = append(parts, lb)
			used += need
		}
		if len(parts) > 0 {
			labelBadges = strings.Join(parts, " ")
		}
	}

	// Calculate actual title width
	labelWidth := lipgloss.Width(labelBadges)
	if labelWidth > 0 {
		labelWidth++
	}
	titleWidth := availableWidth - labelWidth
	if titleWidth < 0 {
		titleWidth = 0
	}
	title := truncateName(pr.Title, titleWidth)

	// Assemble line
	leftPart := "  " + number + " " + title
	rightPart := ""
	if labelBadges != "" {
		rightPart += " " + labelBadges
	}
	rightPart += " " + badge

	spacerWidth := m.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	text := leftPart + strings.Repeat(" ", spacerWidth) + rightPart

	if pad := m.width - lipgloss.Width(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}

	if isSelected {
		style := SelectionStyle(m.focused).MaxWidth(m.width)
		return style.Render(text)
	}

	style := lipgloss.NewStyle().MaxWidth(m.width)
	return style.Render(text)
}

// statusBadge returns a display badge for the PR state.
func (m PRListModel) statusBadge(pr *PRData) string {
	if pr.Draft {
		return "[Draft]"
	}
	if pr.Merged {
		return "[Merged]"
	}
	if pr.State == "closed" {
		return "[Closed]"
	}
	return "[Open]"
}

func (m PRListModel) handleKeyMsg(msg tea.KeyMsg) (PRListModel, tea.Cmd) {
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

func (m PRListModel) handleNavigation(msg tea.KeyMsg) (PRListModel, tea.Cmd) {
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

func (m PRListModel) emitSelectionMsg() tea.Cmd {
	if len(m.navigable) == 0 || m.cursor >= len(m.navigable) {
		return nil
	}

	prIdx := m.navigable[m.cursor]
	pr := m.prs[prIdx]
	return func() tea.Msg {
		return PRSelectedMsg{Number: pr.Number, Title: pr.Title, Index: m.cursor}
	}
}

func (m *PRListModel) buildNavigableItems() {
	query := strings.ToLower(m.filterQuery)
	m.navigable = nil

	for i := range m.prs {
		if query != "" && !m.prMatchesFilter(&m.prs[i], query) {
			continue
		}
		m.navigable = append(m.navigable, i)
	}
}

func (m *PRListModel) prMatchesFilter(pr *PRData, query string) bool {
	if strings.Contains(strings.ToLower(pr.Title), query) {
		return true
	}
	if strings.Contains(fmt.Sprintf("#%d", pr.Number), query) {
		return true
	}
	if strings.Contains(strings.ToLower(pr.Author), query) {
		return true
	}
	for _, l := range pr.Labels {
		if strings.Contains(strings.ToLower(l), query) {
			return true
		}
	}
	return false
}

func (m *PRListModel) adjustScrollOffset(visibleHeight int) {
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

func (m PRListModel) fetchPRData() tea.Msg {
	if m.provider == nil {
		return PRDataMsg{Err: nil}
	}

	prs, err := m.provider.FetchPRs()
	return PRDataMsg{PRs: prs, Err: err}
}
