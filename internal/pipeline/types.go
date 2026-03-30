package pipeline

import (
	"fmt"
	"sort"
	"time"

	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/timeouts"
)

// Step lifecycle state constants — canonical source: state.StepState.
const (
	StatePending   = string(state.StatePending)
	StateRunning   = string(state.StateRunning)
	StateCompleted = string(state.StateCompleted)
	StateFailed    = string(state.StateFailed)
	StateRetrying  = string(state.StateRetrying)
	StateSkipped   = string(state.StateSkipped)
	StateReworking = string(state.StateReworking)
)

// OnFailure policy constants for contract and step failure handling.
const (
	OnFailureFail     = "fail"
	OnFailureSkip     = "skip"
	OnFailureContinue = "continue"
	OnFailureRework   = "rework"
	OnFailureRetry    = "retry"
)

// Fidelity constants control how much prior thread context a step receives.
const (
	FidelityFull    = "full"    // Complete conversation history (default when thread is set)
	FidelityCompact = "compact" // Step ID + status + truncated content summary
	FidelitySummary = "summary" // LLM-generated summary via relay CompactionAdapter
	FidelityFresh   = "fresh"   // No prior context (default when no thread)
)

// Step type constants for graph-mode pipelines.
const (
	StepTypeConditional = "conditional"
	StepTypeCommand     = "command"
)

type Pipeline struct {
	Kind            string                    `yaml:"kind"`
	Metadata        PipelineMetadata          `yaml:"metadata"`
	Requires        *Requires                 `yaml:"requires,omitempty"`
	Input           InputConfig               `yaml:"input"`
	Steps           []Step                    `yaml:"steps"`
	Hooks           []hooks.LifecycleHookDef  `yaml:"hooks,omitempty"`            // Pipeline-scoped lifecycle hooks
	PipelineOutputs map[string]PipelineOutput `yaml:"pipeline_outputs,omitempty"` // Named output aliases
	ChatContext     *ChatContextConfig        `yaml:"chat_context,omitempty"`     // Chat session context injection
	Skills          []string                  `yaml:"skills,omitempty"`           // Declarative skill references
	MaxStepVisits   int                       `yaml:"max_step_visits,omitempty"`  // Graph-level max total visits across all steps (default 50)
}

// ChatContextConfig configures what context to inject into post-pipeline chat sessions.
type ChatContextConfig struct {
	ArtifactSummaries  []string `yaml:"artifact_summaries,omitempty"`  // Artifact names to summarize in chat
	SuggestedQuestions []string `yaml:"suggested_questions,omitempty"` // Pipeline-specific opening questions
	FocusAreas         []string `yaml:"focus_areas,omitempty"`         // Areas to highlight in chat
	MaxContextTokens   int      `yaml:"max_context_tokens,omitempty"`  // Token budget for injected content (default 8000)
}

// EffectiveMaxContextTokens returns the token budget, defaulting to 8000.
func (c *ChatContextConfig) EffectiveMaxContextTokens() int {
	if c == nil || c.MaxContextTokens <= 0 {
		return 8000
	}
	return c.MaxContextTokens
}

// Requires declares pipeline dependencies that must be satisfied before execution.
type Requires struct {
	Skills map[string]skill.SkillConfig `yaml:"skills,omitempty"` // Skill configs keyed by name
	Tools  []string                     `yaml:"tools,omitempty"`  // CLI tools that must be on PATH
}

// SkillNames returns the skill names in sorted order for deterministic iteration.
func (r *Requires) SkillNames() []string {
	if r == nil {
		return nil
	}
	names := make([]string, 0, len(r.Skills))
	for name := range r.Skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// PipelineName returns the logical pipeline name from metadata.
func (p *Pipeline) PipelineName() string {
	return p.Metadata.Name
}

// EffectiveMaxStepVisits returns the pipeline's max total visits, defaulting to 50.
func (p *Pipeline) EffectiveMaxStepVisits() int {
	if p.MaxStepVisits > 0 {
		return p.MaxStepVisits
	}
	return 50
}

type PipelineMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Release     bool   `yaml:"release,omitempty"`
	Category    string `yaml:"category,omitempty"`
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

// RetryConfig controls step retry behavior on failure.
type RetryConfig struct {
	Policy      string `yaml:"policy,omitempty"`        // "none", "standard", "aggressive", "patient". Resolved to concrete values.
	MaxAttempts int    `yaml:"max_attempts,omitempty"` // Total attempts. Default 1 = no retry
	Backoff     string `yaml:"backoff,omitempty"`      // "fixed", "linear", "exponential". Default: "linear"
	BaseDelay   string `yaml:"base_delay,omitempty"`   // Duration string like "2s". Default: "1s"
	MaxDelay    string `yaml:"max_delay,omitempty"`     // Maximum delay cap. Default: "30s"
	AdaptPrompt bool   `yaml:"adapt_prompt,omitempty"` // Inject prior failure context. Default: false
	OnFailure   string `yaml:"on_failure,omitempty"`   // "fail", "skip", "continue", "rework". Default: "fail"
	ReworkStep  string `yaml:"rework_step,omitempty"`  // Step ID to execute when on_failure is "rework"
}

// Validate checks that the RetryConfig is well-formed.
func (r RetryConfig) Validate() error {
	if r.Policy != "" {
		validPolicies := map[string]bool{"none": true, "standard": true, "aggressive": true, "patient": true}
		if !validPolicies[r.Policy] {
			return fmt.Errorf("unknown retry policy %q (valid: none, standard, aggressive, patient)", r.Policy)
		}
	}
	if r.OnFailure == OnFailureRework && r.ReworkStep == "" {
		return fmt.Errorf("rework_step is required when on_failure is \"rework\"")
	}
	if r.ReworkStep != "" && r.OnFailure != OnFailureRework {
		return fmt.Errorf("rework_step is set but on_failure is %q (must be %q)", r.OnFailure, OnFailureRework)
	}
	return nil
}

// ResolvePolicy fills in retry config fields from a named policy preset.
// Explicit values take precedence over policy defaults.
// Returns an error if the policy name is unrecognized.
func (r *RetryConfig) ResolvePolicy() error {
	if r.Policy == "" {
		return nil
	}

	type policyDefaults struct {
		MaxAttempts int
		Backoff     string
		BaseDelay   string
		MaxDelay    string
	}

	policies := map[string]policyDefaults{
		"none":       {MaxAttempts: 1, Backoff: "fixed", BaseDelay: "0s", MaxDelay: "0s"},
		"standard":   {MaxAttempts: 3, Backoff: "exponential", BaseDelay: "1s", MaxDelay: "30s"},
		"aggressive": {MaxAttempts: 5, Backoff: "exponential", BaseDelay: "200ms", MaxDelay: "30s"},
		"patient":    {MaxAttempts: 3, Backoff: "exponential", BaseDelay: "5s", MaxDelay: "90s"},
	}

	defaults, ok := policies[r.Policy]
	if !ok {
		return fmt.Errorf("unknown retry policy %q (valid: none, standard, aggressive, patient)", r.Policy)
	}

	// Explicit values override policy defaults
	if r.MaxAttempts == 0 {
		r.MaxAttempts = defaults.MaxAttempts
	}
	if r.Backoff == "" {
		r.Backoff = defaults.Backoff
	}
	if r.BaseDelay == "" {
		r.BaseDelay = defaults.BaseDelay
	}
	if r.MaxDelay == "" {
		r.MaxDelay = defaults.MaxDelay
	}
	return nil
}

// EffectiveMaxAttempts returns the number of retry attempts, falling back to 1.
func (r RetryConfig) EffectiveMaxAttempts() int {
	if r.MaxAttempts > 0 {
		return r.MaxAttempts
	}
	return 1
}

// ParseBaseDelay returns the base delay duration, defaulting to 1 second.
func (r RetryConfig) ParseBaseDelay() time.Duration {
	if r.BaseDelay != "" {
		d, err := time.ParseDuration(r.BaseDelay)
		if err == nil {
			return d
		}
	}
	return time.Second
}

// ParseMaxDelay returns the max delay duration from the config.
// If MaxDelay is unset or unparseable, it falls back to timeouts.RetryMaxDelay.
func (r RetryConfig) ParseMaxDelay() time.Duration {
	if r.MaxDelay != "" {
		d, err := time.ParseDuration(r.MaxDelay)
		if err == nil {
			return d
		}
	}
	return timeouts.RetryMaxDelay
}

// ComputeDelay returns the delay for a given attempt number (1-based).
// The result is capped at the configured MaxDelay (or timeouts.RetryMaxDelay).
func (r RetryConfig) ComputeDelay(attempt int) time.Duration {
	base := r.ParseBaseDelay()
	maxDelay := r.ParseMaxDelay()

	var d time.Duration
	switch r.Backoff {
	case "fixed":
		d = base
	case "exponential":
		d = base * time.Duration(1<<uint(attempt-1))
	default: // "linear" or empty
		d = base * time.Duration(attempt)
	}

	if d > maxDelay {
		return maxDelay
	}
	return d
}

// AttemptContext holds failure context from a prior retry attempt for prompt adaptation.
type AttemptContext struct {
	Attempt             int
	MaxAttempts         int
	PriorError          string
	FailureClass        string
	PriorStdout         string            // last 2000 chars
	ContractErrors      []string          // Structured contract validation errors
	StepDuration        time.Duration     // How long the step ran before failing
	PartialArtifacts    map[string]string // Partial artifact paths (name -> path)
	FailedStepID        string            // ID of the step that triggered rework
	ReviewFeedbackPath  string            // Path to review_feedback.json written by agent_review on_failure: rework
}

// EdgeConfig defines an edge from a step to a target step with an optional condition.
type EdgeConfig struct {
	Target    string `yaml:"target"`
	Condition string `yaml:"condition,omitempty"`
}

type Step struct {
	ID                  string           `yaml:"id"`
	Persona             string           `yaml:"persona"`
	Adapter             string           `yaml:"adapter,omitempty"`  // Step-level adapter override (e.g., "codex", "gemini")
	Model               string           `yaml:"model,omitempty"`   // Step-level model override (e.g., "claude-haiku-4-5")
	Dependencies        []string         `yaml:"dependencies,omitempty"`
	TimeoutMinutes      int              `yaml:"timeout_minutes,omitempty"`
	Optional            bool             `yaml:"optional,omitempty"`
	Memory              MemoryConfig     `yaml:"memory"`
	Workspace           WorkspaceConfig  `yaml:"workspace"`
	Exec                ExecConfig       `yaml:"exec"`
	OutputArtifacts     []ArtifactDef    `yaml:"output_artifacts,omitempty"`
	Outcomes            []OutcomeDef     `yaml:"outcomes,omitempty"`
	Handover            HandoverConfig   `yaml:"handover,omitempty"`
	Retry               RetryConfig      `yaml:"retry,omitempty"`
	ReworkOnly          bool             `yaml:"rework_only,omitempty"` // Only runs via rework trigger, not normal DAG scheduling
	Strategy            *MatrixStrategy  `yaml:"strategy,omitempty"`
	Validation          []ValidationRule `yaml:"validation,omitempty"`
	MaxConcurrentAgents int              `yaml:"max_concurrent_agents,omitempty"`
	Concurrency         int              `yaml:"concurrency,omitempty"`

	// Graph-mode fields
	Type      string       `yaml:"type,omitempty"`       // "conditional", "command", or empty (default prompt)
	Edges     []EdgeConfig `yaml:"edges,omitempty"`      // Outgoing edges for graph-mode routing
	MaxVisits int          `yaml:"max_visits,omitempty"` // Max times this step can be visited in a loop (default 10)
	Script    string       `yaml:"script,omitempty"`     // Shell script for command steps

	// Thread conversation continuity — steps sharing the same thread value
	// participate in a conversation thread, receiving prior step transcripts.
	Thread   string `yaml:"thread,omitempty"`   // Thread group ID (opt-in; empty = fresh memory)
	Fidelity string `yaml:"fidelity,omitempty"` // Context fidelity: full, compact, summary, fresh

	// Ontology context filter — when set, only these bounded contexts are injected
	Contexts []string `yaml:"contexts,omitempty"`

	// Composition primitives
	SubPipeline string           `yaml:"pipeline,omitempty"`  // Child pipeline to execute
	SubInput    string              `yaml:"input,omitempty"`        // Input template for child pipeline
	Config      *SubPipelineConfig  `yaml:"config,omitempty"`       // Sub-pipeline configuration (artifact flow, lifecycle)
	Iterate     *IterateConfig   `yaml:"iterate,omitempty"`   // Iteration over items
	Branch      *BranchConfig    `yaml:"branch,omitempty"`    // Conditional branching
	Gate        *GateConfig      `yaml:"gate,omitempty"`      // Approval/timer/merge gates
	Loop        *LoopConfig      `yaml:"loop,omitempty"`      // Feedback loops
	Aggregate   *AggregateConfig `yaml:"aggregate,omitempty"` // Output aggregation
}

// IsOptional returns whether this step is marked as optional.
func (s *Step) IsOptional() bool {
	return s.Optional
}

// EffectiveFidelity returns the fidelity level for this step.
// Defaults to "full" when thread is set, "fresh" when no thread.
func (s *Step) EffectiveFidelity() string {
	if s.Fidelity != "" {
		return s.Fidelity
	}
	if s.Thread != "" {
		return FidelityFull
	}
	return FidelityFresh
}

// GetTimeout returns the step-level timeout duration.
// Returns zero if no step-level timeout is configured.
func (s *Step) GetTimeout() time.Duration {
	if s.TimeoutMinutes > 0 {
		return time.Duration(s.TimeoutMinutes) * time.Minute
	}
	return 0
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
	Pipeline   string `yaml:"pipeline,omitempty"`    // Cross-pipeline artifact source (pipeline name)
}

// Validate checks that the ArtifactRef is well-formed.
// Step and Pipeline are mutually exclusive: Step references an artifact from
// another step in the same pipeline, while Pipeline references an artifact
// from a different pipeline's outputs.
func (r ArtifactRef) Validate(stepID string, idx int) error {
	if r.Step != "" && r.Pipeline != "" {
		return fmt.Errorf("step %q inject_artifacts[%d]: step and pipeline are mutually exclusive (got step=%q, pipeline=%q)",
			stepID, idx, r.Step, r.Pipeline)
	}
	return nil
}

type WorkspaceConfig struct {
	Root   string  `yaml:"root,omitempty"`
	Mount  []Mount `yaml:"mount,omitempty"`
	Type   string  `yaml:"type,omitempty"`   // "worktree" for git worktree, empty for basic directory workspace
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
	Source     string `yaml:"source,omitempty"`      // Inline prompt content
	SourcePath string `yaml:"source_path,omitempty"` // Path to prompt file
	Command    string `yaml:"command,omitempty"`     // Slash command name (for type: slash_command)
	Args       string `yaml:"args,omitempty"`        // Arguments for slash command
}

type ArtifactDef struct {
	Name     string `yaml:"name"`
	Path     string `yaml:"path,omitempty"` // Optional when Source is "stdout"
	Type     string `yaml:"type,omitempty"` // "json", "text", "markdown", "binary"
	Required bool   `yaml:"required,omitempty"`
	Source   string `yaml:"source,omitempty"` // "file" (default) or "stdout"
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
	Contracts    []ContractConfig `yaml:"contracts,omitempty"`
	Compaction   CompactionConfig `yaml:"compaction,omitempty"`
	OnReviewFail string           `yaml:"on_review_fail,omitempty"`
	TargetStep   string           `yaml:"target_step,omitempty"`
}

// EffectiveContracts returns the ordered list of contracts to validate.
// If Contracts (plural) is non-empty, it takes precedence.
// If only the singular Contract is set, it is wrapped in a slice.
// If neither is set, nil is returned.
func (h *HandoverConfig) EffectiveContracts() []ContractConfig {
	if len(h.Contracts) > 0 {
		return h.Contracts
	}
	if h.Contract.Type != "" {
		return []ContractConfig{h.Contract}
	}
	return nil
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

	// LLM judge settings
	Model     string   `yaml:"model,omitempty"`    // LLM model for judge evaluation
	Criteria  []string `yaml:"criteria,omitempty"` // Evaluation criteria for LLM judge
	Threshold float64  `yaml:"threshold,omitempty"` // Pass threshold (0.0-1.0), default 1.0

	// Agent review settings
	Persona      string                `yaml:"persona,omitempty"`       // Reviewer persona name (must differ from step persona)
	CriteriaPath string                `yaml:"criteria_path,omitempty"` // Path to review criteria markdown file
	Context      []ReviewContextSource `yaml:"context,omitempty"`       // Context sources for the reviewer
	TokenBudget  int                   `yaml:"token_budget,omitempty"`  // Max tokens for review agent (0 = unlimited)
	Timeout      string                `yaml:"timeout,omitempty"`       // Duration string for review timeout (e.g. "60s")
	ReworkStep   string                `yaml:"rework_step,omitempty"`   // Step ID to execute on review failure with on_failure: rework
}

// ReviewContextSource defines a single context item provided to the reviewing agent.
type ReviewContextSource struct {
	Source   string `yaml:"source,omitempty"`   // "git_diff" or "artifact"
	Artifact string `yaml:"artifact,omitempty"` // Artifact name when source is "artifact"
	MaxSize  int    `yaml:"max_size,omitempty"` // Max bytes for this source (0 = use default)
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
	ItemIDKey      string `yaml:"item_id_key,omitempty"`
	DependencyKey  string `yaml:"dependency_key,omitempty"`
	ChildPipeline  string `yaml:"child_pipeline,omitempty"`
	InputTemplate  string `yaml:"input_template,omitempty"`
	Stacked        bool   `yaml:"stacked,omitempty"`
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
	Type          string `yaml:"type"`                      // "pr", "issue", "url", "deployment"
	ExtractFrom   string `yaml:"extract_from"`              // Artifact path relative to workspace (e.g., "output/publish-result.json")
	JSONPath      string `yaml:"json_path"`                 // Dot notation path (e.g., ".comment_url")
	JSONPathLabel string `yaml:"json_path_label,omitempty"` // Label extraction path for [*] array items
	Label         string `yaml:"label,omitempty"`
}

// validOutcomeTypes enumerates the accepted outcome types.
var validOutcomeTypes = map[string]bool{
	"pr": true, "issue": true, "url": true, "deployment": true,
}

// Validate checks that required fields are set and the type is recognized.
func (o OutcomeDef) Validate(stepID string, idx int) error {
	if o.Type == "" {
		return fmt.Errorf("step %q outcome[%d]: type is required", stepID, idx)
	}
	if !validOutcomeTypes[o.Type] {
		return fmt.Errorf("step %q outcome[%d]: unknown type %q (valid: pr, issue, url, deployment)", stepID, idx, o.Type)
	}
	if o.ExtractFrom == "" {
		return fmt.Errorf("step %q outcome[%d]: extract_from is required", stepID, idx)
	}
	if o.JSONPath == "" {
		return fmt.Errorf("step %q outcome[%d]: json_path is required", stepID, idx)
	}
	return nil
}

// IsGraphStep returns true if this step has graph-mode routing (edges or conditional type).
func (s *Step) IsGraphStep() bool {
	return len(s.Edges) > 0 || s.Type == StepTypeConditional
}

// EffectiveMaxVisits returns the step's max visit count, defaulting to 10.
func (s *Step) EffectiveMaxVisits() int {
	if s.MaxVisits > 0 {
		return s.MaxVisits
	}
	return 10
}

// IsCompositionStep returns true if the step uses any composition primitive.
func (s *Step) IsCompositionStep() bool {
	return s.SubPipeline != "" || s.Iterate != nil || s.Branch != nil || s.Gate != nil || s.Loop != nil || s.Aggregate != nil
}

// IterateConfig configures iteration over a collection of items.
type IterateConfig struct {
	Over          string `yaml:"over"`                     // Template expression resolving to JSON array
	Mode          string `yaml:"mode"`                     // "sequential" or "parallel"
	MaxConcurrent int    `yaml:"max_concurrent,omitempty"` // Max parallel workers (parallel mode)
}

// BranchConfig configures conditional pipeline selection.
type BranchConfig struct {
	On    string            `yaml:"on"`    // Template expression to evaluate
	Cases map[string]string `yaml:"cases"` // value → pipeline name ("skip" = no-op)
}

// GateChoice defines a single choice option for an approval gate.
type GateChoice struct {
	Label  string `yaml:"label"`            // Human-readable label (e.g. "Approve")
	Key    string `yaml:"key"`              // Keyboard shortcut key (e.g. "a")
	Target string `yaml:"target,omitempty"` // Target step ID on selection, or "_fail" to abort pipeline
}

// GateConfig configures a blocking gate step.
type GateConfig struct {
	Type      string `yaml:"type"`                 // "approval", "pr_merge", "ci_pass", "timer"
	Auto      bool   `yaml:"auto,omitempty"`       // Auto-approve (for testing)
	Timeout   string `yaml:"timeout,omitempty"`    // Duration string (e.g. "30m", "2h")
	Message   string `yaml:"message,omitempty"`    // Display message while waiting
	TUIAction string `yaml:"tui_action,omitempty"` // TUI action identifier

	// Human approval gate fields
	Choices  []GateChoice `yaml:"choices,omitempty"`  // Choice options for interactive approval
	Freeform bool         `yaml:"freeform,omitempty"` // Allow freeform text input alongside choices
	Default  string       `yaml:"default,omitempty"`  // Default choice key (used on timeout or auto-approve)
	Prompt   string       `yaml:"prompt,omitempty"`   // Prompt text displayed to the user

	// Poll gate fields (pr_merge, ci_pass)
	PRNumber int    `yaml:"pr_number,omitempty"` // PR number for pr_merge gate
	Repo     string `yaml:"repo,omitempty"`      // "owner/repo" slug; detected from git remotes if empty
	Branch   string `yaml:"branch,omitempty"`    // Branch name for ci_pass gate; detected from git if empty
	Interval string `yaml:"interval,omitempty"`  // Poll interval (e.g. "30s"); default 30s

	// Runtime-only field set by the executor before invoking the gate handler.
	// Not serialized to YAML. Used by the WebUI gate handler to track which
	// step a pending gate belongs to.
	RuntimeStepID string `yaml:"-" json:"-"`
}

// Validate checks that the GateConfig is well-formed when it uses choices.
func (g *GateConfig) Validate(stepIDs map[string]bool) error {
	if len(g.Choices) == 0 {
		return nil // No choices — legacy gate, no validation needed
	}

	// Validate unique keys
	keys := make(map[string]bool, len(g.Choices))
	for i, c := range g.Choices {
		if c.Key == "" {
			return fmt.Errorf("gate choice[%d]: key is required", i)
		}
		if c.Label == "" {
			return fmt.Errorf("gate choice[%d]: label is required", i)
		}
		if keys[c.Key] {
			return fmt.Errorf("gate choice[%d]: duplicate key %q", i, c.Key)
		}
		keys[c.Key] = true

		// Validate target references
		if c.Target != "" && c.Target != "_fail" && stepIDs != nil {
			if !stepIDs[c.Target] {
				return fmt.Errorf("gate choice[%d]: target %q is not a valid step ID or _fail", i, c.Target)
			}
		}
	}

	// Validate default references a valid choice key
	if g.Default != "" {
		found := false
		for _, c := range g.Choices {
			if c.Key == g.Default {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("gate default %q does not match any choice key", g.Default)
		}
	}

	return nil
}

// FindChoiceByKey returns the GateChoice matching the given key, or nil.
func (g *GateConfig) FindChoiceByKey(key string) *GateChoice {
	for i := range g.Choices {
		if g.Choices[i].Key == key {
			return &g.Choices[i]
		}
	}
	return nil
}

// LoopConfig configures a feedback loop with termination condition.
type LoopConfig struct {
	MaxIterations int    `yaml:"max_iterations"`  // Hard limit on iterations
	Until         string `yaml:"until,omitempty"` // Template condition for early exit
	Steps         []Step `yaml:"steps,omitempty"` // Sub-steps to execute per iteration
}

// AggregateConfig configures output collection from prior steps.
type AggregateConfig struct {
	From     string `yaml:"from"`     // Template expression for source data
	Into     string `yaml:"into"`     // Output file path
	Strategy string `yaml:"strategy"` // "merge_arrays", "concat", "reduce"
}

type SubPipelineConfig struct {
	Inject        []string `yaml:"inject,omitempty"`         // Parent artifact names to inject into child
	Extract       []string `yaml:"extract,omitempty"`        // Child artifact names to extract back to parent
	Timeout       string   `yaml:"timeout,omitempty"`        // Hard timeout for child execution (e.g. "3600s")
	MaxCycles     int      `yaml:"max_cycles,omitempty"`     // Max iterations for child loop steps
	StopCondition string   `yaml:"stop_condition,omitempty"` // Template expression for early termination
}

// Validate checks that the SubPipelineConfig is well-formed.
func (c *SubPipelineConfig) Validate() error {
	if c == nil {
		return nil
	}
	if c.Timeout != "" {
		if _, err := time.ParseDuration(c.Timeout); err != nil {
			return fmt.Errorf("invalid timeout %q: %w", c.Timeout, err)
		}
	}
	if c.MaxCycles < 0 {
		return fmt.Errorf("max_cycles must be >= 0, got %d", c.MaxCycles)
	}
	return nil
}

// ParseTimeout returns the parsed timeout duration, or zero if not set.
func (c *SubPipelineConfig) ParseTimeout() time.Duration {
	if c == nil || c.Timeout == "" {
		return 0
	}
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 0
	}
	return d
}

// PipelineOutput defines a named output alias for a pipeline.
type PipelineOutput struct {
	Step     string `yaml:"step"`            // Source step ID
	Artifact string `yaml:"artifact"`        // Artifact name
	Field    string `yaml:"field,omitempty"` // Optional JSON field extraction
}
