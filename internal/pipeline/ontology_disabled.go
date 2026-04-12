//go:build !ontology

package pipeline

// buildOntologySection is a no-op when built without the "ontology" tag.
func (e *DefaultPipelineExecutor) buildOntologySection(execution *PipelineExecution, step *Step, pipelineID string) string {
	return ""
}

// recordOntologyUsage is a no-op when built without the "ontology" tag.
func (e *DefaultPipelineExecutor) recordOntologyUsage(execution *PipelineExecution, step *Step, stepStatus string) {
}
