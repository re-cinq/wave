package state

// EventStore is the domain-scoped persistence surface for event log entries,
// audit-log queries, and artifact registration/metadata. Consumers that only
// emit or query events/artifacts should depend on this interface rather than
// the aggregate StateStore.
type EventStore interface {
	// Event logging
	LogEvent(runID string, stepID string, state string, persona string, message string, tokens int, durationMs int64, model string, configuredModel string, adapter string) error
	GetEvents(runID string, opts EventQueryOptions) ([]LogRecord, error)
	GetEventAggregateStats(runID string) (*EventAggregateStats, error)

	// Audit log (cross-run event queries)
	GetAuditEvents(states []string, limit, offset int) ([]LogRecord, error)

	// Artifact tracking
	RegisterArtifact(runID string, stepID string, name string, path string, artifactType string, sizeBytes int64) error
	GetArtifacts(runID string, stepID string) ([]ArtifactRecord, error)
	SaveArtifactMetadata(artifactID int64, runID string, stepID string, previewText string, mimeType string, encoding string, metadataJSON string) error
	GetArtifactMetadata(artifactID int64) (*ArtifactMetadataRecord, error)
}
