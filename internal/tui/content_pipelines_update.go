package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/state"
)

// updatePipelineMessage routes pipeline-specific messages (selection, data, events,
// launch lifecycle, polling, focus). Returns (model, cmd, handled). handled=true
// means the message was a pipeline message.
func (m ContentModel) updatePipelineMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case PipelineSelectedMsg:
		var listCmd, detailCmd tea.Cmd
		m.list, listCmd = m.list.Update(msg)
		// When in Issues view, show pipeline detail in right pane — but only
		// when the selection came from the issue list (user interaction), not
		// from the pipeline list's periodic data refresh.
		if m.currentView == ViewIssues && msg.FromIssueList {
			m.issueShowPipeline = true
		}
		// Wire live output from SQLite events for running pipelines on hover
		if msg.Kind == itemKindRunning && msg.RunID != "" && m.launcher != nil {
			if m.detail.liveOutput == nil || m.detail.liveOutput.runID != msg.RunID {
				var startedAt time.Time
				// Check both pipeline list and issue list for the running entry.
				for _, r := range m.list.running {
					if r.RunID == msg.RunID {
						startedAt = r.StartedAt
						break
					}
				}
				if startedAt.IsZero() && m.issueList != nil {
					for _, r := range m.issueList.running {
						if r.RunID == msg.RunID {
							startedAt = r.StartedAt
							break
						}
					}
				}
				buf := NewEventBuffer(1000)
				liveModel := NewLiveOutputModel(msg.RunID, msg.Name, buf, startedAt, 0)
				liveModel.input = msg.Input
				var maxID int64
				if m.launcher.deps.Store != nil {
					events, err := m.launcher.deps.Store.GetEvents(msg.RunID, state.EventQueryOptions{})
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
				m.detachedPollRunID = msg.RunID
				m.detachedPollAfterID = maxID
				// Check if already completed before starting poll
				alreadyCompleted := false
				if m.launcher.deps.Store != nil {
					if run, runErr := m.launcher.deps.Store.GetRun(msg.RunID); runErr == nil && run != nil {
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
							alreadyCompleted = true
						}
					}
				}
				capturedRunID := msg.RunID
				cmds = append(cmds, func() tea.Msg {
					return LiveOutputActiveMsg{Active: true}
				})
				if !alreadyCompleted {
					cmds = append(cmds, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
						return DetachedEventPollTickMsg{RunID: capturedRunID}
					}))
				}
			}
		}
		m.detail, detailCmd = m.detail.Update(msg)
		if listCmd != nil {
			cmds = append(cmds, listCmd)
		}
		if detailCmd != nil {
			cmds = append(cmds, detailCmd)
		}
		return m, tea.Batch(cmds...), true

	case DetailDataMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd, true

	case PipelineRefreshTickMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd, true

	case PipelineDataMsg:
		// Detached pipelines are tracked via SQLite — no in-memory merge needed.
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		// Also update issue list with pipeline data so it can show children.
		if m.issueList != nil {
			var issueCmd tea.Cmd
			*m.issueList, issueCmd = m.issueList.Update(msg)
			return m, tea.Batch(cmd, issueCmd), true
		}
		return m, cmd, true

	case PipelineEventMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd, true

	case DashboardTickMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd, true

	case TransitionTimerMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd, true

	case ChatSessionEndedMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd, true

	case BranchCheckoutMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd, true

	case DiffViewEndedMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd, true

	case ElapsedTickMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		// Also forward to issue list for running pipeline elapsed time updates.
		if m.issueList != nil {
			var issueCmd tea.Cmd
			*m.issueList, issueCmd = m.issueList.Update(msg)
			return m, tea.Batch(cmd, issueCmd), true
		}
		return m, cmd, true

	case ConfigureFormMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd, true

	case LaunchRequestMsg:
		if m.launcher != nil {
			// Show "Starting pipeline..." while the async launch runs.
			// Without this, compose-launched pipelines briefly flash stale
			// detail content before PipelineLaunchedMsg arrives.
			m.detail.paneState = stateLaunching
			m.detail.selectedName = msg.Config.PipelineName
			cmd := m.launcher.Launch(msg.Config)
			return m, cmd, true
		}
		return m, nil, true

	case PipelineLaunchedMsg:
		// Forward to list for running entry insertion
		var listCmd tea.Cmd
		m.list, listCmd = m.list.Update(msg)

		// Create live output model with display flags from launch config
		buf := NewEventBuffer(1000)
		live := NewLiveOutputModel(msg.RunID, msg.PipelineName, buf, time.Now(), 0)
		live.input = msg.Input
		if msg.Verbose {
			live.flags.Verbose = true
		}
		if msg.Debug {
			live.flags.Debug = true
		}

		// Load existing events from SQLite (will be empty for just-launched pipeline)
		var maxID int64
		if m.launcher != nil && m.launcher.deps.Store != nil {
			events, err := m.launcher.deps.Store.GetEvents(msg.RunID, state.EventQueryOptions{})
			if err == nil {
				for _, ev := range events {
					live.storedRecords = append(live.storedRecords, ev)
					live.updateDashStepFromRecord(ev)
					live.updateStepTrackingFromRecord(ev)
					if shouldFormatRecord(ev, live.flags) {
						buf.Append(formatStoredEvent(ev))
					}
					if ev.ID > maxID {
						maxID = ev.ID
					}
				}
			}
		}
		live.SetSize(m.detail.width, m.detail.height)
		m.detail.liveOutput = &live
		m.detail.paneState = stateRunningLive
		m.detail.selectedRunID = msg.RunID
		m.detail.selectedName = msg.PipelineName
		m.detail.selectedKind = itemKindRunning
		m.detachedPollRunID = msg.RunID
		m.detachedPollAfterID = maxID

		// Check if already completed (guards against race conditions)
		alreadyCompleted := false
		if m.launcher != nil && m.launcher.deps.Store != nil {
			if run, runErr := m.launcher.deps.Store.GetRun(msg.RunID); runErr == nil && run != nil {
				if run.Status == "completed" || run.Status == "failed" || run.Status == "cancelled" {
					live.completed = true
					live.tailingPersisted = false
					elapsed := time.Since(live.startedAt)
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
					live.updateViewportContent()
					alreadyCompleted = true
				}
			}
		}

		// Switch focus to right pane for live output
		m.focus = FocusPaneRight
		m.list.SetFocused(false)
		m.detail.SetFocused(true)
		focusCmd := func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneRight} }
		formCmd := func() tea.Msg { return FormActiveMsg{Active: false} }
		liveCmd := func() tea.Msg { return LiveOutputActiveMsg{Active: true} }
		batchCmds := []tea.Cmd{focusCmd, formCmd, liveCmd}
		if !alreadyCompleted {
			capturedRunID := msg.RunID
			batchCmds = append(batchCmds, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return DetachedEventPollTickMsg{RunID: capturedRunID}
			}))
			batchCmds = append(batchCmds, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
				return DashboardTickMsg{}
			}))
		}
		if listCmd != nil {
			batchCmds = append(batchCmds, listCmd)
		}
		return m, tea.Batch(batchCmds...), true

	case PipelineLaunchResultMsg:
		if m.launcher != nil {
			m.launcher.Cleanup(msg.RunID)
		}
		// Remove synthetic running entry so it doesn't ghost after completion
		var newRunning []RunningPipeline
		for _, r := range m.list.running {
			if r.RunID != msg.RunID {
				newRunning = append(newRunning, r)
			}
		}
		m.list.running = newRunning
		m.list.buildNavigableItems()
		// If the pipeline failed and the detail pane still shows live output,
		// let the live output display the error (failure event was already sent).
		// If no live output is active, show the error in the detail pane directly.
		if msg.Err != nil && (m.detail.liveOutput == nil || m.detail.liveOutput.runID != msg.RunID) {
			m.detail.launchError = msg.Err.Error()
			m.detail.launchErrorTitle = "Pipeline Failed"
			m.detail.paneState = stateError
			m.detail.updateViewportContent()
		}
		// Trigger data refresh so the pipeline appears in Finished
		return m, m.list.fetchPipelineData, true

	case LaunchErrorMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		// Transition focus to left pane
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
		focusCmd := func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneRight} }
		formCmd := func() tea.Msg { return FormActiveMsg{Active: false} }
		batchCmds := []tea.Cmd{focusCmd, formCmd}
		if cmd != nil {
			batchCmds = append(batchCmds, cmd)
		}
		return m, tea.Batch(batchCmds...), true

	case FocusChangedMsg:
		if msg.Pane == FocusPaneLeft {
			m.focus = FocusPaneLeft
			m.list.SetFocused(true)
			m.detail.SetFocused(false)
		}
		return m, nil, true

	case RunEventsMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd, true

	case DetachedEventPollTickMsg:
		// Stop polling if the run ID changed or we left the live output pane
		if m.detachedPollRunID != msg.RunID || m.detail.paneState != stateRunningLive {
			m.detachedPollRunID = ""
			return m, nil, true
		}
		// Fetch new events since last seen ID
		if m.launcher != nil && m.launcher.deps.Store != nil {
			events, err := m.launcher.deps.Store.GetEvents(msg.RunID, state.EventQueryOptions{
				Limit:   1000,
				AfterID: m.detachedPollAfterID,
			})
			if err == nil && len(events) > 0 {
				if m.detail.liveOutput != nil {
					for _, ev := range events {
						m.detail.liveOutput.storedRecords = append(m.detail.liveOutput.storedRecords, ev)
						m.detail.liveOutput.updateDashStepFromRecord(ev)
						m.detail.liveOutput.updateStepTrackingFromRecord(ev)
						if shouldFormatRecord(ev, m.detail.liveOutput.flags) {
							m.detail.liveOutput.buffer.Append(formatStoredEvent(ev))
						}
						if ev.ID > m.detachedPollAfterID {
							m.detachedPollAfterID = ev.ID
						}
					}
					m.detail.liveOutput.updateViewportContent()
					if m.detail.liveOutput.autoScroll {
						m.detail.liveOutput.viewport.GotoBottom()
					}
				}
			}
			// Check if the run has terminated — stop polling and transition
			if run, runErr := m.launcher.deps.Store.GetRun(msg.RunID); runErr == nil && run != nil {
				if run.Status == "completed" || run.Status == "failed" || run.Status == "cancelled" {
					if m.detail.liveOutput != nil {
						m.detail.liveOutput.completed = true
						m.detail.liveOutput.tailingPersisted = false
						elapsed := time.Since(m.detail.liveOutput.startedAt)
						var summaryLine string
						switch {
						case noColor():
							summaryLine = fmt.Sprintf("Pipeline %s in %s", run.Status, formatElapsed(elapsed))
						case run.Status == "completed":
							summaryLine = fmt.Sprintf("\u2713 Pipeline completed in %s", formatElapsed(elapsed))
						default:
							summaryLine = fmt.Sprintf("\u2717 Pipeline %s in %s", run.Status, formatElapsed(elapsed))
						}
						m.detail.liveOutput.buffer.Append(summaryLine)
						m.detail.liveOutput.updateViewportContent()
						if m.detail.liveOutput.autoScroll {
							m.detail.liveOutput.viewport.GotoBottom()
						}
					}
					m.detachedPollRunID = ""
					return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
						return TransitionTimerMsg(msg)
					}), true
				}
				// Detect stale process: PID set but process no longer alive
				if run.PID > 0 && !IsProcessAlive(run.PID) {
					_ = m.launcher.deps.Store.UpdateRunStatus(msg.RunID, "failed", "executor process no longer running", 0)
					if m.detail.liveOutput != nil {
						m.detail.liveOutput.completed = true
						m.detail.liveOutput.tailingPersisted = false
						elapsed := time.Since(m.detail.liveOutput.startedAt)
						var summaryLine string
						if noColor() {
							summaryLine = fmt.Sprintf("Pipeline failed in %s (executor process died)", formatElapsed(elapsed))
						} else {
							summaryLine = fmt.Sprintf("\u2717 Pipeline failed in %s (executor process died)", formatElapsed(elapsed))
						}
						m.detail.liveOutput.buffer.Append(summaryLine)
						m.detail.liveOutput.updateViewportContent()
						if m.detail.liveOutput.autoScroll {
							m.detail.liveOutput.viewport.GotoBottom()
						}
					}
					m.detachedPollRunID = ""
					return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
						return TransitionTimerMsg(msg)
					}), true
				}
			}
		}
		capturedRunID := msg.RunID
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return DetachedEventPollTickMsg{RunID: capturedRunID}
		}), true
	}
	return m, nil, false
}
