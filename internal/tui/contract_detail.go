package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ContractDetailModel is the right pane model for the Contracts view.
type ContractDetailModel struct {
	width    int
	height   int
	focused  bool
	viewport viewport.Model
	selected *ContractInfo
}

// NewContractDetailModel creates a new contract detail model.
func NewContractDetailModel() ContractDetailModel {
	return ContractDetailModel{
		viewport: viewport.New(0, 0),
	}
}

// Init returns nil.
func (m ContractDetailModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the model dimensions.
func (m *ContractDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if m.selected != nil {
		m.viewport.SetContent(renderContractDetail(m.selected, w))
	}
}

// SetFocused updates the focused state.
func (m *ContractDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetContract sets the selected contract and updates the viewport.
func (m *ContractDetailModel) SetContract(info *ContractInfo) {
	m.selected = info
	m.viewport.SetContent(renderContractDetail(info, m.width))
	m.viewport.GotoTop()
}

// Update handles messages.
func (m ContractDetailModel) Update(msg tea.Msg) (ContractDetailModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && m.focused {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the detail pane.
func (m ContractDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if m.selected == nil {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Select a contract to view details")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return m.viewport.View()
}

func renderContractDetail(info *ContractInfo, _ int) string {
	if info == nil {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render(info.Label))
	sb.WriteString("\n")

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Type:"), info.Type))

	if info.SchemaPath != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Schema:"), info.SchemaPath))
	}

	if info.SchemaPreview != "" {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Schema Preview:"))
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render(info.SchemaPreview))
		sb.WriteString("\n")
	}

	if len(info.PipelineUsage) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Used By:"))
		sb.WriteString("\n")
		for _, ref := range info.PipelineUsage {
			sb.WriteString(fmt.Sprintf("  • %s / %s\n", ref.PipelineName, ref.StepID))
		}
	}

	return sb.String()
}
