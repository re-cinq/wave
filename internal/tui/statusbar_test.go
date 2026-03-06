package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusBarModel_View_ContainsHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)
	view := sb.View()

	assert.Contains(t, view, "q: quit")
	assert.Contains(t, view, "ctrl+c: exit")
}

func TestStatusBarModel_View_ContainsContextLabel(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)
	view := sb.View()

	assert.Contains(t, view, "Dashboard")
}

func TestStatusBarModel_SetWidth(t *testing.T) {
	sb := NewStatusBarModel()
	assert.Equal(t, 0, sb.width)

	sb.SetWidth(80)
	assert.Equal(t, 80, sb.width)
}

func TestStatusBarModel_DefaultContextLabel(t *testing.T) {
	sb := NewStatusBarModel()
	assert.Equal(t, "Dashboard", sb.contextLabel)
}

func TestStatusBarModel_View_DefaultLeftPaneHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)
	view := sb.View()

	assert.Contains(t, view, "Enter: view")
	assert.Contains(t, view, "↑↓: navigate")
	assert.Contains(t, view, "/: filter")
}

func TestStatusBarModel_Update_FocusChangedToRight(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})
	view := sb.View()

	assert.Contains(t, view, "↑↓: scroll")
	assert.Contains(t, view, "Esc: back")
	assert.NotContains(t, view, "/: filter")
}

func TestStatusBarModel_Update_FocusChangedBackToLeft(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	// Switch to right
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})
	// Switch back to left
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneLeft})
	view := sb.View()

	assert.Contains(t, view, "↑↓: navigate")
	assert.Contains(t, view, "Enter: view")
	assert.Contains(t, view, "/: filter")
}
