package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ensureOntologyModels lazily creates the ontology list and detail models.
func (m *ContentModel) ensureOntologyModels(leftWidth, rightWidth, h int) tea.Cmd {
	var initCmd tea.Cmd
	if m.ontologyList == nil && m.ontologyProvider != nil {
		ol := NewOntologyListModel(m.ontologyProvider)
		ol.SetSize(leftWidth, h)
		m.ontologyList = &ol
		od := NewOntologyDetailModel()
		od.SetSize(rightWidth, h)
		m.ontologyDetail = &od
		initCmd = m.ontologyList.Init()
	}
	if m.ontologyList != nil {
		m.ontologyList.SetFocused(true)
	}
	if m.ontologyDetail != nil {
		m.ontologyDetail.SetFocused(false)
	}
	return initCmd
}

// setSizeOntology propagates size changes to the ontology models.
func (m *ContentModel) setSizeOntology(leftWidth, rightWidth, h int) {
	if m.ontologyList != nil {
		m.ontologyList.SetSize(leftWidth, h)
	}
	if m.ontologyDetail != nil {
		m.ontologyDetail.SetSize(rightWidth, h)
	}
}

// updateOntologyMessage routes ontology messages to the ontology list and detail.
// Returns (model, cmd, handled). handled=true means the message belonged to this panel.
func (m ContentModel) updateOntologyMessage(msg tea.Msg) (ContentModel, tea.Cmd, bool) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case OntologyDataMsg:
		if m.ontologyList != nil {
			var cmd tea.Cmd
			*m.ontologyList, cmd = m.ontologyList.Update(msg)
			return m, cmd, true
		}
		return m, nil, true

	case OntologySelectedMsg:
		if m.ontologyList != nil {
			var listCmd tea.Cmd
			*m.ontologyList, listCmd = m.ontologyList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
		}
		if m.ontologyDetail != nil && m.ontologyList != nil {
			for i := range m.ontologyList.navigable {
				if m.ontologyList.navigable[i].Name == msg.Name {
					m.ontologyDetail.SetContext(&m.ontologyList.navigable[i])
					break
				}
			}
		}
		return m, tea.Batch(cmds...), true
	}
	return m, nil, false
}

// viewOntology renders the ontology panel.
func (m ContentModel) viewOntology() (left, right string) {
	if m.ontologyList != nil {
		left = m.ontologyList.View()
	} else {
		left = renderPlaceholder(m.leftPaneWidth(), m.height, "No ontology configured")
	}
	if m.ontologyDetail != nil {
		right = m.ontologyDetail.View()
	} else {
		right = renderPlaceholder(m.width-m.leftPaneWidth()-3, m.height, "Select a context to view details")
	}
	return left, right
}
