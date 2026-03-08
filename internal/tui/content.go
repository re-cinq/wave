package tui

import (
	"fmt"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/state"
)

// ContentProviders holds data providers for alternative views.
type ContentProviders struct {
	PersonaProvider  PersonaDataProvider
	ContractProvider ContractDataProvider
	SkillProvider    SkillDataProvider
	HealthProvider   HealthDataProvider
}

// ContentModel is the main content area component composing a left pipeline list pane and a right detail pane.
type ContentModel struct {
	width    int
	height   int
	list     PipelineListModel
	detail   PipelineDetailModel
	focus    FocusPane
	launcher *PipelineLauncher

	// View switching
	currentView ViewType

	// Lazy-initialized alternative view models (nil until first access)
	personaList    *PersonaListModel
	personaDetail  *PersonaDetailModel
	contractList   *ContractListModel
	contractDetail *ContractDetailModel
	skillList      *SkillListModel
	skillDetail    *SkillDetailModel
	healthList     *HealthListModel
	healthDetail   *HealthDetailModel

	// Data providers for alternative views
	personaProvider  PersonaDataProvider
	contractProvider ContractDataProvider
	skillProvider    SkillDataProvider
	healthProvider   HealthDataProvider

	// Compose mode (nil when inactive)
	composing     bool
	composeList   *ComposeListModel
	composeDetail *ComposeDetailModel

	// Detached pipeline event polling
	detachedPollRunID  string // Run ID currently being polled for events
	detachedPollOffset int    // Offset for fetching new events
}

// NewContentModel creates a new content model with the given pipeline data providers.
func NewContentModel(provider PipelineDataProvider, detailProvider DetailDataProvider, deps LaunchDependencies, providers ...ContentProviders) ContentModel {
	var launcher *PipelineLauncher
	if deps.Manifest != nil {
		launcher = NewPipelineLauncher(deps)
	}

	m := ContentModel{
		list:        NewPipelineListModel(provider),
		detail:      NewPipelineDetailModel(detailProvider),
		focus:       FocusPaneLeft,
		launcher:    launcher,
		currentView: ViewPipelines,
	}

	if len(providers) > 0 {
		p := providers[0]
		m.personaProvider = p.PersonaProvider
		m.contractProvider = p.ContractProvider
		m.skillProvider = p.SkillProvider
		m.healthProvider = p.HealthProvider
	}

	return m
}

// IsFiltering returns true if the active view's list is in filter mode.
func (m ContentModel) IsFiltering() bool {
	switch m.currentView {
	case ViewPipelines:
		return m.list.filtering
	case ViewPersonas:
		return m.personaList != nil && m.personaList.filtering
	case ViewContracts:
		return m.contractList != nil && m.contractList.filtering
	case ViewSkills:
		return m.skillList != nil && m.skillList.filtering
	}
	return false
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
	rightWidth := w - leftWidth

	m.list.SetSize(leftWidth, h)
	m.detail.SetSize(rightWidth, h)

	// Propagate to non-nil alternative view models
	if m.personaList != nil {
		m.personaList.SetSize(leftWidth, h)
	}
	if m.personaDetail != nil {
		m.personaDetail.SetSize(rightWidth, h)
	}
	if m.contractList != nil {
		m.contractList.SetSize(leftWidth, h)
	}
	if m.contractDetail != nil {
		m.contractDetail.SetSize(rightWidth, h)
	}
	if m.skillList != nil {
		m.skillList.SetSize(leftWidth, h)
	}
	if m.skillDetail != nil {
		m.skillDetail.SetSize(rightWidth, h)
	}
	if m.healthList != nil {
		m.healthList.SetSize(leftWidth, h)
	}
	if m.healthDetail != nil {
		m.healthDetail.SetSize(rightWidth, h)
	}
	if m.composeList != nil {
		m.composeList.SetSize(leftWidth, h)
	}
	if m.composeDetail != nil {
		m.composeDetail.SetSize(rightWidth, h)
	}
}

// cycleView moves to the next view and returns init commands if the view was just created.
func (m *ContentModel) cycleView() tea.Cmd {
	m.currentView = (m.currentView + 1) % 5
	m.focus = FocusPaneLeft

	var initCmd tea.Cmd

	leftWidth := m.leftPaneWidth()
	rightWidth := m.width - leftWidth

	switch m.currentView {
	case ViewPipelines:
		m.list.SetFocused(true)
		m.detail.SetFocused(false)

	case ViewPersonas:
		if m.personaList == nil && m.personaProvider != nil {
			pl := NewPersonaListModel(m.personaProvider)
			pl.SetSize(leftWidth, m.height)
			m.personaList = &pl
			pd := NewPersonaDetailModel(m.personaProvider)
			pd.SetSize(rightWidth, m.height)
			m.personaDetail = &pd
			initCmd = m.personaList.Init()
		}
		if m.personaList != nil {
			m.personaList.SetFocused(true)
		}
		if m.personaDetail != nil {
			m.personaDetail.SetFocused(false)
		}

	case ViewContracts:
		if m.contractList == nil && m.contractProvider != nil {
			cl := NewContractListModel(m.contractProvider)
			cl.SetSize(leftWidth, m.height)
			m.contractList = &cl
			cd := NewContractDetailModel()
			cd.SetSize(rightWidth, m.height)
			m.contractDetail = &cd
			initCmd = m.contractList.Init()
		}
		if m.contractList != nil {
			m.contractList.SetFocused(true)
		}
		if m.contractDetail != nil {
			m.contractDetail.SetFocused(false)
		}

	case ViewSkills:
		if m.skillList == nil && m.skillProvider != nil {
			sl := NewSkillListModel(m.skillProvider)
			sl.SetSize(leftWidth, m.height)
			m.skillList = &sl
			sd := NewSkillDetailModel()
			sd.SetSize(rightWidth, m.height)
			m.skillDetail = &sd
			initCmd = m.skillList.Init()
		}
		if m.skillList != nil {
			m.skillList.SetFocused(true)
		}
		if m.skillDetail != nil {
			m.skillDetail.SetFocused(false)
		}

	case ViewHealth:
		if m.healthList == nil && m.healthProvider != nil {
			hl := NewHealthListModel(m.healthProvider)
			hl.SetSize(leftWidth, m.height)
			m.healthList = &hl
			hd := NewHealthDetailModel()
			hd.SetSize(rightWidth, m.height)
			m.healthDetail = &hd
			initCmd = m.healthList.Init()
		}
		if m.healthList != nil {
			m.healthList.SetFocused(true)
		}
		if m.healthDetail != nil {
			m.healthDetail.SetFocused(false)
		}
	}

	batchCmds := []tea.Cmd{
		func() tea.Msg { return ViewChangedMsg{View: m.currentView} },
		func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} },
	}
	if initCmd != nil {
		batchCmds = append(batchCmds, initCmd)
	}

	return tea.Batch(batchCmds...)
}

// Update handles messages by forwarding to child components with focus-aware routing.
func (m ContentModel) Update(msg tea.Msg) (ContentModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Intercept Tab for view cycling BEFORE focus-based child routing
		if msg.Type == tea.KeyTab {
			// Block Tab cycling during compose mode
			if m.composing {
				return m, nil
			}
			// Only forward Tab to form if pipeline detail is in stateConfiguring
			if m.currentView == ViewPipelines && m.detail.paneState == stateConfiguring {
				var cmd tea.Cmd
				m.detail, cmd = m.detail.Update(msg)
				return m, cmd
			}
			// Otherwise, cycle view
			cmd := m.cycleView()
			return m, cmd
		}

		// Handle Enter for alternative views — focus right pane
		if msg.Type == tea.KeyEnter && m.focus == FocusPaneLeft && m.currentView != ViewPipelines {
			return m.handleAlternativeViewEnter()
		}

		// Handle Escape for alternative views — return to left pane
		if msg.Type == tea.KeyEscape && m.focus == FocusPaneRight && m.currentView != ViewPipelines {
			return m.handleAlternativeViewEscape()
		}

		// Handle '/' filter for alternative views
		if msg.String() == "/" && m.focus == FocusPaneLeft && m.currentView != ViewPipelines && m.currentView != ViewHealth {
			return m.routeToActiveList(msg)
		}

		// Handle 'r' for health view recheck
		if msg.String() == "r" && m.currentView == ViewHealth && m.focus == FocusPaneLeft {
			return m.routeToActiveList(msg)
		}

		// Pipeline view Enter handling
		if msg.Type == tea.KeyEnter && m.focus == FocusPaneLeft && !m.list.filtering && m.currentView == ViewPipelines {
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

				// For running items, load historical events from store and activate live output
				if item.kind == itemKindRunning && item.dataIndex >= 0 && item.dataIndex < len(m.list.running) {
					r := m.list.running[item.dataIndex]
					buf := NewEventBuffer(1000)
					// Load historical events from SQLite
					var eventCount int
					if m.launcher != nil && m.launcher.deps.Store != nil {
						events, err := m.launcher.deps.Store.GetEvents(r.RunID, state.EventQueryOptions{})
						if err == nil {
							eventCount = len(events)
							for _, ev := range events {
								buf.Append(formatStoredEvent(ev))
							}
						}
					}
					liveModel := NewLiveOutputModel(r.RunID, r.Name, buf, r.StartedAt, 0)
					liveModel.SetSize(m.detail.width, m.detail.height)
					m.detail.liveOutput = &liveModel
					m.detail.paneState = stateRunningLive
					m.detachedPollRunID = r.RunID
					m.detachedPollOffset = eventCount
					enterCmds = append(enterCmds, func() tea.Msg {
						return LiveOutputActiveMsg{Active: true}
					})
					// Start event polling for detached pipeline
					capturedRunID := r.RunID
					enterCmds = append(enterCmds, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
						return DetachedEventPollTickMsg{RunID: capturedRunID}
					}))
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

		// Pipeline view Escape handling
		if msg.Type == tea.KeyEscape && m.focus == FocusPaneRight && m.currentView == ViewPipelines {
			m.detachedPollRunID = "" // Stop event polling
			m.focus = FocusPaneLeft
			m.list.SetFocused(true)
			m.detail.SetFocused(false)
			return m, tea.Batch(
				func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} },
				func() tea.Msg { return LiveOutputActiveMsg{Active: false} },
				func() tea.Msg { return FinishedDetailActiveMsg{Active: false} },
			)
		}

		// Cancel running pipeline with 'c' key — only in pipeline view
		if msg.String() == "c" && m.focus == FocusPaneLeft && m.launcher != nil && m.currentView == ViewPipelines {
			if len(m.list.navigable) > 0 && m.list.cursor < len(m.list.navigable) {
				item := m.list.navigable[m.list.cursor]
				if item.kind == itemKindRunning && item.dataIndex >= 0 && item.dataIndex < len(m.list.running) {
					r := m.list.running[item.dataIndex]
					m.launcher.Cancel(r.RunID)
					// Start 30s force-kill timer for detached pipelines (SC-003)
					if r.Detached && r.PID > 0 {
						capturedRunID := r.RunID
						capturedPID := r.PID
						return m, tea.Tick(30*time.Second, func(time.Time) tea.Msg {
							return CancelForceKillMsg{RunID: capturedRunID, PID: capturedPID}
						})
					}
				}
			}
			return m, nil
		}

		// Enter compose mode with 's' key — only for available pipelines
		if msg.String() == "s" && m.currentView == ViewPipelines && m.focus == FocusPaneLeft && !m.list.filtering && !m.composing {
			if len(m.list.navigable) > 0 && m.list.cursor < len(m.list.navigable) {
				item := m.list.navigable[m.list.cursor]
				if item.kind == itemKindAvailable && item.dataIndex >= 0 && item.dataIndex < len(m.list.available) {
					selectedPipeline := m.list.available[item.dataIndex]
					loadedPipeline, err := LoadPipelineByName(m.launcher.deps.PipelinesDir, selectedPipeline.Name)
					if err == nil {
						cl := NewComposeListModel(selectedPipeline, loadedPipeline, m.list.available)
						cd := NewComposeDetailModel()
						m.composing = true
						m.composeList = &cl
						m.composeDetail = &cd

						leftWidth := m.leftPaneWidth()
						rightWidth := m.width - leftWidth
						m.composeList.SetSize(leftWidth, m.height)
						m.composeDetail.SetSize(rightWidth, m.height)

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
						)
					}
				}
			}
			return m, nil
		}

		// When composing, route keys to compose models
		if m.composing {
			if m.focus == FocusPaneLeft && m.composeList != nil {
				var cmd tea.Cmd
				*m.composeList, cmd = m.composeList.Update(msg)
				return m, cmd
			}
			if m.focus == FocusPaneRight && m.composeDetail != nil {
				var cmd tea.Cmd
				*m.composeDetail, cmd = m.composeDetail.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		// Route key messages to the focused child
		if m.focus == FocusPaneRight {
			return m.routeToActiveDetail(msg)
		}

		// Route to active list (left pane)
		return m.routeToActiveList(msg)

	// Compose mode messages
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
		)

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
			)
		}
		// T031: Multi-pipeline sequence — show informational message in the
		// compose detail pane. Keep compose mode active so the user can read
		// the message and press Esc to exit.
		if m.composeDetail != nil {
			infoMsg := "Sequential pipeline execution requires cross-pipeline " +
				"artifact handoff (#249). Build and validate your sequence now " +
				"— execution will be enabled in a future release."
			m.composeDetail.viewport.SetContent(infoMsg)
			m.composeDetail.viewport.GotoTop()
		}
		return m, nil

	case ComposeSequenceChangedMsg:
		if m.composeList != nil {
			m.composeList.validation = msg.Validation
		}
		if m.composeDetail != nil {
			var cmd tea.Cmd
			*m.composeDetail, cmd = m.composeDetail.Update(msg)
			return m, cmd
		}
		return m, nil

	case ComposeFocusDetailMsg:
		m.focus = FocusPaneRight
		if m.composeList != nil {
			m.composeList.SetFocused(false)
		}
		if m.composeDetail != nil {
			m.composeDetail.SetFocused(true)
		}
		return m, func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneRight} }

	// Route alternative view messages
	case PersonaDataMsg:
		if m.personaList != nil {
			var cmd tea.Cmd
			*m.personaList, cmd = m.personaList.Update(msg)
			return m, cmd
		}
		return m, nil

	case PersonaSelectedMsg:
		if m.personaList != nil {
			var listCmd tea.Cmd
			*m.personaList, listCmd = m.personaList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.personaDetail != nil {
			// Find the matching persona and set it on the detail model
			if m.personaList != nil {
				for i := range m.personaList.navigable {
					if m.personaList.navigable[i].Name == msg.Name {
						m.personaDetail.SetPersona(&m.personaList.navigable[i])
						break
					}
				}
			}
			var detailCmd tea.Cmd
			*m.personaDetail, detailCmd = m.personaDetail.Update(msg)
			if detailCmd != nil {
				cmds = append(cmds, detailCmd)
			}
		}
		return m, tea.Batch(cmds...)

	case PersonaStatsMsg:
		if m.personaDetail != nil {
			var cmd tea.Cmd
			*m.personaDetail, cmd = m.personaDetail.Update(msg)
			return m, cmd
		}
		return m, nil

	case ContractDataMsg:
		if m.contractList != nil {
			var cmd tea.Cmd
			*m.contractList, cmd = m.contractList.Update(msg)
			return m, cmd
		}
		return m, nil

	case ContractSelectedMsg:
		if m.contractList != nil {
			var listCmd tea.Cmd
			*m.contractList, listCmd = m.contractList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.contractDetail != nil && m.contractList != nil {
			for i := range m.contractList.navigable {
				if m.contractList.navigable[i].Label == msg.Label {
					m.contractDetail.SetContract(&m.contractList.navigable[i])
					break
				}
			}
		}
		return m, tea.Batch(cmds...)

	case SkillDataMsg:
		if m.skillList != nil {
			var cmd tea.Cmd
			*m.skillList, cmd = m.skillList.Update(msg)
			return m, cmd
		}
		return m, nil

	case SkillSelectedMsg:
		if m.skillList != nil {
			var listCmd tea.Cmd
			*m.skillList, listCmd = m.skillList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.skillDetail != nil && m.skillList != nil {
			for i := range m.skillList.navigable {
				if m.skillList.navigable[i].Name == msg.Name {
					m.skillDetail.SetSkill(&m.skillList.navigable[i])
					break
				}
			}
		}
		return m, tea.Batch(cmds...)

	case HealthCheckResultMsg:
		if m.healthList != nil {
			var listCmd tea.Cmd
			*m.healthList, listCmd = m.healthList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.healthDetail != nil {
			var detailCmd tea.Cmd
			*m.healthDetail, detailCmd = m.healthDetail.Update(msg)
			if detailCmd != nil {
				cmds = append(cmds, detailCmd)
			}
		}
		return m, tea.Batch(cmds...)

	case HealthSelectedMsg:
		if m.healthList != nil && msg.Index < len(m.healthList.checks) {
			check := m.healthList.checks[msg.Index]
			if m.healthDetail != nil {
				m.healthDetail.SetCheck(&check)
			}
		}
		return m, nil

	// Pipeline-specific messages — always route to pipeline models
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

	case DetachedEventPollTickMsg:
		// Poll for new events from detached pipeline
		if m.detachedPollRunID != msg.RunID || m.detail.paneState != stateRunningLive {
			// Stop polling — user navigated away or run changed
			m.detachedPollRunID = ""
			return m, nil
		}
		if m.launcher != nil && m.launcher.deps.Store != nil {
			events, err := m.launcher.deps.Store.GetEvents(msg.RunID, state.EventQueryOptions{
				Offset: m.detachedPollOffset,
			})
			if err == nil && len(events) > 0 {
				m.detachedPollOffset += len(events)
				if m.detail.liveOutput != nil {
					for _, ev := range events {
						m.detail.liveOutput.buffer.Append(formatStoredEvent(ev))
					}
					m.detail.liveOutput.updateViewportContent()
					if m.detail.liveOutput.autoScroll {
						m.detail.liveOutput.viewport.GotoBottom()
					}
				}
			}
		}
		// Schedule next poll
		capturedRunID := msg.RunID
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return DetachedEventPollTickMsg{RunID: capturedRunID}
		})

	case PipelineDataMsg, PipelineRefreshTickMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case PipelineEventMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case CancelForceKillMsg:
		// Force-kill escalation after 30s cancellation grace period (SC-003)
		if msg.PID > 0 && IsProcessAlive(msg.PID) {
			// Kill the entire process group
			_ = syscall.Kill(-msg.PID, syscall.SIGKILL)
			// Update run status to failed
			if m.launcher != nil && m.launcher.deps.Store != nil {
				_ = m.launcher.deps.Store.UpdateRunStatus(msg.RunID, "failed",
					"cancellation timeout — force killed", 0)
			}
		}
		// Refresh the list to reflect the status change
		return m, m.list.fetchPipelineData

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

	// Default: forward to both pipeline children
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

// handleAlternativeViewEnter handles Enter key in alternative view left panes.
func (m ContentModel) handleAlternativeViewEnter() (ContentModel, tea.Cmd) {
	m.focus = FocusPaneRight

	switch m.currentView {
	case ViewPersonas:
		if m.personaList != nil {
			m.personaList.SetFocused(false)
		}
		if m.personaDetail != nil {
			m.personaDetail.SetFocused(true)
		}
	case ViewContracts:
		if m.contractList != nil {
			m.contractList.SetFocused(false)
		}
		if m.contractDetail != nil {
			m.contractDetail.SetFocused(true)
		}
	case ViewSkills:
		if m.skillList != nil {
			m.skillList.SetFocused(false)
		}
		if m.skillDetail != nil {
			m.skillDetail.SetFocused(true)
		}
	case ViewHealth:
		if m.healthList != nil {
			m.healthList.SetFocused(false)
		}
		if m.healthDetail != nil {
			m.healthDetail.SetFocused(true)
		}
	}

	return m, func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneRight} }
}

// handleAlternativeViewEscape handles Escape key in alternative view right panes.
func (m ContentModel) handleAlternativeViewEscape() (ContentModel, tea.Cmd) {
	m.focus = FocusPaneLeft

	switch m.currentView {
	case ViewPersonas:
		if m.personaList != nil {
			m.personaList.SetFocused(true)
		}
		if m.personaDetail != nil {
			m.personaDetail.SetFocused(false)
		}
	case ViewContracts:
		if m.contractList != nil {
			m.contractList.SetFocused(true)
		}
		if m.contractDetail != nil {
			m.contractDetail.SetFocused(false)
		}
	case ViewSkills:
		if m.skillList != nil {
			m.skillList.SetFocused(true)
		}
		if m.skillDetail != nil {
			m.skillDetail.SetFocused(false)
		}
	case ViewHealth:
		if m.healthList != nil {
			m.healthList.SetFocused(true)
		}
		if m.healthDetail != nil {
			m.healthDetail.SetFocused(false)
		}
	}

	return m, func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} }
}

// routeToActiveList routes a key message to the active view's list model.
func (m ContentModel) routeToActiveList(msg tea.Msg) (ContentModel, tea.Cmd) {
	switch m.currentView {
	case ViewPipelines:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case ViewPersonas:
		if m.personaList != nil {
			var cmd tea.Cmd
			*m.personaList, cmd = m.personaList.Update(msg)
			return m, cmd
		}
	case ViewContracts:
		if m.contractList != nil {
			var cmd tea.Cmd
			*m.contractList, cmd = m.contractList.Update(msg)
			return m, cmd
		}
	case ViewSkills:
		if m.skillList != nil {
			var cmd tea.Cmd
			*m.skillList, cmd = m.skillList.Update(msg)
			return m, cmd
		}
	case ViewHealth:
		if m.healthList != nil {
			var cmd tea.Cmd
			*m.healthList, cmd = m.healthList.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// routeToActiveDetail routes a key message to the active view's detail model.
func (m ContentModel) routeToActiveDetail(msg tea.Msg) (ContentModel, tea.Cmd) {
	switch m.currentView {
	case ViewPipelines:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	case ViewPersonas:
		if m.personaDetail != nil {
			var cmd tea.Cmd
			*m.personaDetail, cmd = m.personaDetail.Update(msg)
			return m, cmd
		}
	case ViewContracts:
		if m.contractDetail != nil {
			var cmd tea.Cmd
			*m.contractDetail, cmd = m.contractDetail.Update(msg)
			return m, cmd
		}
	case ViewSkills:
		if m.skillDetail != nil {
			var cmd tea.Cmd
			*m.skillDetail, cmd = m.skillDetail.Update(msg)
			return m, cmd
		}
	case ViewHealth:
		if m.healthDetail != nil {
			var cmd tea.Cmd
			*m.healthDetail, cmd = m.healthDetail.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// cursorOnFocusableItem returns true if the cursor is on an available, finished, or running item.
func (m ContentModel) cursorOnFocusableItem() bool {
	if len(m.list.navigable) == 0 || m.list.cursor >= len(m.list.navigable) {
		return false
	}
	kind := m.list.navigable[m.list.cursor].kind
	return kind == itemKindAvailable || kind == itemKindFinished || kind == itemKindRunning
}

// View renders the content area with left list and right detail pane.
func (m ContentModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	var leftView, rightView string

	switch m.currentView {
	case ViewPipelines:
		if m.composing && m.composeList != nil && m.composeDetail != nil {
			leftView = m.composeList.View()
			rightView = m.composeDetail.View()
		} else {
			leftView = m.list.View()
			rightView = m.detail.View()
		}

	case ViewPersonas:
		if m.personaList != nil {
			leftView = m.personaList.View()
		} else {
			leftView = renderPlaceholder(m.leftPaneWidth(), m.height, "Select a persona to view details")
		}
		if m.personaDetail != nil {
			rightView = m.personaDetail.View()
		} else {
			rightView = renderPlaceholder(m.width-m.leftPaneWidth(), m.height, "Select a persona to view details")
		}

	case ViewContracts:
		if m.contractList != nil {
			leftView = m.contractList.View()
		} else {
			leftView = renderPlaceholder(m.leftPaneWidth(), m.height, "Select a contract to view details")
		}
		if m.contractDetail != nil {
			rightView = m.contractDetail.View()
		} else {
			rightView = renderPlaceholder(m.width-m.leftPaneWidth(), m.height, "Select a contract to view details")
		}

	case ViewSkills:
		if m.skillList != nil {
			leftView = m.skillList.View()
		} else {
			leftView = renderPlaceholder(m.leftPaneWidth(), m.height, "Select a skill to view details")
		}
		if m.skillDetail != nil {
			rightView = m.skillDetail.View()
		} else {
			rightView = renderPlaceholder(m.width-m.leftPaneWidth(), m.height, "Select a skill to view details")
		}

	case ViewHealth:
		if m.healthList != nil {
			leftView = m.healthList.View()
		} else {
			leftView = renderPlaceholder(m.leftPaneWidth(), m.height, "Select a health check to view details")
		}
		if m.healthDetail != nil {
			rightView = m.healthDetail.View()
		} else {
			rightView = renderPlaceholder(m.width-m.leftPaneWidth(), m.height, "Select a health check to view details")
		}
	}

	// Apply dimming when focus is on the right pane
	if m.focus == FocusPaneRight {
		leftView = lipgloss.NewStyle().
			Faint(true).
			Render(leftView)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)
}

// renderPlaceholder renders a centered placeholder message.
func renderPlaceholder(width, height int, message string) string {
	content := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render(message)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
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

// formatStoredEvent converts a persisted LogRecord into a display line for the event buffer.
func formatStoredEvent(ev state.LogRecord) string {
	prefix := ""
	if ev.StepID != "" {
		prefix = fmt.Sprintf("[%s] ", ev.StepID)
	}
	if ev.Message != "" {
		return prefix + ev.Message
	}
	return fmt.Sprintf("%s%s", prefix, ev.State)
}
