package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ensureHealthModels lazily creates the health list and detail models.
func (m *ContentModel) ensureHealthModels(leftWidth, rightWidth, h int) tea.Cmd {
	var initCmd tea.Cmd
	if m.healthList == nil && m.healthProvider != nil {
		hl := NewHealthListModel(m.healthProvider)
		hl.SetSize(leftWidth, h)
		m.healthList = &hl
		hd := NewHealthDetailModel()
		hd.SetSize(rightWidth, h)
		m.healthDetail = &hd
		initCmd = m.healthList.Init()
	}
	if m.healthList != nil {
		m.healthList.SetFocused(true)
	}
	if m.healthDetail != nil {
		m.healthDetail.SetFocused(false)
	}
	return initCmd
}

// setSizeHealth propagates size changes to the health models.
func (m *ContentModel) setSizeHealth(leftWidth, rightWidth, h int) {
	if m.healthList != nil {
		m.healthList.SetSize(leftWidth, h)
	}
	if m.healthDetail != nil {
		m.healthDetail.SetSize(rightWidth, h)
	}
}

// updateHealthMessage routes health messages to the health list and detail.
// Returns (model, cmd, handled). handled=true means the message belonged to this panel.
func (m ContentModel) updateHealthMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case HealthCheckResultMsg:
		if m.healthList != nil {
			var listCmd tea.Cmd
			*m.healthList, listCmd = m.healthList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.healthDetail != nil {
			var detailCmd tea.Cmd
			*m.healthDetail, detailCmd = m.healthDetail.Update(msg)
			if detailCmd != nil {
				cmds = append(cmds, detailCmd)
			}
		}
		return m, tea.Batch(cmds...), true

	case HealthSelectedMsg:
		if m.healthList != nil && msg.Index < len(m.healthList.checks) {
			check := m.healthList.checks[msg.Index]
			if m.healthDetail != nil {
				m.healthDetail.SetCheck(&check)
			}
		}
		return m, nil, true

	case HealthAllCompleteMsg:
		return m, nil, true

	case HealthContinueMsg:
		return m, nil, true
	}
	return m, nil, false
}

// viewHealth renders the health panel.
func (m ContentModel) viewHealth() (left, right string) {
	if m.healthList != nil {
		left = m.healthList.View()
	} else {
		left = renderPlaceholder(m.leftPaneWidth(), m.height, "Select a health check to view details")
	}
	if m.healthDetail != nil {
		right = m.healthDetail.View()
	} else {
		right = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a health check to view details")
	}
	return left, right
}
