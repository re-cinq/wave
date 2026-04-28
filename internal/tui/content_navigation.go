package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// cycleView moves to the next view and returns init commands if the view was just created.
func (m *ContentModel) cycleView() tea.Cmd {
	m.currentView = (m.currentView + 1) % 9
	m.focus = FocusPaneLeft

	var initCmd tea.Cmd

	leftWidth := m.leftPaneWidth()
	rightWidth := m.width - leftWidth - 3
	ch := m.childHeight()

	switch m.currentView {
	case ViewPipelines:
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
	case ViewPersonas:
		initCmd = m.ensurePersonaModels(leftWidth, rightWidth, ch)
	case ViewContracts:
		initCmd = m.ensureContractModels(leftWidth, rightWidth, ch)
	case ViewSkills:
		initCmd = m.ensureSkillModels(leftWidth, rightWidth, ch)
	case ViewHealth:
		initCmd = m.ensureHealthModels(leftWidth, rightWidth, ch)
	case ViewIssues:
		initCmd = m.ensureIssueModels(leftWidth, rightWidth, ch)
	case ViewPullRequests:
		initCmd = m.ensurePRModels(leftWidth, rightWidth, ch)
	case ViewSuggest:
		initCmd = m.ensureSuggestModels(leftWidth, rightWidth, ch)
	case ViewOntology:
		initCmd = m.ensureOntologyModels(leftWidth, rightWidth, ch)
	}

	batchCmds := []tea.Cmd{
		func() tea.Msg { return ViewChangedMsg{View: m.currentView} },
		func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} },
	}
	if initCmd != nil {
		batchCmds = append(batchCmds, initCmd)
	}

	return tea.Batch(batchCmds...)
}

// setView switches directly to a specific view, lazy-initializing models as needed.
func (m *ContentModel) setView(v ViewType) tea.Cmd {
	m.currentView = v
	m.focus = FocusPaneLeft

	var initCmd tea.Cmd
	leftWidth := m.leftPaneWidth()
	rightWidth := m.width - leftWidth - 3
	ch := m.childHeight()

	switch v {
	case ViewPipelines:
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
	case ViewPersonas:
		initCmd = m.ensurePersonaModels(leftWidth, rightWidth, ch)
	case ViewContracts:
		initCmd = m.ensureContractModels(leftWidth, rightWidth, ch)
	case ViewSkills:
		initCmd = m.ensureSkillModels(leftWidth, rightWidth, ch)
	case ViewHealth:
		initCmd = m.ensureHealthModels(leftWidth, rightWidth, ch)
	case ViewIssues:
		initCmd = m.ensureIssueModels(leftWidth, rightWidth, ch)
	case ViewPullRequests:
		initCmd = m.ensurePRModels(leftWidth, rightWidth, ch)
	case ViewSuggest:
		initCmd = m.ensureSuggestModels(leftWidth, rightWidth, ch)
	case ViewOntology:
		initCmd = m.ensureOntologyModels(leftWidth, rightWidth, ch)
	}

	batchCmds := []tea.Cmd{
		func() tea.Msg { return ViewChangedMsg{View: m.currentView} },
		func() tea.Msg { return FocusChangedMsg{Pane: FocusPaneLeft} },
	}
	if initCmd != nil {
		batchCmds = append(batchCmds, initCmd)
	}
	return tea.Batch(batchCmds...)
}

// numberKeyToView maps number key strings to view types.
func numberKeyToView(key string) (ViewType, bool) {
	switch key {
	case "1":
		return ViewPipelines, true
	case "2":
		return ViewPersonas, true
	case "3":
		return ViewContracts, true
	case "4":
		return ViewSkills, true
	case "5":
		return ViewHealth, true
	case "6":
		return ViewIssues, true
	case "7":
		return ViewPullRequests, true
	case "8":
		return ViewSuggest, true
	case "9":
		return ViewOntology, true
	default:
		return 0, false
	}
}
