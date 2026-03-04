package meta

import (
	"testing"

	"github.com/recinq/wave/internal/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeReport creates a HealthReport with the given parameters for testing.
func makeReport(
	family string,
	openIssues, openPRs, recentCommits int,
	tools []DependencyStatus,
	skills []DependencyStatus,
) *HealthReport {
	pt := platform.PlatformUnknown
	switch family {
	case "gh":
		pt = platform.PlatformGitHub
	case "gl":
		pt = platform.PlatformGitLab
	case "bb":
		pt = platform.PlatformBitbucket
	case "gt":
		pt = platform.PlatformGitea
	}
	return &HealthReport{
		Platform: platform.PlatformProfile{
			Type:           pt,
			PipelineFamily: family,
		},
		Codebase: CodebaseMetrics{
			OpenIssueCount: openIssues,
			OpenPRCount:    openPRs,
			RecentCommits:  recentCommits,
		},
		Dependencies: DependencyReport{
			Tools:  tools,
			Skills: skills,
		},
	}
}

func allToolsAvailable() []DependencyStatus {
	return []DependencyStatus{
		{Name: "git", Kind: "tool", Available: true},
		{Name: "claude", Kind: "tool", Available: true},
	}
}

func TestProposalEngine_GitHubWithOpenIssues(t *testing.T) {
	report := makeReport("gh", 2, 0, 20, allToolsAvailable(), nil)
	pipelines := []string{"gh-implement", "gh-research", "gh-pr-review", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	require.NotEmpty(t, proposals)

	// Should have gh-implement (rule 1) and gh-research → gh-implement sequence (rule 5).
	var foundImplement, foundSequence bool
	for _, p := range proposals {
		if p.Type == ProposalSingle && len(p.Pipelines) == 1 && p.Pipelines[0] == "gh-implement" {
			foundImplement = true
			assert.Equal(t, 1, p.Priority)
			assert.True(t, p.DepsReady)
			assert.Contains(t, p.Rationale, "2 open issue(s)")
			assert.Contains(t, p.PrefilledInput, "2 found")
		}
		if p.Type == ProposalSequence && len(p.Pipelines) == 2 {
			assert.Equal(t, "gh-research", p.Pipelines[0])
			assert.Equal(t, "gh-implement", p.Pipelines[1])
			foundSequence = true
			assert.Equal(t, 2, p.Priority)
		}
	}
	assert.True(t, foundImplement, "expected gh-implement proposal")
	assert.True(t, foundSequence, "expected gh-research → gh-implement sequence proposal")
}

func TestProposalEngine_GitLabWithOpenIssues(t *testing.T) {
	report := makeReport("gl", 1, 0, 20, allToolsAvailable(), nil)
	pipelines := []string{"gl-implement", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	require.NotEmpty(t, proposals)

	found := false
	for _, p := range proposals {
		if p.Type == ProposalSingle && len(p.Pipelines) == 1 && p.Pipelines[0] == "gl-implement" {
			found = true
			assert.Equal(t, 1, p.Priority)
			assert.Contains(t, p.Rationale, "1 open issue(s)")
		}
	}
	assert.True(t, found, "expected gl-implement proposal")
}

func TestProposalEngine_ManyOpenIssues(t *testing.T) {
	report := makeReport("gh", 10, 0, 20, allToolsAvailable(), nil)
	pipelines := []string{"gh-implement", "gh-implement-epic", "gh-research", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	require.NotEmpty(t, proposals)

	// Should propose gh-implement-epic (rule 2), NOT gh-implement (rule 1 requires <=3).
	var foundEpic, foundSingle bool
	for _, p := range proposals {
		if p.Type == ProposalSingle && len(p.Pipelines) == 1 && p.Pipelines[0] == "gh-implement-epic" {
			foundEpic = true
			assert.Equal(t, 1, p.Priority)
			assert.Contains(t, p.Rationale, "10 open issues")
		}
		if p.Type == ProposalSingle && len(p.Pipelines) == 1 && p.Pipelines[0] == "gh-implement" {
			foundSingle = true
		}
	}
	assert.True(t, foundEpic, "expected gh-implement-epic proposal for >3 issues")
	assert.False(t, foundSingle, "gh-implement should not be proposed when issues >3")
}

func TestProposalEngine_OpenPRs(t *testing.T) {
	report := makeReport("gh", 0, 3, 20, allToolsAvailable(), nil)
	pipelines := []string{"gh-implement", "gh-pr-review", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	require.NotEmpty(t, proposals)

	found := false
	for _, p := range proposals {
		if p.Type == ProposalSingle && len(p.Pipelines) == 1 && p.Pipelines[0] == "gh-pr-review" {
			found = true
			assert.Equal(t, 2, p.Priority)
			assert.Contains(t, p.Rationale, "3 open PR(s)")
		}
	}
	assert.True(t, found, "expected gh-pr-review proposal")
}

func TestProposalEngine_OpenPRs_PipelineNotInList(t *testing.T) {
	report := makeReport("gh", 0, 3, 20, allToolsAvailable(), nil)
	// gh-pr-review is NOT in the available list.
	pipelines := []string{"gh-implement", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	for _, p := range proposals {
		for _, name := range p.Pipelines {
			assert.NotEqual(t, "gh-pr-review", name, "gh-pr-review should not be proposed when not in available list")
		}
	}
}

func TestProposalEngine_LowCommits(t *testing.T) {
	report := makeReport("gh", 0, 0, 3, allToolsAvailable(), nil)
	pipelines := []string{"gh-implement", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	require.NotEmpty(t, proposals)

	found := false
	for _, p := range proposals {
		if p.Type == ProposalSingle && len(p.Pipelines) == 1 && p.Pipelines[0] == "wave-evolve" {
			found = true
			assert.Equal(t, 3, p.Priority)
			assert.Contains(t, p.Rationale, "3 commits")
		}
	}
	assert.True(t, found, "expected wave-evolve proposal for low commits")
}

func TestProposalEngine_NoActionableSignals(t *testing.T) {
	report := makeReport("gh", 0, 0, 50, allToolsAvailable(), nil)
	pipelines := []string{"gh-implement", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	require.Len(t, proposals, 1)
	assert.Equal(t, ProposalSingle, proposals[0].Type)
	assert.Equal(t, []string{"wave-evolve"}, proposals[0].Pipelines)
	assert.Equal(t, 4, proposals[0].Priority)
	assert.Contains(t, proposals[0].Rationale, "No actionable signals")
}

func TestProposalEngine_MissingDependencies(t *testing.T) {
	tools := []DependencyStatus{
		{Name: "git", Kind: "tool", Available: true},
		{Name: "claude", Kind: "tool", Available: false, Message: "not found"},
	}
	skills := []DependencyStatus{
		{Name: "speckit", Kind: "skill", Available: false, Message: "not installed"},
	}
	report := makeReport("gh", 2, 0, 20, tools, skills)
	pipelines := []string{"gh-implement", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	require.NotEmpty(t, proposals)

	for _, p := range proposals {
		assert.False(t, p.DepsReady, "proposals should have DepsReady=false when dependencies are missing")
		assert.Contains(t, p.MissingDeps, "claude")
		assert.Contains(t, p.MissingDeps, "speckit")
	}
}

func TestProposalEngine_PriorityOrdering(t *testing.T) {
	// Set up conditions to trigger multiple rules at different priorities.
	report := makeReport("gh", 2, 2, 3, allToolsAvailable(), nil)
	pipelines := []string{"gh-implement", "gh-pr-review", "gh-research", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	require.True(t, len(proposals) >= 3, "expected at least 3 proposals, got %d", len(proposals))

	// Verify proposals are sorted by priority (ascending).
	for i := 1; i < len(proposals); i++ {
		assert.LessOrEqual(t, proposals[i-1].Priority, proposals[i].Priority,
			"proposals should be sorted by priority: proposal[%d].Priority=%d > proposal[%d].Priority=%d",
			i-1, proposals[i-1].Priority, i, proposals[i].Priority)
	}

	// First proposal should be priority 1 (gh-implement).
	assert.Equal(t, 1, proposals[0].Priority)
}

func TestProposalEngine_PipelineNotInAvailableList(t *testing.T) {
	report := makeReport("gh", 2, 0, 20, allToolsAvailable(), nil)
	// Only wave-evolve available, not gh-implement.
	pipelines := []string{"wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	// gh-implement should not be proposed since it's not in the list.
	for _, p := range proposals {
		for _, name := range p.Pipelines {
			assert.NotEqual(t, "gh-implement", name, "gh-implement should not appear when not in available list")
		}
	}
}

func TestProposalEngine_UniqueIDs(t *testing.T) {
	report := makeReport("gh", 2, 2, 3, allToolsAvailable(), nil)
	pipelines := []string{"gh-implement", "gh-pr-review", "gh-research", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	require.NotEmpty(t, proposals)

	ids := make(map[string]struct{})
	for _, p := range proposals {
		assert.NotEmpty(t, p.ID)
		_, exists := ids[p.ID]
		assert.False(t, exists, "duplicate proposal ID: %s", p.ID)
		ids[p.ID] = struct{}{}
	}
}

func TestProposalEngine_SequenceNotProposedWithoutResearch(t *testing.T) {
	report := makeReport("gh", 2, 0, 20, allToolsAvailable(), nil)
	// gh-research is NOT available.
	pipelines := []string{"gh-implement", "wave-evolve"}

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	for _, p := range proposals {
		if p.Type == ProposalSequence {
			t.Error("sequence should not be proposed when research pipeline is missing")
		}
	}
}

func TestProposalEngine_EmptyPipelineList(t *testing.T) {
	report := makeReport("gh", 2, 2, 3, allToolsAvailable(), nil)
	pipelines := []string{} // no pipelines available

	engine := NewProposalEngine(report, pipelines)
	proposals := engine.GenerateProposals()

	assert.Empty(t, proposals, "no proposals should be generated when no pipelines are available")
}

func TestNewProposalEngine(t *testing.T) {
	report := &HealthReport{}
	pipelines := []string{"a", "b"}

	engine := NewProposalEngine(report, pipelines)

	assert.NotNil(t, engine)
	assert.Equal(t, report, engine.report)
	assert.Equal(t, pipelines, engine.pipelines)
}

func TestMergeMissingDeps(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []string
		expected []string
	}{
		{
			name:     "no duplicates",
			a:        []string{"git"},
			b:        []string{"claude"},
			expected: []string{"git", "claude"},
		},
		{
			name:     "with duplicates",
			a:        []string{"git", "claude"},
			b:        []string{"claude", "tea"},
			expected: []string{"git", "claude", "tea"},
		},
		{
			name:     "both empty",
			a:        nil,
			b:        nil,
			expected: nil,
		},
		{
			name:     "one empty",
			a:        []string{"git"},
			b:        nil,
			expected: []string{"git"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeMissingDeps(tt.a, tt.b)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestProposalTypes(t *testing.T) {
	assert.Equal(t, ProposalType("single"), ProposalSingle)
	assert.Equal(t, ProposalType("parallel"), ProposalParallel)
	assert.Equal(t, ProposalType("sequence"), ProposalSequence)
}
