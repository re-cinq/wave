package hooks

import "time"

// EventType represents a lifecycle event that hooks can subscribe to.
type EventType string

const (
	EventRunStart          EventType = "run_start"
	EventRunCompleted      EventType = "run_completed"
	EventRunFailed         EventType = "run_failed"
	EventStepStart         EventType = "step_start"
	EventStepCompleted     EventType = "step_completed"
	EventStepFailed        EventType = "step_failed"
	EventStepRetrying      EventType = "step_retrying"
	EventContractValidated EventType = "contract_validated"
	EventArtifactCreated   EventType = "artifact_created"
	EventWorkspaceCreated  EventType = "workspace_created"
)

// ValidEventTypes is the set of all valid lifecycle event types.
var ValidEventTypes = map[EventType]bool{
	EventRunStart:          true,
	EventRunCompleted:      true,
	EventRunFailed:         true,
	EventStepStart:         true,
	EventStepCompleted:     true,
	EventStepFailed:        true,
	EventStepRetrying:      true,
	EventContractValidated: true,
	EventArtifactCreated:   true,
	EventWorkspaceCreated:  true,
}

// HookType represents the execution type of a hook.
type HookType string

const (
	HookTypeCommand  HookType = "command"
	HookTypeHTTP     HookType = "http"
	HookTypeLLMJudge HookType = "llm_judge"
	HookTypeScript   HookType = "script"
)

// ValidHookTypes is the set of all valid hook types.
var ValidHookTypes = map[HookType]bool{
	HookTypeCommand:  true,
	HookTypeHTTP:     true,
	HookTypeLLMJudge: true,
	HookTypeScript:   true,
}

// LifecycleHookDef defines a lifecycle hook in the manifest.
type LifecycleHookDef struct {
	Name     string    `yaml:"name"`
	Event    EventType `yaml:"event"`
	Type     HookType  `yaml:"type"`
	Command  string    `yaml:"command,omitempty"`
	URL      string    `yaml:"url,omitempty"`
	Model    string    `yaml:"model,omitempty"`
	Prompt   string    `yaml:"prompt,omitempty"`
	Script   string    `yaml:"script,omitempty"`
	Matcher  string    `yaml:"matcher,omitempty"`
	Blocking *bool     `yaml:"blocking,omitempty"`
	FailOpen *bool     `yaml:"fail_open,omitempty"`
	Timeout  string    `yaml:"timeout,omitempty"`
}

// IsBlocking returns whether this hook blocks execution on failure.
// Uses explicit setting if provided, otherwise defaults per event type.
func (h *LifecycleHookDef) IsBlocking() bool {
	if h.Blocking != nil {
		return *h.Blocking
	}
	switch h.Event {
	case EventRunStart, EventStepStart, EventStepCompleted:
		return true
	default:
		return false
	}
}

// IsFailOpen returns whether this hook fails open (allows pipeline to continue)
// when the hook itself errors. Defaults: true for LLM/HTTP, false for command/script.
func (h *LifecycleHookDef) IsFailOpen() bool {
	if h.FailOpen != nil {
		return *h.FailOpen
	}
	switch h.Type {
	case HookTypeLLMJudge, HookTypeHTTP:
		return true
	default:
		return false
	}
}

// GetTimeout returns the configured timeout duration for this hook.
// Falls back to type-specific defaults: 30s for commands, 10s for HTTP, 60s for LLM, 30s for scripts.
func (h *LifecycleHookDef) GetTimeout() time.Duration {
	if h.Timeout != "" {
		if d, err := time.ParseDuration(h.Timeout); err == nil {
			return d
		}
	}
	switch h.Type {
	case HookTypeHTTP:
		return 10 * time.Second
	case HookTypeLLMJudge:
		return 60 * time.Second
	default:
		return 30 * time.Second
	}
}

// HookEvent carries contextual information to hook executors.
type HookEvent struct {
	Type       EventType `json:"type"`
	PipelineID string    `json:"pipeline_id"`
	StepID     string    `json:"step_id,omitempty"`
	Input      string    `json:"input,omitempty"`
	Workspace  string    `json:"workspace,omitempty"`
	Artifacts  []string  `json:"artifacts,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// HookDecision represents the decision from a hook execution.
type HookDecision string

const (
	DecisionProceed HookDecision = "proceed"
	DecisionBlock   HookDecision = "block"
	DecisionSkip    HookDecision = "skip"
)

// HookResult captures the outcome of a single hook execution.
type HookResult struct {
	HookName string
	Decision HookDecision
	Reason   string
	Duration time.Duration
	Err      error
}
