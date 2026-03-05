package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// ContentModel is the main content area component.
type ContentModel struct {
	width  int
	height int
}

// NewContentModel creates a new content model.
func NewContentModel() ContentModel {
	return ContentModel{}
}

// SetSize updates the content area dimensions for reflow.
func (m *ContentModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the content area with centered placeholder text.
func (m ContentModel) View() string {
	placeholder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render("Wave TUI — Pipelines view coming soon")

	if m.width <= 0 || m.height <= 0 {
		return placeholder
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, placeholder)
}
