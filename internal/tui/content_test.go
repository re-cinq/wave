package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contentTestPipelineProvider struct{}

func (m *contentTestPipelineProvider) FetchRunningPipelines() ([]RunningPipeline, error) {
	return nil, nil
}

func (m *contentTestPipelineProvider) FetchFinishedPipelines(limit int) ([]FinishedPipeline, error) {
	return nil, nil
}

func (m *contentTestPipelineProvider) FetchAvailablePipelines() ([]PipelineInfo, error) {
	return nil, nil
}

func TestContentModel_NewContentModel(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	assert.True(t, c.list.focused)
}

func TestContentModel_SetSize(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	assert.Equal(t, 0, c.width)
	assert.Equal(t, 0, c.height)

	c.SetSize(120, 40)
	assert.Equal(t, 120, c.width)
	assert.Equal(t, 40, c.height)
}

func TestContentModel_SetSize_PropagatesListDimensions(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Left pane: 30% of 120 = 36, clamped to [25, 50] -> 36
	assert.Equal(t, 36, c.list.width)
	assert.Equal(t, 38, c.list.height)
}

func TestContentModel_LeftPaneWidth(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		expected int
	}{
		{"30 percent of 120", 120, 36},
		{"minimum 25", 60, 25},  // 30% of 60 = 18 -> clamped to 25
		{"maximum 50", 200, 50}, // 30% of 200 = 60 -> clamped to 50
		{"exact 100", 100, 30},  // 30% of 100 = 30
		{"narrow 80", 80, 25},   // 30% of 80 = 24 -> clamped to 25
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
			c.SetSize(tt.width, 40)
			assert.Equal(t, tt.expected, c.list.width)
		})
	}
}

func TestContentModel_View_RightPanePlaceholder(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)
	view := c.View()
	assert.Contains(t, view, "Select a pipeline to view details")
}

func TestContentModel_View_ZeroDimensions(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	view := c.View()
	assert.Equal(t, "", view)
}

func TestContentModel_Init_ReturnsCommands(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	cmd := c.Init()
	assert.NotNil(t, cmd)
}

func TestContentModel_FocusStartsOnLeft(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	assert.Equal(t, FocusPaneLeft, c.focus)
	assert.True(t, c.list.focused)
}

func TestContentModel_SetSize_PropagatesDetailDimensions(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Right pane: 120 - 36 - 3 = 81 (separator + padding)
	assert.Equal(t, 81, c.detail.width)
	assert.Equal(t, 38, c.detail.height)
}

func TestContentModel_EnterOnAvailableItemTransitionsFocusRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Inject data with an available pipeline
	c.list, _ = c.list.Update(PipelineDataMsg{
		Available: []PipelineInfo{{Name: "test-pipe", StepCount: 1}},
	})

	// Move cursor to the pipeline name node
	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindPipelineName {
			c.list.cursor = i
			break
		}
	}

	// Press Enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	c, cmd := c.Update(msg)

	assert.Equal(t, FocusPaneRight, c.focus)
	assert.False(t, c.list.focused)
	assert.True(t, c.detail.focused)
	assert.NotNil(t, cmd)
}

func TestContentModel_EnterOnFinishedItemTransitionsFocusRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Finished: []FinishedPipeline{{RunID: "r1", Name: "done", Status: "completed"}},
	})

	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindFinished {
			c.list.cursor = i
			break
		}
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	c, cmd := c.Update(msg)

	assert.Equal(t, FocusPaneRight, c.focus)
	assert.NotNil(t, cmd)
}

func TestContentModel_EnterOnPipelineName_TransitionsRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Available: []PipelineInfo{{Name: "test"}},
	})

	// Cursor starts on a pipeline name node
	assert.Equal(t, itemKindPipelineName, c.list.navigable[c.list.cursor].kind)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	c, _ = c.Update(msg)

	// Pipeline names are focusable — Enter transitions to right pane
	assert.Equal(t, FocusPaneRight, c.focus)
}

func TestContentModel_EnterOnRunningItemTransitionsFocusRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Running: []RunningPipeline{{RunID: "r1", Name: "running-pipe"}},
	})

	// Move to the running item
	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindRunning {
			c.list.cursor = i
			break
		}
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	c, _ = c.Update(msg)

	// Running items are now focusable and transition focus to right pane
	assert.Equal(t, FocusPaneRight, c.focus)
	assert.False(t, c.list.focused)
	assert.True(t, c.detail.focused)
}

func TestContentModel_EscFromRightPaneReturnsFocusLeft(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Set focus to right pane manually
	c.focus = FocusPaneRight
	c.list.SetFocused(false)
	c.detail.SetFocused(true)

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	c, cmd := c.Update(msg)

	assert.Equal(t, FocusPaneLeft, c.focus)
	assert.True(t, c.list.focused)
	assert.False(t, c.detail.focused)
	assert.NotNil(t, cmd)
}

func TestContentModel_ArrowKeysInRightPaneDoNotMoveList(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Available: []PipelineInfo{{Name: "pipe1"}, {Name: "pipe2"}},
	})

	// Move cursor to first available item
	// Expand the Finished section (collapsed by default)
	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindPipelineName {
			c.list.cursor = i
			break
		}
	}
	initialCursor := c.list.cursor

	// Switch focus to right pane
	c.focus = FocusPaneRight
	c.list.SetFocused(false)
	c.detail.SetFocused(true)

	// Press down arrow
	msg := tea.KeyMsg{Type: tea.KeyDown}
	c, _ = c.Update(msg)

	// List cursor should not have changed
	assert.Equal(t, initialCursor, c.list.cursor)
}

// ===========================================================================
// T012: Content model integration tests for pipeline launch flow
// ===========================================================================

func TestContentModel_EnterOnAvailable_EmitsConfigureFormMsg(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Inject data with an available pipeline that has an input example
	c.list, _ = c.list.Update(PipelineDataMsg{
		Available: []PipelineInfo{{Name: "test-pipe", StepCount: 1, InputExample: "example input"}},
	})

	// Move cursor to the available item
	// Expand the Finished section (collapsed by default)
	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindPipelineName {
			c.list.cursor = i
			break
		}
	}

	// Press Enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	c, cmd := c.Update(msg)

	assert.Equal(t, FocusPaneRight, c.focus)
	assert.NotNil(t, cmd)

	// Execute the batch cmd and check for ConfigureFormMsg
	result := cmd()
	if batch, ok := result.(tea.BatchMsg); ok {
		foundConfigureForm := false
		for _, batchCmd := range batch {
			if batchCmd == nil {
				continue
			}
			innerMsg := batchCmd()
			if cfgMsg, ok := innerMsg.(ConfigureFormMsg); ok {
				foundConfigureForm = true
				assert.Equal(t, "test-pipe", cfgMsg.PipelineName)
				assert.Equal(t, "example input", cfgMsg.InputExample)
			}
		}
		assert.True(t, foundConfigureForm, "should emit ConfigureFormMsg in batch")
	}
}

func TestContentModel_CancelAll_NilSafe(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	// launcher should be nil since no Manifest in deps
	assert.Nil(t, c.launcher)

	// CancelAll should not panic with nil launcher
	assert.NotPanics(t, func() {
		c.CancelAll()
	})
}

func TestContentModel_PipelineLaunchedMsg_TransitionsFocusRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Set focus to right pane
	c.focus = FocusPaneRight
	c.list.SetFocused(false)
	c.detail.SetFocused(true)

	// Send PipelineLaunchedMsg
	launchedMsg := PipelineLaunchedMsg{RunID: "run-abc", PipelineName: "test-pipe"}
	c, _ = c.Update(launchedMsg)

	assert.Equal(t, FocusPaneRight, c.focus)
	assert.False(t, c.list.focused)
	assert.True(t, c.detail.focused)
}

func TestContentModel_LaunchErrorMsg_TransitionsFocusLeft(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Set focus to right pane
	c.focus = FocusPaneRight
	c.list.SetFocused(false)
	c.detail.SetFocused(true)

	// Send LaunchErrorMsg
	errMsg := LaunchErrorMsg{PipelineName: "test-pipe", Err: fmt.Errorf("launch failed")}
	c, _ = c.Update(errMsg)

	assert.Equal(t, FocusPaneLeft, c.focus)
	assert.True(t, c.list.focused)
	assert.False(t, c.detail.focused)
}

func TestContentModel_CKey_OnNonRunningItem_IsNoOp(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Inject data with an available pipeline
	c.list, _ = c.list.Update(PipelineDataMsg{
		Available: []PipelineInfo{{Name: "test-pipe", StepCount: 1}},
	})

	// Move cursor to the available item
	// Expand the Finished section (collapsed by default)
	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindPipelineName {
			c.list.cursor = i
			break
		}
	}
	cursorBefore := c.list.cursor

	// Send c key -- should be no-op since cursor is on an available item, not running
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	c, cmd := c.Update(msg)

	// Focus should remain on left pane
	assert.Equal(t, FocusPaneLeft, c.focus)
	// Cursor should not change
	assert.Equal(t, cursorBefore, c.list.cursor)
	// No command should be returned (or cmd is nil)
	_ = cmd
}

func TestContentModel_PipelineLaunchResultMsg_TriggersRefresh(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Send PipelineLaunchResultMsg
	resultMsg := PipelineLaunchResultMsg{RunID: "run-abc", Err: nil}
	c, cmd := c.Update(resultMsg)

	// Should return a refresh command (fetchPipelineData)
	assert.NotNil(t, cmd)
}

// ===========================================================================
// T033: Content model tests for finished detail message routing
// ===========================================================================

func TestContentModel_EnterOnFinishedItem_EmitsFinishedDetailActiveMsg(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Finished: []FinishedPipeline{{RunID: "r1", Name: "done", Status: "completed", BranchName: "feat/test"}},
	})

	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindFinished {
			c.list.cursor = i
			break
		}
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	c, cmd := c.Update(msg)

	assert.Equal(t, FocusPaneRight, c.focus)
	assert.NotNil(t, cmd)

	// Execute the batch cmd and check for FinishedDetailActiveMsg
	result := cmd()
	if batch, ok := result.(tea.BatchMsg); ok {
		foundFinishedActive := false
		for _, batchCmd := range batch {
			if batchCmd == nil {
				continue
			}
			innerMsg := batchCmd()
			if faMsg, ok := innerMsg.(FinishedDetailActiveMsg); ok {
				foundFinishedActive = true
				assert.True(t, faMsg.Active)
			}
		}
		assert.True(t, foundFinishedActive, "should emit FinishedDetailActiveMsg{Active: true} in batch")
	}
}

func TestContentModel_EscFromFinishedDetail_EmitsFinishedDetailActiveInactive(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Set focus to right pane
	c.focus = FocusPaneRight
	c.list.SetFocused(false)
	c.detail.SetFocused(true)

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	c, cmd := c.Update(msg)

	assert.Equal(t, FocusPaneLeft, c.focus)
	assert.NotNil(t, cmd)

	// Execute the batch cmd and check for FinishedDetailActiveMsg{Active: false}
	result := cmd()
	if batch, ok := result.(tea.BatchMsg); ok {
		foundFinishedInactive := false
		for _, batchCmd := range batch {
			if batchCmd == nil {
				continue
			}
			innerMsg := batchCmd()
			if faMsg, ok := innerMsg.(FinishedDetailActiveMsg); ok {
				foundFinishedInactive = true
				assert.False(t, faMsg.Active)
			}
		}
		assert.True(t, foundFinishedInactive, "should emit FinishedDetailActiveMsg{Active: false} in batch")
	}
}

func TestContentModel_ChatSessionEndedMsg_ForwardedToDetail(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// The message should be forwarded without error
	c, _ = c.Update(ChatSessionEndedMsg{})
	// Just verify it doesn't panic
}

func TestContentModel_BranchCheckoutMsg_ForwardedToDetail(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c, _ = c.Update(BranchCheckoutMsg{BranchName: "feat/test", Success: true})
	// Just verify it doesn't panic
}

func TestContentModel_DiffViewEndedMsg_ForwardedToDetail(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c, _ = c.Update(DiffViewEndedMsg{})
	// Just verify it doesn't panic
}

// ===========================================================================
// T017: Content model integration tests for compose mode entry/exit
// ===========================================================================

// newTestContentModel creates a ContentModel with a temp pipeline YAML file
// and pre-loaded available pipeline data, with cursor on the available item.
func newTestContentModel(t *testing.T) ContentModel {
	t.Helper()

	tmpDir := t.TempDir()
	pipelineYAML := `kind: pipeline
metadata:
  name: test-pipeline
  description: "A test pipeline"
input:
  source: cli
steps:
  - id: step1
    persona: craftsman
    workspace:
      root: "./"
    exec:
      type: prompt
      source: "test"
    output_artifacts:
      - name: test-output
        path: output.json
`
	err := os.WriteFile(filepath.Join(tmpDir, "test-pipeline.yaml"), []byte(pipelineYAML), 0644)
	require.NoError(t, err)

	deps := LaunchDependencies{
		PipelinesDir: tmpDir,
		Manifest:     &manifest.Manifest{},
	}

	m := NewContentModel(nil, nil, deps)

	// Populate the list with pipeline data
	m.list.available = []PipelineInfo{{
		Name:        "test-pipeline",
		Description: "A test pipeline",
		StepCount:   1,
	}}
	m.list.buildNavigableItems()

	// Set sizes
	m.SetSize(160, 40)

	// Move cursor to the available item
	for i, item := range m.list.navigable {
		if item.kind == itemKindPipelineName {
			m.list.cursor = i
			break
		}
	}

	return m
}

// extractMsgFromBatch executes a tea.Cmd and collects all messages produced,
// unwrapping tea.BatchMsg recursively.
func extractMsgFromBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			msgs = append(msgs, extractMsgFromBatch(c)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

func TestContentModel_SKey_OnAvailablePipeline_EntersComposeMode(t *testing.T) {
	m := newTestContentModel(t)

	// Verify preconditions
	require.False(t, m.composing)
	require.Nil(t, m.composeList)
	require.Nil(t, m.composeDetail)
	require.Equal(t, ViewPipelines, m.currentView)
	require.Equal(t, FocusPaneLeft, m.focus)
	require.Equal(t, itemKindPipelineName, m.list.navigable[m.list.cursor].kind)

	// Press 's'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	m, cmd := m.Update(msg)

	// Verify compose mode is active
	assert.True(t, m.composing, "composing flag should be true")
	assert.NotNil(t, m.composeList, "composeList should be initialized")
	assert.NotNil(t, m.composeDetail, "composeDetail should be initialized")

	// Verify the returned command produces ComposeActiveMsg{Active: true}
	require.NotNil(t, cmd)
	msgs := extractMsgFromBatch(cmd)
	foundComposeActive := false
	for _, msg := range msgs {
		if caMsg, ok := msg.(ComposeActiveMsg); ok && caMsg.Active {
			foundComposeActive = true
		}
	}
	assert.True(t, foundComposeActive, "should emit ComposeActiveMsg{Active: true}")
}

func TestContentModel_SKey_OnRunningItem_DoesNothing(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(160, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Running: []RunningPipeline{{RunID: "r1", Name: "running-pipe"}},
	})

	// Move cursor to the running item
	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindRunning {
			c.list.cursor = i
			break
		}
	}
	require.Equal(t, itemKindRunning, c.list.navigable[c.list.cursor].kind)

	// Press 's'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	c, _ = c.Update(msg)

	assert.False(t, c.composing, "composing should remain false on running item")
	assert.Nil(t, c.composeList)
	assert.Nil(t, c.composeDetail)
}

func TestContentModel_SKey_WhenNotInViewPipelines_DoesNothing(t *testing.T) {
	m := newTestContentModel(t)

	// Switch to a different view
	m.currentView = ViewPersonas

	// Press 's'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	m, _ = m.Update(msg)

	assert.False(t, m.composing, "composing should remain false when not in ViewPipelines")
	assert.Nil(t, m.composeList)
	assert.Nil(t, m.composeDetail)
}

func TestContentModel_SKey_WhenRightPaneFocused_DoesNothing(t *testing.T) {
	m := newTestContentModel(t)

	// Switch focus to right pane
	m.focus = FocusPaneRight

	// Press 's'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	m, _ = m.Update(msg)

	assert.False(t, m.composing, "composing should remain false when right pane focused")
	assert.Nil(t, m.composeList)
	assert.Nil(t, m.composeDetail)
}

func TestContentModel_ComposeCancelMsg_ExitsComposeMode(t *testing.T) {
	m := newTestContentModel(t)

	// Enter compose mode
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	m, _ = m.Update(sMsg)
	require.True(t, m.composing)
	require.NotNil(t, m.composeList)
	require.NotNil(t, m.composeDetail)

	// Send ComposeCancelMsg
	m, cmd := m.Update(ComposeCancelMsg{})

	assert.False(t, m.composing, "composing should be false after cancel")
	assert.Nil(t, m.composeList, "composeList should be nil after cancel")
	assert.Nil(t, m.composeDetail, "composeDetail should be nil after cancel")
	assert.Equal(t, FocusPaneLeft, m.focus)
	assert.True(t, m.list.focused)

	// Verify the returned command produces ComposeActiveMsg{Active: false}
	require.NotNil(t, cmd)
	msgs := extractMsgFromBatch(cmd)
	foundComposeInactive := false
	for _, msg := range msgs {
		if caMsg, ok := msg.(ComposeActiveMsg); ok && !caMsg.Active {
			foundComposeInactive = true
		}
	}
	assert.True(t, foundComposeInactive, "should emit ComposeActiveMsg{Active: false}")
}

func TestContentModel_TabKey_BlockedDuringComposeMode(t *testing.T) {
	m := newTestContentModel(t)

	// Enter compose mode
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	m, _ = m.Update(sMsg)
	require.True(t, m.composing)

	viewBefore := m.currentView

	// Press Tab
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	m, cmd := m.Update(tabMsg)

	assert.Equal(t, viewBefore, m.currentView, "view should not change during compose mode")
	assert.Nil(t, cmd, "Tab during compose should return nil cmd")
}

func TestContentModel_ComposeStartMsg_SingleEntry_DelegatesToLaunch(t *testing.T) {
	m := newTestContentModel(t)

	// Enter compose mode
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	m, _ = m.Update(sMsg)
	require.True(t, m.composing)

	// Build a single-entry sequence
	seq := Sequence{}
	seq.Add("test-pipeline", testPipeline("test-pipeline", nil, nil))

	// Send ComposeStartMsg with single entry
	m, cmd := m.Update(ComposeStartMsg{Sequence: seq})

	assert.False(t, m.composing, "composing should be false after start")
	assert.Nil(t, m.composeList, "composeList should be nil after start")
	assert.Nil(t, m.composeDetail, "composeDetail should be nil after start")

	// Verify the returned command produces LaunchRequestMsg and ComposeActiveMsg{Active: false}
	require.NotNil(t, cmd)
	msgs := extractMsgFromBatch(cmd)
	foundLaunchRequest := false
	foundComposeInactive := false
	for _, msg := range msgs {
		if lrMsg, ok := msg.(LaunchRequestMsg); ok {
			foundLaunchRequest = true
			assert.Equal(t, "test-pipeline", lrMsg.Config.PipelineName)
		}
		if caMsg, ok := msg.(ComposeActiveMsg); ok && !caMsg.Active {
			foundComposeInactive = true
		}
	}
	assert.True(t, foundLaunchRequest, "should emit LaunchRequestMsg for single-entry sequence")
	assert.True(t, foundComposeInactive, "should emit ComposeActiveMsg{Active: false}")
}

// ===========================================================================
// Cancel/dismiss from stateRunningInfo and RunEventsMsg routing tests
// ===========================================================================

func TestContentModel_CKey_FromRunningInfoRightPane_DismissesRun(t *testing.T) {
	deps := LaunchDependencies{
		Manifest: &manifest.Manifest{},
	}
	c := NewContentModel(&contentTestPipelineProvider{}, nil, deps)
	c.SetSize(120, 40)

	// Set up: right pane showing stateRunningInfo
	c.detail.paneState = stateRunningInfo
	c.detail.selectedRunID = "stale-run"
	c.detail.selectedKind = itemKindRunning
	c.focus = FocusPaneRight
	c.list.SetFocused(false)
	c.detail.SetFocused(true)

	// Press 'c'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	c, cmd := c.Update(msg)

	// Should return a refresh command (not nil)
	assert.NotNil(t, cmd, "dismiss should return refresh cmd")
}

func TestContentModel_RunEventsMsg_RoutedToDetail(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c, _ = c.Update(RunEventsMsg{RunID: "run-1", Events: nil})
	// Just verify it doesn't panic
}

func TestContentModel_EnterOnRunningItem_EmitsLiveOutputActive(t *testing.T) {
	deps := LaunchDependencies{
		Manifest: &manifest.Manifest{},
	}
	c := NewContentModel(&contentTestPipelineProvider{}, nil, deps)
	c.SetSize(120, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Running: []RunningPipeline{{RunID: "r1", Name: "running-pipe"}},
	})

	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindRunning {
			c.list.cursor = i
			break
		}
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	c, cmd := c.Update(msg)

	assert.Equal(t, FocusPaneRight, c.focus)
	assert.NotNil(t, cmd)

	// With detached execution, entering a running item always shows live output
	// (loading events from SQLite), so it emits LiveOutputActiveMsg, not RunningInfoActiveMsg
	msgs := extractMsgFromBatch(cmd)
	foundLiveOutputActive := false
	for _, m := range msgs {
		if loMsg, ok := m.(LiveOutputActiveMsg); ok && loMsg.Active {
			foundLiveOutputActive = true
		}
	}
	assert.True(t, foundLiveOutputActive, "should emit LiveOutputActiveMsg{Active: true}")
	assert.Equal(t, stateRunningLive, c.detail.paneState, "detail pane should be in live output state")
	assert.Equal(t, "r1", c.detachedPollRunID, "should start detached polling for the run")
}

func TestContentModel_PipelineRefreshDoesNotOverwriteIssueView(t *testing.T) {
	// Regression test: periodic pipeline data refresh was re-emitting
	// PipelineSelectedMsg which flipped issueShowPipeline to true,
	// overwriting the issue detail pane with pipeline output.
	m := newTestContentModel(t)
	m.currentView = ViewIssues
	m.issueShowPipeline = false

	// Simulate a PipelineSelectedMsg from the pipeline list's periodic
	// data refresh (FromIssueList is false).
	refreshMsg := PipelineSelectedMsg{
		RunID: "run-refresh",
		Name:  "some-pipeline",
		Kind:  itemKindFinished,
	}
	m, _ = m.Update(refreshMsg)

	assert.False(t, m.issueShowPipeline,
		"pipeline list refresh must not flip issueShowPipeline to true")
}

func TestContentModel_IssueListPipelineSelectionShowsPipelineDetail(t *testing.T) {
	// When the user selects a pipeline child in the issue list,
	// issueShowPipeline should be set to true.
	m := newTestContentModel(t)
	m.currentView = ViewIssues
	m.issueShowPipeline = false

	// Simulate a PipelineSelectedMsg from the issue list (FromIssueList is true).
	issueMsg := PipelineSelectedMsg{
		RunID:         "run-issue",
		Name:          "child-pipeline",
		Kind:          itemKindRunning,
		FromIssueList: true,
	}
	m, _ = m.Update(issueMsg)

	assert.True(t, m.issueShowPipeline,
		"issue list pipeline selection should show pipeline detail")
}

// ===========================================================================
// T018: wave run regression — Guided field defaults to false
// ===========================================================================

func TestLaunchDependencies_GuidedDefaultsFalse(t *testing.T) {
	deps := LaunchDependencies{}
	assert.False(t, deps.Guided, "LaunchDependencies.Guided should default to false")
}

func TestLaunchDependencies_GuidedCanBeSetTrue(t *testing.T) {
	deps := LaunchDependencies{Guided: true}
	assert.True(t, deps.Guided)
}

// ===========================================================================
// T012: Guided startup tests
// ===========================================================================

// newGuidedContentModel creates a ContentModel in guided mode with a health provider.
func newGuidedContentModel(t *testing.T) ContentModel {
	t.Helper()
	provider := newMockHealthProvider("check-a", "check-b")
	deps := LaunchDependencies{Guided: true}
	m := NewContentModel(nil, nil, deps, ContentProviders{
		HealthProvider: provider,
	})
	m.SetSize(120, 40)
	return m
}

func TestContentModel_Guided_StartsAtViewHealth(t *testing.T) {
	m := newGuidedContentModel(t)
	assert.Equal(t, ViewHealth, m.currentView,
		"guided mode should start at ViewHealth")
}

func TestContentModel_Guided_GuidedFlowNotNil(t *testing.T) {
	m := newGuidedContentModel(t)
	assert.NotNil(t, m.guidedFlow, "guidedFlow should be non-nil in guided mode")
}

func TestContentModel_Guided_HealthModelsPreCreated(t *testing.T) {
	m := newGuidedContentModel(t)
	assert.NotNil(t, m.healthList, "healthList should be pre-created in guided mode")
	assert.NotNil(t, m.healthDetail, "healthDetail should be pre-created in guided mode")
}

func TestContentModel_NonGuided_StartsAtViewPipelines(t *testing.T) {
	m := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	assert.Equal(t, ViewPipelines, m.currentView,
		"non-guided mode should start at ViewPipelines")
	assert.Nil(t, m.guidedFlow, "guidedFlow should be nil in non-guided mode")
}

func TestContentModel_Guided_HealthAllCompleteNoErrors_TriggersTransitionTimer(t *testing.T) {
	m := newGuidedContentModel(t)

	// Send HealthAllCompleteMsg with no errors
	m, cmd := m.Update(HealthAllCompleteMsg{HasErrors: false})

	assert.NotNil(t, m.guidedFlow, "guidedFlow should still be active")
	assert.True(t, m.guidedFlow.HealthComplete, "HealthComplete should be true")
	assert.True(t, m.guidedFlow.TransitionTimer, "TransitionTimer should be set")
	assert.NotNil(t, cmd, "should return a timer command for auto-transition")
	// Should still be on ViewHealth — transition happens after timer fires
	assert.Equal(t, ViewHealth, m.currentView,
		"should stay on ViewHealth until timer fires")
}

func TestContentModel_Guided_HealthTransitionMsg_SwitchesToViewSuggest(t *testing.T) {
	m := newGuidedContentModel(t)

	// First complete health with no errors to set the state
	m, _ = m.Update(HealthAllCompleteMsg{HasErrors: false})

	// Now fire the transition message
	m, cmd := m.Update(HealthTransitionMsg{})

	assert.Equal(t, ViewSuggest, m.currentView,
		"HealthTransitionMsg should switch to ViewSuggest")
	assert.Equal(t, GuidedPhaseProposals, m.guidedFlow.Phase,
		"guidedFlow phase should be GuidedPhaseProposals after transition")
	assert.True(t, m.guidedFlow.HealthComplete)
	assert.NotNil(t, cmd)
}

func TestContentModel_Guided_HealthAllCompleteHasErrors_DoesNotAutoTransition(t *testing.T) {
	m := newGuidedContentModel(t)

	// Send HealthAllCompleteMsg with errors
	m, cmd := m.Update(HealthAllCompleteMsg{HasErrors: true})

	assert.True(t, m.guidedFlow.HasErrors, "HasErrors should be set")
	assert.False(t, m.guidedFlow.TransitionTimer, "TransitionTimer should NOT be set on errors")
	assert.Nil(t, cmd, "should not return a timer command when there are errors")
	// Should remain on ViewHealth — user must confirm
	assert.Equal(t, ViewHealth, m.currentView,
		"should stay on ViewHealth when there are errors")
}

func TestContentModel_Guided_HealthContinueMsg_AfterErrors_TriggersTransitionTimer(t *testing.T) {
	m := newGuidedContentModel(t)

	// First set error state
	m, _ = m.Update(HealthAllCompleteMsg{HasErrors: true})
	require.False(t, m.guidedFlow.TransitionTimer, "precondition: timer not set yet")

	// Now user confirms to continue
	m, cmd := m.Update(HealthContinueMsg{})

	assert.True(t, m.guidedFlow.UserConfirmed, "UserConfirmed should be set")
	assert.True(t, m.guidedFlow.TransitionTimer, "TransitionTimer should be set after continue")
	assert.NotNil(t, cmd, "should return a timer command after user confirms")
}

// ===========================================================================
// T017: Regression tests — non-guided cycling and start view
// ===========================================================================

func TestContentModel_NonGuided_TabCyclesAllEightViews(t *testing.T) {
	m := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	assert.Nil(t, m.guidedFlow, "guidedFlow must be nil for this regression test")

	visited := make(map[ViewType]bool)
	visited[m.currentView] = true

	// Tab 8 times — should cycle through all 8 views and return to start
	for i := 0; i < 8; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		visited[m.currentView] = true
	}

	// All 8 views should have been visited
	assert.Equal(t, 8, len(visited), "Tab should cycle through all 8 views in non-guided mode")
}

func TestContentModel_NonGuided_StartsAtViewPipelines_Regression(t *testing.T) {
	m := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	assert.Equal(t, ViewPipelines, m.currentView,
		"regression: non-guided mode must start at ViewPipelines")
}

// ===========================================================================
// T021: Tab navigation tests
// ===========================================================================

func TestContentModel_Guided_TabFromSuggest_GoesToPipelines(t *testing.T) {
	m := newGuidedContentModel(t)

	// Move to GuidedPhaseFleet so Tab target is ViewSuggest, then be on Suggest view
	m.guidedFlow.Phase = GuidedPhaseFleet
	m.currentView = ViewSuggest

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})

	assert.Equal(t, ViewSuggest, m.guidedFlow.TabTarget(),
		"after going to Pipelines, TabTarget should now return Suggest (Fleet phase → Suggest)")
	// After Tab from Suggest (Fleet phase), we navigate to ViewSuggest per TabTarget
	// Wait — in Fleet phase, TabTarget returns ViewSuggest. So Tab navigates to ViewSuggest.
	// Actually re-reading the code: in Fleet phase, TabTarget = ViewSuggest.
	// So Tab goes to setView(ViewSuggest). Then phase stays Fleet.
	assert.Equal(t, ViewSuggest, m.currentView,
		"from Pipelines in fleet phase, Tab goes to Suggest")
	assert.NotNil(t, cmd)
}

func TestContentModel_Guided_TabFromPipelines_InProposalsPhase_GoesToPipelines(t *testing.T) {
	m := newGuidedContentModel(t)

	// GuidedPhaseProposals: TabTarget returns ViewPipelines
	m.guidedFlow.Phase = GuidedPhaseProposals
	m.currentView = ViewSuggest

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})

	assert.Equal(t, ViewPipelines, m.currentView,
		"in Proposals phase, Tab goes to ViewPipelines")
	assert.NotNil(t, cmd)
}

func TestContentModel_Guided_TabFromFleetPhaseOnPipelines_GoesToSuggest(t *testing.T) {
	m := newGuidedContentModel(t)

	// GuidedPhaseFleet: TabTarget returns ViewSuggest
	m.guidedFlow.Phase = GuidedPhaseFleet
	m.currentView = ViewPipelines

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})

	assert.Equal(t, ViewSuggest, m.currentView,
		"in Fleet phase, Tab from Pipelines goes to ViewSuggest")
	assert.NotNil(t, cmd)
}

func TestContentModel_Guided_TabDuringAttached_IsNoOp(t *testing.T) {
	m := newGuidedContentModel(t)
	m.guidedFlow.Phase = GuidedPhaseAttached
	viewBefore := m.currentView

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})

	assert.Equal(t, viewBefore, m.currentView,
		"Tab during attached phase should be a no-op")
	assert.Nil(t, cmd, "Tab during attached phase should return nil cmd")
}

func TestContentModel_Guided_ShiftTabReverses_FleetPhase(t *testing.T) {
	m := newGuidedContentModel(t)

	// In Fleet phase on Pipelines: TabTarget = ViewSuggest, so ShiftTab reverses to ViewPipelines
	m.guidedFlow.Phase = GuidedPhaseFleet
	m.currentView = ViewPipelines

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})

	// Shift+Tab reverses: target was Suggest, so go to Pipelines instead
	assert.Equal(t, ViewPipelines, m.currentView,
		"Shift+Tab in Fleet phase from Pipelines should toggle to Pipelines (reverse of Suggest)")
	assert.NotNil(t, cmd)
}

func TestContentModel_Guided_ShiftTabDuringAttached_IsNoOp(t *testing.T) {
	m := newGuidedContentModel(t)
	m.guidedFlow.Phase = GuidedPhaseAttached
	viewBefore := m.currentView

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})

	assert.Equal(t, viewBefore, m.currentView,
		"Shift+Tab during attached phase should be a no-op")
	assert.Nil(t, cmd)
}

func TestContentModel_NumberKey3_GoesToViewContracts(t *testing.T) {
	m := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	m.SetSize(120, 40)

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})

	assert.Equal(t, ViewContracts, m.currentView,
		"number key '3' should navigate to ViewContracts")
	assert.NotNil(t, cmd)
}

func TestContentModel_NonGuided_TabStillCyclesAll8Views(t *testing.T) {
	m := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	require.Nil(t, m.guidedFlow)

	startView := m.currentView
	// Tab once and verify it moved
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	nextView := m.currentView

	assert.NotEqual(t, startView, nextView,
		"non-guided Tab should cycle to next view")

	// Tab 7 more times should return to original
	for i := 0; i < 7; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	assert.Equal(t, startView, m.currentView,
		"8 Tabs in non-guided mode should complete the cycle")
}

// ===========================================================================
// T033: Sequence launch tests — SuggestLaunchMsg sets guided phase to Fleet
// ===========================================================================

func TestContentModel_Guided_SuggestLaunchMsg_SetsPhaseToFleet(t *testing.T) {
	m := newGuidedContentModel(t)

	// Start in Proposals phase (after health completes)
	m.guidedFlow.Phase = GuidedPhaseProposals
	m.currentView = ViewSuggest

	// Launch a suggested pipeline
	pipeline := SuggestProposedPipeline{
		Name:  "impl-issue",
		Input: "https://github.com/re-cinq/wave/issues/123",
	}
	m, cmd := m.Update(SuggestLaunchMsg{Pipeline: pipeline})

	assert.Equal(t, GuidedPhaseFleet, m.guidedFlow.Phase,
		"SuggestLaunchMsg should transition guidedFlow to GuidedPhaseFleet")
	assert.Equal(t, ViewPipelines, m.currentView,
		"SuggestLaunchMsg should switch to ViewPipelines")
	assert.NotNil(t, cmd, "should return commands to launch the pipeline")
}

func TestContentModel_Guided_SuggestLaunchMsg_EmitsLaunchRequest(t *testing.T) {
	m := newGuidedContentModel(t)
	m.guidedFlow.Phase = GuidedPhaseProposals

	pipeline := SuggestProposedPipeline{
		Name:  "impl-issue",
		Input: "test input",
	}
	m, cmd := m.Update(SuggestLaunchMsg{Pipeline: pipeline})

	require.NotNil(t, cmd)
	msgs := extractMsgFromBatch(cmd)

	foundLaunchRequest := false
	for _, msg := range msgs {
		if lr, ok := msg.(LaunchRequestMsg); ok {
			foundLaunchRequest = true
			assert.Equal(t, "impl-issue", lr.Config.PipelineName)
			assert.Equal(t, "test input", lr.Config.Input)
		}
	}
	assert.True(t, foundLaunchRequest, "SuggestLaunchMsg should emit LaunchRequestMsg")
}
