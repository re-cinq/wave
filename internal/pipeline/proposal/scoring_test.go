package proposal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultScorerBasic(t *testing.T) {
	scorer := &DefaultScorer{}

	health := HealthArtifact{
		Signals: []HealthSignal{
			{Category: "test_failures", Severity: "high", Count: 5, Score: 0.8},
		},
	}

	t.Run("matching pipeline scores positively", func(t *testing.T) {
		entry := CatalogEntry{Name: "test-gen"}
		score := scorer.Score(entry, health)
		assert.InDelta(t, 0.8, score, 0.001)
	})

	t.Run("non-matching pipeline scores zero", func(t *testing.T) {
		entry := CatalogEntry{Name: "doc-fix"}
		score := scorer.Score(entry, health)
		assert.InDelta(t, 0.0, score, 0.001)
	})
}

func TestDefaultScorerMultipleSignals(t *testing.T) {
	scorer := &DefaultScorer{}

	health := HealthArtifact{
		Signals: []HealthSignal{
			{Category: "test_failures", Severity: "high", Count: 5, Score: 0.8},
			{Category: "missing_tests", Severity: "medium", Count: 3, Score: 0.6},
		},
	}

	// test-gen matches both "test_failures" and "missing_tests"
	entry := CatalogEntry{Name: "test-gen"}
	score := scorer.Score(entry, health)
	// Average of 0.8 and 0.6 = 0.7
	assert.InDelta(t, 0.7, score, 0.001)
}

func TestDefaultScorerEmptySignals(t *testing.T) {
	scorer := &DefaultScorer{}
	health := HealthArtifact{Signals: nil}
	entry := CatalogEntry{Name: "test-gen"}
	assert.InDelta(t, 0.0, scorer.Score(entry, health), 0.001)
}

func TestDefaultScorerUnknownCategory(t *testing.T) {
	scorer := &DefaultScorer{}
	health := HealthArtifact{
		Signals: []HealthSignal{
			{Category: "unknown_category", Severity: "high", Score: 0.9},
		},
	}
	entry := CatalogEntry{Name: "test-gen"}
	assert.InDelta(t, 0.0, scorer.Score(entry, health), 0.001)
}

func TestDefaultScorerAllCategories(t *testing.T) {
	scorer := &DefaultScorer{}

	tests := []struct {
		category string
		pipeline string
	}{
		{"test_failures", "test-gen"},
		{"test_failures", "smoke-test"},
		{"dead_code", "dead-code"},
		{"dead_code", "refactor"},
		{"doc_issues", "doc-audit"},
		{"doc_issues", "doc-fix"},
		{"security", "security-scan"},
		{"code_quality", "refactor"},
		{"code_quality", "improve"},
		{"missing_tests", "test-gen"},
		{"api_issues", "refactor"},
		{"api_issues", "improve"},
		{"build_failures", "debug"},
		{"build_failures", "hotfix"},
	}

	for _, tt := range tests {
		t.Run(tt.category+"/"+tt.pipeline, func(t *testing.T) {
			health := HealthArtifact{
				Signals: []HealthSignal{
					{Category: tt.category, Score: 0.7},
				},
			}
			entry := CatalogEntry{Name: tt.pipeline}
			score := scorer.Score(entry, health)
			assert.Greater(t, score, 0.0, "expected positive score for %s → %s", tt.category, tt.pipeline)
		})
	}
}

func TestDefaultScorerClampResult(t *testing.T) {
	scorer := &DefaultScorer{}

	// Even with a high score input, result should not exceed 1.0
	health := HealthArtifact{
		Signals: []HealthSignal{
			{Category: "security", Score: 1.0},
		},
	}
	entry := CatalogEntry{Name: "security-scan"}
	score := scorer.Score(entry, health)
	assert.LessOrEqual(t, score, 1.0)
	assert.GreaterOrEqual(t, score, 0.0)
}

// customScorer implements Scorer for testing the interface.
type customScorer struct {
	fixedScore float64
}

func (c *customScorer) Score(_ CatalogEntry, _ HealthArtifact) float64 {
	return c.fixedScore
}

func TestCustomScorerViaInterface(t *testing.T) {
	var s Scorer = &customScorer{fixedScore: 0.42}
	entry := CatalogEntry{Name: "anything"}
	health := HealthArtifact{}
	assert.InDelta(t, 0.42, s.Score(entry, health), 0.001)
}
