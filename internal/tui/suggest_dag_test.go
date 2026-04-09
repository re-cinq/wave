package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// T024: DAG rendering
// ===========================================================================

func TestRenderDAG_SingleProposal_ReturnsEmpty(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Name: "pipeline-a",
		Type: "single",
	}
	result := RenderDAG(proposal)
	assert.Empty(t, result, "single proposal with no Sequence should produce empty string")
}

func TestRenderDAG_SingleProposalWithOneSequenceEntry_ReturnsEmpty(t *testing.T) {
	// Sequence of length 1 is treated the same as single — no DAG needed.
	proposal := SuggestProposedPipeline{
		Name:     "pipeline-a",
		Type:     "sequence",
		Sequence: []string{"pipeline-a"},
	}
	result := RenderDAG(proposal)
	assert.Empty(t, result, "sequence of length 1 should produce empty string")
}

func TestRenderDAG_EmptySequenceList_ReturnsEmpty(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Name:     "pipeline-a",
		Type:     "sequence",
		Sequence: []string{},
	}
	result := RenderDAG(proposal)
	assert.Empty(t, result, "empty sequence should produce empty string")
}

func TestRenderDAG_Sequence_TwoPipelines_ContainsBothNames(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "sequence",
		Sequence: []string{"pipeline-a", "pipeline-b"},
	}
	result := RenderDAG(proposal)

	assert.Contains(t, result, "[pipeline-a]")
	assert.Contains(t, result, "[pipeline-b]")
}

func TestRenderDAG_Sequence_TwoPipelines_ContainsArrow(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "sequence",
		Sequence: []string{"pipeline-a", "pipeline-b"},
	}
	result := RenderDAG(proposal)

	assert.Contains(t, result, "▼", "sequence DAG should include downward arrow")
}

func TestRenderDAG_Sequence_TwoPipelines_OrderPreserved(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "sequence",
		Sequence: []string{"pipeline-a", "pipeline-b"},
	}
	result := RenderDAG(proposal)

	idxA := strings.Index(result, "[pipeline-a]")
	idxB := strings.Index(result, "[pipeline-b]")
	assert.Greater(t, idxB, idxA, "pipeline-b should appear after pipeline-a")
}

func TestRenderDAG_Sequence_ThreePipelines_MultiStepArrows(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "sequence",
		Sequence: []string{"step-1", "step-2", "step-3"},
	}
	result := RenderDAG(proposal)

	assert.Contains(t, result, "[step-1]")
	assert.Contains(t, result, "[step-2]")
	assert.Contains(t, result, "[step-3]")

	// Two arrows between three entries
	arrowCount := strings.Count(result, "▼")
	assert.Equal(t, 2, arrowCount, "three-step sequence should have two arrows")
}

func TestRenderDAG_Sequence_NoArrowAfterLastEntry(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "sequence",
		Sequence: []string{"pipeline-a", "pipeline-b"},
	}
	result := RenderDAG(proposal)

	// The last entry [pipeline-b] should not be followed by an arrow.
	idxB := strings.Index(result, "[pipeline-b]")
	require.GreaterOrEqual(t, idxB, 0, "expected [pipeline-b] in DAG output")
	textAfterB := result[idxB:]
	assert.NotContains(t, textAfterB, "▼", "no arrow should follow the last sequence entry")
}

func TestRenderDAG_Parallel_TwoPipelines_HasGroupedLayout(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "parallel",
		Sequence: []string{"pipeline-a", "pipeline-b"},
	}
	result := RenderDAG(proposal)

	assert.Contains(t, result, "[pipeline-a]")
	assert.Contains(t, result, "[pipeline-b]")
	// First entry uses opening bracket
	assert.Contains(t, result, "┌", "parallel first entry should use ┌")
	// Last entry uses closing bracket
	assert.Contains(t, result, "└", "parallel last entry should use └")
}

func TestRenderDAG_Parallel_TwoPipelines_NoMiddleConnector(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "parallel",
		Sequence: []string{"pipeline-a", "pipeline-b"},
	}
	result := RenderDAG(proposal)

	// Two entries — only first (┌) and last (└), no middle (├)
	assert.NotContains(t, result, "├", "two-entry parallel should have no middle connector")
}

func TestRenderDAG_Parallel_ThreePipelines_HasMiddleConnector(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "parallel",
		Sequence: []string{"pipeline-a", "pipeline-b", "pipeline-c"},
	}
	result := RenderDAG(proposal)

	assert.Contains(t, result, "┌", "parallel should have opening connector")
	assert.Contains(t, result, "├", "parallel with 3 entries should have middle connector")
	assert.Contains(t, result, "└", "parallel should have closing connector")
}

func TestRenderDAG_Parallel_ThreePipelines_ContainsAllNames(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "parallel",
		Sequence: []string{"pipeline-a", "pipeline-b", "pipeline-c"},
	}
	result := RenderDAG(proposal)

	assert.Contains(t, result, "[pipeline-a]")
	assert.Contains(t, result, "[pipeline-b]")
	assert.Contains(t, result, "[pipeline-c]")
}

func TestRenderDAG_Parallel_ContainsConcurrentLabel(t *testing.T) {
	proposal := SuggestProposedPipeline{
		Type:     "parallel",
		Sequence: []string{"pipeline-a", "pipeline-b"},
	}
	result := RenderDAG(proposal)

	assert.Contains(t, result, "(concurrent)", "parallel DAG should label execution as concurrent")
}

func TestRenderDAG_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		proposal        SuggestProposedPipeline
		wantEmpty       bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "single type no sequence",
			proposal: SuggestProposedPipeline{
				Type: "single",
				Name: "only-one",
			},
			wantEmpty: true,
		},
		{
			name: "sequence of 2",
			proposal: SuggestProposedPipeline{
				Type:     "sequence",
				Sequence: []string{"alpha", "beta"},
			},
			wantContains: []string{"[alpha]", "[beta]", "▼"},
		},
		{
			name: "parallel of 2",
			proposal: SuggestProposedPipeline{
				Type:     "parallel",
				Sequence: []string{"worker-1", "worker-2"},
			},
			wantContains:    []string{"[worker-1]", "[worker-2]", "┌", "└", "(concurrent)"},
			wantNotContains: []string{"├"},
		},
		{
			name: "parallel of 3",
			proposal: SuggestProposedPipeline{
				Type:     "parallel",
				Sequence: []string{"a", "b", "c"},
			},
			wantContains: []string{"[a]", "[b]", "[c]", "┌", "├", "└"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := RenderDAG(tc.proposal)

			if tc.wantEmpty {
				assert.Empty(t, result)
				return
			}

			for _, want := range tc.wantContains {
				assert.Contains(t, result, want)
			}
			for _, notWant := range tc.wantNotContains {
				assert.NotContains(t, result, notWant)
			}
		})
	}
}
