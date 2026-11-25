package manifest

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// LoadPipelineHeader parses the metadata header of a pipeline YAML document
// from a byte slice. Unknown fields (steps, hooks, requires, ...) are
// ignored, so the loader is suitable for read-only scanners that only care
// about pipeline metadata.
//
// This is the single canonical entry point for header-only pipeline parsing
// across the codebase. Adding a new metadata field requires editing
// PipelineHeader/PipelineMetadata in this package — not a per-call ad-hoc
// struct in the consumer.
func LoadPipelineHeader(data []byte) (*PipelineHeader, error) {
	var header PipelineHeader
	if err := yaml.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("parse pipeline header: %w", err)
	}
	return &header, nil
}
