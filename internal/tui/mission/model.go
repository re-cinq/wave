package mission

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/meta"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/tui"
)

// staleRunThreshold is the age after which a non-local "running" run is marked stale.
const staleRunThreshold = 30 * time.Minute

// Options configures the mission control TUI.
type Options struct {
	ManifestPath  string
	Debug         bool
	Mock          bool
	ModelOverride string
}

// TickMsg triggers periodic UI updates.
type TickMsg time.Time

// PipelinesDiscoveredMsg carries discovered pipelines (from Init).
type PipelinesDiscoveredMsg struct {
	Pipelines []tui.PipelineInfo
}

// InstallResultMsg carries the results of auto-installing dependencies.
type InstallResultMsg struct {
	Results []meta.InstallResult
	Err     error
}

// ChatFinishedMsg is sent when a wave chat subprocess exits.
type ChatFinishedMsg struct{ Err error }

// RunSnapshot holds the display state for a single pipeline run.
type RunSnapshot struct {
	RunID          string
	PipelineName   string
	Status         string
	CurrentStep    string
	ErrorMessage   string
	Input          string
	Progress       int
	TotalSteps     int
	CompletedSteps int
	TotalTokens    int
	TokensIn       int
	TokensOut      int
	StartedAt      time.Time
	Elapsed        time.Duration
	Local          bool // true = managed by this TUI session
}

// isActive returns true if the run is still active.
func (r *RunSnapshot) isActive() bool {
	switch r.Status {
	case "running", "queued", "pending":
		return true
	default:
		return false
	}
}

// storeRecord is a minimal interface for store data used in merging.
type storeRecord struct {
	RunID        string
	PipelineName string
	Status       string
	CurrentStep  string
	TotalTokens  int
	ErrorMessage string
	StartedAt    time.Time
	CompletedAt  *time.Time
}

// MissionControlModel is the root tea.Model — guided workflow orchestrator.
//
// State machine: ViewHealthPhase → ViewProposals ←Tab→ ViewFleet → ViewAttached
// Overlays: OverlayForm, OverlayHealth, OverlayHelp
type MissionControlModel struct {
	// View state
	activeView ViewID
	overlay    OverlayID

	// Run list (fleet view)
	runs       []RunSnapshot
	cursor     int
	scrollOff  int
	filter     string
	filterMode bool

	// Per-run rendering context (maps runID -> RunContext)
	runContexts map[string]*RunContext

	// Attached mode
	attachedRunID string

	// Health phase state
	healthChecks []HealthCheckStatus
	healthReport *meta.HealthReport

	// Proposal state
	proposals       []meta.PipelineProposal
	proposalCursor  int
	proposalSelect  map[int]bool // multi-select: index → selected
	proposalSkipped map[int]bool // skipped proposals

	// Health overlay state (for fleet view 'h' key)
	healthContent   string
	healthScrollOff int

	// Embedded huh form (replaces broken form.Run() calls)
	activeForm   *huh.Form
	formKind     string                 // "pipeline-select" or "modify-input"
	formSelected *string                // pointer bound to Select widget
	formInput    *string                // pointer bound to Input/Text widget
	formProposal *meta.PipelineProposal // for modify: stores proposal

	// Infrastructure
	healthCache   *HealthCache
	runManager    *RunManager
	eventBus      *EventBus
	store         state.StateStore
	width         int
	height        int
	opts          Options
	quitting      bool
	err           error
	pipelineNames []string
	pipelineInfos []tui.PipelineInfo
	healthLoaded  bool
}

// NewMissionControlModel creates a new mission control model.
func NewMissionControlModel(opts Options, store state.StateStore) MissionControlModel {
	bus := NewEventBus()

	rmConfig := RunManagerConfig{
		ManifestPath:  opts.ManifestPath,
		Mock:          opts.Mock,
		Debug:         opts.Debug,
		ModelOverride: opts.ModelOverride,
	}

	healthCache := NewHealthCache(opts.ManifestPath, getVersion())

	return MissionControlModel{
		activeView:      ViewHealthPhase, // Guided workflow starts with health
		overlay:         OverlayNone,
		runContexts:     make(map[string]*RunContext),
		proposalSelect:  make(map[int]bool),
		proposalSkipped: make(map[int]bool),
		healthChecks:    []HealthCheckStatus{},
		healthCache:     healthCache,
		runManager:      NewRunManager(rmConfig, bus, store),
		eventBus:        bus,
		store:           store,
		opts:            opts,
	}
}

// Init implements tea.Model.
func (m MissionControlModel) Init() tea.Cmd {
	return tea.Batch(
		WaitForRunEvent(m.eventBus),
		InitialPoll(m.store),
		missionTickCmd(),
		m.healthCache.RefreshCmd(),
		discoverPipelines(),
	)
}

// discoverPipelines returns a cmd that discovers available pipelines.
func discoverPipelines() tea.Cmd {
	return func() tea.Msg {
		pipelines, _ := tui.DiscoverPipelines(".wave/pipelines")
		return PipelinesDiscoveredMsg{Pipelines: pipelines}
	}
}

// Update implements tea.Model.
func (m MissionControlModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case RunEventMsg:
		m.applyRunEvent(msg.RunID, msg.Event)
		return m, WaitForRunEvent(m.eventBus)

	case StatePolledMsg:
		if msg.Err == nil && len(msg.Records) > 0 {
			newRuns := m.mergeFromStore(msg.Records)
			var cmds []tea.Cmd
			cmds = append(cmds, PollState(m.store))
			for _, runID := range newRuns {
				cmds = append(cmds, LoadRunStepData(m.store, runID))
			}
			return m, tea.Batch(cmds...)
		}
		return m, PollState(m.store)

	case StepDataMsg:
		m.buildRunContextFromStore(msg)
		return m, nil

	case TickMsg:
		for i := range m.runs {
			r := &m.runs[i]
			if r.Status == "running" && !r.StartedAt.IsZero() {
				r.Elapsed = time.Since(r.StartedAt)
			}
		}
		return m, missionTickCmd()

	case PipelinesDiscoveredMsg:
		m.pipelineInfos = msg.Pipelines
		m.pipelineNames = make([]string, len(msg.Pipelines))
		for i, p := range msg.Pipelines {
			m.pipelineNames[i] = p.Name
		}
		return m, nil

	case HealthCacheMsg:
		if msg.Err == nil && msg.Report != nil {
			m.healthReport = msg.Report
			m.healthContent = tui.RenderHealthReport(msg.Report)

			// Build health check statuses for inline display
			m.healthChecks = []HealthCheckStatus{
				{
					Name:    "Wave initialized",
					Done:    true,
					Success: msg.Report.Init.ManifestFound,
					Detail:  trimVersion(msg.Report.Init.WaveVersion),
				},
			}

			allDeps := true
			toolCount := 0
			skillCount := 0
			for _, t := range msg.Report.Dependencies.Tools {
				if t.Available {
					toolCount++
				} else {
					allDeps = false
				}
			}
			for _, s := range msg.Report.Dependencies.Skills {
				if s.Available {
					skillCount++
				} else {
					allDeps = false
				}
			}
			m.healthChecks = append(m.healthChecks, HealthCheckStatus{
				Name:    "Dependencies verified",
				Done:    true,
				Success: allDeps,
				Detail:  fmt.Sprintf("%d tools, %d skills", toolCount, skillCount),
			})

			m.healthChecks = append(m.healthChecks, HealthCheckStatus{
				Name:    "Codebase analyzed",
				Done:    true,
				Success: true,
				Detail:  fmt.Sprintf("%d issues, %d PRs", msg.Report.Codebase.OpenIssueCount, msg.Report.Codebase.OpenPRCount),
			})

			platformName := string(msg.Report.Platform.Type)
			if platformName == "" {
				platformName = "unknown"
			}
			m.healthChecks = append(m.healthChecks, HealthCheckStatus{
				Name:    "Platform detected",
				Done:    true,
				Success: msg.Report.Platform.Type != "",
				Detail:  platformName,
			})

			// Generate proposals
			engine := meta.NewProposalEngine(msg.Report, m.pipelineNames)
			m.proposals = engine.GenerateProposals()
			m.proposalCursor = 0
			m.proposalSelect = make(map[int]bool)
			m.proposalSkipped = make(map[int]bool)

			// Check for auto-installable deps
			installable := meta.GetInstallable(msg.Report.Dependencies)
			if !allDeps && len(installable) > 0 {
				m.healthChecks = append(m.healthChecks, HealthCheckStatus{
					Name: fmt.Sprintf("Auto-installing %d dep(s)", len(installable)),
					Done: false,
				})
				m.healthLoaded = true
				// Auto-transition from health phase to proposals
				if m.activeView == ViewHealthPhase {
					m.activeView = ViewProposals
				}
				return m, installDepsCmd(installable)
			}

			m.healthLoaded = true

			// Auto-transition from health phase to proposals
			if m.activeView == ViewHealthPhase {
				m.activeView = ViewProposals
			}
		} else if msg.Err != nil {
			// Mark health as loaded but with error
			m.healthChecks = []HealthCheckStatus{
				{Name: "Health check", Done: true, Success: false, Detail: "check failed"},
			}
			m.healthLoaded = true
			// Auto-transition even on error
			if m.activeView == ViewHealthPhase {
				m.activeView = ViewProposals
			}
		}
		return m, nil

	case InstallResultMsg:
		// Update the auto-install health check
		allOK := true
		for _, r := range msg.Results {
			if !r.Success {
				allOK = false
				break
			}
		}
		for i := range m.healthChecks {
			if strings.HasPrefix(m.healthChecks[i].Name, "Auto-installing") {
				m.healthChecks[i].Done = true
				m.healthChecks[i].Success = allOK
				if allOK {
					m.healthChecks[i].Detail = "all installed"
				} else {
					failed := 0
					for _, r := range msg.Results {
						if !r.Success {
							failed++
						}
					}
					m.healthChecks[i].Detail = fmt.Sprintf("%d failed", failed)
				}
				break
			}
		}
		return m, nil

	case ChatFinishedMsg:
		// TUI has resumed after chat session — nothing to update
		return m, nil
	}

	return m, nil
}

// handleKey processes keyboard input.
func (m MissionControlModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Embedded form captures all keys when active
	if m.activeForm != nil {
		return m.handleFormKey(msg)
	}

	// Filter mode captures all keys
	if m.filterMode {
		return m.handleFilterKey(key)
	}

	// Overlay captures keys when active
	if m.overlay != OverlayNone {
		return m.handleOverlayKey(key)
	}

	// Global quit
	switch key {
	case "q", "ctrl+c":
		m.quitting = true
		m.runManager.Shutdown()
		m.eventBus.Close()
		return m, tea.Quit
	}

	// View-specific keys
	switch m.activeView {
	case ViewHealthPhase:
		return m.handleHealthPhaseKey(key)
	case ViewProposals:
		return m.handleProposalsKey(key)
	case ViewFleet:
		return m.handleFleetKey(key)
	case ViewAttached:
		return m.handleAttachedKey(key)
	}

	return m, nil
}

// handleHealthPhaseKey processes keys during health phase.
func (m MissionControlModel) handleHealthPhaseKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "tab":
		m.activeView = ViewFleet
	}
	return m, nil
}

// handleProposalsKey processes keys when proposals view is active.
func (m MissionControlModel) handleProposalsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		m.moveProposalCursorDown()
	case "k", "up":
		m.moveProposalCursorUp()
	case " ":
		// Toggle selection
		if m.proposalCursor >= 0 && m.proposalCursor < len(m.proposals) {
			if !m.proposalSkipped[m.proposalCursor] {
				m.proposalSelect[m.proposalCursor] = !m.proposalSelect[m.proposalCursor]
			}
		}
	case "s":
		// Skip proposal
		if m.proposalCursor >= 0 && m.proposalCursor < len(m.proposals) {
			m.proposalSkipped[m.proposalCursor] = true
			delete(m.proposalSelect, m.proposalCursor)
			// If all proposals are skipped, transition to fleet
			if m.allProposalsSkipped() {
				m.activeView = ViewFleet
			}
		}
	case "m":
		// Modify input — open embedded huh form
		if m.proposalCursor >= 0 && m.proposalCursor < len(m.proposals) {
			prop := m.proposals[m.proposalCursor]
			cmd := m.openModifyInputForm(&prop)
			return m, cmd
		}
	case "n":
		cmd := m.openPipelineSelectorForm()
		return m, cmd
	case "enter":
		return m.launchProposals()
	case "tab":
		m.activeView = ViewFleet
	case "?":
		m.overlay = OverlayHelp
	}
	return m, nil
}

// handleFleetKey processes keys when fleet view is active.
func (m MissionControlModel) handleFleetKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		m.moveDown()
	case "k", "up":
		m.moveUp()
	case "enter":
		if sel := m.selectedRun(); sel != nil {
			m.activeView = ViewAttached
			m.attachedRunID = sel.RunID
		}
	case "n":
		cmd := m.openPipelineSelectorForm()
		return m, cmd
	case "p", "tab":
		m.activeView = ViewProposals
	case "h":
		m.overlay = OverlayHealth
		m.healthScrollOff = 0
	case "c":
		if sel := m.selectedRun(); sel != nil && sel.Status == "running" {
			m.runManager.CancelRun(sel.RunID)
		}
	case "r":
		if sel := m.selectedRun(); sel != nil && sel.Status == "failed" {
			result, err := m.runManager.StartPipeline(sel.PipelineName, sel.Input)
			if err != nil {
				m.err = err
			} else {
				m.initRunContext(result)
				m.applyRunEvent(result.RunID, newStartedEvent(sel.PipelineName))
			}
		}
	case "o":
		if sel := m.selectedRun(); sel != nil && !sel.isActive() {
			return m, openChatCmd(sel.RunID)
		}
	case "/":
		m.filterMode = true
		m.filter = ""
	case "esc":
		m.filter = ""
	case "?":
		m.overlay = OverlayHelp
	}
	return m, nil
}

// handleAttachedKey processes keys when attached to a run.
func (m MissionControlModel) handleAttachedKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.activeView = ViewFleet
		m.attachedRunID = ""
	case "c":
		if m.attachedRunID != "" {
			m.runManager.CancelRun(m.attachedRunID)
		}
	case "o":
		if m.attachedRunID != "" {
			idx := m.findRun(m.attachedRunID)
			if idx >= 0 && !m.runs[idx].isActive() {
				return m, openChatCmd(m.attachedRunID)
			}
		}
	case "?":
		m.overlay = OverlayHelp
	}
	return m, nil
}

// handleFilterKey handles keys when filter mode is active.
func (m MissionControlModel) handleFilterKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.filterMode = false
		m.filter = ""
	case "ctrl+c":
		m.quitting = true
		m.runManager.Shutdown()
		m.eventBus.Close()
		return m, tea.Quit
	case "enter":
		m.filterMode = false
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
		}
		m.cursor = 0
		m.scrollOff = 0
	default:
		if len(key) == 1 {
			m.filter += key
		}
		m.cursor = 0
		m.scrollOff = 0
	}
	return m, nil
}

// handleOverlayKey processes keys when an overlay is active.
func (m MissionControlModel) handleOverlayKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.overlay = OverlayNone
		return m, nil
	case "ctrl+c":
		m.quitting = true
		m.runManager.Shutdown()
		m.eventBus.Close()
		return m, tea.Quit
	case "q":
		m.quitting = true
		m.runManager.Shutdown()
		m.eventBus.Close()
		return m, tea.Quit
	}

	switch m.overlay {
	case OverlayHealth:
		switch key {
		case "j", "down":
			m.healthScrollOff++
		case "k", "up":
			if m.healthScrollOff > 0 {
				m.healthScrollOff--
			}
		case "R":
			m.healthScrollOff = 0
			m.healthContent = ""
			return m, m.healthCache.RefreshCmd()
		}
	}

	return m, nil
}

// --- Embedded huh form management ---

// handleFormKey forwards key messages to the embedded huh form.
func (m MissionControlModel) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Allow quit even in form
	if key == "ctrl+c" {
		m.quitting = true
		m.runManager.Shutdown()
		m.eventBus.Close()
		return m, tea.Quit
	}

	// Esc cancels the form
	if key == "esc" {
		m.activeForm = nil
		m.formKind = ""
		m.overlay = OverlayNone
		return m, nil
	}

	// Forward to form
	_, cmd := m.activeForm.Update(msg)

	// Check form state after update
	switch m.activeForm.State {
	case huh.StateCompleted:
		return m.handleFormCompleted()
	case huh.StateAborted:
		m.activeForm = nil
		m.formKind = ""
		m.overlay = OverlayNone
		return m, nil
	}

	return m, cmd
}

// openPipelineSelectorForm creates an embedded form for pipeline selection.
func (m *MissionControlModel) openPipelineSelectorForm() tea.Cmd {
	pipelines := m.pipelineInfos
	if len(pipelines) == 0 {
		return nil
	}

	options := make([]huh.Option[string], len(pipelines))
	for i, p := range pipelines {
		label := p.Name
		if p.Description != "" {
			label = fmt.Sprintf("%-24s %s", p.Name, p.Description)
		}
		options[i] = huh.NewOption(label, p.Name)
	}

	var selected string
	var input string
	m.formSelected = &selected
	m.formInput = &input

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select pipeline").
				Options(options...).
				Value(&selected).
				Height(min(len(pipelines)+2, 15)),
			huh.NewInput().
				Title("Input").
				Placeholder("Describe what to do...").
				Value(&input),
		),
	).WithTheme(tui.WaveTheme())

	m.activeForm = form
	m.formKind = "pipeline-select"
	m.formProposal = nil
	m.overlay = OverlayForm

	return m.activeForm.Init()
}

// openModifyInputForm creates an embedded form for modifying proposal input.
func (m *MissionControlModel) openModifyInputForm(prop *meta.PipelineProposal) tea.Cmd {
	var input string
	input = prop.PrefilledInput
	m.formInput = &input
	m.formSelected = nil
	m.formProposal = prop

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Modify input for: "+strings.Join(prop.Pipelines, " → ")).
				Value(&input),
		),
	).WithTheme(tui.WaveTheme())

	m.activeForm = form
	m.formKind = "modify-input"
	m.overlay = OverlayForm

	return m.activeForm.Init()
}

// handleFormCompleted reads values from the completed form and dispatches.
func (m MissionControlModel) handleFormCompleted() (tea.Model, tea.Cmd) {
	kind := m.formKind
	m.activeForm = nil
	m.formKind = ""
	m.overlay = OverlayNone

	switch kind {
	case "pipeline-select":
		name := ""
		input := ""
		if m.formSelected != nil {
			name = *m.formSelected
		}
		if m.formInput != nil {
			input = *m.formInput
		}
		if name != "" {
			result, err := m.runManager.StartPipeline(name, input)
			if err != nil {
				m.err = err
			} else {
				m.initRunContext(result)
				m.applyRunEvent(result.RunID, newStartedEvent(name))
				m.activeView = ViewFleet
			}
		}

	case "modify-input":
		input := ""
		if m.formInput != nil {
			input = *m.formInput
		}
		if m.formProposal != nil {
			prop := *m.formProposal
			prop.PrefilledInput = input
			m.launchSingleProposal(prop)
		}
	}

	return m, nil
}

// allProposalsSkipped returns true if all proposals are skipped or there are none.
func (m *MissionControlModel) allProposalsSkipped() bool {
	if len(m.proposals) == 0 {
		return true
	}
	for i := range m.proposals {
		if !m.proposalSkipped[i] {
			return false
		}
	}
	return true
}

// launchProposals launches selected proposals (or cursor proposal if none selected).
func (m MissionControlModel) launchProposals() (tea.Model, tea.Cmd) {
	// Collect proposals to launch
	var toLaunch []meta.PipelineProposal
	hasSelected := false
	for i, sel := range m.proposalSelect {
		if sel && !m.proposalSkipped[i] {
			hasSelected = true
			toLaunch = append(toLaunch, m.proposals[i])
		}
	}

	if !hasSelected {
		// Launch cursor proposal
		if m.proposalCursor >= 0 && m.proposalCursor < len(m.proposals) && !m.proposalSkipped[m.proposalCursor] {
			toLaunch = append(toLaunch, m.proposals[m.proposalCursor])
		}
	}

	if len(toLaunch) == 0 {
		return m, nil
	}

	// Launch each proposal
	for _, prop := range toLaunch {
		m.launchSingleProposal(prop)
	}

	// Transition to fleet after launching
	m.activeView = ViewFleet
	return m, nil
}

// launchSingleProposal launches a single proposal by type.
func (m *MissionControlModel) launchSingleProposal(prop meta.PipelineProposal) {
	switch prop.Type {
	case meta.ProposalSequence:
		results, err := m.runManager.StartSequence(prop.Pipelines, prop.PrefilledInput)
		if err != nil {
			m.err = err
		} else {
			for i, r := range results {
				m.initRunContext(r)
				if i == 0 {
					m.applyRunEvent(r.RunID, newStartedEvent(prop.Pipelines[0]))
				} else {
					// Mark queued runs
					m.applyRunEvent(r.RunID, event.Event{
						Timestamp:  time.Now(),
						PipelineID: prop.Pipelines[i],
						State:      "queued",
					})
				}
			}
		}

	case meta.ProposalParallel:
		inputs := make(map[string]string, len(prop.Pipelines))
		for _, name := range prop.Pipelines {
			inputs[name] = prop.PrefilledInput
		}
		results, err := m.runManager.StartParallel(prop.Pipelines, inputs)
		if err != nil {
			m.err = err
		} else {
			for i, r := range results {
				m.initRunContext(r)
				m.applyRunEvent(r.RunID, newStartedEvent(prop.Pipelines[i]))
			}
		}

	case meta.ProposalSingle:
		for _, name := range prop.Pipelines {
			result, err := m.runManager.StartPipeline(name, prop.PrefilledInput)
			if err != nil {
				m.err = err
			} else {
				m.initRunContext(result)
				m.applyRunEvent(result.RunID, newStartedEvent(name))
			}
		}
	}
}

// View implements tea.Model.
func (m MissionControlModel) View() string {
	if m.quitting {
		return ""
	}

	contentHeight := m.height - 2 // 1 for help bar, 1 for lipgloss height offset
	if contentHeight < 1 {
		contentHeight = 1
	}

	var content string

	switch m.activeView {
	case ViewHealthPhase:
		content = renderHealthPhaseView(m.healthChecks, !m.healthLoaded, m.width, contentHeight)

	case ViewProposals:
		healthSummary := buildInlineHealthSummary(m.healthChecks, !m.healthLoaded)
		content = renderProposalsView(m.proposals, m.proposalCursor, m.proposalSelect, m.proposalSkipped, m.pipelineNames, healthSummary, m.width, contentHeight)

	case ViewFleet:
		healthSummary := buildInlineHealthSummary(m.healthChecks, !m.healthLoaded)
		proposalCount := len(m.proposals) - len(m.proposalSkipped)
		if proposalCount < 0 {
			proposalCount = 0
		}
		list := renderListPane(m.runs, m.cursor, m.scrollOff, m.filter, m.filterMode, healthSummary, proposalCount, m.listWidth(), contentHeight)
		var preview string
		sel := m.selectedRun()
		if sel != nil {
			rc := m.runContexts[sel.RunID]
			preview = renderPreviewPane(rc, sel, m.previewWidth(), contentHeight)
		}
		content = renderTwoPaneLayout(list, preview, m.width, contentHeight)

	case ViewAttached:
		rc := m.runContexts[m.attachedRunID]
		if rc != nil {
			content = display.RenderPipelineView(rc.Ctx)
		} else {
			content = styleMuted.Render("\n  Run context not available")
		}
	}

	// Overlay on top
	if m.activeForm != nil {
		title := "Pipeline Selector"
		if m.formKind == "modify-input" {
			title = "Modify Input"
		}
		content = renderFormOverlay(m.activeForm.View(), title, m.width, contentHeight)
	} else if m.overlay != OverlayNone {
		switch m.overlay {
		case OverlayHealth:
			content = renderHealthOverlay(m.healthContent, m.healthScrollOff, m.width, contentHeight)
		case OverlayHelp:
			content = renderHelpOverlay(m.width, contentHeight)
		}
	}

	// Error display
	if m.err != nil {
		content += "\n" + styleStatusFailed.Render("  Error: "+m.err.Error()) + "\n"
		m.err = nil
	}

	// Help bar
	helpBar := renderHelpBar(m.activeView, m.overlay, m.activeForm != nil, m.width)

	// Pad to fill screen — pin help bar to bottom
	content = strings.TrimRight(content, "\n")
	var b strings.Builder
	b.WriteString(content)
	contentLines := strings.Count(content, "\n") + 1
	padding := m.height - 1 - contentLines
	if padding > 0 {
		b.WriteString(strings.Repeat("\n", padding))
	}
	b.WriteByte('\n')
	b.WriteString(helpBar)

	return b.String()
}

// listWidth returns the width of the list pane.
func (m *MissionControlModel) listWidth() int {
	if m.width < 50 {
		return m.width
	}
	if m.width < 80 {
		return int(float64(m.width) * 0.40)
	}
	return int(float64(m.width) * 0.35)
}

// previewWidth returns the width of the preview pane.
func (m *MissionControlModel) previewWidth() int {
	return m.width - m.listWidth() - 1
}

// --- Run list management ---

// applyRunEvent updates a run snapshot and its RunContext from an event.
func (m *MissionControlModel) applyRunEvent(runID string, evt event.Event) {
	idx := m.findRun(runID)
	if idx == -1 {
		snap := RunSnapshot{
			RunID:     runID,
			Status:    "running",
			StartedAt: time.Now(),
			Local:     true,
		}
		m.runs = append(m.runs, snap)
		idx = len(m.runs) - 1
	}

	r := &m.runs[idx]

	if evt.PipelineID != "" && r.PipelineName == "" {
		r.PipelineName = evt.PipelineID
	}

	switch evt.State {
	case event.StateStarted:
		r.Status = "running"
		if r.PipelineName == "" {
			r.PipelineName = evt.PipelineID
		}
	case event.StateRunning:
		r.Status = "running"
		if evt.StepID != "" {
			r.CurrentStep = evt.StepID
		}
	case event.StateCompleted:
		if evt.StepID == "" {
			r.Status = "completed"
		} else {
			r.CurrentStep = evt.StepID
		}
	case event.StateFailed:
		if evt.StepID == "" {
			r.Status = "failed"
			r.ErrorMessage = evt.Message
		} else {
			r.CurrentStep = evt.StepID
		}
	case event.StateStreamActivity:
		if evt.StepID != "" {
			r.CurrentStep = evt.StepID
		}
	case "queued":
		r.Status = "queued"
	case "cancelled":
		r.Status = "cancelled"
		if evt.Message != "" {
			r.ErrorMessage = evt.Message
		}
	}

	if evt.TotalSteps > 0 {
		r.TotalSteps = evt.TotalSteps
	}
	if evt.CompletedSteps > 0 {
		r.CompletedSteps = evt.CompletedSteps
	}
	if evt.Progress > 0 {
		r.Progress = evt.Progress
	}
	if evt.TokensIn > 0 {
		r.TokensIn = evt.TokensIn
	}
	if evt.TokensOut > 0 {
		r.TokensOut = evt.TokensOut
	}
	if evt.TokensUsed > 0 {
		r.TotalTokens = evt.TokensUsed
	}

	if !r.StartedAt.IsZero() {
		r.Elapsed = time.Since(r.StartedAt)
	}

	// Update RunContext for step-level rendering
	rc, exists := m.runContexts[runID]
	if !exists {
		rc = NewRunContext(runID, r.PipelineName, nil)
		m.runContexts[runID] = rc
	}
	if rc.Ctx.PipelineName == "" && r.PipelineName != "" {
		rc.Ctx.PipelineName = r.PipelineName
		rc.Pipeline = r.PipelineName
	}
	rc.ApplyEvent(evt)

	m.sortRuns()
}

// initRunContext creates a RunContext from a StartResult with proper step order.
func (m *MissionControlModel) initRunContext(result *StartResult) {
	rc := NewRunContext(result.RunID, "", result.StepOrder)
	for stepID, persona := range result.StepPersonas {
		rc.Ctx.StepPersonas[stepID] = persona
	}
	m.runContexts[result.RunID] = rc
}

// buildRunContextFromStore creates/updates a RunContext from state store data.
func (m *MissionControlModel) buildRunContextFromStore(msg StepDataMsg) {
	rc, exists := m.runContexts[msg.RunID]
	if !exists {
		rc = NewRunContext(msg.RunID, "", nil)
		m.runContexts[msg.RunID] = rc
	}

	// Set pipeline name and start time from run snapshot
	for _, r := range m.runs {
		if r.RunID == msg.RunID {
			rc.Ctx.PipelineName = r.PipelineName
			rc.Pipeline = r.PipelineName
			if !r.StartedAt.IsZero() {
				rc.Ctx.PipelineStartTime = r.StartedAt.UnixNano()
			}
			break
		}
	}

	// Populate from pipeline progress (if available)
	if msg.Progress != nil {
		rc.Ctx.TotalSteps = msg.Progress.TotalSteps
		rc.Ctx.CompletedSteps = msg.Progress.CompletedSteps
		rc.Ctx.CurrentStepNum = msg.Progress.CurrentStepIndex
		rc.Ctx.OverallProgress = msg.Progress.OverallProgress
	}

	// Populate from step progress records (if available)
	for _, sp := range msg.StepData {
		if _, exists := rc.Ctx.StepStatuses[sp.StepID]; !exists {
			rc.Ctx.StepOrder = append(rc.Ctx.StepOrder, sp.StepID)
		}
		switch sp.State {
		case "running":
			rc.Ctx.StepStatuses[sp.StepID] = display.StateRunning
		case "completed":
			rc.Ctx.StepStatuses[sp.StepID] = display.StateCompleted
		case "failed":
			rc.Ctx.StepStatuses[sp.StepID] = display.StateFailed
		default:
			rc.Ctx.StepStatuses[sp.StepID] = display.StateNotStarted
		}
		if sp.Persona != "" {
			rc.Ctx.StepPersonas[sp.StepID] = sp.Persona
		}
		if sp.TokensUsed > 0 {
			rc.Ctx.StepTokens[sp.StepID] = sp.TokensUsed
		}
	}

	// PRIMARY DATA SOURCE: extract step data from event_log
	stepSeen := make(map[string]bool)
	for _, evt := range msg.Events {
		if evt.StepID == "" {
			continue
		}

		if !stepSeen[evt.StepID] {
			stepSeen[evt.StepID] = true
			if _, exists := rc.Ctx.StepStatuses[evt.StepID]; !exists {
				rc.Ctx.StepOrder = append(rc.Ctx.StepOrder, evt.StepID)
				rc.Ctx.StepStatuses[evt.StepID] = display.StateNotStarted
			}
		}

		switch evt.State {
		case "running", "started":
			rc.Ctx.StepStatuses[evt.StepID] = display.StateRunning
		case "completed":
			rc.Ctx.StepStatuses[evt.StepID] = display.StateCompleted
		case "failed":
			rc.Ctx.StepStatuses[evt.StepID] = display.StateFailed
		case "cancelled":
			rc.Ctx.StepStatuses[evt.StepID] = display.StateCancelled
		case "skipped":
			rc.Ctx.StepStatuses[evt.StepID] = display.StateSkipped
		}

		if evt.Persona != "" {
			rc.Ctx.StepPersonas[evt.StepID] = evt.Persona
		}
		if evt.DurationMs > 0 {
			rc.Ctx.StepDurations[evt.StepID] = evt.DurationMs
		}
		if evt.TokensUsed > 0 {
			rc.Ctx.StepTokens[evt.StepID] = evt.TokensUsed
		}
	}

	if rc.Ctx.TotalSteps == 0 {
		rc.Ctx.TotalSteps = len(rc.Ctx.StepOrder)
	}
	rc.recount()
}

// mergeFromStore merges run records from SQLite. Returns IDs of newly added runs.
func (m *MissionControlModel) mergeFromStore(records []storeRecord) []string {
	var newRunIDs []string

	existing := make(map[string]bool, len(m.runs))
	for i := range m.runs {
		existing[m.runs[i].RunID] = true
		if !m.runs[i].Local {
			for _, rec := range records {
				if rec.RunID == m.runs[i].RunID {
					m.runs[i].Status = rec.Status
					m.runs[i].PipelineName = rec.PipelineName
					m.runs[i].CurrentStep = rec.CurrentStep
					m.runs[i].TotalTokens = rec.TotalTokens
					m.runs[i].ErrorMessage = rec.ErrorMessage
					if !rec.StartedAt.IsZero() {
						m.runs[i].StartedAt = rec.StartedAt
						m.runs[i].Elapsed = computeElapsed(rec)
					}
					break
				}
			}
		}
	}

	for _, rec := range records {
		if !existing[rec.RunID] {
			snap := RunSnapshot{
				RunID:        rec.RunID,
				PipelineName: rec.PipelineName,
				Status:       rec.Status,
				CurrentStep:  rec.CurrentStep,
				TotalTokens:  rec.TotalTokens,
				ErrorMessage: rec.ErrorMessage,
				StartedAt:    rec.StartedAt,
				Local:        false,
			}
			snap.Elapsed = computeElapsed(rec)
			m.runs = append(m.runs, snap)
			newRunIDs = append(newRunIDs, rec.RunID)
		}
	}

	// Detect stale runs: non-local "running" runs older than threshold
	now := time.Now()
	for i := range m.runs {
		r := &m.runs[i]
		if !r.Local && (r.Status == "running" || r.Status == "queued" || r.Status == "pending") && !r.StartedAt.IsZero() {
			if now.Sub(r.StartedAt) > staleRunThreshold {
				r.Status = "stale"
			}
		}
	}

	m.sortRuns()
	return newRunIDs
}

// computeElapsed calculates elapsed time from a store record.
func computeElapsed(rec storeRecord) time.Duration {
	if rec.StartedAt.IsZero() {
		return 0
	}
	if rec.Status == "running" {
		return time.Since(rec.StartedAt)
	}
	if rec.CompletedAt != nil && !rec.CompletedAt.IsZero() {
		return rec.CompletedAt.Sub(rec.StartedAt)
	}
	return 0
}

// sortRuns sorts active runs above archived (completed/failed/cancelled) runs,
// then by start time descending within each group.
func (m *MissionControlModel) sortRuns() {
	sort.SliceStable(m.runs, func(i, j int) bool {
		iActive := m.runs[i].isActive()
		jActive := m.runs[j].isActive()
		if iActive != jActive {
			return iActive // active runs sort first
		}
		return m.runs[i].StartedAt.After(m.runs[j].StartedAt)
	})
}

// findRun returns the index of a run by ID, or -1.
func (m *MissionControlModel) findRun(runID string) int {
	for i := range m.runs {
		if m.runs[i].RunID == runID {
			return i
		}
	}
	return -1
}

// visibleRuns returns all runs matching the filter.
func (m *MissionControlModel) visibleRuns() []RunSnapshot {
	return filterRuns(m.runs, m.filter)
}

// selectedRun returns the currently selected run, or nil.
func (m *MissionControlModel) selectedRun() *RunSnapshot {
	visible := m.visibleRuns()
	if m.cursor >= 0 && m.cursor < len(visible) {
		return &visible[m.cursor]
	}
	return nil
}

// moveUp moves the cursor up.
func (m *MissionControlModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.scrollOff {
			m.scrollOff = m.cursor
		}
	}
}

// moveDown moves the cursor down.
func (m *MissionControlModel) moveDown() {
	visible := m.visibleRuns()
	if m.cursor < len(visible)-1 {
		m.cursor++
		maxVisible := m.height - 10
		if maxVisible < 1 {
			maxVisible = 1
		}
		if m.cursor >= m.scrollOff+maxVisible {
			m.scrollOff = m.cursor - maxVisible + 1
		}
	}
}

// moveProposalCursorDown moves the proposal cursor down, skipping skipped proposals.
func (m *MissionControlModel) moveProposalCursorDown() {
	for i := m.proposalCursor + 1; i < len(m.proposals); i++ {
		if !m.proposalSkipped[i] {
			m.proposalCursor = i
			return
		}
	}
}

// moveProposalCursorUp moves the proposal cursor up, skipping skipped proposals.
func (m *MissionControlModel) moveProposalCursorUp() {
	for i := m.proposalCursor - 1; i >= 0; i-- {
		if !m.proposalSkipped[i] {
			m.proposalCursor = i
			return
		}
	}
}

// openChatCmd returns a tea.Cmd that launches wave chat <runID>,
// suspending the TUI while the chat session runs.
func openChatCmd(runID string) tea.Cmd {
	wavePath, err := os.Executable()
	if err != nil {
		wavePath = "wave"
	}
	c := exec.Command(wavePath, "chat", runID)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ChatFinishedMsg{Err: err}
	})
}

// missionTickCmd returns a tea.Cmd that ticks at 200ms for responsive elapsed time updates.
func missionTickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// installDepsCmd returns a tea.Cmd that installs missing dependencies asynchronously.
func installDepsCmd(deps []meta.DependencyStatus) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		installer := meta.NewInstaller()
		results := installer.Install(ctx, deps, nil)
		return InstallResultMsg{Results: results}
	}
}

// newStartedEvent creates a pipeline started event.
func newStartedEvent(pipelineName string) event.Event {
	return event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineName,
		State:      "started",
	}
}

// Run starts the mission control TUI.
func Run(opts Options) error {
	stateDB := ".wave/state.db"
	store, err := state.NewStateStore(stateDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: state persistence disabled: %v\n", err)
		store = nil
	}
	if store != nil {
		defer store.Close()
	}

	model := NewMissionControlModel(opts, store)
	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err = p.Run()
	return err
}

// trimVersion truncates pseudo-version suffixes for display.
// "v0.49.3-0.20260304164656-7c734c30fdf7+dirty" → "v0.49.3"
func trimVersion(v string) string {
	if idx := strings.Index(v, "-0."); idx > 0 {
		v = v[:idx]
	}
	return v
}

// getVersion returns the Wave binary version.
func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
