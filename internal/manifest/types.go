package manifest

import (
	"path/filepath"
	"time"
)

type Manifest struct {
	APIVersion  string              `yaml:"apiVersion"`
	Kind        string              `yaml:"kind"`
	Metadata    Metadata            `yaml:"metadata"`
	Adapters    map[string]Adapter  `yaml:"adapters,omitempty"`
	Personas    map[string]Persona  `yaml:"personas,omitempty"`
	Runtime     Runtime             `yaml:"runtime"`
	SkillMounts []SkillMount        `yaml:"skill_mounts,omitempty"`
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
	WorkspaceRoot        string         `yaml:"workspace_root"`
	MaxConcurrentWorkers int            `yaml:"max_concurrent_workers,omitempty"`
	DefaultTimeoutMin    int            `yaml:"default_timeout_minutes,omitempty"`
	PipelineIDHashLength int            `yaml:"pipeline_id_hash_length,omitempty"`
	Relay                RelayConfig    `yaml:"relay,omitempty"`
	Audit                AuditConfig    `yaml:"audit,omitempty"`
	MetaPipeline         MetaConfig     `yaml:"meta_pipeline,omitempty"`
	Routing              RoutingConfig  `yaml:"routing,omitempty"`
	Sandbox              RuntimeSandbox `yaml:"sandbox,omitempty"`
}

type RuntimeSandbox struct {
	Enabled               bool     `yaml:"enabled"`
	DefaultAllowedDomains []string `yaml:"default_allowed_domains,omitempty"`
	EnvPassthrough        []string `yaml:"env_passthrough,omitempty"`
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

type SkillMount struct {
	Path string `yaml:"path"`
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
