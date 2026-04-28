package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ensureContractModels lazily creates the contract list and detail models.
func (m *ContentModel) ensureContractModels(leftWidth, rightWidth, h int) tea.Cmd {
	var initCmd tea.Cmd
	if m.contractList == nil && m.contractProvider != nil {
		cl := NewContractListModel(m.contractProvider)
		cl.SetSize(leftWidth, h)
		m.contractList = &cl
		cd := NewContractDetailModel()
		cd.SetSize(rightWidth, h)
		m.contractDetail = &cd
		initCmd = m.contractList.Init()
	}
	if m.contractList != nil {
		m.contractList.SetFocused(true)
	}
	if m.contractDetail != nil {
		m.contractDetail.SetFocused(false)
	}
	return initCmd
}

// setSizeContract propagates size changes to the contract models.
func (m *ContentModel) setSizeContract(leftWidth, rightWidth, h int) {
	if m.contractList != nil {
		m.contractList.SetSize(leftWidth, h)
	}
	if m.contractDetail != nil {
		m.contractDetail.SetSize(rightWidth, h)
	}
}

// updateContractMessage routes contract messages to the contract list and detail.
// Returns (model, cmd, handled). handled=true means the message belonged to this panel.
func (m ContentModel) updateContractMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case ContractDataMsg:
		if m.contractList != nil {
			var cmd tea.Cmd
			*m.contractList, cmd = m.contractList.Update(msg)
			return m, cmd, true
		}
		return m, nil, true

	case ContractSelectedMsg:
		if m.contractList != nil {
			var listCmd tea.Cmd
			*m.contractList, listCmd = m.contractList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.contractDetail != nil && m.contractList != nil {
			for i := range m.contractList.navigable {
				if m.contractList.navigable[i].Label == msg.Label {
					m.contractDetail.SetContract(&m.contractList.navigable[i])
					break
				}
			}
		}
		return m, tea.Batch(cmds...), true
	}
	return m, nil, false
}

// viewContract renders the contract panel.
func (m ContentModel) viewContract() (left, right string) {
	if m.contractList != nil {
		left = m.contractList.View()
	} else {
		left = renderPlaceholder(m.leftPaneWidth(), m.height, "Select a contract to view details")
	}
	if m.contractDetail != nil {
		right = m.contractDetail.View()
	} else {
		right = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a contract to view details")
	}
	return left, right
}
