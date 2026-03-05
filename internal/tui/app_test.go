package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewAppModel_InitialState(t *testing.T) {
	m := NewAppModel()
	assert.False(t, m.ready)
	assert.False(t, m.shuttingDown)
	assert.Equal(t, 0, m.width)
	assert.Equal(t, 0, m.height)
	assert.Equal(t, "Dashboard", m.statusBar.contextLabel)
}

func TestAppModel_Init_ReturnsNil(t *testing.T) {
	m := NewAppModel()
	cmd := m.Init()
	assert.Nil(t, cmd)
}

func TestAppModel_Update_WindowSizeMsg(t *testing.T) {
	m := NewAppModel()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	updated, cmd := m.Update(msg)
	model := updated.(AppModel)

	assert.Nil(t, cmd)
	assert.True(t, model.ready)
	assert.Equal(t, 120, model.width)
	assert.Equal(t, 40, model.height)
	assert.Equal(t, 120, model.header.width)
	assert.Equal(t, 120, model.statusBar.width)
	assert.Equal(t, 120, model.content.width)
	assert.Equal(t, 40-headerHeight-statusBarHeight, model.content.height)
}

func TestAppModel_Update_QuitOnQ(t *testing.T) {
	m := NewAppModel()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	_, cmd := m.Update(msg)
	assert.NotNil(t, cmd)

	// tea.Quit returns a special quit message
	quitMsg := cmd()
	assert.IsType(t, tea.QuitMsg{}, quitMsg)
}

func TestAppModel_Update_CtrlC_SetsShuttingDown(t *testing.T) {
	m := NewAppModel()
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}

	updated, cmd := m.Update(msg)
	model := updated.(AppModel)

	assert.True(t, model.shuttingDown)
	assert.NotNil(t, cmd)
	quitMsg := cmd()
	assert.IsType(t, tea.QuitMsg{}, quitMsg)
}

func TestAppModel_View_BeforeReady(t *testing.T) {
	m := NewAppModel()
	view := m.View()
	assert.Equal(t, "Initializing...", view)
}

func TestAppModel_View_AfterReady(t *testing.T) {
	m := NewAppModel()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	view := model.View()

	// Should contain header content
	assert.Contains(t, view, "Pipeline Orchestrator")
	// Should contain content placeholder
	assert.Contains(t, view, "Pipelines view coming soon")
	// Should contain status bar hints
	assert.Contains(t, view, "q: quit")
	assert.Contains(t, view, "ctrl+c: exit")
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
			m := NewAppModel()
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
	m := NewAppModel()
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	view := model.View()
	// At exactly minimum size, should render normally, not show degradation
	assert.NotContains(t, view, "Terminal too small")
	assert.Contains(t, view, "Pipeline Orchestrator")
}
