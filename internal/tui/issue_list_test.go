package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// ===========================================================================
// IssueListModel tests
// ===========================================================================

func TestIssueListModel_Init_FetchesData(t *testing.T) {
	provider := &mockIssueDataProvider{
		issues: []IssueData{
			{Number: 1, Title: "First issue"},
		},
	}
	m := NewIssueListModel(provider)
	cmd := m.Init()
	assert.NotNil(t, cmd)

	// Execute the command and verify it returns IssueDataMsg
	msg := cmd()
	dataMsg, ok := msg.(IssueDataMsg)
	assert.True(t, ok)
	assert.Nil(t, dataMsg.Err)
	assert.Equal(t, 1, len(dataMsg.Issues))
}

func TestIssueListModel_DataLoading(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)

	msg := IssueDataMsg{
		Issues: []IssueData{
			{Number: 123, Title: "Fix authentication bug", Labels: []string{"bug", "P1"}},
			{Number: 120, Title: "Add dark mode support", Labels: []string{"enhancement"}},
		},
	}

	m, _ = m.Update(msg)
	assert.Equal(t, 2, len(m.issues))
	assert.Equal(t, 2, len(m.navigable))
	assert.True(t, m.loaded)
}

func TestIssueListModel_Navigation(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "First"},
			{Number: 2, Title: "Second"},
			{Number: 3, Title: "Third"},
		},
	})

	assert.Equal(t, 0, m.cursor)

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.cursor)

	// Can't go past end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.cursor)

	// Move up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, m.cursor)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.cursor)

	// Can't go before start
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.cursor)
}

func TestIssueListModel_Filtering(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix authentication bug", Labels: []string{"bug"}},
			{Number: 2, Title: "Add dark mode support", Labels: []string{"enhancement"}},
			{Number: 3, Title: "Performance regression", Labels: []string{"bug"}},
		},
	})

	assert.Equal(t, 3, len(m.navigable))

	// Activate filter with '/'
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.filtering)

	// Type 'bug' to filter
	for _, r := range "bug" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	// Filter matches: #1 by title ("Fix authentication bug"), #3 by label ("bug")
	// #2 doesn't match title or labels
	assert.Equal(t, 2, len(m.navigable))

	// Clear filter with Escape
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, m.filtering)
	assert.Equal(t, 3, len(m.navigable))
}

func TestIssueListModel_FilterByTitle(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix authentication bug"},
			{Number: 2, Title: "Add dark mode support"},
			{Number: 3, Title: "Performance regression"},
		},
	})

	// Activate filter
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type 'dark' to filter
	for _, r := range "dark" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	assert.Equal(t, 1, len(m.navigable))
	assert.Equal(t, "Add dark mode support", m.navigable[0].Title)
}

func TestIssueListModel_EmptyList(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)
	m, _ = m.Update(IssueDataMsg{})

	view := m.View()
	assert.Contains(t, view, "No issues found")
}

func TestIssueListModel_EmptyFilterResult(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Test issue"},
		},
	})

	// Activate filter and type something that doesn't match
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "zzzzz" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	view := m.View()
	assert.Contains(t, view, "No matching issues")
}

func TestIssueListModel_ViewRendersIssues(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 123, Title: "Fix authentication bug", Labels: []string{"bug"}, Comments: 3},
		},
	})

	view := m.View()
	assert.Contains(t, view, "#123")
	assert.Contains(t, view, "Fix authentication bug")
	assert.Contains(t, view, "[bug]")
}

func TestIssueListModel_NavigationEmitsSelectionMsg(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "First"},
			{Number: 2, Title: "Second"},
		},
	})

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.NotNil(t, cmd)

	msg := cmd()
	selMsg, ok := msg.(IssueSelectedMsg)
	assert.True(t, ok)
	assert.Equal(t, 2, selMsg.Number)
	assert.Equal(t, "Second", selMsg.Title)
}

func TestIssueListModel_UnfocusedIgnoresKeys(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)
	m.SetFocused(false)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "First"},
			{Number: 2, Title: "Second"},
		},
	})

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Nil(t, cmd)
	assert.Equal(t, 0, m.cursor) // Cursor shouldn't move
}

func TestIssueListModel_NilProvider(t *testing.T) {
	m := NewIssueListModel(nil)
	cmd := m.Init()
	assert.NotNil(t, cmd)

	msg := cmd()
	dataMsg, ok := msg.(IssueDataMsg)
	assert.True(t, ok)
	assert.Nil(t, dataMsg.Err)
	assert.Nil(t, dataMsg.Issues)
}
