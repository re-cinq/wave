package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ContentModel is the main content area component composing a left pipeline list pane and a right detail pane.
type ContentModel struct {
	width  int
	height int
	list   PipelineListModel
}

// NewContentModel creates a new content model with the given pipeline data provider.
func NewContentModel(provider PipelineDataProvider) ContentModel {
	return ContentModel{
		list: NewPipelineListModel(provider),
	}
}

// Init returns commands from child components.
func (m ContentModel) Init() tea.Cmd {
	return m.list.Init()
}

// SetSize updates the content area dimensions and propagates to children.
func (m *ContentModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	leftWidth := m.leftPaneWidth()
	m.list.SetSize(leftWidth, h)
}

// Update handles messages by forwarding to child components.
func (m ContentModel) Update(msg tea.Msg) (ContentModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the content area with left pipeline list and right detail placeholder.
func (m ContentModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	leftWidth := m.leftPaneWidth()
	rightWidth := m.width - leftWidth

	leftView := m.list.View()

	rightPlaceholder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))
	rightContent := rightPlaceholder.Render("Select a pipeline to view details")

	rightView := lipgloss.Place(rightWidth, m.height, lipgloss.Center, lipgloss.Center, rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)
}

// leftPaneWidth computes the left pane width: 30% of total, min 25, max 50.
func (m ContentModel) leftPaneWidth() int {
	w := m.width * 30 / 100
	if w < 25 {
		w = 25
	}
	if w > 50 {
		w = 50
	}
	if w > m.width {
		w = m.width
	}
	return w
}
