package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SkillDetailModel is the right pane model for the Skills view.
type SkillDetailModel struct {
	width    int
	height   int
	focused  bool
	viewport viewport.Model
	selected *SkillInfo
}

// NewSkillDetailModel creates a new skill detail model.
func NewSkillDetailModel() SkillDetailModel {
	return SkillDetailModel{
		viewport: viewport.New(0, 0),
	}
}

// Init returns nil.
func (m SkillDetailModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the model dimensions.
func (m *SkillDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if m.selected != nil {
		m.viewport.SetContent(renderSkillDetail(m.selected, w))
	}
}

// SetFocused updates the focused state.
func (m *SkillDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetSkill sets the selected skill and updates the viewport.
func (m *SkillDetailModel) SetSkill(info *SkillInfo) {
	m.selected = info
	m.viewport.SetContent(renderSkillDetail(info, m.width))
	m.viewport.GotoTop()
}

// Update handles messages.
func (m SkillDetailModel) Update(msg tea.Msg) (SkillDetailModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.focused {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the detail pane.
func (m SkillDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if m.selected == nil {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Select a skill to view details")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return m.viewport.View()
}

func renderSkillDetail(info *SkillInfo, _ int) string {
	if info == nil {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render(info.Name))
	sb.WriteString("\n")

	if info.CommandsGlob != "" {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Commands glob:"), info.CommandsGlob))
	}

	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("Command Files:"))
	sb.WriteString("\n")
	if len(info.CommandFiles) == 0 {
		sb.WriteString(fmt.Sprintf("  %s\n", labelStyle.Render("No commands found")))
	} else {
		for _, f := range info.CommandFiles {
			sb.WriteString(fmt.Sprintf("  • %s\n", f))
		}
	}

	if info.InstallCmd != "" {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Install:"), info.InstallCmd))
	}

	if info.CheckCmd != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Check:"), info.CheckCmd))
	}

	if len(info.PipelineUsage) > 0 {
		sb.WriteString("\n")
		sb.WriteString(sectionStyle.Render("Used By:"))
		sb.WriteString("\n")
		for _, name := range info.PipelineUsage {
			sb.WriteString(fmt.Sprintf("  • %s\n", name))
		}
	}

	return sb.String()
}
