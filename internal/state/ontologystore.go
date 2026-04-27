package state

// OntologyStore is the domain-scoped persistence surface for ontology usage
// tracking and aggregate stats. Consumers that only record or query ontology
// usage should depend on this interface rather than the aggregate StateStore.
type OntologyStore interface {
	RecordOntologyUsage(runID, stepID, contextName string, invariantCount int, status string, contractPassed *bool) error
	GetOntologyStats(contextName string) (*OntologyStats, error)
	GetOntologyStatsAll() ([]OntologyStats, error)
}
