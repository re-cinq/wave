package pipeline

const (
	StatePending   = "pending"
	StateRunning   = "running"
	StateCompleted = "completed"
	StateFailed    = "failed"
	StateRetrying  = "retrying"
)

type Pipeline struct {
	Kind     string           `yaml:"kind"`
	Metadata PipelineMetadata `yaml:"metadata"`
	Input    InputConfig      `yaml:"input"`
	Steps    []Step           `yaml:"steps"`
}

type PipelineMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Release     bool   `yaml:"release,omitempty"`
	Disabled    bool   `yaml:"disabled,omitempty"`
}

type InputConfig struct {
	Source      string `yaml:"source"`
	LabelFilter string `yaml:"label_filter,omitempty"`
	BatchSize   int    `yaml:"batch_size,omitempty"`
}

type Step struct {
	ID              string           `yaml:"id"`
	Persona         string           `yaml:"persona"`
	Dependencies    []string         `yaml:"dependencies,omitempty"`
	Memory          MemoryConfig     `yaml:"memory"`
	Workspace       WorkspaceConfig  `yaml:"workspace"`
	Exec            ExecConfig       `yaml:"exec"`
	OutputArtifacts []ArtifactDef    `yaml:"output_artifacts,omitempty"`
	Handover        HandoverConfig   `yaml:"handover,omitempty"`
	Strategy        *MatrixStrategy  `yaml:"strategy,omitempty"`
	Validation      []ValidationRule `yaml:"validation,omitempty"`
}

type MemoryConfig struct {
	Strategy        string        `yaml:"strategy"`
	InjectArtifacts []ArtifactRef `yaml:"inject_artifacts,omitempty"`
}

type ArtifactRef struct {
	Step     string `yaml:"step"`
	Artifact string `yaml:"artifact"`
	As       string `yaml:"as"`
}

type WorkspaceConfig struct {
	Root  string  `yaml:"root,omitempty"`
	Mount []Mount `yaml:"mount,omitempty"`
}

type Mount struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Mode   string `yaml:"mode,omitempty"`
}

type ExecConfig struct {
	Type       string `yaml:"type"`
	Source     string `yaml:"source,omitempty"`
	SourcePath string `yaml:"source_path,omitempty"`
}

type ArtifactDef struct {
	Name     string `yaml:"name"`
	Path     string `yaml:"path"`
	Type     string `yaml:"type,omitempty"`
	Required bool   `yaml:"required,omitempty"`
}

type HandoverConfig struct {
	Contract     ContractConfig   `yaml:"contract,omitempty"`
	Compaction   CompactionConfig `yaml:"compaction,omitempty"`
	OnReviewFail string           `yaml:"on_review_fail,omitempty"`
	TargetStep   string           `yaml:"target_step,omitempty"`
	MaxRetries   int              `yaml:"max_retries,omitempty"`
}

type ContractConfig struct {
	Type       string `yaml:"type,omitempty"`
	Schema     string `yaml:"schema,omitempty"`
	Source     string `yaml:"source,omitempty"`
	SchemaPath string `yaml:"schema_path,omitempty"`
	Validate   bool   `yaml:"validate,omitempty"`
	Command    string `yaml:"command,omitempty"`
	Dir        string `yaml:"dir,omitempty"` // Working directory: "project_root", absolute path, or empty for workspace
	MustPass   bool   `yaml:"must_pass,omitempty"`
	OnFailure  string `yaml:"on_failure,omitempty"`
	MaxRetries int    `yaml:"max_retries,omitempty"`
}

type CompactionConfig struct {
	Trigger string `yaml:"trigger,omitempty"`
	Persona string `yaml:"persona,omitempty"`
}

type MatrixStrategy struct {
	Type           string `yaml:"type"`
	ItemsSource    string `yaml:"items_source"`
	ItemKey        string `yaml:"item_key"`
	MaxConcurrency int    `yaml:"max_concurrency,omitempty"`
}

type ValidationRule struct {
	Type       string `yaml:"type"`
	Target     string `yaml:"target,omitempty"`
	Schema     string `yaml:"schema,omitempty"`
	OnFailure  string `yaml:"on_failure,omitempty"`
	MaxRetries int    `yaml:"max_retries,omitempty"`
}
