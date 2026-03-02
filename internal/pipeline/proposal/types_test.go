package proposal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgeTypeConstants(t *testing.T) {
	assert.Equal(t, ForgeType("github"), ForgeGitHub)
	assert.Equal(t, ForgeType("gitlab"), ForgeGitLab)
	assert.Equal(t, ForgeType("gitea"), ForgeGitea)
	assert.Equal(t, ForgeType("bitbucket"), ForgeBitBkt)
	assert.Equal(t, ForgeType("unknown"), ForgeUnknown)
}

func TestValidForgeTypes(t *testing.T) {
	types := ValidForgeTypes()
	assert.Len(t, types, 4)
	assert.NotContains(t, types, ForgeUnknown)
	assert.Contains(t, types, ForgeGitHub)
	assert.Contains(t, types, ForgeGitLab)
	assert.Contains(t, types, ForgeGitea)
	assert.Contains(t, types, ForgeBitBkt)
}

func TestHealthSignalMarshalRoundTrip(t *testing.T) {
	signal := HealthSignal{
		Category: "test_failures",
		Severity: "high",
		Count:    5,
		Score:    0.8,
		Detail:   "5 tests failing in pkg/auth",
	}

	data, err := json.Marshal(signal)
	require.NoError(t, err)

	var decoded HealthSignal
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, signal, decoded)
}

func TestHealthArtifactMarshalRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	artifact := HealthArtifact{
		Version:   "1.0",
		Timestamp: now,
		Signals: []HealthSignal{
			{Category: "security", Severity: "critical", Count: 1, Score: 1.0},
			{Category: "dead_code", Severity: "low", Count: 10, Score: 0.2},
		},
		ForgeType: ForgeGitHub,
		Summary:   "2 issues detected",
	}

	data, err := json.Marshal(artifact)
	require.NoError(t, err)

	var decoded HealthArtifact
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, artifact.Version, decoded.Version)
	assert.Equal(t, artifact.ForgeType, decoded.ForgeType)
	assert.Equal(t, artifact.Summary, decoded.Summary)
	assert.Len(t, decoded.Signals, 2)
	assert.Equal(t, artifact.Timestamp, decoded.Timestamp)
}

func TestHealthArtifactSignalsByCategory(t *testing.T) {
	artifact := HealthArtifact{
		Signals: []HealthSignal{
			{Category: "security", Score: 1.0},
			{Category: "dead_code", Score: 0.2},
			{Category: "security", Score: 0.5},
		},
	}

	sec := artifact.SignalsByCategory("security")
	assert.Len(t, sec, 2)

	dc := artifact.SignalsByCategory("dead_code")
	assert.Len(t, dc, 1)

	none := artifact.SignalsByCategory("nonexistent")
	assert.Empty(t, none)
}

func TestHealthArtifactMaxSeverityScore(t *testing.T) {
	t.Run("with signals", func(t *testing.T) {
		artifact := HealthArtifact{
			Signals: []HealthSignal{
				{Score: 0.3},
				{Score: 0.9},
				{Score: 0.5},
			},
		}
		assert.InDelta(t, 0.9, artifact.MaxSeverityScore(), 0.001)
	})

	t.Run("empty signals", func(t *testing.T) {
		artifact := HealthArtifact{}
		assert.InDelta(t, 0.0, artifact.MaxSeverityScore(), 0.001)
	})
}

func TestProposalItemValidate(t *testing.T) {
	tests := []struct {
		name    string
		item    ProposalItem
		wantErr string
	}{
		{
			name: "valid item",
			item: ProposalItem{
				Pipeline:      "gh-implement",
				Rationale:     "test failures detected",
				Priority:      1,
				Score:         0.8,
				ParallelGroup: 0,
			},
		},
		{
			name:    "missing pipeline",
			item:    ProposalItem{Rationale: "reason", Priority: 1, Score: 0.5},
			wantErr: "pipeline name is required",
		},
		{
			name:    "missing rationale",
			item:    ProposalItem{Pipeline: "test", Priority: 1, Score: 0.5},
			wantErr: "rationale is required",
		},
		{
			name:    "zero priority",
			item:    ProposalItem{Pipeline: "test", Rationale: "reason", Priority: 0, Score: 0.5},
			wantErr: "priority must be >= 1",
		},
		{
			name:    "negative score",
			item:    ProposalItem{Pipeline: "test", Rationale: "reason", Priority: 1, Score: -0.1},
			wantErr: "score must be between 0.0 and 1.0",
		},
		{
			name:    "score too high",
			item:    ProposalItem{Pipeline: "test", Rationale: "reason", Priority: 1, Score: 1.1},
			wantErr: "score must be between 0.0 and 1.0",
		},
		{
			name:    "negative parallel group",
			item:    ProposalItem{Pipeline: "test", Rationale: "reason", Priority: 1, Score: 0.5, ParallelGroup: -1},
			wantErr: "parallel_group must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestProposalValidate(t *testing.T) {
	now := time.Now().UTC()

	t.Run("valid proposal", func(t *testing.T) {
		p := Proposal{
			ForgeType: ForgeGitHub,
			Timestamp: now,
			Proposals: []ProposalItem{
				{Pipeline: "test-gen", Rationale: "tests needed", Priority: 1, Score: 0.9, ParallelGroup: 0},
			},
		}
		assert.NoError(t, p.Validate())
	})

	t.Run("empty proposals list is valid", func(t *testing.T) {
		p := Proposal{
			ForgeType: ForgeGitHub,
			Timestamp: now,
			Proposals: nil,
		}
		assert.NoError(t, p.Validate())
	})

	t.Run("missing forge type", func(t *testing.T) {
		p := Proposal{Timestamp: now}
		err := p.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "forge_type is required")
	})

	t.Run("missing timestamp", func(t *testing.T) {
		p := Proposal{ForgeType: ForgeGitHub}
		err := p.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp is required")
	})

	t.Run("invalid item propagates", func(t *testing.T) {
		p := Proposal{
			ForgeType: ForgeGitHub,
			Timestamp: now,
			Proposals: []ProposalItem{
				{Pipeline: "", Rationale: "reason", Priority: 1, Score: 0.5},
			},
		}
		err := p.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "proposal[0]")
	})
}

func TestProposalMarshalJSON(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	p := Proposal{
		ForgeType: ForgeGitHub,
		Timestamp: now,
		Proposals: []ProposalItem{
			{
				Pipeline:      "test-gen",
				Rationale:     "test coverage low",
				Priority:      1,
				Score:         0.9,
				ParallelGroup: 0,
			},
		},
		HealthSummary: "1 issue found",
	}

	data, err := json.Marshal(&p)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, "github", raw["forge_type"])
	assert.Equal(t, "1 issue found", raw["health_summary"])

	proposals, ok := raw["proposals"].([]interface{})
	require.True(t, ok)
	assert.Len(t, proposals, 1)
}

func TestProposalParallelGroups(t *testing.T) {
	p := Proposal{
		Proposals: []ProposalItem{
			{Pipeline: "a", Rationale: "r", Priority: 1, Score: 0.5, ParallelGroup: 0},
			{Pipeline: "b", Rationale: "r", Priority: 2, Score: 0.4, ParallelGroup: 0},
			{Pipeline: "c", Rationale: "r", Priority: 3, Score: 0.3, ParallelGroup: 1},
			{Pipeline: "d", Rationale: "r", Priority: 4, Score: 0.2, ParallelGroup: 2},
		},
	}

	groups := p.ParallelGroups()
	assert.Len(t, groups, 3)
	assert.Len(t, groups[0], 2)
	assert.Len(t, groups[1], 1)
	assert.Len(t, groups[2], 1)
}
