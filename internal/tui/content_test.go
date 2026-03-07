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
	assert.Equal(t, 39, c.list.height)
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
	assert.Equal(t, 39, c.detail.height)
}

func TestContentModel_EnterOnAvailableItemTransitionsFocusRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Inject data with an available pipeline
	c.list, _ = c.list.Update(PipelineDataMsg{
		Available: []PipelineInfo{{Name: "test-pipe", StepCount: 1}},
	})

	// Move cursor to the available item (past Running(0), Finished(0), Available(1) headers)
	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindAvailable {
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

	// Expand the Finished section (collapsed by default)
	c.list.collapsed[2] = false
	c.list.buildNavigableItems()

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

func TestContentModel_EnterOnSectionHeaderDoesNotTransition(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Available: []PipelineInfo{{Name: "test"}},
	})

	// Cursor starts on a section header
	assert.Equal(t, itemKindSectionHeader, c.list.navigable[c.list.cursor].kind)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	c, _ = c.Update(msg)

	assert.Equal(t, FocusPaneLeft, c.focus)
}

func TestContentModel_EnterOnRunningItemTransitionsFocusRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Running: []RunningPipeline{{RunID: "r1", Name: "running-pipe"}},
	})

	// Move to the running item
	// Expand the Finished section (collapsed by default)
	c.list.collapsed[2] = false
	c.list.buildNavigableItems()

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
	c.list.collapsed[2] = false
	c.list.buildNavigableItems()

	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindAvailable {
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
	c.list.collapsed[2] = false
	c.list.buildNavigableItems()

	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindAvailable {
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
	c.list.collapsed[2] = false
	c.list.buildNavigableItems()

	for i := 0; i < len(c.list.navigable); i++ {
		if c.list.navigable[i].kind == itemKindAvailable {
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

	// Expand the Finished section (collapsed by default)
	c.list.collapsed[2] = false
	c.list.buildNavigableItems()

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
		if item.kind == itemKindAvailable {
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
	require.Equal(t, itemKindAvailable, m.list.navigable[m.list.cursor].kind)

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

func TestContentModel_SKey_OnNonAvailableItem_DoesNothing(t *testing.T) {
	m := newTestContentModel(t)

	// Move cursor to a section header (index 0 is always a section header)
	m.list.cursor = 0
	require.Equal(t, itemKindSectionHeader, m.list.navigable[m.list.cursor].kind)

	// Press 's'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	m, _ = m.Update(msg)

	assert.False(t, m.composing, "composing should remain false on section header")
	assert.Nil(t, m.composeList)
	assert.Nil(t, m.composeDetail)
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
