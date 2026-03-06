package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ContentModel is the main content area component composing a left pipeline list pane and a right detail pane.
type ContentModel struct {
	width    int
	height   int
	list     PipelineListModel
	detail   PipelineDetailModel
	focus    FocusPane
	launcher *PipelineLauncher
}

// NewContentModel creates a new content model with the given pipeline data providers.
func NewContentModel(provider PipelineDataProvider, detailProvider DetailDataProvider, deps LaunchDependencies) ContentModel {
	var launcher *PipelineLauncher
	if deps.Manifest != nil {
		launcher = NewPipelineLauncher(deps)
	}
	return ContentModel{
		list:     NewPipelineListModel(provider),
		detail:   NewPipelineDetailModel(detailProvider),
		focus:    FocusPaneLeft,
		launcher: launcher,
	}
}

// CancelAll cancels all running pipelines managed by the launcher.
func (m *ContentModel) CancelAll() {
	if m.launcher != nil {
		m.launcher.CancelAll()
	}
}

// Init returns commands from child components.
func (m ContentModel) Init() tea.Cmd {
	return tea.Batch(m.list.Init(), m.detail.Init())
}

// SetSize updates the content area dimensions and propagates to children.
func (m *ContentModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	leftWidth := m.leftPaneWidth()
	m.list.SetSize(leftWidth, h)

	rightWidth := w - leftWidth
	m.detail.SetSize(rightWidth, h)
}

// Update handles messages by forwarding to child components with focus-aware routing.
func (m ContentModel) Update(msg tea.Msg) (ContentModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter && m.focus == FocusPaneLeft && !m.list.filtering {
			if m.cursorOnFocusableItem() {
				item := m.list.navigable[m.list.cursor]
				m.focus = FocusPaneRight
				m.list.SetFocused(false)
				m.detail.SetFocused(true)

				enterCmds := []tea.Cmd{
					func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneRight} },
				}

				// For available items, also send ConfigureFormMsg to show the launch form
				if item.kind == itemKindAvailable && item.dataIndex >= 0 && item.dataIndex < len(m.list.available) {
					a := m.list.available[item.dataIndex]
					enterCmds = append(enterCmds, func() tea.Msg {
						return ConfigureFormMsg{PipelineName: a.Name, InputExample: a.InputExample}
					})
					enterCmds = append(enterCmds, func() tea.Msg {
						return FormActiveMsg{Active: true}
					})
				}

				// For running items with TUI buffer, activate live output
				if item.kind == itemKindRunning && item.dataIndex >= 0 && item.dataIndex < len(m.list.running) {
					r := m.list.running[item.dataIndex]
					if m.launcher != nil && m.launcher.HasBuffer(r.RunID) {
						buf := m.launcher.GetBuffer(r.RunID)
						liveModel := NewLiveOutputModel(r.RunID, r.Name, buf, r.StartedAt, 0)
						liveModel.SetSize(m.detail.width, m.detail.height)
						m.detail.liveOutput = &liveModel
						m.detail.paneState = stateRunningLive
						enterCmds = append(enterCmds, func() tea.Msg {
							return LiveOutputActiveMsg{Active: true}
						})
					}
				}

				// For finished items, activate finished detail hints
				if item.kind == itemKindFinished {
					enterCmds = append(enterCmds, func() tea.Msg {
						return FinishedDetailActiveMsg{Active: true}
					})
				}

				return m, tea.Batch(enterCmds...)
			}
			// Section header or non-focusable — forward to list for collapse/no-op
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		if msg.Type == tea.KeyEscape && m.focus == FocusPaneRight {
			m.focus = FocusPaneLeft
			m.list.SetFocused(true)
			m.detail.SetFocused(false)
			return m, tea.Batch(
				func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} },
				func() tea.Msg { return LiveOutputActiveMsg{Active: false} },
				func() tea.Msg { return FinishedDetailActiveMsg{Active: false} },
			)
		}

		// Cancel running pipeline with 'c' key
		if msg.String() == "c" && m.focus == FocusPaneLeft && m.launcher != nil {
			if len(m.list.navigable) > 0 && m.list.cursor < len(m.list.navigable) {
				item := m.list.navigable[m.list.cursor]
				if item.kind == itemKindRunning && item.dataIndex >= 0 && item.dataIndex < len(m.list.running) {
					r := m.list.running[item.dataIndex]
					m.launcher.Cancel(r.RunID)
				}
			}
			return m, nil
		}

		// Route key messages to the focused child only
		if m.focus == FocusPaneRight {
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case PipelineSelectedMsg:
		var listCmd, detailCmd tea.Cmd
		m.list, listCmd = m.list.Update(msg)
		m.detail, detailCmd = m.detail.Update(msg)
		if listCmd != nil {
			cmds = append(cmds, listCmd)
		}
		if detailCmd != nil {
			cmds = append(cmds, detailCmd)
		}
		return m, tea.Batch(cmds...)

	case DetailDataMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case PipelineDataMsg, PipelineRefreshTickMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case PipelineEventMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case TransitionTimerMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case ChatSessionEndedMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case BranchCheckoutMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case DiffViewEndedMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case ElapsedTickMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case ConfigureFormMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case LaunchRequestMsg:
		if m.launcher != nil {
			cmd := m.launcher.Launch(msg.Config)
			return m, cmd
		}
		return m, nil

	case PipelineLaunchedMsg:
		// Forward to list for running entry insertion
		var listCmd tea.Cmd
		m.list, listCmd = m.list.Update(msg)
		// Transition focus to left pane
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
		focusCmd := func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} }
		formCmd := func() tea.Msg { return FormActiveMsg{Active: false} }
		batchCmds := []tea.Cmd{focusCmd, formCmd}
		if listCmd != nil {
			batchCmds = append(batchCmds, listCmd)
		}
		return m, tea.Batch(batchCmds...)

	case PipelineLaunchResultMsg:
		if m.launcher != nil {
			m.launcher.Cleanup(msg.RunID)
		}
		// Trigger data refresh so the pipeline moves from Running to Finished
		return m, m.list.fetchPipelineData

	case LaunchErrorMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		// Transition focus to left pane
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
		focusCmd := func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} }
		formCmd := func() tea.Msg { return FormActiveMsg{Active: false} }
		batchCmds := []tea.Cmd{focusCmd, formCmd}
		if cmd != nil {
			batchCmds = append(batchCmds, cmd)
		}
		return m, tea.Batch(batchCmds...)

	case FocusChangedMsg:
		if msg.Pane == FocusPaneLeft {
			m.focus = FocusPaneLeft
			m.list.SetFocused(true)
			m.detail.SetFocused(false)
		}
		return m, nil
	}

	// Default: forward to both children
	var listCmd, detailCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	m.detail, detailCmd = m.detail.Update(msg)
	if listCmd != nil {
		cmds = append(cmds, listCmd)
	}
	if detailCmd != nil {
		cmds = append(cmds, detailCmd)
	}
	return m, tea.Batch(cmds...)
}

// cursorOnFocusableItem returns true if the cursor is on an available, finished, or running item.
func (m ContentModel) cursorOnFocusableItem() bool {
	if len(m.list.navigable) == 0 || m.list.cursor >= len(m.list.navigable) {
		return false
	}
	kind := m.list.navigable[m.list.cursor].kind
	return kind == itemKindAvailable || kind == itemKindFinished || kind == itemKindRunning
}

// View renders the content area with left pipeline list and right detail pane.
func (m ContentModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	leftView := m.list.View()
	rightView := m.detail.View()

	// Apply dimming when focus is on the right pane
	if m.focus == FocusPaneRight {
		leftView = lipgloss.NewStyle().
			Faint(true).
			Render(leftView)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)
}

// leftPaneWidth computes the left pane width: 30% of total, min 25, max 50.
func (m ContentModel) leftPaneWidth() int {
	w := m.width * 30 / 100
	if w < 25 {
		w = 25
	}
	if w > 50 {
		w = 50
	}
	if w > m.width {
		w = m.width
	}
	return w
}
