package proposal

// Scorer computes relevance scores for pipeline catalog entries given
// a codebase health artifact. Higher scores indicate stronger relevance.
type Scorer interface {
	Score(entry CatalogEntry, health HealthArtifact) float64
}

// signalMapping maps health signal categories to the pipeline names
// they are most relevant to.
var signalMapping = map[string][]string{
	"test_failures":  {"test-gen", "smoke-test"},
	"dead_code":      {"dead-code", "refactor"},
	"doc_issues":     {"doc-audit", "doc-fix"},
	"security":       {"security-scan"},
	"code_quality":   {"refactor", "improve"},
	"missing_tests":  {"test-gen"},
	"api_issues":     {"refactor", "improve"},
	"build_failures": {"debug", "hotfix"},
}

// DefaultScorer maps health signals to pipeline relevance using a
// predefined mapping from signal categories to pipeline names.
type DefaultScorer struct{}

// Score computes a relevance score for a catalog entry based on how
// strongly the health artifact's signals relate to the pipeline.
// Returns a value between 0.0 and 1.0.
func (s *DefaultScorer) Score(entry CatalogEntry, health HealthArtifact) float64 {
	if len(health.Signals) == 0 {
		return 0
	}

	var totalScore float64
	var matchCount int

	for _, signal := range health.Signals {
		pipelines, ok := signalMapping[signal.Category]
		if !ok {
			continue
		}
		for _, p := range pipelines {
			if p == entry.Name {
				totalScore += signal.Score
				matchCount++
			}
		}
	}

	if matchCount == 0 {
		return 0
	}

	// Average of matched signal scores, clamped to [0, 1].
	avg := totalScore / float64(matchCount)
	if avg > 1.0 {
		return 1.0
	}
	if avg < 0.0 {
		return 0.0
	}
	return avg
}
