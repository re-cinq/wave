package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortProposalsByPriority(t *testing.T) {
	tests := []struct {
		name string
		in   []Proposal
		want []string // expected pipeline order
	}{
		{
			name: "already sorted",
			in: []Proposal{
				{Pipeline: "A", Priority: 1},
				{Pipeline: "B", Priority: 2},
				{Pipeline: "C", Priority: 3},
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "reverse order",
			in: []Proposal{
				{Pipeline: "C", Priority: 3},
				{Pipeline: "B", Priority: 2},
				{Pipeline: "A", Priority: 1},
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "equal priorities preserve order",
			in: []Proposal{
				{Pipeline: "X", Priority: 1},
				{Pipeline: "Y", Priority: 1},
				{Pipeline: "Z", Priority: 1},
			},
			want: []string{"X", "Y", "Z"},
		},
		{
			name: "single proposal",
			in: []Proposal{
				{Pipeline: "only"},
			},
			want: []string{"only"},
		},
		{
			name: "mixed priorities",
			in: []Proposal{
				{Pipeline: "mid", Priority: 5},
				{Pipeline: "low", Priority: 10},
				{Pipeline: "high", Priority: 1},
			},
			want: []string{"high", "mid", "low"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted := sortProposalsByPriority(tt.in)
			got := make([]string, len(sorted))
			for i, p := range sorted {
				got[i] = p.Pipeline
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSortProposalsByPriority_DoesNotMutateInput(t *testing.T) {
	original := []Proposal{
		{Pipeline: "C", Priority: 3},
		{Pipeline: "A", Priority: 1},
	}
	sortProposalsByPriority(original)
	assert.Equal(t, "C", original[0].Pipeline, "original slice should not be mutated")
	assert.Equal(t, "A", original[1].Pipeline, "original slice should not be mutated")
}

func TestGroupProposalsByParallel(t *testing.T) {
	tests := []struct {
		name       string
		proposals  []Proposal
		wantGroups []struct {
			groupName string
			pipelines []string
		}
	}{
		{
			name: "all ungrouped become singleton groups",
			proposals: []Proposal{
				{Pipeline: "A", Priority: 1},
				{Pipeline: "B", Priority: 2},
				{Pipeline: "C", Priority: 3},
			},
			wantGroups: []struct {
				groupName string
				pipelines []string
			}{
				{groupName: "", pipelines: []string{"A"}},
				{groupName: "", pipelines: []string{"B"}},
				{groupName: "", pipelines: []string{"C"}},
			},
		},
		{
			name: "single parallel group",
			proposals: []Proposal{
				{Pipeline: "A", ParallelGroup: "g1", Priority: 2},
				{Pipeline: "B", ParallelGroup: "g1", Priority: 1},
			},
			wantGroups: []struct {
				groupName string
				pipelines []string
			}{
				{groupName: "g1", pipelines: []string{"B", "A"}},
			},
		},
		{
			name: "mixed grouped and ungrouped",
			proposals: []Proposal{
				{Pipeline: "solo", Priority: 1},
				{Pipeline: "par1", ParallelGroup: "batch", Priority: 2},
				{Pipeline: "par2", ParallelGroup: "batch", Priority: 3},
			},
			wantGroups: []struct {
				groupName string
				pipelines []string
			}{
				{groupName: "", pipelines: []string{"solo"}},
				{groupName: "batch", pipelines: []string{"par1", "par2"}},
			},
		},
		{
			name: "multiple parallel groups",
			proposals: []Proposal{
				{Pipeline: "A", ParallelGroup: "g1", Priority: 1},
				{Pipeline: "B", ParallelGroup: "g2", Priority: 2},
				{Pipeline: "C", ParallelGroup: "g1", Priority: 3},
				{Pipeline: "D", ParallelGroup: "g2", Priority: 4},
			},
			wantGroups: []struct {
				groupName string
				pipelines []string
			}{
				{groupName: "g1", pipelines: []string{"A", "C"}},
				{groupName: "g2", pipelines: []string{"B", "D"}},
			},
		},
		{
			name: "priority ordering within groups",
			proposals: []Proposal{
				{Pipeline: "low", ParallelGroup: "g1", Priority: 10},
				{Pipeline: "high", ParallelGroup: "g1", Priority: 1},
			},
			wantGroups: []struct {
				groupName string
				pipelines []string
			}{
				{groupName: "g1", pipelines: []string{"high", "low"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := groupProposalsByParallel(tt.proposals)
			assert.Len(t, groups, len(tt.wantGroups))

			for i, wg := range tt.wantGroups {
				assert.Equal(t, wg.groupName, groups[i].GroupName, "group %d name", i)
				gotPipelines := make([]string, len(groups[i].Proposals))
				for j, p := range groups[i].Proposals {
					gotPipelines[j] = p.Pipeline
				}
				assert.Equal(t, wg.pipelines, gotPipelines, "group %d pipelines", i)
			}
		})
	}
}

func TestBuildAcceptedPipelines(t *testing.T) {
	tests := []struct {
		name      string
		proposals []Proposal
		decisions map[string]proposalDecisionResult
		want      []string // expected pipeline order
	}{
		{
			name: "all accepted in dependency order",
			proposals: []Proposal{
				{Pipeline: "A"},
				{Pipeline: "B", Dependencies: []string{"A"}},
				{Pipeline: "C", Dependencies: []string{"B"}},
			},
			decisions: map[string]proposalDecisionResult{
				"A": {Decision: Accept, Input: ""},
				"B": {Decision: Accept, Input: ""},
				"C": {Decision: Accept, Input: ""},
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "middle dependency skipped",
			proposals: []Proposal{
				{Pipeline: "A"},
				{Pipeline: "B", Dependencies: []string{"A"}},
				{Pipeline: "C", Dependencies: []string{"B"}},
			},
			decisions: map[string]proposalDecisionResult{
				"A": {Decision: Accept},
				"B": {Decision: Skip},
				"C": {Decision: Accept},
			},
			// C depends on B which is skipped, so C has no accepted dependencies
			// and appears after A (both are roots).
			want: []string{"A", "C"},
		},
		{
			name: "all skipped returns empty",
			proposals: []Proposal{
				{Pipeline: "A"},
				{Pipeline: "B"},
			},
			decisions: map[string]proposalDecisionResult{
				"A": {Decision: Skip},
				"B": {Decision: Skip},
			},
			want: nil,
		},
		{
			name: "modified proposals included with correct input",
			proposals: []Proposal{
				{Pipeline: "build"},
				{Pipeline: "test", Dependencies: []string{"build"}},
			},
			decisions: map[string]proposalDecisionResult{
				"build": {Decision: Modify, Input: "custom input", Flags: []string{"--verbose"}},
				"test":  {Decision: Accept, Input: "run all"},
			},
			want: []string{"build", "test"},
		},
		{
			name: "diamond dependency order",
			proposals: []Proposal{
				{Pipeline: "A"},
				{Pipeline: "B", Dependencies: []string{"A"}},
				{Pipeline: "C", Dependencies: []string{"A"}},
				{Pipeline: "D", Dependencies: []string{"B", "C"}},
			},
			decisions: map[string]proposalDecisionResult{
				"A": {Decision: Accept},
				"B": {Decision: Accept},
				"C": {Decision: Accept},
				"D": {Decision: Accept},
			},
			// Kahn's: A first (0 in-degree), then B,C (sorted), then D.
			want: []string{"A", "B", "C", "D"},
		},
		{
			name: "no dependencies all accepted",
			proposals: []Proposal{
				{Pipeline: "Z"},
				{Pipeline: "A"},
				{Pipeline: "M"},
			},
			decisions: map[string]proposalDecisionResult{
				"Z": {Decision: Accept},
				"A": {Decision: Accept},
				"M": {Decision: Accept},
			},
			// All roots, sorted alphabetically by Kahn's.
			want: []string{"A", "M", "Z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAcceptedPipelines(tt.proposals, tt.decisions)
			got := make([]string, len(result))
			for i, a := range result {
				got[i] = a.Pipeline
			}
			if tt.want == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuildAcceptedPipelines_PreservesInputAndFlags(t *testing.T) {
	proposals := []Proposal{
		{Pipeline: "build"},
	}
	decisions := map[string]proposalDecisionResult{
		"build": {
			Decision: Modify,
			Input:    "custom build args",
			Flags:    []string{"--verbose", "--dry-run"},
		},
	}

	result := buildAcceptedPipelines(proposals, decisions)
	assert.Len(t, result, 1)
	assert.Equal(t, "build", result[0].Pipeline)
	assert.Equal(t, "custom build args", result[0].Input)
	assert.Equal(t, []string{"--verbose", "--dry-run"}, result[0].Flags)
}

func TestFormatAcceptedSummary(t *testing.T) {
	tests := []struct {
		name     string
		accepted []AcceptedPipeline
		want     string
	}{
		{
			name:     "empty",
			accepted: nil,
			want:     "No pipelines selected",
		},
		{
			name: "single pipeline",
			accepted: []AcceptedPipeline{
				{Pipeline: "build"},
			},
			want: "build",
		},
		{
			name: "multiple pipelines",
			accepted: []AcceptedPipeline{
				{Pipeline: "build"},
				{Pipeline: "test"},
				{Pipeline: "deploy"},
			},
			want: "build -> test -> deploy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAcceptedSummary(tt.accepted)
			assert.Equal(t, tt.want, got)
		})
	}
}
