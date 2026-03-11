package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSuggestList(proposals []SuggestProposedPipeline) SuggestListModel {
	m := NewSuggestListModel(nil)
	m.proposals = proposals
	m.loaded = true
	m.focused = true
	m.SetSize(40, 20)
	return m
}

func suggestSendKey(m SuggestListModel, key string) (SuggestListModel, tea.Cmd) {
	if key == " " {
		return m.Update(tea.KeyMsg{Type: tea.KeySpace})
	}
	return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
}

func suggestSendKeyType(m SuggestListModel, kt tea.KeyType) (SuggestListModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: kt})
}

func TestSuggestListModel_MultiSelect(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1, Reason: "Fix CI"},
		{Name: "pipeline-b", Priority: 2, Reason: "Open issues"},
		{Name: "pipeline-c", Priority: 3, Reason: "Review PRs"},
	}

	t.Run("Space toggles selection", func(t *testing.T) {
		m := newTestSuggestList(proposals)
		assert.Empty(t, m.selected)

		m, _ = suggestSendKey(m, " ")
		assert.True(t, m.selected[0], "Space should select item at cursor")

		m, _ = suggestSendKey(m, " ")
		assert.False(t, m.selected[0], "Space again should deselect")
	})

	t.Run("multi-select Enter emits SuggestComposeMsg", func(t *testing.T) {
		m := newTestSuggestList(proposals)

		// Select first two
		m, _ = suggestSendKey(m, " ")       // select 0
		m, _ = suggestSendKey(m, "j")       // move down
		m, _ = suggestSendKey(m, " ")       // select 1
		require.Equal(t, 2, len(m.selected))

		_, cmd := suggestSendKeyType(m, tea.KeyEnter)
		compose := extractMsg[SuggestComposeMsg](cmd)
		require.NotNil(t, compose, "Enter with multi-select should emit SuggestComposeMsg")
		assert.Equal(t, 2, len(compose.Pipelines))
		assert.Equal(t, "pipeline-a", compose.Pipelines[0].Name)
		assert.Equal(t, "pipeline-b", compose.Pipelines[1].Name)
	})

	t.Run("single-select Enter emits SuggestLaunchMsg", func(t *testing.T) {
		m := newTestSuggestList(proposals)

		_, cmd := suggestSendKeyType(m, tea.KeyEnter)
		launch := extractMsg[SuggestLaunchMsg](cmd)
		require.NotNil(t, launch, "Enter without multi-select should emit SuggestLaunchMsg")
		assert.Equal(t, "pipeline-a", launch.Pipeline.Name)
	})

	t.Run("selection state in emitSelection", func(t *testing.T) {
		m := newTestSuggestList(proposals)
		m.selected = map[int]bool{0: true, 2: true}

		cmd := m.emitSelection()
		require.NotNil(t, cmd)
		msg := cmd()
		sel, ok := msg.(SuggestSelectedMsg)
		require.True(t, ok)
		assert.Equal(t, 2, len(sel.MultiSelected))
	})
}
