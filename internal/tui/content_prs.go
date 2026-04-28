package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ensurePRModels lazily creates the PR list and detail models.
func (m *ContentModel) ensurePRModels(leftWidth, rightWidth, h int) tea.Cmd {
	var initCmd tea.Cmd
	if m.prList == nil && m.prProvider != nil {
		pl := NewPRListModel(m.prProvider)
		pl.SetSize(leftWidth, h)
		m.prList = &pl
		pd := NewPRDetailModel()
		pd.SetSize(rightWidth, h)
		m.prDetail = &pd
		initCmd = m.prList.Init()
	}
	if m.prList != nil {
		m.prList.SetFocused(true)
	}
	if m.prDetail != nil {
		m.prDetail.SetFocused(false)
	}
	return initCmd
}

// setSizePR propagates size changes to the PR models.
func (m *ContentModel) setSizePR(leftWidth, rightWidth, h int) {
	if m.prList != nil {
		m.prList.SetSize(leftWidth, h)
	}
	if m.prDetail != nil {
		m.prDetail.SetSize(rightWidth, h)
	}
}

// updatePRMessage routes PR messages to the PR list and detail.
// Returns (model, cmd, handled). handled=true means the message belonged to this panel.
func (m ContentModel) updatePRMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case PRDataMsg:
		if m.prList != nil {
			var cmd tea.Cmd
			*m.prList, cmd = m.prList.Update(msg)
			return m, cmd, true
		}
		return m, nil, true

	case PRSelectedMsg:
		if m.prList != nil {
			var listCmd tea.Cmd
			*m.prList, listCmd = m.prList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.prDetail != nil && m.prList != nil {
			if msg.Index >= 0 && msg.Index < len(m.prList.navigable) {
				prIdx := m.prList.navigable[msg.Index]
				m.prDetail.SetPR(&m.prList.prs[prIdx])
			}
		}
		return m, tea.Batch(cmds...), true
	}
	return m, nil, false
}

// viewPR renders the PR panel.
func (m ContentModel) viewPR() (left, right string) {
	if m.prList != nil {
		left = m.prList.View()
	} else {
		left = renderPlaceholder(m.leftPaneWidth(), m.height, "No repository configured")
	}
	if m.prDetail != nil {
		right = m.prDetail.View()
	} else {
		right = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a pull request to view details")
	}
	return left, right
}
