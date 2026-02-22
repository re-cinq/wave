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
	Requires *Requires        `yaml:"requires,omitempty"`
	Input    InputConfig      `yaml:"input"`
	Steps    []Step           `yaml:"steps"`
}

// Requires declares pipeline dependencies that must be satisfied before execution.
type Requires struct {
	Skills []string `yaml:"skills,omitempty"` // Skill names that must be installed
	Tools  []string `yaml:"tools,omitempty"`  // CLI tools that must be on PATH
}

// PipelineName returns the logical pipeline name from metadata.
func (p *Pipeline) PipelineName() string {
	return p.Metadata.Name
}

type PipelineMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Release     bool   `yaml:"release,omitempty"`
	Disabled    bool   `yaml:"disabled,omitempty"`
}

type InputConfig struct {
	Source      string       `yaml:"source"`
	Schema      *InputSchema `yaml:"schema,omitempty"`
	Example     string       `yaml:"example,omitempty"`
	LabelFilter string       `yaml:"label_filter,omitempty"`
	BatchSize   int          `yaml:"batch_size,omitempty"`
}

type InputSchema struct {
	Type        string `yaml:"type,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Step struct {
	ID              string           `yaml:"id"`
	Persona         string           `yaml:"persona"`
	Dependencies    []string         `yaml:"dependencies,omitempty"`
	Memory          MemoryConfig     `yaml:"memory"`
	Workspace       WorkspaceConfig  `yaml:"workspace"`
	Exec            ExecConfig       `yaml:"exec"`
	OutputArtifacts []ArtifactDef    `yaml:"output_artifacts,omitempty"`
	Outcomes        []OutcomeDef     `yaml:"outcomes,omitempty"`
	Handover        HandoverConfig   `yaml:"handover,omitempty"`
	Strategy        *MatrixStrategy  `yaml:"strategy,omitempty"`
	Validation      []ValidationRule `yaml:"validation,omitempty"`
}

type MemoryConfig struct {
	Strategy        string        `yaml:"strategy"`
	InjectArtifacts []ArtifactRef `yaml:"inject_artifacts,omitempty"`
}

type ArtifactRef struct {
	Step       string `yaml:"step"`
	Artifact   string `yaml:"artifact"`
	As         string `yaml:"as"`
	Type       string `yaml:"type,omitempty"`        // Expected artifact type for validation
	SchemaPath string `yaml:"schema_path,omitempty"` // JSON schema path for input validation
	Optional   bool   `yaml:"optional,omitempty"`    // If true, missing artifact doesn't fail
}

type WorkspaceConfig struct {
	Root   string  `yaml:"root,omitempty"`
	Mount  []Mount `yaml:"mount,omitempty"`
	Type   string  `yaml:"type,omitempty"`   // "worktree" for git worktree, empty for legacy directory
	Branch string  `yaml:"branch,omitempty"` // Branch name for worktree workspaces
	Base   string  `yaml:"base,omitempty"`   // Start point for worktree (e.g. "main")
	Ref    string  `yaml:"ref,omitempty"`    // Reference another step's workspace (shared worktree)
}

type Mount struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Mode   string `yaml:"mode,omitempty"`
}

type ExecConfig struct {
	Type       string `yaml:"type"`                  // "prompt", "command", or "slash_command"
	Source     string `yaml:"source,omitempty"`       // Inline prompt content
	SourcePath string `yaml:"source_path,omitempty"`  // Path to prompt file
	Command    string `yaml:"command,omitempty"`      // Slash command name (for type: slash_command)
	Args       string `yaml:"args,omitempty"`         // Arguments for slash command
}

type ArtifactDef struct {
	Name     string `yaml:"name"`
	Path     string `yaml:"path,omitempty"`           // Optional when Source is "stdout"
	Type     string `yaml:"type,omitempty"`           // "json", "text", "markdown", "binary"
	Required bool   `yaml:"required,omitempty"`
	Source   string `yaml:"source,omitempty"`         // "file" (default) or "stdout"
}

// IsStdoutArtifact returns true if this artifact is captured from stdout.
func (a *ArtifactDef) IsStdoutArtifact() bool {
	return a.Source == "stdout"
}

// GetEffectiveSource returns the effective source, defaulting to "file".
func (a *ArtifactDef) GetEffectiveSource() string {
	if a.Source == "" {
		return "file"
	}
	return a.Source
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

// OutcomeDef declares a structured outcome to extract from step artifacts.
// Outcomes are extracted from JSON artifacts and registered with the deliverable
// tracker, making them appear in the pipeline output summary.
type OutcomeDef struct {
	Type        string `yaml:"type"`         // "pr", "issue", "url", "deployment"
	ExtractFrom string `yaml:"extract_from"` // Artifact path relative to workspace (e.g., "output/publish-result.json")
	JSONPath    string `yaml:"json_path"`    // Dot notation path (e.g., ".comment_url")
	Label       string `yaml:"label,omitempty"`
}
