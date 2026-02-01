// Package contracts defines the core interfaces for the Wave
// orchestrator. These are the contracts between internal packages.
// Implementation details belong in internal/; these interfaces
// define WHAT each component must do, not HOW.
package contracts

import (
	"context"
	"io"
	"time"
)

// --- Manifest Types ---

// Manifest is the parsed representation of wave.yaml.
type Manifest struct {
	APIVersion  string             `yaml:"apiVersion"`
	Kind        string             `yaml:"kind"`
	Metadata    Metadata           `yaml:"metadata"`
	Adapters    map[string]Adapter `yaml:"adapters"`
	Personas    map[string]Persona `yaml:"personas"`
	Runtime     Runtime            `yaml:"runtime"`
	SkillMounts []SkillMount       `yaml:"skill_mounts"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Repo        string `yaml:"repo"`
}

type Adapter struct {
	Binary             string      `yaml:"binary"`
	Mode               string      `yaml:"mode"`
	OutputFormat       string      `yaml:"output_format"`
	ProjectFiles       []string    `yaml:"project_files"`
	DefaultPermissions Permissions `yaml:"default_permissions"`
	HooksTemplate      string      `yaml:"hooks_template"`
}

type Persona struct {
	Adapter          string      `yaml:"adapter"`
	Description      string      `yaml:"description"`
	SystemPromptFile string      `yaml:"system_prompt_file"`
	Temperature      float64     `yaml:"temperature"`
	Permissions      Permissions `yaml:"permissions"`
	Hooks            HookConfig  `yaml:"hooks"`
}

type Permissions struct {
	AllowedTools []string `yaml:"allowed_tools"`
	Deny         []string `yaml:"deny"`
}

type HookConfig struct {
	PreToolUse  []HookRule `yaml:"PreToolUse"`
	PostToolUse []HookRule `yaml:"PostToolUse"`
}

type HookRule struct {
	Matcher string `yaml:"matcher"`
	Command string `yaml:"command"`
}

type Runtime struct {
	WorkspaceRoot        string      `yaml:"workspace_root"`
	MaxConcurrentWorkers int         `yaml:"max_concurrent_workers"`
	DefaultTimeoutMin    int         `yaml:"default_timeout_minutes"`
	Relay                RelayConfig `yaml:"relay"`
	Audit                AuditConfig `yaml:"audit"`
	MetaPipeline         MetaConfig  `yaml:"meta_pipeline"`
}

type RelayConfig struct {
	TokenThresholdPercent int    `yaml:"token_threshold_percent"`
	Strategy              string `yaml:"strategy"`
}

type AuditConfig struct {
	LogDir          string `yaml:"log_dir"`
	LogAllToolCalls bool   `yaml:"log_all_tool_calls"`
	LogAllFileOps   bool   `yaml:"log_all_file_operations"`
}

type MetaConfig struct {
	MaxDepth       int `yaml:"max_depth"`
	MaxTotalSteps  int `yaml:"max_total_steps"`
	MaxTotalTokens int `yaml:"max_total_tokens"`
	TimeoutMin     int `yaml:"timeout_minutes"`
}

type SkillMount struct {
	Path string `yaml:"path"`
}

// --- Pipeline Types ---

type Pipeline struct {
	Kind     string      `yaml:"kind"`
	Metadata Metadata    `yaml:"metadata"`
	Input    InputConfig `yaml:"input"`
	Steps    []Step      `yaml:"steps"`
}

type InputConfig struct {
	Source      string `yaml:"source"`
	LabelFilter string `yaml:"label_filter"`
	BatchSize   int    `yaml:"batch_size"`
}

type Step struct {
	ID              string           `yaml:"id"`
	Persona         string           `yaml:"persona"`
	Dependencies    []string         `yaml:"dependencies"`
	Memory          MemoryConfig     `yaml:"memory"`
	Workspace       WorkspaceConfig  `yaml:"workspace"`
	Exec            ExecConfig       `yaml:"exec"`
	OutputArtifacts []ArtifactDef    `yaml:"output_artifacts"`
	Handover        HandoverConfig   `yaml:"handover"`
	Strategy        *MatrixStrategy  `yaml:"strategy"`
	Validation      []ValidationRule `yaml:"validation"`
}

type MemoryConfig struct {
	Strategy        string        `yaml:"strategy"`
	InjectArtifacts []ArtifactRef `yaml:"inject_artifacts"`
}

type WorkspaceConfig struct {
	Root  string  `yaml:"root"`
	Mount []Mount `yaml:"mount"`
}

type Mount struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Mode   string `yaml:"mode"`
}

type ExecConfig struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
}

type ArtifactDef struct {
	Path     string `yaml:"path"`
	Type     string `yaml:"type"`
	Required bool   `yaml:"required"`
}

type ArtifactRef struct {
	Step     string `yaml:"step"`
	Artifact string `yaml:"artifact"`
	As       string `yaml:"as"`
}

type HandoverConfig struct {
	Contract     ContractConfig   `yaml:"contract"`
	Compaction   CompactionConfig `yaml:"compaction"`
	OnReviewFail string           `yaml:"on_review_fail"`
	TargetStep   string           `yaml:"target_step"`
	MaxRetries   int              `yaml:"max_retries"`
}

type ContractConfig struct {
	Type       string `yaml:"type"`
	Schema     string `yaml:"schema"`
	Source     string `yaml:"source"`
	Validate   bool   `yaml:"validate"`
	Command    string `yaml:"command"`
	MustPass   bool   `yaml:"must_pass"`
	OnFailure  string `yaml:"on_failure"`
	MaxRetries int    `yaml:"max_retries"`
}

type CompactionConfig struct {
	Trigger string `yaml:"trigger"`
	Persona string `yaml:"persona"`
}

type MatrixStrategy struct {
	Type           string `yaml:"type"`
	ItemsSource    string `yaml:"items_source"`
	ItemKey        string `yaml:"item_key"`
	MaxConcurrency int    `yaml:"max_concurrency"`
}

type ValidationRule struct {
	Type       string `yaml:"type"`
	Target     string `yaml:"target"`
	Schema     string `yaml:"schema"`
	OnFailure  string `yaml:"on_failure"`
	MaxRetries int    `yaml:"max_retries"`
}

// --- Step State ---

type StepState string

const (
	StatePending   StepState = "pending"
	StateRunning   StepState = "running"
	StateCompleted StepState = "completed"
	StateFailed    StepState = "failed"
	StateRetrying  StepState = "retrying"
)

// --- Core Interfaces ---

// ManifestLoader parses and validates a wave.yaml file.
type ManifestLoader interface {
	Load(path string) (*Manifest, error)
	Validate(m *Manifest) []error
}

// PipelineLoader parses and validates a pipeline YAML file.
type PipelineLoader interface {
	Load(path string) (*Pipeline, error)
	ValidateDAG(p *Pipeline) error
}

// PipelineExecutor runs a pipeline to completion.
type PipelineExecutor interface {
	Execute(ctx context.Context, p *Pipeline, m *Manifest, input string) error
	Resume(ctx context.Context, pipelineID string, fromStep string) error
}

// AdapterRunner invokes an LLM CLI as a subprocess.
type AdapterRunner interface {
	Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error)
}

type AdapterRunConfig struct {
	Adapter       Adapter
	Persona       Persona
	WorkspacePath string
	Prompt        string
	Timeout       time.Duration
	Env           []string
}

type AdapterResult struct {
	ExitCode   int
	Stdout     io.Reader
	TokensUsed int
	Artifacts  []string
}

// ContractValidator checks step output against a handover contract.
type ContractValidator interface {
	Validate(cfg ContractConfig, workspacePath string) error
}

// WorkspaceManager creates and cleans up ephemeral workspaces.
type WorkspaceManager interface {
	Create(cfg WorkspaceConfig, templateVars map[string]string) (string, error)
	InjectArtifacts(workspacePath string, refs []ArtifactRef, resolvedPaths map[string]string) error
	CleanAll(root string) error
}

// StateStore persists and retrieves pipeline execution state.
type StateStore interface {
	SavePipelineState(id string, status string, input string) error
	SaveStepState(pipelineID string, stepID string, state StepState, err string) error
	GetPipelineState(id string) (*PipelineStateRecord, error)
	GetStepStates(pipelineID string) ([]StepStateRecord, error)
}

type PipelineStateRecord struct {
	PipelineID string
	Name       string
	Status     string
	Input      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type StepStateRecord struct {
	StepID        string
	PipelineID    string
	State         StepState
	RetryCount    int
	StartedAt     *time.Time
	CompletedAt   *time.Time
	WorkspacePath string
	ErrorMessage  string
}

// EventEmitter sends structured progress events to stdout.
type EventEmitter interface {
	Emit(event Event)
}

type Event struct {
	Timestamp  time.Time `json:"timestamp"`
	PipelineID string    `json:"pipeline_id"`
	StepID     string    `json:"step_id"`
	State      StepState `json:"state"`
	DurationMs int64     `json:"duration_ms,omitempty"`
	Message    string    `json:"message,omitempty"`
}

// AuditLogger logs tool calls and file operations.
type AuditLogger interface {
	LogToolCall(pipelineID, stepID, tool, args string)
	LogFileOp(pipelineID, stepID, op, path string)
	Close() error
}

// RelayMonitor watches token usage and triggers compaction.
type RelayMonitor interface {
	ShouldCompact(tokensUsed int, contextWindow int, thresholdPercent int) bool
	Compact(ctx context.Context, chatHistory string, summarizerPersona Persona, adapter Adapter, workspacePath string) (string, error)
}
