package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ensureSkillModels lazily creates the skill list and detail models.
func (m *ContentModel) ensureSkillModels(leftWidth, rightWidth, h int) tea.Cmd {
	var initCmd tea.Cmd
	if m.skillList == nil && m.skillProvider != nil {
		sl := NewSkillListModel(m.skillProvider)
		sl.SetSize(leftWidth, h)
		m.skillList = &sl
		sd := NewSkillDetailModel()
		sd.SetSize(rightWidth, h)
		m.skillDetail = &sd
		initCmd = m.skillList.Init()
	}
	if m.skillList != nil {
		m.skillList.SetFocused(true)
	}
	if m.skillDetail != nil {
		m.skillDetail.SetFocused(false)
	}
	return initCmd
}

// setSizeSkill propagates size changes to the skill models.
func (m *ContentModel) setSizeSkill(leftWidth, rightWidth, h int) {
	if m.skillList != nil {
		m.skillList.SetSize(leftWidth, h)
	}
	if m.skillDetail != nil {
		m.skillDetail.SetSize(rightWidth, h)
	}
}

// updateSkillMessage routes skill messages to the skill list and detail.
// Returns (model, cmd, handled). handled=true means the message belonged to this panel.
func (m ContentModel) updateSkillMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case SkillDataMsg:
		if m.skillList != nil {
			var cmd tea.Cmd
			*m.skillList, cmd = m.skillList.Update(msg)
			return m, cmd, true
		}
		return m, nil, true

	case SkillSelectedMsg:
		if m.skillList != nil {
			var listCmd tea.Cmd
			*m.skillList, listCmd = m.skillList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.skillDetail != nil && m.skillList != nil {
			for i := range m.skillList.navigable {
				if m.skillList.navigable[i].Name == msg.Name {
					m.skillDetail.SetSkill(&m.skillList.navigable[i])
					break
				}
			}
		}
		return m, tea.Batch(cmds...), true
	}
	return m, nil, false
}

// viewSkill renders the skill panel.
func (m ContentModel) viewSkill() (left, right string) {
	if m.skillList != nil {
		left = m.skillList.View()
	} else {
		left = renderPlaceholder(m.leftPaneWidth(), m.height, "Select a skill to view details")
	}
	if m.skillDetail != nil {
		right = m.skillDetail.View()
	} else {
		right = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a skill to view details")
	}
	return left, right
}
