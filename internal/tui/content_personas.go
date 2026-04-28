package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ensurePersonaModels lazily creates the persona list and detail models.
func (m *ContentModel) ensurePersonaModels(leftWidth, rightWidth, h int) tea.Cmd {
	var initCmd tea.Cmd
	if m.personaList == nil && m.personaProvider != nil {
		pl := NewPersonaListModel(m.personaProvider)
		pl.SetSize(leftWidth, h)
		m.personaList = &pl
		pd := NewPersonaDetailModel(m.personaProvider)
		pd.SetSize(rightWidth, h)
		m.personaDetail = &pd
		initCmd = m.personaList.Init()
	}
	if m.personaList != nil {
		m.personaList.SetFocused(true)
	}
	if m.personaDetail != nil {
		m.personaDetail.SetFocused(false)
	}
	return initCmd
}

// setSizePersona propagates size changes to the persona models.
func (m *ContentModel) setSizePersona(leftWidth, rightWidth, h int) {
	if m.personaList != nil {
		m.personaList.SetSize(leftWidth, h)
	}
	if m.personaDetail != nil {
		m.personaDetail.SetSize(rightWidth, h)
	}
}

// updatePersonaMessage routes persona messages to the persona list and detail.
// Returns (model, cmd, handled). handled=true means the message belonged to this panel.
func (m ContentModel) updatePersonaMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case PersonaDataMsg:
		if m.personaList != nil {
			var cmd tea.Cmd
			*m.personaList, cmd = m.personaList.Update(msg)
			return m, cmd, true
		}
		return m, nil, true

	case PersonaSelectedMsg:
		if m.personaList != nil {
			var listCmd tea.Cmd
			*m.personaList, listCmd = m.personaList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.personaDetail != nil {
			// Find the matching persona and set it on the detail model
			if m.personaList != nil {
				for i := range m.personaList.navigable {
					if m.personaList.navigable[i].Name == msg.Name {
						m.personaDetail.SetPersona(&m.personaList.navigable[i])
						break
					}
				}
			}
			var detailCmd tea.Cmd
			*m.personaDetail, detailCmd = m.personaDetail.Update(msg)
			if detailCmd != nil {
				cmds = append(cmds, detailCmd)
			}
		}
		return m, tea.Batch(cmds...), true

	case PersonaStatsMsg:
		if m.personaDetail != nil {
			var cmd tea.Cmd
			*m.personaDetail, cmd = m.personaDetail.Update(msg)
			return m, cmd, true
		}
		return m, nil, true
	}
	return m, nil, false
}

// viewPersona renders the persona panel.
func (m ContentModel) viewPersona() (left, right string) {
	if m.personaList != nil {
		left = m.personaList.View()
	} else {
		left = renderPlaceholder(m.leftPaneWidth(), m.height, "Select a persona to view details")
	}
	if m.personaDetail != nil {
		right = m.personaDetail.View()
	} else {
		right = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a persona to view details")
	}
	return left, right
}
