package manifest

import (
	"path/filepath"
	"time"

	"github.com/recinq/wave/internal/hooks"
)

// ServiceConfig describes a single service in a multi-service project.
type ServiceConfig struct {
	Path                string `yaml:"path"`
	Language            string `yaml:"language,omitempty"`
	BuildCommand        string `yaml:"build_command,omitempty"`
	TestCommand         string `yaml:"test_command,omitempty"`
	ContractTestCommand string `yaml:"contract_test_command,omitempty"`
	SourceGlob          string `yaml:"source_glob,omitempty"`
}

type Project struct {
	Language            string                   `yaml:"language,omitempty"`
	Flavour             string                   `yaml:"flavour,omitempty"`
	TestCommand         string                   `yaml:"test_command,omitempty"`
	ContractTestCommand string                   `yaml:"contract_test_command,omitempty"`
	LintCommand         string                   `yaml:"lint_command,omitempty"`
	BuildCommand        string                   `yaml:"build_command,omitempty"`
	FormatCommand       string                   `yaml:"format_command,omitempty"`
	SourceGlob          string                   `yaml:"source_glob,omitempty"`
	TestFilePattern     []string                 `yaml:"test_file_pattern,omitempty"` // test_diff / test_count_baseline pathspecs (#1583, #1584)
	TestFuncPattern     string                   `yaml:"test_func_pattern,omitempty"` // test_diff / test_count_baseline regex (#1583, #1584)
	Skill               string                   `yaml:"skill,omitempty"`
	Services            map[string]ServiceConfig `yaml:"services,omitempty"`
}

type Manifest struct {
	APIVersion string                   `yaml:"apiVersion"`
	Kind       string                   `yaml:"kind"`
	Metadata   Metadata                 `yaml:"metadata"`
	Project    *Project                 `yaml:"project,omitempty"`
	Adapters   map[string]Adapter       `yaml:"adapters,omitempty"`
	Personas   map[string]Persona       `yaml:"personas,omitempty"`
	Server     *ServerConfig            `yaml:"server,omitempty"`
	Skills     []string                 `yaml:"skills,omitempty"`
	Hooks      []hooks.LifecycleHookDef `yaml:"hooks,omitempty"`
	Runtime    Runtime                  `yaml:"runtime"`
	Evolution  *EvolutionYAML           `yaml:"evolution,omitempty"`

	// RootDir is the directory containing wave.yaml. Set by the loader.
	RootDir string `yaml:"-"`
}

// EvolutionYAML is the operator-facing override for the Phase 3.3 trigger
// thresholds. Field names mirror evolution.Config; zero values fall back to
// the compiled-in defaults so partial overrides keep the rest sane.
//
// Conversion to evolution.Config happens at the executor wiring layer to
// avoid an import cycle between manifest and evolution.
type EvolutionYAML struct {
	Enabled           *bool   `yaml:"enabled,omitempty"`
	EveryNWindow      int     `yaml:"every_n_window,omitempty"`
	EveryNJudgeDrop   float64 `yaml:"every_n_judge_drop,omitempty"`
	DriftWindow       int     `yaml:"drift_window,omitempty"`
	DriftPassDrop     float64 `yaml:"drift_pass_drop,omitempty"`
	RetryWindow       int     `yaml:"retry_window,omitempty"`
	RetryAvgThreshold float64 `yaml:"retry_avg_threshold,omitempty"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Repo        string `yaml:"repo,omitempty"`
	Forge       string `yaml:"forge,omitempty"` // Manual forge override (e.g. "github", "gitlab", "gitea", "forgejo", "bitbucket")
}

type Adapter struct {
	Binary             string      `yaml:"binary"`
	DefaultModel       string      `yaml:"default_model,omitempty"`
	Mode               string      `yaml:"mode"`
	OutputFormat       string      `yaml:"output_format,omitempty"`
	ProjectFiles       []string    `yaml:"project_files,omitempty"`
	DefaultPermissions Permissions `yaml:"default_permissions,omitempty"`
	HooksTemplate      string      `yaml:"hooks_template,omitempty"`
	// TierModels maps complexity tiers to model identifiers for auto-routing.
	// Tiers: "cheapest" (cost-optimized), "balanced" (quality/cost), "strongest" (capability-optimized).
	// If not set, falls back to routing.complexity_map, then adapter default_model.
	TierModels map[string]string `yaml:"tier_models,omitempty"`
}

type Persona struct {
	Adapter          string          `yaml:"adapter"`
	Description      string          `yaml:"description,omitempty"`
	SystemPromptFile string          `yaml:"system_prompt_file"`
	Temperature      float64         `yaml:"temperature,omitempty"`
	Model            string          `yaml:"model,omitempty"` // Model tier (cheapest, balanced, strongest) or literal model identifier
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

// CircuitBreakerConfig controls failure fingerprint tracking and circuit breaking.
type CircuitBreakerConfig struct {
	Limit          int      `yaml:"limit,omitempty"`           // Same failure N times = terminate (default: 3)
	TrackedClasses []string `yaml:"tracked_classes,omitempty"` // Failure classes to track (default: deterministic, contract_failure, test_failure)
}

// RetrosConfig controls automatic retrospective generation after pipeline runs.
type RetrosConfig struct {
	Enabled      *bool  `yaml:"enabled,omitempty"`       // default: true
	Narrate      *bool  `yaml:"narrate,omitempty"`       // LLM narrative (default: true)
	NarrateModel string `yaml:"narrate_model,omitempty"` // model for narration; defaults to cheapest tier model
}

// IsEnabled returns whether retro generation is enabled (default: true).
func (c *RetrosConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsNarrateEnabled returns whether LLM narrative generation is enabled (default: true).
func (c *RetrosConfig) IsNarrateEnabled() bool {
	if c == nil || c.Narrate == nil {
		return true
	}
	return *c.Narrate
}

// GetNarrateModel returns the model to use for narration.
// Defaults to the cheapest tier model from DefaultComplexityMap.
// Set runtime.retros.narrate_model in wave.yaml to override.
func (c *RetrosConfig) GetNarrateModel() string {
	if c == nil || c.NarrateModel == "" {
		return DefaultComplexityMap()["cheapest"]
	}
	return c.NarrateModel
}

type Runtime struct {
	WorkspaceRoot        string                 `yaml:"workspace_root"`
	MaxConcurrentWorkers int                    `yaml:"max_concurrent_workers,omitempty"`
	DefaultTimeoutMin    int                    `yaml:"default_timeout_minutes,omitempty"`
	PipelineIDHashLength int                    `yaml:"pipeline_id_hash_length,omitempty"`
	MaxConcurrency       int                    `yaml:"max_concurrency,omitempty"`
	Timeouts             Timeouts               `yaml:"timeouts,omitempty"`
	Relay                RelayConfig            `yaml:"relay,omitempty"`
	Audit                AuditConfig            `yaml:"audit,omitempty"`
	MetaPipeline         MetaConfig             `yaml:"meta_pipeline,omitempty"`
	Routing              RoutingConfig          `yaml:"routing,omitempty"`
	Sandbox              RuntimeSandbox         `yaml:"sandbox,omitempty"`
	Artifacts            RuntimeArtifactsConfig `yaml:"artifacts,omitempty"`
	CircuitBreaker       CircuitBreakerConfig   `yaml:"circuit_breaker,omitempty"`
	Retros               RetrosConfig           `yaml:"retros,omitempty"`
	Cost                 CostConfig             `yaml:"cost,omitempty"`
	Fallbacks            map[string][]string    `yaml:"fallbacks,omitempty"`     // Adapter fallback chains (e.g., anthropic: [openai, gemini])
	StallTimeout         string                 `yaml:"stall_timeout,omitempty"` // Duration string (e.g. "30m", "1800s"). 0 or empty = disabled.
}

// CostConfig holds cost tracking and budget enforcement settings.
type CostConfig struct {
	// Enabled activates cost tracking for pipeline runs.
	Enabled bool `yaml:"enabled,omitempty"`
	// BudgetCeiling is the maximum cost in USD per pipeline run. 0 = unlimited.
	BudgetCeiling float64 `yaml:"budget_ceiling,omitempty"`
	// WarnAt is the cost threshold (USD) at which to emit a warning. 0 = disabled.
	WarnAt float64 `yaml:"warn_at,omitempty"`
	// Currency is the display currency (default: "USD").
	Currency string `yaml:"currency,omitempty"`
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
	DefaultArtifactDir string `yaml:"default_artifact_dir,omitempty"` // Base directory for artifacts (default: ".agents/artifacts")
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
	return ".agents/artifacts"
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

	// AutoRoute enables automatic model routing based on step complexity heuristics.
	// When enabled, steps without an explicit model override are assigned a model
	// from the ComplexityMap based on their classified complexity tier.
	AutoRoute bool `yaml:"auto_route,omitempty"`

	// ComplexityMap maps complexity tier names to model identifiers.
	// Tiers: "cheapest" (cost-optimized), "balanced" (quality/cost), "strongest" (capability-optimized).
	// Default mapping: "cheapest" -> "claude-haiku-4-5", "balanced" -> "" (adapter default), "strongest" -> "claude-opus-4".
	ComplexityMap map[string]string `yaml:"complexity_map,omitempty"`

	// DefaultTier is the fallback complexity tier when classification is inconclusive.
	// Defaults to "balanced" if not set.
	DefaultTier string `yaml:"default_tier,omitempty"`
}

// DefaultComplexityMap returns the built-in complexity-to-model mapping for Claude adapter.
func DefaultComplexityMap() map[string]string {
	return map[string]string{
		"cheapest":  "claude-haiku-4-5",
		"balanced":  "",
		"strongest": "claude-opus-4",
	}
}

// ResolveComplexityModel returns the model for a given complexity tier,
// consulting the configured ComplexityMap first, then falling back to defaults.
// Returns empty string for the "balanced" tier (use adapter default).
func (r *RoutingConfig) ResolveComplexityModel(tier string) string {
	if r != nil && len(r.ComplexityMap) > 0 {
		if model, ok := r.ComplexityMap[tier]; ok {
			return model
		}
	}
	defaults := DefaultComplexityMap()
	return defaults[tier]
}

// EffectiveDefaultTier returns the configured default tier, falling back to "balanced".
func (r *RoutingConfig) EffectiveDefaultTier() string {
	if r != nil && r.DefaultTier != "" {
		return r.DefaultTier
	}
	return "balanced"
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
	// contract_test_command falls back to test_command when not explicitly set,
	// so command steps using {{ project.contract_test_command }} always resolve.
	switch {
	case p.ContractTestCommand != "":
		vars["project.contract_test_command"] = p.ContractTestCommand
	case p.TestCommand != "":
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

	// Per-service template variables: {{ project.services.<name>.<field> }}
	for name, svc := range p.Services {
		prefix := "project.services." + name + "."
		if svc.Path != "" {
			vars[prefix+"path"] = svc.Path
		}
		if svc.Language != "" {
			vars[prefix+"language"] = svc.Language
		}
		if svc.BuildCommand != "" {
			vars[prefix+"build_command"] = svc.BuildCommand
		}
		if svc.TestCommand != "" {
			vars[prefix+"test_command"] = svc.TestCommand
		}
		switch {
		case svc.ContractTestCommand != "":
			vars[prefix+"contract_test_command"] = svc.ContractTestCommand
		case svc.TestCommand != "":
			vars[prefix+"contract_test_command"] = svc.TestCommand
		}
		if svc.SourceGlob != "" {
			vars[prefix+"source_glob"] = svc.SourceGlob
		}
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
	// Legacy field takes precedence for backward compatibility
	if r.DefaultTimeoutMin > 0 {
		return time.Duration(r.DefaultTimeoutMin) * time.Minute
	}
	return r.Timeouts.GetStepDefault()
}

// ServerConfig holds server-mode configuration from the manifest.
type ServerConfig struct {
	Bind          string           `yaml:"bind,omitempty"`
	MaxConcurrent int              `yaml:"max_concurrent,omitempty"`
	Auth          ServerAuthConfig `yaml:"auth,omitempty"`
	TLS           ServerTLSConfig  `yaml:"tls,omitempty"`
}

// ServerAuthConfig holds authentication configuration for server mode.
type ServerAuthConfig struct {
	Mode      string `yaml:"mode,omitempty"`       // "jwt", "mtls", "bearer", "none"
	JWTSecret string `yaml:"jwt_secret,omitempty"` // supports ${ENV_VAR} expansion
}

// ServerTLSConfig holds TLS configuration for server mode.
type ServerTLSConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Cert    string `yaml:"cert,omitempty"`
	Key     string `yaml:"key,omitempty"`
	CA      string `yaml:"ca,omitempty"` // CA cert for mTLS client verification
}
