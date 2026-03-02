package proposal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCatalog(t *testing.T, entries ...CatalogEntry) *Catalog {
	t.Helper()
	return &Catalog{entries: entries}
}

func TestEnginePropose(t *testing.T) {
	catalog := testCatalog(t,
		CatalogEntry{Name: "test-gen", Description: "Generate tests", Category: "testing"},
		CatalogEntry{Name: "dead-code", Description: "Remove dead code", Category: "quality"},
		CatalogEntry{Name: "security-scan", Description: "Security audit"},
		CatalogEntry{Name: "gh-implement", Description: "Implement issue"},
		CatalogEntry{Name: "refactor", Description: "Refactor code"},
	)

	health := HealthArtifact{
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		Signals: []HealthSignal{
			{Category: "test_failures", Severity: "high", Count: 5, Score: 0.8, Detail: "5 tests failing"},
			{Category: "security", Severity: "critical", Count: 1, Score: 1.0, Detail: "SQL injection found"},
			{Category: "dead_code", Severity: "low", Count: 10, Score: 0.3, Detail: "10 unused functions"},
		},
		Summary: "3 issues detected",
	}

	engine := NewEngine(catalog)
	proposal, err := engine.Propose(health, ForgeGitHub)
	require.NoError(t, err)

	assert.Equal(t, ForgeGitHub, proposal.ForgeType)
	assert.Equal(t, "3 issues detected", proposal.HealthSummary)
	assert.NotEmpty(t, proposal.Proposals)
	assert.False(t, proposal.Timestamp.IsZero())

	// Check that security-scan (score 1.0) is highest priority
	assert.Equal(t, "security-scan", proposal.Proposals[0].Pipeline)
	assert.InDelta(t, 1.0, proposal.Proposals[0].Score, 0.001)
	assert.Equal(t, 1, proposal.Proposals[0].Priority)

	// Check that test-gen is present (matches test_failures)
	found := false
	for _, item := range proposal.Proposals {
		if item.Pipeline == "test-gen" {
			found = true
			assert.Greater(t, item.Score, 0.0)
			assert.NotEmpty(t, item.Rationale)
		}
	}
	assert.True(t, found, "test-gen should be in proposals")

	// Validate the entire proposal
	assert.NoError(t, proposal.Validate())
}

func TestEngineProposeEmptyCatalog(t *testing.T) {
	catalog := testCatalog(t)
	health := HealthArtifact{
		Signals: []HealthSignal{
			{Category: "test_failures", Score: 0.8},
		},
	}

	engine := NewEngine(catalog)
	proposal, err := engine.Propose(health, ForgeGitHub)
	require.NoError(t, err)
	assert.Empty(t, proposal.Proposals)
}

func TestEngineProposeNoSignals(t *testing.T) {
	catalog := testCatalog(t,
		CatalogEntry{Name: "test-gen"},
		CatalogEntry{Name: "dead-code"},
	)
	health := HealthArtifact{
		Summary: "no issues",
	}

	engine := NewEngine(catalog)
	proposal, err := engine.Propose(health, ForgeGitHub)
	require.NoError(t, err)
	assert.Empty(t, proposal.Proposals)
}

func TestEngineProposeForgeFiltering(t *testing.T) {
	catalog := testCatalog(t,
		CatalogEntry{Name: "gh-implement"},
		CatalogEntry{Name: "gl-implement"},
		CatalogEntry{Name: "test-gen"},
	)

	// Use a custom scorer that always returns 0.5 so all pipelines appear
	engine := NewEngine(catalog, WithScorer(&customScorer{fixedScore: 0.5}))

	t.Run("github filter", func(t *testing.T) {
		proposal, err := engine.Propose(HealthArtifact{
			Signals: []HealthSignal{{Category: "test_failures", Score: 0.5}},
		}, ForgeGitHub)
		require.NoError(t, err)

		names := proposalNames(proposal)
		assert.Contains(t, names, "gh-implement")
		assert.Contains(t, names, "test-gen") // forge-agnostic
		assert.NotContains(t, names, "gl-implement")
	})

	t.Run("gitlab filter", func(t *testing.T) {
		proposal, err := engine.Propose(HealthArtifact{
			Signals: []HealthSignal{{Category: "test_failures", Score: 0.5}},
		}, ForgeGitLab)
		require.NoError(t, err)

		names := proposalNames(proposal)
		assert.Contains(t, names, "gl-implement")
		assert.Contains(t, names, "test-gen")
		assert.NotContains(t, names, "gh-implement")
	})
}

func TestEngineProposeNilCatalog(t *testing.T) {
	engine := &Engine{catalog: nil, scorer: &DefaultScorer{}}
	_, err := engine.Propose(HealthArtifact{}, ForgeGitHub)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "catalog is nil")
}

func TestEngineProposeWithCustomScorer(t *testing.T) {
	catalog := testCatalog(t,
		CatalogEntry{Name: "test-gen"},
		CatalogEntry{Name: "refactor"},
	)

	engine := NewEngine(catalog, WithScorer(&customScorer{fixedScore: 0.77}))
	proposal, err := engine.Propose(HealthArtifact{
		Signals: []HealthSignal{{Category: "test_failures", Score: 0.5}},
	}, ForgeGitHub)
	require.NoError(t, err)

	for _, item := range proposal.Proposals {
		assert.InDelta(t, 0.77, item.Score, 0.001)
	}
}

func TestEngineProposeDependencyEdges(t *testing.T) {
	catalog := testCatalog(t,
		CatalogEntry{Name: "security-scan"},  // phase 1
		CatalogEntry{Name: "test-gen"},       // phase 2
		CatalogEntry{Name: "doc-fix"},        // phase 4
	)

	health := HealthArtifact{
		Signals: []HealthSignal{
			{Category: "security", Score: 0.9},
			{Category: "test_failures", Score: 0.7},
			{Category: "doc_issues", Score: 0.5},
		},
	}

	engine := NewEngine(catalog)
	proposal, err := engine.Propose(health, ForgeUnknown)
	require.NoError(t, err)

	// Find doc-fix — it should depend on security-scan and test-gen
	for _, item := range proposal.Proposals {
		if item.Pipeline == "doc-fix" {
			assert.Contains(t, item.DependsOn, "security-scan",
				"doc-fix (phase 4) should depend on security-scan (phase 1)")
			assert.Contains(t, item.DependsOn, "test-gen",
				"doc-fix (phase 4) should depend on test-gen (phase 2)")
		}
		if item.Pipeline == "security-scan" {
			assert.Empty(t, item.DependsOn,
				"security-scan (phase 1) should have no dependencies")
		}
	}
}

func TestEngineProposeParallelGroups(t *testing.T) {
	catalog := testCatalog(t,
		CatalogEntry{Name: "security-scan"}, // phase 1
		CatalogEntry{Name: "dead-code"},     // phase 2
		CatalogEntry{Name: "test-gen"},      // phase 2
	)

	health := HealthArtifact{
		Signals: []HealthSignal{
			{Category: "security", Score: 0.9},
			{Category: "dead_code", Score: 0.5},
			{Category: "test_failures", Score: 0.7},
		},
	}

	engine := NewEngine(catalog)
	proposal, err := engine.Propose(health, ForgeUnknown)
	require.NoError(t, err)

	groups := proposal.ParallelGroups()
	// security-scan is phase 1 → group 0
	// dead-code and test-gen are phase 2, depend on security-scan → group 1
	for _, item := range proposal.Proposals {
		if item.Pipeline == "security-scan" {
			assert.Equal(t, 0, item.ParallelGroup, "security-scan should be in first group")
		}
		if item.Pipeline == "dead-code" || item.Pipeline == "test-gen" {
			assert.Equal(t, 1, item.ParallelGroup, "%s should be in second group", item.Pipeline)
		}
	}
	assert.GreaterOrEqual(t, len(groups), 2)
}

func TestEngineProposeForgeTypeFallback(t *testing.T) {
	catalog := testCatalog(t, CatalogEntry{Name: "test-gen"})

	t.Run("uses health forge when forge param empty", func(t *testing.T) {
		engine := NewEngine(catalog, WithScorer(&customScorer{fixedScore: 0.5}))
		proposal, err := engine.Propose(HealthArtifact{
			ForgeType: ForgeGitLab,
			Signals:   []HealthSignal{{Category: "test_failures", Score: 0.5}},
		}, "")
		require.NoError(t, err)
		assert.Equal(t, ForgeGitLab, proposal.ForgeType)
	})

	t.Run("falls back to unknown when both empty", func(t *testing.T) {
		engine := NewEngine(catalog, WithScorer(&customScorer{fixedScore: 0.5}))
		proposal, err := engine.Propose(HealthArtifact{
			Signals: []HealthSignal{{Category: "test_failures", Score: 0.5}},
		}, "")
		require.NoError(t, err)
		assert.Equal(t, ForgeUnknown, proposal.ForgeType)
	})
}

func proposalNames(p *Proposal) []string {
	names := make([]string, len(p.Proposals))
	for i, item := range p.Proposals {
		names[i] = item.Pipeline
	}
	return names
}
