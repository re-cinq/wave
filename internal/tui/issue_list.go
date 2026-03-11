package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// IssueListModel is the left pane model for the Issues view.
type IssueListModel struct {
	width        int
	height       int
	issues       []IssueData
	cursor       int
	navigable    []IssueData
	filtering    bool
	filterInput  textinput.Model
	filterQuery  string
	focused      bool
	scrollOffset int
	provider     IssueDataProvider
	loaded       bool
}

// NewIssueListModel creates a new issue list model.
func NewIssueListModel(provider IssueDataProvider) IssueListModel {
	ti := textinput.New()
	ti.Placeholder = "Filter issues..."
	ti.CharLimit = 100

	return IssueListModel{
		provider:    provider,
		filterInput: ti,
		focused:     true,
	}
}

// Init returns the command to fetch issue data.
func (m IssueListModel) Init() tea.Cmd {
	return m.fetchIssueData
}

// SetSize updates the list dimensions.
func (m *IssueListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state.
func (m *IssueListModel) SetFocused(focused bool) {
	m.focused = focused
}

// IsFiltering returns true if the list is in filter mode.
func (m IssueListModel) IsFiltering() bool {
	return m.filtering
}

// Update handles messages to update list state.
func (m IssueListModel) Update(msg tea.Msg) (IssueListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case IssueDataMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.issues = msg.Issues
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

// View renders the issue list.
func (m IssueListModel) View() string {
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
		emptyMsg := "No issues found"
		if m.filtering && m.filterQuery != "" {
			emptyMsg = "No matching issues"
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
		issue := m.navigable[i]
		isSelected := i == m.cursor

		line := m.renderIssueLine(issue, isSelected)
		lines = append(lines, line)
	}

	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m IssueListModel) renderIssueLine(issue IssueData, isSelected bool) string {
	prefix := "  "
	if isSelected {
		prefix = "▶ "
	}

	number := fmt.Sprintf("#%d", issue.Number)

	// Build comment indicator
	var commentStr string
	if issue.Comments > 0 {
		commentStr = fmt.Sprintf(" %d", issue.Comments)
	}

	// Fixed portions: prefix + number + space + commentStr (visual widths)
	fixedWidth := lipgloss.Width(prefix) + lipgloss.Width(number) + 1 + lipgloss.Width(commentStr)
	availableWidth := m.width - fixedWidth
	if availableWidth < 0 {
		availableWidth = 0
	}

	// Give title at least 60% of available space, labels get the rest
	minTitleWidth := availableWidth * 60 / 100
	if minTitleWidth < 10 {
		minTitleWidth = availableWidth // tiny pane: give all to title
	}

	// Build truncated labels that fit in remaining space
	labelBudget := availableWidth - minTitleWidth - 1 // -1 for separator space
	var labelBadges string
	if labelBudget > 4 && len(issue.Labels) > 0 {
		var parts []string
		used := 0
		for _, l := range issue.Labels {
			badge := "[" + l + "]"
			need := lipgloss.Width(badge)
			if used > 0 {
				need++ // space between badges
			}
			if used+need > labelBudget {
				break
			}
			parts = append(parts, badge)
			used += need
		}
		if len(parts) > 0 {
			labelBadges = strings.Join(parts, " ")
		}
	}

	// Calculate actual title width now that we know label width
	labelWidth := lipgloss.Width(labelBadges)
	if labelWidth > 0 {
		labelWidth++ // leading space
	}
	titleWidth := availableWidth - labelWidth
	if titleWidth < 0 {
		titleWidth = 0
	}
	title := truncateName(issue.Title, titleWidth)

	// Assemble: prefix + number + " " + title + spacer + labels + comments
	leftPart := prefix + number + " " + title
	rightPart := ""
	if labelBadges != "" {
		rightPart += " " + labelBadges
	}
	rightPart += commentStr

	spacerWidth := m.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	text := leftPart + strings.Repeat(" ", spacerWidth) + rightPart

	// Pad to full width if needed (for consistent highlight background)
	if pad := m.width - lipgloss.Width(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}

	// Use only MaxWidth (truncates without wrapping) to prevent multi-line output.
	// Width() wraps text and can produce 2+ lines that corrupt the split-pane layout.
	style := lipgloss.NewStyle().
		MaxWidth(m.width)
	if isSelected {
		style = style.Foreground(lipgloss.Color("6"))
	}
	return style.Render(text)
}

func (m IssueListModel) handleKeyMsg(msg tea.KeyMsg) (IssueListModel, tea.Cmd) {
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

func (m IssueListModel) handleNavigation(msg tea.KeyMsg) (IssueListModel, tea.Cmd) {
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

func (m IssueListModel) emitSelectionMsg() tea.Cmd {
	if len(m.navigable) == 0 || m.cursor >= len(m.navigable) {
		return nil
	}

	issue := m.navigable[m.cursor]
	return func() tea.Msg {
		return IssueSelectedMsg{Number: issue.Number, Title: issue.Title, Index: m.cursor}
	}
}

func (m *IssueListModel) buildNavigableItems() {
	query := strings.ToLower(m.filterQuery)
	m.navigable = nil

	for _, issue := range m.issues {
		if query == "" {
			m.navigable = append(m.navigable, issue)
			continue
		}
		// Match against title, number, labels, and assignees
		if strings.Contains(strings.ToLower(issue.Title), query) {
			m.navigable = append(m.navigable, issue)
			continue
		}
		if strings.Contains(fmt.Sprintf("#%d", issue.Number), query) {
			m.navigable = append(m.navigable, issue)
			continue
		}
		matched := false
		for _, l := range issue.Labels {
			if strings.Contains(strings.ToLower(l), query) {
				matched = true
				break
			}
		}
		if matched {
			m.navigable = append(m.navigable, issue)
			continue
		}
		for _, a := range issue.Assignees {
			if strings.Contains(strings.ToLower(a), query) {
				matched = true
				break
			}
		}
		if matched {
			m.navigable = append(m.navigable, issue)
		}
	}
}

func (m *IssueListModel) adjustScrollOffset(visibleHeight int) {
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

func (m IssueListModel) fetchIssueData() tea.Msg {
	if m.provider == nil {
		return IssueDataMsg{Err: nil}
	}

	issues, err := m.provider.FetchIssues()
	return IssueDataMsg{Issues: issues, Err: err}
}
