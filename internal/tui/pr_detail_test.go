package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ===========================================================================
// PRDetailModel tests
// ===========================================================================

func TestPRDetailModel_EmptyState(t *testing.T) {
	m := NewPRDetailModel()
	m.SetSize(80, 40)

	view := m.View()
	assert.Contains(t, view, "Select a pull request to view details")
}

func TestPRDetailModel_SetPR(t *testing.T) {
	m := NewPRDetailModel()
	m.SetSize(80, 40)

	pr := &PRData{
		Number:       42,
		Title:        "Add PR list view",
		State:        "open",
		Author:       "testuser",
		Labels:       []string{"enhancement", "tui"},
		HeadBranch:   "feat/pr-list",
		BaseBranch:   "main",
		Additions:    150,
		Deletions:    30,
		ChangedFiles: 8,
		Body:         "This PR adds a pull request list view",
		CreatedAt:    time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		Comments:     5,
		Commits:      3,
	}
	m.SetPR(pr)

	view := m.View()
	assert.Contains(t, view, "#42")
	assert.Contains(t, view, "Add PR list view")
	assert.Contains(t, view, "Open")
	assert.Contains(t, view, "testuser")
	assert.Contains(t, view, "enhancement, tui")
	assert.Contains(t, view, "feat/pr-list")
	assert.Contains(t, view, "main")
	assert.Contains(t, view, "+150/-30")
	assert.Contains(t, view, "8 files")
	assert.Contains(t, view, "2025-03-15")
	assert.Contains(t, view, "This PR adds a pull request list view")
}

func TestPRDetailModel_SetSize(t *testing.T) {
	m := NewPRDetailModel()
	m.SetSize(80, 40)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 40, m.height)
}

func TestPRDetailModel_SelectionUpdatesContent(t *testing.T) {
	m := NewPRDetailModel()
	m.SetSize(80, 40)

	pr1 := &PRData{
		Number: 1,
		Title:  "First PR",
		State:  "open",
		Body:   "First body",
	}
	m.SetPR(pr1)

	view := m.View()
	assert.Contains(t, view, "First PR")

	pr2 := &PRData{
		Number: 2,
		Title:  "Second PR",
		State:  "closed",
		Merged: true,
		Body:   "Second body",
	}
	m.SetPR(pr2)

	view = m.View()
	assert.Contains(t, view, "Second PR")
	assert.Contains(t, view, "Merged")
}

func TestPRDetailModel_DraftStatus(t *testing.T) {
	m := NewPRDetailModel()
	m.SetSize(80, 40)

	pr := &PRData{
		Number: 1,
		Title:  "WIP PR",
		State:  "open",
		Draft:  true,
	}
	m.SetPR(pr)

	view := m.View()
	assert.Contains(t, view, "Draft")
}

func TestPRDetailModel_ClosedStatus(t *testing.T) {
	m := NewPRDetailModel()
	m.SetSize(80, 40)

	pr := &PRData{
		Number: 1,
		Title:  "Old PR",
		State:  "closed",
	}
	m.SetPR(pr)

	view := m.View()
	assert.Contains(t, view, "Closed")
}
