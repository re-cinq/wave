package tui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NotNil(t, m.navigable[0].issue)
	assert.Equal(t, "Add dark mode support", m.navigable[0].issue.Title)
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

func TestIssueListModel_FilterByAssignee(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix bug", Assignees: []string{"alice"}},
			{Number: 2, Title: "Add feature", Assignees: []string{"bob"}},
			{Number: 3, Title: "Refactor code", Assignees: []string{"alice", "charlie"}},
		},
	})

	// Activate filter and search for "bob"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "bob" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	assert.Equal(t, 1, len(m.navigable))
	require.NotNil(t, m.navigable[0].issue)
	assert.Equal(t, "Add feature", m.navigable[0].issue.Title)
}

func TestIssueListModel_FilterByAssignee_Partial(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix bug", Assignees: []string{"alice"}},
			{Number: 2, Title: "Add feature", Assignees: []string{"bob"}},
			{Number: 3, Title: "Refactor code", Assignees: []string{"alice", "charlie"}},
		},
	})

	// Activate filter and search for "ali" (partial match for alice)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "ali" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	assert.Equal(t, 2, len(m.navigable))
	require.NotNil(t, m.navigable[0].issue)
	assert.Equal(t, "Fix bug", m.navigable[0].issue.Title)
	require.NotNil(t, m.navigable[1].issue)
	assert.Equal(t, "Refactor code", m.navigable[1].issue.Title)
}

// ===========================================================================
// Pipeline children tests
// ===========================================================================

func TestIssueListModel_PipelineDataMsg_AddsChildren(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	// Load issues first
	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
			{Number: 2, Title: "Add dark mode", HTMLURL: "https://github.com/org/repo/issues/2"},
		},
	})
	assert.Equal(t, 2, len(m.navigable)) // 2 issues, no children

	// Send pipeline data with a running pipeline linked to issue #1
	m, _ = m.Update(PipelineDataMsg{
		Running: []RunningPipeline{
			{RunID: "run-001", Name: "plan-speckit", Input: "https://github.com/org/repo/issues/1 fix the auth bug", StartedAt: time.Now()},
		},
	})

	// Issue #1 should now have a running child (always visible even when collapsed)
	assert.Equal(t, 3, len(m.navigable)) // issue1 + running child + issue2
	assert.Equal(t, issueNavKindIssue, m.navigable[0].kind)
	assert.Equal(t, issueNavKindRunning, m.navigable[1].kind)
	assert.Equal(t, issueNavKindIssue, m.navigable[2].kind)
}

func TestIssueListModel_FinishedChildren_HiddenWhenCollapsed(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
		},
	})

	// Send finished pipeline data
	m, _ = m.Update(PipelineDataMsg{
		Finished: []FinishedPipeline{
			{RunID: "run-001", Name: "plan-speckit", Input: "https://github.com/org/repo/issues/1", StartedAt: time.Now(), Duration: 30 * time.Second},
		},
	})

	// Issue is collapsed by default — finished children should be hidden
	assert.Equal(t, 1, len(m.navigable)) // just the issue
	assert.True(t, m.collapsed["https://github.com/org/repo/issues/1"])

	// Expand by pressing space
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})

	// Now finished child should appear
	assert.Equal(t, 2, len(m.navigable))
	assert.Equal(t, issueNavKindIssue, m.navigable[0].kind)
	assert.Equal(t, issueNavKindFinished, m.navigable[1].kind)
}

func TestIssueListModel_RunningChildren_AlwaysVisible(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
		},
	})

	m, _ = m.Update(PipelineDataMsg{
		Running: []RunningPipeline{
			{RunID: "run-001", Name: "plan-speckit", Input: "https://github.com/org/repo/issues/1", StartedAt: time.Now()},
		},
	})

	// Collapsed by default — running children should still be visible
	assert.True(t, m.collapsed["https://github.com/org/repo/issues/1"])
	assert.Equal(t, 2, len(m.navigable))
	assert.Equal(t, issueNavKindRunning, m.navigable[1].kind)
}

func TestIssueListModel_SelectingRunningChild_EmitsPipelineSelectedMsg(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
		},
	})

	m, _ = m.Update(PipelineDataMsg{
		Running: []RunningPipeline{
			{RunID: "run-001", Name: "plan-speckit", Input: "https://github.com/org/repo/issues/1", StartedAt: time.Now()},
		},
	})

	// Navigate to running child
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	require.NotNil(t, cmd)

	msg := cmd()
	selMsg, ok := msg.(PipelineSelectedMsg)
	assert.True(t, ok)
	assert.Equal(t, "run-001", selMsg.RunID)
	assert.Equal(t, "plan-speckit", selMsg.Name)
	assert.Equal(t, itemKindRunning, selMsg.Kind)
}

func TestIssueListModel_FinishedChildrenLimitedToMax(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
		},
	})

	finished := make([]FinishedPipeline, 5)
	for i := range finished {
		finished[i] = FinishedPipeline{
			RunID:    fmt.Sprintf("run-%03d", i),
			Name:     "plan-speckit",
			Input:    "https://github.com/org/repo/issues/1",
			Duration: time.Duration(i+1) * time.Minute,
		}
	}
	m, _ = m.Update(PipelineDataMsg{Finished: finished})

	// Expand the issue
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})

	// Should show issue + max 3 finished children
	assert.Equal(t, 1+issueFinishedPerMax, len(m.navigable))
}

func TestIssueListModel_UnlinkedPipelines_NotShown(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
		},
	})

	// Pipeline with input that doesn't reference any issue
	m, _ = m.Update(PipelineDataMsg{
		Running: []RunningPipeline{
			{RunID: "run-001", Name: "plan-speckit", Input: "some unrelated input", StartedAt: time.Now()},
		},
	})

	// No children should appear
	assert.Equal(t, 1, len(m.navigable))
}

func TestIssueListModel_CollapseToggle(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
		},
	})

	m, _ = m.Update(PipelineDataMsg{
		Finished: []FinishedPipeline{
			{RunID: "run-001", Name: "plan-speckit", Input: "https://github.com/org/repo/issues/1", Duration: 30 * time.Second},
		},
	})

	// Collapsed: only issue visible
	assert.Equal(t, 1, len(m.navigable))

	// Expand
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, 2, len(m.navigable))

	// Collapse again
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, 1, len(m.navigable))
}

func TestIssueListModel_RenderRunningChild(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
		},
	})

	m, _ = m.Update(PipelineDataMsg{
		Running: []RunningPipeline{
			{RunID: "run-001", Name: "plan-speckit", Input: "https://github.com/org/repo/issues/1", StartedAt: time.Now()},
		},
	})

	view := m.View()
	assert.Contains(t, view, "●")
	assert.Contains(t, view, "plan-speckit")
	assert.Contains(t, view, "run-001")
}

func TestIssueListModel_RenderFinishedChild(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
		},
	})

	m, _ = m.Update(PipelineDataMsg{
		Finished: []FinishedPipeline{
			{RunID: "run-001", Name: "plan-speckit", Input: "https://github.com/org/repo/issues/1", Status: "completed", Duration: 30 * time.Second},
		},
	})

	// Expand to see finished children
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})

	view := m.View()
	assert.Contains(t, view, "✓")
	assert.Contains(t, view, "plan-speckit")
	assert.Contains(t, view, "run-001")
}

func TestIssueListModel_FailedPipelineShowsCross(t *testing.T) {
	m := NewIssueListModel(&mockIssueDataProvider{})
	m.SetSize(80, 20)

	m, _ = m.Update(IssueDataMsg{
		Issues: []IssueData{
			{Number: 1, Title: "Fix auth", HTMLURL: "https://github.com/org/repo/issues/1"},
		},
	})

	m, _ = m.Update(PipelineDataMsg{
		Finished: []FinishedPipeline{
			{RunID: "run-001", Name: "plan-speckit", Input: "https://github.com/org/repo/issues/1", Status: "failed", Duration: 10 * time.Second},
		},
	})

	// Expand
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})

	view := m.View()
	assert.Contains(t, view, "✗")
}
