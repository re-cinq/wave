package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// T025: Suggest detail rendering
// ===========================================================================

func TestSuggestDetailModel_EmptySelection_ShowsPlaceholder(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 20)

	view := m.View()
	assert.Contains(t, view, "Select a suggestion to view details")
}

func TestSuggestDetailModel_SingleProposal_ShowsTitle(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 20)
	m.selected = &SuggestProposedPipeline{
		Name:     "impl-issue",
		Priority: 1,
		Reason:   "Fix the CI pipeline",
	}

	view := m.View()
	assert.Contains(t, view, "impl-issue")
	assert.Contains(t, view, "1") // priority
}

func TestSuggestDetailModel_ShowsDescription(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 20)
	m.selected = &SuggestProposedPipeline{
		Name:        "impl-issue",
		Priority:    1,
		Description: "Implement a single issue from start to finish",
	}

	view := m.View()
	assert.Contains(t, view, "Implement a single issue from start to finish")
}

func TestSuggestDetailModel_ShowsComplexity(t *testing.T) {
	tests := []struct {
		name       string
		complexity string
		wantLabel  string
	}{
		{"low complexity", "low", "Low"},
		{"medium complexity", "medium", "Medium"},
		{"high complexity", "high", "High"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSuggestDetailModel()
			m.SetSize(60, 20)
			m.selected = &SuggestProposedPipeline{
				Name:       "test-pipeline",
				Priority:   1,
				Complexity: tt.complexity,
			}

			view := m.View()
			assert.Contains(t, view, "Complexity:")
			assert.Contains(t, view, tt.wantLabel)
		})
	}
}

func TestSuggestDetailModel_ShowsSteps(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 30)
	m.selected = &SuggestProposedPipeline{
		Name:     "impl-issue",
		Priority: 1,
		Steps:    []string{"fetch-assess", "plan", "implement", "validate"},
	}

	view := m.View()
	assert.Contains(t, view, "Pipeline Steps")
	assert.Contains(t, view, "fetch-assess")
	assert.Contains(t, view, "plan")
	assert.Contains(t, view, "implement")
	assert.Contains(t, view, "validate")
}

func TestSuggestDetailModel_StepsUseTreeConnectors(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 30)
	m.selected = &SuggestProposedPipeline{
		Name:     "test-pipeline",
		Priority: 1,
		Steps:    []string{"step-a", "step-b", "step-c"},
	}

	view := m.View()
	// Middle steps use branch connector, last uses end connector
	lines := strings.Split(view, "\n")
	var stepLines []string
	for _, line := range lines {
		if strings.Contains(line, "step-") {
			stepLines = append(stepLines, line)
		}
	}
	require.GreaterOrEqual(t, len(stepLines), 3, "should have 3 step lines")

	// First two should have branch connector
	for _, line := range stepLines[:2] {
		assert.True(t, strings.Contains(line, "\u251c\u2500") || strings.Contains(line, "\u2514\u2500"),
			"step line should have a tree connector: %s", line)
	}
}

func TestSuggestDetailModel_ShowsWhySuggested(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 20)
	m.selected = &SuggestProposedPipeline{
		Name:     "impl-issue",
		Priority: 1,
		Reason:   "Issue #42 has clear acceptance criteria and is well-scoped",
	}

	view := m.View()
	assert.Contains(t, view, "Why Suggested")
	assert.Contains(t, view, "Issue #42 has clear acceptance criteria and is well-scoped")
}

func TestSuggestDetailModel_ShowsInput(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 20)
	m.selected = &SuggestProposedPipeline{
		Name:     "impl-issue",
		Priority: 1,
		Input:    "https://github.com/org/repo/issues/42",
	}

	view := m.View()
	assert.Contains(t, view, "Input:")
	assert.Contains(t, view, "https://github.com/org/repo/issues/42")
}

func TestSuggestDetailModel_SequenceType_ShowsSequenceLabel(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 30)
	m.selected = &SuggestProposedPipeline{
		Name:     "multi-step",
		Priority: 1,
		Type:     "sequence",
		Sequence: []string{"research", "implement", "review"},
	}

	view := m.View()
	assert.Contains(t, view, "Sequence:")
	assert.Contains(t, view, "research")
	assert.Contains(t, view, "implement")
	assert.Contains(t, view, "review")
}

func TestSuggestDetailModel_SequenceType_ShowsDAG(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 30)
	m.selected = &SuggestProposedPipeline{
		Name:     "multi-step",
		Priority: 1,
		Type:     "sequence",
		Sequence: []string{"research", "implement"},
	}

	view := m.View()
	assert.Contains(t, view, "Execution Flow")
	assert.Contains(t, view, "[research]")
	assert.Contains(t, view, "[implement]")
}

func TestSuggestDetailModel_MultiSelect_ShowsExecutionPlan(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 30)
	m.selected = &SuggestProposedPipeline{Name: "pipeline-a", Priority: 1}
	m.multiSelected = []SuggestProposedPipeline{
		{Name: "pipeline-a", Priority: 1, Reason: "Fix CI"},
		{Name: "pipeline-b", Priority: 2, Reason: "Review PR"},
	}

	view := m.View()
	assert.Contains(t, view, "Execution Plan")
	assert.Contains(t, view, "2 pipelines")
	assert.Contains(t, view, "pipeline-a")
	assert.Contains(t, view, "pipeline-b")
	assert.Contains(t, view, "Fix CI")
	assert.Contains(t, view, "Review PR")
}

func TestSuggestDetailModel_NoSteps_OmitsStepSection(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 20)
	m.selected = &SuggestProposedPipeline{
		Name:     "impl-issue",
		Priority: 1,
	}

	view := m.View()
	assert.NotContains(t, view, "Pipeline Steps")
}

func TestSuggestDetailModel_NoComplexity_OmitsComplexityLine(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 20)
	m.selected = &SuggestProposedPipeline{
		Name:     "impl-issue",
		Priority: 1,
	}

	view := m.View()
	assert.NotContains(t, view, "Complexity:")
}

func TestSuggestDetailModel_NoDescription_OmitsDescriptionLine(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 20)
	m.selected = &SuggestProposedPipeline{
		Name:     "impl-issue",
		Priority: 1,
	}

	view := m.View()
	// Should not have empty description line between title and priority
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		if strings.Contains(line, "impl-issue") {
			// Next non-empty line should be metadata (Priority:)
			for j := i + 1; j < len(lines); j++ {
				trimmed := strings.TrimSpace(lines[j])
				if trimmed == "" {
					continue
				}
				assert.Contains(t, trimmed, "Priority",
					"first content after title should be Priority")
				break
			}
			break
		}
	}
}

func TestSuggestDetailModel_Update_SuggestSelectedMsg(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(60, 20)

	pipeline := SuggestProposedPipeline{
		Name:     "test-pipeline",
		Priority: 2,
		Reason:   "Test reason",
	}

	m, _ = m.Update(SuggestSelectedMsg{Pipeline: pipeline})

	require.NotNil(t, m.selected)
	assert.Equal(t, "test-pipeline", m.selected.Name)
	assert.Equal(t, 2, m.selected.Priority)
}

func TestSuggestDetailModel_ShowsKeyHints(t *testing.T) {
	m := NewSuggestDetailModel()
	m.SetSize(80, 30)
	m.selected = &SuggestProposedPipeline{
		Name:     "impl-issue",
		Priority: 1,
	}

	view := m.View()
	assert.Contains(t, view, "Enter to launch")
	assert.Contains(t, view, "Space to select")
}

// ===========================================================================
// T026: renderComplexity styling
// ===========================================================================

func TestRenderComplexity_Low_ContainsLow(t *testing.T) {
	result := renderComplexity("low")
	assert.Contains(t, result, "Low")
}

func TestRenderComplexity_Medium_ContainsMedium(t *testing.T) {
	result := renderComplexity("medium")
	assert.Contains(t, result, "Medium")
}

func TestRenderComplexity_High_ContainsHigh(t *testing.T) {
	result := renderComplexity("high")
	assert.Contains(t, result, "High")
}

func TestRenderComplexity_Unknown_PassesThrough(t *testing.T) {
	result := renderComplexity("custom-value")
	assert.Equal(t, "custom-value", result)
}
