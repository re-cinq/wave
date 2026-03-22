package manifest

import (
	"path/filepath"
	"time"
)

type Project struct {
	Language            string `yaml:"language,omitempty"`
	Flavour             string `yaml:"flavour,omitempty"`
	TestCommand         string `yaml:"test_command,omitempty"`
	ContractTestCommand string `yaml:"contract_test_command,omitempty"`
	LintCommand         string `yaml:"lint_command,omitempty"`
	BuildCommand        string `yaml:"build_command,omitempty"`
	FormatCommand       string `yaml:"format_command,omitempty"`
	SourceGlob          string `yaml:"source_glob,omitempty"`
	Skill               string `yaml:"skill,omitempty"`
}

type Manifest struct {
	APIVersion  string              `yaml:"apiVersion"`
	Kind        string              `yaml:"kind"`
	Metadata    Metadata            `yaml:"metadata"`
	Project     *Project            `yaml:"project,omitempty"`
	Adapters    map[string]Adapter  `yaml:"adapters,omitempty"`
	Personas    map[string]Persona  `yaml:"personas,omitempty"`
	Skills      []string            `yaml:"skills,omitempty"`
	Runtime     Runtime                    `yaml:"runtime"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Repo        string `yaml:"repo,omitempty"`
}

type Adapter struct {
	Binary             string      `yaml:"binary"`
	Mode               string      `yaml:"mode"`
	OutputFormat       string      `yaml:"output_format,omitempty"`
	ProjectFiles       []string    `yaml:"project_files,omitempty"`
	DefaultPermissions Permissions `yaml:"default_permissions,omitempty"`
	HooksTemplate      string      `yaml:"hooks_template,omitempty"`
}

type Persona struct {
	Adapter          string          `yaml:"adapter"`
	Description      string          `yaml:"description,omitempty"`
	SystemPromptFile string          `yaml:"system_prompt_file"`
	Temperature      float64         `yaml:"temperature,omitempty"`
	Model            string          `yaml:"model,omitempty"` // Model to use (e.g., "opus", "sonnet")
	Permissions      Permissions     `yaml:"permissions,omitempty"`
	Hooks            HookConfig      `yaml:"hooks,omitempty"`
	Sandbox          *PersonaSandbox `yaml:"sandbox,omitempty"`
	Skills           []string        `yaml:"skills,omitempty"`
	TokenScopes      []string        `yaml:"token_scopes,omitempty"`
}

type PersonaSandbox struct {
	AllowedDomains []string `yaml:"allowed_domains,omitempty"`
}

type Permissions struct {
	AllowedTools []string `yaml:"allowed_tools,omitempty"`
	Deny         []string `yaml:"deny,omitempty"`
}

type HookConfig struct {
	PreToolUse  []HookRule `yaml:"PreToolUse,omitempty"`
	PostToolUse []HookRule `yaml:"PostToolUse,omitempty"`
}

type HookRule struct {
	Matcher string `yaml:"matcher"`
	Command string `yaml:"command"`
}

type Runtime struct {
	WorkspaceRoot        string                 `yaml:"workspace_root"`
	MaxConcurrentWorkers int                    `yaml:"max_concurrent_workers,omitempty"`
	DefaultTimeoutMin    int                    `yaml:"default_timeout_minutes,omitempty"`
	PipelineIDHashLength int                    `yaml:"pipeline_id_hash_length,omitempty"`
	MaxConcurrency       int                    `yaml:"max_concurrency,omitempty"`
	Relay                RelayConfig            `yaml:"relay,omitempty"`
	Audit                AuditConfig            `yaml:"audit,omitempty"`
	MetaPipeline         MetaConfig             `yaml:"meta_pipeline,omitempty"`
	Routing              RoutingConfig          `yaml:"routing,omitempty"`
	Sandbox              RuntimeSandbox         `yaml:"sandbox,omitempty"`
	Artifacts            RuntimeArtifactsConfig `yaml:"artifacts,omitempty"`
}

// GetMaxConcurrency returns the configured maximum step concurrency, defaulting to 10.
func (r *Runtime) GetMaxConcurrency() int {
	if r.MaxConcurrency > 0 {
		return r.MaxConcurrency
	}
	return 10
}

// RuntimeArtifactsConfig holds global configuration for artifact handling.
type RuntimeArtifactsConfig struct {
	MaxStdoutSize      int64  `yaml:"max_stdout_size,omitempty"`      // Max bytes to capture from stdout (default: 10MB)
	DefaultArtifactDir string `yaml:"default_artifact_dir,omitempty"` // Base directory for artifacts (default: ".wave/artifacts")
}

// GetMaxStdoutSize returns the configured max stdout size or the default (10MB).
func (c *RuntimeArtifactsConfig) GetMaxStdoutSize() int64 {
	if c.MaxStdoutSize > 0 {
		return c.MaxStdoutSize
	}
	return 10 * 1024 * 1024 // 10MB default
}

// GetDefaultArtifactDir returns the configured artifact directory or the default.
func (c *RuntimeArtifactsConfig) GetDefaultArtifactDir() string {
	if c.DefaultArtifactDir != "" {
		return c.DefaultArtifactDir
	}
	return ".wave/artifacts"
}

type RuntimeSandbox struct {
	Enabled               bool     `yaml:"enabled"`
	Backend               string   `yaml:"backend,omitempty"`
	DockerImage           string   `yaml:"docker_image,omitempty"`
	DefaultAllowedDomains []string `yaml:"default_allowed_domains,omitempty"`
	EnvPassthrough        []string `yaml:"env_passthrough,omitempty"`
}

// ResolveBackend returns the effective sandbox backend type.
// The Backend field supersedes the Enabled boolean when set.
func (s *RuntimeSandbox) ResolveBackend() string {
	if s.Backend != "" {
		return s.Backend
	}
	if s.Enabled {
		return "bubblewrap"
	}
	return "none"
}

// GetDockerImage returns the Docker image to use, defaulting to ubuntu:24.04.
func (s *RuntimeSandbox) GetDockerImage() string {
	if s.DockerImage != "" {
		return s.DockerImage
	}
	return "ubuntu:24.04"
}

// RoutingConfig holds pipeline routing configuration.
type RoutingConfig struct {
	// Default is the pipeline to use when no routing rules match.
	Default string `yaml:"default,omitempty"`

	// Rules is the list of routing rules evaluated in priority order.
	Rules []RoutingRule `yaml:"rules,omitempty"`
}

// RoutingRule defines a rule for matching work items to pipelines.
type RoutingRule struct {
	// Pattern is a glob pattern for matching input strings.
	// Supports standard glob syntax: *, ?, [abc], [a-z].
	Pattern string `yaml:"pattern,omitempty"`

	// Pipeline is the name of the pipeline to route to when this rule matches.
	Pipeline string `yaml:"pipeline"`

	// Priority determines evaluation order. Higher priority rules are evaluated first.
	// Rules with equal priority are evaluated in definition order.
	Priority int `yaml:"priority,omitempty"`

	// MatchLabels specifies label key-value patterns that must all match.
	// Keys are exact matches, values support glob patterns.
	MatchLabels map[string]string `yaml:"match_labels,omitempty"`
}

type RelayConfig struct {
	TokenThresholdPercent int    `yaml:"token_threshold_percent,omitempty"`
	Strategy              string `yaml:"strategy,omitempty"`
	ContextWindow         int    `yaml:"context_window,omitempty"`
	SummarizerPersona     string `yaml:"summarizer_persona,omitempty"`
}

type AuditConfig struct {
	LogDir               string `yaml:"log_dir,omitempty"`
	LogAllToolCalls      bool   `yaml:"log_all_tool_calls,omitempty"`
	LogAllFileOperations bool   `yaml:"log_all_file_operations,omitempty"`
}

type MetaConfig struct {
	MaxDepth       int `yaml:"max_depth,omitempty"`
	MaxTotalSteps  int `yaml:"max_total_steps,omitempty"`
	MaxTotalTokens int `yaml:"max_total_tokens,omitempty"`
	TimeoutMin     int `yaml:"timeout_minutes,omitempty"`
}

// ProjectVars returns project config as a key-value map for template resolution.
func (p *Project) ProjectVars() map[string]string {
	vars := make(map[string]string)
	if p == nil {
		return vars
	}
	if p.Language != "" {
		vars["project.language"] = p.Language
	}
	if p.TestCommand != "" {
		vars["project.test_command"] = p.TestCommand
	}
	// contract_test_command falls back to test_command if not set.
	// This allows lighter test runs during contract validation (e.g. without -race).
	if p.ContractTestCommand != "" {
		vars["project.contract_test_command"] = p.ContractTestCommand
	} else if p.TestCommand != "" {
		vars["project.contract_test_command"] = p.TestCommand
	}
	if p.LintCommand != "" {
		vars["project.lint_command"] = p.LintCommand
	}
	if p.BuildCommand != "" {
		vars["project.build_command"] = p.BuildCommand
	}
	if p.Flavour != "" {
		vars["project.flavour"] = p.Flavour
	}
	if p.FormatCommand != "" {
		vars["project.format_command"] = p.FormatCommand
	}
	if p.SourceGlob != "" {
		vars["project.source_glob"] = p.SourceGlob
	}
	if p.Skill != "" {
		vars["project.skill"] = p.Skill
	}
	return vars
}

func (m *Manifest) GetAdapter(name string) *Adapter {
	if a, ok := m.Adapters[name]; ok {
		return &a
	}
	return nil
}

func (m *Manifest) GetPersona(name string) *Persona {
	if p, ok := m.Personas[name]; ok {
		return &p
	}
	return nil
}


func (p *Persona) GetSystemPromptPath(root string) string {
	if filepath.IsAbs(p.SystemPromptFile) {
		return p.SystemPromptFile
	}
	return filepath.Join(root, p.SystemPromptFile)
}

func (r *Runtime) GetDefaultTimeout() time.Duration {
	if r.DefaultTimeoutMin > 0 {
		return time.Duration(r.DefaultTimeoutMin) * time.Minute
	}
	return 5 * time.Minute
}
