package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleAlternativeViewEnter handles Enter key in alternative view left panes.
func (m ContentModel) handleAlternativeViewEnter() (ContentModel, tea.Cmd) {
	m.focus = FocusPaneRight

	switch m.currentView {
	case ViewPersonas:
		if m.personaList != nil {
			m.personaList.SetFocused(false)
		}
		if m.personaDetail != nil {
			m.personaDetail.SetFocused(true)
		}
	case ViewContracts:
		if m.contractList != nil {
			m.contractList.SetFocused(false)
		}
		if m.contractDetail != nil {
			m.contractDetail.SetFocused(true)
		}
	case ViewSkills:
		if m.skillList != nil {
			m.skillList.SetFocused(false)
		}
		if m.skillDetail != nil {
			m.skillDetail.SetFocused(true)
		}
	case ViewHealth:
		if m.healthList != nil {
			m.healthList.SetFocused(false)
		}
		if m.healthDetail != nil {
			m.healthDetail.SetFocused(true)
		}
	case ViewIssues:
		if m.issueList != nil {
			m.issueList.SetFocused(false)
		}
		if m.issueShowPipeline {
			m.detail.SetFocused(true)
		} else if m.issueDetail != nil {
			m.issueDetail.SetFocused(true)
		}
	case ViewPullRequests:
		if m.prList != nil {
			m.prList.SetFocused(false)
		}
		if m.prDetail != nil {
			m.prDetail.SetFocused(true)
		}
	case ViewSuggest:
		if m.suggestList != nil {
			m.suggestList.SetFocused(false)
		}
		if m.suggestDetail != nil {
			m.suggestDetail.SetFocused(true)
		}
	}

	return m, func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneRight} }
}

// handleAlternativeViewEscape handles Escape key in alternative view right panes.
func (m ContentModel) handleAlternativeViewEscape() (ContentModel, tea.Cmd) {
	m.focus = FocusPaneLeft

	switch m.currentView {
	case ViewPersonas:
		if m.personaList != nil {
			m.personaList.SetFocused(true)
		}
		if m.personaDetail != nil {
			m.personaDetail.SetFocused(false)
		}
	case ViewContracts:
		if m.contractList != nil {
			m.contractList.SetFocused(true)
		}
		if m.contractDetail != nil {
			m.contractDetail.SetFocused(false)
		}
	case ViewSkills:
		if m.skillList != nil {
			m.skillList.SetFocused(true)
		}
		if m.skillDetail != nil {
			m.skillDetail.SetFocused(false)
		}
	case ViewHealth:
		if m.healthList != nil {
			m.healthList.SetFocused(true)
		}
		if m.healthDetail != nil {
			m.healthDetail.SetFocused(false)
		}
	case ViewIssues:
		if m.issueList != nil {
			m.issueList.SetFocused(true)
		}
		if m.issueShowPipeline {
			m.detail.SetFocused(false)
		} else if m.issueDetail != nil {
			m.issueDetail.SetFocused(false)
		}
	case ViewPullRequests:
		if m.prList != nil {
			m.prList.SetFocused(true)
		}
		if m.prDetail != nil {
			m.prDetail.SetFocused(false)
		}
	case ViewSuggest:
		if m.suggestList != nil {
			m.suggestList.SetFocused(true)
		}
		if m.suggestDetail != nil {
			m.suggestDetail.SetFocused(false)
		}
	}

	return m, func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} }
}

// routeToActiveList routes a key message to the active view's list model.
func (m ContentModel) routeToActiveList(msg tea.Msg) (ContentModel, tea.Cmd) {
	switch m.currentView {
	case ViewPipelines:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case ViewPersonas:
		if m.personaList != nil {
			var cmd tea.Cmd
			*m.personaList, cmd = m.personaList.Update(msg)
			return m, cmd
		}
	case ViewContracts:
		if m.contractList != nil {
			var cmd tea.Cmd
			*m.contractList, cmd = m.contractList.Update(msg)
			return m, cmd
		}
	case ViewSkills:
		if m.skillList != nil {
			var cmd tea.Cmd
			*m.skillList, cmd = m.skillList.Update(msg)
			return m, cmd
		}
	case ViewHealth:
		if m.healthList != nil {
			var cmd tea.Cmd
			*m.healthList, cmd = m.healthList.Update(msg)
			return m, cmd
		}
	case ViewIssues:
		if m.issueList != nil {
			var cmd tea.Cmd
			*m.issueList, cmd = m.issueList.Update(msg)
			return m, cmd
		}
	case ViewPullRequests:
		if m.prList != nil {
			var cmd tea.Cmd
			*m.prList, cmd = m.prList.Update(msg)
			return m, cmd
		}
	case ViewSuggest:
		if m.suggestList != nil {
			var cmd tea.Cmd
			*m.suggestList, cmd = m.suggestList.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// routeToActiveDetail routes a key message to the active view's detail model.
func (m ContentModel) routeToActiveDetail(msg tea.Msg) (ContentModel, tea.Cmd) {
	switch m.currentView {
	case ViewPipelines:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	case ViewPersonas:
		if m.personaDetail != nil {
			var cmd tea.Cmd
			*m.personaDetail, cmd = m.personaDetail.Update(msg)
			return m, cmd
		}
	case ViewContracts:
		if m.contractDetail != nil {
			var cmd tea.Cmd
			*m.contractDetail, cmd = m.contractDetail.Update(msg)
			return m, cmd
		}
	case ViewSkills:
		if m.skillDetail != nil {
			var cmd tea.Cmd
			*m.skillDetail, cmd = m.skillDetail.Update(msg)
			return m, cmd
		}
	case ViewHealth:
		if m.healthDetail != nil {
			var cmd tea.Cmd
			*m.healthDetail, cmd = m.healthDetail.Update(msg)
			return m, cmd
		}
	case ViewIssues:
		if m.issueShowPipeline {
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}
		if m.issueDetail != nil {
			var cmd tea.Cmd
			*m.issueDetail, cmd = m.issueDetail.Update(msg)
			return m, cmd
		}
	case ViewPullRequests:
		if m.prDetail != nil {
			var cmd tea.Cmd
			*m.prDetail, cmd = m.prDetail.Update(msg)
			return m, cmd
		}
	case ViewSuggest:
		if m.suggestDetail != nil {
			var cmd tea.Cmd
			*m.suggestDetail, cmd = m.suggestDetail.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}
