package classify

// LoreProvider supplies historical hints that enrich task classification.
// Implementations may pull from past orchestration decisions, retrospectives,
// or external knowledge bases.
type LoreProvider interface {
	// Hints returns domain/complexity hints for the given input text.
	// Returned hints are advisory — they enrich but never override keyword matching.
	Hints(input string) []LoreHint
}

// LoreHint is a single advisory signal from historical data.
type LoreHint struct {
	Domain     Domain     // suggested domain (empty = no opinion)
	Complexity Complexity // suggested complexity (empty = no opinion)
	Confidence float64    // 0.0–1.0 confidence in this hint
	Source     string     // e.g. "orchestration_history", "retrospective"
}

// NoOpLoreProvider returns no hints. Used as the default when no lore source is configured.
type NoOpLoreProvider struct{}

// Hints always returns nil for the no-op provider.
func (NoOpLoreProvider) Hints(string) []LoreHint { return nil }

var activeLoreProvider LoreProvider = NoOpLoreProvider{}

// RegisterLoreProvider sets the active lore provider. Not concurrency-safe;
// call during init or startup before any classification runs.
func RegisterLoreProvider(p LoreProvider) {
	if p == nil {
		p = NoOpLoreProvider{}
	}
	activeLoreProvider = p
}

// loreProvider returns the currently registered provider.
func loreProvider() LoreProvider {
	return activeLoreProvider
}
