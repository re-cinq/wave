package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	pipelineRefreshInterval = 5 * time.Second
	finishedPipelineLimit   = 20
)

// itemKind identifies the type of navigable item in the flat list.
type itemKind int

const (
	itemKindSectionHeader itemKind = iota
	itemKindRunning
	itemKindFinished
	itemKindAvailable
)

// navigableItem is a single entry in the flat navigation list.
type navigableItem struct {
	kind         itemKind
	sectionIndex int    // 0=Running, 1=Finished, 2=Available
	dataIndex    int    // index into section's data slice (-1 for headers)
	label        string // display text
}

// PipelineListModel is the Bubble Tea model for the pipeline list left pane.
type PipelineListModel struct {
	width    int
	height   int
	provider PipelineDataProvider

	// Section data
	running   []RunningPipeline
	finished  []FinishedPipeline
	available []PipelineInfo

	// Navigation state
	cursor    int
	navigable []navigableItem

	// Filter state
	filtering   bool
	filterInput textinput.Model
	filterQuery string

	// Section collapse state
	collapsed [3]bool // [Running, Finished, Available]

	// Focus state
	focused bool

	// Scroll state
	scrollOffset int
}

// NewPipelineListModel creates a new pipeline list model with the given data provider.
func NewPipelineListModel(provider PipelineDataProvider) PipelineListModel {
	ti := textinput.New()
	ti.Placeholder = "Filter pipelines..."
	ti.CharLimit = 100

	return PipelineListModel{
		provider:    provider,
		filterInput: ti,
		focused:     true,
	}
}

// Init returns commands to fetch initial data and start the refresh timer.
func (m PipelineListModel) Init() tea.Cmd {
	return tea.Batch(m.fetchPipelineData, m.refreshTick())
}

// SetSize updates the list dimensions for reflow.
func (m *PipelineListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state of the list model.
func (m *PipelineListModel) SetFocused(focused bool) {
	m.focused = focused
}

// Update handles messages to update list state.
func (m PipelineListModel) Update(msg tea.Msg) (PipelineListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case PipelineDataMsg:
		return m.handleDataMsg(msg)

	case PipelineRefreshTickMsg:
		return m, tea.Batch(m.fetchPipelineData, m.refreshTick())

	case PipelineLaunchedMsg:
		// Insert synthetic running entry at the top
		newRunning := RunningPipeline{
			RunID:     msg.RunID,
			Name:      msg.PipelineName,
			StartedAt: time.Now(),
		}
		m.running = append([]RunningPipeline{newRunning}, m.running...)
		m.buildNavigableItems()

		// Move cursor to the new running entry
		for i, item := range m.navigable {
			if item.kind == itemKindRunning {
				m.cursor = i
				break
			}
		}

		// Emit running count and selection messages
		cmds := []tea.Cmd{
			func() tea.Msg { return RunningCountMsg{Count: len(m.running)} },
		}
		if cmd := m.emitSelectionMsg(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// View renders the pipeline list.
func (m PipelineListModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	var lines []string
	visibleHeight := m.height

	// Render filter input if active
	if m.filtering {
		filterLine := "/ " + m.filterInput.View()
		lines = append(lines, filterLine)
		visibleHeight--
	}

	if len(m.navigable) == 0 {
		// Empty state
		emptyMsg := "No pipelines found"
		if m.filtering && m.filterQuery != "" {
			emptyMsg = "No matching pipelines"
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

	// Ensure scroll offset keeps cursor visible
	m.adjustScrollOffset(visibleHeight)

	// Render visible items
	endOffset := m.scrollOffset + visibleHeight
	if endOffset > len(m.navigable) {
		endOffset = len(m.navigable)
	}

	for i := m.scrollOffset; i < endOffset; i++ {
		item := m.navigable[i]
		isSelected := i == m.cursor
		lines = append(lines, m.renderItem(item, isSelected))
	}

	// Pad remaining height
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// handleDataMsg processes a PipelineDataMsg.
func (m PipelineListModel) handleDataMsg(msg PipelineDataMsg) (PipelineListModel, tea.Cmd) {
	if msg.Err != nil {
		return m, nil
	}

	m.running = msg.Running
	m.finished = msg.Finished
	m.available = msg.Available
	m.buildNavigableItems()

	// Clamp cursor to new bounds
	if len(m.navigable) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.navigable) {
		m.cursor = len(m.navigable) - 1
	}

	cmds := []tea.Cmd{
		func() tea.Msg { return RunningCountMsg{Count: len(m.running)} },
	}

	// Re-emit PipelineSelectedMsg if cursor is on a pipeline item
	if cmd := m.emitSelectionMsg(); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg processes keyboard input.
func (m PipelineListModel) handleKeyMsg(msg tea.KeyMsg) (PipelineListModel, tea.Cmd) {
	// When filtering, forward most keys to filter input
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
			// Allow navigation while filtering
			return m.handleNavigation(msg)

		case tea.KeyEnter:
			// Enter while filtering: deactivate filter but keep results
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

	case tea.KeyEnter:
		// Toggle collapse on section headers
		if len(m.navigable) > 0 && m.cursor < len(m.navigable) {
			item := m.navigable[m.cursor]
			if item.kind == itemKindSectionHeader {
				m.collapsed[item.sectionIndex] = !m.collapsed[item.sectionIndex]
				m.buildNavigableItems()
				// Clamp cursor after rebuild
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

// handleNavigation processes up/down arrow key events.
func (m PipelineListModel) handleNavigation(msg tea.KeyMsg) (PipelineListModel, tea.Cmd) {
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

// emitSelectionMsg returns a command to emit PipelineSelectedMsg if cursor is on a pipeline item.
func (m PipelineListModel) emitSelectionMsg() tea.Cmd {
	if len(m.navigable) == 0 || m.cursor >= len(m.navigable) {
		return nil
	}

	item := m.navigable[m.cursor]
	switch item.kind {
	case itemKindRunning:
		if item.dataIndex >= 0 && item.dataIndex < len(m.running) {
			r := m.running[item.dataIndex]
			return func() tea.Msg {
				return PipelineSelectedMsg{
					RunID:      r.RunID,
					Name:       r.Name,
					BranchName: r.BranchName,
					Kind:       itemKindRunning,
				}
			}
		}
	case itemKindFinished:
		if item.dataIndex >= 0 && item.dataIndex < len(m.finished) {
			f := m.finished[item.dataIndex]
			return func() tea.Msg {
				return PipelineSelectedMsg{
					RunID:      f.RunID,
					Name:       f.Name,
					BranchName: f.BranchName,
					Kind:       itemKindFinished,
				}
			}
		}
	case itemKindAvailable:
		if item.dataIndex >= 0 && item.dataIndex < len(m.available) {
			a := m.available[item.dataIndex]
			return func() tea.Msg {
				return PipelineSelectedMsg{
					RunID: "",
					Name:  a.Name,
					Kind:  itemKindAvailable,
				}
			}
		}
	}

	return nil
}

// buildNavigableItems rebuilds the flat navigable item list from section data.
func (m *PipelineListModel) buildNavigableItems() {
	m.navigable = nil
	query := strings.ToLower(m.filterQuery)

	// Running section
	var filteredRunning []int
	for i, r := range m.running {
		if query == "" || strings.Contains(strings.ToLower(r.Name), query) {
			filteredRunning = append(filteredRunning, i)
		}
	}
	if len(filteredRunning) > 0 || query == "" {
		m.navigable = append(m.navigable, navigableItem{
			kind:         itemKindSectionHeader,
			sectionIndex: 0,
			dataIndex:    -1,
			label:        fmt.Sprintf("Running (%d)", len(filteredRunning)),
		})
		if !m.collapsed[0] {
			for _, idx := range filteredRunning {
				m.navigable = append(m.navigable, navigableItem{
					kind:         itemKindRunning,
					sectionIndex: 0,
					dataIndex:    idx,
					label:        m.running[idx].Name,
				})
			}
		}
	}

	// Finished section
	var filteredFinished []int
	for i, f := range m.finished {
		if query == "" || strings.Contains(strings.ToLower(f.Name), query) {
			filteredFinished = append(filteredFinished, i)
		}
	}
	if len(filteredFinished) > 0 || query == "" {
		m.navigable = append(m.navigable, navigableItem{
			kind:         itemKindSectionHeader,
			sectionIndex: 1,
			dataIndex:    -1,
			label:        fmt.Sprintf("Finished (%d)", len(filteredFinished)),
		})
		if !m.collapsed[1] {
			for _, idx := range filteredFinished {
				m.navigable = append(m.navigable, navigableItem{
					kind:         itemKindFinished,
					sectionIndex: 1,
					dataIndex:    idx,
					label:        m.finished[idx].Name,
				})
			}
		}
	}

	// Available section
	var filteredAvailable []int
	for i, a := range m.available {
		if query == "" || strings.Contains(strings.ToLower(a.Name), query) {
			filteredAvailable = append(filteredAvailable, i)
		}
	}
	if len(filteredAvailable) > 0 || query == "" {
		m.navigable = append(m.navigable, navigableItem{
			kind:         itemKindSectionHeader,
			sectionIndex: 2,
			dataIndex:    -1,
			label:        fmt.Sprintf("Available (%d)", len(filteredAvailable)),
		})
		if !m.collapsed[2] {
			for _, idx := range filteredAvailable {
				m.navigable = append(m.navigable, navigableItem{
					kind:         itemKindAvailable,
					sectionIndex: 2,
					dataIndex:    idx,
					label:        m.available[idx].Name,
				})
			}
		}
	}
}

// adjustScrollOffset ensures the cursor is within the visible window.
func (m *PipelineListModel) adjustScrollOffset(visibleHeight int) {
	if visibleHeight <= 0 {
		return
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.cursor - visibleHeight + 1
	}
	// Clamp scroll offset
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

// renderItem renders a single navigable item line.
func (m PipelineListModel) renderItem(item navigableItem, isSelected bool) string {
	maxWidth := m.width
	if maxWidth <= 0 {
		maxWidth = 40
	}

	switch item.kind {
	case itemKindSectionHeader:
		return m.renderSectionHeader(item, isSelected, maxWidth)
	case itemKindRunning:
		return m.renderRunningItem(item, isSelected, maxWidth)
	case itemKindFinished:
		return m.renderFinishedItem(item, isSelected, maxWidth)
	case itemKindAvailable:
		return m.renderAvailableItem(item, isSelected, maxWidth)
	}
	return ""
}

// renderSectionHeader renders a section header line.
func (m PipelineListModel) renderSectionHeader(item navigableItem, isSelected bool, maxWidth int) string {
	// Collapse indicator
	indicator := "▾"
	if m.collapsed[item.sectionIndex] {
		indicator = "▸"
	}

	text := fmt.Sprintf("%s %s", indicator, item.label)

	if isSelected {
		style := lipgloss.NewStyle().
			Bold(true).
			Reverse(true).
			Width(maxWidth)
		return style.Render(text)
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("7")).
		Width(maxWidth)
	return style.Render(text)
}

// renderRunningItem renders a running pipeline item line.
func (m PipelineListModel) renderRunningItem(item navigableItem, isSelected bool, maxWidth int) string {
	if item.dataIndex < 0 || item.dataIndex >= len(m.running) {
		return ""
	}
	r := m.running[item.dataIndex]

	elapsed := formatDuration(time.Since(r.StartedAt))
	// Reserve space: prefix (3) + elapsed + padding
	nameMaxWidth := maxWidth - 3 - len(elapsed) - 3
	name := truncateName(r.Name, nameMaxWidth)

	if isSelected {
		spacer := maxWidth - lipgloss.Width("▶ "+name) - lipgloss.Width(elapsed) - 1
		if spacer < 1 {
			spacer = 1
		}
		text := fmt.Sprintf("▶ %s%s%s", name, strings.Repeat(" ", spacer), elapsed)
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Width(maxWidth)
		return style.Render(text)
	}

	spacer := maxWidth - lipgloss.Width("  "+name) - lipgloss.Width(elapsed) - 1
	if spacer < 1 {
		spacer = 1
	}
	text := fmt.Sprintf("  %s%s%s", name, strings.Repeat(" ", spacer), elapsed)
	style := lipgloss.NewStyle().
		Width(maxWidth)
	return style.Render(text)
}

// renderFinishedItem renders a finished pipeline item line.
func (m PipelineListModel) renderFinishedItem(item navigableItem, isSelected bool, maxWidth int) string {
	if item.dataIndex < 0 || item.dataIndex >= len(m.finished) {
		return ""
	}
	f := m.finished[item.dataIndex]

	statusIcon := "✓"
	if f.Status == "failed" || f.Status == "cancelled" {
		statusIcon = "✗"
	}

	duration := formatDuration(f.Duration)
	suffix := fmt.Sprintf("%s %s  %s", statusIcon, f.Status, duration)

	nameMaxWidth := maxWidth - 3 - len(suffix) - 1
	name := truncateName(f.Name, nameMaxWidth)

	if isSelected {
		spacer := maxWidth - lipgloss.Width("▶ "+name) - lipgloss.Width(suffix) - 1
		if spacer < 1 {
			spacer = 1
		}
		text := fmt.Sprintf("▶ %s%s%s", name, strings.Repeat(" ", spacer), suffix)
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Width(maxWidth)
		return style.Render(text)
	}

	spacer := maxWidth - lipgloss.Width("  "+name) - lipgloss.Width(suffix) - 1
	if spacer < 1 {
		spacer = 1
	}
	text := fmt.Sprintf("  %s%s%s", name, strings.Repeat(" ", spacer), suffix)
	style := lipgloss.NewStyle().
		Width(maxWidth)
	return style.Render(text)
}

// renderAvailableItem renders an available pipeline item line.
func (m PipelineListModel) renderAvailableItem(item navigableItem, isSelected bool, maxWidth int) string {
	if item.dataIndex < 0 || item.dataIndex >= len(m.available) {
		return ""
	}
	a := m.available[item.dataIndex]

	nameMaxWidth := maxWidth - 3
	name := truncateName(a.Name, nameMaxWidth)

	if isSelected {
		text := fmt.Sprintf("▶ %s", name)
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Width(maxWidth)
		return style.Render(text)
	}

	text := fmt.Sprintf("  %s", name)
	style := lipgloss.NewStyle().
		Width(maxWidth)
	return style.Render(text)
}

// truncateName truncates a name with ellipsis if it exceeds maxWidth.
func truncateName(name string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if len(name) <= maxWidth {
		return name
	}
	if maxWidth <= 1 {
		return "…"
	}
	return name[:maxWidth-1] + "…"
}

// formatDuration formats a duration in a compact human-readable form.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", h, m)
}

// Async command factories

func (m PipelineListModel) fetchPipelineData() tea.Msg {
	if m.provider == nil {
		return PipelineDataMsg{Err: fmt.Errorf("no provider")}
	}

	running, runErr := m.provider.FetchRunningPipelines()
	if runErr != nil {
		return PipelineDataMsg{Err: runErr}
	}

	finished, finErr := m.provider.FetchFinishedPipelines(finishedPipelineLimit)
	if finErr != nil {
		return PipelineDataMsg{Err: finErr}
	}

	available, avErr := m.provider.FetchAvailablePipelines()
	if avErr != nil {
		return PipelineDataMsg{Err: avErr}
	}

	return PipelineDataMsg{
		Running:   running,
		Finished:  finished,
		Available: available,
	}
}

func (m PipelineListModel) refreshTick() tea.Cmd {
	return tea.Tick(pipelineRefreshInterval, func(time.Time) tea.Msg {
		return PipelineRefreshTickMsg{}
	})
}
