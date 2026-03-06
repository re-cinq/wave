package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stretchr/testify/assert"
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
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
	assert.True(t, c.list.focused)
}

func TestContentModel_SetSize(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
	assert.Equal(t, 0, c.width)
	assert.Equal(t, 0, c.height)

	c.SetSize(120, 40)
	assert.Equal(t, 120, c.width)
	assert.Equal(t, 40, c.height)
}

func TestContentModel_SetSize_PropagatesListDimensions(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
	c.SetSize(120, 40)

	// Left pane: 30% of 120 = 36, clamped to [25, 50] -> 36
	assert.Equal(t, 36, c.list.width)
	assert.Equal(t, 40, c.list.height)
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
			c := NewContentModel(&contentTestPipelineProvider{}, nil)
			c.SetSize(tt.width, 40)
			assert.Equal(t, tt.expected, c.list.width)
		})
	}
}

func TestContentModel_View_RightPanePlaceholder(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
	c.SetSize(120, 40)
	view := c.View()
	assert.Contains(t, view, "Select a pipeline to view details")
}

func TestContentModel_View_ZeroDimensions(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
	view := c.View()
	assert.Equal(t, "", view)
}

func TestContentModel_Init_ReturnsCommands(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
	cmd := c.Init()
	assert.NotNil(t, cmd)
}

func TestContentModel_FocusStartsOnLeft(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
	assert.Equal(t, FocusPaneLeft, c.focus)
	assert.True(t, c.list.focused)
}

func TestContentModel_SetSize_PropagatesDetailDimensions(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
	c.SetSize(120, 40)

	// Right pane: 120 - 36 = 84
	assert.Equal(t, 84, c.detail.width)
	assert.Equal(t, 40, c.detail.height)
}

func TestContentModel_EnterOnAvailableItemTransitionsFocusRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
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

	// Verify FocusChangedMsg is emitted
	result := cmd()
	fcm, ok := result.(FocusChangedMsg)
	assert.True(t, ok)
	assert.Equal(t, FocusPaneRight, fcm.Pane)
}

func TestContentModel_EnterOnFinishedItemTransitionsFocusRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
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

func TestContentModel_EnterOnSectionHeaderDoesNotTransition(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
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

func TestContentModel_EnterOnRunningItemDoesNotTransition(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
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

	assert.Equal(t, FocusPaneLeft, c.focus)
}

func TestContentModel_EscFromRightPaneReturnsFocusLeft(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
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

	result := cmd()
	fcm, ok := result.(FocusChangedMsg)
	assert.True(t, ok)
	assert.Equal(t, FocusPaneLeft, fcm.Pane)
}

func TestContentModel_ArrowKeysInRightPaneDoNotMoveList(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil)
	c.SetSize(120, 40)

	c.list, _ = c.list.Update(PipelineDataMsg{
		Available: []PipelineInfo{{Name: "pipe1"}, {Name: "pipe2"}},
	})

	// Move cursor to first available item
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
