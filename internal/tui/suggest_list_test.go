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

// ===========================================================================
// T024: Skip/modify keybinding tests
// ===========================================================================

func TestSuggestListModel_SkipToggle(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
	}

	t.Run("s toggles skip", func(t *testing.T) {
		m := newTestSuggestList(proposals)
		assert.Empty(t, m.skipped)

		m, _ = suggestSendKey(m, "s")
		assert.True(t, m.skipped[0])

		m, _ = suggestSendKey(m, "s")
		assert.False(t, m.skipped[0])
	})

	t.Run("skipped items are excluded from enter", func(t *testing.T) {
		m := newTestSuggestList(proposals)
		m, _ = suggestSendKey(m, "s") // skip first

		// Single select enter should not launch skipped item
		_, cmd := suggestSendKeyType(m, tea.KeyEnter)
		launch := extractMsg[SuggestLaunchMsg](cmd)
		assert.Nil(t, launch, "Enter on skipped item should not emit SuggestLaunchMsg")
	})
}

func TestSuggestListModel_ModifyOverlay(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1, Input: "original input"},
	}

	t.Run("m opens input overlay", func(t *testing.T) {
		m := newTestSuggestList(proposals)
		assert.Nil(t, m.inputOverlay)

		m, _ = suggestSendKey(m, "m")
		assert.NotNil(t, m.inputOverlay)
		assert.Equal(t, 0, m.overlayTarget)
		assert.True(t, m.IsInputActive())
	})

	t.Run("Escape cancels overlay", func(t *testing.T) {
		m := newTestSuggestList(proposals)
		m, _ = suggestSendKey(m, "m")
		require.NotNil(t, m.inputOverlay)

		m, _ = suggestSendKeyType(m, tea.KeyEscape)
		assert.Nil(t, m.inputOverlay)
		assert.False(t, m.IsInputActive())
	})

	t.Run("Enter confirms modification", func(t *testing.T) {
		m := newTestSuggestList(proposals)
		m, _ = suggestSendKey(m, "m")
		require.NotNil(t, m.inputOverlay)

		m.inputOverlay.SetValue("modified input")
		m, _ = suggestSendKeyType(m, tea.KeyEnter)
		assert.Nil(t, m.inputOverlay)
		assert.Equal(t, "modified input", m.proposals[0].Input)
	})
}

func TestSuggestListModel_BatchLaunchExcludesSkipped(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
		{Name: "pipeline-c", Priority: 3},
	}
	m := newTestSuggestList(proposals)

	// Select all three
	m, _ = suggestSendKey(m, " ")         // select 0
	m, _ = suggestSendKey(m, "j")         // move down
	m, _ = suggestSendKey(m, " ")         // select 1
	m, _ = suggestSendKey(m, "j")         // move down
	m, _ = suggestSendKey(m, " ")         // select 2
	require.Equal(t, 3, len(m.selected))

	// Skip item 1
	m.cursor = 1
	m, _ = suggestSendKey(m, "s")

	// Enter should emit compose with only 2 pipelines (a and c)
	_, cmd := suggestSendKeyType(m, tea.KeyEnter)
	compose := extractMsg[SuggestComposeMsg](cmd)
	require.NotNil(t, compose)
	assert.Equal(t, 2, len(compose.Pipelines))
	assert.Equal(t, "pipeline-a", compose.Pipelines[0].Name)
	assert.Equal(t, "pipeline-c", compose.Pipelines[1].Name)
}
