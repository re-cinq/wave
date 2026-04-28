package manifest

// PipelineMetadata is the minimal subset of pipeline.PipelineMetadata that
// non-orchestrator packages need in order to filter/select shipped pipelines
// without depending on internal/pipeline.
//
// The full type lives in internal/pipeline.PipelineMetadata; the YAML tags
// here intentionally mirror that type so any YAML pipeline definition can
// be unmarshalled into a PipelineHeader without losing the metadata fields.
type PipelineMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Release     bool   `yaml:"release,omitempty"`
	Category    string `yaml:"category,omitempty"`
	Disabled    bool   `yaml:"disabled,omitempty"`
}

// PipelineHeader is a partial view of a pipeline YAML file that exposes only
// the metadata block. It is used by leaf-level packages (e.g. internal/defaults)
// that need to inspect release/disabled flags on shipped pipelines without
// pulling in the full orchestrator type tree from internal/pipeline.
//
// Consumers unmarshal pipeline YAML into this struct; unknown fields
// (steps, hooks, etc.) are ignored by the YAML decoder.
type PipelineHeader struct {
	Metadata PipelineMetadata `yaml:"metadata"`
}
