package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/humanize"
)

const (
	pipelineRefreshInterval = 5 * time.Second
	finishedPipelineLimit   = 20
	finishedPerPipelineMax  = 3 // max finished runs shown per pipeline when expanded
)

// itemKind identifies the type of navigable item in the flat list.
type itemKind int

const (
	itemKindPipelineName itemKind = iota // tree root: pipeline name
	itemKindRunning
	itemKindFinished
	itemKindAvailable
	itemKindDivider // visual separator between active and archived runs
)

// navigableItem is a single entry in the flat navigation list.
type navigableItem struct {
	kind         itemKind
	pipelineName string // parent pipeline name for tree grouping
	dataIndex    int    // index into section's data slice (-1 for pipeline names)
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

	// Per-pipeline collapse state (collapsed by default)
	collapsed map[string]bool

	// Focus state
	focused bool

	// Scroll state
	scrollOffset int

	// Elapsed ticker state
	tickerActive bool
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
		collapsed:   make(map[string]bool),
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
		// Start elapsed ticker if not already running
		if !m.tickerActive && len(m.running) > 0 {
			m.tickerActive = true
			cmds = append(cmds, tea.Tick(time.Second, func(time.Time) tea.Msg {
				return ElapsedTickMsg{}
			}))
		}
		return m, tea.Batch(cmds...)

	case ElapsedTickMsg:
		if len(m.running) > 0 {
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

// selectedItemID returns a stable identity string for the currently selected
// navigable item, or "" if nothing is selected.
func (m *PipelineListModel) selectedItemID() string {
	if len(m.navigable) == 0 || m.cursor >= len(m.navigable) {
		return ""
	}
	item := m.navigable[m.cursor]
	switch item.kind {
	case itemKindPipelineName:
		return "name:" + item.pipelineName
	case itemKindRunning:
		if item.dataIndex >= 0 && item.dataIndex < len(m.running) {
			return "run:" + m.running[item.dataIndex].RunID
		}
	case itemKindFinished:
		if item.dataIndex >= 0 && item.dataIndex < len(m.finished) {
			return "fin:" + m.finished[item.dataIndex].RunID
		}
	}
	return ""
}

// restoreCursor finds the navigable item matching prevID and moves the cursor
// to it, preserving the user's selection across data refreshes.
func (m *PipelineListModel) restoreCursor(prevID string) {
	if prevID == "" || len(m.navigable) == 0 {
		if len(m.navigable) == 0 {
			m.cursor = 0
		}
		return
	}
	for i, item := range m.navigable {
		var id string
		switch item.kind {
		case itemKindPipelineName:
			id = "name:" + item.pipelineName
		case itemKindRunning:
			if item.dataIndex >= 0 && item.dataIndex < len(m.running) {
				id = "run:" + m.running[item.dataIndex].RunID
			}
		case itemKindFinished:
			if item.dataIndex >= 0 && item.dataIndex < len(m.finished) {
				id = "fin:" + m.finished[item.dataIndex].RunID
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

// handleDataMsg processes a PipelineDataMsg.
func (m PipelineListModel) handleDataMsg(msg PipelineDataMsg) (PipelineListModel, tea.Cmd) {
	if msg.Err != nil {
		return m, nil
	}

	// Capture selected item identity before rebuild.
	prevID := m.selectedItemID()

	m.running = msg.Running
	m.finished = msg.Finished
	m.available = msg.Available
	m.buildNavigableItems()

	// Restore cursor to the same item if it still exists.
	m.restoreCursor(prevID)

	cmds := []tea.Cmd{
		func() tea.Msg { return RunningCountMsg{Count: len(m.running)} },
	}

	// Only re-emit selection if the selected item actually changed.
	if newID := m.selectedItemID(); newID != prevID {
		if cmd := m.emitSelectionMsg(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Start/stop elapsed ticker based on running pipeline count
	if len(m.running) > 0 && !m.tickerActive {
		m.tickerActive = true
		cmds = append(cmds, tea.Tick(time.Second, func(time.Time) tea.Msg {
			return ElapsedTickMsg{}
		}))
	} else if len(m.running) == 0 {
		m.tickerActive = false
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
			// Enter while filtering: deactivate filter but keep results.
			// If the filter matches nothing, stay in filter mode so the
			// user can edit or Escape instead of getting stuck.
			if len(m.navigable) == 0 {
				return m, nil
			}
			m.filtering = false
			return m, nil

		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			newQuery := m.filterInput.Value()
			if newQuery != m.filterQuery {
				m.filterQuery = newQuery
				m.buildNavigableItems()
				// Clamp cursor to valid range after filter narrows results
				if len(m.navigable) == 0 {
					m.cursor = 0
				} else if m.cursor >= len(m.navigable) {
					m.cursor = len(m.navigable) - 1
				}
				if selCmd := m.emitSelectionMsg(); selCmd != nil {
					cmd = tea.Batch(cmd, selCmd)
				}
			}
			return m, cmd
		}
	}

	switch msg.Type {
	case tea.KeyUp, tea.KeyDown:
		return m.handleNavigation(msg)

	case tea.KeyEnter, tea.KeySpace:
		// Toggle collapse on pipeline name nodes
		if len(m.navigable) > 0 && m.cursor < len(m.navigable) {
			item := m.navigable[m.cursor]
			if item.kind == itemKindPipelineName {
				m.collapsed[item.pipelineName] = !m.collapsed[item.pipelineName]
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
			m.buildNavigableItems()
			m.cursor = 0
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
			// Skip divider items
			if m.navigable[m.cursor].kind == itemKindDivider && m.cursor > 0 {
				m.cursor--
			}
		}
	case tea.KeyDown:
		if m.cursor < len(m.navigable)-1 {
			m.cursor++
			// Skip divider items
			if m.navigable[m.cursor].kind == itemKindDivider && m.cursor < len(m.navigable)-1 {
				m.cursor++
			}
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
	case itemKindPipelineName:
		// Pipeline name node emits as itemKindAvailable to preserve detail pane behavior.
		name := item.pipelineName
		return func() tea.Msg {
			return PipelineSelectedMsg{
				RunID: "",
				Name:  name,
				Kind:  itemKindAvailable,
			}
		}
	case itemKindRunning:
		if item.dataIndex >= 0 && item.dataIndex < len(m.running) {
			r := m.running[item.dataIndex]
			return func() tea.Msg {
				return PipelineSelectedMsg{
					RunID:      r.RunID,
					Name:       r.Name,
					Input:      r.Input,
					BranchName: r.BranchName,
					Kind:       itemKindRunning,
					StartedAt:  r.StartedAt,
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
					Input:      f.Input,
					BranchName: f.BranchName,
					Kind:       itemKindFinished,
					StartedAt:  f.StartedAt,
				}
			}
		}
	}

	return nil
}

// buildNavigableItems rebuilds the flat navigable item list as a pipeline tree.
// Each unique pipeline name becomes a collapsible tree root. Running instances
// are always visible; finished runs only appear when the node is expanded
// (limited to finishedPerPipelineMax most recent).
func (m *PipelineListModel) buildNavigableItems() {
	m.navigable = nil
	query := strings.ToLower(m.filterQuery)

	// Index running and finished entries by pipeline name.
	runningByName := make(map[string][]int)
	for i, r := range m.running {
		runningByName[r.Name] = append(runningByName[r.Name], i)
	}
	finishedByName := make(map[string][]int)
	for i, f := range m.finished {
		finishedByName[f.Name] = append(finishedByName[f.Name], i)
	}

	// Collect all unique pipeline names.
	nameSet := make(map[string]bool)
	for _, r := range m.running {
		nameSet[r.Name] = true
	}
	for _, f := range m.finished {
		nameSet[f.Name] = true
	}
	for _, a := range m.available {
		nameSet[a.Name] = true
	}

	// Sort alphabetically.
	names := make([]string, 0, len(nameSet))
	for n := range nameSet {
		names = append(names, n)
	}
	sort.Strings(names)

	// Build available index for fast lookup.
	availableIdx := make(map[string]int)
	for i, a := range m.available {
		availableIdx[a.Name] = i
	}

	for _, name := range names {
		// Apply filter at the pipeline-name level.
		if query != "" && !strings.Contains(strings.ToLower(name), query) {
			continue
		}

		running := runningByName[name]
		finished := finishedByName[name]

		// Default to collapsed for pipelines not yet toggled.
		if _, seen := m.collapsed[name]; !seen {
			m.collapsed[name] = true
		}

		// Pipeline name entry (tree root).
		m.navigable = append(m.navigable, navigableItem{
			kind:         itemKindPipelineName,
			pipelineName: name,
			dataIndex:    -1,
			label:        name,
		})

		// Running entries — always visible regardless of collapse state.
		for _, idx := range running {
			m.navigable = append(m.navigable, navigableItem{
				kind:         itemKindRunning,
				pipelineName: name,
				dataIndex:    idx,
				label:        m.running[idx].Name,
			})
		}

		// Finished entries — only when expanded.
		if !m.collapsed[name] || query != "" {
			limit := finishedPerPipelineMax
			if limit > len(finished) {
				limit = len(finished)
			}
			for _, idx := range finished[:limit] {
				m.navigable = append(m.navigable, navigableItem{
					kind:         itemKindFinished,
					pipelineName: name,
					dataIndex:    idx,
					label:        m.finished[idx].Name,
				})
			}
		}

		// No special entry for available — the pipeline name node itself
		// serves that role. The availableIdx map is used by emitSelectionMsg.
		_ = availableIdx
	}
}

// availableIndexForName returns the index into m.available for the given
// pipeline name, or -1 if not found.
func (m *PipelineListModel) availableIndexForName(name string) int {
	for i, a := range m.available {
		if a.Name == name {
			return i
		}
	}
	return -1
}

// pipelineHasChildren returns true if a pipeline name has any running or
// finished entries.
func (m *PipelineListModel) pipelineHasChildren(name string) bool {
	for _, r := range m.running {
		if r.Name == name {
			return true
		}
	}
	for _, f := range m.finished {
		if f.Name == name {
			return true
		}
	}
	return false
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
	case itemKindPipelineName:
		return m.renderPipelineName(item, isSelected, maxWidth)
	case itemKindRunning:
		return m.renderRunningItem(item, isSelected, maxWidth)
	case itemKindFinished:
		return m.renderFinishedItem(item, isSelected, maxWidth)
	case itemKindDivider:
		return m.renderDivider(maxWidth)
	}
	return ""
}

// renderDivider renders a visual separator line.
func (m PipelineListModel) renderDivider(maxWidth int) string {
	label := "─── Archive ───"
	padLen := maxWidth - lipgloss.Width(label)
	if padLen < 0 {
		padLen = 0
	}
	text := label + strings.Repeat("─", padLen)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(maxWidth).
		Render(text)
}

// isLastChildOf returns true if the given navigable item at index i is the last
// child entry under its parent pipeline name.
func (m PipelineListModel) isLastChildOf(i int) bool {
	if i+1 >= len(m.navigable) {
		return true
	}
	return m.navigable[i+1].kind == itemKindPipelineName
}

// renderPipelineName renders a pipeline tree root node.
func (m PipelineListModel) renderPipelineName(item navigableItem, isSelected bool, maxWidth int) string {
	hasChildren := m.pipelineHasChildren(item.pipelineName)

	nameMaxWidth := maxWidth - 3
	name := truncateName(item.label, nameMaxWidth)

	prefix := "  "
	if hasChildren {
		if m.collapsed[item.pipelineName] {
			prefix = "▶ "
		} else {
			prefix = "▼ "
		}
	}
	text := fmt.Sprintf("%s%s", prefix, name)

	if isSelected {
		style := SelectionStyle(m.focused).
			Bold(true).
			Width(maxWidth)
		return style.Render(text)
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Width(maxWidth)
	return style.Render(text)
}

// renderRunningItem renders a running pipeline item line with tree connector.
func (m PipelineListModel) renderRunningItem(item navigableItem, isSelected bool, maxWidth int) string {
	if item.dataIndex < 0 || item.dataIndex >= len(m.running) {
		return ""
	}
	r := m.running[item.dataIndex]

	elapsed := formatElapsed(time.Since(r.StartedAt))

	// Find this item's index in navigable to determine tree connector
	itemIdx := -1
	for i, n := range m.navigable {
		if n.kind == item.kind && n.dataIndex == item.dataIndex {
			itemIdx = i
			break
		}
	}
	connector := "├ "
	if itemIdx >= 0 && m.isLastChildOf(itemIdx) {
		connector = "└ "
	}

	statusIcon := "●"
	// Strip pipeline name prefix from run ID since the parent node already shows it.
	runLabel := r.RunID
	if strings.HasPrefix(runLabel, r.Name+"-") {
		runLabel = runLabel[len(r.Name)+1:]
	}
	displayName := fmt.Sprintf("%s %s", statusIcon, runLabel)

	// Reserve space: connector (2) + displayName + elapsed + padding
	nameMaxWidth := maxWidth - 2 - len(elapsed) - 3
	displayName = truncateName(displayName, nameMaxWidth)

	spacer := maxWidth - lipgloss.Width(connector+displayName) - lipgloss.Width(elapsed) - 1
	if spacer < 1 {
		spacer = 1
	}
	text := fmt.Sprintf("%s%s%s%s", connector, displayName, strings.Repeat(" ", spacer), elapsed)

	if isSelected {
		style := SelectionStyle(m.focused).
			Width(maxWidth)
		return style.Render(text)
	}

	style := lipgloss.NewStyle().
		Width(maxWidth)
	return style.Render(text)
}

// renderFinishedItem renders a finished pipeline item line with tree connector.
func (m PipelineListModel) renderFinishedItem(item navigableItem, isSelected bool, maxWidth int) string {
	if item.dataIndex < 0 || item.dataIndex >= len(m.finished) {
		return ""
	}
	f := m.finished[item.dataIndex]

	// Find this item's index in navigable to determine tree connector
	itemIdx := -1
	for i, n := range m.navigable {
		if n.kind == item.kind && n.dataIndex == item.dataIndex {
			itemIdx = i
			break
		}
	}
	connector := "├ "
	if itemIdx >= 0 && m.isLastChildOf(itemIdx) {
		connector = "└ "
	}

	statusIcon := "✓"
	if f.Status == "failed" || f.Status == "cancelled" {
		statusIcon = "✗"
	}

	duration := humanize.Duration(f.Duration)
	// Strip pipeline name prefix from run ID since the parent node already shows it.
	runLabel := f.RunID
	if strings.HasPrefix(runLabel, f.Name+"-") {
		runLabel = runLabel[len(f.Name)+1:]
	}
	displayName := fmt.Sprintf("%s %s", statusIcon, runLabel)
	suffix := duration

	nameMaxWidth := maxWidth - 2 - len(suffix) - 3
	displayName = truncateName(displayName, nameMaxWidth)

	spacer := maxWidth - lipgloss.Width(connector+displayName) - lipgloss.Width(suffix) - 1
	if spacer < 1 {
		spacer = 1
	}
	text := fmt.Sprintf("%s%s%s%s", connector, displayName, strings.Repeat(" ", spacer), suffix)

	if isSelected {
		style := SelectionStyle(m.focused).
			Width(maxWidth)
		return style.Render(text)
	}

	style := lipgloss.NewStyle().
		Width(maxWidth)
	return style.Render(text)
}

// truncateName truncates a name with ellipsis if it exceeds maxWidth visual columns.
func truncateName(name string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(name) <= maxWidth {
		return name
	}
	if maxWidth <= 1 {
		return "…"
	}
	// Walk rune by rune, tracking visual width, to find the longest
	// prefix that fits within (maxWidth - 1) columns (reserving 1 for …).
	target := maxWidth - 1
	w := 0
	for i, r := range name {
		rw := lipgloss.Width(string(r))
		if w+rw > target {
			return name[:i] + "…"
		}
		w += rw
	}
	return name + "…"
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
