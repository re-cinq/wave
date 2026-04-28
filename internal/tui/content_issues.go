package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ensureIssueModels lazily creates the issue list and detail models.
func (m *ContentModel) ensureIssueModels(leftWidth, rightWidth, h int) tea.Cmd {
	var initCmd tea.Cmd
	if m.issueList == nil && m.issueProvider != nil {
		il := NewIssueListModel(m.issueProvider)
		il.SetSize(leftWidth, h)
		m.issueList = &il
		id := NewIssueDetailModel()
		id.SetSize(rightWidth, h)
		// Populate available pipelines for the chooser
		if m.list.available != nil {
			id.SetPipelines(m.list.available)
		}
		m.issueDetail = &id
		initCmd = m.issueList.Init()
	}
	if m.issueList != nil {
		m.issueList.SetFocused(true)
	}
	if m.issueDetail != nil {
		m.issueDetail.SetFocused(false)
	}
	return initCmd
}

// setSizeIssue propagates size changes to the issue models.
func (m *ContentModel) setSizeIssue(leftWidth, rightWidth, h int) {
	if m.issueList != nil {
		m.issueList.SetSize(leftWidth, h)
	}
	if m.issueDetail != nil {
		m.issueDetail.SetSize(rightWidth, h)
	}
}

// updateIssueMessage routes issue messages to the issue list and detail.
// Returns (model, cmd, handled). handled=true means the message belonged to this panel.
func (m ContentModel) updateIssueMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case IssueDataMsg:
		if m.issueList != nil {
			var cmd tea.Cmd
			*m.issueList, cmd = m.issueList.Update(msg)
			return m, cmd, true
		}
		return m, nil, true

	case IssueSelectedMsg:
		// Switch back to issue detail when an issue row is selected.
		m.issueShowPipeline = false
		if m.issueList != nil {
			var listCmd tea.Cmd
			*m.issueList, listCmd = m.issueList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.issueDetail != nil && m.issueList != nil {
			if msg.Index >= 0 && msg.Index < len(m.issueList.navigable) {
				item := m.issueList.navigable[msg.Index]
				if item.issue != nil {
					m.issueDetail.SetIssue(item.issue)
				}
			}
		}
		return m, tea.Batch(cmds...), true

	case IssueLaunchMsg:
		// Convert to a LaunchRequestMsg and switch to Pipelines view
		m.currentView = ViewPipelines
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
		launchCmd := func() tea.Msg {
			return LaunchRequestMsg{Config: LaunchConfig{
				PipelineName: msg.PipelineName,
				Input:        msg.IssueURL,
			}}
		}
		viewCmd := func() tea.Msg {
			return ViewChangedMsg{View: ViewPipelines}
		}
		focusCmd := func() tea.Msg {
			return FocusChangedMsg{Pane: FocusPaneLeft}
		}
		return m, tea.Batch(launchCmd, viewCmd, focusCmd), true
	}
	return m, nil, false
}

// viewIssue renders the issue panel.
func (m ContentModel) viewIssue() (left, right string) {
	if m.issueList != nil {
		left = m.issueList.View()
	} else {
		left = renderPlaceholder(m.leftPaneWidth(), m.height, "No repository configured")
	}
	switch {
	case m.issueShowPipeline:
		// Show pipeline detail when a pipeline child is selected.
		right = m.detail.View()
	case m.issueDetail != nil:
		right = m.issueDetail.View()
	default:
		right = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select an issue to view details")
	}
	return left, right
}
