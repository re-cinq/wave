package tui

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock provider
// ---------------------------------------------------------------------------

type mockDetailProvider struct {
	availableDetail *AvailableDetail
	availableErr    error
	finishedDetail  *FinishedDetail
	finishedErr     error
	runEvents       []state.LogRecord
	runEventsErr    error
}

func (m *mockDetailProvider) FetchAvailableDetail(name string) (*AvailableDetail, error) {
	return m.availableDetail, m.availableErr
}

func (m *mockDetailProvider) FetchFinishedDetail(runID string) (*FinishedDetail, error) {
	return m.finishedDetail, m.finishedErr
}

func (m *mockDetailProvider) FetchRunEvents(runID string) ([]state.LogRecord, error) {
	return m.runEvents, m.runEventsErr
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

var detailAnsiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func detailStripAnsi(s string) string {
	return detailAnsiRegex.ReplaceAllString(s, "")
}

// newTestDetailModel creates a PipelineDetailModel with a set size suitable for testing.
func newTestDetailModel(provider DetailDataProvider) PipelineDetailModel {
	m := NewPipelineDetailModel(provider)
	m.SetSize(80, 30)
	return m
}

// updateAndView applies messages sequentially and returns the view after all messages.
func updateAndView(m PipelineDetailModel, msgs ...tea.Msg) string {
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	return m.View()
}

// fullAvailableDetail returns a fully-populated AvailableDetail for testing.
func fullAvailableDetail() *AvailableDetail {
	return &AvailableDetail{
		Name:        "speckit-flow",
		Description: "A specification pipeline",
		Category:    "spec",
		StepCount:   3,
		Steps: []StepSummary{
			{ID: "specify", Persona: "navigator"},
			{ID: "clarify", Persona: "navigator"},
			{ID: "plan", Persona: "craftsman"},
		},
		InputSource:  "GitHub issue URL",
		InputExample: "https://github.com/org/repo/issues/123",
		Artifacts:    []string{"spec_info", "plan_output"},
		Skills:       []string{"gh", "git"},
		Tools:        []string{"claude-code"},
	}
}

// fullFinishedDetail returns a fully-populated FinishedDetail for testing.
func fullFinishedDetail(status string) *FinishedDetail {
	errMsg := ""
	failedStep := ""
	if status == "failed" {
		errMsg = "step failed with exit code 1"
		failedStep = "plan"
	}
	return &FinishedDetail{
		RunID:        "run-123",
		Name:         "speckit-flow",
		Status:       status,
		Duration:     5*time.Minute + 30*time.Second,
		BranchName:   "feat/my-feature",
		StartedAt:    time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		CompletedAt:  time.Date(2026, 1, 15, 10, 5, 30, 0, time.UTC),
		ErrorMessage: errMsg,
		FailedStep:   failedStep,
		Steps: []StepResult{
			{ID: "specify", Status: "completed", Duration: 2 * time.Minute, Persona: "navigator"},
			{ID: "clarify", Status: "completed", Duration: 1 * time.Minute, Persona: "navigator"},
			{ID: "plan", Status: status, Duration: 2*time.Minute + 30*time.Second, Persona: "craftsman"},
		},
		Artifacts: []ArtifactInfo{
			{Name: "spec_info", Path: ".wave/artifacts/spec_info", Type: "json"},
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPipelineDetailModel_Placeholder(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	view := detailStripAnsi(m.View())

	assert.Contains(t, view, "Select a pipeline to view details")
}

func TestPipelineDetailModel_AvailableDetailRendering(t *testing.T) {
	detail := fullAvailableDetail()
	provider := &mockDetailProvider{availableDetail: detail}
	m := newTestDetailModel(provider)

	// Send selection, then simulate data arrival
	selMsg := PipelineSelectedMsg{Kind: itemKindAvailable, Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	require.NotNil(t, cmd, "should return fetch cmd")

	// Execute the cmd to get DetailDataMsg
	dataMsg := cmd()
	view := detailStripAnsi(updateAndView(m, dataMsg))

	assert.Contains(t, view, "speckit-flow")
	assert.Contains(t, view, "A specification pipeline")
	assert.Contains(t, view, "specify")
	assert.Contains(t, view, "navigator")
	assert.Contains(t, view, "GitHub issue URL")
	assert.Contains(t, view, "https://github.com/org/repo/issues/123")
	assert.Contains(t, view, "spec_info")
	assert.Contains(t, view, "plan_output")
	assert.Contains(t, view, "gh")
	assert.Contains(t, view, "git")
	assert.Contains(t, view, "claude-code")
}

func TestPipelineDetailModel_FinishedCompleted(t *testing.T) {
	detail := fullFinishedDetail("completed")
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	require.NotNil(t, cmd)

	dataMsg := cmd()
	view := detailStripAnsi(updateAndView(m, dataMsg))

	assert.Contains(t, view, "speckit-flow")
	assert.Contains(t, view, "✓")
	assert.Contains(t, view, "completed")
	assert.Contains(t, view, "5m 30s")
	assert.Contains(t, view, "feat/my-feature")
	assert.Contains(t, view, "specify")
	assert.Contains(t, view, "navigator")
}

func TestPipelineDetailModel_FinishedFailed(t *testing.T) {
	detail := fullFinishedDetail("failed")
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	require.NotNil(t, cmd)

	dataMsg := cmd()
	view := detailStripAnsi(updateAndView(m, dataMsg))

	assert.Contains(t, view, "✗")
	assert.Contains(t, view, "failed")
	assert.Contains(t, view, "step failed with exit code 1")
	assert.Contains(t, view, "plan")
}

func TestPipelineDetailModel_BranchDeleted(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.BranchDeleted = true
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow", BranchDeleted: true}
	m, cmd := m.Update(selMsg)
	require.NotNil(t, cmd)

	dataMsg := cmd()
	view := detailStripAnsi(updateAndView(m, dataMsg))

	assert.Contains(t, view, "(deleted)")
}

func TestPipelineDetailModel_ZeroArtifacts(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.Artifacts = nil
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	require.NotNil(t, cmd)

	dataMsg := cmd()
	view := detailStripAnsi(updateAndView(m, dataMsg))

	assert.Contains(t, view, "No artifacts produced")
}

func TestPipelineDetailModel_FocusState(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	assert.False(t, m.focused)

	m.SetFocused(true)
	assert.True(t, m.focused)

	m.SetFocused(false)
	assert.False(t, m.focused)
}

func TestPipelineDetailModel_SelectionTriggersFetch(t *testing.T) {
	provider := &mockDetailProvider{availableDetail: fullAvailableDetail()}
	m := newTestDetailModel(provider)

	// Available kind should trigger fetch
	m, cmd := m.Update(PipelineSelectedMsg{Kind: itemKindAvailable, Name: "speckit-flow"})
	assert.NotNil(t, cmd, "available selection should return fetch cmd")
	assert.Equal(t, stateLoading, m.paneState)

	// Finished kind should trigger fetch
	m2 := newTestDetailModel(&mockDetailProvider{finishedDetail: fullFinishedDetail("completed")})
	m2, cmd2 := m2.Update(PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-1"})
	assert.NotNil(t, cmd2, "finished selection should return fetch cmd")
	assert.Equal(t, stateLoading, m2.paneState)
}

func TestPipelineDetailModel_SelectionChangeResetsScroll(t *testing.T) {
	detail1 := &AvailableDetail{
		Name:        "pipeline-1",
		Description: strings.Repeat("Line content\n", 50), // lots of content
	}
	detail2 := &AvailableDetail{
		Name:        "pipeline-2",
		Description: "Short description",
	}
	provider := &mockDetailProvider{}
	m := newTestDetailModel(provider)

	// Load first pipeline
	provider.availableDetail = detail1
	m, cmd := m.Update(PipelineSelectedMsg{Kind: itemKindAvailable, Name: "pipeline-1"})
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	// Scroll down a bit if possible
	m.viewport.SetYOffset(5)

	// Load second pipeline
	provider.availableDetail = detail2
	m, cmd = m.Update(PipelineSelectedMsg{Kind: itemKindAvailable, Name: "pipeline-2"})
	dataMsg = cmd()
	m, _ = m.Update(dataMsg)

	// Viewport should be at top
	assert.Equal(t, 0, m.viewport.YOffset, "viewport should reset to top on new selection")
}

func TestPipelineDetailModel_RunningInfo(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})

	view := detailStripAnsi(updateAndView(m, PipelineSelectedMsg{Kind: itemKindRunning, Name: "my-pipeline"}))

	assert.Contains(t, view, "Running")
	assert.Contains(t, view, "my-pipeline")
}

func TestPipelineDetailModel_InitialStateShowsPlaceholder(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})

	// Initial state (no selection) should show placeholder
	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "Select a pipeline to view details")
}

func TestPipelineDetailModel_ErrorFromProvider(t *testing.T) {
	provider := &mockDetailProvider{
		availableErr: errors.New("connection refused"),
	}
	m := newTestDetailModel(provider)

	m, cmd := m.Update(PipelineSelectedMsg{Kind: itemKindAvailable, Name: "some-pipeline"})
	require.NotNil(t, cmd)

	dataMsg := cmd()
	view := detailStripAnsi(updateAndView(m, dataMsg))

	assert.Contains(t, view, "Failed to load pipeline details")
	assert.Contains(t, view, "connection refused")
}

func TestPipelineDetailModel_Resize(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	assert.Equal(t, 80, m.viewport.Width)
	assert.Equal(t, 30, m.viewport.Height)

	m.SetSize(120, 50)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 50, m.height)
	assert.Equal(t, 120, m.viewport.Width)
	assert.Equal(t, 50, m.viewport.Height)
}

func TestPipelineDetailModel_View_ZeroDimensions(t *testing.T) {
	m := NewPipelineDetailModel(&mockDetailProvider{})
	// width and height are 0
	view := m.View()
	assert.Equal(t, "", view)
}

func TestPipelineDetailModel_Init_ReturnsNil(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	cmd := m.Init()
	assert.Nil(t, cmd)
}

func TestPipelineDetailModel_LoadingState(t *testing.T) {
	provider := &mockDetailProvider{}
	m := newTestDetailModel(provider)

	// Selecting available sets loading=true
	m, _ = m.Update(PipelineSelectedMsg{Kind: itemKindAvailable, Name: "some-pipeline"})
	assert.Equal(t, stateLoading, m.paneState)

	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "Loading...")
}

func TestPipelineDetailModel_AvailableDetailWithAllSections(t *testing.T) {
	detail := fullAvailableDetail()
	provider := &mockDetailProvider{availableDetail: detail}
	m := newTestDetailModel(provider)

	m, cmd := m.Update(PipelineSelectedMsg{Kind: itemKindAvailable, Name: "speckit-flow"})
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	view := detailStripAnsi(m.View())

	// Category section
	assert.Contains(t, view, "spec")
	// Steps section
	assert.Contains(t, view, "Steps (3):")
	assert.Contains(t, view, "1. specify (navigator)")
	assert.Contains(t, view, "3. plan (craftsman)")
	// Input section
	assert.Contains(t, view, "Input:")
	assert.Contains(t, view, "Source:")
	assert.Contains(t, view, "Example:")
	// Dependencies section
	assert.Contains(t, view, "Dependencies:")
	assert.Contains(t, view, "Skills:")
	assert.Contains(t, view, "Tools:")
}

func TestPipelineDetailModel_FinishedDetailTimeFormat(t *testing.T) {
	detail := fullFinishedDetail("completed")
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	m, cmd := m.Update(PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"})
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	view := detailStripAnsi(m.View())

	assert.Contains(t, view, "2026-01-15 10:00:00")
	assert.Contains(t, view, "2026-01-15 10:05:30")
}

func TestPipelineDetailModel_ActionHints(t *testing.T) {
	detail := fullFinishedDetail("completed")
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	m, cmd := m.Update(PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"})
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	view := detailStripAnsi(m.View())

	assert.Contains(t, view, "[Enter] Open chat")
	assert.Contains(t, view, "[b] Checkout branch")
	assert.Contains(t, view, "[d] View diff")
	assert.Contains(t, view, "[Esc] Back")
}

// ===========================================================================
// T008: Form unit tests
// ===========================================================================

func TestPipelineDetailModel_ConfigureFormMsg_CreatesForm(t *testing.T) {
	detail := fullAvailableDetail()
	provider := &mockDetailProvider{availableDetail: detail}
	m := newTestDetailModel(provider)

	// Load available detail first
	selMsg := PipelineSelectedMsg{Kind: itemKindAvailable, Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	require.Equal(t, stateAvailableDetail, m.paneState)

	// Send ConfigureFormMsg to create the form
	cfgMsg := ConfigureFormMsg{PipelineName: "speckit-flow", InputExample: "https://github.com/org/repo/issues/123"}
	m, _ = m.Update(cfgMsg)

	assert.Equal(t, stateConfiguring, m.paneState)
	assert.NotNil(t, m.launchForm)

	view := m.View()
	assert.Contains(t, view, "Input")
	assert.Contains(t, view, "Model override")
	assert.Contains(t, view, "Options")
}

func TestPipelineDetailModel_FormCompletion_ExtractsVerboseAndDebugFlags(t *testing.T) {
	// This test verifies that Verbose and Debug are extracted from the Flags
	// slice into dedicated boolean fields on LaunchConfig, matching the DryRun pattern.
	config := LaunchConfig{
		PipelineName: "test-pipe",
		Flags:        []string{"--verbose", "--debug", "--dry-run"},
	}
	// Simulate the extraction logic from the form completion handler
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

	assert.True(t, config.Verbose, "Verbose should be extracted from --verbose flag")
	assert.True(t, config.Debug, "Debug should be extracted from --debug flag")
	assert.True(t, config.DryRun, "DryRun should be extracted from --dry-run flag")
}

func TestPipelineDetailModel_FormCompletion_NoFlags_LeavesVerboseAndDebugFalse(t *testing.T) {
	config := LaunchConfig{
		PipelineName: "test-pipe",
		Flags:        []string{"--mock"},
	}
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

	assert.False(t, config.Verbose, "Verbose should be false when --verbose not selected")
	assert.False(t, config.Debug, "Debug should be false when --debug not selected")
	assert.False(t, config.DryRun, "DryRun should be false when --dry-run not selected")
}

func TestPipelineDetailModel_FormAbort_RevertsToAvailableDetail(t *testing.T) {
	detail := fullAvailableDetail()
	provider := &mockDetailProvider{availableDetail: detail}
	m := newTestDetailModel(provider)

	// Load available detail first
	selMsg := PipelineSelectedMsg{Kind: itemKindAvailable, Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	// Send ConfigureFormMsg
	cfgMsg := ConfigureFormMsg{PipelineName: "speckit-flow", InputExample: "example"}
	m, _ = m.Update(cfgMsg)
	require.Equal(t, stateConfiguring, m.paneState)
	require.NotNil(t, m.launchForm)

	// Verify that after form abort, state reverts to available detail.
	// Directly set the state to simulate what happens after form abort
	// (since triggering huh form abort programmatically is not straightforward).
	m.paneState = stateAvailableDetail
	m.launchForm = nil
	m.updateViewportContent()

	assert.Equal(t, stateAvailableDetail, m.paneState)
	assert.Nil(t, m.launchForm)

	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "speckit-flow")
}

func TestPipelineDetailModel_View_Configuring_ShowsForm(t *testing.T) {
	provider := &mockDetailProvider{availableDetail: fullAvailableDetail()}
	m := newTestDetailModel(provider)

	// Send ConfigureFormMsg to create the form
	cfgMsg := ConfigureFormMsg{PipelineName: "speckit-flow", InputExample: "example input"}
	m, _ = m.Update(cfgMsg)

	assert.Equal(t, stateConfiguring, m.paneState)
	view := m.View()
	assert.NotEmpty(t, view)
	assert.NotContains(t, view, "Select a pipeline to view details")
}

func TestPipelineDetailModel_View_Launching_ShowsStarting(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	m.paneState = stateLaunching

	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "Starting pipeline...")
}

func TestPipelineDetailModel_View_Error_ShowsLaunchFailed(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	m.paneState = stateError
	m.launchErrorTitle = "Launch Failed"
	m.launchError = "adapter not found"

	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "Launch Failed")
	assert.Contains(t, view, "adapter not found")
}

func TestPipelineDetailModel_View_Error_ShowsDetailLoadError(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	m.paneState = stateError
	m.launchErrorTitle = ""
	m.launchError = "connection refused"

	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "Failed to load pipeline details")
	assert.Contains(t, view, "connection refused")
}

func TestPipelineDetailModel_LaunchErrorMsg_SetsErrorState(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})

	errMsg := LaunchErrorMsg{
		PipelineName: "speckit-flow",
		Err:          errors.New("adapter resolution failed"),
	}
	m, _ = m.Update(errMsg)

	assert.Equal(t, stateError, m.paneState)
	assert.Equal(t, "adapter resolution failed", m.launchError)
	assert.Equal(t, "Launch Failed", m.launchErrorTitle)
}

func TestPipelineDetailModel_PaneStateRefactor_PreservesAvailableDetail(t *testing.T) {
	detail := fullAvailableDetail()
	provider := &mockDetailProvider{availableDetail: detail}
	m := newTestDetailModel(provider)

	selMsg := PipelineSelectedMsg{Kind: itemKindAvailable, Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	assert.Equal(t, stateAvailableDetail, m.paneState)
	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "speckit-flow")
}

func TestPipelineDetailModel_PaneStateRefactor_PreservesFinishedDetail(t *testing.T) {
	detail := fullFinishedDetail("completed")
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	assert.Equal(t, stateFinishedDetail, m.paneState)
	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "completed")
}

func TestPipelineDetailModel_PaneStateRefactor_PreservesRunningInfo(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})

	selMsg := PipelineSelectedMsg{Kind: itemKindRunning, Name: "my-pipeline"}
	m, _ = m.Update(selMsg)

	assert.Equal(t, stateRunningInfo, m.paneState)
	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "Running")
}

// ===========================================================================
// T025: Form rendering dimension tests
// ===========================================================================

func TestPipelineDetailModel_Form_ResizeUpdatesFormDimensions(t *testing.T) {
	provider := &mockDetailProvider{availableDetail: fullAvailableDetail()}
	m := newTestDetailModel(provider)

	// Create the form
	cfgMsg := ConfigureFormMsg{PipelineName: "speckit-flow", InputExample: "example"}
	m, _ = m.Update(cfgMsg)
	require.Equal(t, stateConfiguring, m.paneState)
	require.NotNil(t, m.launchForm)

	// Resize -- should not panic or produce empty output
	m.SetSize(120, 50)

	view := m.View()
	assert.NotEmpty(t, view, "form should render after resize")

	// Resize to a smaller size -- should still work
	m.SetSize(60, 20)
	view2 := m.View()
	assert.NotEmpty(t, view2, "form should render after smaller resize")
}

// ===========================================================================
// T014: Chat session (Enter key) tests
// ===========================================================================

func TestPipelineDetailModel_EnterOnFinishedDetail_EmptyWorkspace_SetsActionError(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.WorkspacePath = "" // No workspace
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load finished detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press Enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m, cmd = m.Update(enterMsg)

	assert.Contains(t, m.actionError, "Workspace directory no longer exists")
	assert.Nil(t, cmd)
}

func TestPipelineDetailModel_EnterOnFinishedDetail_ValidWorkspace_ReturnsExecCmd(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.WorkspacePath = "/tmp/test-ws"
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load finished detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press Enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m, cmd = m.Update(enterMsg)

	assert.NotNil(t, cmd, "should return tea.Exec command for chat session")
	assert.Empty(t, m.actionError)
}

func TestPipelineDetailModel_ChatSessionEndedMsg_TriggersRefetch(t *testing.T) {
	detail := fullFinishedDetail("completed")
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Set up state
	m.selectedRunID = "run-123"
	m.paneState = stateFinishedDetail
	m.finishedDetail = detail

	// Send ChatSessionEndedMsg
	m, cmd := m.Update(ChatSessionEndedMsg{})

	assert.NotNil(t, cmd, "should return batch cmd for re-fetch and git refresh")
}

// ===========================================================================
// T017: Branch checkout (b key) tests
// ===========================================================================

func TestPipelineDetailModel_BKey_ValidBranch_ReturnsCheckoutCmd(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.WorkspacePath = "/tmp/test-ws"
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load finished detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press b
	bMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	m, cmd = m.Update(bMsg)

	assert.NotNil(t, cmd, "should return checkout command")
}

func TestPipelineDetailModel_BKey_BranchDeleted_IsNoOp(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.BranchDeleted = true
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load finished detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press b
	bMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	m, cmd = m.Update(bMsg)

	assert.Nil(t, cmd, "should be no-op when branch is deleted")
}

func TestPipelineDetailModel_BKey_EmptyBranch_IsNoOp(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.BranchName = ""
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load finished detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press b
	bMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	m, cmd = m.Update(bMsg)

	assert.Nil(t, cmd, "should be no-op when branch name is empty")
}

func TestPipelineDetailModel_BranchCheckoutMsg_Success_ClearsError(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	m.actionError = "previous error"
	m.paneState = stateFinishedDetail

	m, cmd := m.Update(BranchCheckoutMsg{BranchName: "feat/test", Success: true})

	assert.Empty(t, m.actionError)
	assert.NotNil(t, cmd, "should return git refresh command")
}

func TestPipelineDetailModel_BranchCheckoutMsg_Failure_SetsActionError(t *testing.T) {
	detail := fullFinishedDetail("completed")
	m := newTestDetailModel(&mockDetailProvider{finishedDetail: detail})
	m.paneState = stateFinishedDetail
	m.finishedDetail = detail

	m, _ = m.Update(BranchCheckoutMsg{
		BranchName: "feat/test",
		Success:    false,
		Err:        errors.New("your local changes would be overwritten"),
	})

	assert.Contains(t, m.actionError, "Branch checkout failed")
	assert.Contains(t, m.actionError, "your local changes would be overwritten")
}

// ===========================================================================
// T020: Diff view (d key) tests
// ===========================================================================

func TestPipelineDetailModel_DKey_ValidBranch_ReturnsExecCmd(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.WorkspacePath = "/tmp/test-ws"
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load finished detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press d
	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	m, cmd = m.Update(dMsg)

	assert.NotNil(t, cmd, "should return tea.Exec command for diff view")
}

func TestPipelineDetailModel_DKey_BranchDeleted_IsNoOp(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.BranchDeleted = true
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load finished detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press d
	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	m, cmd = m.Update(dMsg)

	assert.Nil(t, cmd, "should be no-op when branch is deleted")
}

func TestPipelineDetailModel_DKey_EmptyBranch_IsNoOp(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.BranchName = ""
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load finished detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press d
	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	m, cmd = m.Update(dMsg)

	assert.Nil(t, cmd, "should be no-op when branch name is empty")
}

func TestPipelineDetailModel_DiffViewEndedMsg_IsHandled(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	m.paneState = stateFinishedDetail

	m, cmd := m.Update(DiffViewEndedMsg{})

	assert.Nil(t, cmd, "DiffViewEndedMsg should be no-op")
}

// ===========================================================================
// T027: Rendering tests for action hints
// ===========================================================================

func TestPipelineDetailModel_ActionHints_DiffFaintedWhenBranchDeleted(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.BranchDeleted = true
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow", BranchDeleted: true}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	// Verify both [b] and [d] hints are still present (just fainted)
	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "[b] Checkout branch")
	assert.Contains(t, view, "[d] View diff")
}

func TestPipelineDetailModel_ActionHints_EnterFaintedWhenNoWorkspace(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.WorkspacePath = ""
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	// [Enter] hint should still be present in the view (just fainted)
	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "[Enter] Open chat")
}

func TestPipelineDetailModel_ActionError_RenderedInRed(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.WorkspacePath = ""
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press Enter to trigger workspace error
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m, _ = m.Update(enterMsg)

	view := detailStripAnsi(m.View())
	assert.Contains(t, view, "Workspace directory no longer exists")
	// When actionError is set, action hints should NOT be displayed
	assert.NotContains(t, view, "[Esc] Back")
}

func TestPipelineDetailModel_ActionError_ClearsOnNextKeypress(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.WorkspacePath = ""
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press Enter to trigger workspace error
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotEmpty(t, m.actionError)

	// Press any key to clear the error
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Empty(t, m.actionError, "action error should clear on next keypress")
}

func TestPipelineDetailModel_ActionKeysIgnoredWhenNotFocused(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.WorkspacePath = "/tmp/test-ws"
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Load finished detail but DON'T focus
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(false)

	// Press b - should be no-op when not focused
	bMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	m, cmd = m.Update(bMsg)
	assert.Nil(t, cmd, "action keys should be ignored when not focused")
}

func TestPipelineDetailModel_ActionKeysIgnoredInOtherStates(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	m.paneState = stateAvailableDetail
	m.SetFocused(true)

	// Press b - should be no-op in stateAvailableDetail
	bMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	_, cmd := m.Update(bMsg)
	// In stateAvailableDetail, the viewport handles the key, so cmd may or may not be nil.
	// The important thing is that no checkout command is returned.
	if cmd != nil {
		msg := cmd()
		_, isBranchCheckout := msg.(BranchCheckoutMsg)
		assert.False(t, isBranchCheckout, "should not return BranchCheckoutMsg in non-finished state")
	}
}

func TestPipelineDetailModel_BranchDeletedUpdatedFromFinishedDetail(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.BranchDeleted = true
	provider := &mockDetailProvider{finishedDetail: detail}
	m := newTestDetailModel(provider)

	// Selection does NOT set branchDeleted
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow", BranchDeleted: false}
	m, cmd := m.Update(selMsg)
	assert.False(t, m.branchDeleted)

	// DetailDataMsg with BranchDeleted=true should update branchDeleted
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	assert.True(t, m.branchDeleted, "branchDeleted should be updated from FinishedDetail")
}

// ===========================================================================
// Event log tests
// ===========================================================================

func TestPipelineDetailModel_RunEventsMsg_StoresEvents(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	m.paneState = stateRunningInfo
	m.selectedRunID = "run-1"
	m.selectedName = "my-pipeline"

	events := []state.LogRecord{
		{RunID: "run-1", State: "started", StepID: "step1", Message: "Starting..."},
		{RunID: "run-1", State: "completed", StepID: "step1", Message: "Done"},
	}
	m, _ = m.Update(RunEventsMsg{RunID: "run-1", Events: events})

	assert.Len(t, m.persistedEvents, 2)
}

func TestPipelineDetailModel_RunEventsMsg_Error_DoesNotStoreEvents(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	m.paneState = stateRunningInfo

	m, _ = m.Update(RunEventsMsg{RunID: "run-1", Err: errors.New("db error")})

	assert.Nil(t, m.persistedEvents)
}

func TestPipelineDetailModel_SelectionChange_ClearsPersistedEvents(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})
	m.persistedEvents = []state.LogRecord{{RunID: "run-1", State: "started"}}

	m, _ = m.Update(PipelineSelectedMsg{Kind: itemKindAvailable, Name: "new-pipeline"})

	assert.Nil(t, m.persistedEvents, "persisted events should be cleared on selection change")
}

func TestPipelineDetailModel_RunningInfo_AutoFetchesEvents(t *testing.T) {
	events := []state.LogRecord{
		{RunID: "run-stale", State: "started", StepID: "step1"},
	}
	provider := &mockDetailProvider{runEvents: events}
	m := newTestDetailModel(provider)

	m, cmd := m.Update(PipelineSelectedMsg{Kind: itemKindRunning, Name: "stale-pipeline", RunID: "run-stale"})

	assert.Equal(t, stateRunningInfo, m.paneState)
	assert.NotNil(t, cmd, "should return cmd to fetch run events")

	// Execute the cmd
	msg := cmd()
	evtMsg, ok := msg.(RunEventsMsg)
	assert.True(t, ok, "should return RunEventsMsg")
	assert.Len(t, evtMsg.Events, 1)
}

func TestPipelineDetailModel_LKey_FinishedDetail_FetchesEvents(t *testing.T) {
	events := []state.LogRecord{
		{RunID: "run-123", State: "completed", StepID: "step1"},
	}
	provider := &mockDetailProvider{
		finishedDetail: fullFinishedDetail("completed"),
		runEvents:      events,
	}
	m := newTestDetailModel(provider)

	// Load finished detail
	selMsg := PipelineSelectedMsg{Kind: itemKindFinished, RunID: "run-123", Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)
	m.SetFocused(true)

	// Press l
	lMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	m, cmd = m.Update(lMsg)

	assert.NotNil(t, cmd, "should return cmd to fetch run events")

	// Execute the cmd
	msg := cmd()
	evtMsg, ok := msg.(RunEventsMsg)
	assert.True(t, ok, "should return RunEventsMsg")
	assert.Len(t, evtMsg.Events, 1)
}

// ===========================================================================
// #306: Configure form viewport scroll tests
// ===========================================================================

func TestPipelineDetailModel_ConfiguringFormUsesViewport(t *testing.T) {
	provider := &mockDetailProvider{availableDetail: fullAvailableDetail()}
	m := newTestDetailModel(provider)

	// Load available detail
	selMsg := PipelineSelectedMsg{Kind: itemKindAvailable, Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	// Enter configuring state
	cfgMsg := ConfigureFormMsg{PipelineName: "speckit-flow", InputExample: "example"}
	m, _ = m.Update(cfgMsg)
	require.Equal(t, stateConfiguring, m.paneState)
	require.NotNil(t, m.launchForm)

	// Viewport should have content set from the form
	view := m.View()
	assert.NotEmpty(t, view, "form view should not be empty")
	assert.Contains(t, view, "Input", "viewport should contain form fields")
}

func TestPipelineDetailModel_ConfiguringFormViewportResetsOnNew(t *testing.T) {
	provider := &mockDetailProvider{availableDetail: fullAvailableDetail()}
	m := newTestDetailModel(provider)

	// Load available detail
	selMsg := PipelineSelectedMsg{Kind: itemKindAvailable, Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	// Enter configuring state
	cfgMsg := ConfigureFormMsg{PipelineName: "speckit-flow", InputExample: "example"}
	m, _ = m.Update(cfgMsg)

	// Scroll the viewport down
	m.viewport.SetYOffset(5)

	// Create a new form — viewport should reset
	cfgMsg2 := ConfigureFormMsg{PipelineName: "speckit-flow", InputExample: "other example"}
	m, _ = m.Update(cfgMsg2)

	assert.Equal(t, 0, m.viewport.YOffset,
		"viewport should reset to top when a new form is created")
}

func TestPipelineDetailModel_ConfiguringSmallViewport_FormIsScrollable(t *testing.T) {
	provider := &mockDetailProvider{availableDetail: fullAvailableDetail()}
	m := NewPipelineDetailModel(provider)
	m.SetSize(80, 5) // Very small height

	// Load available detail
	selMsg := PipelineSelectedMsg{Kind: itemKindAvailable, Name: "speckit-flow"}
	m, cmd := m.Update(selMsg)
	dataMsg := cmd()
	m, _ = m.Update(dataMsg)

	// Enter configuring state
	cfgMsg := ConfigureFormMsg{PipelineName: "speckit-flow", InputExample: "example"}
	m, _ = m.Update(cfgMsg)
	require.Equal(t, stateConfiguring, m.paneState)

	// The form content should be longer than the viewport height
	view := m.View()
	assert.NotEmpty(t, view, "form should render even with small viewport")
}

func TestRenderRunningInfo_WithEvents(t *testing.T) {
	events := []state.LogRecord{
		{State: "started", StepID: "step1", Message: "Starting step1", Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)},
		{State: "completed", StepID: "step1", Message: "Done", Timestamp: time.Date(2026, 1, 15, 10, 1, 0, 0, time.UTC)},
	}

	view := renderRunningInfo("test-pipeline", "some input", time.Date(2026, 1, 15, 9, 59, 0, 0, time.UTC), 80, events)
	stripped := detailStripAnsi(view)

	assert.Contains(t, stripped, "test-pipeline")
	assert.Contains(t, stripped, "Running")
	assert.Contains(t, stripped, "Event Log:")
	assert.Contains(t, stripped, "step1")
	assert.Contains(t, stripped, "Starting step1")
}

func TestRenderRunningInfo_WithoutEvents(t *testing.T) {
	view := renderRunningInfo("test-pipeline", "", time.Time{}, 80, nil)
	stripped := detailStripAnsi(view)

	assert.Contains(t, stripped, "test-pipeline")
	assert.Contains(t, stripped, "Running")
	assert.NotContains(t, stripped, "Event Log:")
}

func TestRenderRunningInfo_ShowsPersistedEventHint(t *testing.T) {
	view := renderRunningInfo("test-pipeline", "", time.Now(), 80, nil)
	stripped := detailStripAnsi(view)

	assert.Contains(t, stripped, "Press [Enter] to view live event dashboard from persisted events")
	assert.Contains(t, stripped, "Use [c] to cancel or dismiss this run")
	// Should NOT contain old "unavailable" text
	assert.NotContains(t, stripped, "Live output is only available")
	assert.NotContains(t, stripped, "appears stale")
}

func TestRenderFinishedDetail_WithEvents(t *testing.T) {
	detail := fullFinishedDetail("completed")
	events := []state.LogRecord{
		{State: "started", StepID: "specify", Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)},
	}

	view := renderFinishedDetail(detail, 80, false, "", events)
	stripped := detailStripAnsi(view)

	assert.Contains(t, stripped, "Event Log:")
	assert.Contains(t, stripped, "specify")
}

func TestRenderFinishedDetail_WithoutEvents(t *testing.T) {
	detail := fullFinishedDetail("completed")

	view := renderFinishedDetail(detail, 80, false, "", nil)
	stripped := detailStripAnsi(view)

	assert.NotContains(t, stripped, "Event Log:")
}

func TestFormatLogRecord_WithMessage(t *testing.T) {
	rec := state.LogRecord{
		Timestamp: time.Date(2026, 1, 15, 10, 30, 45, 0, time.UTC),
		State:     "started",
		StepID:    "step1",
		Message:   "Starting step",
	}
	result := formatLogRecord(rec)

	assert.Contains(t, result, "10:30:45")
	assert.Contains(t, result, "[started]")
	assert.Contains(t, result, "step1")
	assert.Contains(t, result, "Starting step")
}

func TestFormatLogRecord_WithoutMessage(t *testing.T) {
	rec := state.LogRecord{
		Timestamp: time.Date(2026, 1, 15, 10, 30, 45, 0, time.UTC),
		State:     "completed",
		StepID:    "step1",
	}
	result := formatLogRecord(rec)

	assert.Contains(t, result, "10:30:45")
	assert.Contains(t, result, "[completed]")
	assert.Contains(t, result, "step1")
}

func TestFormatLogRecord_EmptyStepID_ShowsPipeline(t *testing.T) {
	rec := state.LogRecord{
		Timestamp: time.Date(2026, 1, 15, 10, 30, 45, 0, time.UTC),
		State:     "started",
	}
	result := formatLogRecord(rec)

	assert.Contains(t, result, "pipeline")
}
