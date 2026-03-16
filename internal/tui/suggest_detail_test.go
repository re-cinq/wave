package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuggestDetailModel_View_SingleProposal(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 30)

	p := SuggestProposedPipeline{Name: "test-pipe", Priority: 1, Reason: "Fix CI", Input: "gh#123"}
	m, _ = m.Update(SuggestSelectedMsg{Pipeline: p})

	view := m.View()
	assert.Contains(t, view, "test-pipe")
	assert.Contains(t, view, "Priority")
	assert.Contains(t, view, "Fix CI")
	assert.Contains(t, view, "gh#123")
}

func TestSuggestDetailModel_View_SequenceDAG(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 30)

	p := SuggestProposedPipeline{
		Name:     "research-implement",
		Priority: 1,
		Type:     "sequence",
		Sequence: []string{"research", "implement", "test"},
		Reason:   "Complex feature",
	}
	m, _ = m.Update(SuggestSelectedMsg{Pipeline: p})

	view := m.View()
	assert.Contains(t, view, "Execution:")
	assert.Contains(t, view, "[1] research")
	assert.Contains(t, view, "[2] implement")
	assert.Contains(t, view, "[3] test")
	assert.Contains(t, view, "\u2193") // ↓
}

func TestSuggestDetailModel_View_ParallelDAG(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 30)

	p := SuggestProposedPipeline{
		Name:     "parallel-audit",
		Priority: 2,
		Type:     "parallel",
		Sequence: []string{"audit-security", "audit-dx"},
		Reason:   "Code quality",
	}
	m, _ = m.Update(SuggestSelectedMsg{Pipeline: p})

	view := m.View()
	assert.Contains(t, view, "parallel")
	assert.Contains(t, view, "audit-security")
	assert.Contains(t, view, "audit-dx")
}

func TestSuggestDetailModel_View_MultiSelect(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 30)

	p1 := SuggestProposedPipeline{Name: "pipe-a", Priority: 1, Reason: "Fix bugs"}
	p2 := SuggestProposedPipeline{Name: "pipe-b", Priority: 2, Reason: "Review code", Type: "sequence"}
	m, _ = m.Update(SuggestSelectedMsg{
		Pipeline:      p1,
		MultiSelected: []SuggestProposedPipeline{p1, p2},
	})

	view := m.View()
	assert.Contains(t, view, "Execution Plan")
	assert.Contains(t, view, "2 pipelines")
	assert.Contains(t, view, "pipe-a")
	assert.Contains(t, view, "pipe-b")
}
