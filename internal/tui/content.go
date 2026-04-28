package tui

import (
	"strings"

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

// childHeight returns the usable height for child models (minus top and bottom padding lines).
func (m ContentModel) childHeight() int {
	if m.height <= 2 {
		return 0
	}
	return m.height - 2
}

// SetSize updates the content area dimensions and propagates to children.
func (m *ContentModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	leftWidth := m.leftPaneWidth()
	rightWidth := w - leftWidth - 3 // 3 chars for separator: space + │ + space
	ch := m.childHeight()

	m.setSizePipeline(leftWidth, rightWidth, ch)
	m.setSizePersona(leftWidth, rightWidth, ch)
	m.setSizeContract(leftWidth, rightWidth, ch)
	m.setSizeSkill(leftWidth, rightWidth, ch)
	m.setSizeHealth(leftWidth, rightWidth, ch)
	m.setSizeIssue(leftWidth, rightWidth, ch)
	m.setSizePR(leftWidth, rightWidth, ch)
	m.setSizeSuggest(leftWidth, rightWidth, ch)
	m.setSizeOntology(leftWidth, rightWidth, ch)
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

		// Pipeline-specific key handling (Enter, Escape, 'c' cancel, 's' compose).
		if mm, cmd, handled := m.handlePipelineKeyMsg(msg); handled {
			return mm, cmd
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
	}

	// Compose-mode messages
	if mm, cmd, handled := m.updateComposeMessage(msg); handled {
		return mm, cmd
	}

	// Per-panel message routing
	if mm, cmd, handled := m.updatePersonaMessage(msg); handled {
		return mm, cmd
	}
	if mm, cmd, handled := m.updateContractMessage(msg); handled {
		return mm, cmd
	}
	if mm, cmd, handled := m.updateSkillMessage(msg); handled {
		return mm, cmd
	}
	if mm, cmd, handled := m.updateHealthMessage(msg); handled {
		return mm, cmd
	}
	if mm, cmd, handled := m.updateIssueMessage(msg); handled {
		return mm, cmd
	}
	if mm, cmd, handled := m.updatePRMessage(msg); handled {
		return mm, cmd
	}
	if mm, cmd, handled := m.updateSuggestMessage(msg); handled {
		return mm, cmd
	}
	if mm, cmd, handled := m.updateOntologyMessage(msg); handled {
		return mm, cmd
	}
	if mm, cmd, handled := m.updatePipelineMessage(msg); handled {
		return mm, cmd
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

// View renders the content area with left list and right detail pane.
func (m ContentModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	var leftView, rightView string

	switch m.currentView {
	case ViewPipelines:
		leftView, rightView = m.viewPipeline()
	case ViewPersonas:
		leftView, rightView = m.viewPersona()
	case ViewContracts:
		leftView, rightView = m.viewContract()
	case ViewSkills:
		leftView, rightView = m.viewSkill()
	case ViewHealth:
		leftView, rightView = m.viewHealth()
	case ViewIssues:
		leftView, rightView = m.viewIssue()
	case ViewPullRequests:
		leftView, rightView = m.viewPR()
	case ViewSuggest:
		leftView, rightView = m.viewSuggest()
	case ViewOntology:
		leftView, rightView = m.viewOntology()
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
