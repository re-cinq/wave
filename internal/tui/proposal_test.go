package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProposalDecisionString(t *testing.T) {
	tests := []struct {
		name     string
		decision ProposalDecision
		want     string
	}{
		{
			name:     "accept",
			decision: Accept,
			want:     "accept",
		},
		{
			name:     "modify",
			decision: Modify,
			want:     "modify",
		},
		{
			name:     "skip",
			decision: Skip,
			want:     "skip",
		},
		{
			name:     "unknown value",
			decision: ProposalDecision(99),
			want:     "ProposalDecision(99)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.decision.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateProposals(t *testing.T) {
	tests := []struct {
		name      string
		proposals []Proposal
		wantErr   string // empty means no error expected
	}{
		{
			name:      "empty proposals",
			proposals: []Proposal{},
			wantErr:   "empty",
		},
		{
			name: "valid single proposal",
			proposals: []Proposal{
				{Pipeline: "build", Reason: "compile the project"},
			},
			wantErr: "",
		},
		{
			name: "valid multiple proposals",
			proposals: []Proposal{
				{Pipeline: "build", Reason: "compile"},
				{Pipeline: "test", Reason: "run tests"},
				{Pipeline: "deploy", Reason: "ship it"},
			},
			wantErr: "",
		},
		{
			name: "empty pipeline name",
			proposals: []Proposal{
				{Pipeline: "", Reason: "no name"},
			},
			wantErr: "empty pipeline",
		},
		{
			name: "linear chain is valid",
			proposals: []Proposal{
				{Pipeline: "A"},
				{Pipeline: "B", Dependencies: []string{"A"}},
				{Pipeline: "C", Dependencies: []string{"B"}},
			},
			wantErr: "",
		},
		{
			name: "circular dependency A→B→A",
			proposals: []Proposal{
				{Pipeline: "A", Dependencies: []string{"B"}},
				{Pipeline: "B", Dependencies: []string{"A"}},
			},
			wantErr: "circular",
		},
		{
			name: "self-referencing dependency",
			proposals: []Proposal{
				{Pipeline: "A", Dependencies: []string{"A"}},
			},
			wantErr: "circular",
		},
		{
			name: "diamond dependency is valid",
			proposals: []Proposal{
				{Pipeline: "A"},
				{Pipeline: "B", Dependencies: []string{"A"}},
				{Pipeline: "C", Dependencies: []string{"A"}},
				{Pipeline: "D", Dependencies: []string{"B", "C"}},
			},
			wantErr: "",
		},
		{
			name: "dependencies on unknown names are ignored",
			proposals: []Proposal{
				{Pipeline: "A", Dependencies: []string{"unknown"}},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProposals(tt.proposals)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestProposalResultComposition(t *testing.T) {
	result := ProposalResult{
		Pipelines: []AcceptedPipeline{
			{
				Pipeline: "build",
				Input:    "compile all targets",
				Flags:    []string{"--verbose"},
			},
			{
				Pipeline: "test",
				Input:    "run integration suite",
				Flags:    []string{"--race", "--timeout=5m"},
			},
		},
		Aborted: false,
	}

	assert.Len(t, result.Pipelines, 2)
	assert.False(t, result.Aborted)

	assert.Equal(t, "build", result.Pipelines[0].Pipeline)
	assert.Equal(t, "compile all targets", result.Pipelines[0].Input)
	assert.Equal(t, []string{"--verbose"}, result.Pipelines[0].Flags)

	assert.Equal(t, "test", result.Pipelines[1].Pipeline)
	assert.Equal(t, "run integration suite", result.Pipelines[1].Input)
	assert.Equal(t, []string{"--race", "--timeout=5m"}, result.Pipelines[1].Flags)

	// Verify aborted result.
	aborted := ProposalResult{Aborted: true}
	assert.True(t, aborted.Aborted)
	assert.Empty(t, aborted.Pipelines)
}
