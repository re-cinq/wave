package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentModel_View_ContainsPlaceholder(t *testing.T) {
	c := NewContentModel()
	view := c.View()
	assert.Contains(t, view, "Pipelines view coming soon")
}

func TestContentModel_SetSize(t *testing.T) {
	c := NewContentModel()
	assert.Equal(t, 0, c.width)
	assert.Equal(t, 0, c.height)

	c.SetSize(120, 40)
	assert.Equal(t, 120, c.width)
	assert.Equal(t, 40, c.height)
}

func TestContentModel_View_CentersContent(t *testing.T) {
	c := NewContentModel()
	c.SetSize(120, 40)
	view := c.View()

	// With centering, the view should contain the placeholder text
	assert.Contains(t, view, "Pipelines view coming soon")
	// The view should contain spaces/newlines for centering
	assert.Greater(t, len(view), len("Wave TUI — Pipelines view coming soon"))
}

func TestContentModel_View_ZeroDimensions(t *testing.T) {
	c := NewContentModel()
	// With zero dimensions, should still return placeholder text
	view := c.View()
	assert.Contains(t, view, "Pipelines view coming soon")
}
