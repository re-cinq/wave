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
