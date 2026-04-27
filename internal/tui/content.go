package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/state"
)

// ContentProviders holds data providers for alternative views.
type ContentProviders struct {
	PersonaProvider  PersonaDataProvider
	ContractProvider ContractDataProvider
	SkillProvider    SkillDataProvider
	HealthProvider   HealthDataProvider
	IssueProvider    IssueDataProvider
	PRProvider       PRDataProvider
	SuggestProvider  SuggestDataProvider
	OntologyProvider OntologyDataProvider
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
	issueList      *IssueListModel
	issueDetail    *IssueDetailModel
	prList         *PRListModel
	prDetail       *PRDetailModel
	suggestList    *SuggestListModel
	suggestDetail  *SuggestDetailModel
	ontologyList   *OntologyListModel
	ontologyDetail *OntologyDetailModel

	// Data providers for alternative views
	personaProvider  PersonaDataProvider
	contractProvider ContractDataProvider
	skillProvider    SkillDataProvider
	healthProvider   HealthDataProvider
	issueProvider    IssueDataProvider
	prProvider       PRDataProvider
	suggestProvider  SuggestDataProvider
	ontologyProvider OntologyDataProvider

	// Compose mode (nil when inactive)
	composing     bool
	composeList   *ComposeListModel
	composeDetail *ComposeDetailModel

	// When true, the Issues view right pane shows pipeline detail instead of issue detail.
	issueShowPipeline bool

	// Detached pipeline event polling
	detachedPollRunID   string
	detachedPollAfterID int64
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
		m.issueProvider = p.IssueProvider
		m.prProvider = p.PRProvider
		m.suggestProvider = p.SuggestProvider
		m.ontologyProvider = p.OntologyProvider
	}

	return m
}

// IsInputActive returns true if any text input is active (filter, form, compose) and
// printable key events should be consumed rather than treated as shortcuts.
func (m ContentModel) IsInputActive() bool {
	if m.IsFiltering() {
		return true
	}
	if m.currentView == ViewPipelines && m.detail.paneState == stateConfiguring {
		return true
	}
	if m.composing {
		return true
	}
	return false
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
	case ViewIssues:
		return m.issueList != nil && m.issueList.filtering
	case ViewPullRequests:
		return m.prList != nil && m.prList.filtering
	case ViewSuggest:
		return m.suggestList != nil && m.suggestList.filtering
	case ViewOntology:
		return m.ontologyList != nil && m.ontologyList.filtering
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
// childHeight returns the usable height for child models (minus top and bottom padding lines).
func (m ContentModel) childHeight() int {
	if m.height <= 2 {
		return 0
	}
	return m.height - 2
}

func (m *ContentModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	leftWidth := m.leftPaneWidth()
	rightWidth := w - leftWidth - 3 // 3 chars for separator: space + │ + space

	m.list.SetSize(leftWidth, m.childHeight())
	m.detail.SetSize(rightWidth, m.childHeight())

	// Propagate to non-nil alternative view models
	if m.personaList != nil {
		m.personaList.SetSize(leftWidth, m.childHeight())
	}
	if m.personaDetail != nil {
		m.personaDetail.SetSize(rightWidth, m.childHeight())
	}
	if m.contractList != nil {
		m.contractList.SetSize(leftWidth, m.childHeight())
	}
	if m.contractDetail != nil {
		m.contractDetail.SetSize(rightWidth, m.childHeight())
	}
	if m.skillList != nil {
		m.skillList.SetSize(leftWidth, m.childHeight())
	}
	if m.skillDetail != nil {
		m.skillDetail.SetSize(rightWidth, m.childHeight())
	}
	if m.healthList != nil {
		m.healthList.SetSize(leftWidth, m.childHeight())
	}
	if m.healthDetail != nil {
		m.healthDetail.SetSize(rightWidth, m.childHeight())
	}
	if m.issueList != nil {
		m.issueList.SetSize(leftWidth, m.childHeight())
	}
	if m.issueDetail != nil {
		m.issueDetail.SetSize(rightWidth, m.childHeight())
	}
	if m.prList != nil {
		m.prList.SetSize(leftWidth, m.childHeight())
	}
	if m.prDetail != nil {
		m.prDetail.SetSize(rightWidth, m.childHeight())
	}
	if m.suggestList != nil {
		m.suggestList.SetSize(leftWidth, m.childHeight())
	}
	if m.suggestDetail != nil {
		m.suggestDetail.SetSize(rightWidth, m.childHeight())
	}
	if m.ontologyList != nil {
		m.ontologyList.SetSize(leftWidth, m.childHeight())
	}
	if m.ontologyDetail != nil {
		m.ontologyDetail.SetSize(rightWidth, m.childHeight())
	}
	if m.composeList != nil {
		m.composeList.SetSize(leftWidth, m.childHeight())
	}
	if m.composeDetail != nil {
		m.composeDetail.SetSize(rightWidth, m.childHeight())
	}
}

// cycleView moves to the next view and returns init commands if the view was just created.
func (m *ContentModel) cycleView() tea.Cmd {
	m.currentView = (m.currentView + 1) % 9
	m.focus = FocusPaneLeft

	var initCmd tea.Cmd

	leftWidth := m.leftPaneWidth()
	rightWidth := m.width - leftWidth - 3

	switch m.currentView {
	case ViewPipelines:
		m.list.SetFocused(true)
		m.detail.SetFocused(false)

	case ViewPersonas:
		if m.personaList == nil && m.personaProvider != nil {
			pl := NewPersonaListModel(m.personaProvider)
			pl.SetSize(leftWidth, m.childHeight())
			m.personaList = &pl
			pd := NewPersonaDetailModel(m.personaProvider)
			pd.SetSize(rightWidth, m.childHeight())
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
			cl.SetSize(leftWidth, m.childHeight())
			m.contractList = &cl
			cd := NewContractDetailModel()
			cd.SetSize(rightWidth, m.childHeight())
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
			sl.SetSize(leftWidth, m.childHeight())
			m.skillList = &sl
			sd := NewSkillDetailModel()
			sd.SetSize(rightWidth, m.childHeight())
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
			hl.SetSize(leftWidth, m.childHeight())
			m.healthList = &hl
			hd := NewHealthDetailModel()
			hd.SetSize(rightWidth, m.childHeight())
			m.healthDetail = &hd
			initCmd = m.healthList.Init()
		}
		if m.healthList != nil {
			m.healthList.SetFocused(true)
		}
		if m.healthDetail != nil {
			m.healthDetail.SetFocused(false)
		}

	case ViewIssues:
		if m.issueList == nil && m.issueProvider != nil {
			il := NewIssueListModel(m.issueProvider)
			il.SetSize(leftWidth, m.childHeight())
			m.issueList = &il
			id := NewIssueDetailModel()
			id.SetSize(rightWidth, m.childHeight())
			// Populate available pipelines for the chooser
			if m.list.available != nil {
				id.SetPipelines(m.list.available)
			}
			m.issueDetail = &id
			initCmd = m.issueList.Init()
		}
		if m.issueList != nil {
			m.issueList.SetFocused(true)
		}
		if m.issueDetail != nil {
			m.issueDetail.SetFocused(false)
		}

	case ViewPullRequests:
		if m.prList == nil && m.prProvider != nil {
			pl := NewPRListModel(m.prProvider)
			pl.SetSize(leftWidth, m.childHeight())
			m.prList = &pl
			pd := NewPRDetailModel()
			pd.SetSize(rightWidth, m.childHeight())
			m.prDetail = &pd
			initCmd = m.prList.Init()
		}
		if m.prList != nil {
			m.prList.SetFocused(true)
		}
		if m.prDetail != nil {
			m.prDetail.SetFocused(false)
		}

	case ViewSuggest:
		if m.suggestList == nil && m.suggestProvider != nil {
			sl := NewSuggestListModel(m.suggestProvider)
			sl.SetSize(leftWidth, m.childHeight())
			m.suggestList = &sl
			sd := NewSuggestDetailModel()
			sd.SetSize(rightWidth, m.childHeight())
			m.suggestDetail = &sd
			initCmd = m.suggestList.Init()
		}
		if m.suggestList != nil {
			m.suggestList.SetFocused(true)
		}
		if m.suggestDetail != nil {
			m.suggestDetail.SetFocused(false)
		}

	case ViewOntology:
		if m.ontologyList == nil && m.ontologyProvider != nil {
			ol := NewOntologyListModel(m.ontologyProvider)
			ol.SetSize(leftWidth, m.childHeight())
			m.ontologyList = &ol
			od := NewOntologyDetailModel()
			od.SetSize(rightWidth, m.childHeight())
			m.ontologyDetail = &od
			initCmd = m.ontologyList.Init()
		}
		if m.ontologyList != nil {
			m.ontologyList.SetFocused(true)
		}
		if m.ontologyDetail != nil {
			m.ontologyDetail.SetFocused(false)
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

// setView switches directly to a specific view, lazy-initializing models as needed.
func (m *ContentModel) setView(v ViewType) tea.Cmd {
	m.currentView = v
	m.focus = FocusPaneLeft

	var initCmd tea.Cmd
	leftWidth := m.leftPaneWidth()
	rightWidth := m.width - leftWidth - 3

	switch v {
	case ViewPipelines:
		m.list.SetFocused(true)
		m.detail.SetFocused(false)

	case ViewPersonas:
		if m.personaList == nil && m.personaProvider != nil {
			pl := NewPersonaListModel(m.personaProvider)
			pl.SetSize(leftWidth, m.childHeight())
			m.personaList = &pl
			pd := NewPersonaDetailModel(m.personaProvider)
			pd.SetSize(rightWidth, m.childHeight())
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
			cl.SetSize(leftWidth, m.childHeight())
			m.contractList = &cl
			cd := NewContractDetailModel()
			cd.SetSize(rightWidth, m.childHeight())
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
			sl.SetSize(leftWidth, m.childHeight())
			m.skillList = &sl
			sd := NewSkillDetailModel()
			sd.SetSize(rightWidth, m.childHeight())
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
			hl.SetSize(leftWidth, m.childHeight())
			m.healthList = &hl
			hd := NewHealthDetailModel()
			hd.SetSize(rightWidth, m.childHeight())
			m.healthDetail = &hd
			initCmd = m.healthList.Init()
		}
		if m.healthList != nil {
			m.healthList.SetFocused(true)
		}
		if m.healthDetail != nil {
			m.healthDetail.SetFocused(false)
		}

	case ViewIssues:
		if m.issueList == nil && m.issueProvider != nil {
			il := NewIssueListModel(m.issueProvider)
			il.SetSize(leftWidth, m.childHeight())
			m.issueList = &il
			id := NewIssueDetailModel()
			id.SetSize(rightWidth, m.childHeight())
			if m.list.available != nil {
				id.SetPipelines(m.list.available)
			}
			m.issueDetail = &id
			initCmd = m.issueList.Init()
		}
		if m.issueList != nil {
			m.issueList.SetFocused(true)
		}
		if m.issueDetail != nil {
			m.issueDetail.SetFocused(false)
		}

	case ViewPullRequests:
		if m.prList == nil && m.prProvider != nil {
			pl := NewPRListModel(m.prProvider)
			pl.SetSize(leftWidth, m.childHeight())
			m.prList = &pl
			pd := NewPRDetailModel()
			pd.SetSize(rightWidth, m.childHeight())
			m.prDetail = &pd
			initCmd = m.prList.Init()
		}
		if m.prList != nil {
			m.prList.SetFocused(true)
		}
		if m.prDetail != nil {
			m.prDetail.SetFocused(false)
		}

	case ViewSuggest:
		if m.suggestList == nil && m.suggestProvider != nil {
			sl := NewSuggestListModel(m.suggestProvider)
			sl.SetSize(leftWidth, m.childHeight())
			m.suggestList = &sl
			sd := NewSuggestDetailModel()
			sd.SetSize(rightWidth, m.childHeight())
			m.suggestDetail = &sd
			initCmd = m.suggestList.Init()
		}
		if m.suggestList != nil {
			m.suggestList.SetFocused(true)
		}
		if m.suggestDetail != nil {
			m.suggestDetail.SetFocused(false)
		}

	case ViewOntology:
		if m.ontologyList == nil && m.ontologyProvider != nil {
			ol := NewOntologyListModel(m.ontologyProvider)
			ol.SetSize(leftWidth, m.childHeight())
			m.ontologyList = &ol
			od := NewOntologyDetailModel()
			od.SetSize(rightWidth, m.childHeight())
			m.ontologyDetail = &od
			initCmd = m.ontologyList.Init()
		}
		if m.ontologyList != nil {
			m.ontologyList.SetFocused(true)
		}
		if m.ontologyDetail != nil {
			m.ontologyDetail.SetFocused(false)
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

// numberKeyToView maps number key strings to view types.
func numberKeyToView(key string) (ViewType, bool) {
	switch key {
	case "1":
		return ViewPipelines, true
	case "2":
		return ViewPersonas, true
	case "3":
		return ViewContracts, true
	case "4":
		return ViewSkills, true
	case "5":
		return ViewHealth, true
	case "6":
		return ViewIssues, true
	case "7":
		return ViewPullRequests, true
	case "8":
		return ViewSuggest, true
	case "9":
		return ViewOntology, true
	default:
		return 0, false
	}
}

// Update handles messages by forwarding to child components with focus-aware routing.
func (m ContentModel) Update(msg tea.Msg) (ContentModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Intercept Shift+Tab for reverse view cycling
		if msg.Type == tea.KeyShiftTab {
			if m.composing {
				return m, nil
			}
			// Decrement twice: once to undo the +1 in cycleView, once for the actual back
			m.currentView = (m.currentView + 7) % 9 // net effect: -1 after cycleView adds +1
			cmd := m.cycleView()
			return m, cmd
		}

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
			// Cycle view
			cmd := m.cycleView()
			return m, cmd
		}

		// Number key direct-jump navigation (1-9) when in left pane and no input active
		if m.focus == FocusPaneLeft && !m.IsInputActive() {
			if v, ok := numberKeyToView(msg.String()); ok {
				cmd := m.setView(v)
				return m, cmd
			}
		}

		// Handle Enter for alternative views — focus right pane.
		// Suggest view is excluded: Enter launches the selected pipeline there.
		if msg.Type == tea.KeyEnter && m.focus == FocusPaneLeft && m.currentView != ViewPipelines && m.currentView != ViewSuggest {
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

				return m, tea.Batch(enterCmds...)
			}
			// Section header or non-focusable — forward to list for collapse/no-op
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
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
			)
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
				return m, m.list.fetchPipelineData
			}
			return m, nil
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
			return m, tea.Batch(cmds...)
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

	case IssueDataMsg:
		if m.issueList != nil {
			var cmd tea.Cmd
			*m.issueList, cmd = m.issueList.Update(msg)
			return m, cmd
		}
		return m, nil

	case IssueSelectedMsg:
		// Switch back to issue detail when an issue row is selected.
		m.issueShowPipeline = false
		if m.issueList != nil {
			var listCmd tea.Cmd
			*m.issueList, listCmd = m.issueList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.issueDetail != nil && m.issueList != nil {
			if msg.Index >= 0 && msg.Index < len(m.issueList.navigable) {
				item := m.issueList.navigable[msg.Index]
				if item.issue != nil {
					m.issueDetail.SetIssue(item.issue)
				}
			}
		}
		return m, tea.Batch(cmds...)

	case IssueLaunchMsg:
		// Convert to a LaunchRequestMsg and switch to Pipelines view
		m.currentView = ViewPipelines
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
		launchCmd := func() tea.Msg {
			return LaunchRequestMsg{Config: LaunchConfig{
				PipelineName: msg.PipelineName,
				Input:        msg.IssueURL,
			}}
		}
		viewCmd := func() tea.Msg {
			return ViewChangedMsg{View: ViewPipelines}
		}
		focusCmd := func() tea.Msg {
			return FocusChangedMsg{Pane: FocusPaneLeft}
		}
		return m, tea.Batch(launchCmd, viewCmd, focusCmd)

	case PRDataMsg:
		if m.prList != nil {
			var cmd tea.Cmd
			*m.prList, cmd = m.prList.Update(msg)
			return m, cmd
		}
		return m, nil

	case PRSelectedMsg:
		if m.prList != nil {
			var listCmd tea.Cmd
			*m.prList, listCmd = m.prList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.prDetail != nil && m.prList != nil {
			if msg.Index >= 0 && msg.Index < len(m.prList.navigable) {
				prIdx := m.prList.navigable[msg.Index]
				m.prDetail.SetPR(&m.prList.prs[prIdx])
			}
		}
		return m, tea.Batch(cmds...)

	case SuggestDataMsg:
		if m.suggestList != nil {
			var cmd tea.Cmd
			*m.suggestList, cmd = m.suggestList.Update(msg)
			return m, cmd
		}
		return m, nil

	case SuggestLaunchedMsg:
		if m.suggestList != nil {
			var cmd tea.Cmd
			*m.suggestList, cmd = m.suggestList.Update(msg)
			return m, cmd
		}
		return m, nil

	case SuggestSelectedMsg:
		if m.suggestList != nil {
			var listCmd tea.Cmd
			*m.suggestList, listCmd = m.suggestList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.suggestDetail != nil {
			var detailCmd tea.Cmd
			*m.suggestDetail, detailCmd = m.suggestDetail.Update(msg)
			if detailCmd != nil {
				cmds = append(cmds, detailCmd)
			}
		}
		return m, tea.Batch(cmds...)

	case OntologyDataMsg:
		if m.ontologyList != nil {
			var cmd tea.Cmd
			*m.ontologyList, cmd = m.ontologyList.Update(msg)
			return m, cmd
		}
		return m, nil

	case OntologySelectedMsg:
		if m.ontologyList != nil {
			var listCmd tea.Cmd
			*m.ontologyList, listCmd = m.ontologyList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.ontologyDetail != nil && m.ontologyList != nil {
			for i := range m.ontologyList.navigable {
				if m.ontologyList.navigable[i].Name == msg.Name {
					m.ontologyDetail.SetContext(&m.ontologyList.navigable[i])
					break
				}
			}
		}
		return m, tea.Batch(cmds...)

	case SuggestLaunchMsg:
		// Switch to Pipelines view and launch the suggested pipeline
		m.currentView = ViewPipelines
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
		pipelineName := msg.Pipeline.Name
		launchCmd := func() tea.Msg {
			return LaunchRequestMsg{Config: LaunchConfig{
				PipelineName: pipelineName,
				Input:        msg.Pipeline.Input,
			}}
		}
		viewCmd := func() tea.Msg {
			return ViewChangedMsg{View: ViewPipelines}
		}
		focusCmd := func() tea.Msg {
			return FocusChangedMsg{Pane: FocusPaneLeft}
		}
		launchedCmd := func() tea.Msg {
			return SuggestLaunchedMsg{Name: pipelineName}
		}
		refreshCmd := tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
			return PipelineRefreshTickMsg{}
		})
		return m, tea.Batch(launchCmd, viewCmd, focusCmd, launchedCmd, refreshCmd)

	case SuggestComposeMsg:
		// Bridge suggest multi-select to compose mode: switch to Pipelines view,
		// enter compose mode with the selected proposals pre-populated.
		m.currentView = ViewPipelines
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)

		if len(msg.Pipelines) > 0 && m.launcher != nil {
			// Build compose sequence from selected proposals
			var seq Sequence
			for _, p := range msg.Pipelines {
				loaded, err := LoadPipelineByName(m.launcher.deps.PipelinesDir, p.Name)
				if err == nil {
					seq.Add(p.Name, loaded)
				} else {
					seq.Add(p.Name, nil)
				}
			}

			cl := ComposeListModel{
				available:  m.list.available,
				sequence:   seq,
				validation: ValidateSequence(seq),
				focused:    true,
			}
			cd := NewComposeDetailModel()

			m.composing = true
			m.composeList = &cl
			m.composeDetail = &cd

			leftWidth := m.leftPaneWidth()
			rightWidth := m.width - leftWidth - 3
			m.composeList.SetSize(leftWidth, m.childHeight())
			m.composeDetail.SetSize(rightWidth, m.childHeight())

			seqCopy := cl.sequence
			val := cl.validation
			return m, tea.Batch(
				func() tea.Msg { return ViewChangedMsg{View: ViewPipelines} },
				func() tea.Msg { return ComposeActiveMsg{Active: true} },
				func() tea.Msg {
					return ComposeSequenceChangedMsg{
						Sequence:   seqCopy,
						Validation: val,
					}
				},
			)
		}
		return m, func() tea.Msg { return ViewChangedMsg{View: ViewPipelines} }

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

	case HealthAllCompleteMsg:
		return m, nil

	case HealthContinueMsg:
		return m, nil

	case SuggestModifyMsg:
		// Open configure form pre-populated with the proposal's pipeline and input
		m.currentView = ViewPipelines
		m.focus = FocusPaneRight
		m.list.SetFocused(false)
		m.detail.SetFocused(true)
		return m, func() tea.Msg {
			return ConfigureFormMsg{
				PipelineName: msg.Pipeline.Name,
				InputExample: msg.Pipeline.Input,
			}
		}

	// Pipeline-specific messages — always route to pipeline models
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
		return m, tea.Batch(cmds...)

	case DetailDataMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case PipelineRefreshTickMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case PipelineDataMsg:
		// Detached pipelines are tracked via SQLite — no in-memory merge needed.
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		// Also update issue list with pipeline data so it can show children.
		if m.issueList != nil {
			var issueCmd tea.Cmd
			*m.issueList, issueCmd = m.issueList.Update(msg)
			return m, tea.Batch(cmd, issueCmd)
		}
		return m, cmd

	case PipelineEventMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case DashboardTickMsg:
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
		// Also forward to issue list for running pipeline elapsed time updates.
		if m.issueList != nil {
			var issueCmd tea.Cmd
			*m.issueList, issueCmd = m.issueList.Update(msg)
			return m, tea.Batch(cmd, issueCmd)
		}
		return m, cmd

	case ConfigureFormMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case LaunchRequestMsg:
		if m.launcher != nil {
			// Show "Starting pipeline..." while the async launch runs.
			// Without this, compose-launched pipelines briefly flash stale
			// detail content before PipelineLaunchedMsg arrives.
			m.detail.paneState = stateLaunching
			m.detail.selectedName = msg.Config.PipelineName
			cmd := m.launcher.Launch(msg.Config)
			return m, cmd
		}
		return m, nil

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
		return m, tea.Batch(batchCmds...)

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
		return m, m.list.fetchPipelineData

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
		return m, tea.Batch(batchCmds...)

	case FocusChangedMsg:
		if msg.Pane == FocusPaneLeft {
			m.focus = FocusPaneLeft
			m.list.SetFocused(true)
			m.detail.SetFocused(false)
		}
		return m, nil

	case RunEventsMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case DetachedEventPollTickMsg:
		// Stop polling if the run ID changed or we left the live output pane
		if m.detachedPollRunID != msg.RunID || m.detail.paneState != stateRunningLive {
			m.detachedPollRunID = ""
			return m, nil
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
					})
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
					})
				}
			}
		}
		capturedRunID := msg.RunID
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return DetachedEventPollTickMsg{RunID: capturedRunID}
		})
	}

	// When composing, forward non-key messages (huh internal ticks, updateFieldMsg,
	// WindowSize responses, etc.) to compose models so the picker form works.
	if m.composing {
		if m.composeList != nil {
			var cmd tea.Cmd
			*m.composeList, cmd = m.composeList.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if m.composeDetail != nil {
			var cmd tea.Cmd
			*m.composeDetail, cmd = m.composeDetail.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
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
	case ViewIssues:
		if m.issueList != nil {
			m.issueList.SetFocused(false)
		}
		if m.issueShowPipeline {
			m.detail.SetFocused(true)
		} else if m.issueDetail != nil {
			m.issueDetail.SetFocused(true)
		}
	case ViewPullRequests:
		if m.prList != nil {
			m.prList.SetFocused(false)
		}
		if m.prDetail != nil {
			m.prDetail.SetFocused(true)
		}
	case ViewSuggest:
		if m.suggestList != nil {
			m.suggestList.SetFocused(false)
		}
		if m.suggestDetail != nil {
			m.suggestDetail.SetFocused(true)
		}
	case ViewOntology:
		if m.ontologyList != nil {
			m.ontologyList.SetFocused(false)
		}
		if m.ontologyDetail != nil {
			m.ontologyDetail.SetFocused(true)
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
	case ViewIssues:
		if m.issueList != nil {
			m.issueList.SetFocused(true)
		}
		if m.issueShowPipeline {
			m.detail.SetFocused(false)
		} else if m.issueDetail != nil {
			m.issueDetail.SetFocused(false)
		}
	case ViewPullRequests:
		if m.prList != nil {
			m.prList.SetFocused(true)
		}
		if m.prDetail != nil {
			m.prDetail.SetFocused(false)
		}
	case ViewSuggest:
		if m.suggestList != nil {
			m.suggestList.SetFocused(true)
		}
		if m.suggestDetail != nil {
			m.suggestDetail.SetFocused(false)
		}
	case ViewOntology:
		if m.ontologyList != nil {
			m.ontologyList.SetFocused(true)
		}
		if m.ontologyDetail != nil {
			m.ontologyDetail.SetFocused(false)
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
	case ViewIssues:
		if m.issueList != nil {
			var cmd tea.Cmd
			*m.issueList, cmd = m.issueList.Update(msg)
			return m, cmd
		}
	case ViewPullRequests:
		if m.prList != nil {
			var cmd tea.Cmd
			*m.prList, cmd = m.prList.Update(msg)
			return m, cmd
		}
	case ViewSuggest:
		if m.suggestList != nil {
			var cmd tea.Cmd
			*m.suggestList, cmd = m.suggestList.Update(msg)
			return m, cmd
		}
	case ViewOntology:
		if m.ontologyList != nil {
			var cmd tea.Cmd
			*m.ontologyList, cmd = m.ontologyList.Update(msg)
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
	case ViewIssues:
		if m.issueShowPipeline {
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}
		if m.issueDetail != nil {
			var cmd tea.Cmd
			*m.issueDetail, cmd = m.issueDetail.Update(msg)
			return m, cmd
		}
	case ViewPullRequests:
		if m.prDetail != nil {
			var cmd tea.Cmd
			*m.prDetail, cmd = m.prDetail.Update(msg)
			return m, cmd
		}
	case ViewSuggest:
		if m.suggestDetail != nil {
			var cmd tea.Cmd
			*m.suggestDetail, cmd = m.suggestDetail.Update(msg)
			return m, cmd
		}
	case ViewOntology:
		if m.ontologyDetail != nil {
			var cmd tea.Cmd
			*m.ontologyDetail, cmd = m.ontologyDetail.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// cursorOnFocusableItem returns true if the cursor is on a pipeline name, finished, or running item.
func (m ContentModel) cursorOnFocusableItem() bool {
	if len(m.list.navigable) == 0 || m.list.cursor >= len(m.list.navigable) {
		return false
	}
	kind := m.list.navigable[m.list.cursor].kind
	return kind == itemKindPipelineName || kind == itemKindFinished || kind == itemKindRunning
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
			rightView = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a persona to view details")
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
			rightView = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a contract to view details")
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
			rightView = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a skill to view details")
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
			rightView = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a health check to view details")
		}

	case ViewIssues:
		if m.issueList != nil {
			leftView = m.issueList.View()
		} else {
			leftView = renderPlaceholder(m.leftPaneWidth(), m.height, "No repository configured")
		}
		switch {
		case m.issueShowPipeline:
			// Show pipeline detail when a pipeline child is selected.
			rightView = m.detail.View()
		case m.issueDetail != nil:
			rightView = m.issueDetail.View()
		default:
			rightView = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select an issue to view details")
		}

	case ViewPullRequests:
		if m.prList != nil {
			leftView = m.prList.View()
		} else {
			leftView = renderPlaceholder(m.leftPaneWidth(), m.height, "No repository configured")
		}
		if m.prDetail != nil {
			rightView = m.prDetail.View()
		} else {
			rightView = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a pull request to view details")
		}

	case ViewSuggest:
		if m.suggestList != nil {
			leftView = m.suggestList.View()
		} else {
			leftView = renderPlaceholder(m.leftPaneWidth(), m.height, "No suggest provider configured")
		}
		if m.suggestDetail != nil {
			rightView = m.suggestDetail.View()
		} else {
			rightView = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a suggestion to view details")
		}

	case ViewOntology:
		if m.ontologyList != nil {
			leftView = m.ontologyList.View()
		} else {
			leftView = renderPlaceholder(m.leftPaneWidth(), m.height, "No ontology configured")
		}
		if m.ontologyDetail != nil {
			rightView = m.ontologyDetail.View()
		} else {
			rightView = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a context to view details")
		}
	}

	// Add top padding (blank line) for visual separation from status bar divider
	leftView = "\n" + leftView
	rightView = "\n" + rightView

	// L5: Apply focus styling to panes
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Repeat("│\n", m.height))
	separatorLines := strings.Split(separator, "\n")
	if len(separatorLines) > m.height {
		separatorLines = separatorLines[:m.height]
	}
	separator = strings.Join(separatorLines, "\n")

	switch m.focus {
	case FocusPaneRight:
		leftView = lipgloss.NewStyle().
			Faint(true).
			Render(leftView)
	case FocusPaneLeft:
		rightView = lipgloss.NewStyle().
			Faint(true).
			Render(rightView)
	}

	// L1: Add padding via separator between panes
	result := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().PaddingLeft(1).Render(leftView),
		separator,
		lipgloss.NewStyle().PaddingLeft(1).Render(rightView),
	)

	// Enforce exact height to prevent header clipping from stray extra lines
	resultLines := strings.Split(result, "\n")
	for len(resultLines) < m.height {
		resultLines = append(resultLines, "")
	}
	if len(resultLines) > m.height {
		resultLines = resultLines[:m.height]
	}
	return strings.Join(resultLines, "\n")
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

// formatStoredEvent formats a persisted LogRecord for display in the live
// output buffer. Thin wrapper that adapts the LogRecord into an event.Event
// view and delegates to the canonical display.EventLine formatter. The
// canonical implementation lives in internal/display/eventline.go.
func formatStoredEvent(ev state.LogRecord) string {
	evt := eventFromLogRecord(ev)
	line, _ := display.EventLine(evt, display.LiveTUIProfile(!noColor()))
	return line
}
