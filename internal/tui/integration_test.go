package tui

// Integration-level TUI tests using teatest (charmbracelet's official Bubble Tea testing library).
// These tests exercise multi-model composition — the full AppModel with header, content, and
// status bar working together — rather than testing individual models in isolation.
//
// References: issue #372

import (
	"bytes"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/recinq/wave/internal/pipelinecatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Integration test helpers
// ---------------------------------------------------------------------------

// integrationMockProvider is a MetadataProvider returning stable test data
// suitable for integration tests where we verify rendered output.
type integrationMockProvider struct {
	projectName string
	branch      string
}

func (p *integrationMockProvider) FetchGitState() (GitState, error) {
	branch := p.branch
	if branch == "" {
		branch = "main"
	}
	return GitState{Branch: branch, CommitHash: "abc1234", IsDirty: false, RemoteName: "origin"}, nil
}

func (p *integrationMockProvider) FetchManifestInfo() (ManifestInfo, error) {
	name := p.projectName
	if name == "" {
		name = "wave-test"
	}
	return ManifestInfo{ProjectName: name, RepoName: ""}, nil
}

func (p *integrationMockProvider) FetchGitHubInfo(repo string) (GitHubInfo, error) {
	return GitHubInfo{AuthState: GitHubNotConfigured}, nil
}

func (p *integrationMockProvider) FetchPipelineHealth() (HealthStatus, error) {
	return HealthOK, nil
}

// integrationPipelineProvider is a PipelineDataProvider returning stable pipeline data.
type integrationPipelineProvider struct {
	available []pipelinecatalog.PipelineInfo
	running   []RunningPipeline
	finished  []FinishedPipeline
}

func (p *integrationPipelineProvider) FetchRunningPipelines() ([]RunningPipeline, error) {
	return p.running, nil
}

func (p *integrationPipelineProvider) FetchFinishedPipelines(limit int) ([]FinishedPipeline, error) {
	return p.finished, nil
}

func (p *integrationPipelineProvider) FetchAvailablePipelines() ([]pipelinecatalog.PipelineInfo, error) {
	return p.available, nil
}

// newIntegrationApp creates an AppModel wired with integration-level mock providers.
func newIntegrationApp(projectName string, pipelines []pipelinecatalog.PipelineInfo) AppModel { //nolint:unparam // test helper
	meta := &integrationMockProvider{projectName: projectName}
	pipeProv := &integrationPipelineProvider{available: pipelines}
	return NewAppModel(meta, pipeProv, nil, LaunchDependencies{})
}

// ---------------------------------------------------------------------------
// TestApp_Renders
// ---------------------------------------------------------------------------

// TestApp_Renders verifies that after a WindowSizeMsg the rendered view contains
// the Wave logo characters and navigation hints from the status bar.
//
// We capture the output stream while the program is running (before quit) rather
// than from FinalOutput, because teatest's FinalOutput only captures the last
// frame emitted after the program quits (which is typically a blank cleared screen).
func TestApp_Renders(t *testing.T) {
	m := newIntegrationApp("wave-project", nil)

	// Use teatest to drive the model in a real tea.Program
	tm := teatest.NewTestModel(
		t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	// Accumulate output while we wait for the logo to appear.
	var captured []byte
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		captured = append(captured, bts...)
		// Wait until we can see the Wave logo characters
		return bytes.Contains(captured, []byte("╦")) && bytes.Contains(captured, []byte("q: quit"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))

	_ = tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	outStr := stripAnsi(string(captured))

	// Wave logo must be present
	assert.Contains(t, outStr, "╦", "Wave logo box-drawing char must appear in rendered output")
	assert.Contains(t, outStr, "╚╩╝", "Wave logo bottom must appear in rendered output")

	// Navigation hints must be present (status bar)
	assert.Contains(t, outStr, "q: quit", "quit hint must appear in status bar")
}

// ---------------------------------------------------------------------------
// TestApp_TabCycling
// ---------------------------------------------------------------------------

// TestApp_TabCycling sends a Tab key event and verifies the view changes from
// Pipelines to the next view (Personas). The status bar label should update.
func TestApp_TabCycling(t *testing.T) {
	m := newIntegrationApp("wave-project", nil)

	// First set up the model through Update() — teatest drives real program execution
	// but for view cycling we can also test via direct model Update() since the
	// integration value is verifying that Tab on the composed AppModel propagates
	// through ContentModel's cycleView and reaches the StatusBarModel.

	// Get to ready state
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := updated.(AppModel)

	// Verify initial view is Pipelines
	assert.Equal(t, ViewPipelines, app.content.currentView)
	assert.Equal(t, "Pipelines", app.statusBar.contextLabel)

	// Send Tab key through the full app — this should cycle to ViewPersonas
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = updated.(AppModel)

	// View should have changed from ViewPipelines (0) to ViewPersonas (1)
	assert.Equal(t, ViewPersonas, app.content.currentView,
		"Tab should cycle the active view from Pipelines to Personas")

	// The view's rendered output should change — Personas view has no data so
	// it will render an empty/placeholder state, but the structure should differ
	view := app.View()
	assert.NotEqual(t, "Initializing...", view,
		"app should not be in initializing state after WindowSizeMsg")

	// Tab again — should advance to Contracts
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = updated.(AppModel)
	assert.Equal(t, ViewContracts, app.content.currentView,
		"second Tab should advance to Contracts view")

	// Tab enough times to wrap back to Pipelines (7 more = total 9 = full cycle)
	for range 7 {
		updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
		app = updated.(AppModel)
	}
	assert.Equal(t, ViewPipelines, app.content.currentView,
		"9 Tabs should complete a full view cycle back to Pipelines")
}

// ---------------------------------------------------------------------------
// TestApp_QuitOnQ
// ---------------------------------------------------------------------------

// TestApp_QuitOnQ uses teatest to send a 'q' keypress through the real program
// and verifies the program exits cleanly.
func TestApp_QuitOnQ(t *testing.T) {
	m := newIntegrationApp("wave-project", nil)

	tm := teatest.NewTestModel(
		t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for the TUI to become ready (logo rendered)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("╦"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))

	// Send 'q' to quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Program should finish cleanly within a reasonable timeout
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	// If we reach here without timeout, the program exited cleanly
	final := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	finalApp, ok := final.(AppModel)
	require.True(t, ok, "final model should be an AppModel")

	// After 'q', shuttingDown should be set or the model should have requested quit
	// Either shuttingDown=true or we simply reached this line (program exited)
	_ = finalApp // the fact that WaitFinished didn't timeout is the assertion
}

// ---------------------------------------------------------------------------
// TestPipelineList_EmptyState
// ---------------------------------------------------------------------------

// TestPipelineList_EmptyState creates a pipeline list model with no pipelines
// and verifies the empty state message is rendered.
func TestPipelineList_EmptyState(t *testing.T) {
	// Create a PipelineListModel with no data — test View() output directly
	// since PipelineListModel is a core component of the composed app.
	provider := &integrationPipelineProvider{} // no pipelines
	list := NewPipelineListModel(provider)
	list.SetSize(80, 20)

	// Inject empty data message (simulates provider returning nothing)
	list, _ = list.Update(PipelineDataMsg{
		Running:   nil,
		Finished:  nil,
		Available: nil,
	})

	view := listStripAnsi(list.View())
	assert.Contains(t, view, "No pipelines found",
		"empty pipeline list should show 'No pipelines found' placeholder")
}

// TestPipelineList_EmptyState_InApp verifies that the empty-state message
// propagates correctly when the full AppModel renders with no pipeline data.
func TestPipelineList_EmptyState_InApp(t *testing.T) {
	m := newIntegrationApp("wave-project", nil /* no pipelines */)

	// Bring to ready state
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := updated.(AppModel)

	// Inject empty pipeline data into the list via message
	updated, _ = app.Update(PipelineDataMsg{Running: nil, Finished: nil, Available: nil})
	app = updated.(AppModel)

	view := stripAnsi(app.View())
	assert.Contains(t, view, "No pipelines found",
		"app should render empty-state message when no pipelines are available")
}

// ---------------------------------------------------------------------------
// TestHeader_ShowsMetadata
// ---------------------------------------------------------------------------

// TestHeader_ShowsMetadata wires a MetadataProvider with a known project name,
// injects the manifest message, and verifies the project name appears in the view.
func TestHeader_ShowsMetadata(t *testing.T) {
	const projectName = "my-test-project"

	h := NewHeaderModel(nil) // nil provider — we'll inject via message
	h.SetWidth(200)

	// Simulate the async manifest fetch completing
	updated, _ := h.Update(ManifestInfoMsg{
		Info: ManifestInfo{ProjectName: projectName},
		Err:  nil,
	})

	view := stripAnsi(updated.View())
	assert.Contains(t, view, projectName,
		"header view should contain the project name after ManifestInfoMsg")
}

// TestHeader_ShowsMetadata_ViaApp verifies the metadata flows correctly through
// the full AppModel composition — MetadataProvider → Header → View().
func TestHeader_ShowsMetadata_ViaApp(t *testing.T) {
	const projectName = "integration-project"

	meta := &integrationMockProvider{projectName: projectName}
	pipeProv := &integrationPipelineProvider{}
	app := NewAppModel(meta, pipeProv, nil, LaunchDependencies{})

	// Set up size
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	app = updated.(AppModel)

	// Manually inject the manifest info (bypassing async provider)
	updated, _ = app.Update(ManifestInfoMsg{
		Info: ManifestInfo{ProjectName: projectName},
		Err:  nil,
	})
	app = updated.(AppModel)

	view := stripAnsi(app.View())
	assert.Contains(t, view, projectName,
		"project name from MetadataProvider should appear in full app render")
}

// ---------------------------------------------------------------------------
// TestApp_ComposedLayout
// ---------------------------------------------------------------------------

// TestApp_ComposedLayout is an end-to-end integration test that composes all
// sub-models (header, statusbar, content/pipeline-list) and verifies the
// assembled view contains expected structural elements at a realistic terminal size.
func TestApp_ComposedLayout(t *testing.T) {
	const projectName = "composed-layout-test"

	pipelines := []pipelinecatalog.PipelineInfo{
		{Name: "impl-issue", Description: "Implement a GitHub issue", StepCount: 5},
		{Name: "ops-pr-review", Description: "Review a pull request", StepCount: 3},
	}

	meta := &integrationMockProvider{projectName: projectName, branch: "main"}
	pipeProv := &integrationPipelineProvider{available: pipelines}
	app := NewAppModel(meta, pipeProv, nil, LaunchDependencies{})

	// Apply window size
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	app = updated.(AppModel)

	// Inject sync data from providers
	updated, _ = app.Update(ManifestInfoMsg{
		Info: ManifestInfo{ProjectName: projectName},
		Err:  nil,
	})
	app = updated.(AppModel)

	updated, _ = app.Update(GitStateMsg{
		State: GitState{Branch: "main", CommitHash: "deadbeef", IsDirty: false, RemoteName: "origin"},
		Err:   nil,
	})
	app = updated.(AppModel)

	// Inject pipeline data directly into the list
	updated, _ = app.Update(PipelineDataMsg{Available: pipelines})
	app = updated.(AppModel)

	view := stripAnsi(app.View())

	// Header section
	assert.Contains(t, view, "╦", "Wave logo must be present")
	assert.Contains(t, view, projectName, "project name must appear in header")
	assert.Contains(t, view, "main", "current branch must appear in header")

	// Pipeline list section
	assert.Contains(t, view, "impl-issue", "available pipeline names must appear in list")
	assert.Contains(t, view, "ops-pr-review", "available pipeline names must appear in list")

	// Status bar section
	assert.Contains(t, view, "q: quit", "quit navigation hint must appear")

	// Verify layout has multiple lines (not collapsed to a single line)
	lines := strings.Split(view, "\n")
	assert.Greater(t, len(lines), 5, "composed layout must span multiple lines")
}

// ---------------------------------------------------------------------------
// TestApp_ShiftTabCyclesBackward
// ---------------------------------------------------------------------------

// TestApp_ShiftTabCyclesBackward verifies that Shift+Tab cycles the view backward.
func TestApp_ShiftTabCyclesBackward(t *testing.T) {
	m := newIntegrationApp("wave-project", nil)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := updated.(AppModel)

	// Tab forward once to ViewPersonas
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = updated.(AppModel)
	require.Equal(t, ViewPersonas, app.content.currentView)

	// Shift+Tab backward — should return to ViewPipelines
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	app = updated.(AppModel)
	assert.Equal(t, ViewPipelines, app.content.currentView,
		"Shift+Tab should cycle the view backward from Personas to Pipelines")
}
