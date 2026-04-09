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

func suggestSendKeyType(m SuggestListModel, kt tea.KeyType) (SuggestListModel, tea.Cmd) { //nolint:unparam // test helper
	return m.Update(tea.KeyMsg{Type: kt})
}

// ===========================================================================
// T016: Suggest list key handlers
// ===========================================================================

func TestSuggestListModel_SkipKey_RemovesProposal(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
		{Name: "pipeline-c", Priority: 3},
	}
	m := newTestSuggestList(proposals)

	require.Equal(t, 3, len(m.proposals))
	require.Equal(t, 0, m.cursor)

	// Press 's' to skip/dismiss the first proposal
	m, _ = suggestSendKey(m, "s")

	assert.Equal(t, 2, len(m.proposals), "skip should remove the proposal")
	assert.Equal(t, "pipeline-b", m.proposals[0].Name, "pipeline-b should now be first")
	assert.Equal(t, "pipeline-c", m.proposals[1].Name)
}

func TestSuggestListModel_SkipKey_LastProposal_EmptiesList(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
	}
	m := newTestSuggestList(proposals)

	m, _ = suggestSendKey(m, "s")

	assert.Equal(t, 0, len(m.proposals), "skipping last proposal should empty the list")
}

func TestSuggestListModel_SkipKey_CursorAdjustedAfterDismiss(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
	}
	m := newTestSuggestList(proposals)

	// Move to the last entry, then skip it
	m, _ = suggestSendKey(m, "j") // move down to index 1
	require.Equal(t, 1, m.cursor)

	m, _ = suggestSendKey(m, "s") // skip pipeline-b

	assert.Equal(t, 1, len(m.proposals), "one proposal should remain")
	assert.Equal(t, 0, m.cursor, "cursor should be adjusted to last valid index")
}

func TestSuggestListModel_SkipKey_MiddleItem_CursorUnchanged(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
		{Name: "pipeline-c", Priority: 3},
	}
	m := newTestSuggestList(proposals)

	// Move to the middle, skip it
	m, _ = suggestSendKey(m, "j") // cursor=1
	require.Equal(t, 1, m.cursor)

	m, _ = suggestSendKey(m, "s") // skip pipeline-b

	// Cursor stays at 1 (now points to pipeline-c)
	assert.Equal(t, 2, len(m.proposals))
	assert.Equal(t, 1, m.cursor)
	assert.Equal(t, "pipeline-c", m.proposals[m.cursor].Name)
}

func TestSuggestListModel_ModifyKey_EmitsSuggestModifyMsg(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1, Input: "some input"},
		{Name: "pipeline-b", Priority: 2, Input: "other input"},
	}
	m := newTestSuggestList(proposals)

	// Press 'm' on the first proposal
	_, cmd := suggestSendKey(m, "m")
	require.NotNil(t, cmd, "m key should return a command")

	msg := cmd()
	modifyMsg, ok := msg.(SuggestModifyMsg)
	require.True(t, ok, "expected SuggestModifyMsg, got %T", msg)
	assert.Equal(t, "pipeline-a", modifyMsg.Pipeline.Name)
	assert.Equal(t, "some input", modifyMsg.Pipeline.Input)
}

func TestSuggestListModel_ModifyKey_SecondItem_EmitsCorrectPipeline(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2, Input: "pipeline-b input"},
	}
	m := newTestSuggestList(proposals)

	m, _ = suggestSendKey(m, "j") // move to pipeline-b
	_, cmd := suggestSendKey(m, "m")

	require.NotNil(t, cmd)
	msg := cmd()
	modifyMsg, ok := msg.(SuggestModifyMsg)
	require.True(t, ok)
	assert.Equal(t, "pipeline-b", modifyMsg.Pipeline.Name)
	assert.Equal(t, "pipeline-b input", modifyMsg.Pipeline.Input)
}

func TestSuggestListModel_MultiSelectEnter_EmitsSuggestComposeMsg(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
		{Name: "pipeline-c", Priority: 3},
	}
	m := newTestSuggestList(proposals)

	// Select pipeline-a and pipeline-c
	m, _ = suggestSendKey(m, " ") // select pipeline-a at index 0
	m, _ = suggestSendKey(m, "j") // move to index 1
	m, _ = suggestSendKey(m, "j") // move to index 2
	m, _ = suggestSendKey(m, " ") // select pipeline-c at index 2
	require.Equal(t, 2, len(m.selected))

	_, cmd := suggestSendKeyType(m, tea.KeyEnter)
	compose := extractMsg[SuggestComposeMsg](cmd)
	require.NotNil(t, compose, "Enter with multi-select should emit SuggestComposeMsg")
	assert.Equal(t, 2, len(compose.Pipelines))
}

func TestSuggestListModel_SingleEnter_EmitsSuggestLaunchMsg(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
	}
	m := newTestSuggestList(proposals)

	_, cmd := suggestSendKeyType(m, tea.KeyEnter)
	launch := extractMsg[SuggestLaunchMsg](cmd)
	require.NotNil(t, launch, "Enter without multi-select should emit SuggestLaunchMsg")
	assert.Equal(t, "pipeline-a", launch.Pipeline.Name)
}

func TestSuggestListModel_SkipSelectedItem_AdjustsSelectedMap(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
		{Name: "pipeline-c", Priority: 3},
	}
	m := newTestSuggestList(proposals)

	// Select pipeline-b (index 1) and pipeline-c (index 2), then skip pipeline-a (index 0)
	m.selected = map[int]bool{1: true, 2: true}

	// Skip pipeline-a at index 0 (cursor is at 0)
	m, _ = suggestSendKey(m, "s")

	// After skip: pipeline-b is now at 0, pipeline-c at 1
	// Selected indices should have been shifted down by 1
	assert.Equal(t, 2, len(m.proposals))
	assert.True(t, m.selected[0], "pipeline-b's new index 0 should be selected")
	assert.True(t, m.selected[1], "pipeline-c's new index 1 should be selected")
	assert.False(t, m.selected[2], "old index 2 should not be selected")
}

func TestSuggestListModel_EmptyProposals_SkipDoesNothing(t *testing.T) {
	m := newTestSuggestList(nil)

	m, _ = suggestSendKey(m, "s")
	assert.Equal(t, 0, len(m.proposals))
}

func TestSuggestListModel_LaunchedBadge_ShowsCheckmark(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
	}
	m := newTestSuggestList(proposals)

	// Mark pipeline-a as launched
	m.launched = map[string]bool{"pipeline-a": true}

	view := m.View()
	assert.Contains(t, view, "✓", "launched proposal should show checkmark badge")
}

func TestSuggestListModel_LaunchedBadge_NotShownForUnlaunched(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
	}
	m := newTestSuggestList(proposals)

	view := m.View()
	assert.NotContains(t, view, "✓", "unlaunched proposal should not show checkmark badge")
}

func TestSuggestListModel_SuggestLaunchedMsg_UpdatesTracking(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
	}
	m := newTestSuggestList(proposals)

	assert.Nil(t, m.launched)

	m, _ = m.Update(SuggestLaunchedMsg{Name: "pipeline-a"})

	require.NotNil(t, m.launched)
	assert.True(t, m.launched["pipeline-a"])
	assert.False(t, m.launched["pipeline-b"])
}

func TestSuggestListModel_SuggestLaunchedMsg_MultipleLaunches(t *testing.T) {
	proposals := []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1},
		{Name: "pipeline-b", Priority: 2},
	}
	m := newTestSuggestList(proposals)

	m, _ = m.Update(SuggestLaunchedMsg{Name: "pipeline-a"})
	m, _ = m.Update(SuggestLaunchedMsg{Name: "pipeline-b"})

	assert.True(t, m.launched["pipeline-a"])
	assert.True(t, m.launched["pipeline-b"])
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
		m, _ = suggestSendKey(m, " ") // select 0
		m, _ = suggestSendKey(m, "j") // move down
		m, _ = suggestSendKey(m, " ") // select 1
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
