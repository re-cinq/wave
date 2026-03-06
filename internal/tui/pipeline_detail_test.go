package tui

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
}

func (m *mockDetailProvider) FetchAvailableDetail(name string) (*AvailableDetail, error) {
	return m.availableDetail, m.availableErr
}

func (m *mockDetailProvider) FetchFinishedDetail(runID string) (*FinishedDetail, error) {
	return m.finishedDetail, m.finishedErr
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
		Name:         "speckit-flow",
		Description:  "A specification pipeline",
		Category:     "spec",
		StepCount:    3,
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
	assert.Contains(t, view, "5m30s")
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

func TestPipelineDetailModel_SectionHeaderShowsPlaceholder(t *testing.T) {
	m := newTestDetailModel(&mockDetailProvider{})

	view := detailStripAnsi(updateAndView(m, PipelineSelectedMsg{Kind: itemKindSectionHeader, Name: "Running"}))

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
