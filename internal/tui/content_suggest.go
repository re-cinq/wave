package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ensureSuggestModels lazily creates the suggest list and detail models.
func (m *ContentModel) ensureSuggestModels(leftWidth, rightWidth, h int) tea.Cmd {
	var initCmd tea.Cmd
	if m.suggestList == nil && m.suggestProvider != nil {
		sl := NewSuggestListModel(m.suggestProvider)
		sl.SetSize(leftWidth, h)
		m.suggestList = &sl
		sd := NewSuggestDetailModel()
		sd.SetSize(rightWidth, h)
		m.suggestDetail = &sd
		initCmd = m.suggestList.Init()
	}
	if m.suggestList != nil {
		m.suggestList.SetFocused(true)
	}
	if m.suggestDetail != nil {
		m.suggestDetail.SetFocused(false)
	}
	return initCmd
}

// setSizeSuggest propagates size changes to the suggest models.
func (m *ContentModel) setSizeSuggest(leftWidth, rightWidth, h int) {
	if m.suggestList != nil {
		m.suggestList.SetSize(leftWidth, h)
	}
	if m.suggestDetail != nil {
		m.suggestDetail.SetSize(rightWidth, h)
	}
}

// updateSuggestMessage routes suggest messages to the suggest list and detail.
// Returns (model, cmd, handled). handled=true means the message belonged to this panel.
func (m ContentModel) updateSuggestMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case SuggestDataMsg:
		if m.suggestList != nil {
			var cmd tea.Cmd
			*m.suggestList, cmd = m.suggestList.Update(msg)
			return m, cmd, true
		}
		return m, nil, true

	case SuggestLaunchedMsg:
		if m.suggestList != nil {
			var cmd tea.Cmd
			*m.suggestList, cmd = m.suggestList.Update(msg)
			return m, cmd, true
		}
		return m, nil, true

	case SuggestSelectedMsg:
		if m.suggestList != nil {
			var listCmd tea.Cmd
			*m.suggestList, listCmd = m.suggestList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.suggestDetail != nil {
			var detailCmd tea.Cmd
			*m.suggestDetail, detailCmd = m.suggestDetail.Update(msg)
			if detailCmd != nil {
				cmds = append(cmds, detailCmd)
			}
		}
		return m, tea.Batch(cmds...), true

	case SuggestLaunchMsg:
		// Switch to Pipelines view and launch the suggested pipeline
		m.currentView = ViewPipelines
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)
		pipelineName := msg.Pipeline.Name
		launchCmd := func() tea.Msg {
			return LaunchRequestMsg{Config: LaunchConfig{
				PipelineName: pipelineName,
				Input:        msg.Pipeline.Input,
			}}
		}
		viewCmd := func() tea.Msg {
			return ViewChangedMsg{View: ViewPipelines}
		}
		focusCmd := func() tea.Msg {
			return FocusChangedMsg{Pane: FocusPaneLeft}
		}
		launchedCmd := func() tea.Msg {
			return SuggestLaunchedMsg{Name: pipelineName}
		}
		refreshCmd := tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
			return PipelineRefreshTickMsg{}
		})
		return m, tea.Batch(launchCmd, viewCmd, focusCmd, launchedCmd, refreshCmd), true

	case SuggestComposeMsg:
		// Bridge suggest multi-select to compose mode: switch to Pipelines view,
		// enter compose mode with the selected proposals pre-populated.
		m.currentView = ViewPipelines
		m.focus = FocusPaneLeft
		m.list.SetFocused(true)
		m.detail.SetFocused(false)

		if len(msg.Pipelines) > 0 && m.launcher != nil {
			// Build compose sequence from selected proposals
			var seq Sequence
			for _, p := range msg.Pipelines {
				loaded, err := LoadPipelineByName(m.launcher.deps.PipelinesDir, p.Name)
				if err == nil {
					seq.Add(p.Name, loaded)
				} else {
					seq.Add(p.Name, nil)
				}
			}

			cl := ComposeListModel{
				available:  m.list.available,
				sequence:   seq,
				validation: ValidateSequence(seq),
				focused:    true,
			}
			cd := NewComposeDetailModel()

			m.composing = true
			m.composeList = &cl
			m.composeDetail = &cd

			leftWidth := m.leftPaneWidth()
			rightWidth := m.width - leftWidth - 3
			m.composeList.SetSize(leftWidth, m.childHeight())
			m.composeDetail.SetSize(rightWidth, m.childHeight())

			seqCopy := cl.sequence
			val := cl.validation
			return m, tea.Batch(
				func() tea.Msg { return ViewChangedMsg{View: ViewPipelines} },
				func() tea.Msg { return ComposeActiveMsg{Active: true} },
				func() tea.Msg {
					return ComposeSequenceChangedMsg{
						Sequence:   seqCopy,
						Validation: val,
					}
				},
			), true
		}
		return m, func() tea.Msg { return ViewChangedMsg{View: ViewPipelines} }, true

	case SuggestModifyMsg:
		// Open configure form pre-populated with the proposal's pipeline and input
		m.currentView = ViewPipelines
		m.focus = FocusPaneRight
		m.list.SetFocused(false)
		m.detail.SetFocused(true)
		return m, func() tea.Msg {
			return ConfigureFormMsg{
				PipelineName: msg.Pipeline.Name,
				InputExample: msg.Pipeline.Input,
			}
		}, true
	}
	return m, nil, false
}

// viewSuggest renders the suggest panel.
func (m ContentModel) viewSuggest() (left, right string) {
	if m.suggestList != nil {
		left = m.suggestList.View()
	} else {
		left = renderPlaceholder(m.leftPaneWidth(), m.height, "No suggest provider configured")
	}
	if m.suggestDetail != nil {
		right = m.suggestDetail.View()
	} else {
		right = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a suggestion to view details")
	}
	return left, right
}
