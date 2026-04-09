package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/pathfmt"
	"github.com/recinq/wave/internal/state"
)

// PipelineDetailModel is the Bubble Tea model for the right pane.
type PipelineDetailModel struct {
	width    int
	height   int
	focused  bool
	viewport viewport.Model

	selectedName      string
	selectedInput     string
	selectedKind      itemKind
	selectedRunID     string
	selectedStartedAt time.Time

	availableDetail *AvailableDetail
	finishedDetail  *FinishedDetail
	branchDeleted   bool

	// State machine for right pane rendering
	paneState   DetailPaneState
	actionError string // Transient error for action keys
	launchError string // Error message for stateError

	// Launch form state — pointers are heap-allocated so huh bindings survive
	// across Bubble Tea's value-copy Update cycles.
	launchForm       *huh.Form
	launchInput      *string   // Bound to form input field
	launchModel      *string   // Bound to form model override field
	launchFlags      *[]string // Bound to form flag multi-select
	launchErrorTitle string    // "Launch Failed" for launch errors, empty for detail load errors

	provider DetailDataProvider

	// Persisted event log (for stale/finished runs)
	persistedEvents []state.LogRecord

	// Live output state
	liveOutput *LiveOutputModel
}

// NewPipelineDetailModel creates a new pipeline detail model with the given provider.
func NewPipelineDetailModel(provider DetailDataProvider) PipelineDetailModel {
	return PipelineDetailModel{
		viewport:  viewport.New(0, 0),
		provider:  provider,
		paneState: stateEmpty,
	}
}

// SetSize updates the model dimensions and re-renders content if data exists.
func (m *PipelineDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if m.launchForm != nil {
		m.launchForm.WithWidth(w).WithHeight(h - 3)
	}
	if m.liveOutput != nil {
		m.liveOutput.SetSize(w, h)
	}
	m.updateViewportContent()
}

// SetFocused updates the focused state.
func (m *PipelineDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// Init implements tea.Model. Returns nil.
func (m PipelineDetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages to update model state.
func (m PipelineDetailModel) Update(msg tea.Msg) (PipelineDetailModel, tea.Cmd) {
	// When the form is active, forward ALL messages to it before any other handler.
	// This ensures the form receives tick messages, key messages, etc.
	if m.paneState == stateConfiguring && m.launchForm != nil {
		model, cmd := m.launchForm.Update(msg)
		m.launchForm = model.(*huh.Form)

		switch m.launchForm.State {
		case huh.StateCompleted:
			// Extract bound values and build LaunchConfig
			config := LaunchConfig{
				PipelineName:  m.selectedName,
				Input:         *m.launchInput,
				ModelOverride: *m.launchModel,
				Flags:         *m.launchFlags,
			}
			// Extract convenience booleans from flags
			for _, f := range config.Flags {
				switch f {
				case "--dry-run":
					config.DryRun = true
				case "--verbose":
					config.Verbose = true
				case "--debug":
					config.Debug = true
				}
			}
			m.paneState = stateLaunching
			m.launchForm = nil
			return m, tea.Batch(cmd, func() tea.Msg {
				return LaunchRequestMsg{Config: config}
			})

		case huh.StateAborted:
			m.launchForm = nil
			m.paneState = stateAvailableDetail
			m.updateViewportContent()
			return m, tea.Batch(cmd, func() tea.Msg {
				return FocusChangedMsg{Pane: FocusPaneLeft}
			}, func() tea.Msg {
				return FormActiveMsg{Active: false}
			})
		}

		// Forward to viewport for scroll support
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		return m, tea.Batch(cmd, vpCmd)
	}

	switch msg := msg.(type) {
	case PipelineSelectedMsg:
		// If re-selecting the same item (e.g. from a periodic refresh tick),
		// preserve expensive state like fetched detail and event logs.
		sameItem := msg.RunID != "" && msg.RunID == m.selectedRunID && msg.Kind == m.selectedKind
		if !sameItem {
			sameItem = msg.RunID == "" && msg.Name == m.selectedName && msg.Kind == m.selectedKind
		}

		m.selectedName = msg.Name
		m.selectedInput = msg.Input
		m.selectedKind = msg.Kind
		m.selectedRunID = msg.RunID
		m.selectedStartedAt = msg.StartedAt
		m.branchDeleted = msg.BranchDeleted
		if !sameItem {
			m.availableDetail = nil
			m.finishedDetail = nil
			m.persistedEvents = nil
		}
		m.launchError = ""
		m.launchErrorTitle = ""
		m.launchForm = nil

		// Pipeline name nodes emit Kind=itemKindAvailable, so no special
		// empty-state handling is needed here (old section headers are gone).

		if msg.Kind == itemKindRunning {
			// Preserve live output if it matches this run (set by PipelineLaunchedMsg)
			if m.liveOutput != nil && m.liveOutput.runID == msg.RunID {
				m.paneState = stateRunningLive
			} else {
				m.liveOutput = nil
				m.paneState = stateRunningInfo
				m.updateViewportContent()
				// Auto-fetch persisted events for stale runs (skip if already loaded)
				if m.provider != nil && m.persistedEvents == nil {
					runID := msg.RunID
					provider := m.provider
					return m, func() tea.Msg {
						events, err := provider.FetchRunEvents(runID)
						return RunEventsMsg{RunID: runID, Events: events, Err: err}
					}
				}
			}
			return m, nil
		}

		// Skip redundant fetch if we already have detail for this item
		if sameItem {
			m.updateViewportContent()
			return m, nil
		}

		m.paneState = stateLoading

		if msg.Kind == itemKindAvailable {
			name := msg.Name
			provider := m.provider
			return m, func() tea.Msg {
				detail, err := provider.FetchAvailableDetail(name)
				return DetailDataMsg{AvailableDetail: detail, Err: err}
			}
		}

		if msg.Kind == itemKindFinished {
			runID := msg.RunID
			provider := m.provider
			return m, func() tea.Msg {
				detail, err := provider.FetchFinishedDetail(runID)
				return DetailDataMsg{FinishedDetail: detail, Err: err}
			}
		}

	case DetailDataMsg:
		switch {
		case msg.Err != nil:
			m.launchError = msg.Err.Error()
			m.launchErrorTitle = ""
			m.paneState = stateError
		case msg.AvailableDetail != nil:
			m.availableDetail = msg.AvailableDetail
			m.paneState = stateAvailableDetail
		case msg.FinishedDetail != nil:
			m.finishedDetail = msg.FinishedDetail
			m.branchDeleted = msg.FinishedDetail.BranchDeleted
			m.paneState = stateFinishedDetail
		}
		m.updateViewportContent()
		m.viewport.GotoTop()
		return m, nil

	case ConfigureFormMsg:
		// Allocate fresh heap pointers so huh bindings survive value-copy Update cycles.
		m.launchInput = new(string)
		m.launchModel = new(string)
		m.launchFlags = new([]string)

		// Create the form with input, model override, and flag fields
		m.launchForm = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Input").
					Placeholder(msg.InputExample).
					Value(m.launchInput),
				huh.NewInput().
					Title("Model override (optional)").
					Value(m.launchModel),
				huh.NewMultiSelect[string]().
					Title("Options").
					Options(buildFlagOptions(DefaultFlags())...).
					Value(m.launchFlags),
			),
		).WithTheme(WaveTheme()).WithWidth(m.width).WithHeight(m.height - 3)

		m.paneState = stateConfiguring
		m.viewport.SetContent(m.launchForm.View())
		m.viewport.GotoTop()
		return m, m.launchForm.Init()

	case LaunchErrorMsg:
		m.launchError = msg.Err.Error()
		m.launchErrorTitle = "Launch Failed"
		m.paneState = stateError
		m.launchForm = nil
		return m, nil

	case PipelineEventMsg:
		if m.paneState == stateRunningLive && m.liveOutput != nil && msg.RunID == m.liveOutput.runID {
			var cmd tea.Cmd
			*m.liveOutput, cmd = m.liveOutput.Update(msg)
			return m, cmd
		}
		return m, nil

	case DashboardTickMsg:
		if m.paneState == stateRunningLive && m.liveOutput != nil {
			var cmd tea.Cmd
			*m.liveOutput, cmd = m.liveOutput.Update(msg)
			return m, cmd
		}
		return m, nil

	case TransitionTimerMsg:
		if m.paneState == stateRunningLive && m.liveOutput != nil && msg.RunID == m.liveOutput.runID {
			// Transition to loading state to fetch finished detail
			m.paneState = stateLoading
			m.liveOutput = nil
			runID := m.selectedRunID
			provider := m.provider
			return m, tea.Batch(
				func() tea.Msg { return LiveOutputActiveMsg{Active: false} },
				func() tea.Msg {
					detail, err := provider.FetchFinishedDetail(runID)
					return DetailDataMsg{FinishedDetail: detail, Err: err}
				},
			)
		}
		return m, nil

	case ChatSessionEndedMsg:
		// Re-fetch finished detail to reflect changes made during chat
		if m.selectedRunID != "" {
			runID := m.selectedRunID
			provider := m.provider
			return m, tea.Batch(
				func() tea.Msg {
					detail, err := provider.FetchFinishedDetail(runID)
					return DetailDataMsg{FinishedDetail: detail, Err: err}
				},
				func() tea.Msg { return GitRefreshTickMsg{} },
			)
		}
		return m, nil

	case BranchCheckoutMsg:
		if msg.Success {
			m.actionError = ""
			return m, func() tea.Msg { return GitRefreshTickMsg{} }
		}
		if msg.Err != nil {
			m.actionError = fmt.Sprintf("Branch checkout failed: %s", msg.Err)
		}
		m.updateViewportContent()
		return m, nil

	case RunEventsMsg:
		if msg.Err == nil {
			m.persistedEvents = msg.Events
			m.updateViewportContent()
		}
		return m, nil

	case DiffViewEndedMsg:
		return m, nil

	case tea.KeyMsg:
		// Handle Esc from live output state
		if m.paneState == stateRunningLive && msg.Type == tea.KeyEscape {
			m.liveOutput = nil
			m.paneState = stateRunningInfo
			m.updateViewportContent()
			return m, tea.Batch(
				func() tea.Msg { return LiveOutputActiveMsg{Active: false} },
				func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} },
			)
		}

		// Handle Esc from error state
		if m.paneState == stateError && msg.Type == tea.KeyEscape {
			m.paneState = stateAvailableDetail
			m.launchError = ""
			m.launchErrorTitle = ""
			m.updateViewportContent()
			return m, func() tea.Msg {
				return FocusChangedMsg{Pane: FocusPaneLeft}
			}
		}

		// Forward keys to liveOutput when in stateRunningLive
		if m.paneState == stateRunningLive && m.liveOutput != nil && m.focused {
			var cmd tea.Cmd
			*m.liveOutput, cmd = m.liveOutput.Update(msg)
			return m, cmd
		}

		// Handle 'l' key for event logs in stateRunningInfo
		if m.paneState == stateRunningInfo && m.focused {
			if msg.String() == "l" && m.provider != nil && m.selectedRunID != "" {
				runID := m.selectedRunID
				provider := m.provider
				return m, func() tea.Msg {
					events, err := provider.FetchRunEvents(runID)
					return RunEventsMsg{RunID: runID, Events: events, Err: err}
				}
			}
		}

		// Handle action keys in stateFinishedDetail
		if m.paneState == stateFinishedDetail && m.focused {
			// Clear transient error on any key press (T021)
			m.actionError = ""

			switch msg.Type {
			case tea.KeyEnter:
				// Open chat session (T012)
				if m.finishedDetail == nil || m.finishedDetail.WorkspacePath == "" {
					m.actionError = "Workspace directory no longer exists — the worktree may have been cleaned up"
					m.updateViewportContent()
					return m, nil
				}
				cmd := exec.Command("claude")
				cmd.Dir = m.finishedDetail.WorkspacePath
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
					return ChatSessionEndedMsg{Err: err}
				})
			default:
				switch msg.String() {
				case "b":
					// Branch checkout (T015)
					if m.branchDeleted || m.finishedDetail == nil || m.finishedDetail.BranchName == "" {
						return m, nil
					}
					branch := m.finishedDetail.BranchName
					return m, func() tea.Msg {
						out, err := exec.Command("git", "checkout", branch).CombinedOutput()
						if err != nil {
							return BranchCheckoutMsg{BranchName: branch, Success: false, Err: fmt.Errorf("%s", strings.TrimSpace(string(out)))}
						}
						return BranchCheckoutMsg{BranchName: branch, Success: true}
					}
				case "l":
					// Fetch event logs
					if m.provider != nil && m.selectedRunID != "" {
						runID := m.selectedRunID
						provider := m.provider
						return m, func() tea.Msg {
							events, err := provider.FetchRunEvents(runID)
							return RunEventsMsg{RunID: runID, Events: events, Err: err}
						}
					}
				case "d":
					// Diff view (T018)
					if m.branchDeleted || m.finishedDetail == nil || m.finishedDetail.BranchName == "" {
						return m, nil
					}
					diffCmd := exec.Command("git", "diff", "main..."+m.finishedDetail.BranchName)
					return m, tea.ExecProcess(diffCmd, func(err error) tea.Msg {
						return DiffViewEndedMsg{Err: err}
					})
				}
			}
			m.updateViewportContent()
		}

		if m.focused {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// updateViewportContent re-renders the appropriate content and sets it on the viewport.
func (m *PipelineDetailModel) updateViewportContent() {
	switch m.paneState {
	case stateAvailableDetail:
		if m.availableDetail != nil {
			m.viewport.SetContent(renderAvailableDetail(m.availableDetail, m.width))
		}
	case stateFinishedDetail:
		if m.finishedDetail != nil {
			m.viewport.SetContent(renderFinishedDetail(m.finishedDetail, m.width, m.branchDeleted, m.actionError, m.persistedEvents))
		}
	case stateRunningInfo:
		if m.selectedName != "" {
			m.viewport.SetContent(renderRunningInfo(m.selectedName, m.selectedInput, m.selectedStartedAt, m.width, m.persistedEvents))
		}
	}
}

// View renders the detail pane.
func (m PipelineDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	switch m.paneState {
	case stateEmpty:
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Select a pipeline to view details")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)

	case stateLoading:
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Loading...")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)

	case stateConfiguring:
		if m.launchForm != nil {
			var header string
			if m.availableDetail != nil {
				header = lipgloss.NewStyle().Bold(true).Render("Pipeline: " + m.availableDetail.Name)
			}

			formView := m.launchForm.View()

			var content string
			if header != "" {
				content = header + "\n\n" + formView
			} else {
				content = formView
			}

			m.viewport.SetContent(content)
			return m.viewport.View()
		}

	case stateLaunching:
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Starting pipeline...")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)

	case stateError:
		if m.launchError != "" {
			redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
			mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
			var content string
			if m.launchErrorTitle != "" {
				content = redStyle.Bold(true).Render(m.launchErrorTitle) + "\n\n" +
					redStyle.Render(m.launchError) + "\n\n" +
					mutedStyle.Render("[Esc] Back")
			} else {
				content = redStyle.Render(fmt.Sprintf("Failed to load pipeline details: %s", m.launchError))
			}
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
		}

	case stateRunningLive:
		if m.liveOutput != nil {
			return m.liveOutput.View()
		}
	}

	return m.viewport.View()
}

// renderAvailableDetail renders the detail view for an available pipeline.
func renderAvailableDetail(detail *AvailableDetail, width int) string {
	_ = width
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder

	sb.WriteString(labelStyle.Render("Pipeline: "))
	sb.WriteString(titleStyle.Render(detail.Name))
	sb.WriteString("\n")

	if detail.Description != "" {
		sb.WriteString("\n")
		sb.WriteString(detail.Description)
		sb.WriteString("\n")
	}

	if detail.Category != "" {
		sb.WriteString(fmt.Sprintf("\n%s %s\n", labelStyle.Render("Category:"), detail.Category))
	}

	if detail.StepCount > 0 || len(detail.Steps) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render(fmt.Sprintf("Steps (%d):", detail.StepCount)))
		sb.WriteString("\n")
		for i, step := range detail.Steps {
			if step.Persona != "" {
				fmt.Fprintf(&sb, "  %d. %s (%s)\n", i+1, step.ID, step.Persona)
			} else {
				fmt.Fprintf(&sb, "  %d. %s\n", i+1, step.ID)
			}
		}
	}

	if detail.InputSource != "" || detail.InputExample != "" {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Input:"))
		sb.WriteString("\n")
		if detail.InputSource != "" {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Source:"), detail.InputSource))
		}
		if detail.InputExample != "" {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Example:"), detail.InputExample))
		}
	}

	if len(detail.Artifacts) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Artifacts:"))
		sb.WriteString("\n")
		for _, a := range detail.Artifacts {
			sb.WriteString(fmt.Sprintf("  • %s\n", a))
		}
	}

	if len(detail.Skills) > 0 || len(detail.Tools) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Dependencies:"))
		sb.WriteString("\n")
		if len(detail.Skills) > 0 {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Skills:"), strings.Join(detail.Skills, ", ")))
		}
		if len(detail.Tools) > 0 {
			sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Tools:"), strings.Join(detail.Tools, ", ")))
		}
	}

	return sb.String()
}

// renderFinishedDetail renders the detail view for a finished pipeline run.
func renderFinishedDetail(detail *FinishedDetail, width int, branchDeleted bool, actionError string, events []state.LogRecord) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder

	sb.WriteString(labelStyle.Render("Pipeline: "))
	sb.WriteString(titleStyle.Render(detail.Name))
	sb.WriteString("\n\n")

	// Status
	var statusStr string
	switch detail.Status {
	case "completed":
		statusStr = greenStyle.Render("\u2713 completed")
	case "failed":
		statusStr = redStyle.Render("\u2717 failed")
	case "cancelled":
		statusStr = yellowStyle.Render("\u2717 cancelled")
	default:
		statusStr = detail.Status
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), statusStr))

	// Input
	if detail.Input != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Input:"), detail.Input))
	}

	// Duration
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Duration:"), formatDuration(detail.Duration)))

	// Branch
	if detail.BranchName != "" {
		branchName := detail.BranchName
		if branchDeleted {
			branchName = mutedStyle.Render(branchName + " (deleted)")
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Branch:"), branchName))
	}

	// Times
	if !detail.StartedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Started:"), detail.StartedAt.Format("2006-01-02 15:04:05")))
	}
	if !detail.CompletedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Finished:"), detail.CompletedAt.Format("2006-01-02 15:04:05")))
	}

	// Error info
	if detail.ErrorMessage != "" {
		sb.WriteString("\n")
		errWidth := width - 4
		if errWidth < 10 {
			errWidth = 10
		}
		sb.WriteString(redStyle.Render("Error: "))
		sb.WriteString(redStyle.Width(errWidth).Render(detail.ErrorMessage))
		sb.WriteString("\n")
	}
	if detail.FailedStep != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Failed step:"), detail.FailedStep))
	}

	// Steps
	if len(detail.Steps) > 0 {
		orangeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
		purpleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Steps:"))
		sb.WriteString("\n")
		for _, step := range detail.Steps {
			var iconStr string
			switch step.Status {
			case "completed":
				iconStr = greenStyle.Render("\u2713")
			case "failed":
				iconStr = redStyle.Render("\u2717")
			default:
				iconStr = mutedStyle.Render("\u2014")
			}
			var failureTag string
			if step.FailureClass != "" {
				switch step.FailureClass {
				case "transient":
					failureTag = " " + yellowStyle.Render("["+step.FailureClass+"]")
				case "deterministic":
					failureTag = " " + redStyle.Render("["+step.FailureClass+"]")
				case "contract_failure":
					failureTag = " " + orangeStyle.Render("["+step.FailureClass+"]")
				case "test_failure":
					failureTag = " " + purpleStyle.Render("["+step.FailureClass+"]")
				default:
					failureTag = " " + mutedStyle.Render("["+step.FailureClass+"]")
				}
			}
			if step.Persona != "" {
				fmt.Fprintf(&sb, "  %s %-20s  %s  (%s)%s\n",
					iconStr,
					step.ID,
					formatDuration(step.Duration),
					step.Persona,
					failureTag,
				)
			} else {
				fmt.Fprintf(&sb, "  %s %-20s  %s%s\n",
					iconStr,
					step.ID,
					formatDuration(step.Duration),
					failureTag,
				)
			}
		}
	}

	// Artifacts
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("Artifacts:"))
	sb.WriteString("\n")
	if len(detail.Artifacts) == 0 {
		sb.WriteString(fmt.Sprintf("  %s\n", mutedStyle.Render("No artifacts produced")))
	} else {
		for _, a := range detail.Artifacts {
			displayPath := a.Path
			if displayPath == "" {
				displayPath = a.Name
			}
			displayPath = pathfmt.FileURI(toAbsPath(displayPath))
			sb.WriteString(fmt.Sprintf("  %s\n", displayPath))
		}
	}

	// Event log (if loaded)
	if len(events) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Event Log:"))
		sb.WriteString("\n")
		for _, ev := range events {
			sb.WriteString(formatLogRecord(ev))
			sb.WriteString("\n")
		}
	}

	// Action hints
	sb.WriteString("\n")
	if actionError != "" {
		sb.WriteString(redStyle.Render(actionError))
		sb.WriteString("\n")
	} else {
		branchDisabled := branchDeleted || detail.BranchName == ""
		enterHint := mutedStyle.Render("[Enter] Open chat")
		if detail.WorkspacePath == "" {
			enterHint = mutedStyle.Faint(true).Render("[Enter] Open chat")
		}
		branchHint := mutedStyle.Render("[b] Checkout branch")
		if branchDisabled {
			branchHint = mutedStyle.Faint(true).Render("[b] Checkout branch")
		}
		diffHint := mutedStyle.Render("[d] View diff")
		if branchDisabled {
			diffHint = mutedStyle.Faint(true).Render("[d] View diff")
		}
		logsHint := mutedStyle.Render("[l] Logs")
		escHint := mutedStyle.Render("[Esc] Back")
		sb.WriteString(fmt.Sprintf("%s  %s  %s  %s  %s\n", enterHint, branchHint, diffHint, logsHint, escHint))
	}

	return sb.String()
}

// renderRunningInfo renders a brief info view for a running pipeline.
func renderRunningInfo(name string, input string, startedAt time.Time, width int, events []state.LogRecord) string {
	_ = width
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render(name))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), greenStyle.Render("\u25b6 Running")))
	if input != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Input:"), input))
	}
	if !startedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Started:"), startedAt.Format("2006-01-02 15:04:05")))
	}
	sb.WriteString("\n")
	sb.WriteString(labelStyle.Render("Press [Enter] to view live event dashboard from persisted events."))
	sb.WriteString("\n")
	sb.WriteString(labelStyle.Render("Use [c] to cancel or dismiss this run."))

	if len(events) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString(sectionStyle.Render("Event Log:"))
		sb.WriteString("\n")
		for _, ev := range events {
			sb.WriteString(formatLogRecord(ev))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// toAbsPath converts a relative path to an absolute path.
// Already-absolute paths are returned unchanged.
func toAbsPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// formatLogRecord formats a single persisted log record for display.
func formatLogRecord(rec state.LogRecord) string {
	ts := rec.Timestamp.Format("15:04:05")
	stepID := rec.StepID
	if stepID == "" {
		stepID = "pipeline"
	}
	if rec.Message != "" {
		return fmt.Sprintf("  %s [%s] %s: %s", ts, rec.State, stepID, rec.Message)
	}
	return fmt.Sprintf("  %s [%s] %s", ts, rec.State, stepID)
}
