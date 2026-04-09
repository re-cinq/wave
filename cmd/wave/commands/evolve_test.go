package commands

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEvolveResult_NoOntology(t *testing.T) {
	m := &manifest.Manifest{}
	stats := []state.OntologyStats{
		{ContextName: "auth", TotalRuns: 10, Successes: 9, Failures: 1, SuccessRate: 90.0, LastUsed: time.Now()},
	}

	result := buildEvolveResult(m, stats)
	assert.Equal(t, 10, result.TotalRuns)
	assert.Empty(t, result.Proposals, "no ontology means no proposals")
	assert.Len(t, result.ContextStats, 1)
}

func TestBuildEvolveResult_LowSignal(t *testing.T) {
	m := &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "auth", Description: "Authentication"},
			},
		},
	}
	stats := []state.OntologyStats{
		{ContextName: "auth", TotalRuns: 3, Successes: 3, Failures: 0, SuccessRate: 100.0, LastUsed: time.Now()},
	}

	result := buildEvolveResult(m, stats)
	require.Len(t, result.Proposals, 1)
	assert.Equal(t, "low_signal", result.Proposals[0].Type)
	assert.Equal(t, "auth", result.Proposals[0].Context)
	assert.Equal(t, 3, result.Proposals[0].TotalRuns)
	assert.Contains(t, result.Proposals[0].Reason, "only 3 runs")
}

func TestBuildEvolveResult_Prune(t *testing.T) {
	m := &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "auth", Description: "Authentication"},
				{Name: "billing", Description: "Billing"},
			},
		},
	}
	// auth has data, billing has none. Total system runs are high (60 via auth).
	stats := []state.OntologyStats{
		{ContextName: "auth", TotalRuns: 60, Successes: 55, Failures: 5, SuccessRate: 91.7, LastUsed: time.Now()},
	}

	result := buildEvolveResult(m, stats)
	assert.Equal(t, 60, result.TotalRuns)

	// billing should be pruned (never injected, system has 60+ runs)
	var pruneProposals []EvolveProposal
	for _, p := range result.Proposals {
		if p.Type == "prune" {
			pruneProposals = append(pruneProposals, p)
		}
	}
	require.Len(t, pruneProposals, 1)
	assert.Equal(t, "billing", pruneProposals[0].Context)
	assert.Contains(t, pruneProposals[0].Reason, "never injected")
}

func TestBuildEvolveResult_Demote(t *testing.T) {
	m := &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "auth", Description: "Authentication"},
			},
		},
	}
	stats := []state.OntologyStats{
		{ContextName: "auth", TotalRuns: 20, Successes: 12, Failures: 8, SuccessRate: 60.0, LastUsed: time.Now()},
	}

	result := buildEvolveResult(m, stats)
	require.Len(t, result.Proposals, 1)
	assert.Equal(t, "demote", result.Proposals[0].Type)
	assert.Equal(t, "auth", result.Proposals[0].Context)
	assert.Equal(t, 60.0, result.Proposals[0].SuccessRate)
	assert.Contains(t, result.Proposals[0].Reason, "60.0%")
	assert.Contains(t, result.Proposals[0].Reason, "below 80%")
}

func TestBuildEvolveResult_Promote(t *testing.T) {
	m := &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "auth", Description: "Authentication"},
			},
		},
	}
	stats := []state.OntologyStats{
		{ContextName: "auth", TotalRuns: 25, Successes: 25, Failures: 0, SuccessRate: 100.0, LastUsed: time.Now()},
	}

	result := buildEvolveResult(m, stats)
	require.Len(t, result.Proposals, 1)
	assert.Equal(t, "promote", result.Proposals[0].Type)
	assert.Equal(t, "auth", result.Proposals[0].Context)
	assert.Equal(t, 100.0, result.Proposals[0].SuccessRate)
	assert.Contains(t, result.Proposals[0].Reason, "candidate for invariant promotion")
}

func TestBuildEvolveResult_Stable(t *testing.T) {
	m := &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "auth", Description: "Authentication"},
				{Name: "billing", Description: "Billing"},
			},
		},
	}
	stats := []state.OntologyStats{
		{ContextName: "auth", TotalRuns: 15, Successes: 13, Failures: 2, SuccessRate: 86.7, LastUsed: time.Now()},
		{ContextName: "billing", TotalRuns: 12, Successes: 11, Failures: 1, SuccessRate: 91.7, LastUsed: time.Now()},
	}

	result := buildEvolveResult(m, stats)
	// Both contexts have >80% success and <100%, not enough for promote or demote
	assert.Empty(t, result.Proposals, "no proposals when ontology is stable")
}

func TestBuildEvolveResult_NeverUsedBelowThreshold(t *testing.T) {
	// When system total runs are below prune threshold, unused contexts get low_signal
	m := &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "billing", Description: "Billing"},
			},
		},
	}
	// No stats at all — system has zero total runs
	stats := []state.OntologyStats{}

	result := buildEvolveResult(m, stats)
	require.Len(t, result.Proposals, 1)
	assert.Equal(t, "low_signal", result.Proposals[0].Type)
	assert.Equal(t, "billing", result.Proposals[0].Context)
}

func TestBuildEvolveResult_MultipleProposals(t *testing.T) {
	m := &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "auth", Description: "Auth"},
				{Name: "billing", Description: "Billing"},
				{Name: "search", Description: "Search"},
				{Name: "orphan", Description: "Orphaned context"},
			},
		},
	}
	stats := []state.OntologyStats{
		{ContextName: "auth", TotalRuns: 30, Successes: 30, Failures: 0, SuccessRate: 100.0, LastUsed: time.Now()},
		{ContextName: "billing", TotalRuns: 15, Successes: 10, Failures: 5, SuccessRate: 66.7, LastUsed: time.Now()},
		{ContextName: "search", TotalRuns: 3, Successes: 3, Failures: 0, SuccessRate: 100.0, LastUsed: time.Now()},
		// orphan: not in stats at all, and total_runs > 50 threshold via max
	}

	result := buildEvolveResult(m, stats)

	proposalTypes := make(map[string][]string)
	for _, p := range result.Proposals {
		proposalTypes[p.Type] = append(proposalTypes[p.Type], p.Context)
	}

	assert.Contains(t, proposalTypes["promote"], "auth", "auth should be promoted (100% over 30 runs)")
	assert.Contains(t, proposalTypes["demote"], "billing", "billing should be demoted (<80% over 15 runs)")
	assert.Contains(t, proposalTypes["low_signal"], "search", "search should be low signal (only 3 runs)")
	assert.Contains(t, proposalTypes["low_signal"], "orphan", "orphan should be low signal (totalRuns=30 < 50)")
}

func TestEvolveResult_JSONSerialization(t *testing.T) {
	result := &EvolveResult{
		TotalRuns: 42,
		Proposals: []EvolveProposal{
			{Type: "demote", Context: "auth", Reason: "low success", SuccessRate: 65.0, TotalRuns: 20},
		},
		ContextStats: []state.OntologyStats{
			{ContextName: "auth", TotalRuns: 20, Successes: 13, Failures: 7, SuccessRate: 65.0},
		},
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(result))

	var decoded EvolveResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Equal(t, 42, decoded.TotalRuns)
	require.Len(t, decoded.Proposals, 1)
	assert.Equal(t, "demote", decoded.Proposals[0].Type)
	assert.Equal(t, "auth", decoded.Proposals[0].Context)
}

func TestRenderEvolveText_Stable(t *testing.T) {
	result := &EvolveResult{
		TotalRuns: 15,
		Proposals: nil,
		ContextStats: []state.OntologyStats{
			{ContextName: "auth", TotalRuns: 15, Successes: 14, Failures: 1, SuccessRate: 93.3},
		},
	}

	var buf bytes.Buffer
	f := newTestFormatter()
	renderEvolveText(&buf, result, f)

	output := buf.String()
	assert.Contains(t, output, "15 pipeline runs")
	assert.Contains(t, output, "auth")
	assert.Contains(t, output, "93.3%")
	assert.Contains(t, output, "No changes proposed")
}

func TestRenderEvolveText_WithProposals(t *testing.T) {
	result := &EvolveResult{
		TotalRuns: 60,
		Proposals: []EvolveProposal{
			{Type: "prune", Context: "orphan", Reason: "never injected"},
			{Type: "demote", Context: "billing", Reason: "low success rate"},
			{Type: "promote", Context: "auth", Reason: "high success"},
			{Type: "low_signal", Context: "search", Reason: "only 3 runs"},
		},
		ContextStats: []state.OntologyStats{},
	}

	var buf bytes.Buffer
	f := newTestFormatter()
	renderEvolveText(&buf, result, f)

	output := buf.String()
	assert.Contains(t, output, "Proposed Changes:")
	assert.Contains(t, output, "orphan")
	assert.Contains(t, output, "billing")
	assert.Contains(t, output, "auth")
	assert.Contains(t, output, "search")
	assert.NotContains(t, output, "No changes proposed")
}

// newTestFormatter returns a Formatter that works in test environments
// (no terminal detection, no ANSI codes that could interfere with assertions).
func newTestFormatter() *display.Formatter {
	return display.NewFormatterWithConfig("off", true)
}
