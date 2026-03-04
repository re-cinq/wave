package tui

import (
	"testing"

	"github.com/recinq/wave/internal/meta"
	"github.com/stretchr/testify/assert"
)

func TestFormatProposalOption_Single(t *testing.T) {
	p := meta.PipelineProposal{
		ID:             "p1",
		Type:           meta.ProposalSingle,
		Pipelines:      []string{"gh-implement"},
		Rationale:      "Open issues found — propose implementation",
		PrefilledInput: "Implement open issues",
		Priority:       1,
		DepsReady:      true,
	}

	result := formatProposalOption(p)

	assert.Contains(t, result, "gh-implement")
	assert.Contains(t, result, "[single]")
	assert.Contains(t, result, "Open issues found")
}

func TestFormatProposalOption_Sequence(t *testing.T) {
	p := meta.PipelineProposal{
		ID:             "p2",
		Type:           meta.ProposalSequence,
		Pipelines:      []string{"gh-research", "gh-implement"},
		Rationale:      "Research then implement open issues",
		PrefilledInput: "Research and implement",
		Priority:       2,
		DepsReady:      true,
	}

	result := formatProposalOption(p)

	assert.Contains(t, result, "gh-research → gh-implement")
	assert.Contains(t, result, "[sequence]")
	assert.Contains(t, result, "Research then implement")
}

func TestFormatProposalOption_Parallel(t *testing.T) {
	p := meta.PipelineProposal{
		ID:             "p3",
		Type:           meta.ProposalParallel,
		Pipelines:      []string{"wave-evolve", "wave-security-audit"},
		Rationale:      "Multiple improvements available",
		PrefilledInput: "Run parallel improvements",
		Priority:       3,
		DepsReady:      true,
	}

	result := formatProposalOption(p)

	assert.Contains(t, result, "wave-evolve | wave-security-audit")
	assert.Contains(t, result, "[parallel]")
	assert.Contains(t, result, "Multiple improvements")
}

func TestFormatProposalOption_MissingDeps(t *testing.T) {
	p := meta.PipelineProposal{
		ID:          "p4",
		Type:        meta.ProposalSingle,
		Pipelines:   []string{"gh-implement"},
		Rationale:   "Open issues found",
		DepsReady:   false,
		MissingDeps: []string{"gh", "speckit"},
	}

	result := formatProposalOption(p)

	assert.Contains(t, result, "gh-implement")
	assert.Contains(t, result, "(missing: gh, speckit)")
}

func TestFormatProposalOption_DepsReadyNoWarning(t *testing.T) {
	p := meta.PipelineProposal{
		ID:        "p5",
		Type:      meta.ProposalSingle,
		Pipelines: []string{"wave-evolve"},
		Rationale: "Evolve the project",
		DepsReady: true,
	}

	result := formatProposalOption(p)

	assert.NotContains(t, result, "missing")
}

func TestFormatPipelineNames(t *testing.T) {
	tests := []struct {
		name         string
		pipelines    []string
		proposalType meta.ProposalType
		want         string
	}{
		{
			name:         "single pipeline",
			pipelines:    []string{"gh-implement"},
			proposalType: meta.ProposalSingle,
			want:         "gh-implement",
		},
		{
			name:         "sequence of two",
			pipelines:    []string{"gh-research", "gh-implement"},
			proposalType: meta.ProposalSequence,
			want:         "gh-research → gh-implement",
		},
		{
			name:         "sequence of three",
			pipelines:    []string{"research", "plan", "implement"},
			proposalType: meta.ProposalSequence,
			want:         "research → plan → implement",
		},
		{
			name:         "parallel pipelines",
			pipelines:    []string{"wave-evolve", "wave-security-audit"},
			proposalType: meta.ProposalParallel,
			want:         "wave-evolve | wave-security-audit",
		},
		{
			name:         "single with default join",
			pipelines:    []string{"a", "b"},
			proposalType: meta.ProposalSingle,
			want:         "a, b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPipelineNames(tt.pipelines, tt.proposalType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildProposalSelection_Single(t *testing.T) {
	proposal := meta.PipelineProposal{
		ID:             "p1",
		Type:           meta.ProposalSingle,
		Pipelines:      []string{"gh-implement"},
		Rationale:      "Open issues found",
		PrefilledInput: "Implement open issues",
		Priority:       1,
		DepsReady:      true,
	}

	selection := buildProposalSelection(proposal, "Custom input text")

	assert.Len(t, selection.Proposals, 1)
	assert.Equal(t, "p1", selection.Proposals[0].ID)
	assert.Equal(t, meta.ProposalSingle, selection.ExecutionMode)
	assert.Equal(t, "Custom input text", selection.ModifiedInputs["gh-implement"])
}

func TestBuildProposalSelection_Sequence(t *testing.T) {
	proposal := meta.PipelineProposal{
		ID:             "p2",
		Type:           meta.ProposalSequence,
		Pipelines:      []string{"gh-research", "gh-implement"},
		Rationale:      "Research then implement",
		PrefilledInput: "Research and implement open issues",
		Priority:       2,
		DepsReady:      true,
	}

	selection := buildProposalSelection(proposal, "Modified input")

	assert.Len(t, selection.Proposals, 1)
	assert.Equal(t, meta.ProposalSequence, selection.ExecutionMode)
	assert.Len(t, selection.ModifiedInputs, 2)
	assert.Equal(t, "Modified input", selection.ModifiedInputs["gh-research"])
	assert.Equal(t, "Modified input", selection.ModifiedInputs["gh-implement"])
}

func TestBuildProposalSelection_Parallel(t *testing.T) {
	proposal := meta.PipelineProposal{
		ID:             "p3",
		Type:           meta.ProposalParallel,
		Pipelines:      []string{"wave-evolve", "wave-security-audit"},
		Rationale:      "Multiple improvements",
		PrefilledInput: "Run improvements",
		Priority:       3,
		DepsReady:      true,
	}

	selection := buildProposalSelection(proposal, "Do everything")

	assert.Len(t, selection.Proposals, 1)
	assert.Equal(t, meta.ProposalParallel, selection.ExecutionMode)
	assert.Len(t, selection.ModifiedInputs, 2)
	assert.Equal(t, "Do everything", selection.ModifiedInputs["wave-evolve"])
	assert.Equal(t, "Do everything", selection.ModifiedInputs["wave-security-audit"])
}

func TestBuildProposalSelection_EmptyInput(t *testing.T) {
	proposal := meta.PipelineProposal{
		ID:        "p1",
		Type:      meta.ProposalSingle,
		Pipelines: []string{"wave-evolve"},
		DepsReady: true,
	}

	selection := buildProposalSelection(proposal, "")

	assert.Equal(t, "", selection.ModifiedInputs["wave-evolve"])
}

func TestBuildProposalSelection_PreservesProposalFields(t *testing.T) {
	proposal := meta.PipelineProposal{
		ID:             "p7",
		Type:           meta.ProposalSingle,
		Pipelines:      []string{"gh-implement"},
		Rationale:      "Important rationale",
		PrefilledInput: "Original prefilled",
		Priority:       1,
		DepsReady:      false,
		MissingDeps:    []string{"gh"},
	}

	selection := buildProposalSelection(proposal, "User modified")

	assert.Equal(t, "p7", selection.Proposals[0].ID)
	assert.Equal(t, "Important rationale", selection.Proposals[0].Rationale)
	assert.Equal(t, "Original prefilled", selection.Proposals[0].PrefilledInput)
	assert.Equal(t, 1, selection.Proposals[0].Priority)
	assert.False(t, selection.Proposals[0].DepsReady)
	assert.Equal(t, []string{"gh"}, selection.Proposals[0].MissingDeps)
	// ModifiedInputs contains the user's modified input, not the prefilled one.
	assert.Equal(t, "User modified", selection.ModifiedInputs["gh-implement"])
}

func TestBuildProposalOptions(t *testing.T) {
	proposals := []meta.PipelineProposal{
		{
			ID:        "p1",
			Type:      meta.ProposalSingle,
			Pipelines: []string{"gh-implement"},
			Rationale: "Open issues",
			DepsReady: true,
		},
		{
			ID:        "p2",
			Type:      meta.ProposalSequence,
			Pipelines: []string{"gh-research", "gh-implement"},
			Rationale: "Research then implement",
			DepsReady: true,
		},
	}

	options := buildProposalOptions(proposals)

	assert.Len(t, options, 2)
	assert.Equal(t, "p1", options[0].Value)
	assert.Equal(t, "p2", options[1].Value)
	assert.Contains(t, options[0].Key, "gh-implement")
	assert.Contains(t, options[1].Key, "gh-research → gh-implement")
}
