package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/pipelinecatalog"
	"github.com/stretchr/testify/assert"
)

// ===========================================================================
// IssueDetailModel tests
// ===========================================================================

func TestIssueDetailModel_EmptyState(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)

	view := m.View()
	assert.Contains(t, view, "Select an issue to view details")
}

func TestIssueDetailModel_SetIssue(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)

	issue := &IssueData{
		Number:  123,
		Title:   "Fix authentication bug",
		State:   "open",
		Author:  "testuser",
		Labels:  []string{"bug", "P1"},
		Body:    "The auth flow is broken",
		HTMLURL: "https://github.com/org/repo/issues/123",
	}
	m.SetIssue(issue)

	view := m.View()
	assert.Contains(t, view, "#123")
	assert.Contains(t, view, "Fix authentication bug")
	assert.Contains(t, view, "open")
	assert.Contains(t, view, "testuser")
	assert.Contains(t, view, "bug, P1")
	assert.Contains(t, view, "The auth flow is broken")
}

func TestIssueDetailModel_SetSize(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 40, m.height)
}

func TestIssueDetailModel_SelectionUpdatesContent(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)

	// Set first issue
	issue1 := &IssueData{
		Number: 1,
		Title:  "First issue",
		State:  "open",
		Body:   "First body",
	}
	m.SetIssue(issue1)

	view := m.View()
	assert.Contains(t, view, "First issue")

	// Set second issue
	issue2 := &IssueData{
		Number: 2,
		Title:  "Second issue",
		State:  "closed",
		Body:   "Second body",
	}
	m.SetIssue(issue2)

	view = m.View()
	assert.Contains(t, view, "Second issue")
	assert.Contains(t, view, "closed")
}

func TestIssueDetailModel_PipelineChooser_Open(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)
	m.SetFocused(true)

	m.SetPipelines([]pipelinecatalog.PipelineInfo{
		{Name: "speckit-flow"},
		{Name: "wave-bugfix"},
		{Name: "wave-review"},
	})

	issue := &IssueData{
		Number:  1,
		Title:   "Test issue",
		State:   "open",
		HTMLURL: "https://github.com/org/repo/issues/1",
	}
	m.SetIssue(issue)

	// Press Enter to open chooser
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, m.chooserActive)
	assert.Equal(t, 0, m.chooserCursor)
}

func TestIssueDetailModel_PipelineChooser_Navigate(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)
	m.SetFocused(true)

	m.SetPipelines([]pipelinecatalog.PipelineInfo{
		{Name: "speckit-flow"},
		{Name: "wave-bugfix"},
		{Name: "wave-review"},
	})

	issue := &IssueData{
		Number:  1,
		Title:   "Test issue",
		State:   "open",
		HTMLURL: "https://github.com/org/repo/issues/1",
	}
	m.SetIssue(issue)

	// Open chooser
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Navigate down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.chooserCursor)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.chooserCursor)

	// Can't go past end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.chooserCursor)

	// Navigate up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, m.chooserCursor)
}

func TestIssueDetailModel_PipelineChooser_Launch(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)
	m.SetFocused(true)

	m.SetPipelines([]pipelinecatalog.PipelineInfo{
		{Name: "speckit-flow"},
		{Name: "wave-bugfix"},
	})

	issue := &IssueData{
		Number:  42,
		Title:   "Test issue",
		State:   "open",
		HTMLURL: "https://github.com/org/repo/issues/42",
	}
	m.SetIssue(issue)

	// Open chooser
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Navigate to wave-bugfix
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.chooserCursor)

	// Press Enter to launch
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.chooserActive)
	assert.NotNil(t, cmd)

	// Verify the launched message
	msg := cmd()
	launchMsg, ok := msg.(IssueLaunchMsg)
	assert.True(t, ok)
	assert.Equal(t, "wave-bugfix", launchMsg.PipelineName)
	assert.Equal(t, "https://github.com/org/repo/issues/42", launchMsg.IssueURL)
}

func TestIssueDetailModel_PipelineChooser_Cancel(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)
	m.SetFocused(true)

	m.SetPipelines([]pipelinecatalog.PipelineInfo{
		{Name: "speckit-flow"},
	})

	issue := &IssueData{
		Number:  1,
		Title:   "Test issue",
		State:   "open",
		HTMLURL: "https://github.com/org/repo/issues/1",
	}
	m.SetIssue(issue)

	// Open chooser
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, m.chooserActive)

	// Press Escape to cancel
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, m.chooserActive)
}

func TestIssueDetailModel_NoPipelines_NoChooser(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)
	m.SetFocused(true)

	issue := &IssueData{
		Number:  1,
		Title:   "Test issue",
		State:   "open",
		HTMLURL: "https://github.com/org/repo/issues/1",
	}
	m.SetIssue(issue)

	// Press Enter with no pipelines configured
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.chooserActive) // Should not activate
}

func TestIssueDetailModel_UnfocusedIgnoresKeys(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)
	m.SetFocused(false)

	m.SetPipelines([]pipelinecatalog.PipelineInfo{{Name: "speckit-flow"}})

	issue := &IssueData{
		Number:  1,
		Title:   "Test issue",
		State:   "open",
		HTMLURL: "https://github.com/org/repo/issues/1",
	}
	m.SetIssue(issue)

	// Press Enter while unfocused
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.chooserActive)
}

func TestIssueDetailModel_ViewShowsPipelineChooser(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)

	m.SetPipelines([]pipelinecatalog.PipelineInfo{
		{Name: "speckit-flow"},
		{Name: "wave-bugfix"},
	})

	issue := &IssueData{
		Number: 1,
		Title:  "Test issue",
		State:  "open",
	}
	m.SetIssue(issue)

	view := m.View()
	assert.Contains(t, view, "Launch Pipeline")
	assert.Contains(t, view, "speckit-flow")
	assert.Contains(t, view, "wave-bugfix")
}

func TestPipelineRelevanceScore_NameMatchesTitle(t *testing.T) {
	p := pipelinecatalog.PipelineInfo{Name: "pr-review", Description: "Automated code review workflow"}
	issue := &IssueData{Title: "Review the PR for auth module", Labels: nil}

	score := pipelineRelevanceScore(p, issue)
	assert.Greater(t, score, 0, "pipeline name token 'review' should match title")
}

func TestPipelineRelevanceScore_LabelMatchesName(t *testing.T) {
	p := pipelinecatalog.PipelineInfo{Name: "wave-bugfix", Description: "Fix bugs", Category: "maintenance"}
	issue := &IssueData{Title: "Something unrelated", Labels: []string{"bugfix"}}

	score := pipelineRelevanceScore(p, issue)
	assert.Greater(t, score, 0, "label 'bugfix' should match pipeline name")
}

func TestPipelineRelevanceScore_NoMatch(t *testing.T) {
	p := pipelinecatalog.PipelineInfo{Name: "deploy-prod", Description: "Deploy to production"}
	issue := &IssueData{Title: "Fix authentication bug", Labels: []string{"bug"}}

	score := pipelineRelevanceScore(p, issue)
	assert.Equal(t, 0, score, "no overlap should yield zero score")
}

func TestPipelineRelevanceScore_CategoryMatchesLabel(t *testing.T) {
	p := pipelinecatalog.PipelineInfo{Name: "some-pipeline", Description: "A pipeline", Category: "security"}
	issue := &IssueData{Title: "Unrelated title", Labels: []string{"security"}}

	score := pipelineRelevanceScore(p, issue)
	assert.Greater(t, score, 0, "category 'security' should match label")
}

func TestIssueDetailModel_RelevanceSorting(t *testing.T) {
	m := NewIssueDetailModel()
	m.SetSize(80, 40)

	m.SetPipelines([]pipelinecatalog.PipelineInfo{
		{Name: "deploy-prod", Description: "Deploy to production"},
		{Name: "wave-bugfix", Description: "Fix bugs quickly"},
		{Name: "speckit-flow", Description: "Feature development"},
	})

	issue := &IssueData{
		Number: 1,
		Title:  "Fix a critical bug in auth",
		Labels: []string{"bugfix"},
	}
	m.SetIssue(issue)

	// wave-bugfix should be first because "bugfix" label matches its name
	assert.Equal(t, "wave-bugfix", m.pipelines[0].Name)
}

// ===========================================================================
// Mock provider for tests
// ===========================================================================

type mockIssueDataProvider struct {
	issues []IssueData
	err    error
}

func (m *mockIssueDataProvider) FetchIssues() ([]IssueData, error) {
	return m.issues, m.err
}
