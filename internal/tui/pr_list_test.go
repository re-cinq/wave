package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// PRListModel tests
// ===========================================================================

func TestPRListModel_Init_FetchesData(t *testing.T) {
	provider := &mockPRDataProvider{
		prs: []PRData{
			{Number: 1, Title: "First PR"},
		},
	}
	m := NewPRListModel(provider)
	cmd := m.Init()
	assert.NotNil(t, cmd)

	msg := cmd()
	dataMsg, ok := msg.(PRDataMsg)
	assert.True(t, ok)
	assert.Nil(t, dataMsg.Err)
	assert.Equal(t, 1, len(dataMsg.PRs))
}

func TestPRListModel_DataLoading(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)

	msg := PRDataMsg{
		PRs: []PRData{
			{Number: 100, Title: "Add feature X", Labels: []string{"enhancement"}},
			{Number: 99, Title: "Fix bug Y", Labels: []string{"bug"}},
		},
	}

	m, _ = m.Update(msg)
	assert.Equal(t, 2, len(m.prs))
	assert.Equal(t, 2, len(m.navigable))
	assert.True(t, m.loaded)
}

func TestPRListModel_Navigation(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
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

func TestPRListModel_Filtering_Title(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
			{Number: 1, Title: "Add authentication", Labels: []string{"feat"}},
			{Number: 2, Title: "Fix dark mode", Labels: []string{"bug"}},
			{Number: 3, Title: "Add pagination", Labels: []string{"feat"}},
		},
	})

	assert.Equal(t, 3, len(m.navigable))

	// Activate filter with '/'
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.filtering)

	// Type 'dark' to filter
	for _, r := range "dark" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	assert.Equal(t, 1, len(m.navigable))
	prIdx := m.navigable[0]
	assert.Equal(t, "Fix dark mode", m.prs[prIdx].Title)

	// Clear filter with Escape
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, m.filtering)
	assert.Equal(t, 3, len(m.navigable))
}

func TestPRListModel_Filtering_Number(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
			{Number: 42, Title: "PR forty-two"},
			{Number: 99, Title: "PR ninety-nine"},
		},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "#42" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	assert.Equal(t, 1, len(m.navigable))
	prIdx := m.navigable[0]
	assert.Equal(t, 42, m.prs[prIdx].Number)
}

func TestPRListModel_Filtering_Author(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
			{Number: 1, Title: "PR A", Author: "alice"},
			{Number: 2, Title: "PR B", Author: "bob"},
		},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "bob" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	assert.Equal(t, 1, len(m.navigable))
	prIdx := m.navigable[0]
	assert.Equal(t, "bob", m.prs[prIdx].Author)
}

func TestPRListModel_Filtering_Label(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
			{Number: 1, Title: "PR A", Labels: []string{"bug", "P1"}},
			{Number: 2, Title: "PR B", Labels: []string{"enhancement"}},
			{Number: 3, Title: "PR C", Labels: []string{"bug"}},
		},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "enhancement" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	assert.Equal(t, 1, len(m.navigable))
	prIdx := m.navigable[0]
	assert.Equal(t, 2, m.prs[prIdx].Number)
}

func TestPRListModel_EmptyState(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)
	m, _ = m.Update(PRDataMsg{})

	view := m.View()
	assert.Contains(t, view, "No pull requests found")
}

func TestPRListModel_EmptyFilter(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
			{Number: 1, Title: "Test PR"},
		},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "zzzzz" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	view := m.View()
	assert.Contains(t, view, "No matching pull requests")
}

func TestPRListModel_ViewRendering(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
			{Number: 100, Title: "Add feature X", Draft: true, Labels: []string{"feat"}},
			{Number: 99, Title: "Fix bug Y", State: "open"},
		},
	})

	view := m.View()
	assert.Contains(t, view, "#100")
	assert.Contains(t, view, "Add feature X")
	assert.Contains(t, view, "[Draft]")
	assert.Contains(t, view, "#99")
	assert.Contains(t, view, "[Open]")
}

func TestPRListModel_UnfocusedIgnoresKeys(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)
	m.SetFocused(false)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
			{Number: 1, Title: "First"},
			{Number: 2, Title: "Second"},
		},
	})

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Nil(t, cmd)
	assert.Equal(t, 0, m.cursor)
}

func TestPRListModel_NilProvider(t *testing.T) {
	m := NewPRListModel(nil)
	cmd := m.Init()
	assert.NotNil(t, cmd)

	msg := cmd()
	dataMsg, ok := msg.(PRDataMsg)
	assert.True(t, ok)
	assert.Nil(t, dataMsg.Err)
	assert.Nil(t, dataMsg.PRs)
}

func TestPRListModel_NavigationEmitsSelectionMsg(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
			{Number: 1, Title: "First"},
			{Number: 2, Title: "Second"},
		},
	})

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	require.NotNil(t, cmd)

	msg := cmd()
	selMsg, ok := msg.(PRSelectedMsg)
	assert.True(t, ok)
	assert.Equal(t, 2, selMsg.Number)
	assert.Equal(t, "Second", selMsg.Title)
}

func TestPRListModel_StatusBadges(t *testing.T) {
	m := NewPRListModel(&mockPRDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(PRDataMsg{
		PRs: []PRData{
			{Number: 1, Title: "Draft PR", Draft: true},
			{Number: 2, Title: "Merged PR", Merged: true, State: "closed"},
			{Number: 3, Title: "Closed PR", State: "closed"},
			{Number: 4, Title: "Open PR", State: "open"},
		},
	})

	view := m.View()
	assert.Contains(t, view, "[Draft]")
	assert.Contains(t, view, "[Merged]")
	assert.Contains(t, view, "[Closed]")
	assert.Contains(t, view, "[Open]")
}

// ===========================================================================
// Mock provider
// ===========================================================================

type mockPRDataProvider struct {
	prs []PRData
	err error
}

func (m *mockPRDataProvider) FetchPRs() ([]PRData, error) {
	return m.prs, m.err
}
