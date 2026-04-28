package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/state"
)

// setSizePipeline propagates size changes to the pipeline list/detail and compose models.
func (m *ContentModel) setSizePipeline(leftWidth, rightWidth, h int) {
	m.list.SetSize(leftWidth, h)
	m.detail.SetSize(rightWidth, h)
	if m.composeList != nil {
		m.composeList.SetSize(leftWidth, h)
	}
	if m.composeDetail != nil {
		m.composeDetail.SetSize(rightWidth, h)
	}
}

// cursorOnFocusableItem returns true if the cursor is on a pipeline name, finished, or running item.
func (m ContentModel) cursorOnFocusableItem() bool {
	if len(m.list.navigable) == 0 || m.list.cursor >= len(m.list.navigable) {
		return false
	}
	kind := m.list.navigable[m.list.cursor].kind
	return kind == itemKindPipelineName || kind == itemKindFinished || kind == itemKindRunning
}

// updateComposeMessage routes compose-mode messages.
// Returns (model, cmd, handled). handled=true means the message belonged to compose mode.
func (m ContentModel) updateComposeMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case ComposeCancelMsg:
		m.composing = false
		m.composeList = nil
		m.composeDetail = nil
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
		return m, tea.Batch(
			func() tea.Msg { return ComposeActiveMsg{Active: false} },
			func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} },
		), true

	case ComposeStartMsg:
		if msg.Sequence.IsSingle() {
			// Single-pipeline sequence delegates to normal launch.
			m.composing = false
			m.composeList = nil
			m.composeDetail = nil
			entry := msg.Sequence.Entries[0]
			return m, tea.Batch(
				func() tea.Msg { return ComposeActiveMsg{Active: false} },
				func() tea.Msg {
					return LaunchRequestMsg{Config: LaunchConfig{PipelineName: entry.PipelineName}}
				},
			), true
		}
		// Multi-pipeline sequence — launch via orchestrated `wave compose` subprocess.
		if len(msg.Sequence.Entries) > 0 {
			m.composing = false
			m.composeList = nil
			m.composeDetail = nil

			names := make([]string, len(msg.Sequence.Entries))
			for i, e := range msg.Sequence.Entries {
				names[i] = e.PipelineName
			}

			var cmds []tea.Cmd
			cmds = append(cmds, func() tea.Msg { return ComposeActiveMsg{Active: false} })

			if m.launcher != nil {
				cmds = append(cmds, m.launcher.LaunchSequence(names, "", msg.Parallel, msg.Stages))
			}
			return m, tea.Batch(cmds...), true
		}
		return m, nil, true

	case ComposeSequenceChangedMsg:
		if m.composeList != nil {
			m.composeList.validation = msg.Validation
		}
		if m.composeDetail != nil {
			var cmd tea.Cmd
			*m.composeDetail, cmd = m.composeDetail.Update(msg)
			return m, cmd, true
		}
		return m, nil, true
	}
	return m, nil, false
}

// handlePipelineKeyMsg handles pipeline-view-specific key bindings (Enter, Escape,
// 'c' cancel, 's' compose entry). Returns (model, cmd, handled). handled=true
// means the key was consumed by pipeline view handling.
func (m ContentModel) handlePipelineKeyMsg(msg tea.KeyMsg) (ContentModel, tea.Cmd, bool) {
	// Pipeline view Enter handling (skip when composing — compose list handles its own Enter)
	if msg.Type == tea.KeyEnter && m.focus == FocusPaneLeft && !m.list.filtering && m.currentView == ViewPipelines && !m.composing {
		if m.cursorOnFocusableItem() {
			item := m.list.navigable[m.list.cursor]
			m.focus = FocusPaneRight
			m.list.SetFocused(false)
			m.detail.SetFocused(true)

			enterCmds := []tea.Cmd{
				func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneRight} },
			}

			// For pipeline name nodes, send ConfigureFormMsg to show the launch form
			if item.kind == itemKindPipelineName {
				idx := m.list.availableIndexForName(item.pipelineName)
				if idx >= 0 {
					a := m.list.available[idx]
					enterCmds = append(enterCmds, func() tea.Msg {
						return ConfigureFormMsg{PipelineName: a.Name, InputExample: a.InputExample}
					})
					enterCmds = append(enterCmds, func() tea.Msg {
						return FormActiveMsg{Active: true}
					})
				}
			}

			// For running items, load historical events from SQLite and start polling
			if item.kind == itemKindRunning && item.dataIndex >= 0 && item.dataIndex < len(m.list.running) {
				r := m.list.running[item.dataIndex]
				buf := NewEventBuffer(1000)
				liveModel := NewLiveOutputModel(r.RunID, r.Name, buf, r.StartedAt, 0)
				liveModel.input = r.Input
				var maxID int64
				if m.launcher != nil && m.launcher.deps.Store != nil {
					events, err := m.launcher.deps.Store.GetEvents(r.RunID, state.EventQueryOptions{})
					if err == nil {
						for _, ev := range events {
							liveModel.storedRecords = append(liveModel.storedRecords, ev)
							liveModel.updateDashStepFromRecord(ev)
							liveModel.updateStepTrackingFromRecord(ev)
							if shouldFormatRecord(ev, liveModel.flags) {
								buf.Append(formatStoredEvent(ev))
							}
							if ev.ID > maxID {
								maxID = ev.ID
							}
						}
					}
				}
				liveModel.tailingPersisted = true
				liveModel.SetSize(m.detail.width, m.detail.height)
				m.detail.liveOutput = &liveModel
				m.detail.paneState = stateRunningLive
				m.detachedPollRunID = r.RunID
				m.detachedPollAfterID = maxID
				capturedRunID := r.RunID
				enterCmds = append(enterCmds, func() tea.Msg {
					return LiveOutputActiveMsg{Active: true}
				})
				// Check if already completed before starting poll
				if m.launcher != nil && m.launcher.deps.Store != nil {
					if run, runErr := m.launcher.deps.Store.GetRun(r.RunID); runErr == nil && run != nil {
						if run.Status == "completed" || run.Status == "failed" || run.Status == "cancelled" {
							liveModel.completed = true
							liveModel.tailingPersisted = false
							elapsed := time.Since(liveModel.startedAt)
							var summaryLine string
							switch {
							case noColor():
								summaryLine = fmt.Sprintf("Pipeline %s in %s", run.Status, formatElapsed(elapsed))
							case run.Status == "completed":
								summaryLine = fmt.Sprintf("\u2713 Pipeline completed in %s", formatElapsed(elapsed))
							default:
								summaryLine = fmt.Sprintf("\u2717 Pipeline %s in %s", run.Status, formatElapsed(elapsed))
							}
							buf.Append(summaryLine)
							liveModel.updateViewportContent()
						}
					}
				}
				if !liveModel.completed {
					enterCmds = append(enterCmds, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
						return DetachedEventPollTickMsg{RunID: capturedRunID}
					}))
					enterCmds = append(enterCmds, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
						return DashboardTickMsg{}
					}))
				}
			}

			// For finished items, activate finished detail hints
			if item.kind == itemKindFinished {
				enterCmds = append(enterCmds, func() tea.Msg {
					return FinishedDetailActiveMsg{Active: true}
				})
			}

			return m, tea.Batch(enterCmds...), true
		}
		// Section header or non-focusable — forward to list for collapse/no-op
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd, true
	}

	// Pipeline view Escape handling
	if msg.Type == tea.KeyEscape && m.focus == FocusPaneRight && m.currentView == ViewPipelines {
		// Clear form if it was active (content intercepts Escape before the form sees it)
		if m.detail.paneState == stateConfiguring {
			m.detail.launchForm = nil
			m.detail.paneState = stateAvailableDetail
			m.detail.updateViewportContent()
		}
		// Stop detached event polling
		m.detachedPollRunID = ""
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
		return m, tea.Batch(
			func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} },
			func() tea.Msg { return FormActiveMsg{Active: false} },
			func() tea.Msg { return LiveOutputActiveMsg{Active: false} },
			func() tea.Msg { return FinishedDetailActiveMsg{Active: false} },
			func() tea.Msg { return RunningInfoActiveMsg{Active: false} },
		), true
	}

	// Cancel/dismiss running pipeline with 'c' key — pipeline view, both panes
	if msg.String() == "c" && m.launcher != nil && m.currentView == ViewPipelines {
		var cancelRunID string
		if m.focus == FocusPaneRight && (m.detail.paneState == stateRunningLive || m.detail.paneState == stateRunningInfo) && m.detail.selectedRunID != "" {
			cancelRunID = m.detail.selectedRunID
		} else if m.focus == FocusPaneLeft {
			if len(m.list.navigable) > 0 && m.list.cursor < len(m.list.navigable) {
				item := m.list.navigable[m.list.cursor]
				if item.kind == itemKindRunning && item.dataIndex >= 0 && item.dataIndex < len(m.list.running) {
					cancelRunID = m.list.running[item.dataIndex].RunID
				}
			}
		}
		if cancelRunID != "" {
			m.launcher.Cancel(cancelRunID)
			return m, m.list.fetchPipelineData, true
		}
		return m, nil, true
	}

	// Enter compose mode with 's' key — only for pipeline name nodes (available pipelines)
	if msg.String() == "s" && m.currentView == ViewPipelines && m.focus == FocusPaneLeft && !m.list.filtering && !m.composing {
		if len(m.list.navigable) > 0 && m.list.cursor < len(m.list.navigable) {
			item := m.list.navigable[m.list.cursor]
			idx := m.list.availableIndexForName(item.pipelineName)
			if item.kind == itemKindPipelineName && idx >= 0 {
				selectedPipeline := m.list.available[idx]
				loadedPipeline, err := LoadPipelineByName(m.launcher.deps.PipelinesDir, selectedPipeline.Name)
				if err == nil {
					cl := NewComposeListModel(selectedPipeline, loadedPipeline, m.list.available)
					cd := NewComposeDetailModel()
					m.composing = true
					m.composeList = &cl
					m.composeDetail = &cd

					leftWidth := m.leftPaneWidth()
					rightWidth := m.width - leftWidth - 3
					m.composeList.SetSize(leftWidth, m.childHeight())
					m.composeList.SetFocused(true)
					m.composeDetail.SetSize(rightWidth, m.childHeight())

					seq := cl.sequence
					val := cl.validation
					return m, tea.Batch(
						func() tea.Msg { return ComposeActiveMsg{Active: true} },
						func() tea.Msg {
							return ComposeSequenceChangedMsg{
								Sequence:   seq,
								Validation: val,
							}
						},
					), true
				}
			}
		}
		return m, nil, true
	}

	return m, nil, false
}

// viewPipeline renders the pipelines panel (or the compose panel when active).
func (m ContentModel) viewPipeline() (left, right string) {
	if m.composing && m.composeList != nil && m.composeDetail != nil {
		left = m.composeList.View()
		right = m.composeDetail.View()
	} else {
		left = m.list.View()
		right = m.detail.View()
	}
	return left, right
}
