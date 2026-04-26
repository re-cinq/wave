package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/recinq/wave/internal/humanize"
)

const (
	issueFinishedPerMax = 3 // max finished pipeline runs shown per issue when expanded
)

// issueNavKind identifies the type of navigable item in the issue list.
type issueNavKind int

const (
	issueNavKindIssue    issueNavKind = iota // issue row (parent node)
	issueNavKindRunning                      // running pipeline child
	issueNavKindFinished                     // finished pipeline child
)

// issueNavItem is a single entry in the flat navigation list.
type issueNavItem struct {
	kind      issueNavKind
	issue     *IssueData // non-nil for issue rows
	dataIndex int        // index into running/finished slices for pipeline children
}

// IssueListModel is the left pane model for the Issues view.
type IssueListModel struct {
	width        int
	height       int
	issues       []IssueData
	running      []RunningPipeline
	finished     []FinishedPipeline
	cursor       int
	navigable    []issueNavItem
	collapsed    map[string]bool // keyed by issue HTMLURL
	filtering    bool
	filterInput  textinput.Model
	filterQuery  string
	focused      bool
	scrollOffset int
	provider     IssueDataProvider
	loaded       bool
	tickerActive bool
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
		collapsed:   make(map[string]bool),
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

	case PipelineDataMsg:
		if msg.Err != nil {
			return m, nil
		}

		// Capture selected item identity before rebuild.
		prevID := m.selectedItemID()

		m.running = msg.Running
		m.finished = msg.Finished
		m.buildNavigableItems()

		// Restore cursor to the same item if it still exists.
		m.restoreCursor(prevID)

		var cmds []tea.Cmd
		// Only re-emit selection if the selected item actually changed.
		if newID := m.selectedItemID(); newID != prevID {
			if cmd := m.emitSelectionMsg(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		// Start elapsed ticker if we have running pipelines linked to issues
		if !m.tickerActive && m.hasRunningChildren() {
			m.tickerActive = true
			cmds = append(cmds, tea.Tick(time.Second, func(time.Time) tea.Msg {
				return ElapsedTickMsg{}
			}))
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case ElapsedTickMsg:
		if m.hasRunningChildren() {
			return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
				return ElapsedTickMsg{}
			})
		}
		m.tickerActive = false
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
		item := m.navigable[i]
		isSelected := i == m.cursor

		lines = append(lines, m.renderNavItem(item, i, isSelected))
	}

	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderNavItem dispatches rendering based on item kind.
func (m IssueListModel) renderNavItem(item issueNavItem, idx int, isSelected bool) string {
	switch item.kind {
	case issueNavKindIssue:
		return m.renderIssueLine(item, isSelected)
	case issueNavKindRunning:
		return m.renderRunningChild(item, idx, isSelected)
	case issueNavKindFinished:
		return m.renderFinishedChild(item, idx, isSelected)
	}
	return ""
}

func (m IssueListModel) renderIssueLine(item issueNavItem, isSelected bool) string {
	issue := item.issue
	if issue == nil {
		return ""
	}

	hasChildren := m.issueHasChildren(issue.HTMLURL)
	prefix := "  "
	if hasChildren {
		if m.collapsed[issue.HTMLURL] {
			prefix = "▶ "
		} else {
			prefix = "▼ "
		}
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

	if isSelected {
		style := SelectionStyle(m.focused).
			MaxWidth(m.width)
		return style.Render(text)
	}

	style := lipgloss.NewStyle().
		MaxWidth(m.width)
	return style.Render(text)
}

// renderRunningChild renders a running pipeline child with tree connector.
func (m IssueListModel) renderRunningChild(item issueNavItem, navIdx int, isSelected bool) string {
	if item.dataIndex < 0 || item.dataIndex >= len(m.running) {
		return ""
	}
	r := m.running[item.dataIndex]
	maxWidth := m.width
	if maxWidth <= 0 {
		maxWidth = 40
	}

	connector := "├ "
	if m.isLastChildOf(navIdx) {
		connector = "└ "
	}

	elapsed := formatElapsed(time.Since(r.StartedAt))
	runLabel := r.RunID
	if strings.HasPrefix(runLabel, r.Name+"-") {
		runLabel = runLabel[len(r.Name)+1:]
	}
	displayName := fmt.Sprintf("● %s %s", r.Name, runLabel)

	// Reserve space for connector + displayName + elapsed + padding
	nameMaxWidth := maxWidth - 2 - len(elapsed) - 3
	displayName = truncateName(displayName, nameMaxWidth)

	spacer := maxWidth - lipgloss.Width(connector+displayName) - lipgloss.Width(elapsed) - 1
	if spacer < 1 {
		spacer = 1
	}
	text := fmt.Sprintf("%s%s%s%s", connector, displayName, strings.Repeat(" ", spacer), elapsed)

	if isSelected {
		style := SelectionStyle(m.focused).Width(maxWidth)
		return style.Render(text)
	}

	style := lipgloss.NewStyle().Width(maxWidth)
	return style.Render(text)
}

// renderFinishedChild renders a finished pipeline child with tree connector.
func (m IssueListModel) renderFinishedChild(item issueNavItem, navIdx int, isSelected bool) string {
	if item.dataIndex < 0 || item.dataIndex >= len(m.finished) {
		return ""
	}
	f := m.finished[item.dataIndex]
	maxWidth := m.width
	if maxWidth <= 0 {
		maxWidth = 40
	}

	connector := "├ "
	if m.isLastChildOf(navIdx) {
		connector = "└ "
	}

	statusIcon := "✓"
	if f.Status == "failed" || f.Status == "cancelled" {
		statusIcon = "✗"
	}

	duration := humanize.Duration(f.Duration)
	runLabel := f.RunID
	if strings.HasPrefix(runLabel, f.Name+"-") {
		runLabel = runLabel[len(f.Name)+1:]
	}
	displayName := fmt.Sprintf("%s %s %s", statusIcon, f.Name, runLabel)

	nameMaxWidth := maxWidth - 2 - len(duration) - 3
	displayName = truncateName(displayName, nameMaxWidth)

	spacer := maxWidth - lipgloss.Width(connector+displayName) - lipgloss.Width(duration) - 1
	if spacer < 1 {
		spacer = 1
	}
	text := fmt.Sprintf("%s%s%s%s", connector, displayName, strings.Repeat(" ", spacer), duration)

	if isSelected {
		style := SelectionStyle(m.focused).Width(maxWidth)
		return style.Render(text)
	}

	style := lipgloss.NewStyle().Width(maxWidth)
	return style.Render(text)
}

// isLastChildOf returns true if the navigable item at index i is the last
// child entry under its parent issue.
func (m IssueListModel) isLastChildOf(i int) bool {
	if i+1 >= len(m.navigable) {
		return true
	}
	return m.navigable[i+1].kind == issueNavKindIssue
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

	case tea.KeyEnter, tea.KeySpace:
		// Toggle collapse on issue nodes that have children
		if len(m.navigable) > 0 && m.cursor < len(m.navigable) {
			item := m.navigable[m.cursor]
			if item.kind == issueNavKindIssue && item.issue != nil && m.issueHasChildren(item.issue.HTMLURL) {
				m.collapsed[item.issue.HTMLURL] = !m.collapsed[item.issue.HTMLURL]
				m.buildNavigableItems()
				if m.cursor >= len(m.navigable) {
					m.cursor = len(m.navigable) - 1
				}
				return m, nil
			}
		}
		return m, nil

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

	item := m.navigable[m.cursor]
	switch item.kind {
	case issueNavKindIssue:
		if item.issue == nil {
			return nil
		}
		issue := *item.issue
		return func() tea.Msg {
			return IssueSelectedMsg{Number: issue.Number, Title: issue.Title, Index: m.cursor}
		}
	case issueNavKindRunning:
		if item.dataIndex >= 0 && item.dataIndex < len(m.running) {
			r := m.running[item.dataIndex]
			return func() tea.Msg {
				return PipelineSelectedMsg{
					RunID:         r.RunID,
					Name:          r.Name,
					Input:         r.Input,
					BranchName:    r.BranchName,
					Kind:          itemKindRunning,
					StartedAt:     r.StartedAt,
					FromIssueList: true,
				}
			}
		}
	case issueNavKindFinished:
		if item.dataIndex >= 0 && item.dataIndex < len(m.finished) {
			f := m.finished[item.dataIndex]
			return func() tea.Msg {
				return PipelineSelectedMsg{
					RunID:         f.RunID,
					Name:          f.Name,
					Input:         f.Input,
					BranchName:    f.BranchName,
					Kind:          itemKindFinished,
					StartedAt:     f.StartedAt,
					FromIssueList: true,
				}
			}
		}
	}

	return nil
}

func (m *IssueListModel) buildNavigableItems() {
	query := strings.ToLower(m.filterQuery)
	m.navigable = nil

	for i := range m.issues {
		issue := &m.issues[i]

		if query != "" && !m.issueMatchesFilter(issue, query) {
			continue
		}

		// Default to collapsed for issues not yet toggled.
		if _, seen := m.collapsed[issue.HTMLURL]; !seen {
			m.collapsed[issue.HTMLURL] = true
		}

		// Add issue as parent node.
		m.navigable = append(m.navigable, issueNavItem{
			kind:  issueNavKindIssue,
			issue: issue,
		})

		// Running pipelines linked to this issue — always visible.
		for idx, r := range m.running {
			if m.pipelineLinkedToIssue(r.Input, issue.HTMLURL) {
				m.navigable = append(m.navigable, issueNavItem{
					kind:      issueNavKindRunning,
					issue:     issue,
					dataIndex: idx,
				})
			}
		}

		// Finished pipelines linked to this issue — only when expanded.
		if !m.collapsed[issue.HTMLURL] || query != "" {
			count := 0
			for idx, f := range m.finished {
				if count >= issueFinishedPerMax {
					break
				}
				if m.pipelineLinkedToIssue(f.Input, issue.HTMLURL) {
					m.navigable = append(m.navigable, issueNavItem{
						kind:      issueNavKindFinished,
						issue:     issue,
						dataIndex: idx,
					})
					count++
				}
			}
		}
	}
}

// selectedItemID returns a stable identity string for the currently selected
// navigable item, or "" if nothing is selected.
func (m *IssueListModel) selectedItemID() string {
	if len(m.navigable) == 0 || m.cursor >= len(m.navigable) {
		return ""
	}
	item := m.navigable[m.cursor]
	switch item.kind {
	case issueNavKindIssue:
		if item.issue != nil {
			return fmt.Sprintf("issue:%s", item.issue.HTMLURL)
		}
	case issueNavKindRunning:
		if item.dataIndex >= 0 && item.dataIndex < len(m.running) {
			return fmt.Sprintf("run:%s", m.running[item.dataIndex].RunID)
		}
	case issueNavKindFinished:
		if item.dataIndex >= 0 && item.dataIndex < len(m.finished) {
			return fmt.Sprintf("fin:%s", m.finished[item.dataIndex].RunID)
		}
	}
	return ""
}

// restoreCursor finds the navigable item matching prevID and moves the cursor
// to it, preserving the user's selection across data refreshes.
func (m *IssueListModel) restoreCursor(prevID string) {
	if prevID == "" || len(m.navigable) == 0 {
		if len(m.navigable) == 0 {
			m.cursor = 0
		}
		return
	}
	for i, item := range m.navigable {
		var id string
		switch item.kind {
		case issueNavKindIssue:
			if item.issue != nil {
				id = fmt.Sprintf("issue:%s", item.issue.HTMLURL)
			}
		case issueNavKindRunning:
			if item.dataIndex >= 0 && item.dataIndex < len(m.running) {
				id = fmt.Sprintf("run:%s", m.running[item.dataIndex].RunID)
			}
		case issueNavKindFinished:
			if item.dataIndex >= 0 && item.dataIndex < len(m.finished) {
				id = fmt.Sprintf("fin:%s", m.finished[item.dataIndex].RunID)
			}
		}
		if id == prevID {
			m.cursor = i
			return
		}
	}
	// Item disappeared — clamp cursor.
	if m.cursor >= len(m.navigable) {
		m.cursor = len(m.navigable) - 1
	}
}

// issueMatchesFilter checks if an issue matches the filter query.
func (m *IssueListModel) issueMatchesFilter(issue *IssueData, query string) bool {
	if strings.Contains(strings.ToLower(issue.Title), query) {
		return true
	}
	if strings.Contains(fmt.Sprintf("#%d", issue.Number), query) {
		return true
	}
	for _, l := range issue.Labels {
		if strings.Contains(strings.ToLower(l), query) {
			return true
		}
	}
	for _, a := range issue.Assignees {
		if strings.Contains(strings.ToLower(a), query) {
			return true
		}
	}
	return false
}

// pipelineLinkedToIssue returns true if the pipeline's input references the issue URL.
func (m *IssueListModel) pipelineLinkedToIssue(input, issueURL string) bool {
	if issueURL == "" {
		return false
	}
	return strings.Contains(input, issueURL)
}

// issueHasChildren returns true if any running or finished pipeline is linked to this issue.
func (m *IssueListModel) issueHasChildren(issueURL string) bool {
	if issueURL == "" {
		return false
	}
	for _, r := range m.running {
		if strings.Contains(r.Input, issueURL) {
			return true
		}
	}
	for _, f := range m.finished {
		if strings.Contains(f.Input, issueURL) {
			return true
		}
	}
	return false
}

// hasRunningChildren returns true if any running pipeline is linked to any issue.
func (m *IssueListModel) hasRunningChildren() bool {
	for _, r := range m.running {
		for _, issue := range m.issues {
			if m.pipelineLinkedToIssue(r.Input, issue.HTMLURL) {
				return true
			}
		}
	}
	return false
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
