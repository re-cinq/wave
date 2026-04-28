package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/pipelinecatalog"
	"github.com/stretchr/testify/assert"
)

type mockProvider struct{}

func (m *mockProvider) FetchGitState() (GitState, error) {
	return GitState{Branch: "main"}, nil
}

func (m *mockProvider) FetchManifestInfo() (ManifestInfo, error) {
	return ManifestInfo{ProjectName: "test"}, nil
}

func (m *mockProvider) FetchGitHubInfo(repo string) (GitHubInfo, error) {
	return GitHubInfo{}, nil
}

func (m *mockProvider) FetchPipelineHealth() (HealthStatus, error) {
	return HealthOK, nil
}

type mockPipelineDataProvider struct{}

func (m *mockPipelineDataProvider) FetchRunningPipelines() ([]RunningPipeline, error) {
	return nil, nil
}

func (m *mockPipelineDataProvider) FetchFinishedPipelines(limit int) ([]FinishedPipeline, error) {
	return nil, nil
}

func (m *mockPipelineDataProvider) FetchAvailablePipelines() ([]pipelinecatalog.PipelineInfo, error) {
	return nil, nil
}

func TestNewAppModel_InitialState(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	assert.False(t, m.ready)
	assert.False(t, m.shuttingDown)
	assert.Equal(t, 0, m.width)
	assert.Equal(t, 0, m.height)
	assert.Equal(t, "Pipelines", m.statusBar.contextLabel)
}

func TestAppModel_Init_ReturnsCmds(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	cmd := m.Init()
	// Header.Init() returns a batch of async fetch commands
	assert.NotNil(t, cmd)
}

func TestAppModel_Init_IncludesContentCmds(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestAppModel_Update_WindowSizeMsg(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	assert.True(t, model.ready)
	assert.Equal(t, 120, model.width)
	assert.Equal(t, 40, model.height)
	assert.Equal(t, 120, model.header.width)
	assert.Equal(t, 120, model.statusBar.width)
	assert.Equal(t, 120, model.content.width)
	assert.Equal(t, 40-headerHeight-2*statusBarHeight, model.content.height)
}

func TestAppModel_Update_WindowSizeMsg_PropagatesContent(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	assert.Equal(t, 120, model.content.width)
	contentHeight := 40 - headerHeight - 2*statusBarHeight
	assert.Equal(t, contentHeight, model.content.height)
	// List should have received size too
	assert.Greater(t, model.content.list.width, 0)
	assert.Equal(t, contentHeight-2, model.content.list.height)
}

func TestAppModel_Update_QuitOnQ(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	_, cmd := m.Update(msg)
	assert.NotNil(t, cmd)

	// tea.Quit returns a special quit message
	quitMsg := cmd()
	assert.IsType(t, tea.QuitMsg{}, quitMsg)
}

func TestAppModel_Update_CtrlC_SetsShuttingDown(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}

	updated, cmd := m.Update(msg)
	model := updated.(AppModel)

	assert.True(t, model.shuttingDown)
	assert.NotNil(t, cmd)
	quitMsg := cmd()
	assert.IsType(t, tea.QuitMsg{}, quitMsg)
}

func TestAppModel_View_BeforeReady(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	view := m.View()
	assert.Equal(t, "Initializing...", view)
}

func TestAppModel_View_AfterReady(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	view := model.View()

	// Should contain Wave logo ASCII art characters
	assert.Contains(t, view, "╦")
	assert.Contains(t, view, "╚╩╝")
	// Should contain content placeholder
	assert.Contains(t, view, "Select a pipeline to view details")
	// Should contain status bar hints
	assert.Contains(t, view, "q: quit")
	assert.Contains(t, view, "Tab/Shift+Tab: views")
}

func TestAppModel_View_TooSmall(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"narrow", 60, 30},
		{"short", 100, 20},
		{"both small", 40, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
			msg := tea.WindowSizeMsg{Width: tt.width, Height: tt.height}
			updated, _ := m.Update(msg)
			model := updated.(AppModel)

			view := model.View()
			assert.Contains(t, view, "Terminal too small")
			assert.Contains(t, view, "80×24")
		})
	}
}

func TestAppModel_View_ExactMinimumSize(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	view := model.View()
	// At exactly minimum size, should render normally, not show degradation
	assert.NotContains(t, view, "Terminal too small")
	// Should contain Wave logo ASCII art
	assert.Contains(t, view, "╦")
	assert.Contains(t, view, "╚╩╝")
}

// --- T033: App integration tests for header message forwarding ---

func TestAppModel_Update_ForwardsGitStateMsgToHeader(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	// First, set up with a window size
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	// Send a GitStateMsg through the app
	gitMsg := GitStateMsg{
		State: GitState{
			Branch:     "feature/test",
			CommitHash: "def5678",
			IsDirty:    true,
			RemoteName: "origin",
		},
		Err: nil,
	}
	updated, _ = model.Update(gitMsg)
	model = updated.(AppModel)

	assert.Equal(t, "feature/test", model.header.metadata.Branch)
	assert.Equal(t, "def5678", model.header.metadata.CommitHash)
	assert.True(t, model.header.metadata.IsDirty)
	assert.Equal(t, "origin", model.header.metadata.RemoteName)
}

func TestAppModel_Update_ForwardsManifestInfoMsgToHeader(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	manifestMsg := ManifestInfoMsg{
		Info: ManifestInfo{ProjectName: "wave", RepoName: "re-cinq/wave"},
		Err:  nil,
	}
	updated, _ = model.Update(manifestMsg)
	model = updated.(AppModel)

	assert.Equal(t, "wave", model.header.metadata.ProjectName)
	assert.Equal(t, "re-cinq/wave", model.header.metadata.RepoName)
}

func TestAppModel_Update_ForwardsPipelineHealthMsgToHeader(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	healthMsg := PipelineHealthMsg{Health: HealthWarn, Err: nil}
	updated, _ = model.Update(healthMsg)
	model = updated.(AppModel)

	assert.Equal(t, HealthWarn, model.header.metadata.Health)
}

func TestAppModel_Update_ForwardsRunningCountMsgToHeader(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	countMsg := RunningCountMsg{Count: 3}
	updated, cmd := model.Update(countMsg)
	model = updated.(AppModel)

	assert.Equal(t, 3, model.header.metadata.RunningCount)
	assert.True(t, model.header.logo.IsActive())
	assert.NotNil(t, cmd, "should return logo tick command through app")
}

func TestAppModel_Update_ForwardsPipelineSelectedMsgToHeader(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	selMsg := PipelineSelectedMsg{
		RunID:      "run-123",
		BranchName: "feature/login",
	}
	updated, _ = model.Update(selMsg)
	model = updated.(AppModel)

	assert.Equal(t, "feature/login", model.header.metadata.OverrideBranch)
}

func TestAppModel_View_HeaderRendersForwardedData(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	model := updated.(AppModel)

	// Forward metadata through the app model
	updated, _ = model.Update(GitStateMsg{
		State: GitState{Branch: "feature/tui", CommitHash: "aaa1111"},
		Err:   nil,
	})
	model = updated.(AppModel)

	updated, _ = model.Update(ManifestInfoMsg{
		Info: ManifestInfo{ProjectName: "wave-project"},
		Err:  nil,
	})
	model = updated.(AppModel)

	view := model.View()
	assert.Contains(t, view, "feature/tui")
	assert.Contains(t, view, "wave-project")
}

func TestAppModel_Update_ForwardsFocusChangedMsgToStatusBar(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	// Send FocusChangedMsg
	focusMsg := FocusChangedMsg{Pane: FocusPaneRight}
	updated, _ = model.Update(focusMsg)
	model = updated.(AppModel)

	assert.Equal(t, FocusPaneRight, model.statusBar.focusPane)
}

// ===========================================================================
// T019: App model tests for pipeline launch flow
// ===========================================================================

func TestAppModel_Update_QKeyWithFocusRight_Quits(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	// Set up with a window size so the app is ready
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	// Set focus to right pane (no form/filter active)
	model.content.focus = FocusPaneRight
	model.content.list.SetFocused(false)
	model.content.detail.SetFocused(true)

	// Send q key — should quit since no input is active
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)
	assert.NotNil(t, cmd, "q key should produce a quit command from right pane")

	quitMsg := cmd()
	assert.IsType(t, tea.QuitMsg{}, quitMsg, "q key with right pane focus should quit when no input is active")
}

func TestAppModel_Update_QKeyWhileConfiguring_DoesNotQuit(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	// Simulate form/configuring state
	model.content.detail.paneState = stateConfiguring
	model.content.focus = FocusPaneRight

	// Send q key — should NOT quit because form input is active
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)

	if cmd != nil {
		result := cmd()
		_, isQuit := result.(tea.QuitMsg)
		assert.False(t, isQuit, "q key should not quit when form is active")
	}
}

func TestAppModel_Update_QKeyWithFocusLeft_Quits(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	_, cmd := m.Update(msg)
	assert.NotNil(t, cmd)

	quitMsg := cmd()
	assert.IsType(t, tea.QuitMsg{}, quitMsg)
}

func TestAppModel_Update_CancelAllCalledOnCtrlC(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})

	// CancelAll is called inside Update for CtrlC. Verify the method
	// does not panic when launcher is nil (no deps.Manifest provided).
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updated, cmd := m.Update(msg)
	model := updated.(AppModel)

	assert.True(t, model.shuttingDown)
	assert.NotNil(t, cmd)
}

func TestAppModel_Update_ForwardsFormActiveMsgToStatusBar(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	// Send FormActiveMsg
	formMsg := FormActiveMsg{Active: true}
	updated, _ = model.Update(formMsg)
	model = updated.(AppModel)

	assert.True(t, model.statusBar.formActive)
}

// ===========================================================================
// T039: App model tests for FinishedDetailActiveMsg forwarding
// ===========================================================================

func TestAppModel_Update_ForwardsFinishedDetailActiveMsgToStatusBar(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	// Send FinishedDetailActiveMsg
	finishedMsg := FinishedDetailActiveMsg{Active: true}
	updated, _ = model.Update(finishedMsg)
	model = updated.(AppModel)

	assert.True(t, model.statusBar.finishedDetailActive)
}

func TestAppModel_Update_ForwardsFinishedDetailActiveFalseToStatusBar(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	// Set active first
	updated, _ = model.Update(FinishedDetailActiveMsg{Active: true})
	model = updated.(AppModel)
	assert.True(t, model.statusBar.finishedDetailActive)

	// Now set inactive
	updated, _ = model.Update(FinishedDetailActiveMsg{Active: false})
	model = updated.(AppModel)
	assert.False(t, model.statusBar.finishedDetailActive)
}
